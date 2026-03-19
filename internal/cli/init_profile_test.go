package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeProfileID(t *testing.T) {
	if got := normalizeProfileID(" React Router "); got != "react_router" {
		t.Fatalf("expected react_router, got %q", got)
	}
	if got := normalizeProfileID("123-angular"); got != "123_angular" {
		t.Fatalf("expected 123_angular, got %q", got)
	}
}

func TestInitProfileScaffold(t *testing.T) {
	dir := t.TempDir()
	code, out, errOut := runCmdInDir(t, dir, []string{
		"init", "profile",
		"--name", "react-router",
		"--dir", "scaffold/profiles",
	})
	if code != 0 {
		t.Fatalf("expected init profile exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "Created profile scaffold") {
		t.Fatalf("expected success output, got %s", out)
	}

	profilePath := filepath.Join(dir, "scaffold", "profiles", "react_router", "profile.go")
	if _, err := os.Stat(profilePath); err != nil {
		t.Fatalf("expected profile scaffold to exist: %v", err)
	}
	content, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `return "react_router"`) {
		t.Fatalf("expected generated profile id in scaffold, got: %s", string(content))
	}
}
