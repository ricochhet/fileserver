package browse

import (
	"net/http"
	"os"

	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

// handleDownload serves a file directly or streams a directory as a zip archive.
func handleDownload(w http.ResponseWriter, r *http.Request, root string, stat os.FileInfo) {
	if !stat.IsDir() {
		httputil.ContentDispositionAttachment(w, stat.Name())

		f, err := os.Open(root)
		if err != nil {
			errutil.HTTPInternalServerErrorf(w, "Could not open file: %v\n", err)
			return
		}
		defer f.Close()

		http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)

		return
	}

	name := stat.Name() + ".zip"

	httputil.ContentType(w, httputil.ContentTypeZip)
	httputil.ContentDispositionAttachment(w, name)

	if err := writeZipArchive(w, root); err != nil {
		logutil.Errorf(logutil.Get(), "handleDownload zip walk: %v\n", err)
	}
}
