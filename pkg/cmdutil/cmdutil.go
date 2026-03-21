package cmdutil

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"slices"
	"text/tabwriter"

	"github.com/ricochhet/fileserver/pkg/logutil"
)

type Commands []*Command

type Command struct {
	Usage string
	Desc  string
}

// Usage runs flag.PrintDefaults() and exits with code 0.
func (c *Commands) Usage() {
	tw := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Usage:")

	for _, c := range *c {
		fmt.Fprintf(tw, "  %s\t# %s\n", c.Usage, c.Desc)
	}

	tw.Flush()
	flag.PrintDefaults()
	os.Exit(0)
}

// Supports exits if runtime.GOOS is not in the list.
func Supports(list ...string) {
	if slices.Contains(list, runtime.GOOS) {
		return
	}

	logutil.Errorf(logutil.Get(), "This command is unsupported on %s.", runtime.GOOS)
	os.Exit(1)
}

// Check checks if flag.Narg() < v+1, calling Usage() if true.
func (c *Commands) Check(v int) {
	if flag.NArg() < v+1 {
		c.Usage()
	}
}

// Pause pauses the output so it can be visualized before closing.
func Pause() {
	logutil.Infof(logutil.Get(), "Press Enter to continue...\n")
	bufio.NewScanner(os.Stdin).Scan()
}
