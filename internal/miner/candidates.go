package miner

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/graph"
)

type Options struct {
	MinSupport           int
	MaxPrevalence        float64
	MaxCandidatesPerKind int
	DebugStats           *DebugStats
}

type DebugStats struct {
	Dropped map[string]int
}

func NewDebugStats() *DebugStats {
	return &DebugStats{Dropped: map[string]int{}}
}

func (s *DebugStats) addDrop(kind, reason string) {
	if s == nil {
		return
	}
	if s.Dropped == nil {
		s.Dropped = map[string]int{}
	}
	s.Dropped[dropKey(kind, reason)]++
}

func dropKey(kind, reason string) string {
	return kind + ":" + reason
}

func recordDrop(opts Options, kind, reason string) {
	if opts.DebugStats == nil {
		return
	}
	opts.DebugStats.addDrop(kind, reason)
}

type Candidate struct {
	Kind       string          `json:"kind" yaml:"kind"`
	Scope      []string        `json:"scope" yaml:"scope"`
	Target     []string        `json:"target,omitempty" yaml:"target,omitempty"`
	Severity   string          `json:"severity" yaml:"severity"`
	Support    int             `json:"support" yaml:"support"`
	Violations int             `json:"violations" yaml:"violations"`
	Prevalence float64         `json:"prevalence" yaml:"prevalence"`
	Confidence ConfidenceLevel `json:"confidence" yaml:"confidence"`
	Evidence   string          `json:"evidence" yaml:"evidence"`
}

type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "HIGH"
	ConfidenceMedium ConfidenceLevel = "MEDIUM"
	ConfidenceLow    ConfidenceLevel = "LOW"
)

func Propose(g *graph.Graph, allFiles []string, opts Options) []Candidate {
	if opts.MinSupport <= 0 {
		opts.MinSupport = 20
	}
	if opts.MaxPrevalence <= 0 {
		opts.MaxPrevalence = 0.02
	}
	if opts.MaxCandidatesPerKind < 0 {
		opts.MaxCandidatesPerKind = 0
	}
	if opts.MaxCandidatesPerKind == 0 {
		opts.MaxCandidatesPerKind = 200
	}

	noImportRaw := proposeNoImport(g, opts)
	noImportRaw = aggregateZeroViolationSiblingScopes(noImportRaw, opts)
	noImport := capCandidates(noImportRaw, opts.MaxCandidatesPerKind)
	noPackageRaw := proposeNoPackage(g, opts)
	noPackageRaw = aggregateZeroViolationSiblingScopes(noPackageRaw, opts)
	noPackage := capCandidates(noPackageRaw, opts.MaxCandidatesPerKind)
	filePattern := capCandidates(proposeFilePattern(allFiles, opts), opts.MaxCandidatesPerKind)
	noCycle := capCandidates(proposeNoCycle(g, opts), opts.MaxCandidatesPerKind)

	candidates := make([]Candidate, 0, len(noImport)+len(noPackage)+len(filePattern)+len(noCycle))
	candidates = append(candidates, noImport...)
	candidates = append(candidates, noPackage...)
	candidates = append(candidates, filePattern...)
	candidates = append(candidates, noCycle...)

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Kind != candidates[j].Kind {
			return candidates[i].Kind < candidates[j].Kind
		}
		if len(candidates[i].Scope) == 0 || len(candidates[j].Scope) == 0 {
			return len(candidates[i].Scope) < len(candidates[j].Scope)
		}
		return candidates[i].Scope[0] < candidates[j].Scope[0]
	})
	return candidates
}

func proposeNoImport(g *graph.Graph, opts Options) []Candidate {
	globalTargetUsage := map[string]int{}
	targetSourceSpread := map[string]int{}
	for _, targets := range g.Edges {
		for targetSubtree, count := range targets {
			globalTargetUsage[targetSubtree] += count
			if count > 0 {
				targetSourceSpread[targetSubtree]++
			}
		}
	}

	return mineCandidates(
		g, opts, config.KindNoImport,
		sortedKeys(g.Nodes),
		globalTargetUsage,
		targetSourceSpread,
		minZeroViolationImportTargetUsage(opts.MinSupport),
		func(source, target string) int {
			if edges, ok := g.Edges[source]; ok {
				return edges[target]
			}
			return 0
		},
		func(target string) string { return target + "/**" },
		func(source, target string) bool { return source == target },
	)
}

func proposeNoPackage(g *graph.Graph, opts Options) []Candidate {
	allPackages := map[string]struct{}{}
	globalPackageUsage := map[string]int{}
	packageSourceSpread := map[string]int{}
	for _, packages := range g.PackageEdges {
		for pkg, count := range packages {
			allPackages[pkg] = struct{}{}
			globalPackageUsage[pkg] += count
			if count > 0 {
				packageSourceSpread[pkg]++
			}
		}
	}

	return mineCandidates(
		g, opts, config.KindNoPackage,
		sortedSetKeys(allPackages),
		globalPackageUsage,
		packageSourceSpread,
		minZeroViolationTargetUsage(opts.MinSupport),
		func(source, target string) int {
			if edges, ok := g.PackageEdges[source]; ok {
				return edges[target]
			}
			return 0
		},
		func(target string) string { return target },
		nil,
	)
}

func mineCandidates(
	g *graph.Graph,
	opts Options,
	kind string,
	targets []string,
	targetUsage map[string]int,
	targetSpread map[string]int,
	minGlobalUsage int,
	getViolations func(source, target string) int,
	formatTarget func(target string) string,
	skipTarget func(source, target string) bool,
) []Candidate {
	candidates := make([]Candidate, 0)
	sourceActivity := sourceImportActivity(g)
	sourceSubtrees := sortedKeys(g.Nodes)
	activeSources := 0
	for _, totalFiles := range g.Nodes {
		if totalFiles >= opts.MinSupport {
			activeSources++
		}
	}
	maxSpread := maxZeroViolationSourceSpread(activeSources)
	maxZeroPerScope := maxZeroViolationCandidatesPerScope(opts.MinSupport)
	maxZeroPerTarget := maxZeroViolationCandidatesPerTarget(opts.MinSupport)
	zeroViolationBySource := map[string]int{}
	zeroViolationByTarget := map[string]int{}

	for _, sourceSubtree := range sourceSubtrees {
		totalFiles := g.Nodes[sourceSubtree]
		if totalFiles < opts.MinSupport {
			continue
		}
		sourceImportCount := sourceActivity[sourceSubtree]
		for _, target := range targets {
			if skipTarget != nil && skipTarget(sourceSubtree, target) {
				continue
			}
			usage := targetUsage[target]
			if usage == 0 {
				recordDrop(opts, kind, "target_never_observed")
				continue
			}
			violations := getViolations(sourceSubtree, target)
			if kind == config.KindNoPackage && violations == 0 && !allowZeroViolationNoPackageTarget(target) {
				recordDrop(opts, kind, "zero_non_ecosystem_target")
				continue
			}
			if violations == 0 && usage < minGlobalUsage {
				recordDrop(opts, kind, "zero_low_global_usage")
				continue
			}
			if violations == 0 && sourceImportCount == 0 {
				recordDrop(opts, kind, "zero_inactive_source")
				continue
			}
			if violations == 0 && isLowSignalSourceScope(sourceSubtree) {
				recordDrop(opts, kind, "zero_low_signal_scope")
				continue
			}
			if violations == 0 && targetSpread[target] > maxSpread {
				recordDrop(opts, kind, "zero_target_over_spread")
				continue
			}
			if violations == 0 && zeroViolationBySource[sourceSubtree] >= maxZeroPerScope {
				recordDrop(opts, kind, "zero_source_cap")
				continue
			}
			if violations == 0 && zeroViolationByTarget[target] >= maxZeroPerTarget {
				recordDrop(opts, kind, "zero_target_cap")
				continue
			}
			prevalence := float64(violations) / float64(totalFiles)
			if prevalence > opts.MaxPrevalence {
				recordDrop(opts, kind, "prevalence_over_limit")
				continue
			}
			if violations == 0 {
				zeroViolationBySource[sourceSubtree]++
				zeroViolationByTarget[target]++
			}
			candidates = append(candidates, Candidate{
				Kind:       kind,
				Scope:      []string{sourceSubtree + "/**"},
				Target:     []string{formatTarget(target)},
				Severity:   config.SeverityWarning,
				Support:    totalFiles,
				Violations: violations,
				Prevalence: prevalence,
				Confidence: confidence(prevalence, totalFiles),
				Evidence:   fmt.Sprintf("%d/%d files in %s import %s", violations, totalFiles, sourceSubtree, target),
			})
		}
	}
	return candidates
}

func proposeFilePattern(allFiles []string, opts Options) []Candidate {
	byDir := map[string][]string{}
	for _, f := range allFiles {
		dir := path.Dir(f)
		byDir[dir] = append(byDir[dir], path.Base(f))
	}
	globalBestSuffix, globalBestRatio := dominantFileSuffix(allFiles)

	candidates := make([]Candidate, 0)
	for dir, files := range byDir {
		if len(files) < opts.MinSupport {
			continue
		}
		suffixCount := map[string]int{}
		for _, file := range files {
			parts := strings.SplitN(file, ".", 2)
			if len(parts) == 2 {
				suffixCount["."+parts[1]]++
			}
		}
		bestSuffix := ""
		bestCount := 0
		for suffix, c := range suffixCount {
			if c > bestCount {
				bestCount = c
				bestSuffix = suffix
			}
		}
		if bestCount == 0 {
			continue
		}
		prevalence := float64(len(files)-bestCount) / float64(len(files))
		if prevalence > 0.20 {
			continue
		}
		if shouldDropTrivialFilePattern(bestSuffix, len(files), len(files)-bestCount, globalBestSuffix, globalBestRatio) {
			recordDrop(opts, config.KindFilePattern, "trivial_extension_pattern")
			continue
		}
		regex := "^.*" + regexp.QuoteMeta(bestSuffix) + "$"
		candidates = append(candidates, Candidate{
			Kind:       config.KindFilePattern,
			Scope:      []string{dir + "/**"},
			Target:     []string{regex},
			Severity:   config.SeverityWarning,
			Support:    len(files),
			Violations: len(files) - bestCount,
			Prevalence: prevalence,
			Confidence: ConfidenceHigh,
			Evidence:   fmt.Sprintf("%d/%d files in %s match suffix %s", bestCount, len(files), dir, bestSuffix),
		})
	}
	return candidates
}

func dominantFileSuffix(files []string) (string, float64) {
	if len(files) == 0 {
		return "", 0
	}
	suffixCount := map[string]int{}
	seen := 0
	for _, file := range files {
		base := path.Base(file)
		parts := strings.SplitN(base, ".", 2)
		if len(parts) != 2 {
			continue
		}
		suffix := "." + parts[1]
		suffixCount[suffix]++
		seen++
	}
	if seen == 0 {
		return "", 0
	}
	bestSuffix := ""
	bestCount := 0
	for suffix, count := range suffixCount {
		if count > bestCount {
			bestSuffix = suffix
			bestCount = count
		}
	}
	if bestCount == 0 {
		return "", 0
	}
	return bestSuffix, float64(bestCount) / float64(seen)
}

func shouldDropTrivialFilePattern(suffix string, support, violations int, globalSuffix string, globalRatio float64) bool {
	if support <= 0 || violations != 0 {
		return false
	}
	if suffix != ".php" && suffix != ".phtml" {
		return false
	}
	if suffix != globalSuffix {
		return false
	}
	return globalRatio >= 0.90
}

func proposeNoCycle(g *graph.Graph, opts Options) []Candidate {
	components := DetectCycleComponents(g)
	candidates := make([]Candidate, 0, len(components))
	for _, component := range components {
		if len(component.Nodes) == 0 {
			continue
		}
		support := 0
		for _, node := range component.Nodes {
			support += g.Nodes[node]
		}
		if support < opts.MinSupport {
			continue
		}
		scopeRoot := componentScopeRoot(component.Nodes)
		if scopeRoot == "" || scopeRoot == "." {
			scopeRoot = component.Nodes[0]
		}
		candidates = append(candidates, Candidate{
			Kind:       config.KindNoCycle,
			Scope:      []string{scopeRoot + "/**"},
			Severity:   config.SeverityError,
			Support:    support,
			Violations: len(component.Nodes),
			Prevalence: 1.0,
			Confidence: ConfidenceHigh,
			Evidence:   fmt.Sprintf("cycle component (%d subtrees): %s", len(component.Nodes), strings.Join(component.Nodes, " <-> ")),
		})
	}
	return candidates
}

func componentScopeRoot(nodes []string) string {
	if len(nodes) == 0 {
		return ""
	}
	common := strings.Split(nodes[0], "/")
	for _, node := range nodes[1:] {
		parts := strings.Split(node, "/")
		n := len(common)
		if len(parts) < n {
			n = len(parts)
		}
		i := 0
		for i < n && common[i] == parts[i] {
			i++
		}
		common = common[:i]
		if len(common) == 0 {
			return ""
		}
	}
	return strings.Join(common, "/")
}

func sourceImportActivity(g *graph.Graph) map[string]int {
	activity := map[string]int{}
	for sourceFile, targets := range g.FileEdges {
		if len(targets) == 0 {
			continue
		}
		sourceSubtree := path.Dir(sourceFile)
		activity[sourceSubtree] += len(targets)
	}
	for sourceSubtree, packages := range g.PackageEdges {
		for _, count := range packages {
			activity[sourceSubtree] += count
		}
	}
	return activity
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedSetKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func aggregateZeroViolationSiblingScopes(candidates []Candidate, opts Options) []Candidate {
	if len(candidates) == 0 {
		return candidates
	}

	targetGroups := map[string][]int{}
	for i, c := range candidates {
		if c.Violations != 0 || len(c.Scope) == 0 || len(c.Target) == 0 {
			continue
		}
		targetGroups[c.Target[0]] = append(targetGroups[c.Target[0]], i)
	}

	merged := map[int]struct{}{}
	aggregated := make([]Candidate, 0)
	targets := sortedSetKeysStringSlices(targetGroups)
	for _, target := range targets {
		indices := targetGroups[target]
		parentGroups := map[string][]int{}
		for _, idx := range indices {
			scopeRoot := scopePatternRoot(candidates[idx].Scope[0])
			if scopeRoot == "" {
				continue
			}
			parent := path.Dir(scopeRoot)
			if parent == "." || parent == "/" || parent == "" || parent == scopeRoot {
				continue
			}
			parentGroups[parent] = append(parentGroups[parent], idx)
		}
		parents := sortedSetKeysStringSlices(parentGroups)
		for _, parent := range parents {
			group := parentGroups[parent]
			if len(group) < 2 {
				continue
			}
			totalSupport := 0
			bestSeverity := config.SeverityWarning
			for _, idx := range group {
				c := candidates[idx]
				totalSupport += c.Support
				if c.Severity == config.SeverityError {
					bestSeverity = config.SeverityError
				}
			}
			aggregated = append(aggregated, Candidate{
				Kind:       candidates[group[0]].Kind,
				Scope:      []string{parent + "/**"},
				Target:     []string{target},
				Severity:   bestSeverity,
				Support:    totalSupport,
				Violations: 0,
				Prevalence: 0,
				Confidence: confidence(0, totalSupport),
				Evidence:   fmt.Sprintf("0/%d files across %d sibling scopes under %s import %s", totalSupport, len(group), parent, target),
			})
			for _, idx := range group {
				merged[idx] = struct{}{}
			}
			recordDrop(opts, candidates[group[0]].Kind, "zero_parent_aggregated")
		}
	}

	if len(merged) == 0 {
		return candidates
	}

	out := make([]Candidate, 0, len(candidates)-len(merged)+len(aggregated))
	for i, c := range candidates {
		if _, ok := merged[i]; ok {
			continue
		}
		out = append(out, c)
	}
	out = append(out, aggregated...)
	return out
}

func sortedSetKeysStringSlices[V any](m map[string][]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func scopePatternRoot(scopePattern string) string {
	s := strings.TrimSpace(scopePattern)
	if s == "" {
		return ""
	}
	return strings.TrimSuffix(s, "/**")
}

func capCandidates(candidates []Candidate, max int) []Candidate {
	if max <= 0 || len(candidates) <= max {
		return candidates
	}
	sort.Slice(candidates, func(i, j int) bool {
		return betterCandidate(candidates[i], candidates[j])
	})
	violating := make([]Candidate, 0, len(candidates))
	zeroViolation := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.Violations > 0 {
			violating = append(violating, c)
			continue
		}
		zeroViolation = append(zeroViolation, c)
	}
	if len(violating) >= max {
		return append([]Candidate{}, violating[:max]...)
	}
	out := make([]Candidate, 0, max)
	out = append(out, violating...)
	remaining := max - len(out)
	maxZeroWithinCap := maxZeroCandidatesWithinCap(max, len(violating))
	if remaining > maxZeroWithinCap {
		remaining = maxZeroWithinCap
	}
	if remaining > len(zeroViolation) {
		remaining = len(zeroViolation)
	}
	out = append(out, zeroViolation[:remaining]...)
	return out
}

func maxZeroCandidatesWithinCap(max, violatingCount int) int {
	if max <= 0 {
		return 0
	}
	limit := max / 2
	if limit < 20 {
		limit = max
	}
	// If there are no violating findings, still keep a useful but bounded set of
	// zero-violation candidates.
	if violatingCount == 0 {
		fallback := max / 3
		if fallback < 20 {
			fallback = max
		}
		if fallback < limit {
			limit = fallback
		}
	}
	if limit > max-violatingCount {
		limit = max - violatingCount
	}
	if limit < 0 {
		return 0
	}
	return limit
}

func betterCandidate(a, b Candidate) bool {
	if rankConfidence(a.Confidence) != rankConfidence(b.Confidence) {
		return rankConfidence(a.Confidence) > rankConfidence(b.Confidence)
	}
	if a.Support != b.Support {
		return a.Support > b.Support
	}
	if a.Prevalence != b.Prevalence {
		return a.Prevalence < b.Prevalence
	}
	if a.Violations != b.Violations {
		return a.Violations < b.Violations
	}
	aScope, bScope := "", ""
	if len(a.Scope) > 0 {
		aScope = a.Scope[0]
	}
	if len(b.Scope) > 0 {
		bScope = b.Scope[0]
	}
	if aScope != bScope {
		return aScope < bScope
	}
	aTarget, bTarget := "", ""
	if len(a.Target) > 0 {
		aTarget = a.Target[0]
	}
	if len(b.Target) > 0 {
		bTarget = b.Target[0]
	}
	return aTarget < bTarget
}

func rankConfidence(confidence ConfidenceLevel) int {
	switch confidence {
	case ConfidenceHigh:
		return 3
	case ConfidenceMedium:
		return 2
	case ConfidenceLow:
		return 1
	default:
		return 0
	}
}

func minZeroViolationTargetUsage(minSupport int) int {
	if minSupport <= 1 {
		return 1
	}
	threshold := minSupport / 2
	if threshold < 1 {
		return 1
	}
	return threshold
}

func minZeroViolationImportTargetUsage(minSupport int) int {
	if minSupport <= 1 {
		return 1
	}
	if minSupport < 1 {
		return 1
	}
	return minSupport
}

func maxZeroViolationCandidatesPerScope(minSupport int) int {
	if minSupport >= 50 {
		return 8
	}
	if minSupport >= 20 {
		return 5
	}
	return 3
}

func maxZeroViolationCandidatesPerTarget(minSupport int) int {
	if minSupport >= 50 {
		return 16
	}
	if minSupport >= 20 {
		return 10
	}
	return 5
}

func allowZeroViolationNoPackageTarget(target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	if strings.HasPrefix(target, "node:") {
		return true
	}
	if strings.HasPrefix(target, "@") {
		parts := strings.Split(target, "/")
		return len(parts) >= 2 && parts[0] != "@" && parts[1] != ""
	}
	for _, r := range target {
		if r >= 'A' && r <= 'Z' {
			return false
		}
	}
	return true
}

func isLowSignalSourceScope(scope string) bool {
	if scope == "" {
		return false
	}
	parts := strings.Split(strings.ToLower(strings.Trim(scope, "/")), "/")
	if len(parts) == 0 {
		return false
	}
	lowSignal := map[string]struct{}{
		"test":       {},
		"tests":      {},
		"__tests__":  {},
		"testing":    {},
		"fixture":    {},
		"fixtures":   {},
		"golden":     {},
		"example":    {},
		"examples":   {},
		"sample":     {},
		"samples":    {},
		"benchmark":  {},
		"benchmarks": {},
		"e2e":        {},
		"compliance": {},
		"spec":       {},
		"specs":      {},
	}
	for _, part := range parts {
		if _, ok := lowSignal[part]; ok {
			return true
		}
	}
	return false
}

func maxZeroViolationSourceSpread(totalSources int) int {
	if totalSources <= 4 {
		return totalSources
	}
	spread := totalSources / 4
	if spread < 3 {
		spread = 3
	}
	return spread
}

func confidence(prevalence float64, support int) ConfidenceLevel {
	if prevalence <= 0.01 && support >= 50 {
		return ConfidenceHigh
	}
	if prevalence <= 0.02 && support >= 20 {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

func PrintText(candidates []Candidate) {
	if len(candidates) == 0 {
		fmt.Println("No candidates discovered with current thresholds.")
		return
	}
	for i, c := range candidates {
		if i > 0 {
			fmt.Println("---")
		}
		fmt.Printf("kind: %s\n", c.Kind)
		fmt.Printf("scope: %v\n", c.Scope)
		if len(c.Target) > 0 {
			fmt.Printf("target: %v\n", c.Target)
		}
		fmt.Printf("severity: %s\n", c.Severity)
		fmt.Printf("support: %d\n", c.Support)
		fmt.Printf("violations: %d\n", c.Violations)
		fmt.Printf("prevalence: %.4f\n", c.Prevalence)
		fmt.Printf("confidence: %s\n", c.Confidence)
		fmt.Printf("evidence: %s\n", c.Evidence)
	}
}

func PrintJSON(candidates []Candidate) {
	data, _ := json.MarshalIndent(candidates, "", "  ")
	fmt.Println(string(data))
}

func PrintYAML(candidates []Candidate) {
	fmt.Println("version: 1")
	fmt.Println("rules:")
	for i, c := range candidates {
		fmt.Printf("  - id: MINED-%03d\n", i+1)
		fmt.Printf("    kind: %s\n", c.Kind)
		fmt.Printf("    severity: %s\n", c.Severity)
		fmt.Printf("    scope:\n")
		for _, s := range c.Scope {
			fmt.Printf("      - %q\n", s)
		}
		if len(c.Target) > 0 {
			fmt.Printf("    target:\n")
			for _, t := range c.Target {
				fmt.Printf("      - %q\n", t)
			}
		}
		fmt.Printf("    message: %q\n", c.Evidence)
	}
}

func EmitStarterConfig(candidates []Candidate) string {
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
	return b.String()
}
