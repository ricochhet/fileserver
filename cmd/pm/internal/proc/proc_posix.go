//go:build !windows
// +build !windows

package proc

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"

	"github.com/creack/pty"
	"github.com/ricochhet/fileserver/cmd/pm/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"golang.org/x/sys/unix"
)

const (
	sigint  = unix.SIGINT
	sigterm = unix.SIGTERM
	sighup  = unix.SIGHUP
)

var (
	cmdStart      = []string{"/bin/sh", "-c"}
	procAttrs     = &unix.SysProcAttr{Setpgid: true}
	forkProcAttrs = &unix.SysProcAttr{
		Setsid: true,
	}
)

// terminateProc terminates the process by sending the signal to the process.
func terminateProc(proc *configutil.ProcInfo, signal os.Signal) error {
	p := proc.Cmd.Process
	if p == nil {
		return nil
	}

	pgid, err := unix.Getpgid(p.Pid)
	if err != nil {
		return errutil.New("unix.Getpgid", err)
	}

	// use pgid, ref: http://unix.stackexchange.com/questions/14815/process-descendants
	pid := p.Pid
	if pgid == p.Pid {
		pid = -1 * pid
	}

	target, err := os.FindProcess(pid)
	if err != nil {
		return errutil.New("os.FindProcess", err)
	}

	return target.Signal(signal)
}

// killProc kills the proc with pid pid, as well as its children.
func killProc(process *os.Process) error {
	return unix.Kill(-1*process.Pid, unix.SIGKILL)
}

// NotifyCh creates the terminate/interrupt notifier.
func NotifyCh() <-chan os.Signal {
	sc := make(chan os.Signal, 10)
	signal.Notify(sc, sigterm, sigint, sighup)

	return sc
}

// startPTY starts a PTY terminal.
func (c *Context) startPTY(logger *logutil.Logger, cmd *exec.Cmd) error {
	if c.Options.PTY {
		p, t, err := pty.Open()
		if err != nil {
			return errutil.WithFramef("failed to open PTY: %w", err)
		}

		defer p.Close()
		defer t.Close()

		cmd.Stdout = t
		cmd.Stderr = t

		go func() {
			if _, err := io.Copy(logger, p); err != nil && !errors.Is(err, io.EOF) {
				logutil.Errorf(os.Stderr, "%v\n", err)
			}
		}()
	} else {
		cmd.Stdout = logger
		cmd.Stderr = logger
	}

	return nil
}
