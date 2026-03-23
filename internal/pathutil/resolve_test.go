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

func TestResolvePHPNamespaceImportFromComposerPSR4(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "composer.json"), `{
  "autoload": {
    "psr-4": {
      "App\\\\": "app/"
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "app", "Domain", "User.php"), `<?php`)
	mustWrite(t, filepath.Join(dir, "app", "Infra", "Db.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{Roots: []string{"app"}})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("app/Domain/User.php", `App\Infra\Db`)
	if isPkg || resolved != "app/Infra/Db.php" {
		t.Fatalf("unexpected php namespace resolution: resolved=%s isPkg=%t", resolved, isPkg)
	}

	resolved, isPkg = resolver.Resolve("app/Domain/User.php", `\App\Infra\Db`)
	if isPkg || resolved != "app/Infra/Db.php" {
		t.Fatalf("unexpected leading slash php namespace resolution: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolvePHPNamespaceImportFromWorkspaceComposerPSR4(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "apps", "api", "composer.json"), `{
  "autoload-dev": {
    "psr-4": {
      "Demo\\\\": ["src/"]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "apps", "api", "src", "Domain", "User.php"), `<?php`)
	mustWrite(t, filepath.Join(dir, "apps", "api", "src", "Infra", "Db.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{Roots: []string{"apps/api"}})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("apps/api/src/Domain/User.php", `Demo\Infra\Db`)
	if isPkg || resolved != "apps/api/src/Infra/Db.php" {
		t.Fatalf("unexpected workspace composer namespace resolution: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolvePHPNamespaceImportWithoutComposerFallback(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "common", "models", "User.php"), `<?php`)
	mustWrite(t, filepath.Join(dir, "backend", "controllers", "SiteController.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("backend/controllers/SiteController.php", `common\models\User`)
	if isPkg || resolved != "common/models/User.php" {
		t.Fatalf("unexpected fallback php namespace resolution: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolvePHPNamespaceLocalUnresolvedNotPackage(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "common", "models", "User.php"), `<?php`)
	mustWrite(t, filepath.Join(dir, "backend", "controllers", "SiteController.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("backend/controllers/SiteController.php", `common\models\MissingClass`)
	if isPkg || resolved != "" {
		t.Fatalf("expected unresolved local namespace to remain non-package: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolvePHPNamespaceExternalRemainsPackage(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "backend", "controllers", "SiteController.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("backend/controllers/SiteController.php", `yii\web\Controller`)
	if !isPkg || resolved != "" {
		t.Fatalf("expected external php namespace to remain package import: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolvePHPAliasLikeImportIsNotPackage(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "backend", "config", "main.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("backend/config/main.php", "@common/config/main-local.php")
	if isPkg || resolved != "" {
		t.Fatalf("expected php alias-like import to remain non-package: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolveJSScopePackageRemainsPackage(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "src", "index.ts"), "import x from '@scope/pkg'")

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("src/index.ts", "@scope/pkg")
	if !isPkg || resolved != "" {
		t.Fatalf("expected js scoped package to remain package import: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolvePHPBareSymbolImportIsNotPackage(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "backend", "controllers", "SiteController.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("backend/controllers/SiteController.php", "Exception")
	if isPkg || resolved != "" {
		t.Fatalf("expected php bare class symbol to remain non-package: resolved=%s isPkg=%t", resolved, isPkg)
	}

	resolved, isPkg = resolver.Resolve("backend/controllers/SiteController.php", "str_starts_with")
	if isPkg || resolved != "" {
		t.Fatalf("expected php bare function symbol to remain non-package: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolvePHPIncludeLikePathFromSourceDir(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "backend", "views", "site", "index.php"), `<?php include "inc/header.php";`)
	mustWrite(t, filepath.Join(dir, "backend", "views", "site", "inc", "header.php"), `<?php`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("backend/views/site/index.php", "inc/header.php")
	if isPkg || resolved != "backend/views/site/inc/header.php" {
		t.Fatalf("expected php include-like path to resolve locally: resolved=%s isPkg=%t", resolved, isPkg)
	}
}

func TestResolveUnresolvedPHPIncludeLikePathIsNotPackage(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "backend", "views", "site", "index.php"), `<?php include "inc/header.php";`)

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	resolver, err := pathutil.NewResolver(".", config.ProjectSettings{})
	if err != nil {
		t.Fatal(err)
	}

	resolved, isPkg := resolver.Resolve("backend/views/site/index.php", "inc/header.php")
	if isPkg || resolved != "" {
		t.Fatalf("expected unresolved php include-like path to remain non-package: resolved=%s isPkg=%t", resolved, isPkg)
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
