package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverRootsFromPackageJSONWorkspaces(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package.json"), `{"workspaces":["apps/*","packages/*"]}`)
	mustWrite(t, filepath.Join(dir, "apps", "web", "package.json"), `{"name":"web"}`)
	mustWrite(t, filepath.Join(dir, "packages", "ui", "package.json"), `{"name":"ui"}`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	got, err := DiscoverRoots([]string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"apps/web", "packages/ui"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v got %v", want, got)
	}
}

func TestDiscoverRootsFromPNPMWorkspace(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "pnpm-workspace.yaml"), "packages:\n  - apps/*\n  - packages/*\n")
	mustWrite(t, filepath.Join(dir, "apps", "api", "package.json"), `{"name":"api"}`)
	mustWrite(t, filepath.Join(dir, "packages", "core", "package.json"), `{"name":"core"}`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	got, err := DiscoverRoots([]string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"apps/api", "packages/core"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v got %v", want, got)
	}
}

func TestDiscoverRootsFallsBackToRoot(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package.json"), `{"name":"single"}`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	got, err := DiscoverRoots([]string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v got %v", want, got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
