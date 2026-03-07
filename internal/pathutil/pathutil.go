package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// Resolve expands ~ and resolves the path to an absolute, clean path.
// It does NOT resolve symlinks intentionally so that the user sees the path
// they expect (matching what they typed).
func Resolve(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// IsSafe returns false if the path contains ".." traversal components.
func IsSafe(path string) bool {
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == ".." {
			return false
		}
	}
	return true
}

// Exists returns true if path exists and is a directory.
func Exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
