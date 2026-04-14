package parser

import (
	"os"

	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/model"
)

func ParseFile(path string) ([]model.ImportRef, error) {
	imports, _, err := ParseFileWithDiagnostics(path)
	return imports, err
}

func ParseFileWithDiagnostics(path string) ([]model.ImportRef, contracts.ParseDiagnostics, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, contracts.ParseDiagnostics{}, err
	}
	return ExtractImportsWithDiagnostics(path, content)
}
