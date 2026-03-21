package main

import (
	"github.com/ricochhet/fileserver/cmd/pm/internal/proc"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"github.com/ricochhet/fileserver/pkg/timeutil"
)

// runCmd command.
func runCmd(a ...string) error {
	return timeutil.Timer(func() error {
		err := proc.Run(a[0], a[1:])
		if err != nil {
			logutil.Errorf(logutil.Get(), "Error running RPC command: %v\n", err)
		}

		return err
	}, "Run", func(_, elapsed string) {
		logutil.Infof(logutil.Get(), "Took %s\n", elapsed)
	})
}

// checkCmd command.
func checkCmd(c *proc.Context) error {
	return timeutil.Timer(func() error {
		err := c.Check()
		if err != nil {
			logutil.Errorf(logutil.Get(), "Error checking procfile: %v\n", err)
		}

		return err
	}, "Check", func(_, elapsed string) {
		logutil.Infof(logutil.Get(), "Took %s\n", elapsed)
	})
}

// exportCmd command.
func exportCmd(c *proc.Context, a ...string) error {
	return timeutil.Timer(func() error {
		err := c.Export(a[0], a[1])
		if err != nil {
			logutil.Errorf(logutil.Get(), "Error exporting procfile: %v\n", err)
		}

		return err
	}, "Export", func(_, elapsed string) {
		logutil.Infof(logutil.Get(), "Took %s\n", elapsed)
	})
}
