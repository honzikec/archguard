package miner

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/honzikec/archguard/internal/catalog"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/pathutil"
	"github.com/honzikec/archguard/internal/resolve"
)

const (
	defaultCatalogMinSupport     = 20
	constructionMinScopedFloor   = 3
	constructionPrevalenceTarget = 0.4
)

type PatternMatch struct {
	CatalogID         string         `json:"catalog_id" yaml:"catalog_id"`
	Name              string         `json:"name" yaml:"name"`
	Category          string         `json:"category" yaml:"category"`
	Score             float64        `json:"score" yaml:"score"`
	Confidence        string         `json:"confidence" yaml:"confidence"`
	Evidence          string         `json:"evidence" yaml:"evidence"`
	ScopedFiles       int            `json:"scoped_files" yaml:"scoped_files"`
	EligibleFiles     int            `json:"eligible_files" yaml:"eligible_files"`
	ViolatingFiles    int            `json:"violating_files" yaml:"violating_files"`
	Support           int            `json:"support" yaml:"support"`
	Prevalence        float64        `json:"prevalence" yaml:"prevalence"`
	ScoreComponents   ScoreBreakdown `json:"score_components" yaml:"score_components"`
	ResolvedCount     int            `json:"resolved_count,omitempty" yaml:"resolved_count,omitempty"`
	UnresolvedCount   int            `json:"unresolved_count,omitempty" yaml:"unresolved_count,omitempty"`
	SampleLocations   []string       `json:"sample_locations,omitempty" yaml:"sample_locations,omitempty"`
	ResolvedExamples  []string       `json:"resolved_examples,omitempty" yaml:"resolved_examples,omitempty"`
	UnresolvedReasons []ReasonCount  `json:"unresolved_reasons,omitempty" yaml:"unresolved_reasons,omitempty"`
	ProposedRule      config.Rule    `json:"proposed_rule" yaml:"proposed_rule"`
}

type ScoreBreakdown struct {
	StructuralFit     float64 `json:"structural_fit" yaml:"structural_fit"`
	PrevalenceSupport float64 `json:"prevalence_support" yaml:"prevalence_support"`
	NamingFit         float64 `json:"naming_fit" yaml:"naming_fit"`
}

type ReasonCount struct {
	Reason string `json:"reason" yaml:"reason"`
	Count  int    `json:"count" yaml:"count"`
}

type CatalogOptions struct {
	ShowLowConfidence bool
}

func MatchCatalog(patterns []catalog.Pattern, candidates []Candidate, files []string, project config.ProjectSettings, opts CatalogOptions) ([]PatternMatch, error) {
	matches := make([]PatternMatch, 0)
	for _, pattern := range patterns {
		switch pattern.Detection.Heuristic.Type {
		case "prevalence_boundary":
			matches = append(matches, matchPrevalenceBoundary(pattern, candidates)...)
		case "prevalence_package_boundary":
			matches = append(matches, matchPrevalencePackageBoundary(pattern, candidates)...)
		case "construction_new_outside_root":
			m, ok, err := matchConstructionPattern(pattern, files, project)
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
		effectiveSupport := effectiveMinSupport(minSupport, c.Support)
		prevalenceSupport := prevalenceSupportScore(c.Prevalence, c.Support, maxPrevalence, effectiveSupport)
		naming := namingScore(c.Scope[0], c.Target[0], sourceGlobs, targetGlobs)
		score := weightedScore(structuralFit, prevalenceSupport, naming)

		rule := catalogRuleFromCandidate(pattern, c, map[string]string{"relation": "imports"})
		matches = append(matches, PatternMatch{
			CatalogID:      pattern.ID,
			Name:           pattern.Name,
			Category:       pattern.Category,
			Score:          score,
			Confidence:     confidenceFromScore(score),
			Evidence:       c.Evidence,
			ScopedFiles:    c.Support,
			EligibleFiles:  c.Support,
			ViolatingFiles: c.Violations,
			Support:        c.Support,
			Prevalence:     c.Prevalence,
			ScoreComponents: ScoreBreakdown{
				StructuralFit:     structuralFit,
				PrevalenceSupport: prevalenceSupport,
				NamingFit:         naming,
			},
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
		effectiveSupport := effectiveMinSupport(minSupport, c.Support)
		prevalenceSupport := prevalenceSupportScore(c.Prevalence, c.Support, maxPrevalence, effectiveSupport)
		naming := namingScore(c.Scope[0], c.Target[0], sourceGlobs, packageGlobs)
		score := weightedScore(structuralFit, prevalenceSupport, naming)

		rule := catalogRuleFromCandidate(pattern, c, map[string]string{"relation": "packages"})
		matches = append(matches, PatternMatch{
			CatalogID:      pattern.ID,
			Name:           pattern.Name,
			Category:       pattern.Category,
			Score:          score,
			Confidence:     confidenceFromScore(score),
			Evidence:       c.Evidence,
			ScopedFiles:    c.Support,
			EligibleFiles:  c.Support,
			ViolatingFiles: c.Violations,
			Support:        c.Support,
			Prevalence:     c.Prevalence,
			ScoreComponents: ScoreBreakdown{
				StructuralFit:     structuralFit,
				PrevalenceSupport: prevalenceSupport,
				NamingFit:         naming,
			},
			ProposedRule: rule,
		})
	}
	return matches
}

func matchConstructionPattern(pattern catalog.Pattern, files []string, project config.ProjectSettings) (PatternMatch, bool, error) {
	scopeGlobs := toStringSlice(pattern.Detection.Heuristic.Params["scope_globs"])
	serviceGlobs := toStringSlice(pattern.Detection.Heuristic.Params["service_globs"])
	allowedNewGlobs := toStringSlice(pattern.Detection.Heuristic.Params["allowed_new_globs"])
	maxPrevalence := toFloat(pattern.Detection.Heuristic.Params["max_prevalence"], 0.02)
	minSupport := toInt(pattern.Detection.Heuristic.Params["min_support"], 20)
	serviceNamePattern := toString(pattern.Detection.Heuristic.Params["service_name_regex"], ".*Service$")

	scopedFiles := make([]string, 0)
	for _, file := range files {
		if pathutil.MatchAny(scopeGlobs, file) {
			scopedFiles = append(scopedFiles, file)
		}
	}
	sort.Strings(scopedFiles)
	if len(scopedFiles) == 0 {
		return PatternMatch{}, false, nil
	}

	eligibleFiles := map[string]struct{}{}
	violatingFiles := map[string]struct{}{}
	violationEvents := 0
	resolvedCount := 0
	unresolvedCount := 0
	resolvedExampleSet := map[string]struct{}{}
	sampleLocationSet := map[string]struct{}{}
	unresolvedByReason := map[string]int{}

	constructions, err := resolve.ResolveConstructions(files, project, serviceGlobs, serviceNamePattern)
	if err != nil {
		return PatternMatch{}, false, err
	}
	sort.Slice(constructions, func(i, j int) bool {
		if constructions[i].FilePath != constructions[j].FilePath {
			return constructions[i].FilePath < constructions[j].FilePath
		}
		if constructions[i].Line != constructions[j].Line {
			return constructions[i].Line < constructions[j].Line
		}
		if constructions[i].Column != constructions[j].Column {
			return constructions[i].Column < constructions[j].Column
		}
		if constructions[i].ClassName != constructions[j].ClassName {
			return constructions[i].ClassName < constructions[j].ClassName
		}
		return constructions[i].UnresolvedReason < constructions[j].UnresolvedReason
	})

	for _, c := range constructions {
		if !pathutil.MatchAny(scopeGlobs, c.FilePath) {
			continue
		}
		if pathutil.MatchAny(allowedNewGlobs, c.FilePath) {
			continue
		}
		eligibleFiles[c.FilePath] = struct{}{}
		if c.IsResolved {
			resolvedCount++
		} else {
			unresolvedCount++
			reason := strings.TrimSpace(c.UnresolvedReason)
			if reason == "" {
				reason = "unknown"
			}
			unresolvedByReason[reason]++
		}
		if !c.IsResolved || !c.IsService {
			continue
		}
		violatingFiles[c.FilePath] = struct{}{}
		violationEvents++
		resolvedExampleSet[fmt.Sprintf("%s:%d:new %s", c.FilePath, c.Line, c.ClassName)] = struct{}{}
		sampleLocationSet[fmt.Sprintf("%s:%d", c.FilePath, c.Line)] = struct{}{}
	}

	support := len(eligibleFiles)
	violatingFileCount := len(violatingFiles)
	prevalence := 0.0
	if support > 0 {
		prevalence = float64(violatingFileCount) / float64(support)
	}
	structuralFit := 0.5
	if resolvedCount > 0 {
		structuralFit = clamp(float64(resolvedCount)/float64(resolvedCount+unresolvedCount), 0, 1)
	}
	effectiveSupport := effectiveMinSupport(minSupport, len(scopedFiles))
	prevalenceSupport := constructionPrevalenceSupportScore(prevalence, support, maxPrevalence, effectiveSupport)
	naming := 0.6
	if unresolvedCount > 0 && resolvedCount == 0 {
		naming = 0.3
	}
	score := weightedScore(structuralFit, prevalenceSupport, naming)
	confidence := confidenceFromScore(score)

	resolvedExamples := sortedLimitedKeys(resolvedExampleSet, 3)
	sampleLocations := sortedLimitedKeys(sampleLocationSet, 3)
	unresolvedReasons := sortedReasonCounts(unresolvedByReason)

	evidence := fmt.Sprintf("%d/%d eligible files instantiate service classes (%d events, resolved=%d unresolved=%d)",
		violatingFileCount, support, violationEvents, resolvedCount, unresolvedCount)
	if len(resolvedExamples) > 0 {
		evidence = evidence + "; examples: " + strings.Join(resolvedExamples, ", ")
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
		CatalogID:         pattern.ID,
		Name:              pattern.Name,
		Category:          pattern.Category,
		Score:             score,
		Confidence:        confidence,
		Evidence:          evidence,
		ScopedFiles:       len(scopedFiles),
		EligibleFiles:     support,
		ViolatingFiles:    violatingFileCount,
		Support:           support,
		Prevalence:        prevalence,
		ScoreComponents:   ScoreBreakdown{StructuralFit: structuralFit, PrevalenceSupport: prevalenceSupport, NamingFit: naming},
		ResolvedCount:     resolvedCount,
		UnresolvedCount:   unresolvedCount,
		SampleLocations:   sampleLocations,
		ResolvedExamples:  resolvedExamples,
		UnresolvedReasons: unresolvedReasons,
		ProposedRule:      rule,
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

func constructionPrevalenceSupportScore(prevalence float64, support int, _ float64, minSupport int) float64 {
	supportScore := 0.0
	if minSupport > 0 {
		supportScore = clamp(float64(support)/float64(minSupport), 0, 1)
	}
	// Construction-policy recommendations become more useful as direct construction prevalence increases.
	prevalenceScore := clamp(prevalence/constructionPrevalenceTarget, 0, 1)
	return (supportScore + prevalenceScore) / 2.0
}

func effectiveMinSupport(configuredMinSupport int, scopedFiles int) int {
	if configuredMinSupport <= 0 {
		configuredMinSupport = defaultCatalogMinSupport
	}
	scopedFloor := scopedFiles
	if scopedFloor < constructionMinScopedFloor {
		scopedFloor = constructionMinScopedFloor
	}
	if configuredMinSupport < scopedFloor {
		return configuredMinSupport
	}
	return scopedFloor
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

func sortedLimitedKeys(values map[string]struct{}, limit int) []string {
	if len(values) == 0 || limit <= 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func sortedReasonCounts(reasonMap map[string]int) []ReasonCount {
	if len(reasonMap) == 0 {
		return nil
	}
	out := make([]ReasonCount, 0, len(reasonMap))
	for reason, count := range reasonMap {
		out = append(out, ReasonCount{Reason: reason, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Reason < out[j].Reason
	})
	return out
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
