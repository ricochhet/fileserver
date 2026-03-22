package server

import (
	"fmt"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/cryptoutil"
	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
	"github.com/ricochhet/fileserver/pkg/jsonutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

// newDefaultConfig returns a minimal default server config using embedded assets.
func (c *Context) newDefaultConfig() *configutil.Config {
	return &configutil.Config{
		Servers: []configutil.Server{
			{
				Port: 8080,
				ContentEntries: []configutil.ContentEntry{
					{
						Route:  "/",
						Name:   "index.html",
						Base64: cryptoutil.EncodeB64(embedutil.MaybeRead(c.FS, "index.html")),
					},
					{
						Route:  "/404.html",
						Name:   "404.html",
						Base64: cryptoutil.EncodeB64(embedutil.MaybeRead(c.FS, "404.html")),
					},
					{
						Route:  "/base.css",
						Name:   "base.css",
						Base64: cryptoutil.EncodeB64(embedutil.MaybeRead(c.FS, "base.css")),
					},
				},
			},
		},
	}
}

// maybeReadConfig reads the config file if it exists, falling back to a default config.
func (c *Context) maybeReadConfig(p string) (*configutil.Config, error) {
	exists := fsutil.Exists(p)

	switch {
	case exists:
		config, err := jsonutil.ReadAndUnmarshal[configutil.Config](p)
		if err != nil {
			logutil.Errorf(logutil.Get(), "Error reading server config: %v\n", err)
		}

		return config, err

	case !exists && p != "":
		return nil, fmt.Errorf("path specified but does not exist: %s", p)

	default:
		logutil.Infof(logutil.Get(), "Starting with default server config\n")
		return c.newDefaultConfig(), nil
	}
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
