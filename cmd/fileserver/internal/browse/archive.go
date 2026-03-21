package browse

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// writeZipArchive writes a zip of root to w.
func writeZipArchive(w io.Writer, root string) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	return filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}

		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		fh.Name = filepath.ToSlash(rel)
		fh.Method = zip.Deflate

		fw, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}

		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(fw, f)

		return err
	})
}
