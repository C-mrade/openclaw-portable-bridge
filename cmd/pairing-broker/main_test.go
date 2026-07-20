package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/C-mrade/openclaw-portable-bridge/internal/audit"
	"github.com/C-mrade/openclaw-portable-bridge/internal/auth"
	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
)

func TestDeveloperCapabilityProfileIsAccepted(t *testing.T) {
	developer := []string{
		"system.info", "system.network", "disk.list", "service.list",
		"process.list", "process.start", "process.stop-owned", "shell.run",
		"shell.run-admin", "powershell.run", "shell.start", "shell.status",
		"shell.cancel", "files.list", "files.read", "files.read-chunk",
		"files.write", "files.write-chunk", "files.upload", "files.download",
		"session.disconnect",
	}
	if !validCapabilities(developer) {
		t.Fatalf("Developer profile with %d capabilities was rejected", len(developer))
	}
}

func TestCapabilityValidationRejectsUnknownDuplicateAndOversized(t *testing.T) {
	if validCapabilities([]string{"system.info", "unknown"}) {
		t.Fatal("unknown capability accepted")
	}
	if validCapabilities([]string{"system.info", "system.info"}) {
		t.Fatal("duplicate capability accepted")
	}
	tooMany := make([]string, 33)
	for i := range tooMany {
		tooMany[i] = "system.info"
	}
	if validCapabilities(tooMany) {
		t.Fatal("oversized capability request accepted")
	}
}

func testServer(t *testing.T) (*server, string) {
	t.Helper()
	a, err := audit.Open(t.TempDir() + "/audit.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	token := "session-token"
	x := &pending{
		Req:       protocol.PairRequest{Requested: []string{"system.info"}},
		Reply:     protocol.PairReply{Status: "approved", ExpiresAt: time.Now().Add(time.Minute)},
		TokenHash: auth.Hash(token), Commands: map[string]*commandState{},
	}
	return &server{p: map[string]*pending{"request": x}, audit: a, admin: "administrator-token-for-tests", seen: map[string]time.Time{}, rates: map[string][]time.Time{}}, token
}

func requestJSON(t *testing.T, handler http.HandlerFunc, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&b).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(method, path, &b)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

func TestCommandIDsAreIdempotentAndConflictsAreRejected(t *testing.T) {
	s, _ := testServer(t)
	command := protocol.Command{ID: "same-id", Name: "system.info"}
	body := map[string]any{"requestId": "request", "command": command}
	w := requestJSON(t, s.enqueue, http.MethodPost, "/v1/admin/command", s.admin, body)
	if w.Code != http.StatusAccepted {
		t.Fatalf("first enqueue: %d %s", w.Code, w.Body.String())
	}
	w = requestJSON(t, s.enqueue, http.MethodPost, "/v1/admin/command", s.admin, body)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"idempotent":true`)) {
		t.Fatalf("idempotent retry: %d %s", w.Code, w.Body.String())
	}
	command.Params = json.RawMessage(`{"unexpected":true}`)
	w = requestJSON(t, s.enqueue, http.MethodPost, "/v1/admin/command", s.admin, map[string]any{"requestId": "request", "command": command})
	if w.Code != http.StatusConflict {
		t.Fatalf("conflicting retry accepted: %d %s", w.Code, w.Body.String())
	}
}

func TestLeaseAckAndResultLifecycle(t *testing.T) {
	s, token := testServer(t)
	command := protocol.Command{ID: "lifecycle", Name: "system.info"}
	w := requestJSON(t, s.enqueue, http.MethodPost, "/v1/admin/command", s.admin, map[string]any{"requestId": "request", "command": command})
	if w.Code != http.StatusAccepted {
		t.Fatal(w.Body.String())
	}
	w = requestJSON(t, s.poll, http.MethodPost, "/v1/session/poll", token, map[string]string{})
	if w.Code != http.StatusOK {
		t.Fatalf("poll: %d %s", w.Code, w.Body.String())
	}
	w = requestJSON(t, s.ack, http.MethodPost, "/v1/session/ack", token, map[string]string{"commandId": command.ID})
	if w.Code != http.StatusOK {
		t.Fatalf("ack: %d %s", w.Code, w.Body.String())
	}
	result := protocol.Result{ID: command.ID, Name: command.Name, Output: `{}`}
	w = requestJSON(t, s.result, http.MethodPost, "/v1/session/result", token, result)
	if w.Code != http.StatusOK {
		t.Fatalf("result: %d %s", w.Code, w.Body.String())
	}
	w = requestJSON(t, s.result, http.MethodPost, "/v1/session/result", token, result)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate result accepted: %d %s", w.Code, w.Body.String())
	}
}
