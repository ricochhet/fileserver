package fsutil

import (
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ricochhet/fileserver/pkg/errutil"
)

// Read reads a file from the specified path.
func Read(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errutil.WithFrame(err)
	}

	return data, nil
}

// Write writes to the specified path with the provided data.
func Write(path string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return errutil.New("os.MkdirAll", err)
	}

	err = os.WriteFile(path, data, 0o644)
	if err != nil {
		return errutil.New("os.WriteFile", err)
	}

	return nil
}

// Exists returns true if a file exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Ensure ensures the file path, returning an error if it fails.
func Ensure(path string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		return os.MkdirAll(dir, 0o755)
	}

	return nil
}

// Validate checks if a file exists and matches the given hash.
func Validate(path, hash string, h hash.Hash) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		return false
	}

	sum := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))

	return sum == strings.ToUpper(hash)
}

// JoinEnviron combines the given envs with the env by name.
func JoinEnviron(name string, envs []string) string {
	s := string(filepath.ListSeparator)
	e := os.Getenv(name)
	n := strings.Join(envs, s)

	if e != "" {
		return name + "=" + n + s + e
	}

	return name + "=" + n
}

// SafeJoin ensures the joined path does not escape base.
func SafeJoin(base, rel string) (string, error) {
	joined := filepath.Join(base, rel)

	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}

	if abs != base && !strings.HasPrefix(abs, base+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes base %q", abs, base)
	}

	return abs, nil
}
