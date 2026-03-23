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
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/domain/f%d.ts", i),
			ResolvedPath:    "src/domain/shared.ts",
			IsPackageImport: false,
		})
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

func TestProposeSuppressesTrivialPHPFilePatternCandidates(t *testing.T) {
	files := make([]string, 0, 32)
	for i := 0; i < 16; i++ {
		files = append(files, fmt.Sprintf("backend/controllers/C%d.php", i))
		files = append(files, fmt.Sprintf("common/models/M%d.php", i))
	}
	g := graph.Build(nil, files)

	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    8,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindFilePattern && len(c.Target) > 0 && c.Target[0] == "^.*\\.php$" {
			t.Fatalf("expected trivial php extension file_pattern candidate to be suppressed, got %+v", c)
		}
	}
}

func TestProposeKeepsNonTrivialPHPFilePatternCandidates(t *testing.T) {
	files := make([]string, 0, 25)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("backend/controllers/C%d.php", i))
	}
	for i := 0; i < 5; i++ {
		files = append(files, fmt.Sprintf("common/services/S%d.service.php", i))
	}
	g := graph.Build(nil, files)

	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    5,
		MaxPrevalence: 1,
	})
	found := false
	for _, c := range candidates {
		if c.Kind == config.KindFilePattern && len(c.Scope) > 0 && c.Scope[0] == "common/services/**" && len(c.Target) > 0 && c.Target[0] == "^.*\\.service\\.php$" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected non-trivial php suffix file_pattern candidate to remain")
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

func TestProposeNoImportSkipsNeverObservedTargets(t *testing.T) {
	files := make([]string, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/domain/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/infra/f%d.ts", i))
	}
	g := graph.Build(nil, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoImport {
			t.Fatalf("expected no no_import candidates when no imports are observed, got %+v", c)
		}
	}
}

func TestProposeNoPackageSkipsZeroViolationCandidatesForInactiveSources(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/inactive/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/infra/f%d.ts", i))
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/infra/f%d.ts", i),
			RawImport:       "axios",
			IsPackageImport: true,
		})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoPackage && len(c.Scope) > 0 && c.Scope[0] == "src/inactive/**" {
			t.Fatalf("expected inactive source scope to not emit zero-violation no_package candidate, got %+v", c)
		}
	}
}

func TestProposeNoImportSkipsZeroViolationCandidatesForInactiveSources(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/inactive/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/common/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/active/f%d.ts", i))
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/active/f%d.ts", i),
			ResolvedPath:    fmt.Sprintf("src/common/f%d.ts", i),
			IsPackageImport: false,
		})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoImport && len(c.Scope) > 0 && c.Scope[0] == "src/inactive/**" {
			t.Fatalf("expected inactive source scope to not emit zero-violation no_import candidate, got %+v", c)
		}
	}
}

func TestProposeNoImportSkipsZeroViolationCandidatesForTestLikeScopes(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/feature/test/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/common/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/active/f%d.ts", i))
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/feature/test/f%d.ts", i),
			ResolvedPath:    "src/feature/test/shared.ts",
			IsPackageImport: false,
		})
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/active/f%d.ts", i),
			ResolvedPath:    fmt.Sprintf("src/common/f%d.ts", i),
			IsPackageImport: false,
		})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoImport && len(c.Scope) > 0 && c.Scope[0] == "src/feature/test/**" {
			t.Fatalf("expected test-like source scope to not emit zero-violation no_import candidate, got %+v", c)
		}
	}
}

func TestProposeNoPackageSkipsZeroViolationCandidatesForTestLikeScopes(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/feature/test/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/infra/f%d.ts", i))
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/feature/test/f%d.ts", i),
			ResolvedPath:    "src/feature/test/shared.ts",
			IsPackageImport: false,
		})
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/infra/f%d.ts", i),
			RawImport:       "axios",
			IsPackageImport: true,
		})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoPackage && len(c.Scope) > 0 && c.Scope[0] == "src/feature/test/**" {
			t.Fatalf("expected test-like source scope to not emit zero-violation no_package candidate, got %+v", c)
		}
	}
}

func TestProposeNoPackageRequiresGlobalUsageForZeroViolationCandidates(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/domain/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/infra/f%d.ts", i))
	}
	// Package usage is below default zero-violation evidence threshold (minSupport/2 = 10).
	for i := 0; i < 5; i++ {
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/infra/f%d.ts", i),
			RawImport:       "axios",
			IsPackageImport: true,
		})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoPackage && len(c.Scope) > 0 && c.Scope[0] == "src/domain/**" && len(c.Target) > 0 && c.Target[0] == "axios" {
			t.Fatalf("expected no no_package candidate for low-global-usage package, got %+v", c)
		}
	}
}

func TestProposeNoPackageSkipsZeroViolationClassLikeTargets(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/domain/f%d.php", i))
		files = append(files, fmt.Sprintf("src/infra/f%d.php", i))
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/infra/f%d.php", i),
			RawImport:       "Aws",
			IsPackageImport: true,
		})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoPackage && len(c.Scope) > 0 && c.Scope[0] == "src/domain/**" && len(c.Target) > 0 && c.Target[0] == "Aws" {
			t.Fatalf("expected zero-violation class-like no_package target to be skipped, got %+v", c)
		}
	}
}

func TestProposeNoPackageSkipsHighlySharedZeroViolationTargets(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	// 8 source subtrees, package used in 7 of them => spread too broad for zero-violation candidate.
	for s := 0; s < 8; s++ {
		dir := fmt.Sprintf("src/s%d", s)
		for i := 0; i < 20; i++ {
			file := fmt.Sprintf("%s/f%d.ts", dir, i)
			files = append(files, file)
			if s < 7 {
				imports = append(imports, model.ImportRef{
					SourceFile:      file,
					RawImport:       "rxjs",
					IsPackageImport: true,
				})
			}
		}
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoPackage && len(c.Scope) > 0 && c.Scope[0] == "src/s7/**" && len(c.Target) > 0 && c.Target[0] == "rxjs" {
			t.Fatalf("expected highly shared zero-violation no_package candidate to be skipped, got %+v", c)
		}
	}
}

func TestProposeNoImportSkipsHighlySharedZeroViolationTargets(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/common/f%d.ts", i))
	}
	// 8 source subtrees, 7 import common, 1 does not.
	for s := 0; s < 8; s++ {
		dir := fmt.Sprintf("src/s%d", s)
		for i := 0; i < 20; i++ {
			file := fmt.Sprintf("%s/f%d.ts", dir, i)
			files = append(files, file)
			if s < 7 {
				imports = append(imports, model.ImportRef{
					SourceFile:      file,
					ResolvedPath:    fmt.Sprintf("src/common/f%d.ts", i),
					IsPackageImport: false,
				})
			}
		}
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})
	for _, c := range candidates {
		if c.Kind == config.KindNoImport && len(c.Scope) > 0 && c.Scope[0] == "src/s7/**" && len(c.Target) > 0 && c.Target[0] == "src/common/**" {
			t.Fatalf("expected highly shared zero-violation no_import candidate to be skipped, got %+v", c)
		}
	}
}

func TestProposeNoImportCapsZeroViolationPerScope(t *testing.T) {
	nodes := map[string]int{
		"src/domain": 20,
	}
	edges := map[string]map[string]int{}
	fileEdges := map[string]map[string]struct{}{
		"src/domain/f0.ts": {
			"src/domain/f1.ts": {},
		},
	}
	for i := 0; i < 10; i++ {
		target := fmt.Sprintf("src/t%d", i)
		source := fmt.Sprintf("src/s%d", i)
		nodes[target] = 20
		nodes[source] = 20
		edges[source] = map[string]int{target: 20}
	}

	g := &graph.Graph{
		Nodes:        nodes,
		Edges:        edges,
		PackageEdges: map[string]map[string]int{},
		FileEdges:    fileEdges,
	}

	candidates := miner.Propose(g, nil, miner.Options{
		MinSupport:           20,
		MaxPrevalence:        1,
		MaxCandidatesPerKind: 200,
	})

	count := 0
	for _, c := range candidates {
		if c.Kind == config.KindNoImport && len(c.Scope) > 0 && c.Scope[0] == "src/domain/**" && c.Violations == 0 {
			count++
		}
	}
	if count != 5 {
		t.Fatalf("expected zero-violation no_import candidates for src/domain to be capped at 5, got %d", count)
	}
}

func TestProposeNoPackageCapsZeroViolationPerScope(t *testing.T) {
	nodes := map[string]int{
		"domain": 20,
	}
	packageEdges := map[string]map[string]int{}
	fileEdges := map[string]map[string]struct{}{
		"domain/f0.ts": {
			"domain/f1.ts": {},
		},
	}
	for i := 0; i < 10; i++ {
		source := fmt.Sprintf("s%d", i)
		nodes[source] = 20
		packageEdges[source] = map[string]int{
			fmt.Sprintf("pkg%d", i): 20,
		}
	}

	g := &graph.Graph{
		Nodes:        nodes,
		Edges:        map[string]map[string]int{},
		PackageEdges: packageEdges,
		FileEdges:    fileEdges,
	}

	candidates := miner.Propose(g, nil, miner.Options{
		MinSupport:           20,
		MaxPrevalence:        1,
		MaxCandidatesPerKind: 200,
	})

	count := 0
	for _, c := range candidates {
		if c.Kind == config.KindNoPackage && len(c.Scope) > 0 && c.Scope[0] == "domain/**" && c.Violations == 0 {
			count++
		}
	}
	if count != 5 {
		t.Fatalf("expected zero-violation no_package candidates for domain to be capped at 5, got %d", count)
	}
}

func TestProposeNoImportCapsZeroViolationPerTarget(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)

	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("target/f%d.ts", i))
	}
	for i := 0; i < 20; i++ {
		source := fmt.Sprintf("importer/f%d.ts", i)
		files = append(files, source)
		imports = append(imports, model.ImportRef{
			SourceFile:      source,
			ResolvedPath:    fmt.Sprintf("target/f%d.ts", i),
			IsPackageImport: false,
		})
	}
	for s := 0; s < 15; s++ {
		for i := 0; i < 20; i++ {
			file := fmt.Sprintf("s%d/f%d.ts", s, i)
			files = append(files, file)
			imports = append(imports, model.ImportRef{
				SourceFile:      file,
				RawImport:       "left-pad",
				IsPackageImport: true,
			})
		}
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})

	count := 0
	for _, c := range candidates {
		if c.Kind != config.KindNoImport || c.Violations != 0 || len(c.Target) == 0 {
			continue
		}
		if c.Target[0] == "target/**" {
			count++
		}
	}
	if count != 10 {
		t.Fatalf("expected zero-violation no_import candidates for target/** to be capped at 10, got %d", count)
	}
}

func TestProposeNoPackageCapsZeroViolationPerTarget(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)

	for i := 0; i < 20; i++ {
		source := fmt.Sprintf("importer/f%d.ts", i)
		files = append(files, source)
		imports = append(imports, model.ImportRef{
			SourceFile:      source,
			RawImport:       "rxjs",
			IsPackageImport: true,
		})
	}
	for s := 0; s < 15; s++ {
		for i := 0; i < 20; i++ {
			file := fmt.Sprintf("s%d/f%d.ts", s, i)
			files = append(files, file)
			imports = append(imports, model.ImportRef{
				SourceFile:      file,
				ResolvedPath:    fmt.Sprintf("s%d/local.ts", s),
				IsPackageImport: false,
			})
		}
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:    20,
		MaxPrevalence: 1,
	})

	count := 0
	for _, c := range candidates {
		if c.Kind != config.KindNoPackage || c.Violations != 0 || len(c.Target) == 0 {
			continue
		}
		if c.Target[0] == "rxjs" {
			count++
		}
	}
	if count != 10 {
		t.Fatalf("expected zero-violation no_package candidates for rxjs to be capped at 10, got %d", count)
	}
}

func TestProposeCapsCandidatesPerKindPrefersViolatingCandidates(t *testing.T) {
	files := []string{
		"src/a/f0.ts",
		"src/b/f0.ts",
		"src/b/helper.ts",
		"src/c/f0.ts",
		"src/d/f0.ts",
	}
	imports := []model.ImportRef{
		{
			SourceFile:      "src/a/f0.ts",
			RawImport:       "axios",
			IsPackageImport: true,
		},
		{
			SourceFile:      "src/b/f0.ts",
			ResolvedPath:    "src/b/helper.ts",
			IsPackageImport: false,
		},
		{
			SourceFile:      "src/c/f0.ts",
			ResolvedPath:    "src/d/f0.ts",
			IsPackageImport: false,
		},
		{
			SourceFile:      "src/d/f0.ts",
			ResolvedPath:    "src/d/f0.ts",
			IsPackageImport: false,
		},
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:           1,
		MaxPrevalence:        1,
		MaxCandidatesPerKind: 1,
	})

	var noImport, noPackage *miner.Candidate
	for i := range candidates {
		c := &candidates[i]
		if c.Kind == config.KindNoImport {
			noImport = c
		}
		if c.Kind == config.KindNoPackage {
			noPackage = c
		}
	}
	if noImport == nil {
		t.Fatalf("expected capped no_import candidate")
	}
	if noImport.Violations == 0 {
		t.Fatalf("expected capped no_import candidate to prefer violations>0, got %+v", *noImport)
	}
	if noPackage == nil {
		t.Fatalf("expected capped no_package candidate")
	}
	if noPackage.Violations == 0 {
		t.Fatalf("expected capped no_package candidate to prefer violations>0, got %+v", *noPackage)
	}
}

func TestProposeAggregatesSiblingNoPackageScopes(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)

	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/app/a/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/app/b/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/importer/f%d.ts", i))
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/app/a/f%d.ts", i),
			RawImport:       "left-pad",
			IsPackageImport: true,
		})
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/app/b/f%d.ts", i),
			RawImport:       "left-pad",
			IsPackageImport: true,
		})
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/importer/f%d.ts", i),
			RawImport:       "axios",
			IsPackageImport: true,
		})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:           20,
		MaxPrevalence:        1,
		MaxCandidatesPerKind: 200,
	})

	foundParent := false
	foundChildA := false
	foundChildB := false
	for _, c := range candidates {
		if c.Kind != config.KindNoPackage || len(c.Scope) == 0 || len(c.Target) == 0 {
			continue
		}
		if c.Target[0] != "axios" {
			continue
		}
		switch c.Scope[0] {
		case "src/app/**":
			foundParent = true
		case "src/app/a/**":
			foundChildA = true
		case "src/app/b/**":
			foundChildB = true
		}
	}
	if !foundParent {
		t.Fatalf("expected aggregated parent no_package scope src/app/** for target axios")
	}
	if foundChildA || foundChildB {
		t.Fatalf("expected child scopes src/app/a/** and src/app/b/** to be replaced after aggregation")
	}
}

func TestProposeAggregatesSiblingNoImportScopes(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)

	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/app/a/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/app/b/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/consumer/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/infra/f%d.ts", i))
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/app/a/f%d.ts", i),
			ResolvedPath:    fmt.Sprintf("src/app/a/local%d.ts", i),
			IsPackageImport: false,
		})
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/app/b/f%d.ts", i),
			ResolvedPath:    fmt.Sprintf("src/app/b/local%d.ts", i),
			IsPackageImport: false,
		})
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/consumer/f%d.ts", i),
			ResolvedPath:    fmt.Sprintf("src/infra/f%d.ts", i),
			IsPackageImport: false,
		})
	}

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{
		MinSupport:           20,
		MaxPrevalence:        1,
		MaxCandidatesPerKind: 200,
	})

	foundParent := false
	foundChildA := false
	foundChildB := false
	for _, c := range candidates {
		if c.Kind != config.KindNoImport || len(c.Scope) == 0 || len(c.Target) == 0 {
			continue
		}
		if c.Target[0] != "src/infra/**" {
			continue
		}
		switch c.Scope[0] {
		case "src/app/**":
			foundParent = true
		case "src/app/a/**":
			foundChildA = true
		case "src/app/b/**":
			foundChildB = true
		}
	}
	if !foundParent {
		t.Fatalf("expected aggregated parent no_import scope src/app/** for target src/infra/**")
	}
	if foundChildA || foundChildB {
		t.Fatalf("expected child scopes src/app/a/** and src/app/b/** to be replaced after aggregation")
	}
}

func TestProposeCollectsDebugDropCounters(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	for i := 0; i < 20; i++ {
		files = append(files, fmt.Sprintf("src/a/f%d.ts", i))
		files = append(files, fmt.Sprintf("src/b/f%d.ts", i))
		imports = append(imports, model.ImportRef{
			SourceFile:      fmt.Sprintf("src/a/f%d.ts", i),
			ResolvedPath:    fmt.Sprintf("src/a/local%d.ts", i),
			IsPackageImport: false,
		})
	}

	stats := miner.NewDebugStats()
	_ = miner.Propose(graph.Build(imports, files), files, miner.Options{
		MinSupport:           20,
		MaxPrevalence:        1,
		MaxCandidatesPerKind: 200,
		DebugStats:           stats,
	})

	if len(stats.Dropped) == 0 {
		t.Fatalf("expected debug drop counters to be populated")
	}
	if stats.Dropped["no_import:target_never_observed"] == 0 {
		t.Fatalf("expected no_import:target_never_observed drop counter to be > 0, got %+v", stats.Dropped)
	}
}

func TestProposeCapsCandidatesPerKind(t *testing.T) {
	files := make([]string, 0)
	imports := make([]model.ImportRef, 0)
	pkgs := []string{"axios", "lodash", "zod"}
	dirs := []string{"src/a", "src/b", "src/c"}
	for dirIdx, dir := range dirs {
		for i := 0; i < 25; i++ {
			file := fmt.Sprintf("%s/f%d.ts", dir, i)
			files = append(files, file)
			imports = append(imports, model.ImportRef{
				SourceFile:      file,
				RawImport:       pkgs[i%len(pkgs)],
				IsPackageImport: true,
			})
			targetDir := dirs[(dirIdx+1)%len(dirs)]
			imports = append(imports, model.ImportRef{
				SourceFile:      file,
				ResolvedPath:    fmt.Sprintf("%s/f%d.ts", targetDir, i),
				IsPackageImport: false,
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
