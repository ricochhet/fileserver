package proc

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/ricochhet/fileserver/cmd/pm/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"golang.org/x/sys/windows"
)

var (
	cmdStart  = []string{"cmd", "/c"}
	procAttrs = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_NEW_PROCESS_GROUP,
	}
	forkProcAttrs = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
	}
)

// terminateProc terminates the process by sending the signal to the process.
func terminateProc(proc *configutil.ProcInfo, _ os.Signal) error {
	dll, err := windows.LoadDLL("kernel32.dll")
	if err != nil {
		return errutil.New("windows.LoadDLL", err)
	}

	defer func() {
		if err := dll.Release(); err != nil {
			logutil.Errorf(logutil.Get(), "Error attempting to release DLL: %v\n", err)
		}
	}()

	pid := proc.Cmd.Process.Pid

	f, err := dll.FindProc("AttachConsole")
	if err != nil {
		return errutil.New("dll.FindProc", err)
	}

	r1, _, err := f.Call(uintptr(pid))
	if r1 == 0 && !errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
		return errutil.New("f.Call", err)
	}

	f, err = dll.FindProc("SetConsoleCtrlHandler")
	if err != nil {
		return errutil.New("dll.FindProc", err)
	}

	r1, _, err = f.Call(0, 1)
	if r1 == 0 {
		return errutil.New("f.Call", err)
	}

	f, err = dll.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		return errutil.New("dll.FindProc", err)
	}

	r1, _, err = f.Call(windows.CTRL_BREAK_EVENT, uintptr(pid))
	if r1 == 0 {
		return errutil.New("f.Call", err)
	}

	r1, _, err = f.Call(windows.CTRL_C_EVENT, uintptr(pid))
	if r1 == 0 {
		return errutil.New("f.Call", err)
	}

	return nil
}

// killProc kills the proc with pid pid, as well as its children.
func killProc(process *os.Process) error {
	return process.Kill()
}

// NotifyCh creates the terminate/interrupt notifier.
func NotifyCh() <-chan os.Signal {
	sc := make(chan os.Signal, 10)
	signal.Notify(sc, os.Interrupt)

	return sc
}

// startPTY starts a PTY terminal.
func (c *Context) startPTY(logger *logutil.Logger, cmd *exec.Cmd) error {
	cmd.Stdout = logger
	cmd.Stderr = logger

	return nil
}
