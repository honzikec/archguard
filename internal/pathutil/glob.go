package pathutil

import (
	"path/filepath"
	"strings"
)

// MatchGlob checks if a given file path matches a simplistic glob pattern like "src/domain/**"
func MatchGlob(pattern, path string) bool {
	// Simple implementation for "/**" suffix
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, prefix)
	}

	// Fallback to filepath.Match
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}
