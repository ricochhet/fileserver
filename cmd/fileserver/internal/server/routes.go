package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/httprate"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/browse"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

// startServer registers all handlers and begins listening for a single server config entry.
func (c *Context) startServer(
	baseCtx context.Context,
	ctx *serverutil.Context,
	cfg *configutil.Server,
) error {
	browseRateLimit := 500
	if cfg.BrowseRateLimit != 0 {
		browseRateLimit = cfg.BrowseRateLimit
	}

	fileRateLimit := 100
	if cfg.FileRateLimit != 0 {
		fileRateLimit = cfg.FileRateLimit
	}

	browseLimit := httprate.LimitByIP(browseRateLimit, time.Minute)
	fileLimit := httprate.LimitByIP(fileRateLimit, time.Minute)

	if err := c.serveFileHandler(ctx, cfg, browseLimit, fileLimit); err != nil {
		return errutil.New("c.serveFileHandler", err)
	}

	if err := c.serveContentHandler(ctx, cfg, browseLimit); err != nil {
		return errutil.New("c.serveContentHandler", err)
	}

	srv := ctx.ListenAndServe(baseCtx, fmt.Sprintf(":%d", cfg.Port))
	c.servers = append(c.servers, srv)

	return nil
}

// serveFileHandler registers browse and file-serving routes for each FileEntry.
func (c *Context) serveFileHandler(
	ctx *serverutil.Context,
	cfg *configutil.Server,
	browseLimit, fileLimit func(http.Handler) http.Handler,
) error {
	for _, f := range cfg.FileEntries {
		info, err := os.Stat(f.Path)
		if err != nil {
			return errutil.WithFramef("invalid path %s: %w", f.Path, err)
		}

		if info.IsDir() && f.Browse != "" {
			if cfg.Features.DisableBrowse {
				logutil.Infof(
					logutil.Get(),
					"Port %d: browse feature disabled — skipping browse route for %s\n",
					cfg.Port,
					f.Browse,
				)
			} else {
				route := strings.TrimSuffix(f.Browse, "/")
				h := browseLimit(
					wrapBasicAuth(
						f.BasicAuth,
						browse.Handler(c.FS, f.Path, route, cfg.Hidden, cfg),
					),
				)

				logutil.Infof(
					logutil.Get(),
					"Port %d: %s/** -> %s (browse)\n",
					cfg.Port,
					route,
					f.Path,
				)

				ctx.Handle(route, h)
				ctx.Handle(route+"/*", h)
			}
		}

		if info.IsDir() {
			if err := matchPattern(f, ctx, cfg, fileLimit); err != nil {
				return errutil.New("matchPattern", err)
			}
		} else {
			if err := matchFile(f, ctx, cfg, fileLimit); err != nil {
				return errutil.New("matchFile", err)
			}
		}
	}

	return nil
}

// serveContentHandler registers routes for each ContentEntry.
func (c *Context) serveContentHandler(
	ctx *serverutil.Context,
	cfg *configutil.Server,
	limit func(http.Handler) http.Handler,
) error {
	for _, f := range cfg.ContentEntries {
		if f.Dir != "" {
			if err := c.serveEmbeddedDir(ctx, cfg, f, limit); err != nil {
				return errutil.New("c.serveEmbeddedDir", err)
			}

			continue
		}

		logutil.Infof(
			logutil.Get(),
			"Port %d: %s -> %s\n",
			cfg.Port,
			f.Route,
			f.Name,
		)

		b, err := embedutil.MaybeBase64(c.FS, f.Base64)
		if err != nil {
			return errutil.WithFrame(err)
		}

		ctx.Handle(f.Route, limit(serverutil.ServeContentHandler(f.Info, f.Name, b)))
	}

	return nil
}

// serveEmbeddedDir walks an embedded directory and registers a route for each
// file, skipping any whose relative path matches an Exclude glob pattern.
func (c *Context) serveEmbeddedDir(
	ctx *serverutil.Context,
	cfg *configutil.Server,
	f configutil.ContentEntry,
	limit func(http.Handler) http.Handler,
) error {
	dirPath := strings.TrimPrefix(f.Dir, "asset:")
	fullPath := path.Join(c.FS.Initial, dirPath)
	baseRoute := strings.TrimSuffix(f.Route, "/")

	return c.FS.List(fullPath, func(files []embedutil.File) error {
		for _, file := range files {
			rel, err := filepath.Rel(fullPath, file.Path)
			if err != nil {
				return errutil.WithFramef("cannot relativise %s: %w", file.Path, err)
			}

			rel = filepath.ToSlash(rel)

			if isExcluded(rel, f.Exclude) {
				logutil.Infof(logutil.Get(), "Port %d: skipping excluded asset %s\n", cfg.Port, rel)
				continue
			}

			route := baseRoute + "/" + rel
			b := embedutil.MaybeRead(c.FS, path.Join(dirPath, rel))

			logutil.Infof(
				logutil.Get(),
				"Port %d: %s -> %s (embedded)\n",
				cfg.Port,
				route,
				file.Path,
			)

			ctx.Handle(route, limit(serverutil.ServeContentHandler(f.Info, path.Base(rel), b)))
		}

		return nil
	})
}

// isExcluded reports whether rel matches any of the given glob patterns.
func isExcluded(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := path.Match(pattern, rel)
		if err == nil && matched {
			return true
		}
	}

	return false
}
