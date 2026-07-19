package executor

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
const MaxChunk = 1 << 20

type Executor struct {
	roots []string
	mu    sync.Mutex
	owned map[int]*ownedProcess
	jobs  map[string]*job
}

type processContainer interface {
	Terminate(uint32) error
	Close() error
}

type ownedProcess struct {
	cmd       *exec.Cmd
	container processContainer
}

type job struct {
	cmd       *exec.Cmd
	container processContainer
	cancel    context.CancelFunc
	out       capped
	done      chan struct{}
	err       error
	started   time.Time
	finished  time.Time
}

func New(roots []string) (*Executor, error) {
	e := &Executor{owned: map[int]*ownedProcess{}, jobs: map[string]*job{}}
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
	mu sync.Mutex
	b  bytes.Buffer
	n  int
}

func (c *capped) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
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
func (c *capped) snapshot() ([]byte, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.b.Bytes()...), c.n
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
	case "powershell.run":
		return e.powerShell(cmd)
	case "shell.start":
		return e.shellStart(cmd)
	case "shell.status":
		return e.shellStatus(cmd)
	case "shell.cancel":
		return e.shellCancel(cmd)
	case "files.list":
		return e.filesList(cmd)
	case "files.read", "files.download":
		return e.filesRead(cmd)
	case "files.read-chunk":
		return e.filesReadChunk(cmd)
	case "files.write", "files.upload":
		return e.filesWrite(cmd)
	case "files.write-chunk":
		return e.filesWriteChunk(cmd)
	case "session.disconnect":
		return marshal(map[string]bool{"disconnect": true})
	default:
		return "", errors.New("unauthorized or unknown command")
	}
}

// powerShell executes a script through a temporary UTF-8 file. Keeping the
// script out of cmd.exe and PowerShell's command-line parser avoids nested
// quoting bugs and the CLIXML progress stream emitted by -EncodedCommand.
func (e *Executor) powerShell(c protocol.Command) (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("PowerShell execution is currently supported only on Windows")
	}
	var p struct {
		Script string `json:"script"`
	}
	if decode(c.Params, &p) != nil || p.Script == "" || len(p.Script) > 64<<10 || strings.IndexByte(p.Script, 0) >= 0 {
		return "", errors.New("invalid PowerShell script")
	}
	f, err := os.CreateTemp("", "OpenClawBridge-script-*.ps1")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)
	// Windows PowerShell 5.1 treats UTF-8 files without a BOM as the active ANSI
	// code page and inherits an OEM console encoding when stdout is redirected.
	// Use a BOM for lossless script parsing and force UTF-8 for both native-command
	// input and captured output so characters outside the OEM code page survive.
	preamble := "$ProgressPreference='SilentlyContinue'\r\n" +
		"$utf8 = New-Object System.Text.UTF8Encoding($false)\r\n" +
		"[Console]::InputEncoding = $utf8\r\n" +
		"[Console]::OutputEncoding = $utf8\r\n" +
		"$OutputEncoding = $utf8\r\n"
	script := append([]byte{0xef, 0xbb, 0xbf}, []byte(preamble+p.Script)...)
	if _, err = f.Write(script); err != nil {
		_ = f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout(c))
	defer cancel()
	x := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path)
	var out capped
	x.Stdout = &out
	x.Stderr = &out
	err = x.Run()
	raw, count := out.snapshot()
	result := map[string]any{"output": normalizeOutput(raw), "truncated": count >= MaxOutput, "exitCode": exitCode(err)}
	if ctx.Err() != nil {
		return marshalWithError(result, errors.New("PowerShell timeout"))
	}
	return marshalWithError(result, err)
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
	raw, count := out.snapshot()
	result := map[string]any{"output": normalizeOutput(raw), "truncated": count >= MaxOutput, "exitCode": exitCode(err)}
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
	raw, count := out.snapshot()
	result := map[string]any{"output": normalizeOutput(raw), "truncated": count >= MaxOutput, "exitCode": exitCode(err)}
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
	parentRaw, _ := parentOut.snapshot()
	resultOut := normalizeOutput(parentRaw)
	if f, openErr := os.Open(outPath); openErr == nil {
		defer f.Close()
		data, _ := io.ReadAll(io.LimitReader(f, MaxOutput+1))
		resultOut += normalizeOutput(data[:min(len(data), MaxOutput)])
	}
	result := map[string]any{"output": resultOut, "truncated": len(resultOut) > MaxOutput, "exitCode": exitCode(err), "uacRequired": true}
	if ctx.Err() != nil {
		return marshalWithError(result, errors.New("administrator command timeout or UAC not approved"))
	}
	return marshalWithError(result, err)
}

func normalizeOutput(data []byte) string { return normalizePlatformOutput(data) }

func normalizePortableOutput(data []byte) string {
	if len(data) >= 2 && ((data[0] == 0xff && data[1] == 0xfe) || bytes.Count(data, []byte{0}) > len(data)/4) {
		if data[0] == 0xff && data[1] == 0xfe {
			data = data[2:]
		}
		words := make([]uint16, 0, len(data)/2)
		for i := 0; i+1 < len(data); i += 2 {
			words = append(words, uint16(data[i])|uint16(data[i+1])<<8)
		}
		return string(utf16.Decode(words))
	}
	return strings.ToValidUTF8(string(data), "�")
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (e *Executor) shellStart(c protocol.Command) (string, error) {
	var p struct {
		Command        string `json:"command"`
		TimeoutSeconds int    `json:"timeoutSeconds"`
	}
	if decode(c.Params, &p) != nil || p.Command == "" || len(p.Command) > 32768 || strings.IndexByte(p.Command, 0) >= 0 {
		return "", errors.New("invalid asynchronous shell command")
	}
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = 300
	}
	if p.TimeoutSeconds > 3600 {
		return "", errors.New("timeout exceeds one hour")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.TimeoutSeconds)*time.Second)
	var x *exec.Cmd
	if runtime.GOOS == "windows" {
		x = exec.CommandContext(ctx, "cmd.exe", "/d", "/s", "/c", p.Command)
	} else {
		x = exec.CommandContext(ctx, "/bin/sh", "-c", p.Command)
	}
	j := &job{cmd: x, cancel: cancel, done: make(chan struct{}), started: time.Now()}
	x.Stdout = &j.out
	x.Stderr = &j.out
	if err := x.Start(); err != nil {
		cancel()
		return "", err
	}
	container, err := newProcessContainer(x.Process)
	if err != nil {
		_ = x.Process.Kill()
		cancel()
		return "", err
	}
	j.container = container
	id := newID()
	e.mu.Lock()
	e.jobs[id] = j
	e.mu.Unlock()
	go func() {
		j.err = x.Wait()
		j.finished = time.Now()
		_ = j.container.Close()
		cancel()
		close(j.done)
		time.AfterFunc(5*time.Minute, func() {
			e.mu.Lock()
			if e.jobs[id] == j {
				delete(e.jobs, id)
			}
			e.mu.Unlock()
		})
	}()
	return marshal(map[string]any{"jobId": id, "pid": x.Process.Pid, "startedAt": j.started})
}

func (e *Executor) shellStatus(c protocol.Command) (string, error) {
	var p struct {
		JobID  string `json:"jobId"`
		Offset int    `json:"offset"`
	}
	if decode(c.Params, &p) != nil || p.JobID == "" || p.Offset < 0 {
		return "", errors.New("invalid job status request")
	}
	e.mu.Lock()
	j := e.jobs[p.JobID]
	e.mu.Unlock()
	if j == nil {
		return "", errors.New("job not found")
	}
	raw, total := j.out.snapshot()
	if p.Offset > len(raw) {
		p.Offset = len(raw)
	}
	running := true
	select {
	case <-j.done:
		running = false
	default:
	}
	return marshal(map[string]any{"jobId": p.JobID, "running": running, "output": normalizeOutput(raw[p.Offset:]), "nextOffset": len(raw), "totalBytes": total, "exitCode": exitCode(j.err), "startedAt": j.started, "finishedAt": j.finished})
}

func (e *Executor) shellCancel(c protocol.Command) (string, error) {
	var p struct {
		JobID string `json:"jobId"`
	}
	if decode(c.Params, &p) != nil || p.JobID == "" {
		return "", errors.New("invalid job cancellation request")
	}
	e.mu.Lock()
	j := e.jobs[p.JobID]
	e.mu.Unlock()
	if j == nil {
		return "", errors.New("job not found")
	}
	if j.container != nil {
		_ = j.container.Terminate(1)
	}
	j.cancel()
	return marshal(map[string]any{"jobId": p.JobID, "cancelled": true})
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
	container, err := newProcessContainer(x.Process)
	if err != nil {
		_ = x.Process.Kill()
		return "", err
	}
	e.mu.Lock()
	e.owned[x.Process.Pid] = &ownedProcess{cmd: x, container: container}
	e.mu.Unlock()
	go func() {
		_ = x.Wait()
		_ = container.Close()
		e.mu.Lock()
		delete(e.owned, x.Process.Pid)
		e.mu.Unlock()
	}()
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
	return marshalWithError(map[string]int{"pid": p.PID}, x.container.Terminate(1))
}
func (e *Executor) filesList(c protocol.Command) (string, error) {
	var p struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
		Filter string `json:"filter"`
	}
	if decode(c.Params, &p) != nil || p.Offset < 0 || p.Limit < 0 || p.Limit > 1000 {
		return "", errors.New("invalid list request")
	}
	if p.Limit == 0 {
		p.Limit = 200
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
	out := make([]item, 0, min(len(items), p.Limit))
	for _, v := range items {
		if p.Filter != "" && !strings.Contains(strings.ToLower(v.Name()), strings.ToLower(p.Filter)) {
			continue
		}
		z, _ := v.Info()
		if z != nil {
			out = append(out, item{v.Name(), v.IsDir(), z.Size()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	total := len(out)
	if p.Offset > total {
		p.Offset = total
	}
	end := min(total, p.Offset+p.Limit)
	return marshal(map[string]any{"items": out[p.Offset:end], "offset": p.Offset, "limit": p.Limit, "total": total, "hasMore": end < total})
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

func (e *Executor) filesReadChunk(c protocol.Command) (string, error) {
	var p struct {
		Path   string `json:"path"`
		Offset int64  `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if decode(c.Params, &p) != nil || p.Offset < 0 || p.Limit < 0 || p.Limit > MaxTransfer {
		return "", errors.New("invalid read chunk request")
	}
	if p.Limit == 0 {
		p.Limit = 1 << 20
	}
	path, err := e.resolve(p.Path, false)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || p.Offset > info.Size() {
		return "", errors.New("offset outside file")
	}
	data := make([]byte, min(p.Limit, int(info.Size()-p.Offset)))
	n, err := f.ReadAt(data, p.Offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	data = data[:n]
	sum := sha256.Sum256(data)
	return marshal(map[string]any{"dataBase64": base64.StdEncoding.EncodeToString(data), "offset": p.Offset, "nextOffset": p.Offset + int64(n), "size": info.Size(), "eof": p.Offset+int64(n) >= info.Size(), "chunkSHA256": hex.EncodeToString(sum[:])})
}

func (e *Executor) filesWriteChunk(c protocol.Command) (string, error) {
	var p struct {
		Path           string `json:"path"`
		Offset         int64  `json:"offset"`
		DataBase64     string `json:"dataBase64"`
		Final          bool   `json:"final"`
		ExpectedSHA256 string `json:"expectedSHA256"`
	}
	if decode(c.Params, &p) != nil || p.Offset < 0 {
		return "", errors.New("invalid write chunk request")
	}
	data, err := base64.StdEncoding.DecodeString(p.DataBase64)
	if err != nil || len(data) > MaxChunk || p.Offset+int64(len(data)) > int64(MaxTransfer) {
		return "", errors.New("invalid or oversized chunk")
	}
	target, err := e.resolve(p.Path, true)
	if err != nil {
		return "", err
	}
	part := target + ".openclaw-part"
	flags := os.O_WRONLY | os.O_CREATE
	if p.Offset == 0 {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(part, flags, 0600)
	if err != nil {
		return "", err
	}
	info, statErr := f.Stat()
	if statErr != nil || info.Size() != p.Offset {
		_ = f.Close()
		return "", errors.New("chunk offset mismatch")
	}
	if _, err = f.Seek(p.Offset, io.SeekStart); err != nil {
		_ = f.Close()
		return "", err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if !p.Final {
		return marshal(map[string]any{"written": len(data), "nextOffset": p.Offset + int64(len(data)), "final": false})
	}
	if p.ExpectedSHA256 == "" {
		return "", errors.New("final chunk requires expected SHA-256")
	}
	all, err := os.ReadFile(part)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(all)
	actual := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actual, p.ExpectedSHA256) {
		_ = os.Remove(part)
		return "", errors.New("final SHA-256 mismatch")
	}
	if _, err = os.Lstat(target); err == nil {
		return "", errors.New("refusing to overwrite existing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err = os.Rename(part, target); err != nil {
		return "", err
	}
	return marshal(map[string]any{"written": len(data), "size": len(all), "sha256": actual, "final": true})
}
func marshal(v any) (string, error) { b, e := json.Marshal(v); return string(b), e }
func marshalWithError(v any, runErr error) (string, error) {
	s, err := marshal(v)
	if err != nil {
		return "", err
	}
	return s, runErr
}
