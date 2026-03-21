package proc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/ricochhet/fileserver/cmd/pm/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
)

// RPC is RPC server.
type RPC struct {
	*Context

	rpcChan chan<- *rpcMessage
}

type rpcMessage struct {
	Msg  string
	Args []string
	// sending error (if any) when the task completes
	ErrCh chan error
}

// Start do start.
func (r *RPC) Start(args []string, _ *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	for _, arg := range args {
		if err = r.startProc(arg, nil, nil); err != nil {
			break
		}
	}

	return errutil.WithFrame(err)
}

// Stop do stop.
func (r *RPC) Stop(args []string, _ *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	errChan := make(chan error, 1)
	r.rpcChan <- &rpcMessage{
		Msg:   "stop",
		Args:  args,
		ErrCh: errChan,
	}

	err = <-errChan

	return errutil.WithFrame(err)
}

// StopAll do stop all.
func (r *RPC) StopAll(_ []string, _ *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	for _, proc := range r.GetLocked().Procs {
		if err = r.stopProc(proc.Name, nil); err != nil {
			break
		}
	}

	return errutil.WithFrame(err)
}

// Restart do restart.
func (r *RPC) Restart(args []string, _ *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	for _, arg := range args {
		if err = r.restartProc(arg); err != nil {
			break
		}
	}

	return errutil.WithFrame(err)
}

// RestartAll do restart all.
func (r *RPC) RestartAll(_ []string, _ *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	for _, proc := range r.GetLocked().Procs {
		if err = r.restartProc(proc.Name); err != nil {
			break
		}
	}

	return errutil.WithFrame(err)
}

// List do list.
func (r *RPC) List(_ []string, ret *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	*ret = ""
	for _, proc := range r.GetLocked().Procs {
		*ret += proc.Name + "\n"
	}

	return errutil.WithFrame(err)
}

// Status do status.
func (r *RPC) Status(_ []string, ret *string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	*ret = ""

	for _, proc := range r.GetLocked().Procs {
		if proc.Cmd != nil {
			*ret += "*" + proc.Name + "\n"
		} else {
			*ret += " " + proc.Name + "\n"
		}
	}

	return errutil.WithFrame(err)
}

// Run runs the RPC command.
func Run(cmd string, args []string) error {
	client, err := rpc.Dial("tcp", configutil.DefaultServer(0))
	if err != nil {
		return errutil.WithFrame(err)
	}
	defer client.Close()

	var ret string

	switch cmd {
	case "start":
		return client.Call("RPC.Start", args, &ret)
	case "stop":
		return client.Call("RPC.Stop", args, &ret)
	case "stop-all":
		return client.Call("RPC.StopAll", args, &ret)
	case "restart":
		return client.Call("RPC.Restart", args, &ret)
	case "restart-all":
		return client.Call("RPC.RestartAll", args, &ret)
	case "list":
		err := client.Call("RPC.List", args, &ret)
		fmt.Print(ret)

		return errutil.WithFrame(err)
	case "status":
		err := client.Call("RPC.Status", args, &ret)
		fmt.Print(ret)

		return errutil.WithFrame(err)
	}

	return errors.New("unknown command")
}

func startServer(ctx context.Context, rpcChan chan<- *rpcMessage, listenPort uint) error {
	r := &RPC{
		rpcChan: rpcChan,
	}
	if err := rpc.Register(r); err != nil {
		return errutil.New("rpc.Register", err)
	}

	lc := net.ListenConfig{}

	server, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", configutil.DefaultAddr(), listenPort))
	if err != nil {
		return errutil.New("lc.Listen", err)
	}

	var wg sync.WaitGroup

	acceptingConns := true

outer:
	for acceptingConns {
		conns := make(chan net.Conn, 1)

		go func() {
			conn, err := server.Accept()
			if err != nil {
				return
			}

			conns <- conn
		}()

		select {
		case <-ctx.Done():
			acceptingConns = false //nolint:ineffassign,wastedassign // wontfix
			break outer
		case client := <-conns: // server is not canceled.
			wg.Go(func() {
				rpc.ServeConn(client)
			})
		}
	}

	done := make(chan struct{}, 1)

	go func() {
		wg.Wait()

		done <- struct{}{}
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("RPC server did not shut down in 10 seconds, quitting")
	}
}
