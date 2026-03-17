package parser_test

import (
	"testing"

	"github.com/honzikec/archguard/internal/parser"
)

func TestExtractImportsSupportsImportForms(t *testing.T) {
	source := []byte(`
import x from "./a"
import "./side"
export { y } from "./b"
const z = require("./c")
const d = import("./d")
`)
	imports := parser.ExtractImports("src/file.ts", source)
	if len(imports) != 5 {
		t.Fatalf("expected 5 imports, got %d", len(imports))
	}
}
