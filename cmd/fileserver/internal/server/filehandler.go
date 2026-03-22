package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

// matchPattern walks a directory and registers a route for each file found.
func matchPattern(
	f configutil.FileEntry,
	ctx *serverutil.Context,
	cfg *configutil.Server,
	fileLimit func(http.Handler) http.Handler,
) error {
	return filepath.Walk(f.Path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return errutil.WithFrame(err)
		}

		if info.IsDir() {
			return nil
		}

		abs, err := filepath.Abs(p)
		if err != nil {
			return errutil.WithFramef("invalid path %s: %w", p, err)
		}

		rel, err := filepath.Rel(f.Path, p)
		if err != nil {
			return errutil.WithFramef("cannot get relative path for %s: %w", p, err)
		}

		route := filepath.ToSlash(filepath.Join(f.Route, rel))
		logutil.Infof(logutil.Get(), "Port %d: %s -> %s\n", cfg.Port, route, abs)

		ctx.Handle(
			route,
			fileLimit(wrapBasicAuth(f.BasicAuth, serverutil.ServeFileHandler(f.Info, abs))),
		)

		return nil
	})
}

// matchFile registers a route for a single file path.
func matchFile(
	f configutil.FileEntry,
	ctx *serverutil.Context,
	cfg *configutil.Server,
	fileLimit func(http.Handler) http.Handler,
) error {
	abs, err := filepath.Abs(f.Path)
	if err != nil {
		return errutil.WithFramef("invalid path %s: %w", f.Path, err)
	}

	logutil.Infof(logutil.Get(), "Port %d: %s -> %s\n", cfg.Port, f.Route, abs)

	ctx.Handle(
		f.Route,
		fileLimit(wrapBasicAuth(f.BasicAuth, serverutil.ServeFileHandler(f.Info, abs))),
	)

	return nil
}
