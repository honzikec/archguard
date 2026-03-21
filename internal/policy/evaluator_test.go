package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/policy"
)

func TestEvaluate_NoImport(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Rules: []config.Rule{{
			ID:       "AG-1",
			Kind:     config.KindNoImport,
			Severity: config.SeverityError,
			Scope:    []string{"src/domain/**"},
			Target:   []string{"src/infra/**"},
		}},
	}
	imports := []model.ImportRef{{
		SourceFile:      "src/domain/user.ts",
		RawImport:       "../infra/db",
		ResolvedPath:    "src/infra/db.ts",
		IsPackageImport: false,
		Line:            1,
		Column:          1,
	}}
	files := []string{"src/domain/user.ts", "src/infra/db.ts"}
	g := graph.Build(imports, files)

	findings, err := policy.Evaluate(cfg, imports, files, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].RuleID != "AG-1" {
		t.Fatalf("unexpected rule id: %s", findings[0].RuleID)
	}
}

func TestEvaluate_NoPackage(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Rules: []config.Rule{{
			ID:       "AG-2",
			Kind:     config.KindNoPackage,
			Severity: config.SeverityWarning,
			Scope:    []string{"src/domain/**"},
			Target:   []string{"axios"},
		}},
	}
	imports := []model.ImportRef{{
		SourceFile:      "src/domain/user.ts",
		RawImport:       "axios",
		IsPackageImport: true,
		Line:            1,
		Column:          1,
	}}
	files := []string{"src/domain/user.ts"}
	g := graph.Build(imports, files)

	findings, err := policy.Evaluate(cfg, imports, files, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestEvaluate_NoPackageMatchesSubpathImport(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Rules: []config.Rule{{
			ID:       "AG-2A",
			Kind:     config.KindNoPackage,
			Severity: config.SeverityWarning,
			Scope:    []string{"src/domain/**"},
			Target:   []string{"react-dom"},
		}},
	}
	imports := []model.ImportRef{{
		SourceFile:      "src/domain/user.ts",
		RawImport:       "react-dom/client",
		IsPackageImport: true,
		Line:            1,
		Column:          1,
	}}
	files := []string{"src/domain/user.ts"}
	g := graph.Build(imports, files)

	findings, err := policy.Evaluate(cfg, imports, files, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for subpath package import, got %d", len(findings))
	}
}

func TestEvaluate_FilePattern(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Rules: []config.Rule{{
			ID:       "AG-3",
			Kind:     config.KindFilePattern,
			Severity: config.SeverityWarning,
			Scope:    []string{"src/services/**"},
			Target:   []string{"^.*\\.service\\.ts$"},
		}},
	}
	files := []string{"src/services/user.ts", "src/services/order.service.ts"}
	g := graph.Build(nil, files)

	findings, err := policy.Evaluate(cfg, nil, files, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].FilePath != "src/services/user.ts" {
		t.Fatalf("unexpected finding path: %s", findings[0].FilePath)
	}
}

func TestEvaluate_NoCycle(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Rules: []config.Rule{{
			ID:       "AG-4",
			Kind:     config.KindNoCycle,
			Severity: config.SeverityError,
			Scope:    []string{"src/**"},
		}},
	}
	files := []string{"src/a/a.ts", "src/b/b.ts", "src/c/c.ts"}
	imports := []model.ImportRef{
		{SourceFile: "src/a/a.ts", ResolvedPath: "src/b/b.ts"},
		{SourceFile: "src/b/b.ts", ResolvedPath: "src/c/c.ts"},
		{SourceFile: "src/c/c.ts", ResolvedPath: "src/a/a.ts"},
	}
	g := graph.Build(imports, files)

	findings, err := policy.Evaluate(cfg, imports, files, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected at least one cycle finding")
	}
}

func TestEvaluate_PatternDependencyConstraint(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Rules: []config.Rule{{
			ID:       "AG-PATTERN-1",
			Kind:     config.KindPattern,
			Template: "dependency_constraint",
			Severity: config.SeverityError,
			Scope:    []string{"src/domain/**"},
			Target:   []string{"src/infra/**"},
			Params: map[string]string{
				"relation": "imports",
			},
		}},
	}
	imports := []model.ImportRef{{
		SourceFile:      "src/domain/user.ts",
		RawImport:       "../infra/db",
		ResolvedPath:    "src/infra/db.ts",
		IsPackageImport: false,
		Line:            1,
		Column:          1,
	}}
	files := []string{"src/domain/user.ts", "src/infra/db.ts"}
	g := graph.Build(imports, files)

	findings, err := policy.Evaluate(cfg, imports, files, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestEvaluate_PatternConstructionPolicy(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	mustWriteTestFile(t, filepath.Join("src", "services", "user.service.ts"), "export class UserService {}")
	mustWriteTestFile(t, filepath.Join("src", "feature", "controller.ts"), "import { UserService } from '../services/user.service'\nconst svc = new UserService()")

	cfg := &config.Config{
		Version: 1,
		Rules: []config.Rule{{
			ID:       "AG-PATTERN-2",
			Kind:     config.KindPattern,
			Template: "construction_policy",
			Severity: config.SeverityWarning,
			Scope:    []string{"src/**"},
			Target:   []string{"src/services/**"},
			Except:   []string{"src/bootstrap/**"},
			Params: map[string]string{
				"service_name_regex": ".*Service$",
			},
		}},
	}
	files := []string{"src/services/user.service.ts", "src/feature/controller.ts"}
	g := graph.Build(nil, files)
	findings, err := policy.Evaluate(cfg, nil, files, g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected construction_policy finding")
	}
}

func mustWriteTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
