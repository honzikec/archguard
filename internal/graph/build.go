package graph

import (
	"path/filepath"

	"github.com/honzikec/archguard/internal/model"
)

type Graph struct {
	Nodes map[string]int            // Subtree -> Count of files
	Edges map[string]map[string]int // Subtree A -> Subtree B -> Count of files in A importing B
}

func Build(imports []model.ImportRef, allFiles []string) *Graph {
	g := &Graph{
		Nodes: make(map[string]int),
		Edges: make(map[string]map[string]int),
	}

	// Count files per subtree
	for _, file := range allFiles {
		subtree := filepath.Dir(file)
		g.Nodes[subtree]++
	}

	// Track which files in A have already imported B to avoiding double counting
	seen := make(map[string]map[string]map[string]bool)

	for _, imp := range imports {
		if imp.IsPackageImport {
			continue
		}

		subtreeA := filepath.Dir(imp.SourceFile)
		subtreeB := filepath.Dir(imp.ResolvedPath)

		if subtreeA == subtreeB {
			continue // ignore internal module imports
		}

		if seen[subtreeA] == nil {
			seen[subtreeA] = make(map[string]map[string]bool)
			g.Edges[subtreeA] = make(map[string]int)
		}
		if seen[subtreeA][subtreeB] == nil {
			seen[subtreeA][subtreeB] = make(map[string]bool)
		}

		if !seen[subtreeA][subtreeB][imp.SourceFile] {
			seen[subtreeA][subtreeB][imp.SourceFile] = true
			g.Edges[subtreeA][subtreeB]++
		}
	}

	return g
}
