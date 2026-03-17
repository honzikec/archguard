package graph

import (
	"path"

	"github.com/honzikec/archguard/internal/model"
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

	seenSubtree := map[string]map[string]map[string]struct{}{}
	seenPackages := map[string]map[string]map[string]struct{}{}

	for _, imp := range imports {
		sourceSubtree := path.Dir(imp.SourceFile)
		if imp.IsPackageImport {
			if _, ok := g.PackageEdges[sourceSubtree]; !ok {
				g.PackageEdges[sourceSubtree] = map[string]int{}
			}
			if _, ok := seenPackages[sourceSubtree]; !ok {
				seenPackages[sourceSubtree] = map[string]map[string]struct{}{}
			}
			if _, ok := seenPackages[sourceSubtree][imp.RawImport]; !ok {
				seenPackages[sourceSubtree][imp.RawImport] = map[string]struct{}{}
			}
			if _, ok := seenPackages[sourceSubtree][imp.RawImport][imp.SourceFile]; !ok {
				seenPackages[sourceSubtree][imp.RawImport][imp.SourceFile] = struct{}{}
				g.PackageEdges[sourceSubtree][imp.RawImport]++
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
		if _, ok := seenSubtree[sourceSubtree]; !ok {
			seenSubtree[sourceSubtree] = map[string]map[string]struct{}{}
		}
		if _, ok := seenSubtree[sourceSubtree][targetSubtree]; !ok {
			seenSubtree[sourceSubtree][targetSubtree] = map[string]struct{}{}
		}
		if _, ok := seenSubtree[sourceSubtree][targetSubtree][imp.SourceFile]; !ok {
			seenSubtree[sourceSubtree][targetSubtree][imp.SourceFile] = struct{}{}
			g.Edges[sourceSubtree][targetSubtree]++
		}
	}

	return g
}
