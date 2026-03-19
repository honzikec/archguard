package javascript

import (
	"os"
	"strings"

	"github.com/honzikec/archguard/internal/language/common"
	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/parser"
)

type Adapter struct{}

func New() contracts.Adapter {
	return Adapter{}
}

func (Adapter) ID() string {
	return "javascript"
}

func (Adapter) Detect(roots []string) contracts.Detection {
	if common.HasAnyFileNamed(roots, []string{"tsconfig.json", "jsconfig.json"}) {
		return contracts.Detection{Matched: true, Reason: "tsconfig/jsconfig found"}
	}
	if common.HasFileWithSuffix(roots, []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}, 8) {
		return contracts.Detection{Matched: true, Reason: "js/ts files detected"}
	}
	return contracts.Detection{}
}

func (Adapter) SupportsFile(path string) bool {
	if strings.Contains(path, ".gen.") {
		return false
	}
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func (Adapter) ParseFile(path string) ([]model.ImportRef, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parser.ExtractImports(path, content), nil
}
