package graph

import (
	"path"

	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pkgid"
)

type Graph struct {
	Nodes        map[string]int
	Edges        map[string]map[string]int
	PackageEdges map[string]map[string]int
	FileEdges    map[string]map[string]struct{}
}

func Build(imports []model.ImportRef, allFiles []string) *Graph {
	g := &Graph{
		Nodes:        map[string]int{},
		Edges:        map[string]map[string]int{},
		PackageEdges: map[string]map[string]int{},
		FileEdges:    map[string]map[string]struct{}{},
	}

	for _, file := range allFiles {
		subtree := path.Dir(file)
		g.Nodes[subtree]++
		if _, ok := g.FileEdges[file]; !ok {
			g.FileEdges[file] = map[string]struct{}{}
		}
	}

	type subtreeEdge struct {
		sourceSubtree string
		targetSubtree string
		sourceFile    string
	}
	type packageEdge struct {
		sourceSubtree string
		pkg           string
		sourceFile    string
	}

	seenSubtree := map[subtreeEdge]struct{}{}
	seenPackages := map[packageEdge]struct{}{}

	for _, imp := range imports {
		sourceSubtree := path.Dir(imp.SourceFile)
		if imp.IsPackageImport {
			pkg := pkgid.Canonical(imp.RawImport)
			if pkg == "" {
				continue
			}
			if _, ok := g.PackageEdges[sourceSubtree]; !ok {
				g.PackageEdges[sourceSubtree] = map[string]int{}
			}
			edge := packageEdge{sourceSubtree, pkg, imp.SourceFile}
			if _, ok := seenPackages[edge]; !ok {
				seenPackages[edge] = struct{}{}
				g.PackageEdges[sourceSubtree][pkg]++
			}
			continue
		}

		if imp.ResolvedPath == "" {
			continue
		}
		if _, ok := g.FileEdges[imp.SourceFile]; !ok {
			g.FileEdges[imp.SourceFile] = map[string]struct{}{}
		}
		g.FileEdges[imp.SourceFile][imp.ResolvedPath] = struct{}{}

		targetSubtree := path.Dir(imp.ResolvedPath)
		if sourceSubtree == targetSubtree {
			continue
		}
		if _, ok := g.Edges[sourceSubtree]; !ok {
			g.Edges[sourceSubtree] = map[string]int{}
		}
		edge := subtreeEdge{sourceSubtree, targetSubtree, imp.SourceFile}
		if _, ok := seenSubtree[edge]; !ok {
			seenSubtree[edge] = struct{}{}
			g.Edges[sourceSubtree][targetSubtree]++
		}
	}

	return g
}
