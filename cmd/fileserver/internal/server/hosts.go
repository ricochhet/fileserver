package server

import (
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/hostsutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
)

// addHosts adds config-defined host entries to the system hosts file.
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

// removeHosts removes config-defined host entries from the system hosts file.
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
