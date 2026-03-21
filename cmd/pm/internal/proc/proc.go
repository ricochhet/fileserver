package proc

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/ricochhet/fileserver/cmd/pm/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"go.yaml.in/yaml/v4"
)

type Context struct {
	*configutil.Context

	Options *Options
}

type Options struct {
	ConfigFile string
	ProcFile   string
	EnvFiles   string

	Port           uint
	BasePort       uint
	ExitOnError    bool
	ExitOnStop     bool
	StartRPCServer bool
	PTY            bool
	Interval       uint
	ReverseOnStop  bool
	Overload       bool
	Fork           bool
	InheritStdin   bool
	Silent         bool
}

// Start spawns procs.
func (c *Context) Start(ctx context.Context, sig <-chan os.Signal) error {
	c.Context = configutil.NewContext()
	c.Set(c.readConfig())

	err := c.Parse(c.readProcFile())
	if err != nil {
		return errutil.New("c.config.Parse", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	// Cancel the RPC server when procs have returned/errored, cancel the
	// context anyway in case of early return.
	defer cancel()

	if len(c.Get().Args) > 1 {
		tmp := make([]*configutil.ProcInfo, 0, len(c.Get().Args[0:]))

		c.Get().MaxProcNameLength = 0
		for _, v := range c.Get().Args[0:] {
			proc := c.findProc(v)
			if proc == nil {
				return errutil.Newf("c.findProc", "unknown proc: %s", v)
			}

			tmp = append(tmp, proc)

			if len(v) > c.Get().MaxProcNameLength {
				c.Get().MaxProcNameLength = len(v)
			}
		}

		c.GetLocked().Procs = tmp
	}

	envs := c.Get().EnvFiles
	if len(envs) > 0 {
		if err := c.loadEnv(envs); err != nil {
			return errutil.New("c.loadenv", err)
		}
	}

	rpcChan := make(chan *rpcMessage, 10)

	if c.Options.StartRPCServer {
		go func() {
			if err := startServer(ctx, rpcChan, c.GetLocked().Port); err != nil {
				logutil.Errorf(logutil.Get(), "Error starting RPC server: %v\n", err)
			}
		}()
	}

	//nolint:contextcheck // wontfix
	return c.startProcs(sig, rpcChan, c.Get().ExitOnError)
}

// Check validates the procfile and lists its entries.
func (c *Context) Check() error {
	err := c.Parse(c.readProcFile())
	if err != nil {
		return errutil.New("c.config.Parse", err)
	}

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	keys := make([]string, len(c.Get().Procs))

	i := 0
	for _, proc := range c.Get().Procs {
		keys[i] = proc.Name
		i++
	}

	sort.Strings(keys)
	fmt.Printf("valid procfile detected (%s)\n", strings.Join(keys, ", "))

	return nil
}

// loadEnv loads the env files using godotenv.Load.
func (c *Context) loadEnv(e []string) error {
	if c.Options.Overload {
		if err := godotenv.Overload(e...); err != nil {
			return errutil.New("godotenv.Overload", err)
		}
	}

	if err := godotenv.Load(e...); err != nil {
		return errutil.New("godotenv.Load", err)
	}

	return nil
}

// spawnProc starts the specified proc, and returns any error from running it.
func (c *Context) spawnProc(name string, errCh chan<- error) {
	proc := c.findProc(name)
	logger := logutil.CreateLogger(name, proc.ColorIndex)

	cs := slices.Concat(cmdStart, []string{proc.Cmdline})
	cmd := exec.CommandContext(context.Background(), cs[0], cs[1:]...)

	if c.Options.InheritStdin {
		cmd.Stdin = os.Stdin
	}

	if proc.Fork {
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.SysProcAttr = forkProcAttrs
	} else {
		cmd.Stdout = logger
		cmd.Stderr = logger
		cmd.SysProcAttr = procAttrs
	}

	if proc.Silent {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := c.startPTY(logger, cmd); err != nil {
		select {
		case errCh <- err:
		default:
		}

		logutil.Errorf(logutil.Get(), "Failed to open pty for %s: %s\n", name, err)

		return
	}

	if proc.SetPort {
		cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", proc.Port))
		logutil.Infof(logutil.Get(), "Starting %s on port %d\n", name, proc.Port)
	}

	envs, err := godotenv.Read(c.GetLocked().EnvFiles...)
	if err != nil {
		logutil.Errorf(logutil.Get(), "Failed to read env: %v\n", err)
		return
	}

	for key, value := range envs {
		k, found := strings.CutPrefix(key, "SET_")
		if !found {
			continue
		}

		cmd.Env = append(cmd.Env, fsutil.JoinEnviron(k, []string{value}))
		logutil.Debugf(logutil.Get(), "added env: %s=%s\n", "PATH", value)
	}

	if err := cmd.Start(); err != nil {
		select {
		case errCh <- err:
		default:
		}

		logutil.Errorf(logutil.Get(), "Failed to start %s: %s\n", name, err)

		return
	}

	proc.Cmd = cmd
	proc.StoppedBySupervisor = false

	if !proc.Fork {
		proc.Mutex.Unlock()

		err = cmd.Wait()

		proc.Mutex.Lock()
	}

	proc.Cond.Broadcast()

	if err != nil && !proc.StoppedBySupervisor {
		select {
		case errCh <- err:
		default:
		}
	}

	proc.WaitErr = err
	proc.Cmd = nil

	logutil.Infof(logutil.Get(), "Terminating %s\n", name)
}

// startProc starts the specified proc, if proc is started already, return nil.
func (c *Context) startProc(name string, wg *sync.WaitGroup, errCh chan<- error) error {
	proc := c.findProc(name)
	if proc == nil {
		return errutil.Newf("c.findProc", "unknown name: %s", name)
	}

	proc.Mutex.Lock()

	if proc.Cmd != nil {
		proc.Mutex.Unlock()
		return nil
	}

	if wg != nil {
		wg.Add(1)
	}

	go func() {
		c.spawnProc(name, errCh)

		if wg != nil {
			wg.Done()
		}

		proc.Mutex.Unlock()
	}()

	return nil
}

// stopProc stops the specified proc, issuing os.Kill if it does not terminate within 10
// seconds. If signal is nil, os.Interrupt is used.
func (c *Context) stopProc(name string, signal os.Signal) error {
	if signal == nil {
		signal = os.Interrupt
	}

	proc := c.findProc(name)
	if proc == nil {
		return errors.New("unknown proc: " + name)
	}

	proc.Mutex.Lock()
	defer proc.Mutex.Unlock()

	if proc.Cmd == nil {
		return nil
	}

	proc.StoppedBySupervisor = true

	err := terminateProc(proc, signal)
	if err != nil {
		return errutil.New("terminateProc", err)
	}

	timeout := time.AfterFunc(10*time.Second, func() {
		proc.Mutex.Lock()
		defer proc.Mutex.Unlock()

		if proc.Cmd != nil {
			err = killProc(proc.Cmd.Process)
		}
	})

	proc.Cond.Wait()
	timeout.Stop()

	return errutil.WithFrame(err)
}

// restartProc restarts the proc by name.
func (c *Context) restartProc(name string) error {
	err := c.stopProc(name, nil)
	if err != nil {
		return errutil.WithFrame(err)
	}

	return c.startProc(name, nil, nil)
}

// stopProcs attempts to stop every running process and returns any non-nil
// error, if one exists. stopProcs will wait until all procs have had an
// opportunity to stop.
func (c *Context) stopProcs(sig os.Signal) error {
	var err error

	if c.Options.ReverseOnStop {
		tmp := make([]*configutil.ProcInfo, len(c.Get().Procs))
		for i := range len(c.Get().Procs) {
			tmp[i] = c.readConfig().Procs[(len(c.Get().Procs)-1)-i]
		}

		c.GetLocked().Procs = tmp
	}

	for _, proc := range c.Get().Procs {
		stopErr := c.stopProc(proc.Name, sig)
		if stopErr != nil {
			err = stopErr
		}

		if c.Options.Interval > 0 {
			time.Sleep(time.Second * time.Duration(c.Options.Interval))
		}
	}

	return errutil.WithFrame(err)
}

// startProcs starts all procs.
func (c *Context) startProcs(
	sc <-chan os.Signal,
	rpcCh <-chan *rpcMessage,
	exitOnError bool,
) error {
	var wg sync.WaitGroup

	errCh := make(chan error, 1)

	for _, proc := range c.Get().Procs {
		if err := c.startProc(proc.Name, &wg, errCh); err != nil {
			return errutil.New("c.startProc", err)
		}

		if c.Options.Interval > 0 {
			time.Sleep(time.Second * time.Duration(c.Options.Interval))
		}
	}

	allProcsDone := make(chan struct{}, 1)

	if c.Options.ExitOnStop {
		go func() {
			wg.Wait()

			allProcsDone <- struct{}{}
		}()
	}

	for {
		select {
		case rpcMsg := <-rpcCh:
			switch rpcMsg.Msg {
			// TODO: add more events here.
			case "stop":
				for _, proc := range rpcMsg.Args {
					if err := c.stopProc(proc, nil); err != nil {
						rpcMsg.ErrCh <- err
						break
					}
				}

				close(rpcMsg.ErrCh)
			default:
				panic("unimplemented rpc message type " + rpcMsg.Msg)
			}
		case err := <-errCh:
			if exitOnError {
				if err := c.stopProcs(os.Interrupt); err != nil {
					return errutil.New("c.stopProcs", err)
				}

				return errutil.WithFrame(err)
			}
		case <-allProcsDone:
			return c.stopProcs(os.Interrupt)
		case sig := <-sc:
			return c.stopProcs(sig)
		}
	}
}

// readConfig reads the config file at the specified path.
func (c *Context) readConfig() *configutil.Config {
	config := &configutil.Config{
		ProcFile: c.Options.ProcFile,
		Port:     c.Options.Port,
		BasePort: c.Options.BasePort,
		EnvFiles: strings.FieldsFunc(
			c.Options.EnvFiles,
			func(char rune) bool { return char == ',' },
		),
		ExitOnError: c.Options.ExitOnError,
		Args:        flag.Args(),
	}

	b, err := os.ReadFile(c.Options.ConfigFile)
	if err == nil {
		_ = yaml.Unmarshal(b, &config)
	}

	return config
}

// readProcFile reads the procfile at the specified path.
func (c *Context) readProcFile() []byte {
	b, err := os.ReadFile(c.Options.ProcFile)
	if err != nil {
		return nil
	}

	return b
}

// findProc finds the process in the slice by name.
func (c *Context) findProc(name string) *configutil.ProcInfo {
	for _, proc := range c.GetLocked().Procs {
		if proc.Name == name {
			return proc
		}
	}

	return nil
}
