package serverutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ricochhet/fileserver/pkg/logutil"
)

// ListenAndServe starts the HTTP server in a background goroutine and returns it.
func (h *Context) ListenAndServe(
	baseCtx context.Context,
	addr string,
	wg *sync.WaitGroup,
) *http.Server {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h.Get().Router,
		ReadHeaderTimeout: time.Duration(h.Get().Timeouts.ReadHeader) * time.Second,
		ReadTimeout:       time.Duration(h.Get().Timeouts.Read) * time.Second,
		WriteTimeout:      time.Duration(h.Get().Timeouts.Write) * time.Second,
		IdleTimeout:       time.Duration(h.Get().Timeouts.Idle) * time.Second,
	}

	if baseCtx != nil {
		srv.BaseContext = func(net.Listener) context.Context { return baseCtx }
	}

	if wg != nil {
		srv.ConnState = func(_ net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				wg.Add(1)
			case http.StateClosed, http.StateHijacked:
				wg.Done()
			case http.StateActive, http.StateIdle:
			}
		}
	}

	logutil.Infof(logutil.Get(), "Server listening on %s\n", addr)

	go func() {
		var err error

		if h.Get().TLS.Enabled {
			fmt.Fprintf(os.Stdout, "Server starting with tls: %s (cert) and %s (key)\n",
				h.Get().TLS.CertFile, h.Get().TLS.KeyFile)
			err = srv.ListenAndServeTLS(h.Get().TLS.CertFile, h.Get().TLS.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logutil.Infof(
				logutil.Get(),
				"Server %s failed: %v\n",
				strings.TrimPrefix(addr, ":"),
				err,
			)
		}
	}()

	return srv
}
