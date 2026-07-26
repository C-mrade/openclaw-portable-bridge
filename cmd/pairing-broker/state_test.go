package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/C-mrade/openclaw-portable-bridge/internal/auth"
	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
)

func TestStateStoreRecoversLeasesWithoutReplayingRunningCommands(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "broker-state.json")
	store := &stateStore{path: path}
	entries := map[string]*pending{
		"request": {
			Reply:     protocol.PairReply{Status: "approved", SessionToken: "must-never-reach-disk", ExpiresAt: now.Add(time.Hour)},
			TokenHash: auth.Hash("session"),
			CreatedAt: now,
			Commands: map[string]*commandState{
				"queued":  {Command: protocol.Command{ID: "queued", Name: "system.info"}, Status: "queued"},
				"leased":  {Command: protocol.Command{ID: "leased", Name: "system.info"}, Status: "leased", LeaseUntil: now.Add(time.Minute)},
				"running": {Command: protocol.Command{ID: "running", Name: "system.info"}, Status: "running"},
			},
			Queue: []string{"queued"},
		},
	}
	if err := store.save(entries); err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(onDisk, []byte("must-never-reach-disk")) ||
		bytes.Contains(onDisk, []byte(`"SessionToken"`)) {
		t.Fatalf("clear session token persisted: %s", onDisk)
	}
	_, recovered, err := openStateStore(path, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	got := recovered["request"]
	if got.Reply.SessionToken != "" {
		t.Fatal("clear session token recovered into memory")
	}
	if got.Commands["leased"].Status != "queued" {
		t.Fatalf("leased command recovered as %q", got.Commands["leased"].Status)
	}
	if got.Commands["running"].Status != "running" {
		t.Fatalf("running command recovered as %q", got.Commands["running"].Status)
	}
	if len(got.Queue) != 2 || got.Queue[0] != "queued" || got.Queue[1] != "leased" {
		t.Fatalf("recovered queue = %#v", got.Queue)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state permissions = %o", info.Mode().Perm())
	}
}

func TestStateStoreDropsExpiredAndCorruptState(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "broker-state.json")
	store := &stateStore{path: path}
	entries := map[string]*pending{
		"expired": {
			Reply:     protocol.PairReply{Status: "approved", ExpiresAt: now.Add(-time.Second)},
			CreatedAt: now.Add(-time.Hour),
			Commands:  map[string]*commandState{},
		},
	}
	if err := store.save(entries); err != nil {
		t.Fatal(err)
	}
	_, recovered, err := openStateStore(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 0 {
		t.Fatalf("expired state recovered: %#v", recovered)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openStateStore(path, now); err == nil {
		t.Fatal("corrupt state accepted")
	}
}
