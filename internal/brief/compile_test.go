package brief

import "testing"

func TestCompileResolvesLayersAndBuildsRules(t *testing.T) {
	spec := &Brief{
		Version: 1,
		Layers: []Layer{
			{ID: "domain", Paths: []string{"src/domain/**"}},
			{ID: "infra", Paths: []string{"src/infra/**"}},
			{ID: "services", Paths: []string{"src/services/**"}},
			{ID: "bootstrap", Paths: []string{"src/bootstrap/**"}},
		},
		Policies: []PolicyIntent{
			{
				Type:     "deny_import",
				From:     []string{"layer:domain"},
				To:       []string{"layer:infra"},
				Severity: "error",
			},
			{
				Type:     "deny_package",
				Scope:    []string{"layer:domain"},
				Packages: []string{"axios"},
			},
			{
				Type:             "construction_policy",
				Scope:            []string{"src/**"},
				Services:         []string{"layer:services"},
				AllowIn:          []string{"layer:bootstrap"},
				ServiceNameRegex: ".*Service$",
			},
		},
	}

	cfg, err := Compile(spec)
	if err != nil {
		t.Fatalf("expected compile success, got: %v", err)
	}
	if len(cfg.Rules) != 3 {
		t.Fatalf("expected 3 compiled rules, got %d", len(cfg.Rules))
	}

	if cfg.Rules[0].Kind != "no_import" || cfg.Rules[0].Scope[0] != "src/domain/**" || cfg.Rules[0].Target[0] != "src/infra/**" {
		t.Fatalf("unexpected no_import rule: %+v", cfg.Rules[0])
	}
	if cfg.Rules[1].Kind != "no_package" || cfg.Rules[1].Target[0] != "axios" {
		t.Fatalf("unexpected no_package rule: %+v", cfg.Rules[1])
	}
	if cfg.Rules[2].Kind != "pattern" || cfg.Rules[2].Template != "construction_policy" {
		t.Fatalf("unexpected construction rule: %+v", cfg.Rules[2])
	}
	if len(cfg.Rules[2].Except) != 1 || cfg.Rules[2].Except[0] != "src/bootstrap/**" {
		t.Fatalf("expected allow_in to compile to except globs, got %+v", cfg.Rules[2].Except)
	}
	if cfg.Rules[2].Params["service_name_regex"] != ".*Service$" {
		t.Fatalf("expected service_name_regex param, got %+v", cfg.Rules[2].Params)
	}
}

func TestCompileRejectsUnknownLayerReference(t *testing.T) {
	spec := &Brief{
		Version: 1,
		Policies: []PolicyIntent{
			{
				Type: "deny_import",
				From: []string{"layer:missing"},
				To:   []string{"src/infra/**"},
			},
		},
	}

	_, err := Compile(spec)
	if err == nil {
		t.Fatal("expected compile error for unknown layer reference")
	}
}
