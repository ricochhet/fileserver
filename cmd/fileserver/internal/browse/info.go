package browse

import (
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ricochhet/fileserver/pkg/cryptoutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

// handleInfo writes file or directory metadata as JSON.
func handleInfo(w http.ResponseWriter, _ *http.Request, path, base string, stat os.FileInfo) {
	rel, _ := filepath.Rel(base, path)
	rel = filepath.ToSlash(rel)

	res := fileInfoResponse{
		Name:        stat.Name(),
		Path:        "/" + rel,
		FullPath:    path,
		Size:        stat.Size(),
		Modified:    stat.ModTime().UTC(),
		IsDirectory: stat.IsDir(),
	}

	if !stat.IsDir() {
		ext := filepath.Ext(stat.Name())
		res.Extension = ext
		res.MimeType = mime.TypeByExtension(ext)

		if hash, err := cryptoutil.MD5(path); err != nil {
			logutil.Errorf(logutil.Get(), "handleInfo md5 %q: %v\n", path, err)
		} else {
			res.MD5 = hash
		}
	}

	httputil.ContentType(w, httputil.ContentTypeJSON)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(res); err != nil {
		logutil.Errorf(logutil.Get(), "handleInfo encode: %v\n", err)
	}
}
