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

func TestResolveUnresolvedLocalImportsAreNotPackages(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src", "domain", "user.ts"), "export {}")

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("src/domain/user.ts", "../infra/missing")
	if isPkg || resolved != "" {
		t.Fatalf("expected unresolved relative import to remain non-package: resolved=%s isPkg=%t", resolved, isPkg)
	}

	resolved, isPkg = resolver.Resolve("src/domain/user.ts", "/missing/absolute")
	if isPkg || resolved != "" {
		t.Fatalf("expected unresolved absolute import to remain non-package: resolved=%s isPkg=%t", resolved, isPkg)
	}

	resolved, isPkg = resolver.Resolve("src/domain/user.ts", "react")
	if !isPkg || resolved != "" {
		t.Fatalf("expected bare package import classification: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolveAliasFromJSONCTSConfig(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src", "infra", "db.ts"), "export const db = {}")
	mustWrite(t, filepath.Join(dir, "tsconfig.json"), `/* generated */
{
  // tsconfig can include comments and trailing commas
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*",],
    },
  },
}`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("src/domain/user.ts", "@/infra/db")
	if isPkg || resolved != "src/infra/db.ts" {
		t.Fatalf("unexpected JSONC alias resolution: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolverAcceptsCommentedTSConfigWithoutCompilerOptions(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "tsconfig.json"), `/* generated */
{
  // babel-style config
  "extends": ["./base.json",],
}`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if _, err := pathutil.NewResolver(".", config.ProjectSettings{}); err != nil {
		t.Fatalf("expected resolver to accept commented tsconfig, got error: %v", err)
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
