package executor

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

const MaxOutput = 1 << 20
const MaxTransfer = 10 << 20

type Executor struct {
	roots []string
	mu    sync.Mutex
	owned map[int]*exec.Cmd
}

func New(roots []string) (*Executor, error) {
	e := &Executor{owned: map[int]*exec.Cmd{}}
	for _, r := range roots {
		a, x := filepath.Abs(r)
		if x != nil {
			return nil, x
		}
		a, x = filepath.EvalSymlinks(a)
		if x != nil {
			return nil, fmt.Errorf("allowed directory %q: %w", r, x)
		}
		e.roots = append(e.roots, filepath.Clean(a))
	}
	return e, nil
}
func (e *Executor) resolve(path string, forWrite bool) (string, error) {
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return "", errors.New("invalid path")
	}
	a, x := filepath.Abs(path)
	if x != nil {
		return "", x
	}
	probe := a
	if forWrite {
		probe = filepath.Dir(a)
	}
	canonical, x := filepath.EvalSymlinks(probe)
	if x != nil {
		return "", x
	}
	if forWrite {
		a = filepath.Join(canonical, filepath.Base(a))
	} else {
		a = canonical
	}
	for _, root := range e.roots {
		rel, x := filepath.Rel(root, a)
		if x == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return a, nil
		}
	}
	return "", errors.New("path outside approved directories")
}
func decode(raw json.RawMessage, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

type capped struct {
	b bytes.Buffer
	n int
}

func (c *capped) Write(p []byte) (int, error) {
	n := len(p)
	remain := MaxOutput - c.n
	if remain > 0 {
		if len(p) > remain {
			p = p[:remain]
		}
		_, _ = c.b.Write(p)
		c.n += len(p)
	}
	return n, nil
}
func (e *Executor) Execute(cmd protocol.Command) (string, error) {
	switch cmd.Name {
	case "system.info":
		h, _ := os.Hostname()
		return marshal(map[string]any{"hostname": h, "os": runtime.GOOS, "arch": runtime.GOARCH, "goVersion": runtime.Version()})
	case "system.network":
		if runtime.GOOS == "windows" {
			return e.readOnlyCommand(cmd, []string{"ipconfig.exe", "/all"})
		}
		return e.readOnlyCommand(cmd, []string{"ip", "address"})
	case "disk.list":
		if runtime.GOOS == "windows" {
			return e.readOnlyCommand(cmd, []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-CimInstance Win32_LogicalDisk | Select-Object DeviceID,VolumeName,DriveType,FileSystem,Size,FreeSpace | ConvertTo-Json -Compress"})
		}
		return e.readOnlyCommand(cmd, []string{"df", "-P"})
	case "service.list":
		if runtime.GOOS == "windows" {
			return e.readOnlyCommand(cmd, []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-Service | Select-Object Name,DisplayName,Status,StartType | ConvertTo-Json -Compress"})
		}
		return e.readOnlyCommand(cmd, []string{"systemctl", "list-units", "--type=service", "--all", "--no-pager", "--no-legend"})
	case "process.list":
		return e.processList(cmd)
	case "process.start":
		return e.processStart(cmd)
	case "process.stop-owned":
		return e.processStop(cmd)
	case "shell.run":
		return e.shell(cmd)
	case "shell.run-admin":
		return e.shellAdmin(cmd)
	case "files.list":
		return e.filesList(cmd)
	case "files.read", "files.download":
		return e.filesRead(cmd)
	case "files.write", "files.upload":
		return e.filesWrite(cmd)
	case "session.disconnect":
		return marshal(map[string]bool{"disconnect": true})
	default:
		return "", errors.New("unauthorized or unknown command")
	}
}

// readOnlyCommand executes a fixed, code-owned inspection command. Command names
// and arguments never come from the remote request, so this remains read-only.
func (e *Executor) readOnlyCommand(c protocol.Command, argv []string) (string, error) {
	if len(c.Params) != 0 && string(c.Params) != "null" && string(c.Params) != "{}" {
		return "", errors.New("inspection command does not accept parameters")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout(c))
	defer cancel()
	x := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var out capped
	x.Stdout = &out
	x.Stderr = &out
	err := x.Run()
	result := map[string]any{"output": out.b.String(), "truncated": out.n >= MaxOutput, "exitCode": exitCode(err)}
	if ctx.Err() != nil {
		return marshalWithError(result, errors.New("inspection timeout"))
	}
	return marshalWithError(result, err)
}
func timeout(c protocol.Command) time.Duration {
	d := 30 * time.Second
	if !c.Deadline.IsZero() {
		d = time.Until(c.Deadline)
		if d <= 0 {
			return time.Nanosecond
		}
		if d > 2*time.Minute {
			d = 2 * time.Minute
		}
	}
	return d
}
func (e *Executor) shell(c protocol.Command) (string, error) {
	var p struct {
		Command string `json:"command"`
	}
	if decode(c.Params, &p) != nil || p.Command == "" || len(p.Command) > 8192 || strings.IndexByte(p.Command, 0) >= 0 {
		return "", errors.New("invalid shell command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout(c))
	defer cancel()
	var x *exec.Cmd
	if runtime.GOOS == "windows" {
		x = exec.CommandContext(ctx, "cmd.exe", "/d", "/s", "/c", p.Command)
	} else {
		x = exec.CommandContext(ctx, "/bin/sh", "-c", p.Command)
	}
	var out capped
	x.Stdout = &out
	x.Stderr = &out
	err := x.Run()
	result := map[string]any{"output": out.b.String(), "truncated": out.n >= MaxOutput, "exitCode": exitCode(err)}
	if ctx.Err() != nil {
		return marshalWithError(result, errors.New("command timeout"))
	}
	return marshalWithError(result, err)
}

func (e *Executor) shellAdmin(c protocol.Command) (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("administrator approval is currently supported only on Windows")
	}
	var p struct {
		Command string `json:"command"`
	}
	if decode(c.Params, &p) != nil || p.Command == "" || len(p.Command) > 8192 || strings.IndexByte(p.Command, 0) >= 0 {
		return "", errors.New("invalid administrator command")
	}
	outFile, err := os.CreateTemp("", "OpenClawBridge-admin-*.log")
	if err != nil {
		return "", err
	}
	outPath := outFile.Name()
	_ = outFile.Close()
	defer os.Remove(outPath)
	command64 := base64.StdEncoding.EncodeToString([]byte(p.Command))
	quotedOut := strings.ReplaceAll(outPath, "'", "''")
	script := fmt.Sprintf("$cmd=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('%s')); & cmd.exe /d /s /c $cmd *> '%s'; exit $LASTEXITCODE", command64, quotedOut)
	encodedScript := encodePowerShell(script)
	launcher := fmt.Sprintf("$p=Start-Process -FilePath 'powershell.exe' -Verb RunAs -ArgumentList '-NoProfile','-NonInteractive','-EncodedCommand','%s' -Wait -PassThru; exit $p.ExitCode", encodedScript)
	ctx, cancel := context.WithTimeout(context.Background(), timeout(c))
	defer cancel()
	x := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", launcher)
	var parentOut capped
	x.Stdout = &parentOut
	x.Stderr = &parentOut
	err = x.Run()
	resultOut := parentOut.b.String()
	if f, openErr := os.Open(outPath); openErr == nil {
		defer f.Close()
		data, _ := io.ReadAll(io.LimitReader(f, MaxOutput+1))
		resultOut += string(data[:min(len(data), MaxOutput)])
	}
	result := map[string]any{"output": resultOut, "truncated": len(resultOut) > MaxOutput, "exitCode": exitCode(err), "uacRequired": true}
	if ctx.Err() != nil {
		return marshalWithError(result, errors.New("administrator command timeout or UAC not approved"))
	}
	return marshalWithError(result, err)
}

func encodePowerShell(s string) string {
	words := utf16.Encode([]rune(s))
	b := make([]byte, len(words)*2)
	for i, word := range words {
		b[i*2] = byte(word)
		b[i*2+1] = byte(word >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}
func exitCode(e error) int {
	if e == nil {
		return 0
	}
	var x *exec.ExitError
	if errors.As(e, &x) {
		return x.ExitCode()
	}
	return -1
}
func (e *Executor) processList(c protocol.Command) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout(c))
	defer cancel()
	var x *exec.Cmd
	if runtime.GOOS == "windows" {
		x = exec.CommandContext(ctx, "tasklist.exe", "/FO", "CSV", "/NH")
	} else {
		x = exec.CommandContext(ctx, "ps", "-eo", "pid=,comm=")
	}
	var out capped
	x.Stdout = &out
	x.Stderr = &out
	err := x.Run()
	return marshalWithError(map[string]any{"output": out.b.String(), "truncated": out.n >= MaxOutput}, err)
}
func (e *Executor) processStart(c protocol.Command) (string, error) {
	var p struct {
		Program string   `json:"program"`
		Args    []string `json:"args"`
	}
	if decode(c.Params, &p) != nil || p.Program == "" {
		return "", errors.New("invalid process request")
	}
	x := exec.Command(p.Program, p.Args...)
	if err := x.Start(); err != nil {
		return "", err
	}
	e.mu.Lock()
	e.owned[x.Process.Pid] = x
	e.mu.Unlock()
	go func() { _ = x.Wait(); e.mu.Lock(); delete(e.owned, x.Process.Pid); e.mu.Unlock() }()
	return marshal(map[string]int{"pid": x.Process.Pid})
}
func (e *Executor) processStop(c protocol.Command) (string, error) {
	var p struct {
		PID int `json:"pid"`
	}
	if decode(c.Params, &p) != nil {
		return "", errors.New("invalid pid")
	}
	e.mu.Lock()
	x := e.owned[p.PID]
	e.mu.Unlock()
	if x == nil {
		return "", errors.New("process is not owned by bridge")
	}
	return marshalWithError(map[string]int{"pid": p.PID}, x.Process.Kill())
}
func (e *Executor) filesList(c protocol.Command) (string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if decode(c.Params, &p) != nil {
		return "", errors.New("invalid list request")
	}
	path, x := e.resolve(p.Path, false)
	if x != nil {
		return "", x
	}
	items, x := os.ReadDir(path)
	if x != nil {
		return "", x
	}
	type item struct {
		Name      string `json:"name"`
		Directory bool   `json:"directory"`
		Size      int64  `json:"size"`
	}
	out := make([]item, 0, len(items))
	for _, v := range items {
		z, _ := v.Info()
		if z != nil {
			out = append(out, item{v.Name(), v.IsDir(), z.Size()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return marshal(out)
}
func (e *Executor) filesRead(c protocol.Command) (string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if decode(c.Params, &p) != nil {
		return "", errors.New("invalid read request")
	}
	path, x := e.resolve(p.Path, false)
	if x != nil {
		return "", x
	}
	f, x := os.Open(path)
	if x != nil {
		return "", x
	}
	defer f.Close()
	data, x := io.ReadAll(io.LimitReader(f, MaxTransfer+1))
	if x != nil {
		return "", x
	}
	if len(data) > MaxTransfer {
		return "", errors.New("file too large")
	}
	return marshal(map[string]any{"dataBase64": base64.StdEncoding.EncodeToString(data), "size": len(data)})
}
func (e *Executor) filesWrite(c protocol.Command) (string, error) {
	var p struct {
		Path       string `json:"path"`
		DataBase64 string `json:"dataBase64"`
	}
	if decode(c.Params, &p) != nil {
		return "", errors.New("invalid write request")
	}
	data, x := base64.StdEncoding.DecodeString(p.DataBase64)
	if x != nil || len(data) > MaxTransfer {
		return "", errors.New("invalid or oversized file")
	}
	path, x := e.resolve(p.Path, true)
	if x != nil {
		return "", x
	}
	if _, x = os.Lstat(path); x == nil {
		return "", errors.New("refusing to overwrite existing file")
	}
	if !errors.Is(x, os.ErrNotExist) {
		return "", x
	}
	if x = os.WriteFile(path, data, 0600); x != nil {
		return "", x
	}
	return marshal(map[string]int{"written": len(data)})
}
func marshal(v any) (string, error) { b, e := json.Marshal(v); return string(b), e }
func marshalWithError(v any, runErr error) (string, error) {
	s, err := marshal(v)
	if err != nil {
		return "", err
	}
	return s, runErr
}
