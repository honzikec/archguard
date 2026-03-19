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
