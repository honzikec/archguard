package pathutil

import (
	"path/filepath"
)

// Normalize normalizes paths to be consistent across OSes (mostly relevant for Windows vs Unix)
func Normalize(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}
