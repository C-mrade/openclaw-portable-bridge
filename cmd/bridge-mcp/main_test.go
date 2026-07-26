package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMCPListsNarrowBridgeTools(t *testing.T) {
	t.Setenv("BRIDGE_ADMIN_TOKEN", strings.Repeat("a", 32))
	var output bytes.Buffer
	input := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n",
	)
	if err := run(input, &output); err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(&output)
	if !scanner.Scan() || !strings.Contains(scanner.Text(), `"openclaw-portable-bridge"`) {
		t.Fatalf("missing initialize response: %s", output.String())
	}
	if !scanner.Scan() || !strings.Contains(scanner.Text(), `"bridge_approve"`) ||
		!strings.Contains(scanner.Text(), `"bridge_list_sessions"`) ||
		strings.Contains(scanner.Text(), `"admin_token"`) {
		t.Fatalf("unsafe or incomplete tool list: %s", scanner.Text())
	}
}

func TestMCPResultsAlwaysRequestAgentSafeView(t *testing.T) {
	token := strings.Repeat("c", 32)
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Fatal("missing protected broker credential")
		}
		if r.URL.Path != "/v1/admin/results" ||
			r.URL.Query().Get("view") != "agent" ||
			r.URL.Query().Get("maxOutputBytes") != "4096" {
			t.Fatalf("unsafe result request: %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"trust": "untrusted_guest_data", "results": []any{},
		})
	}))
	defer broker.Close()
	s := &server{baseURL: broker.URL, token: token, http: broker.Client()}
	result, err := s.callTool("bridge_results", map[string]any{
		"request_id": "request", "max_output_bytes": 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "untrusted_guest_data") {
		t.Fatalf("missing safe guest envelope: %s", result)
	}
}

func TestMCPToolCallKeepsTokenServerSide(t *testing.T) {
	token := strings.Repeat("b", 32)
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Fatalf("missing protected broker credential")
		}
		if r.URL.Path != "/v1/admin/pending" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"pending": []any{}})
	}))
	defer broker.Close()

	t.Setenv("BRIDGE_ADMIN_TOKEN", token)
	t.Setenv("BRIDGE_BROKER_URL", broker.URL)
	var output bytes.Buffer
	input := strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"bridge_list_pending","arguments":{}}}` + "\n")
	if err := run(input, &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), token) || !strings.Contains(output.String(), "pending") {
		t.Fatalf("credential leaked or broker result missing: %s", output.String())
	}
}

func TestMCPRejectsUnknownCapabilityBeforeBroker(t *testing.T) {
	s := &server{}
	_, err := s.callTool("bridge_command", map[string]any{
		"request_id": "request",
		"command_id": "command",
		"name":       "arbitrary.http.proxy",
	})
	if err == nil {
		t.Fatal("unknown capability accepted")
	}
}
