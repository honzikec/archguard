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

func TestParseContentCapturesPHPFacts(t *testing.T) {
	content := []byte(`<?php
namespace App\Controllers;
use App\Services\MailService as Mail;
class OrderController {
    public function create(): void {
        $a = new Mail();
        $b = new \App\Services\AuditService();
        $c = new LocalHelper();
    }
}
`)
	facts := astfacts.ParseContent("app/Controllers/OrderController.php", content)

	if facts.Namespace != `App\Controllers` {
		t.Fatalf("expected namespace App\\\\Controllers, got %q", facts.Namespace)
	}
	if len(facts.Classes) != 1 || facts.Classes[0].Name != "OrderController" {
		t.Fatalf("unexpected classes: %+v", facts.Classes)
	}
	if len(facts.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(facts.Imports))
	}
	if facts.Imports[0].Module != `App\Services\MailService` {
		t.Fatalf("unexpected php import module: %+v", facts.Imports[0])
	}
	if facts.Imports[0].Named["Mail"] != "MailService" {
		t.Fatalf("expected alias Mail->MailService, got %+v", facts.Imports[0].Named)
	}
	if len(facts.NewExprs) != 3 {
		t.Fatalf("expected 3 new expressions, got %d", len(facts.NewExprs))
	}
	if facts.NewExprs[0].ClassName != "Mail" || !facts.NewExprs[0].IsIdentifier {
		t.Fatalf("unexpected first php constructor: %+v", facts.NewExprs[0])
	}
	if facts.NewExprs[1].ClassName != `App\Services\AuditService` || !facts.NewExprs[1].IsIdentifier {
		t.Fatalf("unexpected qualified php constructor: %+v", facts.NewExprs[1])
	}
	if facts.NewExprs[2].ClassName != "LocalHelper" || !facts.NewExprs[2].IsIdentifier {
		t.Fatalf("unexpected local php constructor: %+v", facts.NewExprs[2])
	}
}
