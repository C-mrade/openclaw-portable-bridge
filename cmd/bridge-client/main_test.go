package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
)

func TestPollResilientRetriesTransientFailures(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(protocol.Command{ID: "cmd-1", Name: "system.info"})
	}))
	defer srv.Close()

	cmd, err := pollResilient(srv.URL, "token", time.Now().Add(time.Second), time.Millisecond, 2*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.ID != "cmd-1" || calls.Load() != 3 {
		t.Fatalf("unexpected command/calls: %#v %d", cmd, calls.Load())
	}
}

func TestPollResilientStopsOnAuthoritativeRejection(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "revoked", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := pollResilient(srv.URL, "token", time.Now().Add(time.Second), time.Millisecond, 2*time.Millisecond)
	if err == nil || calls.Load() != 1 {
		t.Fatalf("expected one terminal failure, got err=%v calls=%d", err, calls.Load())
	}
}
