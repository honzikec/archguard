package pathutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/pathutil"
)

func TestResolveRelativeAndIndexAndAlias(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src", "domain", "user.ts"), "import x from '../infra/db'")
	mustWrite(t, filepath.Join(dir, "src", "infra", "db.ts"), "export const db = {}")
	mustWrite(t, filepath.Join(dir, "src", "utils", "index.ts"), "export const util = {}")
	mustWrite(t, filepath.Join(dir, "tsconfig.json"), `{"compilerOptions":{"baseUrl":".","paths":{"@/*":["src/*"]}}}`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("src/domain/user.ts", "../infra/db")
	if isPkg || resolved != "src/infra/db.ts" {
		t.Fatalf("unexpected relative resolution: resolved=%s isPkg=%t", resolved, isPkg)
	}

	resolved, isPkg = resolver.Resolve("src/domain/user.ts", "@/infra/db")
	if isPkg || resolved != "src/infra/db.ts" {
		t.Fatalf("unexpected alias resolution: resolved=%s isPkg=%t", resolved, isPkg)
	}

	resolved, isPkg = resolver.Resolve("src/domain/user.ts", "@/utils")
	if isPkg || resolved != "src/utils/index.ts" {
		t.Fatalf("unexpected index resolution: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolvePHPRelativeImport(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "app", "controller.php"), `<?php require "./infra/db.php";`)
	mustWrite(t, filepath.Join(dir, "app", "infra", "db.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("app/controller.php", "./infra/db.php")
	if isPkg || resolved != "app/infra/db.php" {
		t.Fatalf("unexpected php resolution: resolved=%s isPkg=%t", resolved, isPkg)
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
