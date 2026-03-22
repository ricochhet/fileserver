package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"golang.org/x/term"
)

// UserCmd dispatches the user management subcommands: add, remove, list.
func UserCmd(a ...string) error {
	return Flags.runEntityCmd("user", a, []string{
		"  fileserver user add <username> [--display-name NAME] [--admin]\tAdd or update a user",
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
// Pass --admin to grant the user administrative privileges.
func userAddCmd(database *db.DB, args []string) error {
	fs := flag.NewFlagSet("user add", flag.ContinueOnError)
	displayName := fs.String("display-name", "", "Display name (auto-generated if empty)")
	isAdmin := fs.Bool("admin", false, "Grant admin privileges to this user")

	username, flagArgs := extractPositional(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("usage: fileserver user add <username> [--display-name NAME] [--admin]")
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
		*isAdmin,
	); err != nil {
		return fmt.Errorf("user add: %w", err)
	}

	adminNote := ""
	if *isAdmin {
		adminNote = " (admin)"
	}

	logutil.Infof(logutil.Get(), "User %q saved%s.\n", username, adminNote)

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

// userListCmd prints all users in a tabulated format, including their admin status.
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
	fmt.Fprintln(w, "USERNAME\tDISPLAY NAME\tADMIN")
	fmt.Fprintln(w, "--------\t------------\t-----")

	for _, u := range users {
		dn := u.DisplayName
		if dn == "" {
			dn = "(auto)"
		}

		admin := ""
		if u.IsAdmin {
			admin = "yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", u.Username, dn, admin)
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
