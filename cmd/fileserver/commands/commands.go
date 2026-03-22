package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"github.com/ricochhet/fileserver/pkg/timeutil"
)

// Subcommand name constants shared across entity dispatchers.
const (
	subCmdAdd    = "add"
	subCmdRemove = "remove"
	subCmdList   = "list"
)

// errUnknownSubCmd is returned by a dispatch func when the subcommand is not recognized.
var errUnknownSubCmd = errors.New("unknown subcommand")

// DumpCmd dumps embedded files under path to a local dump/ directory.
func DumpCmd(e *embedutil.EmbeddedFileSystem, a ...string) error {
	return timeutil.Timer(func() error {
		err := e.Dump(a[0], "", func(f embedutil.File, b []byte) error {
			logutil.Infof(logutil.Get(), "Writing: %s (%d bytes)\n", f.Path, f.Info.Size())
			return fsutil.Write(filepath.Join("dump", f.Path), b)
		})
		if err != nil {
			logutil.Errorf(logutil.Get(), "Error dumping embedded files: %v\n", err)
		}

		return err
	}, "Dump", func(_, elapsed string) {
		logutil.Infof(logutil.Get(), "Took %s\n", elapsed)
	})
}

// ListCmd lists embedded files under path.
func ListCmd(e *embedutil.EmbeddedFileSystem, a ...string) error {
	return timeutil.Timer(func() error {
		err := e.List(a[0], func(files []embedutil.File) error {
			for _, f := range files {
				logutil.Infof(logutil.Get(), "%s (%d bytes)\n", f.Path, f.Info.Size())
			}

			return nil
		})
		if err != nil {
			logutil.Errorf(logutil.Get(), "Error listing embedded files: %v\n", err)
		}

		return err
	}, "List", func(_, elapsed string) {
		logutil.Infof(logutil.Get(), "Took %s\n", elapsed)
	})
}

// ServerCmd starts the file server.
func ServerCmd(s interface{ StartServer() error }) error {
	return timeutil.Timer(func() error {
		err := s.StartServer()
		if err != nil {
			logutil.Errorf(logutil.Get(), "Error starting server: %v\n", err)
		}

		return err
	}, "Server", func(_, elapsed string) {
		logutil.Infof(logutil.Get(), "Took %s\n", elapsed)
	})
}

// extractPositional splits args into (positional, flagArgs) by pulling out the
// first token that does not start with "-".
func extractPositional(args []string) (positional string, flagArgs []string) {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") && positional == "" {
			positional = a
		} else {
			flagArgs = append(flagArgs, a)
		}
	}

	return positional, flagArgs
}

// runEntityCmd is the shared skeleton for entity management commands.
// It prints usageLines when called with no args, opens the database, then hands off
// to dispatch(sub, args, db). dispatch should return a non-nil error for unknown subs.
func (f *FlagSet) runEntityCmd(
	entity string,
	args []string,
	usageLines []string,
	dispatch func(sub string, args []string, database *db.DB) error,
) error {
	usage := func() {
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Usage:")

		for _, l := range usageLines {
			fmt.Fprintln(w, l)
		}

		w.Flush()
	}

	if len(args) == 0 {
		usage()
		return nil
	}

	path := db.Path(f.DbPath)

	database, err := db.Open(path)
	if err != nil {
		return fmt.Errorf("%s: open database %q: %w", entity, path, err)
	}
	defer database.Close()

	sub := strings.ToLower(args[0])
	rest := args[1:]

	if err := dispatch(sub, rest, database); err != nil {
		if errors.Is(err, errUnknownSubCmd) {
			usage()
			return fmt.Errorf("%s: unknown subcommand %q", entity, args[0])
		}

		return err
	}

	return nil
}
