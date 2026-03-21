package embedutil

import (
	"io/fs"

	"github.com/ricochhet/fileserver/pkg/errutil"
)

type File struct {
	Path string
	Info fs.FileInfo
}

// WalkDir walks the directory starting at the specified root path.
func WalkDir(e fs.FS, root string) ([]File, error) {
	result := []File{}

	err := fs.WalkDir(e, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return errutil.New("fs.WalkDir", err)
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return errutil.New("d.Info", err)
			}

			result = append(result, File{Path: path, Info: info})
		}

		return nil
	})
	if err != nil {
		return nil, errutil.WithFrame(err)
	}

	return result, nil
}
