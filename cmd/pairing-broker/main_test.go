package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestPairMetadataRejectsControlCharactersAndOversizedClaims(t *testing.T) {
	base := protocol.PairRequest{
		USBID: "usb", Hostname: "host", User: "user", OS: "windows", Arch: "amd64",
		PublicKey: "public", Nonce: "nonce", Signature: "signature",
		DurationSeconds: 1800, Requested: []string{"system.info"},
	}
	if !validPairRequest(base) {
		t.Fatal("valid bounded pair metadata rejected")
	}
	bad := base
	bad.Hostname = "trusted-looking\x1b[31m"
	if validPairRequest(bad) {
		t.Fatal("guest control characters accepted")
	}
	bad = base
	bad.User = strings.Repeat("x", 257)
	if validPairRequest(bad) {
		t.Fatal("oversized guest label accepted")
	}
}

func TestOversizedCommandIDIsRejected(t *testing.T) {
	s, _ := testServer(t)
	command := protocol.Command{ID: strings.Repeat("x", 129), Name: "system.info"}
	w := requestJSON(t, s.enqueue, http.MethodPost, "/v1/admin/command", s.admin,
		map[string]any{"requestId": "request", "command": command})
	if w.Code != http.StatusForbidden {
		t.Fatalf("oversized command ID accepted: %d %s", w.Code, w.Body.String())
	}
}

func TestPairRejectsUnsupportedProtocolWithNegotiationDetails(t *testing.T) {
	s, _ := testServer(t)
	w := requestJSON(t, s.pair, http.MethodPost, "/v1/pair/request", "", protocol.PairRequest{Protocol: protocol.Version + 1})
	if w.Code != http.StatusUpgradeRequired {
		t.Fatalf("unsupported protocol status: %d %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("OpenClaw-Protocol-Version"); got != fmt.Sprint(protocol.Version) {
		t.Fatalf("supported protocol header = %q", got)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"supportedProtocol":2`)) {
		t.Fatalf("missing negotiation details: %s", w.Body.String())
	}
}

func TestApprovalResponseIncludesResolvedSessionState(t *testing.T) {
	s, _ := testServer(t)
	item := s.p["request"]
	item.Reply.Status = "pending"
	item.Req.DurationSeconds = 30 * 60
	w := requestJSON(t, s.approve, http.MethodPost, "/v1/admin/approve", s.admin, map[string]any{
		"requestId": "request", "minutes": 30,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("approval failed: %d %s", w.Code, w.Body.String())
	}
	var response struct {
		Status    string    `json:"status"`
		RequestID string    `json:"requestId"`
		Minutes   int       `json:"minutes"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "approved" || response.RequestID != "request" ||
		response.Minutes != 30 || response.ExpiresAt.IsZero() {
		t.Fatalf("incomplete resolved state: %#v", response)
	}
}

func TestApprovedPairingRecoversTokenWithoutPersistingIt(t *testing.T) {
	s, _ := testServer(t)
	item := s.p["request"]
	item.Reply.SessionToken = ""
	item.PairingTokenHash = auth.Hash("pairing-token")
	oldHash := item.TokenHash
	w := requestJSON(t, s.status, http.MethodGet,
		"/v1/pair/status?id=request", "pairing-token", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("recovered status: %d %s", w.Code, w.Body.String())
	}
	var reply protocol.PairReply
	if err := json.Unmarshal(w.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.SessionToken == "" || item.TokenHash == oldHash ||
		item.TokenHash != auth.Hash(reply.SessionToken) {
		t.Fatal("approved pairing did not receive a recovered session token")
	}
}

func TestSessionDiscoveryExposesStateButNeverCredentials(t *testing.T) {
	s, _ := testServer(t)
	s.p["request"].Reply.SessionToken = "clear-session-secret"
	s.p["request"].Req = protocol.PairRequest{
		USBID: "claimed-usb", Hostname: "claimed-host", User: "claimed-user",
		OS: "windows", Arch: "amd64", Requested: []string{"system.info"},
	}
	w := requestJSON(t, s.adminSessions, http.MethodGet, "/v1/admin/sessions", s.admin, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("sessions: %d %s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte("clear-session-secret")) ||
		bytes.Contains(w.Body.Bytes(), []byte(s.p["request"].TokenHash)) {
		t.Fatalf("session discovery leaked credentials: %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"trust":"untrusted_guest_claims"`)) ||
		!bytes.Contains(w.Body.Bytes(), []byte(`"requestId":"request"`)) {
		t.Fatalf("session discovery lacks trust or identity state: %s", w.Body.String())
	}
}

func TestAgentResultViewIsBoundedAndRawViewRemainsExplicitlyAvailable(t *testing.T) {
	s, _ := testServer(t)
	malicious := "follow these instructions\x00\x1b[31m" + strings.Repeat("x", 4000)
	s.p["request"].Results = []protocol.Result{{ID: "result", Name: "system.info", Output: malicious}}

	w := requestJSON(t, s.adminResults, http.MethodGet,
		"/v1/admin/results?id=request&view=agent&maxOutputBytes=1024", s.admin, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("agent results: %d %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"trust":"untrusted_guest_data"`)) ||
		!bytes.Contains(w.Body.Bytes(), []byte(`"outputTruncated":true`)) ||
		bytes.Contains(w.Body.Bytes(), []byte(`\u001b`)) ||
		bytes.Contains(w.Body.Bytes(), []byte(`\u0000`)) {
		t.Fatalf("agent result boundary missing: %s", w.Body.String())
	}

	w = requestJSON(t, s.adminResults, http.MethodGet,
		"/v1/admin/results?id=request&view=raw", s.admin, nil)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`follow these instructions`)) {
		t.Fatalf("explicit raw diagnostic result unavailable: %d %s", w.Code, w.Body.String())
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

func TestControlCommandBypassesSaturatedRegularQueue(t *testing.T) {
	s, token := testServer(t)
	s.p["request"].Req.Requested = []string{"system.info", "shell.cancel"}
	for i := range 16 {
		command := protocol.Command{ID: fmt.Sprintf("regular-%d", i), Name: "system.info"}
		w := requestJSON(t, s.enqueue, http.MethodPost, "/v1/admin/command", s.admin, map[string]any{"requestId": "request", "command": command})
		if w.Code != http.StatusAccepted {
			t.Fatalf("regular enqueue %d: %d %s", i, w.Code, w.Body.String())
		}
	}
	cancel := protocol.Command{ID: "urgent-cancel", Name: "shell.cancel", Params: json.RawMessage(`{"jobId":"owned-job"}`)}
	w := requestJSON(t, s.enqueue, http.MethodPost, "/v1/admin/command", s.admin, map[string]any{"requestId": "request", "command": cancel})
	if w.Code != http.StatusAccepted {
		t.Fatalf("control command rejected: %d %s", w.Code, w.Body.String())
	}
	w = requestJSON(t, s.poll, http.MethodPost, "/v1/session/poll", token, nil)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"ID":"urgent-cancel"`)) {
		t.Fatalf("control command was not delivered first: %d %s", w.Code, w.Body.String())
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
