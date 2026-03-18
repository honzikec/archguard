package catalog

import (
	"fmt"
	"strings"
)

func validatePattern(p Pattern) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(p.Category) == "" {
		return fmt.Errorf("category is required")
	}
	if strings.TrimSpace(p.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if len(p.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}
	for i, s := range p.Sources {
		if strings.TrimSpace(s.Title) == "" {
			return fmt.Errorf("sources[%d].title is required", i)
		}
		if strings.TrimSpace(s.URL) == "" {
			return fmt.Errorf("sources[%d].url is required", i)
		}
		if strings.TrimSpace(s.License) == "" {
			return fmt.Errorf("sources[%d].license is required", i)
		}
	}
	if len(p.Detection.RequiredFacts) == 0 {
		return fmt.Errorf("detection.required_facts is required")
	}
	if strings.TrimSpace(p.Detection.Heuristic.Type) == "" {
		return fmt.Errorf("detection.heuristic.type is required")
	}
	if len(p.Detection.Heuristic.Params) == 0 {
		return fmt.Errorf("detection.heuristic.params is required")
	}
	if strings.TrimSpace(p.RuleTemplate.Kind) == "" {
		return fmt.Errorf("rule_template.kind is required")
	}
	if strings.TrimSpace(p.RuleTemplate.Template) == "" {
		return fmt.Errorf("rule_template.template is required")
	}
	if strings.TrimSpace(p.RuleTemplate.Defaults.Severity) == "" {
		return fmt.Errorf("rule_template.defaults.severity is required")
	}
	if p.RuleTemplate.Defaults.Severity != "error" && p.RuleTemplate.Defaults.Severity != "warning" {
		return fmt.Errorf("rule_template.defaults.severity must be error|warning")
	}

	switch p.Detection.Heuristic.Type {
	case "prevalence_boundary", "prevalence_package_boundary", "construction_new_outside_root":
		// supported
	default:
		return fmt.Errorf("unsupported heuristic type %q", p.Detection.Heuristic.Type)
	}

	return nil
}
