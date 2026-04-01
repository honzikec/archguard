package astfacts

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

func ParseFile(path string) (FileFacts, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return FileFacts{}, err
	}
	return ParseContent(path, content), nil
}

func ParseContent(path string, content []byte) FileFacts {
	facts := FileFacts{
		FilePath:            filepath.ToSlash(filepath.Clean(path)),
		ExportedClassByName: map[string]string{},
	}

	parser := sitter.NewParser()
	defer parser.Close()
	if isPHP(path) {
		parser.SetLanguage(php.GetLanguage())
	} else if isTSX(path) {
		parser.SetLanguage(tsx.GetLanguage())
	} else {
		parser.SetLanguage(typescript.GetLanguage())
	}

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil || tree == nil || tree.RootNode() == nil {
		return facts
	}
	defer tree.Close()

	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}

		switch node.Type() {
		case "class_declaration":
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				facts.Classes = append(facts.Classes, ClassDecl{
					Name: nameNode.Content(content),
					Line: int(nameNode.StartPoint().Row) + 1,
				})
			}
		case "namespace_definition":
			if nsNode := findNamedChildByType(node, "namespace_name"); nsNode != nil {
				facts.Namespace = normalizePHPNamespace(nsNode.Content(content))
			}
		case "namespace_use_declaration":
			facts.Imports = append(facts.Imports, parsePHPNamespaceUseDeclaration(node, content)...)
		case "export_statement":
			parseExportStatement(node, content, &facts)
		case "import_statement":
			if binding, ok := parseImportStatement(node, content); ok {
				facts.Imports = append(facts.Imports, binding)
			}
		case "new_expression":
			constructor := node.ChildByFieldName("constructor")
			if constructor == nil && node.NamedChildCount() > 0 {
				constructor = node.NamedChild(0)
			}
			className := ""
			isIdentifier := false
			kind := ""
			if constructor != nil {
				kind = constructor.Type()
				if constructor.Type() == "identifier" {
					className = constructor.Content(content)
					isIdentifier = true
				}
			}
			facts.NewExprs = append(facts.NewExprs, NewExpression{
				ClassName:       className,
				Line:            int(node.StartPoint().Row) + 1,
				Column:          int(node.StartPoint().Column) + 1,
				Raw:             node.Content(content),
				IsIdentifier:    isIdentifier,
				ConstructorKind: kind,
			})
		case "object_creation_expression":
			constructor := node.NamedChild(0)
			className := ""
			isIdentifier := false
			kind := ""
			if constructor != nil {
				kind = constructor.Type()
				switch constructor.Type() {
				case "name":
					className = constructor.Content(content)
					isIdentifier = true
				case "qualified_name":
					className = normalizePHPNamespace(constructor.Content(content))
					isIdentifier = className != ""
				}
			}
			facts.NewExprs = append(facts.NewExprs, NewExpression{
				ClassName:       className,
				Line:            int(node.StartPoint().Row) + 1,
				Column:          int(node.StartPoint().Column) + 1,
				Raw:             node.Content(content),
				IsIdentifier:    isIdentifier,
				ConstructorKind: kind,
			})
		}

		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i))
		}
	}

	walk(tree.RootNode())
	return facts
}

func parseExportStatement(node *sitter.Node, content []byte, facts *FileFacts) {
	if node == nil {
		return
	}
	isDefault := hasToken(node, "default")

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		switch child.Type() {
		case "class_declaration":
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			className := nameNode.Content(content)
			if isDefault {
				facts.DefaultExportedClass = className
			} else {
				facts.ExportedClassByName[className] = className
			}
		case "identifier":
			if isDefault {
				facts.DefaultExportedClass = child.Content(content)
			}
		case "export_clause":
			for j := 0; j < int(child.NamedChildCount()); j++ {
				spec := child.NamedChild(j)
				if spec.Type() != "export_specifier" {
					continue
				}
				nameNode := spec.ChildByFieldName("name")
				aliasNode := spec.ChildByFieldName("alias")
				if nameNode == nil {
					continue
				}
				local := nameNode.Content(content)
				if aliasNode == nil {
					facts.ExportedClassByName[local] = local
					continue
				}
				alias := aliasNode.Content(content)
				if alias == "default" {
					facts.DefaultExportedClass = local
					continue
				}
				facts.ExportedClassByName[alias] = local
			}
		}
	}
}

func parseImportStatement(node *sitter.Node, content []byte) (ImportBinding, bool) {
	source := node.ChildByFieldName("source")
	if source == nil {
		return ImportBinding{}, false
	}
	module := trimQuotes(source.Content(content))
	binding := ImportBinding{
		Module: module,
		Named:  map[string]string{},
		Line:   int(node.StartPoint().Row) + 1,
	}

	importClause := findNamedChildByType(node, "import_clause")
	if importClause == nil {
		return binding, true
	}

	for i := 0; i < int(importClause.NamedChildCount()); i++ {
		child := importClause.NamedChild(i)
		switch child.Type() {
		case "identifier":
			if binding.Default == "" {
				binding.Default = child.Content(content)
			}
		case "namespace_import":
			id := findNamedChildByType(child, "identifier")
			if id != nil {
				binding.Namespace = id.Content(content)
			}
		case "named_imports":
			for j := 0; j < int(child.NamedChildCount()); j++ {
				spec := child.NamedChild(j)
				if spec.Type() != "import_specifier" {
					continue
				}
				name := spec.ChildByFieldName("name")
				alias := spec.ChildByFieldName("alias")
				if name == nil {
					continue
				}
				imported := name.Content(content)
				local := imported
				if alias != nil {
					local = alias.Content(content)
				}
				binding.Named[local] = imported
			}
		}
	}

	return binding, true
}

func parsePHPNamespaceUseDeclaration(node *sitter.Node, content []byte) []ImportBinding {
	if node == nil {
		return nil
	}

	out := make([]ImportBinding, 0)
	for _, clause := range findNamedChildrenByType(node, "namespace_use_clause") {
		if b, ok := parsePHPUseClause(clause, "", content); ok {
			out = append(out, b)
		}
	}

	prefix := ""
	if groupPrefix := findNamedChildByType(node, "namespace_name"); groupPrefix != nil {
		prefix = normalizePHPNamespace(groupPrefix.Content(content))
	}
	for _, group := range findNamedChildrenByType(node, "namespace_use_group") {
		for _, clause := range findNamedChildrenByType(group, "namespace_use_group_clause") {
			if b, ok := parsePHPUseClause(clause, prefix, content); ok {
				out = append(out, b)
			}
		}
	}
	return out
}

func parsePHPUseClause(clause *sitter.Node, prefix string, content []byte) (ImportBinding, bool) {
	rawPath := extractPHPUsePath(clause, content)
	rawPath = normalizePHPNamespace(rawPath)
	if rawPath == "" {
		return ImportBinding{}, false
	}
	if prefix != "" {
		rawPath = normalizePHPNamespace(prefix + `\` + rawPath)
	}

	importedName := phpNamespaceLeaf(rawPath)
	localName := extractPHPUseAlias(clause, content)
	if localName == "" {
		localName = importedName
	}
	localName = strings.TrimSpace(localName)
	if localName == "" || importedName == "" {
		return ImportBinding{}, false
	}

	return ImportBinding{
		Module: rawPath,
		Named: map[string]string{
			localName: importedName,
		},
		Line: int(clause.StartPoint().Row) + 1,
	}, true
}

func extractPHPUsePath(node *sitter.Node, content []byte) string {
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

func extractPHPUseAlias(node *sitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() != "namespace_aliasing_clause" {
			continue
		}
		if alias := findNamedChildByType(child, "name"); alias != nil {
			return strings.TrimSpace(alias.Content(content))
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

func hasToken(node *sitter.Node, tokenType string) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == tokenType {
			return true
		}
	}
	return false
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")
	s = strings.TrimPrefix(s, "'")
	s = strings.TrimSuffix(s, "'")
	s = strings.TrimPrefix(s, "`")
	s = strings.TrimSuffix(s, "`")
	return s
}

func isTSX(path string) bool {
	path = strings.ToLower(path)
	return strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".jsx")
}

func isPHP(path string) bool {
	path = strings.ToLower(path)
	return strings.HasSuffix(path, ".php") || strings.HasSuffix(path, ".phtml")
}

func normalizePHPNamespace(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, `\`)
	v = strings.TrimSuffix(v, `\`)
	return strings.TrimSpace(v)
}

func phpNamespaceLeaf(v string) string {
	v = normalizePHPNamespace(v)
	if v == "" {
		return ""
	}
	parts := strings.Split(v, `\`)
	return strings.TrimSpace(parts[len(parts)-1])
}
