package configutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"github.com/sasha-s/go-deadlock"
)

type Config struct {
	ProcFile string `yaml:"procfile"`
	// Port for RPC server
	Port     uint `yaml:"port"`
	BasePort uint `yaml:"baseport"`
	Fork     bool `yaml:"fork"`
	Silent   bool `yaml:"silent"`
	Args     []string
	EnvFiles []string
	// If true, exit the supervisor process if a subprocess exits with an error.
	ExitOnError bool `yaml:"exitOnErr"`

	// Context
	Procs             []*ProcInfo
	MaxProcNameLength int
	SetPorts          bool
}

type ProcInfo struct {
	Name       string
	Cmdline    string
	Cmd        *exec.Cmd
	Port       uint
	SetPort    bool
	ColorIndex int
	Fork       bool
	Silent     bool

	// True if we called stopProc to kill the process, in which case an
	// *os.ExitError is not the fault of the subprocess
	StoppedBySupervisor bool

	Mutex   deadlock.Mutex
	Cond    *sync.Cond
	WaitErr error
}

// Parse parses the procfile bytes into the ProcInfo slice.
func (c *Context) Parse(b []byte) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.Get().Procs = []*ProcInfo{}
	re := regexp.MustCompile(`\$([a-zA-Z]+[a-zA-Z0-9_]+)`)
	index := 0

	for l := range strings.SplitSeq(string(b), "\n") {
		tokens := strings.SplitN(l, ":", 2)
		if len(tokens) != 2 || tokens[0][0] == '#' {
			continue
		}

		k, v := strings.TrimSpace(tokens[0]), strings.TrimSpace(tokens[1])
		if runtime.GOOS == "windows" {
			v = re.ReplaceAllStringFunc(v, func(s string) string {
				return "%" + s[1:] + "%"
			})
		}

		proc := &ProcInfo{Name: k, Cmdline: v, ColorIndex: index}
		if c.Get().SetPorts {
			proc.SetPort = true
			proc.Port = c.Get().BasePort
			c.Get().BasePort += 100
		}

		proc.Fork = c.Get().Fork
		proc.Silent = c.Get().Silent
		proc.Cond = sync.NewCond(&proc.Mutex)

		c.Get().Procs = append(c.Get().Procs, proc)

		if len(k) > c.Get().MaxProcNameLength {
			c.Get().MaxProcNameLength = len(k)
		}

		index = (index + 1) % len(logutil.Colors)
	}

	if len(c.Get().Procs) == 0 {
		return errutil.WithFrame(errors.New("no valid entry"))
	}

	return nil
}

// DefaultServer returns the default server IP.
func DefaultServer(p uint) string {
	if s, ok := os.LookupEnv("PM_RPC_SERVER"); ok {
		return s
	}

	return fmt.Sprintf("127.0.0.1:%d", DefaultPort(p))
}

// DefaultAddr returns the default address.
func DefaultAddr() string {
	if s, ok := os.LookupEnv("PM_RPC_ADDR"); ok {
		return s
	}

	return "0.0.0.0"
}

// DefaultPort returns the default port.
func DefaultPort(p uint) uint {
	s := os.Getenv("PM_RPC_PORT")
	if s != "" {
		i, err := strconv.Atoi(s)
		if err == nil {
			return uint(i)
		}
	}

	if p > 0 {
		return p
	}

	return 8555
}
