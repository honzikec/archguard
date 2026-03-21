package pathutil

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func MatchGlob(pattern, input string) bool {
	pattern = Normalize(strings.TrimSpace(pattern))
	input = Normalize(input)
	if pattern == "" {
		return false
	}

	matched, err := doublestar.PathMatch(pattern, input)
	if err != nil {
		return false
	}
	return matched
}

func MatchAny(patterns []string, input string) bool {
	for _, p := range patterns {
		if MatchGlob(p, input) {
			return true
		}
	}
	return false
}
