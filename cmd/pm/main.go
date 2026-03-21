package main

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/ricochhet/fileserver/cmd/pm/internal/proc"
	"github.com/ricochhet/fileserver/pkg/cmdutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"github.com/ricochhet/fileserver/pkg/winutil"
)

var (
	buildDate string
	gitHash   string
	buildOn   string
)

func version() {
	logutil.Infof(logutil.Get(), "pm-%s\n", gitHash)
	logutil.Infof(logutil.Get(), "Build date: %s\n", buildDate)
	logutil.Infof(logutil.Get(), "Build on: %s\n", buildOn)
	os.Exit(0)
}

func main() {
	logutil.LogTime.Store(true)
	logutil.MaxProcNameLength.Store(0)
	logutil.Set(logutil.NewLogger("pm", 0))
	logutil.SetDebug(flags.Debug)
	_ = cmdutil.QuickEdit(flags.QuickEdit)

	ctx := &proc.Context{
		Options: &proc.Options{
			ConfigFile:     flags.ConfigFile,
			ProcFile:       flags.ProcFile,
			EnvFiles:       flags.EnvFiles,
			Port:           flags.Port,
			BasePort:       flags.BasePort,
			ExitOnError:    flags.ExitOnError,
			ExitOnStop:     flags.ExitOnStop,
			StartRPCServer: flags.StartRPCServer,
			PTY:            flags.PTY,
			Interval:       flags.Interval,
			ReverseOnStop:  flags.ReverseOnStop,
			Overload:       flags.Overload,
			Fork:           flags.Fork,
			InheritStdin:   flags.InheritStdin,
			Silent:         flags.Silent,
		},
	}

	cmd, err := commands(ctx)
	if err != nil {
		logutil.Errorf(logutil.Get(), "Error running command: %v\n", err)
	}

	if cmd {
		return
	}

	if err := ctx.Start(context.Background(), proc.NotifyCh()); err != nil {
		logutil.Errorf(logutil.Get(), "%w\n", err)
	}
}

// commands handles the specified command flags.
func commands(ctx *proc.Context) (bool, error) {
	args := flag.Args()
	if len(args) == 0 {
		return false, nil
	}

	cmd := strings.ToLower(args[0])
	rest := args[1:]

	switch cmd {
	case "run":
		cmds.Check(2)
		return true, runCmd(rest...)
	case "check", "c":
		return true, checkCmd(ctx)
	case "export", "e":
		return true, exportCmd(ctx, rest...)
	case "help", "h":
		cmds.Usage()
	case "version", "v":
		version()
	}

	if winutil.IsAdmin() {
		cmdutil.Pause()
	}

	return false, nil
}
