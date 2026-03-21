package browse

import (
	"net/http"
	"os"

	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
)

// handlePreview serves a file inline for browser preview.
func handlePreview(w http.ResponseWriter, r *http.Request, abs string, stat os.FileInfo) {
	if stat.IsDir() {
		errutil.HTTPBadRequestf(w, "Cannot preview a directory")
		return
	}

	httputil.ContentDispositionInline(w, stat.Name())
	http.ServeFile(w, r, abs)
}
