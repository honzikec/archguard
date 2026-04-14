package analysis_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/analysis"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/language/javascript"
)

func BenchmarkRunLargeTypeScriptProject(b *testing.B) {
	dir := b.TempDir()
	mustWriteBenchFile(b, filepath.Join(dir, "tsconfig.json"), `{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@shared/*": ["src/shared/*"]
    }
  }
}`)
	mustWriteBenchFile(b, filepath.Join(dir, "src", "shared", "util.ts"), `export const util = 1`)
	for i := 0; i < 500; i++ {
		mustWriteBenchFile(b, filepath.Join(dir, "src", "feature", fmt.Sprintf("file%03d.ts", i)), `import { util } from "@shared/util"
export const value = util
`)
	}

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		b.Fatal(err)
	}

	project := config.ProjectSettings{
		Roots:   []string{"src"},
		Include: []string{"**/*.ts"},
		Exclude: []string{"**/node_modules/**"},
	}
	adapter := javascript.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := analysis.Run(project, adapter, analysis.Options{})
		if err != nil {
			b.Fatal(err)
		}
		if got, want := len(result.Files), 501; got != want {
			b.Fatalf("expected %d files, got %d", want, got)
		}
	}
}

func mustWriteBenchFile(tb testing.TB, path, content string) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatal(err)
	}
}
