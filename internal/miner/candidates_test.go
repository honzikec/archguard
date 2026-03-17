package miner_test

import (
	"fmt"
	"testing"

	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
)

func TestProposeNoPackageCandidate(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/domain/f%d.ts", i))
	}
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/infra/f%d.ts", i))
		imports = append(imports, model.ImportRef{SourceFile: fmt.Sprintf("src/infra/f%d.ts", i), RawImport: "axios", IsPackageImport: true})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{MinSupport: 20, MaxPrevalence: 0.02})

	found := false
	for _, c := range candidates {
		if c.Kind == "no_package" && len(c.Scope) > 0 && c.Scope[0] == "src/domain/**" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected no_package candidate for src/domain")
	}
}

func TestDetectCycles(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]int{"src/a": 5, "src/b": 5, "src/c": 5},
		Edges: map[string]map[string]int{
			"src/a": {"src/b": 1},
			"src/b": {"src/c": 1},
			"src/c": {"src/a": 1},
		},
		PackageEdges: map[string]map[string]int{},
		FileEdges:    map[string]map[string]struct{}{},
	}
	cycles := miner.DetectCycles(g)
	if len(cycles) == 0 {
		t.Fatal("expected a cycle")
	}
}
