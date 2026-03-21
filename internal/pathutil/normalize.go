package pathutil

import (
	"path/filepath"
	"strings"
)

// Normalize normalizes paths to be consistent across OSes (mostly relevant for Windows vs Unix)
func Normalize(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	return filepath.ToSlash(filepath.Clean(path))
}
