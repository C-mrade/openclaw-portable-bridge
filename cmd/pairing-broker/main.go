package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/C-mrade/openclaw-portable-bridge/internal/audit"
	"github.com/C-mrade/openclaw-portable-bridge/internal/auth"
	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
)

type pending struct {
	Req              protocol.PairRequest
	Reply            protocol.PairReply
	TokenHash        string
	PairingTokenHash string
	Queue            []string
	Commands         map[string]*commandState
	Results          []protocol.Result
	CreatedAt        time.Time
}
type commandState struct {
	Command     protocol.Command
	Fingerprint [32]byte
	Status      string
	LeaseUntil  time.Time
}
type server struct {
	mu    sync.Mutex
	p     map[string]*pending
	audit *audit.Logger
	admin string
	seen  map[string]time.Time
	rates map[string][]time.Time
	state *stateStore
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func limitedJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	return limitedJSONN(w, r, v, 64<<10)
}
func limitedJSONN(w http.ResponseWriter, r *http.Request, v any, max int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		write(w, 400, map[string]string{"error": "invalid request"})
		return false
	}
	return true
}
func (s *server) pair(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		write(w, 405, nil)
		return
	}
	var q protocol.PairRequest
	if !limitedJSON(w, r, &q) {
		return
	}
	if q.Protocol != protocol.Version {
		w.Header().Set("OpenClaw-Protocol-Version", fmt.Sprint(protocol.Version))
		write(w, http.StatusUpgradeRequired, map[string]any{
			"error":             "unsupported protocol version",
			"receivedProtocol":  q.Protocol,
			"supportedProtocol": protocol.Version,
		})
		return
	}
	if q.USBID == "" || q.DurationSeconds < 60 || q.DurationSeconds > 3600 || !validCapabilities(q.Requested) || !auth.Verify(q.PublicKey, q.Signature, protocol.CanonicalPairRequest(q)) {
		write(w, 403, map[string]string{"error": "request rejected"})
		return
	}
	if !s.allowPair(r.RemoteAddr, q.PublicKey+"\x00"+q.Nonce) {
		write(w, 429, map[string]string{"error": "rate limited or replayed"})
		return
	}
	id, _ := auth.Token()
	pairingToken, _ := auth.Token()
	id = auth.Hash(id)[:24]
	rep := protocol.PairReply{RequestID: id, Status: "pending", CompareCode: auth.CompareCode(q.PublicKey, q.Nonce), PairingToken: pairingToken}
	storedReply := rep
	storedReply.PairingToken = ""
	s.mu.Lock()
	s.p[id] = &pending{Req: q, Reply: storedReply, PairingTokenHash: auth.Hash(pairingToken), Commands: map[string]*commandState{}, CreatedAt: time.Now().UTC()}
	if err := s.state.save(s.p); err != nil {
		delete(s.p, id)
		s.mu.Unlock()
		log.Printf("persist pair request: %v", err)
		write(w, 500, map[string]string{"error": "state unavailable"})
		return
	}
	s.mu.Unlock()
	s.audit.Event("pair_requested", map[string]any{"requestId": id, "usbId": q.USBID, "compareCode": rep.CompareCode, "source": r.RemoteAddr})
	write(w, 202, rep)
}
func (s *server) allowPair(remote, replay string) bool {
	host, _, e := net.SplitHostPort(remote)
	if e != nil {
		host = remote
	}
	now := time.Now()
	key := auth.Hash(replay)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, t := range s.seen {
		if now.Sub(t) > 10*time.Minute {
			delete(s.seen, k)
		}
	}
	if _, ok := s.seen[key]; ok {
		return false
	}
	recent := s.rates[host][:0]
	for _, t := range s.rates[host] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 10 {
		s.rates[host] = recent
		return false
	}
	s.rates[host] = append(recent, now)
	s.seen[key] = now
	return true
}
func (s *server) status(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.p[r.URL.Query().Get("id")]
	pollToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if x == nil || subtle.ConstantTimeCompare([]byte(x.PairingTokenHash), []byte(auth.Hash(pollToken))) != 1 {
		write(w, 404, nil)
		return
	}
	reply := x.Reply
	reply.PairingToken = ""
	write(w, 200, reply)
}
func (s *server) approve(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		write(w, 401, nil)
		return
	}
	var q struct {
		RequestID string `json:"requestId"`
		Minutes   int    `json:"minutes"`
	}
	if !limitedJSON(w, r, &q) {
		return
	}
	if q.Minutes < 1 || q.Minutes > 60 {
		write(w, 400, nil)
		return
	}
	expiresAt, err := s.approveRequest(q.RequestID, q.Minutes)
	if err != nil {
		write(w, err.status, map[string]string{"error": err.message})
		return
	}
	s.audit.Event("pair_approved", map[string]any{"requestId": q.RequestID, "minutes": q.Minutes})
	write(w, 200, map[string]any{
		"status": "approved", "requestId": q.RequestID,
		"minutes": q.Minutes, "expiresAt": expiresAt,
	})
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string { return e.message }

func (s *server) approveRequest(requestID string, minutes int) (time.Time, *requestError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.p[requestID]
	if x == nil || x.Reply.Status != "pending" {
		return time.Time{}, &requestError{status: http.StatusNotFound, message: "pending request not found"}
	}
	maxMinutes := int((x.Req.DurationSeconds + 59) / 60)
	if maxMinutes < 1 || maxMinutes > 60 || minutes < 1 || minutes > maxMinutes {
		return time.Time{}, &requestError{status: http.StatusBadRequest, message: "approval duration exceeds guest request"}
	}
	tok, _ := auth.Token()
	previousReply := x.Reply
	previousTokenHash := x.TokenHash
	x.TokenHash = auth.Hash(tok)
	x.Reply.Status = "approved"
	x.Reply.SessionToken = tok
	x.Reply.ExpiresAt = time.Now().UTC().Add(time.Duration(minutes) * time.Minute)
	initial := protocol.Command{ID: "initial-system-info", Name: "system.info", Deadline: time.Now().Add(15 * time.Second)}
	previousInitial, hadInitial := x.Commands[initial.ID]
	previousQueue := append([]string(nil), x.Queue...)
	x.Commands[initial.ID] = newCommandState(initial)
	x.Queue = append(x.Queue, initial.ID)
	if err := s.state.save(s.p); err != nil {
		x.Reply = previousReply
		x.TokenHash = previousTokenHash
		x.Queue = previousQueue
		if hadInitial {
			x.Commands[initial.ID] = previousInitial
		} else {
			delete(x.Commands, initial.ID)
		}
		log.Printf("persist approval: %v", err)
		return time.Time{}, &requestError{status: http.StatusInternalServerError, message: "state unavailable"}
	}
	return x.Reply.ExpiresAt, nil
}

func (s *server) rejectRequest(requestID string) *requestError {
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.p[requestID]
	if x == nil || x.Reply.Status != "pending" {
		return &requestError{status: http.StatusNotFound, message: "pending request not found"}
	}
	previousReply := x.Reply
	x.Reply.Status = "rejected"
	if err := s.state.save(s.p); err != nil {
		x.Reply = previousReply
		return &requestError{status: http.StatusInternalServerError, message: "state unavailable"}
	}
	s.audit.Event("pair_rejected", map[string]any{"requestId": requestID})
	return nil
}
func (s *server) isAdmin(r *http.Request) bool {
	return subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.admin)) == 1
}
func (s *server) enqueue(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		write(w, 401, nil)
		return
	}
	var q struct {
		RequestID string           `json:"requestId"`
		Command   protocol.Command `json:"command"`
	}
	if !limitedJSONN(w, r, &q, 2<<20) {
		return
	}
	maxParams := 32 << 10
	if q.Command.Name == "files.write-chunk" {
		maxParams = 2 << 20
	}
	if q.Command.ID == "" || !contains(s.capabilities(q.RequestID), q.Command.Name) || len(q.Command.Params) > maxParams {
		write(w, 403, map[string]string{"error": "command not authorized"})
		return
	}
	if q.Command.Deadline.IsZero() {
		q.Command.Deadline = time.Now().Add(30 * time.Second)
	}
	if q.Command.Deadline.After(time.Now().Add(time.Hour)) {
		write(w, 400, nil)
		return
	}
	s.mu.Lock()
	x := s.p[q.RequestID]
	if x == nil || x.Reply.Status != "approved" {
		s.mu.Unlock()
		write(w, 409, map[string]any{"error": "session not active"})
		return
	}
	state := newCommandState(q.Command)
	if existing := x.Commands[q.Command.ID]; existing != nil {
		if existing.Fingerprint == state.Fingerprint {
			status := existing.Status
			s.mu.Unlock()
			write(w, 200, map[string]any{"status": status, "commandId": q.Command.ID, "idempotent": true})
			return
		}
		s.mu.Unlock()
		write(w, 409, map[string]any{"error": "command ID already used with different payload", "commandId": q.Command.ID})
		return
	}
	regularDepth, controlDepth := queueDepths(x)
	isControl := isControlCommand(q.Command.Name)
	if (!isControl && regularDepth >= 16) || (isControl && controlDepth >= 4) {
		depth := len(x.Queue)
		limit := 16
		if isControl {
			limit = 4
		}
		s.mu.Unlock()
		w.Header().Set("Retry-After", "1")
		write(w, 409, map[string]any{"error": "queue full", "queueDepth": depth, "queueLimit": limit, "retryAfterSeconds": 1})
		return
	}
	x.Commands[q.Command.ID] = state
	previousQueue := append([]string(nil), x.Queue...)
	if isControl {
		x.Queue = append([]string{q.Command.ID}, x.Queue...)
	} else {
		x.Queue = append(x.Queue, q.Command.ID)
	}
	depth := len(x.Queue)
	if err := s.state.save(s.p); err != nil {
		delete(x.Commands, q.Command.ID)
		x.Queue = previousQueue
		s.mu.Unlock()
		log.Printf("persist command: %v", err)
		write(w, 500, map[string]string{"error": "state unavailable"})
		return
	}
	s.mu.Unlock()
	s.audit.Event("command_queued", map[string]any{"requestId": q.RequestID, "commandId": q.Command.ID, "name": q.Command.Name})
	limit := 16
	if isControl {
		limit = 4
	}
	write(w, 202, map[string]any{"status": "queued", "commandId": q.Command.ID, "queueDepth": depth, "queueLimit": limit})
}

func isControlCommand(name string) bool {
	return name == "shell.cancel" || name == "process.stop-owned" || name == "session.disconnect"
}

func queueDepths(item *pending) (regular, control int) {
	for _, id := range item.Queue {
		state := item.Commands[id]
		if state == nil || state.Status != "queued" {
			continue
		}
		if isControlCommand(state.Command.Name) {
			control++
		} else {
			regular++
		}
	}
	return regular, control
}

func newCommandState(cmd protocol.Command) *commandState {
	b, _ := json.Marshal(struct {
		Name   string          `json:"name"`
		Params json.RawMessage `json:"params,omitempty"`
	}{cmd.Name, cmd.Params})
	return &commandState{Command: cmd, Fingerprint: sha256.Sum256(b), Status: "queued"}
}
func (s *server) capabilities(id string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if x := s.p[id]; x != nil {
		return append([]string(nil), x.Req.Requested...)
	}
	return nil
}
func (s *server) adminRevoke(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		write(w, 401, nil)
		return
	}
	var q struct {
		RequestID string `json:"requestId"`
	}
	if !limitedJSON(w, r, &q) {
		return
	}
	s.mu.Lock()
	x := s.p[q.RequestID]
	if x == nil {
		s.mu.Unlock()
		write(w, 404, nil)
		return
	}
	x.Reply.Status = "revoked"
	x.TokenHash = ""
	x.Queue = nil
	if err := s.state.save(s.p); err != nil {
		s.mu.Unlock()
		log.Printf("persist revocation: %v", err)
		write(w, 500, map[string]string{"error": "state unavailable"})
		return
	}
	s.mu.Unlock()
	s.audit.Event("session_admin_revoked", map[string]any{"requestId": q.RequestID})
	write(w, 200, map[string]string{"status": "revoked"})
}

func (s *server) adminPending(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		write(w, 401, nil)
		return
	}
	type pendingView struct {
		RequestID       string    `json:"requestId"`
		USBID           string    `json:"usbId"`
		Hostname        string    `json:"hostname"`
		OS              string    `json:"os"`
		Arch            string    `json:"arch"`
		User            string    `json:"user"`
		Requested       []string  `json:"requested"`
		DurationSeconds int64     `json:"durationSeconds"`
		CompareCode     string    `json:"compareCode"`
		CreatedAt       time.Time `json:"createdAt"`
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]pendingView, 0)
	for id, item := range s.p {
		if item.Reply.Status == "pending" {
			items = append(items, pendingView{
				RequestID: id, USBID: item.Req.USBID, Hostname: item.Req.Hostname,
				OS: item.Req.OS, Arch: item.Req.Arch, User: item.Req.User,
				Requested:       append([]string(nil), item.Req.Requested...),
				DurationSeconds: item.Req.DurationSeconds, CompareCode: item.Reply.CompareCode,
				CreatedAt: item.CreatedAt,
			})
		}
	}
	write(w, 200, items)
}

func (s *server) adminReject(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		write(w, 401, nil)
		return
	}
	var q struct {
		RequestID string `json:"requestId"`
	}
	if !limitedJSON(w, r, &q) {
		return
	}
	if err := s.rejectRequest(q.RequestID); err != nil {
		write(w, err.status, map[string]string{"error": err.message})
		return
	}
	write(w, 200, map[string]string{"status": "rejected", "requestId": q.RequestID})
}
func (s *server) adminResults(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		write(w, 401, nil)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.p[r.URL.Query().Get("id")]
	if x == nil {
		write(w, 404, nil)
		return
	}
	results := append([]protocol.Result(nil), x.Results...)
	if r.URL.Query().Get("consume") == "true" {
		x.Results = nil
		if err := s.state.save(s.p); err != nil {
			log.Printf("persist consumed results: %v", err)
			write(w, 500, map[string]string{"error": "state unavailable"})
			return
		}
	}
	write(w, 200, results)
}
func (s *server) session(w http.ResponseWriter, r *http.Request) (*pending, bool) {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.p {
		if x.TokenHash != "" && subtle.ConstantTimeCompare([]byte(x.TokenHash), []byte(auth.Hash(tok))) == 1 && x.Reply.Status == "approved" && time.Now().Before(x.Reply.ExpiresAt) {
			return x, true
		}
	}
	write(w, 401, nil)
	return nil, false
}
func (s *server) poll(w http.ResponseWriter, r *http.Request) {
	x, ok := s.session(w, r)
	if !ok {
		return
	}
	deadline := time.Now().Add(25 * time.Second)
	for {
		s.mu.Lock()
		now := time.Now()
		for id, state := range x.Commands {
			if state.Status == "leased" && now.After(state.LeaseUntil) {
				state.Status = "queued"
				state.LeaseUntil = time.Time{}
				x.Queue = append(x.Queue, id)
			}
		}
		if len(x.Queue) > 0 {
			id := x.Queue[0]
			x.Queue = x.Queue[1:]
			state := x.Commands[id]
			if state == nil || state.Status != "queued" {
				s.mu.Unlock()
				continue
			}
			state.Status = "leased"
			state.LeaseUntil = time.Now().Add(10 * time.Second)
			cmd := state.Command
			if err := s.state.save(s.p); err != nil {
				s.mu.Unlock()
				log.Printf("persist command lease: %v", err)
				write(w, 500, map[string]string{"error": "state unavailable"})
				return
			}
			s.mu.Unlock()
			write(w, 200, cmd)
			return
		}
		active := x.Reply.Status == "approved" && time.Now().Before(x.Reply.ExpiresAt)
		s.mu.Unlock()
		if !active || time.Now().After(deadline) {
			w.WriteHeader(204)
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (s *server) ack(w http.ResponseWriter, r *http.Request) {
	x, ok := s.session(w, r)
	if !ok {
		return
	}
	var q struct {
		CommandID string `json:"commandId"`
	}
	if !limitedJSON(w, r, &q) {
		return
	}
	if q.CommandID == "" {
		write(w, 400, map[string]string{"error": "commandId is required"})
		return
	}
	s.mu.Lock()
	state := x.Commands[q.CommandID]
	if state == nil || state.Status != "leased" || time.Now().After(state.LeaseUntil) {
		s.mu.Unlock()
		write(w, 409, map[string]any{"error": "command lease is not active", "commandId": q.CommandID})
		return
	}
	state.Status = "running"
	state.LeaseUntil = time.Time{}
	if err := s.state.save(s.p); err != nil {
		s.mu.Unlock()
		log.Printf("persist command acknowledgement: %v", err)
		write(w, 500, map[string]string{"error": "state unavailable"})
		return
	}
	s.mu.Unlock()
	s.audit.Event("command_acknowledged", map[string]any{"commandId": q.CommandID})
	write(w, 200, map[string]any{"status": "running", "commandId": q.CommandID})
}
func (s *server) result(w http.ResponseWriter, r *http.Request) {
	x, ok := s.session(w, r)
	if !ok {
		return
	}
	_ = x
	var q protocol.Result
	if !limitedJSONN(w, r, &q, 3<<20) {
		return
	}
	if q.ID == "" || !contains(x.Req.Requested, q.Name) || len(q.Output) > (2<<20) {
		write(w, 400, nil)
		return
	}
	s.mu.Lock()
	state := x.Commands[q.ID]
	if state == nil || state.Command.Name != q.Name || state.Status != "running" {
		s.mu.Unlock()
		write(w, 409, map[string]any{"error": "result does not match a running command", "commandId": q.ID})
		return
	}
	state.Status = "completed"
	if len(x.Results) < 128 {
		x.Results = append(x.Results, q)
	}
	if err := s.state.save(s.p); err != nil {
		s.mu.Unlock()
		log.Printf("persist command result: %v", err)
		write(w, 500, map[string]string{"error": "state unavailable"})
		return
	}
	s.mu.Unlock()
	s.audit.Event("command_result", map[string]any{"commandId": q.ID, "name": q.Name, "error": q.Error})
	write(w, 200, map[string]string{"status": "accepted"})
}
func validCapabilities(v []string) bool {
	// Keep a small amount of headroom for additive protocol capabilities while
	// still bounding pairing requests. The Developer profile currently uses 21.
	if len(v) == 0 || len(v) > 32 {
		return false
	}
	allowed := map[string]bool{"system.info": true, "system.network": true, "disk.list": true, "service.list": true, "process.list": true, "process.start": true, "process.stop-owned": true, "shell.run": true, "shell.run-admin": true, "powershell.run": true, "shell.start": true, "shell.status": true, "shell.cancel": true, "files.list": true, "files.read": true, "files.read-chunk": true, "files.write": true, "files.write-chunk": true, "files.upload": true, "files.download": true, "session.disconnect": true}
	seen := map[string]bool{}
	for _, x := range v {
		if !allowed[x] || seen[x] {
			return false
		}
		seen[x] = true
	}
	return true
}
func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}
func (s *server) revoke(w http.ResponseWriter, r *http.Request) {
	x, ok := s.session(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	x.Reply.Status = "revoked"
	x.TokenHash = ""
	if err := s.state.save(s.p); err != nil {
		s.mu.Unlock()
		log.Printf("persist client revocation: %v", err)
		write(w, 500, map[string]string{"error": "state unavailable"})
		return
	}
	s.mu.Unlock()
	s.audit.Event("session_revoked", map[string]any{})
	write(w, 200, map[string]string{"status": "revoked"})
}

func (s *server) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for now := range ticker.C {
		s.mu.Lock()
		changed := false
		for id, x := range s.p {
			pendingExpired := x.Reply.Status == "pending" && now.Sub(x.CreatedAt) > 10*time.Minute
			sessionGone := (x.Reply.Status == "revoked" || x.Reply.Status == "rejected" || (!x.Reply.ExpiresAt.IsZero() && now.After(x.Reply.ExpiresAt))) && now.Sub(x.CreatedAt) > 10*time.Minute
			if pendingExpired || sessionGone {
				delete(s.p, id)
				changed = true
			}
		}
		if changed {
			if err := s.state.save(s.p); err != nil {
				log.Printf("persist janitor cleanup: %v", err)
			}
		}
		s.mu.Unlock()
	}
}
func main() {
	listen := flag.String("listen", "127.0.0.1:17443", "")
	logPath := flag.String("audit", "broker-audit.jsonl", "")
	statePath := flag.String("state", "broker-state.json", "")
	flag.Parse()
	admin := os.Getenv("BRIDGE_ADMIN_TOKEN")
	if len(admin) < 24 {
		log.Fatal("BRIDGE_ADMIN_TOKEN must be at least 24 characters")
	}
	a, e := audit.Open(*logPath)
	if e != nil {
		log.Fatal(e)
	}
	defer a.Close()
	store, entries, e := openStateStore(*statePath, time.Now().UTC())
	if e != nil {
		log.Fatal(e)
	}
	if e := store.save(entries); e != nil {
		log.Fatal(e)
	}
	s := &server{p: entries, audit: a, admin: admin, seen: map[string]time.Time{}, rates: map[string][]time.Time{}, state: store}
	go s.janitor()
	http.HandleFunc("/v1/pair/request", s.pair)
	http.HandleFunc("/v1/pair/status", s.status)
	http.HandleFunc("/v1/admin/approve", s.approve)
	http.HandleFunc("/v1/admin/pending", s.adminPending)
	http.HandleFunc("/v1/admin/reject", s.adminReject)
	http.HandleFunc("/v1/admin/command", s.enqueue)
	http.HandleFunc("/v1/admin/revoke", s.adminRevoke)
	http.HandleFunc("/v1/admin/results", s.adminResults)
	http.HandleFunc("/v1/session/poll", s.poll)
	http.HandleFunc("/v1/session/ack", s.ack)
	http.HandleFunc("/v1/session/result", s.result)
	http.HandleFunc("/v1/session/revoke", s.revoke)
	log.Printf("PoC broker listening on %s (loopback HTTP only)", *listen)
	httpServer := &http.Server{
		Addr: *listen, Handler: nil,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       40 * time.Second,
		WriteTimeout:      40 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(httpServer.ListenAndServe())
}
