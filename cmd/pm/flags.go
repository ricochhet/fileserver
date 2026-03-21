package main

import (
	"flag"

	"github.com/ricochhet/fileserver/cmd/pm/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/cmdutil"
)

type Flags struct {
	Debug     bool
	QuickEdit bool

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

var (
	flags = NewFlags()
	cmds  = cmdutil.Commands{
		{Usage: "pm help", Desc: "Show this help"},
		{Usage: "pm start", Desc: "Start the process manager"},
		{Usage: "pm run [COMMAND] [ARGS]", Desc: "Run an RPC command with the specified args"},
		{Usage: "pm check", Desc: "Check if the provided procfile is valid"},
		{Usage: "pm export [FORMAT] [PATH]", Desc: "Export the procfile in the specified format"},
		{Usage: "pm dump [PATH]", Desc: "Dump embedded files to disk"},
		{Usage: "pm version", Desc: "Display pm version"},
	}
)

// NewFlags creates an empty Flags.
func NewFlags() *Flags {
	return &Flags{}
}

//nolint:gochecknoinits // wontfix
func init() {
	registerFlags(flag.CommandLine, flags)
	flag.Parse()
}

// registerFlags registers all flags to the flagset.
func registerFlags(fs *flag.FlagSet, f *Flags) {
	fs.BoolVar(&f.Debug, "debug", false, "Enable debug mode")
	fs.BoolVar(&f.QuickEdit, "quick-edit", false, "Enable quick edit mode (Windows)")
	fs.StringVar(&f.ConfigFile, "c", ".pm", "Path to the process manager configuration")
	fs.StringVar(&f.ProcFile, "f", "Procfile", "Path to the procfile configuration")
	fs.StringVar(&f.EnvFiles, "env", "", "Environment files to load, comma separated")
	fs.UintVar(&f.Port, "p", configutil.DefaultPort(0), "Port for the RPC server")
	fs.UintVar(&f.BasePort, "b", 5000, "Base port to use for processes")
	fs.BoolVar(&f.ExitOnError, "exit-on-error", false, "Exit if a process quits with an error code")
	fs.BoolVar(&f.ExitOnStop, "exit-on-stop", true, "Exit if all processess stop")
	fs.BoolVar(
		&f.StartRPCServer,
		"rpc-server",
		true,
		"Start an RPC server listening on "+configutil.DefaultAddr(),
	)
	fs.BoolVar(&f.PTY, "pty", false, "Use a PTY for all processes (noop on Windows)")
	fs.UintVar(&f.Interval, "interval", 0, "Seconds to wait between starting each process")
	fs.BoolVar(&f.ReverseOnStop, "reverse-on-stop", false, "Reverse process order when stopping")
	fs.BoolVar(&f.Overload, "overload", false, "Overwrite existing envs with the env file")
	fs.BoolVar(&f.Fork, "fork", false, "Fork processes")
	fs.BoolVar(&f.InheritStdin, "stdin", true, "Inherit stdin")
	fs.BoolVar(&f.Silent, "silent", false, "Prevent processes from using stdout/stderr")
}
