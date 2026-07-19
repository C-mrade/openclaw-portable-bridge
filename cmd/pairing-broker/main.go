package main

import (
	"crypto/subtle"
	"encoding/json"
	"flag"
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
	Queue            []protocol.Command
	Results          []protocol.Result
}
type server struct {
	mu    sync.Mutex
	p     map[string]*pending
	audit *audit.Logger
	admin string
	seen  map[string]time.Time
	rates map[string][]time.Time
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func limitedJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
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
	if q.Protocol != protocol.Version || q.USBID == "" || q.DurationSeconds < 60 || q.DurationSeconds > 3600 || !validCapabilities(q.Requested) || !auth.Verify(q.PublicKey, q.Signature, protocol.CanonicalPairRequest(q)) {
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
	s.mu.Lock()
	s.p[id] = &pending{Req: q, Reply: rep, PairingTokenHash: auth.Hash(pairingToken)}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.p[q.RequestID]
	if x == nil || x.Reply.Status != "pending" {
		write(w, 404, nil)
		return
	}
	tok, _ := auth.Token()
	x.TokenHash = auth.Hash(tok)
	x.Reply.Status = "approved"
	x.Reply.SessionToken = tok
	x.Reply.ExpiresAt = time.Now().UTC().Add(time.Duration(q.Minutes) * time.Minute)
	x.Queue = append(x.Queue, protocol.Command{ID: "initial-system-info", Name: "system.info", Deadline: time.Now().Add(15 * time.Second)})
	s.audit.Event("pair_approved", map[string]any{"requestId": q.RequestID, "minutes": q.Minutes})
	write(w, 200, map[string]string{"status": "approved"})
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
	if !limitedJSON(w, r, &q) {
		return
	}
	if q.Command.ID == "" || !contains(s.capabilities(q.RequestID), q.Command.Name) || len(q.Command.Params) > (32<<10) {
		write(w, 403, map[string]string{"error": "command not authorized"})
		return
	}
	if q.Command.Deadline.IsZero() {
		q.Command.Deadline = time.Now().Add(30 * time.Second)
	}
	if q.Command.Deadline.After(time.Now().Add(2 * time.Minute)) {
		write(w, 400, nil)
		return
	}
	s.mu.Lock()
	x := s.p[q.RequestID]
	if x == nil || x.Reply.Status != "approved" || len(x.Queue) >= 16 {
		s.mu.Unlock()
		write(w, 409, nil)
		return
	}
	x.Queue = append(x.Queue, q.Command)
	s.mu.Unlock()
	s.audit.Event("command_queued", map[string]any{"requestId": q.RequestID, "commandId": q.Command.ID, "name": q.Command.Name})
	write(w, 202, map[string]string{"status": "queued"})
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
	s.mu.Unlock()
	s.audit.Event("session_admin_revoked", map[string]any{"requestId": q.RequestID})
	write(w, 200, map[string]string{"status": "revoked"})
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
	write(w, 200, x.Results)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(x.Queue) == 0 {
		w.WriteHeader(204)
		return
	}
	cmd := x.Queue[0]
	x.Queue = x.Queue[1:]
	write(w, 200, cmd)
}
func (s *server) result(w http.ResponseWriter, r *http.Request) {
	x, ok := s.session(w, r)
	if !ok {
		return
	}
	_ = x
	var q protocol.Result
	if !limitedJSON(w, r, &q) {
		return
	}
	if !contains(x.Req.Requested, q.Name) || len(q.Output) > (2<<20) {
		write(w, 400, nil)
		return
	}
	s.mu.Lock()
	if len(x.Results) < 128 {
		x.Results = append(x.Results, q)
	}
	s.mu.Unlock()
	s.audit.Event("command_result", map[string]any{"commandId": q.ID, "name": q.Name, "error": q.Error})
	write(w, 200, map[string]string{"status": "accepted"})
}
func validCapabilities(v []string) bool {
	if len(v) == 0 || len(v) > 15 {
		return false
	}
	allowed := map[string]bool{"system.info": true, "system.network": true, "disk.list": true, "service.list": true, "process.list": true, "process.start": true, "process.stop-owned": true, "shell.run": true, "shell.run-admin": true, "files.list": true, "files.read": true, "files.write": true, "files.upload": true, "files.download": true, "session.disconnect": true}
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
	s.mu.Unlock()
	s.audit.Event("session_revoked", map[string]any{})
	write(w, 200, map[string]string{"status": "revoked"})
}
func main() {
	listen := flag.String("listen", "127.0.0.1:17443", "")
	logPath := flag.String("audit", "broker-audit.jsonl", "")
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
	s := &server{p: map[string]*pending{}, audit: a, admin: admin, seen: map[string]time.Time{}, rates: map[string][]time.Time{}}
	http.HandleFunc("/v1/pair/request", s.pair)
	http.HandleFunc("/v1/pair/status", s.status)
	http.HandleFunc("/v1/admin/approve", s.approve)
	http.HandleFunc("/v1/admin/command", s.enqueue)
	http.HandleFunc("/v1/admin/revoke", s.adminRevoke)
	http.HandleFunc("/v1/admin/results", s.adminResults)
	http.HandleFunc("/v1/session/poll", s.poll)
	http.HandleFunc("/v1/session/result", s.result)
	http.HandleFunc("/v1/session/revoke", s.revoke)
	log.Printf("PoC broker listening on %s (loopback HTTP only)", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}
