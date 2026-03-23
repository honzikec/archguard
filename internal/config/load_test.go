package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/config"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "archguard.yaml")
	content := []byte(`
version: 1
project:
  language: javascript
rules:
  - id: custom-rule
    kind: no_import
    severity: error
    scope: ["src/**"]
    target: ["test/**"]
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Project.Language != "javascript" {
		t.Errorf("expected language javascript, got %s", cfg.Project.Language)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].ID != "custom-rule" {
		t.Errorf("expected rule ID custom-rule, got %s", cfg.Rules[0].ID)
	}
	
	// Test defaults were applied
	if len(cfg.Project.Roots) == 0 || cfg.Project.Roots[0] != "." {
		t.Errorf("expected default roots ['.'], got %v", cfg.Project.Roots)
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := config.Load("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent config, got nil")
	}
}

func TestLoadConfigInvalidSchema(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "archguard.yaml")
	content := []byte(`
version: 1
project:
  invalidField: true
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(configPath)
	if err == nil {
		t.Fatal("expected error for invalid schema, got nil")
	}
}
