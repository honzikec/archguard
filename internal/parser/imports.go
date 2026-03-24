package parser

import (
	"regexp"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/model"
)

var (
	reImportStmt = regexp.MustCompile(`(?m)^\s*import(?:\s+[^'\n;]*?\s+from)?\s*['\"]([^'\"]+)['\"]`)
	reExportFrom = regexp.MustCompile(`(?m)^\s*export(?:\s+[^'\n;]*?\s+from)\s*['\"]([^'\"]+)['\"]`)
)

type importMatch struct {
	start int
	end   int
	raw   string
	kind  string
}

func ExtractImports(path string, content []byte) []model.ImportRef {
	text := string(content)
	masked := maskComments(text)
	matches := make([]importMatch, 0)
	matches = append(matches, collectMatches(masked, reImportStmt, "import")...)
	matches = append(matches, collectMatches(masked, reExportFrom, "export_from")...)
	matches = append(matches, collectCallMatches(masked, "require", "require")...)
	matches = append(matches, collectCallMatches(masked, "import", "dynamic_import")...)

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start == matches[j].start {
			return matches[i].kind < matches[j].kind
		}
		return matches[i].start < matches[j].start
	})

	seen := map[string]struct{}{}
	imports := make([]model.ImportRef, 0, len(matches))
	for _, m := range matches {
		key := m.kind + "|" + m.raw + "|" + itoa(m.start)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		line, col := lineCol(content, m.start)
		imports = append(imports, model.ImportRef{
			SourceFile:      path,
			RawImport:       strings.TrimSpace(m.raw),
			ResolvedPath:    "",
			IsPackageImport: tentativePackageImport(m.raw),
			Line:            line,
			Column:          col,
			Kind:            m.kind,
		})
	}

	return imports
}

func collectCallMatches(text, keyword, kind string) []importMatch {
	out := make([]importMatch, 0)
	if keyword == "" {
		return out
	}
	inSingle := false
	inDouble := false
	inTemplate := false
	escaped := false

	for i := 0; i < len(text); i++ {
		c := text[i]
		if inSingle {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if inTemplate {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '`' {
				inTemplate = false
			}
			continue
		}

		switch c {
		case '\'':
			inSingle = true
			continue
		case '"':
			inDouble = true
			continue
		case '`':
			inTemplate = true
			continue
		}

		if !hasTokenAt(text, i, keyword) {
			continue
		}
		tokenEnd := i + len(keyword)
		if !isCallTokenBoundary(text, i, tokenEnd) {
			continue
		}

		j := skipWhitespace(text, tokenEnd)
		if j >= len(text) || text[j] != '(' {
			continue
		}
		j = skipWhitespace(text, j+1)
		if j >= len(text) || (text[j] != '"' && text[j] != '\'') {
			continue
		}

		quote := text[j]
		rawStart := j + 1
		j++
		escaped = false
		for ; j < len(text); j++ {
			if escaped {
				escaped = false
				continue
			}
			if text[j] == '\\' {
				escaped = true
				continue
			}
			if text[j] == quote {
				break
			}
		}
		if j >= len(text) {
			continue
		}
		rawEnd := j
		j = skipWhitespace(text, j+1)
		if j >= len(text) || text[j] != ')' {
			continue
		}

		out = append(out, importMatch{
			start: i,
			end:   j + 1,
			raw:   text[rawStart:rawEnd],
			kind:  kind,
		})
		i = j
	}
	return out
}

func collectMatches(text string, re *regexp.Regexp, kind string) []importMatch {
	idx := re.FindAllStringSubmatchIndex(text, -1)
	matches := make([]importMatch, 0, len(idx))
	for _, m := range idx {
		if len(m) < 4 {
			continue
		}
		fullStart := m[0]
		rawStart := m[2]
		rawEnd := m[3]
		if rawStart < 0 || rawEnd <= rawStart {
			continue
		}
		matches = append(matches, importMatch{
			start: fullStart,
			end:   m[1],
			raw:   text[rawStart:rawEnd],
			kind:  kind,
		})
	}
	return matches
}

func maskComments(in string) string {
	out := make([]byte, 0, len(in))
	inString := false
	stringQuote := byte(0)
	escaped := false
	lineComment := false
	blockComment := false

	for i := 0; i < len(in); i++ {
		c := in[i]

		if lineComment {
			if c == '\n' {
				lineComment = false
				out = append(out, c)
			} else {
				out = append(out, ' ')
			}
			continue
		}
		if blockComment {
			if c == '\n' {
				out = append(out, c)
			} else {
				out = append(out, ' ')
			}
			if c == '*' && i+1 < len(in) && in[i+1] == '/' {
				out = append(out, ' ')
				blockComment = false
				i++
			}
			continue
		}

		if inString {
			out = append(out, c)
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == stringQuote {
				inString = false
				stringQuote = byte(0)
			}
			continue
		}

		if c == '\'' || c == '"' || c == '`' {
			inString = true
			stringQuote = c
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(in) {
			next := in[i+1]
			if next == '/' {
				lineComment = true
				out = append(out, ' ', ' ')
				i++
				continue
			}
			if next == '*' {
				blockComment = true
				out = append(out, ' ', ' ')
				i++
				continue
			}
		}
		out = append(out, c)
	}
	return string(out)
}

func hasTokenAt(text string, start int, token string) bool {
	if start < 0 || start+len(token) > len(text) {
		return false
	}
	return text[start:start+len(token)] == token
}

func isCallTokenBoundary(text string, start, end int) bool {
	if start > 0 {
		before := text[start-1]
		if isIdentChar(before) || before == '.' {
			return false
		}
	}
	if end < len(text) && isIdentChar(text[end]) {
		return false
	}
	return true
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '$'
}

func skipWhitespace(text string, start int) int {
	i := start
	for i < len(text) {
		switch text[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

func lineCol(content []byte, index int) (int, int) {
	if index < 0 {
		return 1, 1
	}
	line := 1
	col := 1
	for i := 0; i < len(content) && i < index; i++ {
		if content[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func tentativePackageImport(raw string) bool {
	return !strings.HasPrefix(raw, ".") && !strings.HasPrefix(raw, "/")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	buf := [20]byte{}
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + (i % 10))
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
