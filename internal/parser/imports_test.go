package parser_test

import (
	"reflect"
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
	if got, want := len(imports), 5; got != want {
		t.Fatalf("expected %d imports, got %d", want, got)
	}
	got := []string{
		imports[0].RawImport,
		imports[1].RawImport,
		imports[2].RawImport,
		imports[3].RawImport,
		imports[4].RawImport,
	}
	want := []string{"./a", "./side", "./b", "./c", "./d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected imports, want=%v got=%v", want, got)
	}
}

func TestExtractImportsIgnoresCommentsAndStrings(t *testing.T) {
	source := []byte(`
// require("./comment")
const text = "require('./string')";
/*
import x from "./block"
*/
const a = require("./real")
const b = import("./real-dynamic")
const t = ` + "`import('./template')`" + `
`)
	imports := parser.ExtractImports("src/file.ts", source)
	if got, want := len(imports), 2; got != want {
		t.Fatalf("expected %d imports, got %d", want, got)
	}
	got := []string{imports[0].RawImport, imports[1].RawImport}
	want := []string{"./real", "./real-dynamic"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected imports, want=%v got=%v", want, got)
	}
}

func TestExtractImportsSupportsTypeOnlyAndReExportForms(t *testing.T) {
	source := []byte(`
import type { User } from "./types"
export type { Repo } from "./repo"
export * from "./star"
import data from "./data.json" with { type: "json" }
`)
	imports := parser.ExtractImports("src/file.ts", source)
	if got, want := len(imports), 4; got != want {
		t.Fatalf("expected %d imports, got %d", want, got)
	}
	got := []string{imports[0].RawImport, imports[1].RawImport, imports[2].RawImport, imports[3].RawImport}
	want := []string{"./types", "./repo", "./star", "./data.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected imports, want=%v got=%v", want, got)
	}
}

func TestExtractImportsRequiresStringLiteralForDynamicForms(t *testing.T) {
	source := []byte(`
const a = require(name)
const b = import(path)
const c = import(` + "`./ignored`" + `)
const d = require("./real")
`)
	imports := parser.ExtractImports("src/file.ts", source)
	if got, want := len(imports), 1; got != want {
		t.Fatalf("expected %d import, got %d", want, got)
	}
	if imports[0].RawImport != "./real" || imports[0].Kind != "require" {
		t.Fatalf("unexpected import extracted: %+v", imports[0])
	}
}

func TestExtractImportsSupportsMultilineTSXAndImportAttributes(t *testing.T) {
	source := []byte(`
import {
  Button,
  type ButtonProps,
} from "./button"
export {
  Button,
} from "./public"
export * from "./all"
import data from "./data.json" with { type: "json" }
const lazy = import(
  "./lazy"
)
`)
	imports, diagnostics, err := parser.ExtractImportsWithDiagnostics("src/file.tsx", source)
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}
	if diagnostics.NonLiteralDynamicImports != 0 {
		t.Fatalf("expected no ignored dynamic imports, got %+v", diagnostics)
	}
	got := make([]string, 0, len(imports))
	for _, imp := range imports {
		got = append(got, imp.Kind+":"+imp.RawImport)
	}
	want := []string{
		"import:./button",
		"export_from:./public",
		"export_from:./all",
		"import:./data.json",
		"dynamic_import:./lazy",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected imports, want=%v got=%v", want, got)
	}
}

func TestExtractImportsReportsNonLiteralDynamicImports(t *testing.T) {
	source := []byte(`
const path = "./lazy"
const lazy = import(path)
const ok = import("./ok")
`)
	imports, diagnostics, err := parser.ExtractImportsWithDiagnostics("src/file.ts", source)
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}
	if diagnostics.NonLiteralDynamicImports != 1 {
		t.Fatalf("expected one ignored dynamic import, got %+v", diagnostics)
	}
	if len(imports) != 1 || imports[0].RawImport != "./ok" || imports[0].Kind != "dynamic_import" {
		t.Fatalf("unexpected imports: %+v", imports)
	}
}
