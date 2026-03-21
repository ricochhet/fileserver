package strutil

import (
	"fmt"
	"strings"
)

// ToSlice converts a string to a slice based on the separator provided.
func ToSlice(s, sep string) []string {
	return strings.Split(s, sep)
}

// Empty returns s if it's not empty, otherwise returns "<empty>".
func Empty(s string) string {
	if s != "" {
		return s
	}

	return "<empty>"
}

// Size returns the bytes as a size string.
func Size(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/(1024*1024*1024))
	}
}
