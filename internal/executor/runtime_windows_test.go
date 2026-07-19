//go:build windows

package executor

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
)

func TestPowerShellRunPreservesUnicode(t *testing.T) {
	e, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	params, _ := json.Marshal(map[string]string{"script": "Write-Output 'Sì — èàòù ✓'"})
	out, err := e.Execute(protocol.Command{Name: "powershell.run", Params: params, Deadline: time.Now().Add(20 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Sì — èàòù ✓") {
		t.Fatalf("Unicode output was corrupted: %q", out)
	}
}

func TestWindowsJobTerminatesAssignedProcess(t *testing.T) {
	j, err := newWindowsJob()
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	cmd := exec.Command("cmd.exe", "/d", "/c", "ping -n 30 127.0.0.1 >nul")
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err = j.Assign(cmd.Process); err != nil {
		_ = cmd.Process.Kill()
		t.Fatal(err)
	}
	if err = j.Terminate(125); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("assigned process survived job termination")
	}
}

func TestPseudoConsoleLifecycle(t *testing.T) {
	p, err := newPseudoConsole(80, 25)
	if err != nil {
		t.Fatal(err)
	}
	if p.Handle() == 0 || p.Input == nil || p.Output == nil {
		t.Fatal("incomplete pseudo console")
	}
	if err = p.Resize(120, 40); err != nil {
		t.Fatal(err)
	}
	if err = p.Close(); err != nil {
		t.Fatal(err)
	}
	if p.Handle() != 0 {
		t.Fatal("pseudo console handle remains open")
	}
	if err = p.Resize(80, 25); err == nil {
		t.Fatal("resize accepted after close")
	}
}
