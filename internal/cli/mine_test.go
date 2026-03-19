package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/framework"
)

func TestResolveFrameworkExplicit(t *testing.T) {
	res := framework.Resolve("nextjs", nil)
	if res.Selected != "nextjs" || res.Reason != "explicit" {
		t.Fatalf("expected explicit nextjs selection, got %+v", res)
	}
	res = framework.Resolve("react", nil)
	if res.Selected != "react" || res.Reason != "explicit" {
		t.Fatalf("expected explicit react selection, got %+v", res)
	}

	res = framework.Resolve("generic", nil)
	if res.Selected != "" || res.Reason != "explicit_generic" {
		t.Fatalf("expected explicit generic selection, got %+v", res)
	}
}

func TestResolveFrameworkAutoDetectsNextConfig(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "next.config.mjs"), []byte("export default {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := framework.Resolve("", []string{"."})
	if res.Selected != "nextjs" || res.Reason != "auto_detected" {
		t.Fatalf("expected autodetected nextjs framework, got %+v", res)
	}
}

func TestResolveFrameworkAmbiguousFallsBackToGeneric(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "next.config.js"), []byte("module.exports = {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "angular.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := framework.Resolve("", []string{"."})
	if res.Selected != "" {
		t.Fatalf("expected ambiguous autodetect to fall back to generic, got %+v", res)
	}
	if res.Reason != "auto_ambiguous" {
		t.Fatalf("expected auto_ambiguous reason, got %+v", res)
	}
}
