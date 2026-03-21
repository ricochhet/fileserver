package browse

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/fsutil"
)

const (
	maxContentSearchSize = 10 << 20 // 10 MB
	searchQuery          = "search"
	previewQuery         = "preview"
	downloadQuery        = "download"
	infoQuery            = "info"
	highlightQuery       = "highlight"

	nameQuery    = "name"
	contentQuery = "content"

	browseTmpl     = "browse"
	browseTmplHTML = "browse.html"
)

var defaultImageExts = []string{
	".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".ico",
}

var defaultTextExts = []string{
	".txt", ".md", ".json", ".yaml", ".yml", ".toml", ".xml",
	".html", ".htm", ".js", ".ts", ".css", ".go", ".py", ".rs",
	".java", ".c", ".cpp", ".h", ".hpp", ".sh", ".bash", ".zsh",
	".env", ".log", ".csv", ".ini", ".conf", ".cfg", ".rb", ".php",
	".sql", ".tf", ".hcl", ".lua", ".vim", ".diff", ".patch",
}

var defaultReadmeCandidates = []string{"README.md", "INDEX.md", "index.md"}

type fileInfoResponse struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	FullPath    string    `json:"fullPath"`
	Extension   string    `json:"extension,omitempty"`
	MimeType    string    `json:"mimeType,omitempty"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	IsDirectory bool      `json:"isDirectory"`
	MD5         string    `json:"md5,omitempty"`
}

type searchResult struct {
	Name         string `json:"name"`
	RelPath      string `json:"relPath"`
	HighlightURL string `json:"highlightUrl"`
	DownloadURL  string `json:"downloadUrl"`
	MatchType    string `json:"matchType"`
	Snippet      string `json:"snippet,omitempty"`
}

type breadcrumb struct {
	Name   string
	Link   string
	IsLast bool
}

type entry struct {
	Name        string `json:"name"`
	IsDir       bool   `json:"isDir"`
	SizeStr     string `json:"sizeStr"`
	SizeBytes   int64  `json:"sizeBytes"`
	ModStr      string `json:"modStr"`
	ModUnix     int64  `json:"modUnix"`
	BrowseURL   string `json:"browseUrl"`
	DownloadURL string `json:"downloadUrl"`
	InfoURL     string `json:"infoUrl"`
	PreviewURL  string `json:"previewUrl"`
	Ext         string `json:"ext"`
}

type templateData struct {
	Title       string
	Breadcrumbs []breadcrumb
	Parent      string
	Entries     []entry
	EntriesJSON template.JS
	IsEmpty     bool
	Readme      string
	HasReadme   bool
	Route       string
	FileCount   int
	TotalSize   string

	ImageExtsJSON template.JS
	TextExtsJSON  template.JS
}

// Handler returns an http.Handler for browsing a directory rooted at path.
func Handler(
	fs *embedutil.EmbeddedFileSystem,
	path, route string,
	hidden []string,
	cfg *configutil.Server,
) http.Handler {
	route = strings.TrimSuffix(route, "/")

	imageExts := maybeSlice(cfg.ImageExts, defaultImageExts)
	textExts := maybeSlice(cfg.TextExts, defaultTextExts)
	readmeCandidates := maybeSlice(cfg.ReadmeCandidates, defaultReadmeCandidates)

	imageExtsJSON := extSliceToJSObject(imageExts)
	textExtsJSON := extSliceToJSObject(textExts)

	base, err := filepath.Abs(path)
	if err != nil {
		panic(fmt.Sprintf("BrowseHandler: cannot resolve basePath %q: %v", path, err))
	}

	b := embedutil.MaybeRead(fs, browseTmplHTML)
	tmpl := template.Must(template.New(browseTmpl).Parse(string(b)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has(searchQuery) {
			if err := handleSearch(w, r, base, route, hidden); err != nil {
				errutil.HTTPInternalServerError(w)
			}

			return
		}

		sub := filepath.FromSlash(chi.URLParam(r, "*"))

		abs, err := fsutil.SafeJoin(base, sub)
		if err != nil {
			errutil.HTTPForbidden(w)
			return
		}

		stat, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
			} else {
				errutil.HTTPInternalServerError(w)
			}

			return
		}

		switch {
		case r.URL.Query().Has(previewQuery):
			handlePreview(w, r, abs, stat)
		case r.URL.Query().Has(downloadQuery):
			handleDownload(w, r, abs, stat)
		case r.URL.Query().Has(infoQuery):
			handleInfo(w, r, abs, base, stat)
		case stat.IsDir():
			handleListing(
				w, r, tmpl,
				abs, route, filepath.ToSlash(sub), route,
				hidden,
				imageExtsJSON, textExtsJSON,
				readmeCandidates,
			)
		default:
			http.ServeFile(w, r, abs)
		}
	})
}

// maybeSlice returns s if non-empty, otherwise returns fallback.
func maybeSlice(s, fallback []string) []string {
	if len(s) == 0 {
		return fallback
	}

	return s
}
