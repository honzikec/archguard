package miner_test

import (
	"strings"
	"testing"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/miner"
)

func TestEmitStarterConfigDefaultsNoCycleToWarning(t *testing.T) {
	candidates := []miner.Candidate{
		{
			Kind:     config.KindNoCycle,
			Scope:    []string{"src/**"},
			Severity: config.SeverityError,
			Evidence: "cycle",
		},
	}
	got := miner.EmitStarterConfigWithCatalog(candidates, nil, miner.EmitOptions{})
	if !strings.Contains(got, "kind: no_cycle\n    severity: warning") {
		t.Fatalf("expected default emitted no_cycle severity warning, got:\n%s", got)
	}
}

func TestEmitStarterConfigAllowsNoCycleErrorOverride(t *testing.T) {
	candidates := []miner.Candidate{
		{
			Kind:     config.KindNoCycle,
			Scope:    []string{"src/**"},
			Severity: config.SeverityError,
			Evidence: "cycle",
		},
	}
	got := miner.EmitStarterConfigWithCatalog(candidates, nil, miner.EmitOptions{
		NoCycleSeverity: config.SeverityError,
	})
	if !strings.Contains(got, "kind: no_cycle\n    severity: error") {
		t.Fatalf("expected emitted no_cycle severity error override, got:\n%s", got)
	}
}
