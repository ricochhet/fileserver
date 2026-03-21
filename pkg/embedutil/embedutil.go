package embedutil

import (
	"embed"
	"path/filepath"
	"strings"

	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
)

type EmbeddedFileSystem struct {
	Initial string
	FS      embed.FS
}

// List return a list of files within the embedded fs, calling the list function in return.
func (e *EmbeddedFileSystem) List(pattern string, list func([]File) error) error {
	files, err := WalkDir(e.FS, pattern)
	if err != nil {
		return errutil.WithFrame(err)
	}

	return list(files)
}

// Dump return a []byte within the embedded fs, calling the dump function in return.
func (e *EmbeddedFileSystem) Dump(pattern, name string, dump func(File, []byte) error) error {
	files, err := WalkDir(e.FS, pattern)
	if err != nil {
		return errutil.New("NewFileGetter", err)
	}

	for _, file := range files {
		if !strings.Contains(file.Path, name) {
			continue
		}

		data, err := e.Read(strings.TrimPrefix(file.Path, e.Initial))
		if err != nil {
			return errutil.New("Read", err)
		}

		if err := dump(file, data); err != nil {
			return errutil.New("dump", err)
		}
	}

	return nil
}

// MaybeReadEmbedded reads the filename from a path, falling back to embed if it does not exist.
func (e *EmbeddedFileSystem) MaybeReadEmbedded(name string) ([]byte, error) {
	path := filepath.ToSlash(filepath.Join(e.Initial, name))
	if fsutil.Exists(path) {
		return fsutil.Read(path)
	}

	return e.Read(name)
}

// Read converts a file in the embedded filesystem into an array of bytes.
func (e *EmbeddedFileSystem) Read(name string) ([]byte, error) {
	b, err := e.FS.ReadFile(filepath.ToSlash(filepath.Join(e.Initial, name)))
	if err != nil {
		return nil, errutil.WithFrame(err)
	}

	return b, nil
}
