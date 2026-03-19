package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/config"
)

func TestLoadRejectsUnknownField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archguard.yaml")
	content := `version: 1
project:
  roots: ["."]
rules:
  - id: AG-1
    kind: no_import
    severity: error
    scope: ["src/**"]
    target: ["src/infra/**"]
    unknown_field: true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestValidateRejectsInvalidSeverity(t *testing.T) {
	cfg := &config.Config{Version: 1, Rules: []config.Rule{{
		ID:       "AG-1",
		Kind:     config.KindNoImport,
		Severity: "fatal",
		Scope:    []string{"src/**"},
		Target:   []string{"src/infra/**"},
	}}}
	if err := config.Validate(cfg); err == nil {
		t.Fatal("expected severity validation error")
	}
}

func TestValidateRejectsInvalidRegex(t *testing.T) {
	cfg := &config.Config{Version: 1, Rules: []config.Rule{{
		ID:       "AG-1",
		Kind:     config.KindFilePattern,
		Severity: config.SeverityWarning,
		Scope:    []string{"src/**"},
		Target:   []string{"("},
	}}}
	if err := config.Validate(cfg); err == nil {
		t.Fatal("expected regex validation error")
	}
}

func TestValidatePatternRule(t *testing.T) {
	cfg := &config.Config{Version: 1, Rules: []config.Rule{{
		ID:       "AG-PATTERN",
		Kind:     config.KindPattern,
		Template: "dependency_constraint",
		Severity: config.SeverityWarning,
		Scope:    []string{"src/domain/**"},
		Target:   []string{"src/infra/**"},
		Params: map[string]string{
			"relation": "imports",
		},
	}}}
	if err := config.Validate(cfg); err != nil {
		t.Fatalf("expected pattern rule to validate, got error: %v", err)
	}
}

func TestValidateProjectFramework(t *testing.T) {
	validFrameworks := []string{"generic", "nextjs", "react", "react_router", "react_native", "angular"}
	for _, frameworkID := range validFrameworks {
		valid := &config.Config{
			Version: 1,
			Project: config.ProjectSettings{Framework: frameworkID},
			Rules:   []config.Rule{},
		}
		if err := config.Validate(valid); err != nil {
			t.Fatalf("expected framework %q to validate, got error: %v", frameworkID, err)
		}
	}

	invalid := &config.Config{
		Version: 1,
		Project: config.ProjectSettings{Framework: "rails"},
		Rules:   []config.Rule{},
	}
	if err := config.Validate(invalid); err == nil {
		t.Fatal("expected unsupported framework validation error")
	}
}

func TestValidateProjectLanguage(t *testing.T) {
	validLanguages := []string{"auto", "javascript", "php"}
	for _, languageID := range validLanguages {
		valid := &config.Config{
			Version: 1,
			Project: config.ProjectSettings{Language: languageID},
			Rules:   []config.Rule{},
		}
		if err := config.Validate(valid); err != nil {
			t.Fatalf("expected language %q to validate, got error: %v", languageID, err)
		}
	}

	invalid := &config.Config{
		Version: 1,
		Project: config.ProjectSettings{Language: "python"},
		Rules:   []config.Rule{},
	}
	if err := config.Validate(invalid); err == nil {
		t.Fatal("expected unsupported language validation error")
	}
}
