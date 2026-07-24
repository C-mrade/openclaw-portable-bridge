package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
)

type telegramApproval struct {
	token      string
	approverID int64
	apiBase    string
	client     *http.Client
	server     *server
}

type telegramResponse[T any] struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      T      `json:"result"`
}

type telegramUpdate struct {
	UpdateID      int64 `json:"update_id"`
	CallbackQuery *struct {
		ID   string `json:"id"`
		Data string `json:"data"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
	} `json:"callback_query"`
}

func telegramFromEnvironment(server *server) (*telegramApproval, error) {
	token := strings.TrimSpace(os.Getenv("BRIDGE_TELEGRAM_BOT_TOKEN"))
	approver := strings.TrimSpace(os.Getenv("BRIDGE_TELEGRAM_APPROVER_ID"))
	if token == "" && approver == "" {
		return nil, nil
	}
	if token == "" || approver == "" {
		return nil, errors.New("BRIDGE_TELEGRAM_BOT_TOKEN and BRIDGE_TELEGRAM_APPROVER_ID must be set together")
	}
	approverID, err := strconv.ParseInt(approver, 10, 64)
	if err != nil || approverID <= 0 {
		return nil, errors.New("BRIDGE_TELEGRAM_APPROVER_ID must be a positive numeric Telegram user ID")
	}
	return &telegramApproval{
		token: token, approverID: approverID, apiBase: "https://api.telegram.org",
		client: &http.Client{Timeout: 40 * time.Second}, server: server,
	}, nil
}

func (t *telegramApproval) PairRequested(id string, request protocol.PairRequest, reply protocol.PairReply) {
	go func() {
		minutes := (request.DurationSeconds + 59) / 60
		text := fmt.Sprintf(
			"OpenClaw Bridge approval\n\nDevice: %s\nHost: %s\nPlatform: %s/%s\nUser label: %s\nDuration: %d min\nComparison code: %s\n\nCapabilities:\n%s",
			request.USBID, request.Hostname, request.OS, request.Arch, request.User,
			minutes, reply.CompareCode, strings.Join(request.Requested, "\n"),
		)
		payload := map[string]any{
			"chat_id": t.approverID,
			"text":    text,
			"reply_markup": map[string]any{"inline_keyboard": [][]map[string]string{{
				{"text": "Approve", "callback_data": "bridge:a:" + id},
				{"text": "Reject", "callback_data": "bridge:r:" + id},
			}}},
		}
		if err := t.call(context.Background(), "sendMessage", payload, nil); err != nil {
			log.Printf("telegram approval notification: %v", err)
		}
	}()
}

func (t *telegramApproval) Run(ctx context.Context) {
	var offset int64
	for ctx.Err() == nil {
		var updates []telegramUpdate
		err := t.call(ctx, "getUpdates", map[string]any{
			"offset": offset, "timeout": 25, "allowed_updates": []string{"callback_query"},
		}, &updates)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("telegram approval polling: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}
		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			t.handleCallback(ctx, update.CallbackQuery)
		}
	}
}

func (t *telegramApproval) handleCallback(ctx context.Context, callback *struct {
	ID   string `json:"id"`
	Data string `json:"data"`
	From struct {
		ID int64 `json:"id"`
	} `json:"from"`
}) {
	if callback == nil {
		return
	}
	answer := "Not authorized"
	if callback.From.ID == t.approverID {
		parts := strings.Split(callback.Data, ":")
		if len(parts) == 3 && parts[0] == "bridge" {
			item := t.server.pendingSnapshot(parts[2])
			if item == nil {
				answer = "Request is no longer pending"
			} else if parts[1] == "a" {
				minutes := int((item.Req.DurationSeconds + 59) / 60)
				if err := t.server.approveRequest(parts[2], minutes); err != nil {
					answer = err.message
				} else {
					t.server.audit.Event("pair_approved", map[string]any{"requestId": parts[2], "minutes": minutes, "source": "telegram"})
					answer = "Session approved"
				}
			} else if parts[1] == "r" {
				if err := t.server.rejectRequest(parts[2]); err != nil {
					answer = err.message
				} else {
					answer = "Session rejected"
				}
			}
		}
	}
	_ = t.call(ctx, "answerCallbackQuery", map[string]any{
		"callback_query_id": callback.ID, "text": answer, "show_alert": true,
	}, nil)
}

func (s *server) pendingSnapshot(id string) *pending {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.p[id]
	if item == nil || item.Reply.Status != "pending" {
		return nil
	}
	copy := *item
	return &copy
}

func (t *telegramApproval) call(ctx context.Context, method string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.apiBase+"/bot"+t.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := t.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	var envelope telegramResponse[json.RawMessage]
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("telegram %s returned invalid JSON", method)
	}
	if !envelope.OK || response.StatusCode/100 != 2 {
		return fmt.Errorf("telegram %s failed: %s", method, envelope.Description)
	}
	if out != nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, out); err != nil {
			return err
		}
	}
	return nil
}
