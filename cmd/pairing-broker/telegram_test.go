package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTelegramApprovalRequiresConfiguredApprover(t *testing.T) {
	t.Setenv("BRIDGE_TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("BRIDGE_TELEGRAM_APPROVER_ID", "")
	if _, err := telegramFromEnvironment(nil); err == nil {
		t.Fatal("partial Telegram configuration accepted")
	}
	t.Setenv("BRIDGE_TELEGRAM_APPROVER_ID", "not-numeric")
	if _, err := telegramFromEnvironment(nil); err == nil {
		t.Fatal("invalid Telegram approver accepted")
	}
}

func TestTelegramCallbackApprovesOnlyAllowlistedUser(t *testing.T) {
	s, _ := testServer(t)
	item := s.p["request"]
	item.Reply.Status = "pending"
	item.Reply.ExpiresAt = time.Time{}
	item.Req.DurationSeconds = 120
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer api.Close()
	telegram := &telegramApproval{
		token: "token", approverID: 42, apiBase: api.URL,
		client: api.Client(), server: s,
	}

	var unauthorized telegramUpdate
	if err := json.Unmarshal([]byte(`{"callback_query":{"id":"one","data":"bridge:a:request","from":{"id":7}}}`), &unauthorized); err != nil {
		t.Fatal(err)
	}
	telegram.handleCallback(context.Background(), unauthorized.CallbackQuery)
	if item.Reply.Status != "pending" {
		t.Fatal("unauthorized Telegram user approved the session")
	}

	var authorized telegramUpdate
	if err := json.Unmarshal([]byte(`{"callback_query":{"id":"two","data":"bridge:a:request","from":{"id":42}}}`), &authorized); err != nil {
		t.Fatal(err)
	}
	telegram.handleCallback(context.Background(), authorized.CallbackQuery)
	if item.Reply.Status != "approved" || item.Reply.SessionToken == "" {
		t.Fatalf("allowlisted approval failed: %#v", item.Reply)
	}
}
