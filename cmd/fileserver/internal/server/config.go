package server

import (
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/cryptoutil"
	"github.com/ricochhet/fileserver/pkg/embedutil"
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
