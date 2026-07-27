package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"github.com/C-mrade/openclaw-portable-bridge/internal/release"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFindPortableRootFromNestedLauncher(t *testing.T) {
	root := t.TempDir()
	target := "linux-amd64"
	mustMkdir(t, filepath.Join(root, "config"))
	mustMkdir(t, filepath.Join(root, "payload", target))
	mustMkdir(t, filepath.Join(root, "launchers", target))
	mustFile(t, filepath.Join(root, "config", "bridge-public.json"))
	mustFile(t, filepath.Join(root, "payload", target, "manifest.json"))

	got, err := findPortableRoot(filepath.Join(root, "launchers", target), target)
	if err != nil || got != root {
		t.Fatalf("findPortableRoot() = %q, %v; want %q", got, err, root)
	}
}

func TestSupportedTargets(t *testing.T) {
	for _, target := range [][2]string{{"windows", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}} {
		if !supportedTarget(target[0], target[1]) {
			t.Fatalf("expected supported target %v", target)
		}
	}
	if supportedTarget("freebsd", "amd64") || supportedTarget("linux", "386") {
		t.Fatal("unsupported target accepted")
	}
}

func TestDefaultLauncherVersionIsNotReleaseCompatible(t *testing.T) {
	if release.VersionAtLeast(launcherVersion, "0.1.0") {
		t.Fatalf("development launcher version %q unexpectedly accepted", launcherVersion)
	}
}

func TestLinuxTerminalCommandPrefersDesktopStandard(t *testing.T) {
	available := map[string]string{
		"xdg-terminal-exec": "/usr/bin/xdg-terminal-exec",
		"kitty":             "/usr/bin/kitty",
	}
	lookup := func(name string) (string, error) {
		if path := available[name]; path != "" {
			return path, nil
		}
		return "", errors.New("not found")
	}

	path, args, err := linuxTerminalCommand("/media/USB/OPENCLAW BRIDGE - LINUX.exe", lookup)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/usr/bin/xdg-terminal-exec" {
		t.Fatalf("selected %q; want xdg-terminal-exec", path)
	}
	want := []string{"/media/USB/OPENCLAW BRIDGE - LINUX.exe"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v; want %#v", args, want)
	}
}

func TestLinuxTerminalCommandFallsBackToKittyWithoutShellQuoting(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "kitty" {
			return "/usr/bin/kitty", nil
		}
		return "", errors.New("not found")
	}

	self := "/media/USB drive/OPENCLAW BRIDGE - LINUX.exe"
	path, args, err := linuxTerminalCommand(self, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/usr/bin/kitty" {
		t.Fatalf("selected %q; want kitty", path)
	}
	want := []string{"--title", "OpenClaw Portable Bridge", self}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v; want %#v", args, want)
	}
}

func TestLinuxTerminalCommandFailsClosedWithoutEmulator(t *testing.T) {
	lookup := func(string) (string, error) { return "", errors.New("not found") }
	if _, _, err := linuxTerminalCommand("/tmp/bridge", lookup); err == nil {
		t.Fatal("missing terminal emulator was accepted")
	}
}

func TestLoadPublicConfigRequiresValidSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "config"))
	config := []byte(`{"usbId":"test-usb","brokerUrl":"https://bridge.example.test"}`)
	configPath := filepath.Join(root, "config", "bridge-public.json")
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath+".sig", []byte(release.Sign(priv, config)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPublicConfig(root, pub); err != nil {
		t.Fatalf("valid signed config rejected: %v", err)
	}
	if err := os.WriteFile(configPath, append(config, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPublicConfig(root, pub); err == nil {
		t.Fatal("tampered public config accepted")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func mustFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
}
