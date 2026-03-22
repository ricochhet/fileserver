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
)

// ChannelCmd dispatches the channel management subcommands: add, remove, list.
func ChannelCmd(a ...string) error {
	return Flags.runEntityCmd("channel", a, []string{
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

	code, flagArgs := extractPositional(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	code = strings.TrimSpace(code)
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
