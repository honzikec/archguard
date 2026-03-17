package graph

import (
	"path/filepath"

	"github.com/honzikec/archguard/internal/model"
)

type Graph struct {
	Nodes        map[string]int            // Subtree -> Count of files
	Edges        map[string]map[string]int // Subtree A -> Subtree B -> Count of files in A importing B
	PackageEdges map[string]map[string]int // Subtree -> External Package -> Count of files in Subtree importing Package
}

func Build(imports []model.ImportRef, allFiles []string) *Graph {
	g := &Graph{
		Nodes:        make(map[string]int),
		Edges:        make(map[string]map[string]int),
		PackageEdges: make(map[string]map[string]int),
	}

	// Count files per subtree
	for _, file := range allFiles {
		subtree := filepath.Dir(file)
		g.Nodes[subtree]++
	}

	// Track which files in A have already imported B or Package to avoid double counting
	seenEdges := make(map[string]map[string]map[string]bool)
	seenPackages := make(map[string]map[string]map[string]bool)

	for _, imp := range imports {
		subtreeA := filepath.Dir(imp.SourceFile)

		if imp.IsPackageImport {
			pkg := imp.RawImport
			if seenPackages[subtreeA] == nil {
				seenPackages[subtreeA] = make(map[string]map[string]bool)
				g.PackageEdges[subtreeA] = make(map[string]int)
			}
			if seenPackages[subtreeA][pkg] == nil {
				seenPackages[subtreeA][pkg] = make(map[string]bool)
			}
			if !seenPackages[subtreeA][pkg][imp.SourceFile] {
				seenPackages[subtreeA][pkg][imp.SourceFile] = true
				g.PackageEdges[subtreeA][pkg]++
			}
			continue
		}

		subtreeB := filepath.Dir(imp.ResolvedPath)

		if subtreeA == subtreeB {
			continue // ignore internal module imports
		}

		if seenEdges[subtreeA] == nil {
			seenEdges[subtreeA] = make(map[string]map[string]bool)
			g.Edges[subtreeA] = make(map[string]int)
		}
		if seenEdges[subtreeA][subtreeB] == nil {
			seenEdges[subtreeA][subtreeB] = make(map[string]bool)
		}

		if !seenEdges[subtreeA][subtreeB][imp.SourceFile] {
			seenEdges[subtreeA][subtreeB][imp.SourceFile] = true
			g.Edges[subtreeA][subtreeB]++
		}
	}

	return g
}
