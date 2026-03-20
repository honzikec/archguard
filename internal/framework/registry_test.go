package framework

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestRegisteredFrameworksDeterministic(t *testing.T) {
	first := RegisteredFrameworks()
	second := RegisteredFrameworks()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic framework list, got %v vs %v", first, second)
	}
	if len(first) < 5 {
		t.Fatalf("expected framework list to include generic + builtins, got %v", first)
	}
	if first[0] != "generic" {
		t.Fatalf("expected generic as first framework, got %v", first)
	}
	foundReact := false
	for _, id := range first {
		if id == "react" {
			foundReact = true
			break
		}
	}
	if !foundReact {
		t.Fatalf("expected registered frameworks to include react, got %v", first)
	}
}

func TestResolveAutodetectAmbiguous(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "next.config.js"), []byte("module.exports={}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "angular.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Resolve("", []string{dir})
	if res.Selected != "" {
		t.Fatalf("expected ambiguous detection to avoid auto-selection, got %+v", res)
	}
	if res.Reason != "auto_ambiguous" {
		t.Fatalf("expected auto_ambiguous reason, got %+v", res)
	}
	sorted := append([]string{}, res.Matched...)
	sort.Strings(sorted)
	if len(sorted) < 2 {
		t.Fatalf("expected at least two matched frameworks, got %+v", res)
	}
}

func TestResolveAutodetectRankedSelectsUniqueStrongest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src", "routes"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Both nextjs and react_router dependencies exist; routes directory is a stronger react-router signal.
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"next":"15.0.0","react-router-dom":"7.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Resolve("", []string{dir})
	if res.Selected != "react_router" {
		t.Fatalf("expected react_router from ranked auto-detection, got %+v", res)
	}
	if res.Reason != "auto_ranked" {
		t.Fatalf("expected auto_ranked reason, got %+v", res)
	}
}

func TestResolveAutodetectWeakSignalFallsBackToGeneric(t *testing.T) {
	dir := t.TempDir()
	// Dependency-only nextjs signal is too weak for auto-selection.
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"next":"15.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Resolve("", []string{dir})
	if res.Selected != "" {
		t.Fatalf("expected weak signal fallback to generic, got %+v", res)
	}
	if res.Reason != "auto_weak" {
		t.Fatalf("expected auto_weak reason, got %+v", res)
	}
	if len(res.Matched) != 1 || res.Matched[0] != "nextjs" {
		t.Fatalf("expected matched to include nextjs weak signal, got %+v", res)
	}
}
