package browse

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
)

// handleSearch walks the directory tree and writes JSON search results.
func handleSearch(
	w http.ResponseWriter,
	r *http.Request,
	abs, route string,
	hidden []string,
) error {
	raw := strings.TrimSpace(r.URL.Query().Get(searchQuery))
	inContent := r.URL.Query().Has(contentQuery)

	query, filter := parseSearchQuery(raw)

	var results []searchResult

	httputil.ContentType(w, httputil.ContentTypeJSON)

	if query == "" && filter == "" {
		return errutil.WithFrame(json.NewEncoder(w).Encode(results))
	}

	_ = filepath.Walk(abs, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if isHidden(info.Name(), hidden) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if filter != "" && ext != filter {
			return nil
		}

		rel, err := filepath.Rel(abs, p)
		if err != nil {
			return errutil.WithFrame(err)
		}

		rel = filepath.ToSlash(rel)
		dir := path.Dir(rel)

		var dirURL string
		if dir == "." {
			dirURL = route + "/"
		} else {
			dirURL = route + "/" + dir
		}

		if query == "" || strings.Contains(strings.ToLower(info.Name()), query) {
			results = append(results, searchResult{
				Name:         info.Name(),
				RelPath:      rel,
				HighlightURL: dirURL + "?" + highlightQuery + "=" + url.QueryEscape(info.Name()),
				DownloadURL:  route + "/" + rel + "?" + downloadQuery,
				MatchType:    nameQuery,
			})

			return nil
		}

		if inContent && query != "" {
			if snippet, ok := searchFileContent(p, info, query); ok {
				results = append(results, searchResult{
					Name:    info.Name(),
					RelPath: rel,
					HighlightURL: dirURL + "?" + highlightQuery + "=" + url.QueryEscape(
						info.Name(),
					),
					DownloadURL: route + "/" + rel + "?" + downloadQuery,
					MatchType:   contentQuery,
					Snippet:     snippet,
				})
			}
		}

		return nil
	})

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(results)
}

// searchFileContent returns a context snippet if query is found in a text file.
func searchFileContent(fp string, info os.FileInfo, query string) (string, bool) {
	if info.Size() > maxContentSearchSize {
		return "", false
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		return "", false
	}

	ct := http.DetectContentType(data)
	if !strings.HasPrefix(ct, "text/") &&
		!strings.Contains(ct, "json") &&
		!strings.Contains(ct, "xml") {
		return "", false
	}

	lower := strings.ToLower(string(data))

	idx := strings.Index(lower, query)
	if idx < 0 {
		return "", false
	}

	start := max(idx-60, 0)
	end := min(idx+len(query)+60, len(data))
	snippet := "…" + strings.TrimSpace(string(data[start:end])) + "…"

	return snippet, true
}

// parseSearchQuery splits a raw query into a base term and an optional ext: filter.
func parseSearchQuery(raw string) (query, filter string) {
	prefixes := map[string]*string{
		"extension:": &filter,
		"ext:":       &filter,
	}

	tokens := strings.Fields(raw)

	rest := tokens[:0]
	for _, t := range tokens {
		lower := strings.ToLower(t)
		matched := false

		for prefix, target := range prefixes {
			if after, ok := strings.CutPrefix(lower, prefix); ok && after != "" {
				if !strings.HasPrefix(after, ".") {
					after = "." + after
				}

				*target = after
				matched = true

				break
			}
		}

		if !matched {
			rest = append(rest, t)
		}
	}

	query = strings.ToLower(strings.Join(rest, " "))

	return query, filter
}

// extSliceToJSObject converts a slice of file extensions to a JSON object for O(1) browser lookups.
func extSliceToJSObject(exts []string) template.JS {
	m := make(map[string]int, len(exts))
	for _, e := range exts {
		m[e] = 1
	}

	b, err := json.Marshal(m)
	if err != nil {
		return template.JS("{}")
	}

	return template.JS(b)
}

// isHidden reports whether name matches any of the given glob patterns.
func isHidden(name string, patterns []string) bool {
	for _, p := range patterns {
		matched, err := filepath.Match(p, name)
		if err == nil && matched {
			return true
		}
	}

	return false
}
