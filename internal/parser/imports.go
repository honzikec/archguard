package parser

import (
	"strings"

	"github.com/honzikec/archguard/internal/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractImports(path string, tree *sitter.Tree, content []byte) []model.ImportRef {
	var imports []model.ImportRef

	rootNode := tree.RootNode()
	for i := 0; i < int(rootNode.ChildCount()); i++ {
		child := rootNode.Child(i)
		if child.Type() == "import_statement" {
			sourceNode := child.ChildByFieldName("source")
			if sourceNode != nil {
				rawImport := sourceNode.Content(content)
				// Remove quotes
				rawImport = strings.Trim(rawImport, "\"'`")

				ref := model.ImportRef{
					SourceFile:      path,
					RawImport:       rawImport,
					ResolvedPath:    "", // to be filled later
					IsPackageImport: isPackageImport(rawImport),
					Line:            int(child.StartPoint().Row) + 1,
					Column:          int(child.StartPoint().Column) + 1,
				}
				imports = append(imports, ref)
			}
		}
	}

	return imports
}

func isPackageImport(raw string) bool {
	return !strings.HasPrefix(raw, ".") && !strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "@/")
}
