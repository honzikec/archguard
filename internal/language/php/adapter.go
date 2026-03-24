package php

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/php"

	"github.com/honzikec/archguard/internal/language/common"
	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/model"
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

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(php.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse php file: %w", err)
	}
	if tree == nil || tree.RootNode() == nil {
		return nil, fmt.Errorf("failed to parse php file: empty syntax tree")
	}
	defer tree.Close()
	if tree.RootNode().HasError() {
		return nil, fmt.Errorf("failed to parse php file: syntax errors detected")
	}

	refs := make([]model.ImportRef, 0)
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}

		switch node.Type() {
		case "namespace_use_declaration":
			refs = append(refs, parseNamespaceUseDeclaration(path, node, content)...)
		case "include_expression", "include_once_expression", "require_expression", "require_once_expression":
			if ref, ok := parseIncludeLikeExpression(path, node, content); ok {
				refs = append(refs, ref)
			}
		}

		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i))
		}
	}

	walk(tree.RootNode())

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

func parseNamespaceUseDeclaration(path string, node *sitter.Node, content []byte) []model.ImportRef {
	prefix := ""
	if groupPrefix := findNamedChildByType(node, "namespace_name"); groupPrefix != nil {
		prefix = normalizePHPNamespace(groupPrefix.Content(content))
	}

	clauses := findNamedChildrenByType(node, "namespace_use_clause")
	groupClauses := make([]*sitter.Node, 0)
	for _, group := range findNamedChildrenByType(node, "namespace_use_group") {
		groupClauses = append(groupClauses, findNamedChildrenByType(group, "namespace_use_group_clause")...)
	}

	out := make([]model.ImportRef, 0, len(clauses)+len(groupClauses))
	for _, clause := range clauses {
		raw := extractUsePath(clause, content)
		raw = normalizePHPNamespace(raw)
		if raw == "" {
			continue
		}
		out = append(out, model.ImportRef{
			SourceFile:      path,
			RawImport:       raw,
			IsPackageImport: true,
			Line:            int(clause.StartPoint().Row) + 1,
			Column:          int(clause.StartPoint().Column) + 1,
			Kind:            "php_use",
		})
	}

	for _, clause := range groupClauses {
		item := extractUsePath(clause, content)
		item = normalizePHPNamespace(item)
		if item == "" {
			continue
		}
		raw := item
		if prefix != "" {
			raw = prefix + `\` + item
		}
		out = append(out, model.ImportRef{
			SourceFile:      path,
			RawImport:       raw,
			IsPackageImport: true,
			Line:            int(clause.StartPoint().Row) + 1,
			Column:          int(clause.StartPoint().Column) + 1,
			Kind:            "php_use",
		})
	}

	return out
}

func parseIncludeLikeExpression(path string, node *sitter.Node, content []byte) (model.ImportRef, bool) {
	raw := ""
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil || raw != "" {
			return
		}
		switch n.Type() {
		case "string", "encapsed_string":
			raw = trimStringLiteral(n.Content(content))
			return
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(node)

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return model.ImportRef{}, false
	}
	return model.ImportRef{
		SourceFile:      path,
		RawImport:       raw,
		IsPackageImport: !strings.HasPrefix(raw, ".") && !strings.HasPrefix(raw, "/"),
		Line:            int(node.StartPoint().Row) + 1,
		Column:          int(node.StartPoint().Column) + 1,
		Kind:            "php_include",
	}, true
}

func extractUsePath(node *sitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		switch child.Type() {
		case "qualified_name", "namespace_name", "name":
			return child.Content(content)
		}
	}
	return ""
}

func findNamedChildByType(node *sitter.Node, typeName string) *sitter.Node {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == typeName {
			return child
		}
	}
	return nil
}

func findNamedChildrenByType(node *sitter.Node, typeName string) []*sitter.Node {
	out := make([]*sitter.Node, 0)
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == typeName {
			out = append(out, child)
		}
	}
	return out
}

func trimStringLiteral(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "\"")
	v = strings.TrimSuffix(v, "\"")
	v = strings.TrimPrefix(v, "'")
	v = strings.TrimSuffix(v, "'")
	return v
}

func normalizePHPNamespace(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, `\`)
	v = strings.TrimSuffix(v, `\`)
	return strings.TrimSpace(v)
}
