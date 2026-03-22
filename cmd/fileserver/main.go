package main

import (
	"flag"
	"os"
	"strings"

	"github.com/ricochhet/fileserver/cmd/fileserver/commands"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/server"
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
	logutil.Infof(logutil.Get(), "fileserver-%s\n", gitHash)
	logutil.Infof(logutil.Get(), "Build date: %s\n", buildDate)
	logutil.Infof(logutil.Get(), "Build on: %s\n", buildOn)
	os.Exit(0)
}

func main() {
	logutil.LogTime.Store(true)
	logutil.MaxProcNameLength.Store(0)
	logutil.Set(logutil.NewLogger("fileserver", 0))

	flags := commands.Flags
	logutil.SetDebug(flags.Debug)
	_ = cmdutil.QuickEdit(flags.QuickEdit)

	cmd, err := handleCommands()
	if err != nil {
		logutil.Errorf(logutil.Get(), "Error running command: %v\n", err)
	}

	if cmd {
		return
	}

	s := server.NewServer(flags.ConfigFile, flags.Hosts, &configutil.TLS{
		Enabled:  true,
		CertFile: flags.CertFile,
		KeyFile:  flags.KeyFile,
	}, Embed(), flags.DbPath)
	if err := commands.ServerCmd(s); err != nil {
		logutil.Errorf(logutil.Get(), "%v\n", err)
	}
}

// handleCommands handles the specified command flags.
func handleCommands() (bool, error) {
	args := flag.Args()
	if len(args) == 0 {
		return false, nil
	}

	cmd := strings.ToLower(args[0])
	rest := args[1:]
	cmds := commands.Cmds

	switch cmd {
	case "dump", "d":
		cmds.Check(1)
		return true, commands.DumpCmd(Embed(), rest...)
	case "list", "l":
		cmds.Check(1)
		return true, commands.ListCmd(Embed(), rest...)
	case "user", "u":
		return true, commands.UserCmd(rest...)
	case "channel", "ch":
		return true, commands.ChannelCmd(rest...)
	case "help", "h":
		cmds.Usage()
	case "version", "v":
		version()
	default:
		cmds.Usage()
	}

	if winutil.IsAdmin() {
		cmdutil.Pause()
	}

	return false, nil
}
