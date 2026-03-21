package hostsutil

import (
	"path"
	"strings"

	"github.com/goodhosts/hostsfile"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"github.com/ricochhet/fileserver/pkg/timeutil"
)

// NewHosts creates a hostsfile handle and writes a timestamped backup.
func NewHosts() (*hostsfile.Hosts, error) {
	hf, err := hostsfile.NewHosts()
	if err != nil {
		return nil, errutil.New("hostsfile.NewHosts", err)
	}

	if err := backupHosts(hf); err != nil {
		return nil, errutil.New("backupHosts", err)
	}

	return hf, nil
}

// Add adds all entries in hosts to the hosts file and flushes.
func Add(hf *hostsfile.Hosts, hosts map[string]string) error {
	for ip, host := range hosts {
		if err := addEntry(hf, ip, host); err != nil {
			return err
		}
	}

	return hf.Flush()
}

// Remove removes all entries in hosts from the hosts file and flushes.
func Remove(hf *hostsfile.Hosts, hosts map[string]string) error {
	for ip, host := range hosts {
		if err := removeEntry(hf, ip, host); err != nil {
			return err
		}
	}

	return hf.Flush()
}

// addEntry logs and adds a single IP → hostname entry.
func addEntry(hf *hostsfile.Hosts, ip string, hosts ...string) error {
	logutil.Infof(logutil.Get(), "Adding hostsfile entry: %s %s\n", ip, strings.Join(hosts, " "))
	return hf.Add(ip, hosts...)
}

// removeEntry logs and removes a single IP → hostname entry.
func removeEntry(hf *hostsfile.Hosts, ip string, hosts ...string) error {
	logutil.Infof(logutil.Get(), "Removing hostsfile entry: %s %s\n", ip, strings.Join(hosts, " "))
	return hf.Remove(ip, hosts...)
}

// backupHosts writes a timestamped backup of the current hosts file.
func backupHosts(hf *hostsfile.Hosts) error {
	return fsutil.Write(path.Join("hosts", "hosts_"+timeutil.TimeStamp()), []byte(hf.String()))
}
