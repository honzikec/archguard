package parser

import (
	"regexp"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/model"
)

var (
	reImportStmt    = regexp.MustCompile(`(?m)^\s*import(?:\s+[^'\n;]*?\s+from)?\s*['\"]([^'\"]+)['\"]`)
	reExportFrom    = regexp.MustCompile(`(?m)^\s*export(?:\s+[^'\n;]*?\s+from)\s*['\"]([^'\"]+)['\"]`)
	reRequireCall   = regexp.MustCompile(`require\(\s*['\"]([^'\"]+)['\"]\s*\)`)
	reDynamicImport = regexp.MustCompile(`import\(\s*['\"]([^'\"]+)['\"]\s*\)`)
)

type importMatch struct {
	start int
	end   int
	raw   string
	kind  string
}

func ExtractImports(path string, content []byte) []model.ImportRef {
	text := string(content)
	matches := make([]importMatch, 0)
	matches = append(matches, collectMatches(text, reImportStmt, "import")...)
	matches = append(matches, collectMatches(text, reExportFrom, "export_from")...)
	matches = append(matches, collectMatches(text, reRequireCall, "require")...)
	matches = append(matches, collectMatches(text, reDynamicImport, "dynamic_import")...)

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
