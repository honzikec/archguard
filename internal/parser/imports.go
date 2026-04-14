package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/model"
)

type importMatch struct {
	start  int
	line   int
	column int
	raw    string
	kind   string
}

func ExtractImports(path string, content []byte) []model.ImportRef {
	imports, _, _ := ExtractImportsWithDiagnostics(path, content)
	return imports
}

func ExtractImportsWithDiagnostics(path string, content []byte) ([]model.ImportRef, contracts.ParseDiagnostics, error) {
	p := sitter.NewParser()
	defer p.Close()
	if isJSXLike(path) {
		p.SetLanguage(tsx.GetLanguage())
	} else {
		p.SetLanguage(typescript.GetLanguage())
	}

	tree, err := p.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, contracts.ParseDiagnostics{}, fmt.Errorf("failed to parse js/ts file: %w", err)
	}
	if tree == nil || tree.RootNode() == nil {
		return nil, contracts.ParseDiagnostics{}, fmt.Errorf("failed to parse js/ts file: empty syntax tree")
	}
	defer tree.Close()
	if tree.RootNode().HasError() {
		return nil, contracts.ParseDiagnostics{}, fmt.Errorf("failed to parse js/ts file: syntax errors detected")
	}

	matches := make([]importMatch, 0)
	diagnostics := contracts.ParseDiagnostics{}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}

		switch node.Type() {
		case "import_statement":
			if source := node.ChildByFieldName("source"); source != nil {
				if raw, ok := stringLiteralContent(source, content); ok {
					matches = append(matches, nodeMatch(node, raw, "import"))
				}
			}
		case "export_statement":
			if source := node.ChildByFieldName("source"); source != nil {
				if raw, ok := stringLiteralContent(source, content); ok {
					matches = append(matches, nodeMatch(node, raw, "export_from"))
				}
			}
		case "call_expression":
			match, nonLiteralDynamic := callExpressionImport(node, content)
			if nonLiteralDynamic {
				diagnostics.NonLiteralDynamicImports++
			}
			if match.raw != "" {
				matches = append(matches, match)
			}
		}

		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i))
		}
	}
	walk(tree.RootNode())

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start == matches[j].start {
			return matches[i].kind < matches[j].kind
		}
		return matches[i].start < matches[j].start
	})

	seen := map[string]struct{}{}
	imports := make([]model.ImportRef, 0, len(matches))
	for _, m := range matches {
		key := fmt.Sprintf("%s|%s|%d", m.kind, m.raw, m.start)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		imports = append(imports, model.ImportRef{
			SourceFile:      path,
			RawImport:       strings.TrimSpace(m.raw),
			ResolvedPath:    "",
			IsPackageImport: tentativePackageImport(m.raw),
			Line:            m.line,
			Column:          m.column,
			Kind:            m.kind,
		})
	}

	return imports, diagnostics, nil
}

func callExpressionImport(node *sitter.Node, content []byte) (importMatch, bool) {
	fn := node.ChildByFieldName("function")
	if fn == nil {
		return importMatch{}, false
	}
	name := strings.TrimSpace(fn.Content(content))
	kind := ""
	switch name {
	case "require":
		kind = "require"
	case "import":
		kind = "dynamic_import"
	default:
		return importMatch{}, false
	}

	args := node.ChildByFieldName("arguments")
	if args == nil {
		args = firstNamedChildOfType(node, "arguments")
	}
	if args == nil {
		return importMatch{}, kind == "dynamic_import"
	}

	for i := 0; i < int(args.NamedChildCount()); i++ {
		child := args.NamedChild(i)
		if raw, ok := stringLiteralContent(child, content); ok {
			return nodeMatch(node, raw, kind), false
		}
		break
	}

	return importMatch{}, kind == "dynamic_import"
}

func nodeMatch(node *sitter.Node, raw, kind string) importMatch {
	return importMatch{
		start:  int(node.StartByte()),
		line:   int(node.StartPoint().Row) + 1,
		column: int(node.StartPoint().Column) + 1,
		raw:    raw,
		kind:   kind,
	}
}

func stringLiteralContent(node *sitter.Node, content []byte) (string, bool) {
	if node == nil {
		return "", false
	}
	switch node.Type() {
	case "string", "string_fragment":
	default:
		return "", false
	}
	raw := strings.TrimSpace(node.Content(content))
	if len(raw) < 2 {
		return "", false
	}
	quote := raw[0]
	if (quote != '"' && quote != '\'') || raw[len(raw)-1] != quote {
		return "", false
	}
	return unescapeImportLiteral(raw[1 : len(raw)-1]), true
}

func unescapeImportLiteral(raw string) string {
	raw = strings.ReplaceAll(raw, `\"`, `"`)
	raw = strings.ReplaceAll(raw, `\'`, `'`)
	raw = strings.ReplaceAll(raw, `\\`, `\`)
	return raw
}

func firstNamedChildOfType(node *sitter.Node, typeName string) *sitter.Node {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == typeName {
			return child
		}
	}
	return nil
}

func tentativePackageImport(raw string) bool {
	return !strings.HasPrefix(raw, ".") && !strings.HasPrefix(raw, "/")
}

func isJSXLike(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsx", ".jsx":
		return true
	default:
		return false
	}
}
