package miner_test

import (
	"testing"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
)

func TestMinerProposeBannedPackage(t *testing.T) {
	// Simulate 20 files in src/domain, none of them importing "axios"
	var allFiles []string
	var imports []model.ImportRef

	for i := 0; i < 20; i++ {
		allFiles = append(allFiles, "src/domain/file"+string(rune(i))+".ts")
	}

	for i := 0; i < 20; i++ {
		allFiles = append(allFiles, "src/infra/file"+string(rune(i))+".ts")
		// The infra layer imports axios 20 times package dependency
		imports = append(imports, model.ImportRef{
			SourceFile:      "src/infra/file" + string(rune(i)) + ".ts",
			RawImport:       "axios",
			IsPackageImport: true,
		})
	}

	g := graph.Build(imports, allFiles)
	candidates := miner.Propose(g)

	foundAxiosBan := false
	for _, c := range candidates {
		if c.Kind == "banned_package" && c.FromPaths == "src/domain/**" && c.ForbiddenPaths == "axios" {
			foundAxiosBan = true
			if c.Confidence != "MEDIUM" {
				t.Errorf("expected MEDIUM confidence, got %s", c.Confidence)
			}
		}
	}

	if !foundAxiosBan {
		t.Errorf("expected miner to propose banning 'axios' from 'src/domain/**'")
	}
}
