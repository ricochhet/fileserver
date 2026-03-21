package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ricochhet/fileserver/pkg/errutil"
)

// exportUpstart exports the procfile in upstart format.
func (c *Context) exportUpstart(path string) error {
	for i, proc := range c.GetLocked().Procs {
		f, err := os.Create(filepath.Join(path, "app-"+proc.Name+".conf"))
		if err != nil {
			return errutil.New("os.Create", err)
		}

		fmt.Fprintf(f, "start on starting app-%s\n", proc.Name)
		fmt.Fprintf(f, "stop on stopping app-%s\n", proc.Name)
		fmt.Fprintf(f, "respawn\n")
		fmt.Fprintf(f, "\n")

		env := map[string]string{}

		procfile, err := filepath.Abs(c.GetLocked().ProcFile)
		if err != nil {
			return errutil.New("filepath.Abs", err)
		}

		b, err := os.ReadFile(filepath.Join(filepath.Dir(procfile), ".env"))
		if err == nil {
			for line := range strings.SplitSeq(string(b), "\n") {
				token := strings.SplitN(line, "=", 2)
				if len(token) != 2 {
					continue
				}

				token[0] = strings.TrimPrefix(token[0], "export ")
				token[0] = strings.TrimSpace(token[0])
				token[1] = strings.TrimSpace(token[1])
				env[token[0]] = token[1]
			}
		}

		fmt.Fprintf(f, "env PORT=%d\n", c.GetLocked().BasePort+uint(i))

		for k, v := range env {
			fmt.Fprintf(f, "env %s='%s'\n", k, strings.ReplaceAll(v, "'", "\\'"))
		}

		fmt.Fprintf(f, "\n")
		fmt.Fprintf(f, "setuid app\n")
		fmt.Fprintf(f, "\n")
		fmt.Fprintf(f, "chdir %s\n", filepath.ToSlash(filepath.Dir(procfile)))
		fmt.Fprintf(f, "\n")
		fmt.Fprintf(f, "exec %s\n", proc.Cmdline)

		f.Close()
	}

	return nil
}

func (c *Context) Export(format, path string) error {
	err := c.Parse(c.readProcFile())
	if err != nil {
		return errutil.New("c.config.Parse", err)
	}

	err = os.MkdirAll(path, 0o755)
	if err != nil {
		return errutil.New("os.MkdirAll", err)
	}

	if format == "upstart" {
		return c.exportUpstart(path)
	}

	return nil
}
