package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateVersion = 1

type persistentState struct {
	Version int                 `json:"version"`
	Pending map[string]*pending `json:"pending"`
}

type stateStore struct {
	path string
}

func openStateStore(path string, now time.Time) (*stateStore, map[string]*pending, error) {
	store := &stateStore{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, map[string]*pending{}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read broker state: %w", err)
	}
	var disk persistentState
	if err := json.Unmarshal(data, &disk); err != nil {
		return nil, nil, fmt.Errorf("parse broker state: %w", err)
	}
	if disk.Version != stateVersion || disk.Pending == nil {
		return nil, nil, fmt.Errorf("unsupported broker state version %d", disk.Version)
	}
	recoverState(disk.Pending, now)
	return store, disk.Pending, nil
}

func recoverState(entries map[string]*pending, now time.Time) {
	for id, item := range entries {
		if item == nil || item.Commands == nil {
			delete(entries, id)
			continue
		}
		pendingExpired := item.Reply.Status == "pending" && now.Sub(item.CreatedAt) > 10*time.Minute
		sessionExpired := item.Reply.Status == "revoked" || (!item.Reply.ExpiresAt.IsZero() && !now.Before(item.Reply.ExpiresAt))
		if pendingExpired || sessionExpired {
			delete(entries, id)
			continue
		}
		queued := make([]string, 0, len(item.Commands))
		alreadyQueued := make(map[string]bool, len(item.Queue))
		for _, commandID := range item.Queue {
			state := item.Commands[commandID]
			if state != nil && state.Status == "queued" && !alreadyQueued[commandID] {
				queued = append(queued, commandID)
				alreadyQueued[commandID] = true
			}
		}
		for commandID, state := range item.Commands {
			if state.Status == "leased" {
				state.Status = "queued"
				state.LeaseUntil = time.Time{}
			}
			if state.Status == "queued" && !alreadyQueued[commandID] {
				queued = append(queued, commandID)
			}
		}
		item.Queue = queued
	}
}

func (s *stateStore) save(entries map[string]*pending) error {
	if s == nil || s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(persistentState{Version: stateVersion, Pending: entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode broker state: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create broker state directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".broker-state-*")
	if err != nil {
		return fmt.Errorf("create broker state temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write broker state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync broker state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close broker state: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("publish broker state: %w", err)
	}
	if directory, err := os.Open(dir); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}
