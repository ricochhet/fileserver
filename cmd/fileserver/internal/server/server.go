package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/browse"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/chat"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/hostsutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
	"github.com/ricochhet/fileserver/pkg/jsonutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

const (
	defaultChatRoute = "/chat"
	chatTmpl         = "chat"
	chatTmplHTML     = "chat.html"
)

// Context is the top-level server runtime, created once and shared across all configured instances.
type Context struct {
	ConfigFile string
	Hosts      bool
	TLS        *configutil.TLS
	FS         *embedutil.EmbeddedFileSystem
	DbPath     string

	servers    []*http.Server
	chatStore  *chat.Store
	db         *db.DB
	baseCancel context.CancelFunc
}

// NewServer returns a Context with the database and chat store initialized.
func NewServer(
	configFile string,
	hosts bool,
	tls *configutil.TLS,
	fs *embedutil.EmbeddedFileSystem,
	dbPath string,
) *Context {
	s := &Context{}

	if configFile != "" {
		s.ConfigFile = configFile
	}

	s.Hosts = hosts
	s.TLS = tls
	s.FS = fs
	s.DbPath = dbPath

	database, err := db.Open(db.Path(dbPath))
	if err != nil {
		logutil.Errorf(
			logutil.Get(),
			"Warning: could not open database, running without persistence: %v\n",
			err,
		)
	} else {
		s.db = database

		logutil.Infof(logutil.Get(), "Database opened: %s\n", db.Path(dbPath))
	}

	s.chatStore = chat.NewStore(s.db)

	return s
}

// renderChatTemplate executes chat.html as a Go template, injecting the chat route.
// text/template is used intentionally — html/template applies context-aware JS escaping.
func renderChatTemplate(src []byte, chatRoute string) ([]byte, error) {
	tmpl, err := template.New(chatTmpl).Parse(string(src))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ ChatRoute string }{chatRoute}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// StartServer loads config, starts all HTTP servers, and blocks until shutdown.
func (c *Context) StartServer() error {
	config, err := c.maybeReadConfig(c.ConfigFile)
	if err != nil {
		return errutil.New("c.maybeReadConfig", err)
	}

	if err := c.addHosts(config); err != nil {
		return errutil.New("c.addHosts", err)
	}

	c.maybeTLS(config)

	// baseCtx is canceled before Shutdown() so SSE streams exit promptly rather than timing out.
	baseCtx, baseCancel := context.WithCancel(context.Background())
	c.baseCancel = baseCancel

	for _, cfg := range config.Servers {
		ctx := serverutil.NewContext()

		maxAge := 300
		if cfg.MaxAge != 0 {
			maxAge = cfg.MaxAge
		}

		r := chi.NewRouter()
		r.Use(middleware.Recoverer)
		r.Use(serverutil.WithLogging)
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"https://*", "http://*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: cfg.AllowCredentials,
			MaxAge:           maxAge,
		}))

		r.NotFound(c.NotFoundHandler)

		ctx.SetLocked(&serverutil.HTTPServer{
			Router:   r,
			TLS:      c.TLS,
			Timeouts: &cfg.Timeouts,
		})

		if len(cfg.FormAuth.Users) > 0 {
			secret := resolveFormAuthSecret(cfg.FormAuth.Secret)

			if c.db != nil {
				c.seedUsersFromConfig(cfg.FormAuth.Users)
			}

			r.Use(withFormAuth(cfg.FormAuth.Users, secret, cfg.FormAuth.PublicPrefixes, c.db))
			c.registerAuthRoutes(ctx.Handle, cfg.FormAuth.Users, secret, c.db)

			logutil.Infof(
				logutil.Get(),
				"Port %d: form auth enabled (%d user(s))\n",
				cfg.Port,
				len(cfg.FormAuth.Users),
			)
		} else if cfg.BasicAuth.Username != "" && cfg.BasicAuth.Password != "" {
			r.Use(withBasicAuth(cfg.BasicAuth.Username, cfg.BasicAuth.Password))

			logutil.Infof(
				logutil.Get(),
				"Port %d: basic auth enabled for user %q\n",
				cfg.Port,
				cfg.BasicAuth.Username,
			)
		}

		resolver := func(req *http.Request) (string, string) {
			return usernameFromCtx(req), displayNameFromCtx(req)
		}

		if !cfg.Features.DisableChat {
			chatRoute := cfg.Features.ChatRoute
			if chatRoute == "" {
				chatRoute = defaultChatRoute
			}

			chatHTML, err := renderChatTemplate(embedutil.MaybeRead(c.FS, chatTmplHTML), chatRoute)
			if err != nil {
				return errutil.WithFramef("renderChatTemplate: %w", err)
			}

			r.Mount(chatRoute, chat.Handler(c.chatStore, resolver, chatHTML))
			logutil.Infof(logutil.Get(), "Port %d: %s mounted\n", cfg.Port, chatRoute)

			for _, ch := range cfg.ChatChannels {
				c.chatStore.SeedChannel(ch.Code, ch.Name)
				logutil.Infof(
					logutil.Get(),
					"Port %d: seeded chat channel %q (%s)\n",
					cfg.Port,
					ch.Name,
					ch.Code,
				)
			}
		} else {
			logutil.Infof(logutil.Get(), "Port %d: chat feature disabled\n", cfg.Port)
		}

		if err := c.startServer(baseCtx, ctx, &cfg); err != nil {
			return errutil.New("c.startServer", err)
		}
	}

	if err := c.removeHosts(config); err != nil {
		return errutil.New("c.removeHosts", err)
	}

	c.shutdown()

	return nil
}

// seedUsersFromConfig upserts all config-defined form-auth users into the database.
func (c *Context) seedUsersFromConfig(users []configutil.FormAuthUser) {
	for _, u := range users {
		dn := u.DisplayName
		if dn == "" {
			dn = generateDisplayName(u.Username)
		}

		if err := c.db.UpsertUser(context.Background(), u.Username, u.Password, dn); err != nil {
			logutil.Errorf(logutil.Get(), "seedUsersFromConfig: upsert %q: %v\n", u.Username, err)
		}
	}
}

// withBasicAuth returns middleware enforcing HTTP Basic Authentication.
func withBasicAuth(user, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || u != user || p != password {
				httputil.BasicAuthChallenge(w, "fileserver")
				errutil.HTTPUnauthorized(w)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// wrapBasicAuth wraps a handler with Basic Auth when credentials are non-empty.
func wrapBasicAuth(auth configutil.BasicAuth, h http.Handler) http.Handler {
	if auth.Username == "" || auth.Password == "" {
		return h
	}

	return withBasicAuth(auth.Username, auth.Password)(h)
}

// shutdown waits for interrupt or SIGTERM, cancels in-flight requests, then
// gracefully drains all servers (up to 10 s) before closing the DB.
func (c *Context) shutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Cancel the base context first so SSE streams and other long-lived
	// handlers return promptly rather than waiting out the shutdown timeout.
	if c.baseCancel != nil {
		c.baseCancel()
	}

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	for _, srv := range c.servers {
		if err := srv.Shutdown(shutCtx); err != nil {
			logutil.Errorf(logutil.Get(), "Error shutting down server: %v\n", err)
		}
	}

	// Close the DB after servers drain so any in-flight SaveMessage calls complete.
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			logutil.Errorf(logutil.Get(), "Error closing database: %v\n", err)
		}
	}
}

// addHosts adds config-defined hosts entries to the system hosts file.
func (c *Context) addHosts(cfg *configutil.Config) error {
	if !c.isHostsValid(cfg) {
		return nil
	}

	hf, err := hostsutil.NewHosts()
	if err != nil {
		return errutil.New("hostsutil.NewHosts", err)
	}

	return hostsutil.Add(hf, cfg.Hosts)
}

// removeHosts removes config-defined hosts entries from the system hosts file.
func (c *Context) removeHosts(cfg *configutil.Config) error {
	if !c.isHostsValid(cfg) {
		return nil
	}

	hf, err := hostsutil.NewHosts()
	if err != nil {
		return errutil.New("hostsutil.NewHosts", err)
	}

	return hostsutil.Remove(hf, cfg.Hosts)
}

// isHostsValid reports whether hosts management is enabled and entries are present.
func (c *Context) isHostsValid(cfg *configutil.Config) bool {
	return c.Hosts && len(cfg.Hosts) != 0
}

// maybeTLS resolves TLS config from flags and config file, preferring flags.
func (c *Context) maybeTLS(cfg *configutil.Config) {
	if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
		c.TLS.Enabled = false
	}

	if fsutil.Exists(c.TLS.CertFile) && fsutil.Exists(c.TLS.KeyFile) {
		c.TLS.Enabled = true
		return
	}

	if fsutil.Exists(cfg.TLS.CertFile) && fsutil.Exists(cfg.TLS.KeyFile) {
		c.TLS = &cfg.TLS
	}
}

// maybeReadConfig reads the config file if it exists, falling back to a default config.
func (c *Context) maybeReadConfig(path string) (*configutil.Config, error) {
	exists := fsutil.Exists(path)

	switch {
	case exists:
		config, err := jsonutil.ReadAndUnmarshal[configutil.Config](path)
		if err != nil {
			logutil.Errorf(logutil.Get(), "Error reading server config: %v\n", err)
		}

		return config, err

	case !exists && path != "":
		return nil, fmt.Errorf("path specified but does not exist: %s", path)

	default:
		logutil.Infof(logutil.Get(), "Starting with default server config\n")
		return c.newDefaultConfig(), nil
	}
}

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
