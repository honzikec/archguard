package miner

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/config"
)

type MineOutput struct {
	Candidates     []Candidate    `json:"candidates"`
	CatalogMatches []PatternMatch `json:"catalog_matches,omitempty"`
}

type EmitOptions struct {
	NoCycleSeverity string
	Project         *config.ProjectSettings
}

type MineNormalizationStats struct {
	OriginalNodes   int `json:"original_nodes"`
	NormalizedNodes int `json:"normalized_nodes"`
	OriginalFiles   int `json:"original_files"`
	NormalizedFiles int `json:"normalized_files"`
}

type MineMetadata struct {
	FrameworkProfile string                 `json:"framework_profile"`
	FrameworkReason  string                 `json:"framework_reason"`
	FrameworkMatched []string               `json:"framework_matched,omitempty"`
	LanguageAdapter  string                 `json:"language_adapter"`
	LanguageReason   string                 `json:"language_reason"`
	Normalization    MineNormalizationStats `json:"normalization"`
}

func PrintMineText(candidates []Candidate, catalogMatches []PatternMatch, catalogFormat string, debug bool, metadata MineMetadata) {
	PrintText(candidates)
	if debug {
		fmt.Println("---")
		fmt.Printf("framework_profile: %s\n", metadata.FrameworkProfile)
		fmt.Printf("framework_reason: %s\n", metadata.FrameworkReason)
		if len(metadata.FrameworkMatched) > 0 {
			fmt.Printf("framework_matched: %v\n", metadata.FrameworkMatched)
		}
		fmt.Printf("language_adapter: %s\n", metadata.LanguageAdapter)
		fmt.Printf("language_reason: %s\n", metadata.LanguageReason)
		fmt.Printf("normalization: original_nodes=%d normalized_nodes=%d original_files=%d normalized_files=%d\n",
			metadata.Normalization.OriginalNodes,
			metadata.Normalization.NormalizedNodes,
			metadata.Normalization.OriginalFiles,
			metadata.Normalization.NormalizedFiles)
	}
	if len(catalogMatches) == 0 {
		return
	}
	fmt.Println("---")
	fmt.Println("Catalog matches:")
	if catalogFormat == "json" {
		data, _ := json.MarshalIndent(catalogMatches, "", "  ")
		fmt.Println(string(data))
		return
	}
	for i, m := range catalogMatches {
		if i > 0 {
			fmt.Println("---")
		}
		fmt.Printf("catalog_id: %s\n", m.CatalogID)
		fmt.Printf("name: %s\n", m.Name)
		fmt.Printf("category: %s\n", m.Category)
		fmt.Printf("score: %.3f\n", m.Score)
		fmt.Printf("confidence: %s\n", m.Confidence)
		fmt.Printf("evidence: %s\n", m.Evidence)
		fmt.Printf("proposed_rule_id: %s\n", m.ProposedRule.ID)
		if !debug {
			continue
		}
		fmt.Printf("scoped_files: %d\n", m.ScopedFiles)
		fmt.Printf("eligible_files: %d\n", m.EligibleFiles)
		fmt.Printf("violating_files: %d\n", m.ViolatingFiles)
		fmt.Printf("support: %d\n", m.Support)
		fmt.Printf("prevalence: %.4f\n", m.Prevalence)
		fmt.Printf("score_components: structural_fit=%.3f prevalence_support=%.3f naming_fit=%.3f\n",
			m.ScoreComponents.StructuralFit, m.ScoreComponents.PrevalenceSupport, m.ScoreComponents.NamingFit)
		if m.ResolvedCount > 0 || m.UnresolvedCount > 0 {
			fmt.Printf("resolved_count: %d\n", m.ResolvedCount)
			fmt.Printf("unresolved_count: %d\n", m.UnresolvedCount)
		}
		if len(m.ResolvedExamples) > 0 {
			fmt.Printf("resolved_examples: %v\n", m.ResolvedExamples)
		}
		if len(m.SampleLocations) > 0 {
			fmt.Printf("sample_locations: %v\n", m.SampleLocations)
		}
		if len(m.UnresolvedReasons) > 0 {
			fmt.Printf("unresolved_reasons: %v\n", m.UnresolvedReasons)
		}
	}
}

func PrintMineJSON(candidates []Candidate, catalogMatches []PatternMatch, metadata MineMetadata) {
	_ = metadata
	payload := MineOutput{Candidates: candidates, CatalogMatches: catalogMatches}
	data, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(data))
}

func BuildStarterConfigWithCatalog(candidates []Candidate, adopted []config.Rule, opts EmitOptions) *config.Config {
	project := config.DefaultProjectSettings()
	if opts.Project != nil {
		project = *opts.Project
		if len(project.Roots) == 0 {
			project.Roots = config.DefaultProjectSettings().Roots
		}
		if len(project.Include) == 0 {
			project.Include = config.DefaultProjectSettings().Include
		}
		if len(project.Exclude) == 0 {
			project.Exclude = config.DefaultProjectSettings().Exclude
		}
		if project.Aliases == nil {
			project.Aliases = map[string][]string{}
		}
	}

	cfg := &config.Config{
		Schema:  "./schemas/archguard.v1.schema.json",
		Version: 1,
		Project: project,
		Rules:   make([]config.Rule, 0, len(candidates)+len(adopted)),
	}

	for i, c := range candidates {
		severity := c.Severity
		if c.Kind == config.KindNoCycle {
			override := strings.ToLower(strings.TrimSpace(opts.NoCycleSeverity))
			if override != config.SeverityError && override != config.SeverityWarning {
				override = config.SeverityWarning
			}
			severity = override
		}
		cfg.Rules = append(cfg.Rules, config.Rule{
			ID:       fmt.Sprintf("MINED-%03d", i+1),
			Kind:     c.Kind,
			Severity: severity,
			Scope:    append([]string{}, c.Scope...),
			Target:   append([]string{}, c.Target...),
			Message:  c.Evidence,
		})
	}

	sort.Slice(adopted, func(i, j int) bool {
		return adopted[i].ID < adopted[j].ID
	})
	for _, rule := range adopted {
		cfg.Rules = append(cfg.Rules, deepCopyRule(rule))
	}

	return cfg
}

func EmitStarterConfigWithCatalog(candidates []Candidate, adopted []config.Rule, opts EmitOptions) string {
	cfg := BuildStarterConfigWithCatalog(candidates, adopted, opts)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("$schema: %q\n", cfg.Schema))
	b.WriteString(fmt.Sprintf("version: %d\n", cfg.Version))
	b.WriteString("project:\n")
	b.WriteString(renderProjectBlock(cfg.Project))
	if len(cfg.Rules) == 0 {
		b.WriteString("rules: []\n")
		return b.String()
	}
	b.WriteString("rules:\n")
	for _, rule := range cfg.Rules {
		b.WriteString(renderRuleBlock(rule))
	}
	return b.String()
}

func renderProjectBlock(project config.ProjectSettings) string {
	var b strings.Builder
	b.WriteString(renderInlineStringList("  roots", project.Roots))
	b.WriteString(renderInlineStringList("  include", project.Include))
	b.WriteString(renderInlineStringList("  exclude", project.Exclude))
	if strings.TrimSpace(project.Framework) != "" {
		b.WriteString(fmt.Sprintf("  framework: %s\n", project.Framework))
	}
	if strings.TrimSpace(project.Language) != "" {
		b.WriteString(fmt.Sprintf("  language: %s\n", project.Language))
	}
	if strings.TrimSpace(project.Tsconfig) != "" {
		b.WriteString(fmt.Sprintf("  tsconfig: %q\n", project.Tsconfig))
	}
	if len(project.Aliases) > 0 {
		keys := make([]string, 0, len(project.Aliases))
		for key := range project.Aliases {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		b.WriteString("  aliases:\n")
		for _, key := range keys {
			b.WriteString(fmt.Sprintf("    %q:\n", key))
			for _, target := range project.Aliases[key] {
				b.WriteString(fmt.Sprintf("      - %q\n", target))
			}
		}
	}
	return b.String()
}

func renderRuleBlock(rule config.Rule) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  - id: %s\n", rule.ID))
	b.WriteString(fmt.Sprintf("    kind: %s\n", rule.Kind))
	b.WriteString(fmt.Sprintf("    severity: %s\n", rule.Severity))
	if strings.TrimSpace(rule.Template) != "" {
		b.WriteString(fmt.Sprintf("    template: %s\n", rule.Template))
	}
	b.WriteString("    scope:\n")
	for _, value := range rule.Scope {
		b.WriteString(fmt.Sprintf("      - %q\n", value))
	}
	if len(rule.Target) > 0 {
		b.WriteString("    target:\n")
		for _, value := range rule.Target {
			b.WriteString(fmt.Sprintf("      - %q\n", value))
		}
	}
	if len(rule.Except) > 0 {
		b.WriteString("    except:\n")
		for _, value := range rule.Except {
			b.WriteString(fmt.Sprintf("      - %q\n", value))
		}
	}
	if len(rule.Params) > 0 {
		keys := make([]string, 0, len(rule.Params))
		for key := range rule.Params {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		b.WriteString("    params:\n")
		for _, key := range keys {
			b.WriteString(fmt.Sprintf("      %s: %q\n", key, rule.Params[key]))
		}
	}
	if rule.Message != "" {
		b.WriteString(fmt.Sprintf("    message: %q\n", rule.Message))
	}
	return b.String()
}

func renderStringList(key string, values []string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s:\n", key))
	for _, value := range values {
		b.WriteString(fmt.Sprintf("    - %q\n", value))
	}
	return b.String()
}

func renderInlineStringList(key string, values []string) string {
	if len(values) == 0 {
		return fmt.Sprintf("%s: []\n", key)
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return fmt.Sprintf("%s: [%s]\n", key, strings.Join(quoted, ", "))
}

func deepCopyRule(rule config.Rule) config.Rule {
	out := rule
	out.Scope = append([]string{}, rule.Scope...)
	out.Target = append([]string{}, rule.Target...)
	out.Except = append([]string{}, rule.Except...)
	if rule.Params != nil {
		out.Params = make(map[string]string, len(rule.Params))
		for key, value := range rule.Params {
			out.Params[key] = value
		}
	}
	return out
}
