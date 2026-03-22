package admin

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

const maxUploadSize = 512 << 20 // 512 MB per request.

type UploadedFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

// uploadHandler accepts multipart/form-data file uploads and saves them under path.
func uploadHandler(path string) http.HandlerFunc {
	if path == "" {
		exe, err := os.Executable()
		if err != nil {
			exe = "."
		}

		path = filepath.Join(filepath.Dir(exe), "uploads")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			errutil.HTTPBadRequestf(w, "could not parse multipart form")
			return
		}

		safe := strings.TrimSpace(r.FormValue("path"))

		dest, err := fsutil.SafeJoin(path, safe)
		if err != nil {
			errutil.HTTPBadRequestf(w, "invalid upload path")
			return
		}

		if err := os.MkdirAll(dest, 0o755); err != nil {
			logutil.Errorf(logutil.Get(), "admin: upload: MkdirAll %q: %v\n", dest, err)
			errutil.HTTPInternalServerErrorf(w, "could not create upload directory")

			return
		}

		files := r.MultipartForm.File["file"]
		if len(files) == 0 {
			errutil.HTTPBadRequestf(w, `multipart field "file" is required`)
			return
		}

		saved := make([]UploadedFile, 0, len(files))

		for _, fh := range files {
			uf, err := saveUploadedFile(fh, dest)
			if err != nil {
				logutil.Errorf(logutil.Get(), "admin: upload: save %q: %v\n", fh.Filename, err)
				errutil.HTTPInternalServerErrorf(w, "%s", "could not save file: "+fh.Filename)

				return
			}

			logutil.Infof(logutil.Get(), "admin: uploaded %q -> %q\n", fh.Filename, uf.Path)
			saved = append(saved, uf)
		}

		serverutil.WriteJSON(w, http.StatusCreated, saved)
	}
}

// saveUploadedFile writes a single multipart file header to dest and returns metadata.
func saveUploadedFile(fh *multipart.FileHeader, dest string) (UploadedFile, error,
) {
	src, err := fh.Open()
	if err != nil {
		return UploadedFile{}, err
	}
	defer src.Close()

	safe := filepath.Base(fh.Filename)
	if safe == "." || safe == "" {
		safe = "upload"
	}

	safeDest := filepath.Join(dest, safe)

	dst, err := os.Create(safeDest)
	if err != nil {
		return UploadedFile{}, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return UploadedFile{}, err
	}

	return UploadedFile{Name: safe, Size: n, Path: safeDest}, nil
}
