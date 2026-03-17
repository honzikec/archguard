package miner_test

import (
	"testing"

	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
)

func TestMinerProposeBannedPackage(t *testing.T) {
	var allFiles []string
	var imports []model.ImportRef

	for i := 0; i < 20; i++ {
		allFiles = append(allFiles, "src/domain/file"+string(rune('a'+i))+".ts")
	}
	for i := 0; i < 20; i++ {
		allFiles = append(allFiles, "src/infra/file"+string(rune('a'+i))+".ts")
		imports = append(imports, model.ImportRef{
			SourceFile:      "src/infra/file" + string(rune('a'+i)) + ".ts",
			RawImport:       "axios",
			IsPackageImport: true,
		})
	}

	g := graph.Build(imports, allFiles)
	candidates := miner.Propose(g)

	found := false
	for _, c := range candidates {
		if c.Kind == "banned_package" && c.FromPaths == "src/domain/**" && c.ForbiddenPaths == "axios" {
			found = true
			if c.Confidence != "MEDIUM" {
				t.Errorf("expected MEDIUM confidence, got %s", c.Confidence)
			}
		}
	}
	if !found {
		t.Errorf("expected miner to propose banning 'axios' from 'src/domain/**'")
	}
}

func TestProposeFileConventions(t *testing.T) {
	files := []string{
		"src/services/user.service.ts",
		"src/services/cart.service.ts",
		"src/services/order.service.ts",
		"src/services/payment.service.ts",
		"src/services/auth.service.ts",
	}

	candidates := miner.ProposeFileConventions(files)

	if len(candidates) == 0 {
		t.Fatal("expected at least one file_convention candidate")
	}
	c := candidates[0]
	if c.Kind != "file_convention" {
		t.Errorf("expected kind file_convention, got %s", c.Kind)
	}
	if c.FromPaths != "src/services/**" {
		t.Errorf("unexpected from_paths: %s", c.FromPaths)
	}
	// regex should encode .service.ts
	if c.ForbiddenPaths == "" {
		t.Error("expected a regex in ForbiddenPaths")
	}
	if c.Confidence != "HIGH" {
		t.Errorf("expected HIGH confidence, got %s", c.Confidence)
	}
}

func TestDetectCycles(t *testing.T) {
	// Build a synthetic graph with a cycle: A -> B -> C -> A
	g := &graph.Graph{
		Nodes: map[string]int{
			"src/a": 5,
			"src/b": 5,
			"src/c": 5,
		},
		Edges: map[string]map[string]int{
			"src/a": {"src/b": 1},
			"src/b": {"src/c": 1},
			"src/c": {"src/a": 1},
		},
		PackageEdges: map[string]map[string]int{},
	}

	cycles := miner.DetectCycles(g)
	if len(cycles) == 0 {
		t.Fatal("expected to detect a cycle, got none")
	}
}

func TestProposeCrossAppRules(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]int{
			"apps/frontend/src": 30,
			"apps/storybook/stories": 20,
		},
		Edges: map[string]map[string]int{
			"apps/storybook/stories": {"apps/frontend/src": 5},
		},
		PackageEdges: map[string]map[string]int{},
	}

	candidates := miner.ProposeCrossAppRules(g)
	if len(candidates) == 0 {
		t.Fatal("expected cross-app rule candidate, got none")
	}
	c := candidates[0]
	if c.Kind != "import_boundary" {
		t.Errorf("expected kind import_boundary, got %s", c.Kind)
	}
	if c.Confidence != "HIGH" {
		t.Errorf("expected HIGH confidence for cross-app rule, got %s", c.Confidence)
	}
}
