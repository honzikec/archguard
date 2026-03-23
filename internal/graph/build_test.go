package graph_test

import (
	"reflect"
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

func TestBuildGraph(t *testing.T) {
	allFiles := []string{
		"src/index.ts",
		"src/domain/user.ts",
		"src/domain/auth.ts",
		"src/infra/db.ts",
	}

	imports := []model.ImportRef{
		{
			SourceFile:   "src/index.ts",
			ResolvedPath: "src/domain/auth.ts",
		},
		{
			SourceFile:   "src/domain/auth.ts",
			ResolvedPath: "src/domain/user.ts",
		},
		{
			SourceFile:      "src/domain/auth.ts",
			RawImport:       "axios",
			IsPackageImport: true,
		},
		{
			SourceFile:   "src/domain/user.ts",
			ResolvedPath: "src/infra/db.ts",
		},
		{
			SourceFile:   "src/infra/db.ts",
			ResolvedPath: "src/domain/user.ts", // Cycle!
		},
		{
			SourceFile:   "src/domain/user.ts",
			ResolvedPath: "src/domain/auth.ts", // Internal cycle! Should be ignored since it's same subtree
		},
	}

	g := graph.Build(imports, allFiles)

	expectedNodes := map[string]int{
		"src":        1,
		"src/domain": 2,
		"src/infra":  1,
	}

	if !reflect.DeepEqual(g.Nodes, expectedNodes) {
		t.Errorf("Nodes mismatch. got: %v, want: %v", g.Nodes, expectedNodes)
	}

	expectedEdges := map[string]map[string]int{
		"src": {
			"src/domain": 1,
		},
		"src/domain": {
			"src/infra": 1,
		},
		"src/infra": {
			"src/domain": 1,
		},
	}

	if !reflect.DeepEqual(g.Edges, expectedEdges) {
		t.Errorf("Edges mismatch. got: %v, want: %v", g.Edges, expectedEdges)
	}

	expectedPackageEdges := map[string]map[string]int{
		"src/domain": {
			"axios": 1,
		},
	}

	if !reflect.DeepEqual(g.PackageEdges, expectedPackageEdges) {
		t.Errorf("PackageEdges mismatch. got: %v, want: %v", g.PackageEdges, expectedPackageEdges)
	}
}

func TestBuildGraphEmpty(t *testing.T) {
	g := graph.Build(nil, nil)
	if len(g.Nodes) > 0 || len(g.Edges) > 0 || len(g.PackageEdges) > 0 {
		t.Errorf("Expected empty graph")
	}
}
