package pathutil

import (
	"path"
	"regexp"
	"strings"
)

func MatchGlob(pattern, input string) bool {
	pattern = Normalize(pattern)
	input = Normalize(input)

	re, err := regexp.Compile(globToRegex(pattern))
	if err != nil {
		return false
	}
	return re.MatchString(input)
}

func MatchAny(patterns []string, input string) bool {
	for _, p := range patterns {
		if MatchGlob(p, input) {
			return true
		}
	}
	return false
}

func globToRegex(pattern string) string {
	pattern = path.Clean(pattern)
	if pattern == "." {
		pattern = "**"
	}

	var b strings.Builder
	b.WriteString("^")
	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		switch ch {
		case '*':
			if i+1 < len(runes) && runes[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '(', ')', '+', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteString("\\")
			b.WriteRune(ch)
		default:
			b.WriteRune(ch)
		}
	}
	b.WriteString("$")
	return b.String()
}
