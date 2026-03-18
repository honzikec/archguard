package astfacts_test

import (
	"testing"

	"github.com/honzikec/archguard/internal/astfacts"
)

func TestParseContentCapturesImportsClassesAndNew(t *testing.T) {
	content := []byte(`
import DefaultSvc, { UserService as RenamedService, Other } from "../services/user.service";
import * as NS from "../lib/ns";
import "../polyfills";

class LocalService {}
export class ExportedService {}
export { LocalService as default };
const a = new LocalService();
const b = new RenamedService();
const c = new ctor();
`)
	facts := astfacts.ParseContent("src/feature/controller.ts", content)

	if len(facts.Classes) != 2 || facts.Classes[0].Name != "LocalService" || facts.Classes[1].Name != "ExportedService" {
		t.Fatalf("unexpected classes: %+v", facts.Classes)
	}
	if len(facts.Imports) != 3 {
		t.Fatalf("expected 3 imports, got %d", len(facts.Imports))
	}
	if len(facts.NewExprs) != 3 {
		t.Fatalf("expected 3 new expressions, got %d", len(facts.NewExprs))
	}
	if facts.NewExprs[0].ClassName != "LocalService" || facts.NewExprs[1].ClassName != "RenamedService" {
		t.Fatalf("unexpected new expressions: %+v", facts.NewExprs)
	}
	if facts.NewExprs[2].ClassName != "ctor" || !facts.NewExprs[2].IsIdentifier {
		t.Fatalf("expected identifier constructor for ctor, got %+v", facts.NewExprs[2])
	}
	if facts.ExportedClassByName["ExportedService"] != "ExportedService" {
		t.Fatalf("expected exported class mapping, got %+v", facts.ExportedClassByName)
	}
	if facts.DefaultExportedClass != "LocalService" {
		t.Fatalf("expected default exported class LocalService, got %q", facts.DefaultExportedClass)
	}
}

func TestParseContentCapturesDefaultExportClassDeclaration(t *testing.T) {
	content := []byte(`
export default class UserService {}
`)
	facts := astfacts.ParseContent("src/services/user.service.ts", content)
	if facts.DefaultExportedClass != "UserService" {
		t.Fatalf("expected default export class to be UserService, got %q", facts.DefaultExportedClass)
	}
}
