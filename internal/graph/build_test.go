package graph_test

import (
	"testing"

	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/model"
)

func TestBuildCanonicalizesPackageImports(t *testing.T) {
	files := []string{"src/a.ts", "src/b.ts"}
	imports := []model.ImportRef{
		{SourceFile: "src/a.ts", RawImport: "react-dom/client", IsPackageImport: true},
		{SourceFile: "src/b.ts", RawImport: "react-dom/server", IsPackageImport: true},
	}
	g := graph.Build(imports, files)

	if got := g.PackageEdges["src"]["react-dom"]; got != 2 {
		t.Fatalf("expected canonical package edge count 2 for react-dom, got %d", got)
	}
	if _, ok := g.PackageEdges["src"]["react-dom/client"]; ok {
		t.Fatalf("expected no raw subpath key react-dom/client in package edges: %+v", g.PackageEdges["src"])
	}
}
