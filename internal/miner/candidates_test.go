package miner_test

import (
	"fmt"
	"testing"

	"github.com/honzikec/archguard/internal/config"
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

func TestDetectCycleComponentsCollapsesEquivalentCycles(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]int{"src/a": 5, "src/b": 5, "src/c": 5},
		Edges: map[string]map[string]int{
			"src/a": {"src/b": 1, "src/c": 1},
			"src/b": {"src/c": 1, "src/a": 1},
			"src/c": {"src/a": 1, "src/b": 1},
		},
		PackageEdges: map[string]map[string]int{},
		FileEdges:    map[string]map[string]struct{}{},
	}
	components := miner.DetectCycleComponents(g)
	if len(components) != 1 {
		t.Fatalf("expected one cycle component, got %+v", components)
	}
	if len(components[0].Nodes) != 3 {
		t.Fatalf("expected 3 nodes in component, got %+v", components[0])
	}
}

func TestProposeAppliesMinSupportToFilePattern(t *testing.T) {
	files := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		files = append(files, fmt.Sprintf("src/feature/f%d.ts", i))
	}
	g := graph.Build(nil, files)

	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    10,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindFilePattern {
			t.Fatalf("expected no file_pattern candidate below min-support, got %+v", c)
		}
	}
}

func TestProposeAppliesMinSupportToNoCycle(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]int{"src/a": 3, "src/b": 3, "src/c": 3},
		Edges: map[string]map[string]int{
			"src/a": {"src/b": 1},
			"src/b": {"src/c": 1},
			"src/c": {"src/a": 1},
		},
		PackageEdges: map[string]map[string]int{},
		FileEdges:    map[string]map[string]struct{}{},
	}
	candidates := miner.Propose(g, nil, miner.Options{
		MinSupport:    10,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoCycle {
			t.Fatalf("expected no no_cycle candidate below min-support, got %+v", c)
		}
	}
}

func TestProposeNoCycleUsesComponentCandidates(t *testing.T) {
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
	candidates := miner.Propose(g, nil, miner.Options{
		MinSupport:    1,
		MaxPrevalence: 1,
	})
	found := make([]miner.Candidate, 0)
	for _, c := range candidates {
		if c.Kind == config.KindNoCycle {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly one no_cycle candidate per SCC, got %+v", found)
	}
	if found[0].Support != 15 {
		t.Fatalf("expected component support sum 15, got %+v", found[0])
	}
	if found[0].Violations != 3 {
		t.Fatalf("expected violations to equal component size, got %+v", found[0])
	}
}

func TestProposeCapsCandidatesPerKind(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	pkgs := []string{"axios", "lodash", "zod"}
	dirs := []string{"src/a", "src/b", "src/c"}
	for _, dir := range dirs {
		for i := 0; i < 25; i++ {
			file := fmt.Sprintf("%s/f%d.ts", dir, i)
			files = append(files, file)
			imports = append(imports, model.ImportRef{
				SourceFile:      file,
				RawImport:       pkgs[i%len(pkgs)],
				IsPackageImport: true,
			})
		}
	}
	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:           1,
		MaxPrevalence:        1,
		MaxCandidatesPerKind: 2,
	})

	counts := map[string]int{}
	for _, c := range candidates {
		counts[c.Kind]++
	}
	for kind, count := range counts {
		if count > 2 {
			t.Fatalf("expected at most 2 candidates for kind %s, got %d", kind, count)
		}
	}
	if counts[config.KindNoImport] != 2 {
		t.Fatalf("expected capped no_import candidates to equal 2, got %d", counts[config.KindNoImport])
	}
	if counts[config.KindNoPackage] != 2 {
		t.Fatalf("expected capped no_package candidates to equal 2, got %d", counts[config.KindNoPackage])
	}
}
