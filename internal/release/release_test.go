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
	m := Manifest{Version: "1", OS: "windows", Architecture: "amd64", Filename: "bridge-client.exe", SHA256: Hash(payload), Size: int64(len(payload)), MinimumProtocol: 1}
	mb, _ := json.Marshal(m)
	mustWrite(t, filepath.Join(d, "manifest.json"), mb)
	mustWrite(t, filepath.Join(d, "manifest.json.sig"), []byte(Sign(priv, mb)))
	mustWrite(t, filepath.Join(d, m.Filename), payload)
	mustWrite(t, filepath.Join(d, m.Filename+".sig"), []byte(Sign(priv, payload)))
	if _, _, e := LoadAndVerify(d, pub); e != nil {
		t.Fatal(e)
	}
	mustWrite(t, filepath.Join(d, m.Filename), []byte("evil-client"))
	if _, _, e := LoadAndVerify(d, pub); e == nil {
		t.Fatal("tampered payload accepted")
	}
}
func mustWrite(t *testing.T, p string, b []byte) {
	t.Helper()
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
