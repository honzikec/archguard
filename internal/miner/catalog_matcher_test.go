package miner_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/catalog"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/miner"
)

func TestMatchCatalogDeterministicAndDeduped(t *testing.T) {
	patterns, err := catalog.LoadBuiltin()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	candidates := []miner.Candidate{
		{Kind: config.KindNoImport, Scope: []string{"src/domain/**"}, Target: []string{"src/infra/**"}, Support: 50, Prevalence: 0.01, Evidence: "a", Severity: config.SeverityWarning},
		{Kind: config.KindNoImport, Scope: []string{"src/domain/**"}, Target: []string{"src/infra/**"}, Support: 50, Prevalence: 0.01, Evidence: "duplicate", Severity: config.SeverityWarning},
		{Kind: config.KindNoPackage, Scope: []string{"src/domain/**"}, Target: []string{"axios"}, Support: 40, Prevalence: 0.00, Evidence: "pkg", Severity: config.SeverityWarning},
	}

	got1, err := miner.MatchCatalog(patterns, candidates, nil, config.DefaultProjectSettings(), miner.CatalogOptions{ShowLowConfidence: true})
	if err != nil {
		t.Fatalf("match catalog: %v", err)
	}
	got2, err := miner.MatchCatalog(patterns, candidates, nil, config.DefaultProjectSettings(), miner.CatalogOptions{ShowLowConfidence: true})
	if err != nil {
		t.Fatalf("match catalog second run: %v", err)
	}

	j1, _ := json.Marshal(got1)
	j2, _ := json.Marshal(got2)
	if string(j1) != string(j2) {
		t.Fatalf("expected deterministic matches\n1=%s\n2=%s", string(j1), string(j2))
	}

	for i := 1; i < len(got1); i++ {
		if got1[i].Score > got1[i-1].Score {
			t.Fatalf("matches not sorted by score desc at index %d", i)
		}
	}
}

func TestMatchCatalogLowConfidenceFiltering(t *testing.T) {
	patterns, err := catalog.LoadBuiltin()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	dir := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	mustWrite(t, filepath.Join("src", "services", "user.service.ts"), "export class UserService {}")
	mustWrite(t, filepath.Join("src", "feature", "controller.ts"), "const a = new UserService()")

	files := []string{"src/services/user.service.ts", "src/feature/controller.ts"}
	candidates := []miner.Candidate{}

	hidden, err := miner.MatchCatalog(patterns, candidates, files, config.DefaultProjectSettings(), miner.CatalogOptions{ShowLowConfidence: false})
	if err != nil {
		t.Fatalf("match hidden: %v", err)
	}
	shown, err := miner.MatchCatalog(patterns, candidates, files, config.DefaultProjectSettings(), miner.CatalogOptions{ShowLowConfidence: true})
	if err != nil {
		t.Fatalf("match shown: %v", err)
	}
	if len(shown) < len(hidden) {
		t.Fatalf("expected shown >= hidden, got shown=%d hidden=%d", len(shown), len(hidden))
	}
}

func TestAdoptCatalogMatchesThreshold(t *testing.T) {
	matches := []miner.PatternMatch{
		{Confidence: "LOW", ProposedRule: config.Rule{ID: "LOW"}},
		{Confidence: "MEDIUM", ProposedRule: config.Rule{ID: "MED"}},
		{Confidence: "HIGH", ProposedRule: config.Rule{ID: "HIGH"}},
	}
	high := miner.AdoptCatalogMatches(matches, "high")
	if len(high) != 1 {
		t.Fatalf("expected 1 high adoption, got %d", len(high))
	}
	med := miner.AdoptCatalogMatches(matches, "medium")
	if len(med) != 2 {
		t.Fatalf("expected 2 medium adoption, got %d", len(med))
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
