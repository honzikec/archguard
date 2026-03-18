package miner

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/honzikec/archguard/internal/catalog"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/pathutil"
)

type PatternMatch struct {
	CatalogID    string      `json:"catalog_id" yaml:"catalog_id"`
	Name         string      `json:"name" yaml:"name"`
	Category     string      `json:"category" yaml:"category"`
	Score        float64     `json:"score" yaml:"score"`
	Confidence   string      `json:"confidence" yaml:"confidence"`
	Evidence     string      `json:"evidence" yaml:"evidence"`
	ProposedRule config.Rule `json:"proposed_rule" yaml:"proposed_rule"`
}

type CatalogOptions struct {
	ShowLowConfidence bool
}

func MatchCatalog(patterns []catalog.Pattern, candidates []Candidate, files []string, opts CatalogOptions) ([]PatternMatch, error) {
	matches := make([]PatternMatch, 0)
	for _, pattern := range patterns {
		switch pattern.Detection.Heuristic.Type {
		case "prevalence_boundary":
			matches = append(matches, matchPrevalenceBoundary(pattern, candidates)...)
		case "prevalence_package_boundary":
			matches = append(matches, matchPrevalencePackageBoundary(pattern, candidates)...)
		case "construction_new_outside_root":
			m, ok, err := matchConstructionPattern(pattern, files)
			if err != nil {
				return nil, err
			}
			if ok {
				matches = append(matches, m)
			}
		}
	}

	deduped := dedupeMatches(matches)
	sort.Slice(deduped, func(i, j int) bool {
		if deduped[i].Score != deduped[j].Score {
			return deduped[i].Score > deduped[j].Score
		}
		return deduped[i].CatalogID < deduped[j].CatalogID
	})

	filtered := make([]PatternMatch, 0, len(deduped))
	for _, m := range deduped {
		if !opts.ShowLowConfidence && m.Confidence == "LOW" {
			continue
		}
		filtered = append(filtered, m)
	}

	return filtered, nil
}

func matchPrevalenceBoundary(pattern catalog.Pattern, candidates []Candidate) []PatternMatch {
	sourceGlobs := toStringSlice(pattern.Detection.Heuristic.Params["source_globs"])
	targetGlobs := toStringSlice(pattern.Detection.Heuristic.Params["target_globs"])
	maxPrevalence := toFloat(pattern.Detection.Heuristic.Params["max_prevalence"], 0.02)
	minSupport := toInt(pattern.Detection.Heuristic.Params["min_support"], 20)

	matches := make([]PatternMatch, 0)
	for _, c := range candidates {
		if c.Kind != config.KindNoImport || len(c.Scope) == 0 || len(c.Target) == 0 {
			continue
		}
		sourceMatch := pathutil.MatchAny(sourceGlobs, c.Scope[0])
		targetMatch := pathutil.MatchAny(targetGlobs, c.Target[0])
		if !sourceMatch && !targetMatch {
			continue
		}

		structuralFit := 0.5
		if sourceMatch && targetMatch {
			structuralFit = 1.0
		}
		prevalenceSupport := prevalenceSupportScore(c.Prevalence, c.Support, maxPrevalence, minSupport)
		naming := namingScore(c.Scope[0], c.Target[0], sourceGlobs, targetGlobs)
		score := weightedScore(structuralFit, prevalenceSupport, naming)

		rule := catalogRuleFromCandidate(pattern, c, map[string]string{"relation": "imports"})
		matches = append(matches, PatternMatch{
			CatalogID:    pattern.ID,
			Name:         pattern.Name,
			Category:     pattern.Category,
			Score:        score,
			Confidence:   confidenceFromScore(score),
			Evidence:     c.Evidence,
			ProposedRule: rule,
		})
	}
	return matches
}

func matchPrevalencePackageBoundary(pattern catalog.Pattern, candidates []Candidate) []PatternMatch {
	sourceGlobs := toStringSlice(pattern.Detection.Heuristic.Params["source_globs"])
	packageGlobs := toStringSlice(pattern.Detection.Heuristic.Params["package_globs"])
	maxPrevalence := toFloat(pattern.Detection.Heuristic.Params["max_prevalence"], 0.02)
	minSupport := toInt(pattern.Detection.Heuristic.Params["min_support"], 20)

	matches := make([]PatternMatch, 0)
	for _, c := range candidates {
		if c.Kind != config.KindNoPackage || len(c.Scope) == 0 || len(c.Target) == 0 {
			continue
		}
		sourceMatch := pathutil.MatchAny(sourceGlobs, c.Scope[0])
		packageMatch := pathutil.MatchAny(packageGlobs, c.Target[0]) || containsExact(packageGlobs, c.Target[0])
		if !sourceMatch && !packageMatch {
			continue
		}

		structuralFit := 0.5
		if sourceMatch && packageMatch {
			structuralFit = 1.0
		}
		prevalenceSupport := prevalenceSupportScore(c.Prevalence, c.Support, maxPrevalence, minSupport)
		naming := namingScore(c.Scope[0], c.Target[0], sourceGlobs, packageGlobs)
		score := weightedScore(structuralFit, prevalenceSupport, naming)

		rule := catalogRuleFromCandidate(pattern, c, map[string]string{"relation": "packages"})
		matches = append(matches, PatternMatch{
			CatalogID:    pattern.ID,
			Name:         pattern.Name,
			Category:     pattern.Category,
			Score:        score,
			Confidence:   confidenceFromScore(score),
			Evidence:     c.Evidence,
			ProposedRule: rule,
		})
	}
	return matches
}

func matchConstructionPattern(pattern catalog.Pattern, files []string) (PatternMatch, bool, error) {
	scopeGlobs := toStringSlice(pattern.Detection.Heuristic.Params["scope_globs"])
	serviceGlobs := toStringSlice(pattern.Detection.Heuristic.Params["service_globs"])
	allowedNewGlobs := toStringSlice(pattern.Detection.Heuristic.Params["allowed_new_globs"])
	maxPrevalence := toFloat(pattern.Detection.Heuristic.Params["max_prevalence"], 0.02)
	minSupport := toInt(pattern.Detection.Heuristic.Params["min_support"], 20)
	serviceNamePattern := toString(pattern.Detection.Heuristic.Params["service_name_regex"], ".*Service$")

	serviceNameRegex, err := regexp.Compile(serviceNamePattern)
	if err != nil {
		return PatternMatch{}, false, fmt.Errorf("invalid service_name_regex for %s: %w", pattern.ID, err)
	}

	serviceNames := collectServiceNamesForMatcher(files, serviceGlobs)
	newExpr := regexp.MustCompile(`\bnew\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)

	support := 0
	violations := 0
	evidenceExamples := make([]string, 0, 3)

	for _, file := range files {
		if !pathutil.MatchAny(scopeGlobs, file) {
			continue
		}
		support++
		if pathutil.MatchAny(allowedNewGlobs, file) {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		text := string(content)
		idxs := newExpr.FindAllStringSubmatchIndex(text, -1)
		for _, idx := range idxs {
			if len(idx) < 4 {
				continue
			}
			start, end := idx[2], idx[3]
			if start < 0 || end <= start {
				continue
			}
			className := text[start:end]
			if len(serviceNames) > 0 {
				if _, ok := serviceNames[className]; !ok {
					continue
				}
			} else if !serviceNameRegex.MatchString(className) {
				continue
			}
			violations++
			if len(evidenceExamples) < 3 {
				evidenceExamples = append(evidenceExamples, fmt.Sprintf("%s:new %s", file, className))
			}
		}
	}

	if support == 0 {
		return PatternMatch{}, false, nil
	}

	prevalence := float64(violations) / float64(support)
	structuralFit := 0.7
	if len(serviceNames) > 0 {
		structuralFit = 1.0
	}
	prevalenceSupport := prevalenceSupportScore(prevalence, support, maxPrevalence, minSupport)
	naming := 0.8
	score := weightedScore(structuralFit, prevalenceSupport, naming)
	confidence := confidenceFromScore(score)

	evidence := fmt.Sprintf("%d/%d scoped files instantiate service classes", violations, support)
	if len(evidenceExamples) > 0 {
		evidence = evidence + "; examples: " + strings.Join(evidenceExamples, ", ")
	}

	ruleHash := shortHash(strings.Join(scopeGlobs, ",") + "|" + strings.Join(serviceGlobs, ",") + "|" + strings.Join(allowedNewGlobs, ","))
	rule := config.Rule{
		ID:       fmt.Sprintf("AG-CAT-%s-%s", sanitizeCatalogID(pattern.ID), ruleHash),
		Kind:     config.KindPattern,
		Template: "construction_policy",
		Severity: pattern.RuleTemplate.Defaults.Severity,
		Scope:    scopeGlobs,
		Target:   serviceGlobs,
		Except:   allowedNewGlobs,
		Params: map[string]string{
			"service_name_regex": serviceNamePattern,
		},
		Message: fmt.Sprintf("[derived_from_catalog=%s] %s", pattern.ID, pattern.Description),
	}

	return PatternMatch{
		CatalogID:    pattern.ID,
		Name:         pattern.Name,
		Category:     pattern.Category,
		Score:        score,
		Confidence:   confidence,
		Evidence:     evidence,
		ProposedRule: rule,
	}, true, nil
}

func catalogRuleFromCandidate(pattern catalog.Pattern, c Candidate, params map[string]string) config.Rule {
	rule := config.Rule{
		ID:       fmt.Sprintf("AG-CAT-%s-%s", sanitizeCatalogID(pattern.ID), shortHash(strings.Join(c.Scope, ",")+"|"+strings.Join(c.Target, ","))),
		Kind:     config.KindPattern,
		Template: pattern.RuleTemplate.Template,
		Severity: pattern.RuleTemplate.Defaults.Severity,
		Scope:    append([]string{}, c.Scope...),
		Target:   append([]string{}, c.Target...),
		Message:  fmt.Sprintf("[derived_from_catalog=%s] %s", pattern.ID, pattern.Description),
	}
	rule.Params = map[string]string{}
	for k, v := range pattern.RuleTemplate.Defaults.Params {
		rule.Params[k] = v
	}
	for k, v := range params {
		rule.Params[k] = v
	}
	return rule
}

func dedupeMatches(matches []PatternMatch) []PatternMatch {
	seen := map[string]PatternMatch{}
	for _, m := range matches {
		key := ruleSignature(m.ProposedRule)
		existing, ok := seen[key]
		if !ok || m.Score > existing.Score {
			seen[key] = m
		}
	}
	out := make([]PatternMatch, 0, len(seen))
	for _, m := range seen {
		out = append(out, m)
	}
	return out
}

func ruleSignature(rule config.Rule) string {
	scope := append([]string{}, rule.Scope...)
	target := append([]string{}, rule.Target...)
	except := append([]string{}, rule.Except...)
	sort.Strings(scope)
	sort.Strings(target)
	sort.Strings(except)
	params := make([]string, 0, len(rule.Params))
	for k, v := range rule.Params {
		params = append(params, k+"="+v)
	}
	sort.Strings(params)
	return strings.Join([]string{
		rule.Kind,
		rule.Template,
		rule.Severity,
		strings.Join(scope, ","),
		strings.Join(target, ","),
		strings.Join(except, ","),
		strings.Join(params, ","),
	}, "|")
}

func prevalenceSupportScore(prevalence float64, support int, maxPrevalence float64, minSupport int) float64 {
	supportScore := 1.0
	if minSupport > 0 {
		supportScore = clamp(float64(support)/float64(minSupport), 0, 1)
	}
	prevalenceScore := 1.0
	if prevalence > maxPrevalence {
		if maxPrevalence >= 1.0 {
			prevalenceScore = 0
		} else {
			prevalenceScore = clamp(1.0-((prevalence-maxPrevalence)/(1.0-maxPrevalence)), 0, 1)
		}
	}
	return (supportScore + prevalenceScore) / 2.0
}

func namingScore(scope, target string, sourceGlobs, targetGlobs []string) float64 {
	tokens := []string{"domain", "infra", "application", "app", "ui", "service", "services", "core"}
	hits := 0
	total := 0
	for _, tok := range tokens {
		expected := false
		for _, g := range sourceGlobs {
			if strings.Contains(strings.ToLower(g), tok) {
				expected = true
				break
			}
		}
		for _, g := range targetGlobs {
			if strings.Contains(strings.ToLower(g), tok) {
				expected = true
				break
			}
		}
		if !expected {
			continue
		}
		total++
		if strings.Contains(strings.ToLower(scope), tok) || strings.Contains(strings.ToLower(target), tok) {
			hits++
		}
	}
	if total == 0 {
		return 0.6
	}
	return clamp(float64(hits)/float64(total), 0, 1)
}

func weightedScore(structural, prevalenceSupport, naming float64) float64 {
	return clamp((0.4*structural)+(0.4*prevalenceSupport)+(0.2*naming), 0, 1)
}

func confidenceFromScore(score float64) string {
	switch {
	case score >= 0.85:
		return "HIGH"
	case score >= 0.65:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func AdoptCatalogMatches(matches []PatternMatch, threshold string) []config.Rule {
	threshold = strings.ToLower(strings.TrimSpace(threshold))
	allowed := map[string]bool{"high": true, "medium": true}
	if !allowed[threshold] {
		threshold = "high"
	}

	out := make([]config.Rule, 0)
	for _, m := range matches {
		if threshold == "high" && m.Confidence != "HIGH" {
			continue
		}
		if threshold == "medium" && m.Confidence == "LOW" {
			continue
		}
		out = append(out, m.ProposedRule)
	}
	return out
}

func collectServiceNamesForMatcher(files []string, serviceGlobs []string) map[string]struct{} {
	serviceNames := map[string]struct{}{}
	classDecl := regexp.MustCompile(`\bclass\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`)
	for _, file := range files {
		if !pathutil.MatchAny(serviceGlobs, file) {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		idxs := classDecl.FindAllStringSubmatchIndex(string(content), -1)
		for _, idx := range idxs {
			if len(idx) < 4 {
				continue
			}
			start, end := idx[2], idx[3]
			if start < 0 || end <= start {
				continue
			}
			serviceNames[string(content[start:end])] = struct{}{}
		}
	}
	return serviceNames
}

func toStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return append([]string{}, t...)
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return []string{t}
	default:
		return nil
	}
}

func toFloat(v any, fallback float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err == nil {
			return f
		}
	}
	return fallback
}

func toInt(v any, fallback int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(t))
		if err == nil {
			return i
		}
	}
	return fallback
}

func toString(v any, fallback string) string {
	if s, ok := v.(string); ok {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return fallback
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func containsExact(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func sanitizeCatalogID(id string) string {
	s := strings.ToUpper(id)
	s = strings.ReplaceAll(s, "CAT-", "")
	s = strings.ReplaceAll(s, "_", "-")
	return regexp.MustCompile(`[^A-Z0-9-]+`).ReplaceAllString(s, "")
}

func shortHash(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])[:8]
}
