package main

import (
	"context"
	"errors"
	"flag"
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
	"golang.org/x/term"
)

// dumpCmd dumps embedded files under path to a local dump/ directory.
func dumpCmd(a ...string) error {
	return timeutil.Timer(func() error {
		err := Embed().Dump(a[0], "", func(f embedutil.File, b []byte) error {
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

// listCmd lists embedded files under path.
func listCmd(a ...string) error {
	return timeutil.Timer(func() error {
		err := Embed().List(a[0], func(files []embedutil.File) error {
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

// serverCmd starts the file server.
func serverCmd(s interface{ StartServer() error }) error {
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

// userCmd dispatches the user management subcommands: add, remove, list.
func userCmd(a ...string) error {
	usage := func() {
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintln(
			w,
			"  fileserver user add <username> [--display-name NAME]\tAdd or update a user",
		)
		fmt.Fprintln(w, "  fileserver user remove <username>\tRemove a user")
		fmt.Fprintln(w, "  fileserver user list\tList all users")
		w.Flush()
	}

	if len(a) == 0 {
		usage()
		return nil
	}

	path := db.Path(flags.DbPath)

	database, err := db.Open(path)
	if err != nil {
		return fmt.Errorf("user: open database %q: %w", path, err)
	}
	defer database.Close()

	sub := strings.ToLower(a[0])
	args := a[1:]

	switch sub {
	case "add", "a":
		return userAddCmd(database, args)
	case "remove", "rm", "r":
		return userRemoveCmd(database, args)
	case "list", "l":
		return userListCmd(database)
	default:
		usage()
		return fmt.Errorf("user: unknown subcommand %q", a[0])
	}
}

// userAddCmd adds or updates a user, prompting for a password interactively.
func userAddCmd(database *db.DB, args []string) error {
	fs := flag.NewFlagSet("user add", flag.ContinueOnError)
	displayName := fs.String("display-name", "", "Display name (auto-generated if empty)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	username := strings.TrimSpace(fs.Arg(0))
	if username == "" {
		return errors.New("usage: fileserver user add <username> [--display-name NAME]")
	}

	password, err := readPasswordConfirmed()
	if err != nil {
		return fmt.Errorf("user add: %w", err)
	}

	if err := database.UpsertUser(
		context.Background(),
		username,
		password,
		*displayName,
	); err != nil {
		return fmt.Errorf("user add: %w", err)
	}

	logutil.Infof(logutil.Get(), "User %q saved.\n", username)

	return nil
}

// userRemoveCmd removes a user by username.
func userRemoveCmd(database *db.DB, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: fileserver user remove <username>")
	}

	username := strings.TrimSpace(args[0])
	if username == "" {
		return errors.New("usage: fileserver user remove <username>")
	}

	if err := database.DeleteUser(context.Background(), username); err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return fmt.Errorf("user remove: user %q not found", username)
		}

		return fmt.Errorf("user remove: %w", err)
	}

	logutil.Infof(logutil.Get(), "User %q removed.\n", username)

	return nil
}

// userListCmd prints all users in a tabulated format.
func userListCmd(database *db.DB) error {
	users, err := database.ListUsers(context.Background())
	if err != nil {
		return fmt.Errorf("user list: %w", err)
	}

	if len(users) == 0 {
		logutil.Infof(logutil.Get(), "No users found.\n")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USERNAME\tDISPLAY NAME")
	fmt.Fprintln(w, "--------\t------------")

	for _, u := range users {
		dn := u.DisplayName
		if dn == "" {
			dn = "(auto)"
		}

		fmt.Fprintf(w, "%s\t%s\n", u.Username, dn)
	}

	return w.Flush()
}

// readPasswordConfirmed prompts for a password twice with no echo and returns it when both match.
func readPasswordConfirmed() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("stdin is not a terminal — password must be entered interactively")
	}

	fmt.Fprint(os.Stderr, "Password: ")

	p1, err := term.ReadPassword(fd)

	fmt.Fprintln(os.Stderr)

	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}

	if len(p1) == 0 {
		return "", errors.New("password must not be empty")
	}

	fmt.Fprint(os.Stderr, "Confirm password: ")

	p2, err := term.ReadPassword(fd)

	fmt.Fprintln(os.Stderr)

	if err != nil {
		return "", fmt.Errorf("reading password confirmation: %w", err)
	}

	if string(p1) != string(p2) {
		return "", errors.New("passwords do not match")
	}

	return string(p1), nil
}
