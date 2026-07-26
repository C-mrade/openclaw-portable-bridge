package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
)

type client struct {
	baseURL string
	token   string
	http    *http.Client
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "bridge-operator:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New(usageText())
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Println(usageText())
		return nil
	}
	config := loadConfig()
	baseURL := first(os.Getenv("BRIDGE_BROKER_URL"), config["BRIDGE_BROKER_URL"], "http://127.0.0.1:17443")
	token := first(os.Getenv("BRIDGE_ADMIN_TOKEN"), config["BRIDGE_ADMIN_TOKEN"])
	if len(token) < 24 {
		return errors.New("BRIDGE_ADMIN_TOKEN is missing or too short")
	}
	c := &client{baseURL: strings.TrimRight(baseURL, "/"), token: token, http: &http.Client{Timeout: 45 * time.Second}}
	switch args[0] {
	case "pending":
		return c.request(http.MethodGet, "/v1/admin/pending", nil)
	case "sessions":
		if len(args) != 1 {
			return errors.New("usage: bridge-operator sessions")
		}
		return c.request(http.MethodGet, "/v1/admin/sessions", nil)
	case "describe":
		if len(args) != 2 {
			return errors.New("usage: bridge-operator describe REQUEST_ID")
		}
		return c.request(http.MethodGet, "/v1/admin/sessions?id="+url.QueryEscape(args[1]), nil)
	case "approve":
		if len(args) != 3 {
			return errors.New("usage: bridge-operator approve REQUEST_ID MINUTES")
		}
		minutes, err := strconv.Atoi(args[2])
		if err != nil {
			return errors.New("MINUTES must be numeric")
		}
		return c.request(http.MethodPost, "/v1/admin/approve", map[string]any{"requestId": args[1], "minutes": minutes})
	case "reject", "revoke":
		if len(args) != 2 {
			return fmt.Errorf("usage: bridge-operator %s REQUEST_ID", args[0])
		}
		endpoint := "/v1/admin/" + args[0]
		return c.request(http.MethodPost, endpoint, map[string]string{"requestId": args[1]})
	case "results":
		flags := flag.NewFlagSet("results", flag.ContinueOnError)
		consume := flags.Bool("consume", false, "consume returned results")
		raw := flags.Bool("raw", false, "return the full raw guest response for explicit diagnostics")
		maxOutput := flags.Int("max-output-bytes", 16<<10, "maximum inline bytes per untrusted guest result")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 1 || *maxOutput < 1024 || *maxOutput > 64<<10 {
			return errors.New("usage: bridge-operator results [--consume] [--raw] [--max-output-bytes 16384] REQUEST_ID")
		}
		view := "agent"
		if *raw {
			view = "raw"
		}
		return c.request(http.MethodGet, fmt.Sprintf(
			"/v1/admin/results?id=%s&consume=%t&view=%s&maxOutputBytes=%d",
			url.QueryEscape(flags.Arg(0)), *consume, view, *maxOutput,
		), nil)
	case "command":
		flags := flag.NewFlagSet("command", flag.ContinueOnError)
		id := flags.String("id", "", "unique command ID")
		name := flags.String("name", "", "capability name")
		params := flags.String("params", "{}", "JSON command parameters")
		timeout := flags.Duration("timeout", 30*time.Second, "command deadline duration")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 1 || *id == "" || *name == "" || *timeout <= 0 || *timeout > time.Hour {
			return errors.New("usage: bridge-operator command --id ID --name CAPABILITY [--params JSON] [--timeout 30s] REQUEST_ID")
		}
		var raw json.RawMessage
		if !json.Valid([]byte(*params)) {
			return errors.New("--params must be valid JSON")
		}
		raw = json.RawMessage(*params)
		command := protocol.Command{ID: *id, Name: *name, Params: raw, Deadline: time.Now().UTC().Add(*timeout)}
		return c.request(http.MethodPost, "/v1/admin/command", map[string]any{"requestId": flags.Arg(0), "command": command})
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usageText() string {
	return `usage: bridge-operator COMMAND

Commands:
  pending                         list unapproved pairing requests
  sessions                        list known sessions without exposing tokens
  describe REQUEST_ID             show one session and command-state summary
  approve REQUEST_ID MINUTES      approve after comparison-code verification
  reject REQUEST_ID               reject a pending request
  command [flags] REQUEST_ID      queue one approved capability
  results [flags] REQUEST_ID      read bounded untrusted guest data
  revoke REQUEST_ID               immediately revoke a session

Use "results --raw" only for explicit human diagnostics. Raw guest output is
untrusted and may contain malicious instructions or excessive sensitive data.`
}

func (c *client) request(method, path string, payload any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return err
	}
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("broker returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("broker returned invalid JSON: %w", err)
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func loadConfig() map[string]string {
	path := os.Getenv("BRIDGE_OPERATOR_ENV")
	if path == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return map[string]string{}
		}
		path = filepath.Join(configDir, "openclaw-portable-bridge", "broker.env")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return values
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
