package main

import (
	"flag"

	"github.com/ricochhet/fileserver/pkg/cmdutil"
)

type Flags struct {
	Debug     bool
	QuickEdit bool

	ConfigFile string
	CertFile   string
	KeyFile    string

	DbPath string
	Hosts  bool
}

var (
	flags = NewFlags()
	cmds  = cmdutil.Commands{
		{Usage: "fileserver help", Desc: "Show this help"},
		{Usage: "fileserver list [PATH]", Desc: "List embedded files"},
		{Usage: "fileserver dump [PATH]", Desc: "Dump embedded files to disk"},
		{Usage: "fileserver version", Desc: "Display fileserver version"},
		{
			Usage: "fileserver user add <username> [--display-name NAME] [--admin]",
			Desc:  "Add or update a user (--admin grants administrative privileges)",
		},
		{Usage: "fileserver user remove <username>", Desc: "Remove a user"},
		{Usage: "fileserver user list", Desc: "List all users"},
		{
			Usage: "fileserver channel add <code> [--name NAME]",
			Desc:  "Add or update a channel",
		},
		{Usage: "fileserver channel remove <code>", Desc: "Remove a channel"},
		{Usage: "fileserver channel list", Desc: "List all channels"},
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
	fs.StringVar(&f.ConfigFile, "c", "fileserver.json", "Path to file server configuration")
	fs.StringVar(&f.CertFile, "cert", "", "TLS cert")
	fs.StringVar(&f.KeyFile, "key", "", "TLS key")
	fs.StringVar(&f.DbPath, "dbpath", "fileserver.db", "Fileserver database path")
	fs.BoolVar(&f.Hosts, "hosts", false, "Modify hosts according to configuration")
}
