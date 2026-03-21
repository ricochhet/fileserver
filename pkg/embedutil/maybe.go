package embedutil

import (
	"os"
	"strings"

	"github.com/ricochhet/fileserver/pkg/cryptoutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

// MaybeBase64 checks if name contains a prefix associated with embedded files. If it does, return the embedded file as base64.
func MaybeBase64(fs *EmbeddedFileSystem, name string) ([]byte, error) {
	if after, ok := strings.CutPrefix(name, "asset:"); ok {
		return MaybeRead(fs, after), nil
	}

	b, err := cryptoutil.DecodeB64(name)
	if err != nil {
		return nil, errutil.WithFrame(err)
	}

	return b, nil
}

// MaybeRead reads the specified name from the embedded filesystem. If it cannot be read, the program will exit.
func MaybeRead(fs *EmbeddedFileSystem, name string) []byte {
	b, err := fs.Read(name)
	if err != nil {
		logutil.Errorf(logutil.Get(), "Error reading from embedded filesystem: %v\n", err)
		os.Exit(1)
	}

	return b
}
