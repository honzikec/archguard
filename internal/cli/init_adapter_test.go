package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitAdapterScaffold(t *testing.T) {
	dir := t.TempDir()
	code, out, errOut := runCmdInDir(t, dir, []string{
		"init", "adapter",
		"--name", "python",
		"--dir", "scaffold/languages",
	})
	if code != 0 {
		t.Fatalf("expected init adapter exit 0, got %d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "Created adapter scaffold") {
		t.Fatalf("expected success output, got %s", out)
	}

	adapterPath := filepath.Join(dir, "scaffold", "languages", "python", "adapter.go")
	if _, err := os.Stat(adapterPath); err != nil {
		t.Fatalf("expected adapter scaffold to exist: %v", err)
	}
	content, err := os.ReadFile(adapterPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `return "python"`) {
		t.Fatalf("expected generated adapter id in scaffold, got: %s", string(content))
	}
}
