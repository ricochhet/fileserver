package server

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/admin"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/auth"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/chat"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

const (
	defaultChatRoute = "/chat"
	chatTmpl         = "chat"
	chatTmplHTML     = "chat.html"
)

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

	baseCtx, baseCancel := context.WithCancel(context.Background())
	c.baseCancel = baseCancel

	for _, cfg := range config.Servers {
		if err := c.bootServer(baseCtx, cfg); err != nil {
			return err
		}
	}

	if err := c.removeHosts(config); err != nil {
		return errutil.New("c.removeHosts", err)
	}

	c.shutdown()

	return nil
}

// bootServer configures middleware, mounts features, and starts listening for
// a single server config entry.
func (c *Context) bootServer(baseCtx context.Context, cfg configutil.Server) error {
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
		c.mountFormAuth(baseCtx, r, ctx, cfg)
	} else if cfg.BasicAuth.Username != "" && cfg.BasicAuth.Password != "" {
		r.Use(withBasicAuth(cfg.BasicAuth.Username, cfg.BasicAuth.Password))
		logutil.Infof(
			logutil.Get(),
			"Port %d: basic auth enabled for user %q\n",
			cfg.Port,
			cfg.BasicAuth.Username,
		)
	}

	c.maybeChat(r, cfg)

	return c.startServer(baseCtx, ctx, &cfg)
}

// mountFormAuth enables form auth middleware, seeds users, registers auth
// routes, and mounts the admin handler. Only when form auth is configured.
func (c *Context) mountFormAuth(
	ctx context.Context,
	r *chi.Mux,
	srvCtx *serverutil.Context,
	cfg configutil.Server,
) {
	secret := auth.ResolveFormAuthSecret(cfg.FormAuth.Secret)

	if c.db != nil {
		c.seedUsersFromConfig(ctx, cfg.FormAuth.Users)
	}

	r.Use(auth.WithFormAuth(cfg.FormAuth.Users, secret, cfg.FormAuth.PublicPrefixes, c.db))
	auth.RegisterAuthRoutes(srvCtx.Handle, cfg.FormAuth.Users, secret, c.db, c.FS)

	adminResolver := func(req *http.Request) (string, bool) {
		return auth.UsernameFromCtx(req), auth.IsAdminFromCtx(req)
	}

	adminRoute := admin.DefaultAdminRoute
	if cfg.Features.AdminRoute != "" {
		adminRoute = cfg.Features.AdminRoute
	}

	r.Mount(adminRoute, admin.Handler(c.db, cfg.UploadDir, adminResolver))

	logutil.Infof(
		logutil.Get(),
		"Port %d: form auth enabled (%d user(s))\n",
		cfg.Port,
		len(cfg.FormAuth.Users),
	)
	logutil.Infof(logutil.Get(), "Port %d: admin routes mounted at %s\n", cfg.Port, adminRoute)
}

// maybeChat mounts the chat feature unless it is disabled in config.
func (c *Context) maybeChat(r *chi.Mux, cfg configutil.Server) {
	if cfg.Features.DisableChat {
		logutil.Infof(logutil.Get(), "Port %d: chat feature disabled\n", cfg.Port)
		return
	}

	chatRoute := cfg.Features.ChatRoute
	if chatRoute == "" {
		chatRoute = defaultChatRoute
	}

	resolver := func(req *http.Request) (string, string, bool) {
		return auth.UsernameFromCtx(req), auth.DisplayNameFromCtx(req), auth.IsAdminFromCtx(req)
	}

	chatHTML, err := renderChatTemplate(embedutil.MaybeRead(c.FS, chatTmplHTML), chatRoute)
	if err != nil {
		logutil.Errorf(logutil.Get(), "renderChatTemplate: %v\n", err)
		return
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
}

// seedUsersFromConfig inserts config-defined form-auth users into the database
// on first boot only. Existing rows are left untouched.
func (c *Context) seedUsersFromConfig(ctx context.Context, users []configutil.FormAuthUser) {
	for _, u := range users {
		dn := u.DisplayName
		if dn == "" {
			dn = auth.GenerateDisplayName(u.Username)
		}

		if err := c.db.InsertUserIfNotExists(
			ctx,
			u.Username,
			u.Password,
			dn,
			u.Admin,
		); err != nil {
			logutil.Errorf(logutil.Get(), "seedUsersFromConfig: insert %q: %v\n", u.Username, err)
		}
	}
}

// renderChatTemplate executes chat.html as a Go template, injecting the chat route.
// text/template is used intentionally - html/template applies context-aware JS escaping.
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

// shutdown waits for interrupt or SIGTERM, cancels in-flight requests, then
// gracefully drains all servers (up to 10 s) before closing the DB.
func (c *Context) shutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

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

	if c.db != nil {
		if err := c.db.Close(); err != nil {
			logutil.Errorf(logutil.Get(), "Error closing database: %v\n", err)
		}
	}
}
