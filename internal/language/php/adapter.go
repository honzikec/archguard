package php

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/language/common"
	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/model"
)

var (
	reUse            = regexp.MustCompile(`(?m)^\s*use\s+([^;]+);`)
	reRequireInclude = regexp.MustCompile(`(?m)\b(?:require|require_once|include|include_once)\s*\(?\s*["']([^"']+)["']\s*\)?`)
)

type Adapter struct{}

func New() contracts.Adapter {
	return Adapter{}
}

func (Adapter) ID() string {
	return "php"
}

func (Adapter) Detect(roots []string) contracts.Detection {
	if common.HasAnyFileNamed(roots, []string{"composer.json"}) {
		return contracts.Detection{Matched: true, Reason: "composer.json found"}
	}
	if common.HasFileWithSuffix(roots, []string{".php", ".phtml"}, 8) {
		return contracts.Detection{Matched: true, Reason: "php files detected"}
	}
	return contracts.Detection{}
}

func (Adapter) SupportsFile(path string) bool {
	return strings.HasSuffix(path, ".php") || strings.HasSuffix(path, ".phtml")
}

func (Adapter) ParseFile(path string) ([]model.ImportRef, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(content)

	refs := make([]model.ImportRef, 0)
	refs = append(refs, parseUseImports(path, content, text)...)
	refs = append(refs, parseRequireIncludes(path, content, text)...)

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Line != refs[j].Line {
			return refs[i].Line < refs[j].Line
		}
		if refs[i].Column != refs[j].Column {
			return refs[i].Column < refs[j].Column
		}
		if refs[i].Kind != refs[j].Kind {
			return refs[i].Kind < refs[j].Kind
		}
		return refs[i].RawImport < refs[j].RawImport
	})

	return refs, nil
}

func parseUseImports(path string, content []byte, text string) []model.ImportRef {
	idx := reUse.FindAllStringSubmatchIndex(text, -1)
	refs := make([]model.ImportRef, 0, len(idx))
	for _, m := range idx {
		if len(m) < 4 {
			continue
		}
		raw := strings.TrimSpace(text[m[2]:m[3]])
		line, col := lineCol(content, m[0])
		refs = append(refs, model.ImportRef{
			SourceFile:      path,
			RawImport:       raw,
			IsPackageImport: true,
			Line:            line,
			Column:          col,
			Kind:            "php_use",
		})
	}
	return refs
}

func parseRequireIncludes(path string, content []byte, text string) []model.ImportRef {
	idx := reRequireInclude.FindAllStringSubmatchIndex(text, -1)
	refs := make([]model.ImportRef, 0, len(idx))
	for _, m := range idx {
		if len(m) < 4 {
			continue
		}
		raw := strings.TrimSpace(text[m[2]:m[3]])
		line, col := lineCol(content, m[0])
		refs = append(refs, model.ImportRef{
			SourceFile:      path,
			RawImport:       raw,
			IsPackageImport: !strings.HasPrefix(raw, ".") && !strings.HasPrefix(raw, "/"),
			Line:            line,
			Column:          col,
			Kind:            "php_include",
		})
	}
	return refs
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
