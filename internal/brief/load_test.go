package brief

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsUnknownTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "brief.yaml")
	content := `version: 1
unknown_field: true
policies:
  - type: no_cycle
    scope: ["src/**"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "brief schema validation failed") {
		t.Fatalf("expected schema validation error, got: %v", err)
	}
}

func TestLoadRejectsUnknownPolicyField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "brief.yaml")
	content := `version: 1
policies:
  - type: no_cycle
    scope: ["src/**"]
    unexpected: true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "brief schema validation failed") {
		t.Fatalf("expected schema validation error, got: %v", err)
	}
}
