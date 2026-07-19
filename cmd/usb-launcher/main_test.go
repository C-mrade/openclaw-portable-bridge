package main

import (
	"os"
	"path/filepath"
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
