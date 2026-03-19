package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/config"
)

func TestResolveMiningFramework(t *testing.T) {
	if got := resolveMiningFramework(config.ProjectSettings{Framework: "nextjs"}); got != "nextjs" {
		t.Fatalf("expected explicit nextjs framework, got %q", got)
	}
	if got := resolveMiningFramework(config.ProjectSettings{Framework: "generic"}); got != "" {
		t.Fatalf("expected explicit generic to disable framework profile, got %q", got)
	}
}

func TestResolveMiningFrameworkAutoDetectsNextConfig(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "next.config.mjs"), []byte("export default {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := resolveMiningFramework(config.ProjectSettings{}); got != "nextjs" {
		t.Fatalf("expected autodetected nextjs framework, got %q", got)
	}
}

func TestResolveMiningFrameworkAutoDetectsNestedNextConfig(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(dir, "apps", "frontend")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "next.config.js"), []byte("module.exports = {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := resolveMiningFramework(config.ProjectSettings{Roots: []string{"."}}); got != "nextjs" {
		t.Fatalf("expected nested next config to be detected, got %q", got)
	}
}
