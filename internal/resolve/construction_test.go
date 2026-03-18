package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/resolve"
)

func TestResolveConstructions_LocalAndImportedAndAlias(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	mustWrite(t, "src/services/user.service.ts", "export default class UserService {}\nexport { UserService }")
	mustWrite(t, "src/feature/local.ts", "class LocalService {}\nconst a = new LocalService()")
	mustWrite(t, "src/feature/named.ts", "import { UserService } from '../services/user.service'\nconst b = new UserService()")
	mustWrite(t, "src/feature/default.ts", "import UserService from '../services/user.service'\nconst c = new UserService()")
	mustWrite(t, "src/feature/alias.ts", "import { UserService as Svc } from '@/services/user.service'\nconst d = new Svc()")

	project := config.ProjectSettings{Aliases: map[string][]string{"@/*": []string{"src/*"}}}
	files := []string{
		"src/services/user.service.ts",
		"src/feature/local.ts",
		"src/feature/named.ts",
		"src/feature/default.ts",
		"src/feature/alias.ts",
	}
	resolved, err := resolve.ResolveConstructions(files, project, []string{"src/services/**"}, ".*Service$")
	if err != nil {
		t.Fatalf("resolve constructions failed: %v", err)
	}
	if len(resolved) < 4 {
		t.Fatalf("expected at least 4 constructions, got %d", len(resolved))
	}

	foundNamed := false
	foundDefault := false
	foundAlias := false
	for _, c := range resolved {
		if c.FilePath == "src/feature/named.ts" && c.IsResolved && c.IsService {
			foundNamed = true
		}
		if c.FilePath == "src/feature/default.ts" && c.IsResolved && c.IsService {
			foundDefault = true
		}
		if c.FilePath == "src/feature/alias.ts" && c.IsResolved && c.IsService {
			foundAlias = true
		}
	}
	if !foundNamed || !foundDefault || !foundAlias {
		t.Fatalf("expected resolved service constructions for named/default/alias imports: %+v", resolved)
	}
}

func TestResolveConstructions_UnresolvedDynamic(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	mustWrite(t, "src/feature/dynamic.ts", "const k = 'x'\nconst svc = new services[k]()")
	files := []string{"src/feature/dynamic.ts"}
	resolved, err := resolve.ResolveConstructions(files, config.ProjectSettings{}, []string{"src/services/**"}, ".*Service$")
	if err != nil {
		t.Fatalf("resolve constructions failed: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 construction, got %d", len(resolved))
	}
	if resolved[0].IsResolved {
		t.Fatalf("expected unresolved construction, got %+v", resolved[0])
	}
	if resolved[0].UnresolvedReason != "dynamic_constructor" {
		t.Fatalf("expected dynamic constructor reason, got %+v", resolved[0])
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
