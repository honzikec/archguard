package parser

import (
	"os"

	"github.com/honzikec/archguard/internal/model"
)

func ParseFile(path string) ([]model.ImportRef, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractImports(path, content), nil
}
