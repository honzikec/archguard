package astfacts

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
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
	if isTSX(path) {
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

func findNamedChildByType(node *sitter.Node, typeName string) *sitter.Node {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == typeName {
			return child
		}
	}
	return nil
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
