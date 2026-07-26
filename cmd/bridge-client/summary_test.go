package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
)

func TestCommandSummaryShowsActionWithoutBulkPayload(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"path":       "/customer/report.txt",
		"dataBase64": strings.Repeat("A", 10000),
	})
	summary := commandSummary(protocol.Command{ID: "write", Name: "files.write", Params: params})
	if !strings.Contains(summary, "files.write") ||
		!strings.Contains(summary, "/customer/report.txt") ||
		strings.Contains(summary, strings.Repeat("A", 100)) {
		t.Fatalf("unsafe or incomplete activity summary: %s", summary)
	}
}

func TestPrintableRemovesTerminalControlsAndBoundsText(t *testing.T) {
	got := printable("safe\x00\x1b[31m\n"+strings.Repeat("x", 100), 20)
	if strings.ContainsAny(got, "\x00\x1b\n") || len(got) > 23 {
		t.Fatalf("unsafe printable text: %q", got)
	}
}
