package browse

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
	"github.com/ricochhet/fileserver/pkg/logutil"
	"github.com/ricochhet/fileserver/pkg/strutil"
)

// handleListing renders a directory listing as HTML.
func handleListing(
	w http.ResponseWriter,
	_ *http.Request,
	tmpl *template.Template,
	abs, route, sub string,
	bRoute string,
	hidden []string,
	imageExtsJSON, textExtsJSON template.JS,
	readmeCandidates []string,
) {
	all, err := os.ReadDir(abs)
	if err != nil {
		errutil.HTTPInternalServerErrorf(w, "Failed to read directory: %v\n", err)
		return
	}

	filtered := all[:0]
	for _, e := range all {
		if !isHidden(e.Name(), hidden) {
			filtered = append(filtered, e)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		di, dj := filtered[i].IsDir(), filtered[j].IsDir()
		if di != dj {
			return di
		}

		return strings.ToLower(filtered[i].Name()) < strings.ToLower(filtered[j].Name())
	})

	rel := strings.Trim(sub, "/")

	var total int64

	fileCount := 0

	entries := make([]entry, 0, len(filtered))
	for _, e := range filtered {
		info, err := e.Info()
		if err != nil {
			continue
		}

		var href string
		if rel == "" {
			href = route + "/" + e.Name()
		} else {
			href = route + "/" + rel + "/" + e.Name()
		}

		sizeStr := "—"

		var sizeBytes int64
		if !e.IsDir() {
			sizeBytes = info.Size()
			sizeStr = strutil.Size(sizeBytes)
			total += sizeBytes
			fileCount++
		}

		ext := ""
		previewURL := ""

		if !e.IsDir() {
			ext = strings.ToLower(filepath.Ext(e.Name()))
			previewURL = href + "?" + previewQuery
		}

		entries = append(entries, entry{
			Name:        e.Name(),
			IsDir:       e.IsDir(),
			SizeStr:     sizeStr,
			SizeBytes:   sizeBytes,
			ModStr:      info.ModTime().Format("2006-01-02  15:04"),
			ModUnix:     info.ModTime().Unix(),
			BrowseURL:   href,
			DownloadURL: href + "?" + downloadQuery,
			InfoURL:     href + "?" + infoQuery,
			PreviewURL:  previewURL,
			Ext:         ext,
		})
	}

	raw, err := json.Marshal(entries)
	if err != nil {
		raw = []byte("[]")
	}

	parent := ""

	if rel != "" {
		up := path.Dir("/" + rel)
		if up == "/" {
			parent = route + "/"
		} else {
			parent = route + up
		}
	}

	title := "/"
	if rel != "" {
		title = rel
	}

	readme := ""

	for _, candidate := range readmeCandidates {
		if b, err := os.ReadFile(filepath.Join(abs, candidate)); err == nil {
			readme = string(b)
			break
		}
	}

	totalStr := ""
	if fileCount > 0 {
		totalStr = strutil.Size(total)
	}

	data := templateData{
		Title:         title,
		Breadcrumbs:   buildBreadcrumbs(route, rel),
		Parent:        parent,
		Entries:       entries,
		EntriesJSON:   template.JS(raw),
		IsEmpty:       len(entries) == 0,
		Readme:        readme,
		HasReadme:     readme != "",
		Route:         bRoute,
		FileCount:     fileCount,
		TotalSize:     totalStr,
		ImageExtsJSON: imageExtsJSON,
		TextExtsJSON:  textExtsJSON,
	}

	httputil.ContentType(w, httputil.ContentTypeHTML)

	if err := tmpl.Execute(w, data); err != nil {
		logutil.Errorf(logutil.Get(), "tmpl.Execute: %v\n", err)
	}
}

// buildBreadcrumbs constructs breadcrumb navigation entries for the given path.
func buildBreadcrumbs(route, p string) []breadcrumb {
	root := breadcrumb{Name: "~", Link: route + "/"}

	if p == "" {
		root.IsLast = true
		return []breadcrumb{root}
	}

	parts := strings.Split(p, "/")
	crumbs := []breadcrumb{root}

	for i, part := range parts {
		isLast := i == len(parts)-1

		link := ""
		if !isLast {
			link = route + "/" + strings.Join(parts[:i+1], "/")
		}

		crumbs = append(crumbs, breadcrumb{Name: part, Link: link, IsLast: isLast})
	}

	return crumbs
}
