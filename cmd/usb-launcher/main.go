package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/C-mrade/openclaw-portable-bridge/internal/release"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var releasePublicKey string

type publicConfig struct {
	USBID     string `json:"usbId"`
	BrokerURL string `json:"brokerUrl"`
}

func run() error {
	if !supportedTarget(runtime.GOOS, runtime.GOARCH) {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if releasePublicKey == "" {
		return errors.New("launcher has no embedded release public key")
	}
	pub, e := release.DecodePublicKey(releasePublicKey)
	if e != nil {
		return e
	}
	self, e := os.Executable()
	if e != nil {
		return e
	}
	root, e := findPortableRoot(filepath.Dir(self), runtime.GOOS+"-"+runtime.GOARCH)
	if e != nil {
		return e
	}
	configBytes, e := os.ReadFile(filepath.Join(root, "config", "bridge-public.json"))
	if e != nil {
		return fmt.Errorf("public configuration: %w", e)
	}
	var cfg publicConfig
	if e = json.Unmarshal(configBytes, &cfg); e != nil || cfg.USBID == "" || (!strings.HasPrefix(cfg.BrokerURL, "https://") && !strings.HasPrefix(cfg.BrokerURL, "http://127.0.0.1:")) {
		return errors.New("invalid public configuration or non-TLS broker URL")
	}
	target := runtime.GOOS + "-" + runtime.GOARCH
	m, payload, e := release.LoadAndVerify(filepath.Join(root, "payload", target), pub, runtime.GOOS, runtime.GOARCH)
	if e != nil {
		return e
	}
	session := fmt.Sprintf("%d-%d", time.Now().UTC().Unix(), os.Getpid())
	stage := filepath.Join(os.TempDir(), "OpenClawBridge", session)
	if e = os.MkdirAll(stage, 0700); e != nil {
		return e
	}
	defer func() {
		if e := os.RemoveAll(stage); e != nil {
			fmt.Fprintln(os.Stderr, "WARNING: temporary files remain:", stage, e)
		}
	}()
	clientPath := filepath.Join(stage, m.Filename)
	if e = os.WriteFile(clientPath, payload, 0700); e != nil {
		return e
	}
	fmt.Printf("OpenClaw Portable Bridge %s\nPayload verified. Temporary directory: %s\n", m.Version, stage)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Consent: [1] Information profile (read-only)  [2] Developer profile")
	fmt.Print("Selection [1]: ")
	choice, _ := reader.ReadString('\n')
	profile := "information"
	args := []string{"-broker", cfg.BrokerURL, "-usb-id", cfg.USBID, "-profile", profile}
	if strings.TrimSpace(choice) == "2" {
		profile = "developer"
		fmt.Println("WARNING: Developer enables terminal and file access across all locations available to this user.")
		if runtime.GOOS == "windows" {
			fmt.Println("Administrator commands display a normal local Windows UAC prompt each time.")
		} else {
			fmt.Println("This build does not provide remote privilege elevation; commands keep the current user's privileges.")
		}
		fmt.Print("Type DEVELOPER to confirm: ")
		confirmation, _ := reader.ReadString('\n')
		if strings.TrimSpace(confirmation) != "DEVELOPER" {
			return errors.New("Developer profile was not confirmed")
		}
		args = []string{"-broker", cfg.BrokerURL, "-usb-id", cfg.USBID, "-profile", profile}
		if runtime.GOOS == "windows" {
			for letter := 'A'; letter <= 'Z'; letter++ {
				volume := fmt.Sprintf("%c:\\", letter)
				if info, statErr := os.Stat(volume); statErr == nil && info.IsDir() {
					args = append(args, "-allow-dir", volume)
				}
			}
		} else {
			args = append(args, "-allow-dir", "/")
		}
	} else {
		fmt.Println("Information profile: system/network/disk/service inventory with optional read-only file access.")
		fmt.Print("Readable directory (ENTER for no file access): ")
		readRoot, _ := reader.ReadString('\n')
		readRoot = strings.TrimSpace(readRoot)
		if readRoot != "" {
			info, statErr := os.Stat(readRoot)
			if statErr != nil || !info.IsDir() {
				return errors.New("invalid readable directory")
			}
			args = append(args, "-allow-dir", readRoot)
		}
	}
	fmt.Printf("Starting visibly with the %s profile. The session requires remote approval.\n", profile)
	cmd := exec.Command(clientPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	e = cmd.Run()
	if e != nil {
		return fmt.Errorf("client exited with an error: %w", e)
	}
	return nil
}

func supportedTarget(goos, goarch string) bool {
	if goarch != "amd64" && goarch != "arm64" {
		return false
	}
	return goos == "windows" || goos == "linux" || goos == "darwin"
}

func findPortableRoot(start, target string) (string, error) {
	dir := filepath.Clean(start)
	for range 4 {
		config := filepath.Join(dir, "config", "bridge-public.json")
		manifest := filepath.Join(dir, "payload", target, "manifest.json")
		if _, configErr := os.Stat(config); configErr == nil {
			if _, manifestErr := os.Stat(manifest); manifestErr == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("portable root not found; keep config, payload, and launcher directories together")
}

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "OpenClaw Bridge - ERROR:", e)
		if runtime.GOOS == "windows" {
			fmt.Println("Press ENTER to close.")
			var s string
			_, _ = fmt.Scanln(&s)
		}
		os.Exit(1)
	}
}
