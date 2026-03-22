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

// Subcommand name constants shared across entity dispatchers.
const (
	subCmdAdd    = "add"
	subCmdRemove = "remove"
	subCmdList   = "list"
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

// runEntityCmd is the shared skeleton for entity management commands (user, channel, …).
// It prints usageLines when called with no args, opens the database, then hands off
// to dispatch(sub, args, db). dispatch should return a non-nil error for unknown subs.
func runEntityCmd(
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

	path := db.Path(flags.DbPath)

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

// errUnknownSubCmd is returned by a dispatch func when the subcommand is not recognized.
var errUnknownSubCmd = errors.New("unknown subcommand")

// channelCmd dispatches the channel management subcommands: add, remove, list.
func channelCmd(a ...string) error {
	return runEntityCmd("channel", a, []string{
		"  fileserver channel add <code> [--name NAME]\tAdd or update a channel",
		"  fileserver channel remove <code>\tRemove a channel",
		"  fileserver channel list\tList all channels",
	}, func(sub string, args []string, database *db.DB) error {
		switch sub {
		case subCmdAdd, "a":
			return channelAddCmd(database, args)
		case subCmdRemove, "rm", "r":
			return channelRemoveCmd(database, args)
		case subCmdList, "l":
			return channelListCmd(database)
		default:
			return errUnknownSubCmd
		}
	})
}

// channelAddCmd adds or updates a channel by code with an optional display name.
func channelAddCmd(database *db.DB, args []string) error {
	fs := flag.NewFlagSet("channel add", flag.ContinueOnError)
	name := fs.String("name", "", "Display name (defaults to \"#<code>\" if empty)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	code := strings.TrimSpace(fs.Arg(0))
	if code == "" {
		return errors.New("usage: fileserver channel add <code> [--name NAME]")
	}

	n := strings.TrimSpace(*name)
	if n == "" {
		n = "#" + code
	}

	if err := database.UpsertChannel(context.Background(), code, n); err != nil {
		return fmt.Errorf("channel add: %w", err)
	}

	logutil.Infof(logutil.Get(), "Channel %q (%s) saved.\n", code, n)

	return nil
}

// channelRemoveCmd removes a channel by code.
func channelRemoveCmd(database *db.DB, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: fileserver channel remove <code>")
	}

	code := strings.TrimSpace(args[0])
	if code == "" {
		return errors.New("usage: fileserver channel remove <code>")
	}

	if err := database.DeleteChannel(context.Background(), code); err != nil {
		if errors.Is(err, db.ErrChannelNotFound) {
			return fmt.Errorf("channel remove: channel %q not found", code)
		}

		return fmt.Errorf("channel remove: %w", err)
	}

	logutil.Infof(logutil.Get(), "Channel %q removed.\n", code)

	return nil
}

// channelListCmd prints all channels in a tabulated format.
func channelListCmd(database *db.DB) error {
	channels, err := database.ListChannels(context.Background())
	if err != nil {
		return fmt.Errorf("channel list: %w", err)
	}

	if len(channels) == 0 {
		logutil.Infof(logutil.Get(), "No channels found.\n")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CODE\tNAME")
	fmt.Fprintln(w, "----\t----")

	for _, ch := range channels {
		fmt.Fprintf(w, "%s\t%s\n", ch.Code, ch.Name)
	}

	return w.Flush()
}

// userCmd dispatches the user management subcommands: add, remove, list.
func userCmd(a ...string) error {
	return runEntityCmd("user", a, []string{
		"  fileserver user add <username> [--display-name NAME]\tAdd or update a user",
		"  fileserver user remove <username>\tRemove a user",
		"  fileserver user list\tList all users",
	}, func(sub string, args []string, database *db.DB) error {
		switch sub {
		case subCmdAdd, "a":
			return userAddCmd(database, args)
		case subCmdRemove, "rm", "r":
			return userRemoveCmd(database, args)
		case subCmdList, "l":
			return userListCmd(database)
		default:
			return errUnknownSubCmd
		}
	})
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
