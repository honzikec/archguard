package parser

import (
	"context"
	"os"

	"github.com/honzikec/archguard/internal/model"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

func ParseFile(path string) ([]model.ImportRef, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	if isTSX(path) {
		parser.SetLanguage(tsx.GetLanguage())
	} else {
		parser.SetLanguage(typescript.GetLanguage())
	}

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}

	return ExtractImports(path, tree, content), nil
}

func isTSX(path string) bool {
	return len(path) > 4 && (path[len(path)-4:] == ".tsx" || path[len(path)-4:] == ".jsx")
}
