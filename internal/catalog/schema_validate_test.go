package catalog

import "testing"

func TestValidateSchemaDocumentAcceptsValidPattern(t *testing.T) {
	raw := map[string]any{
		"id":          "CAT-OK",
		"name":        "ok",
		"category":    "layering",
		"description": "d",
		"sources": []any{
			map[string]any{
				"title":   "Clean Architecture",
				"url":     "https://example.com",
				"license": "Reference",
			},
		},
		"detection": map[string]any{
			"required_facts": []any{"imports"},
			"heuristic": map[string]any{
				"type": "prevalence_boundary",
				"params": map[string]any{
					"source_globs":   []any{"src/domain/**"},
					"target_globs":   []any{"src/infra/**"},
					"max_prevalence": 0.1,
					"min_support":    3,
				},
			},
		},
		"rule_template": map[string]any{
			"kind":     "pattern",
			"template": "dependency_constraint",
			"defaults": map[string]any{
				"severity": "warning",
				"params": map[string]any{
					"relation": "imports",
				},
			},
		},
	}

	if err := validateSchemaDocument(raw); err != nil {
		t.Fatalf("expected schema validation to pass: %v", err)
	}
}

func TestValidateSchemaDocumentRejectsUnknownTopLevelKey(t *testing.T) {
	raw := map[string]any{
		"id":          "CAT-OK",
		"name":        "ok",
		"category":    "layering",
		"description": "d",
		"sources": []any{
			map[string]any{
				"title":   "Clean Architecture",
				"url":     "https://example.com",
				"license": "Reference",
			},
		},
		"detection": map[string]any{
			"required_facts": []any{"imports"},
			"heuristic": map[string]any{
				"type": "prevalence_boundary",
				"params": map[string]any{
					"source_globs":   []any{"src/domain/**"},
					"target_globs":   []any{"src/infra/**"},
					"max_prevalence": 0.1,
					"min_support":    3,
				},
			},
		},
		"rule_template": map[string]any{
			"kind":     "pattern",
			"template": "dependency_constraint",
			"defaults": map[string]any{
				"severity": "warning",
			},
		},
		"extra": "boom",
	}

	if err := validateSchemaDocument(raw); err == nil {
		t.Fatal("expected schema validation error for unknown top-level key")
	}
}

func TestValidateSchemaDocumentRejectsUnknownHeuristicParam(t *testing.T) {
	raw := map[string]any{
		"id":          "CAT-OK",
		"name":        "ok",
		"category":    "layering",
		"description": "d",
		"sources": []any{
			map[string]any{
				"title":   "Clean Architecture",
				"url":     "https://example.com",
				"license": "Reference",
			},
		},
		"detection": map[string]any{
			"required_facts": []any{"imports"},
			"heuristic": map[string]any{
				"type": "prevalence_boundary",
				"params": map[string]any{
					"source_globs":   []any{"src/domain/**"},
					"target_globs":   []any{"src/infra/**"},
					"max_prevalence": 0.1,
					"min_support":    3,
					"unknown":        true,
				},
			},
		},
		"rule_template": map[string]any{
			"kind":     "pattern",
			"template": "dependency_constraint",
			"defaults": map[string]any{
				"severity": "warning",
			},
		},
	}

	if err := validateSchemaDocument(raw); err == nil {
		t.Fatal("expected schema validation error for unknown heuristic param")
	}
}
