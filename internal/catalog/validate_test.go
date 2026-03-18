package catalog

import "testing"

func TestValidatePatternRejectsMissingSource(t *testing.T) {
	p := Pattern{
		ID:          "CAT-X",
		Name:        "x",
		Category:    "layering",
		Description: "d",
		Detection: Detection{
			RequiredFacts: []string{"imports"},
			Heuristic:     Heuristic{Type: "prevalence_boundary", Params: map[string]any{"source_globs": []string{"src/**"}}},
		},
		RuleTemplate: RuleTemplate{Kind: "pattern", Template: "dependency_constraint", Defaults: RuleDefault{Severity: "warning"}},
	}
	if err := validatePattern(p); err == nil {
		t.Fatal("expected validation error for missing sources")
	}
}
