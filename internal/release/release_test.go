package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSignedReleaseAndTamper(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	d := t.TempDir()
	payload := []byte("safe-client")
	m := Manifest{Version: "1", OS: "windows", Architecture: "amd64", Filename: "bridge-client.exe", SHA256: Hash(payload), Size: int64(len(payload)), MinimumLauncher: "1.0.0", MinimumProtocol: 1}
	mb, _ := json.Marshal(m)
	mustWrite(t, filepath.Join(d, "manifest.json"), mb)
	mustWrite(t, filepath.Join(d, "manifest.json.sig"), []byte(Sign(priv, mb)))
	mustWrite(t, filepath.Join(d, m.Filename), payload)
	mustWrite(t, filepath.Join(d, m.Filename+".sig"), []byte(Sign(priv, payload)))
	if _, _, e := LoadAndVerify(d, pub, "windows", "amd64"); e != nil {
		t.Fatal(e)
	}
	if _, _, e := LoadAndVerify(d, pub, "linux", "amd64"); e == nil {
		t.Fatal("manifest for the wrong operating system accepted")
	}
	mustWrite(t, filepath.Join(d, m.Filename), []byte("evil-client"))
	if _, _, e := LoadAndVerify(d, pub, "windows", "amd64"); e == nil {
		t.Fatal("tampered payload accepted")
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		current string
		minimum string
		want    bool
	}{
		{"0.5.1-mvp-dev", "0.5.0", true},
		{"v1.0.0", "1.0", true},
		{"0.5.0", "0.5.1", false},
		{"1.2", "1.2.1", false},
		{"broken", "0.1.0", false},
		{"1..0", "1.0.0", false},
	}
	for _, tt := range tests {
		if got := VersionAtLeast(tt.current, tt.minimum); got != tt.want {
			t.Errorf("VersionAtLeast(%q, %q) = %v, want %v", tt.current, tt.minimum, got, tt.want)
		}
	}
}

func mustWrite(t *testing.T, p string, b []byte) {
	t.Helper()
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
