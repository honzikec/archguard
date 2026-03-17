package graph_test

import (
	"fmt"
	"testing"

	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/model"
)

func BenchmarkBuildGraph1k(b *testing.B) {
	benchmarkBuildGraph(b, 1000)
}

func BenchmarkBuildGraph5k(b *testing.B) {
	benchmarkBuildGraph(b, 5000)
}

func benchmarkBuildGraph(b *testing.B, n int) {
	files := make([]string, 0, n)
	imports := make([]model.ImportRef, 0, n)
	for i := 0; i < n; i++ {
		source := fmt.Sprintf("src/mod%d/file%d.ts", i%20, i)
		target := fmt.Sprintf("src/mod%d/file%d.ts", (i+3)%20, i)
		files = append(files, source)
		imports = append(imports, model.ImportRef{SourceFile: source, ResolvedPath: target, IsPackageImport: false})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = graph.Build(imports, files)
	}
}
