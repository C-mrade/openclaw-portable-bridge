package executor

import (
	"encoding/json"
	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
	"os"
	"path/filepath"
	"testing"
)

func TestPathBoundaryAndNoOverwrite(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	e, x := New([]string{root})
	if x != nil {
		t.Fatal(x)
	}
	p, _ := json.Marshal(map[string]string{"path": filepath.Join(outside, "x")})
	if _, x := e.Execute(protocol.Command{Name: "files.list", Params: p}); x == nil {
		t.Fatal("outside path accepted")
	}
	target := filepath.Join(root, "new.txt")
	w, _ := json.Marshal(map[string]string{"path": target, "dataBase64": "b2s="})
	if _, x = e.Execute(protocol.Command{Name: "files.write", Params: w}); x != nil {
		t.Fatal(x)
	}
	if string(mustRead(t, target)) != "ok" {
		t.Fatal("bad write")
	}
	if _, x = e.Execute(protocol.Command{Name: "files.write", Params: w}); x == nil {
		t.Fatal("overwrite accepted")
	}
}
func TestTraversalRejected(t *testing.T) {
	root := t.TempDir()
	e, _ := New([]string{root})
	p, _ := json.Marshal(map[string]string{"path": filepath.Join(root, "..")})
	if _, x := e.Execute(protocol.Command{Name: "files.list", Params: p}); x == nil {
		t.Fatal("traversal accepted")
	}
}
func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
