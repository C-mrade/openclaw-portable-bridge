//go:build windows

package executor

// Native Windows process-containment and pseudo-console primitives.  This file
// intentionally has no third-party dependencies: every entry point is supplied
// by kernel32.dll on supported Windows 10/11 hosts.

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW        = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJob      = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject      = kernel32.NewProc("TerminateJobObject")
	procCloseHandle             = kernel32.NewProc("CloseHandle")
	procCreatePipe              = kernel32.NewProc("CreatePipe")
	procCreatePseudoConsole     = kernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole     = kernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole      = kernel32.NewProc("ClosePseudoConsole")
)

const (
	jobObjectExtendedLimitInformation = 9
	jobObjectLimitKillOnJobClose      = 0x00002000
	processTerminate                  = 0x0001
	processSetQuota                   = 0x0100
)

type ioCounters struct {
	readOperationCount  uint64
	writeOperationCount uint64
	otherOperationCount uint64
	readTransferCount   uint64
	writeTransferCount  uint64
	otherTransferCount  uint64
}

type basicLimitInformation struct {
	perProcessUserTimeLimit int64
	perJobUserTimeLimit     int64
	limitFlags              uint32
	minimumWorkingSetSize   uintptr
	maximumWorkingSetSize   uintptr
	activeProcessLimit      uint32
	affinity                uintptr
	priorityClass           uint32
	schedulingClass         uint32
}

type extendedLimitInformation struct {
	basicLimitInformation basicLimitInformation
	ioInfo                ioCounters
	processMemoryLimit    uintptr
	jobMemoryLimit        uintptr
	peakProcessMemoryUsed uintptr
	peakJobMemoryUsed     uintptr
}

// windowsJob owns a Windows Job Object configured to terminate every assigned
// process (including descendants) when it is closed.  Assign should happen
// immediately after Cmd.Start; creating the process suspended would close the
// remaining small child-spawn race and is left for the shared executor hook.
type windowsJob struct {
	handle syscall.Handle
}

func newProcessContainer(process *os.Process) (processContainer, error) {
	j, err := newWindowsJob()
	if err != nil {
		return nil, err
	}
	if err = j.Assign(process); err != nil {
		_ = j.Close()
		return nil, err
	}
	return j, nil
}

func newWindowsJob() (*windowsJob, error) {
	h, _, callErr := procCreateJobObjectW.Call(0, 0)
	if h == 0 {
		return nil, winCallError("CreateJobObjectW", callErr)
	}
	j := &windowsJob{handle: syscall.Handle(h)}
	info := extendedLimitInformation{}
	info.basicLimitInformation.limitFlags = jobObjectLimitKillOnJobClose
	ok, _, callErr := procSetInformationJobObject.Call(
		uintptr(j.handle), jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info),
	)
	if ok == 0 {
		_ = j.Close()
		return nil, winCallError("SetInformationJobObject", callErr)
	}
	return j, nil
}

func (j *windowsJob) Assign(process *os.Process) error {
	if j == nil || j.handle == 0 || process == nil {
		return errors.New("invalid job or process")
	}
	h, err := syscall.OpenProcess(processSetQuota|processTerminate, false, uint32(process.Pid))
	if err != nil {
		return fmt.Errorf("open process %d: %w", process.Pid, err)
	}
	defer syscall.CloseHandle(h)
	ok, _, callErr := procAssignProcessToJob.Call(uintptr(j.handle), uintptr(h))
	if ok == 0 {
		return winCallError("AssignProcessToJobObject", callErr)
	}
	return nil
}

func (j *windowsJob) Terminate(exitCode uint32) error {
	if j == nil || j.handle == 0 {
		return errors.New("job is closed")
	}
	ok, _, callErr := procTerminateJobObject.Call(uintptr(j.handle), uintptr(exitCode))
	if ok == 0 {
		return winCallError("TerminateJobObject", callErr)
	}
	return nil
}

func (j *windowsJob) Close() error {
	if j == nil || j.handle == 0 {
		return nil
	}
	h := j.handle
	j.handle = 0
	ok, _, callErr := procCloseHandle.Call(uintptr(h))
	if ok == 0 {
		return winCallError("CloseHandle(job)", callErr)
	}
	return nil
}

// pseudoConsole is the native I/O substrate for a future persistent terminal
// session. Input is written to Input and terminal output is read from Output.
// Its HPCON handle can be attached to STARTUPINFOEX via
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE by the process-spawn integration.
type pseudoConsole struct {
	hpc    uintptr
	Input  *os.File
	Output *os.File
}

func newPseudoConsole(cols, rows int16) (*pseudoConsole, error) {
	if cols <= 0 || rows <= 0 {
		return nil, errors.New("pseudo console dimensions must be positive")
	}
	var inRead, inWrite, outRead, outWrite syscall.Handle
	if err := createPipe(&inRead, &inWrite); err != nil {
		return nil, err
	}
	if err := createPipe(&outRead, &outWrite); err != nil {
		syscall.CloseHandle(inRead)
		syscall.CloseHandle(inWrite)
		return nil, err
	}
	var hpc uintptr
	hr, _, _ := procCreatePseudoConsole.Call(
		packCoord(cols, rows),
		uintptr(inRead), uintptr(outWrite), 0, uintptr(unsafe.Pointer(&hpc)),
	)
	syscall.CloseHandle(inRead)
	syscall.CloseHandle(outWrite)
	if int32(hr) < 0 {
		syscall.CloseHandle(inWrite)
		syscall.CloseHandle(outRead)
		return nil, fmt.Errorf("CreatePseudoConsole: HRESULT 0x%08x", uint32(hr))
	}
	return &pseudoConsole{
		hpc:    hpc,
		Input:  os.NewFile(uintptr(inWrite), "conpty-input"),
		Output: os.NewFile(uintptr(outRead), "conpty-output"),
	}, nil
}

func createPipe(read, write *syscall.Handle) error {
	ok, _, callErr := procCreatePipe.Call(uintptr(unsafe.Pointer(read)), uintptr(unsafe.Pointer(write)), 0, 0)
	if ok == 0 {
		return winCallError("CreatePipe", callErr)
	}
	return nil
}

func (p *pseudoConsole) Handle() uintptr {
	if p == nil {
		return 0
	}
	return p.hpc
}

func (p *pseudoConsole) Resize(cols, rows int16) error {
	if p == nil || p.hpc == 0 {
		return errors.New("pseudo console is closed")
	}
	if cols <= 0 || rows <= 0 {
		return errors.New("pseudo console dimensions must be positive")
	}
	hr, _, _ := procResizePseudoConsole.Call(p.hpc, packCoord(cols, rows))
	if int32(hr) < 0 {
		return fmt.Errorf("ResizePseudoConsole: HRESULT 0x%08x", uint32(hr))
	}
	return nil
}

func packCoord(cols, rows int16) uintptr {
	return uintptr(uint16(cols)) | uintptr(uint16(rows))<<16
}

func (p *pseudoConsole) Close() error {
	if p == nil {
		return nil
	}
	if p.hpc != 0 {
		procClosePseudoConsole.Call(p.hpc)
		p.hpc = 0
	}
	var first error
	if p.Input != nil {
		first = p.Input.Close()
		p.Input = nil
	}
	if p.Output != nil {
		if err := p.Output.Close(); first == nil {
			first = err
		}
		p.Output = nil
	}
	return first
}

func winCallError(name string, err error) error {
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		return fmt.Errorf("%s failed", name)
	}
	return fmt.Errorf("%s: %w", name, err)
}
