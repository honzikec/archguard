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

func PrintMineText(candidates []Candidate, catalogMatches []PatternMatch, catalogFormat string) {
	PrintText(candidates)
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
	}
}

func PrintMineJSON(candidates []Candidate, catalogMatches []PatternMatch) {
	payload := MineOutput{Candidates: candidates, CatalogMatches: catalogMatches}
	data, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(data))
}

func EmitStarterConfigWithCatalog(candidates []Candidate, adopted []config.Rule) string {
	var b strings.Builder
	b.WriteString("version: 1\n")
	b.WriteString("project:\n")
	b.WriteString("  roots: [\".\"]\n")
	b.WriteString("  include: [\"**/*.ts\", \"**/*.tsx\", \"**/*.js\", \"**/*.jsx\", \"**/*.mjs\", \"**/*.cjs\"]\n")
	b.WriteString("  exclude: [\"**/node_modules/**\", \"**/dist/**\", \"**/build/**\", \"**/.next/**\", \"**/coverage/**\", \"**/.git/**\"]\n")
	b.WriteString("rules:\n")

	for i, c := range candidates {
		b.WriteString(fmt.Sprintf("  - id: MINED-%03d\n", i+1))
		b.WriteString(fmt.Sprintf("    kind: %s\n", c.Kind))
		b.WriteString(fmt.Sprintf("    severity: %s\n", c.Severity))
		b.WriteString("    scope:\n")
		for _, s := range c.Scope {
			b.WriteString(fmt.Sprintf("      - %q\n", s))
		}
		if len(c.Target) > 0 {
			b.WriteString("    target:\n")
			for _, t := range c.Target {
				b.WriteString(fmt.Sprintf("      - %q\n", t))
			}
		}
		b.WriteString(fmt.Sprintf("    message: %q\n", c.Evidence))
	}

	if len(adopted) == 0 {
		return b.String()
	}

	sort.Slice(adopted, func(i, j int) bool {
		return adopted[i].ID < adopted[j].ID
	})

	for _, rule := range adopted {
		b.WriteString(fmt.Sprintf("  - id: %s\n", rule.ID))
		b.WriteString(fmt.Sprintf("    kind: %s\n", rule.Kind))
		b.WriteString(fmt.Sprintf("    severity: %s\n", rule.Severity))
		if rule.Template != "" {
			b.WriteString(fmt.Sprintf("    template: %s\n", rule.Template))
		}
		b.WriteString("    scope:\n")
		for _, s := range rule.Scope {
			b.WriteString(fmt.Sprintf("      - %q\n", s))
		}
		if len(rule.Target) > 0 {
			b.WriteString("    target:\n")
			for _, t := range rule.Target {
				b.WriteString(fmt.Sprintf("      - %q\n", t))
			}
		}
		if len(rule.Except) > 0 {
			b.WriteString("    except:\n")
			for _, e := range rule.Except {
				b.WriteString(fmt.Sprintf("      - %q\n", e))
			}
		}
		if len(rule.Params) > 0 {
			keys := make([]string, 0, len(rule.Params))
			for k := range rule.Params {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			b.WriteString("    params:\n")
			for _, k := range keys {
				b.WriteString(fmt.Sprintf("      %s: %q\n", k, rule.Params[k]))
			}
		}
		if rule.Message != "" {
			b.WriteString(fmt.Sprintf("    message: %q\n", rule.Message))
		}
	}

	return b.String()
}
