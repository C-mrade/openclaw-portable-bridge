package executor

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeUTF16LE(t *testing.T) {
	raw := []byte{0xff, 0xfe, 'O', 0, 'K', 0, '\r', 0, '\n', 0}
	if got := normalizeOutput(raw); got != "OK\r\n" {
		t.Fatalf("unexpected normalized output %q", got)
	}
}

func TestChunkedTransferAndPagination(t *testing.T) {
	root := t.TempDir()
	e, err := New([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("chunk-one/chunk-two")
	sum := sha256.Sum256(data)
	target := filepath.Join(root, "chunked.bin")
	parts := [][]byte{data[:10], data[10:]}
	offset := 0
	for i, part := range parts {
		params, _ := json.Marshal(map[string]any{"path": target, "offset": offset, "dataBase64": base64.StdEncoding.EncodeToString(part), "final": i == len(parts)-1, "expectedSHA256": hex.EncodeToString(sum[:])})
		if _, err = e.Execute(protocol.Command{Name: "files.write-chunk", Params: params}); err != nil {
			t.Fatal(err)
		}
		offset += len(part)
	}
	if got := mustRead(t, target); string(got) != string(data) {
		t.Fatalf("unexpected chunked file %q", got)
	}
	readParams, _ := json.Marshal(map[string]any{"path": target, "offset": 6, "limit": 3})
	result, err := e.Execute(protocol.Command{Name: "files.read-chunk", Params: readParams})
	if err != nil || !strings.Contains(result, base64.StdEncoding.EncodeToString([]byte("one"))) {
		t.Fatalf("unexpected read chunk result %q: %v", result, err)
	}
	listParams, _ := json.Marshal(map[string]any{"path": root, "offset": 0, "limit": 1, "filter": "chunk"})
	result, err = e.Execute(protocol.Command{Name: "files.list", Params: listParams})
	if err != nil || !strings.Contains(result, `"total":1`) {
		t.Fatalf("unexpected paginated list %q: %v", result, err)
	}
}

func TestChunkedTransferBoundsAndChecksumCleanup(t *testing.T) {
	root := t.TempDir()
	e, err := New([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "bounded.bin")
	oversized, _ := json.Marshal(map[string]any{"path": target, "offset": MaxTransfer, "dataBase64": "eA=="})
	if _, err = e.Execute(protocol.Command{Name: "files.write-chunk", Params: oversized}); err == nil {
		t.Fatal("chunk beyond total transfer limit accepted")
	}
	badSum, _ := json.Marshal(map[string]any{"path": target, "offset": 0, "dataBase64": "eA==", "final": true, "expectedSHA256": strings.Repeat("0", 64)})
	if _, err = e.Execute(protocol.Command{Name: "files.write-chunk", Params: badSum}); err == nil {
		t.Fatal("invalid checksum accepted")
	}
	if _, err = os.Stat(target + ".openclaw-part"); !os.IsNotExist(err) {
		t.Fatalf("partial transfer survived checksum failure: %v", err)
	}
}

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
