package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
)

// Normalize replaces all back slashes with forward slashes.
func Normalize(path string) string {
	clean := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		clean = strings.ReplaceAll(clean, "\\", "/")
	}

	return clean
}
