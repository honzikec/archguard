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

func TestBuildSkipsPathLikePackageSpecifiers(t *testing.T) {
	files := []string{"src/a.ts"}
	imports := []model.ImportRef{
		{SourceFile: "src/a.ts", RawImport: "./local/module", IsPackageImport: true},
		{SourceFile: "src/a.ts", RawImport: "/abs/path/module", IsPackageImport: true},
		{SourceFile: "src/a.ts", RawImport: "../up/module", IsPackageImport: true},
	}
	g := graph.Build(imports, files)

	if len(g.PackageEdges) != 0 {
		t.Fatalf("expected no package edges for path-like specifiers, got %+v", g.PackageEdges)
	}
}
