package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/honzikec/archguard/internal/analysis"
	"github.com/honzikec/archguard/internal/baseline"
	"github.com/honzikec/archguard/internal/catalog"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/framework"
	"github.com/honzikec/archguard/internal/language"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/policy"
	"github.com/honzikec/archguard/internal/workspace"
)

type guidedPreset struct {
	ID      string
	Title   string
	Summary string
}

func runInitGuided(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	setFlagSetOutput(fs)
	common := bindCommonFlags(fs, commonFlags{configPath: "archguard.yaml", format: "text"})
	writeConfig := fs.Bool("write-config", false, "Write the guided starter config to disk")
	writeBaseline := fs.Bool("write-baseline", false, "Write a baseline for current findings under the guided config")
	adoptThreshold := fs.String("adopt-catalog-threshold", "high", "Catalog adoption threshold: high|medium")
	ciMode := fs.String("ci-mode", "enforce", "CI mode for printed GitHub Action snippet: enforce|audit")
	outPath := fs.String("out", "", "Output path for generated config (defaults to --config)")
	baselineOut := fs.String("baseline-out", "archguard-baseline.json", "Baseline output path when using --write-baseline")
	force := fs.Bool("force", false, "Overwrite existing generated config file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *adoptThreshold != "high" && *adoptThreshold != "medium" {
		fmt.Fprintf(os.Stderr, "unsupported adopt-catalog-threshold: %s\n", *adoptThreshold)
		return 2
	}
	if *ciMode != "enforce" && *ciMode != "audit" {
		fmt.Fprintf(os.Stderr, "unsupported ci-mode: %s\n", *ciMode)
		return 2
	}

	loadPath := common.configPath
	targetPath := strings.TrimSpace(*outPath)
	if targetPath == "" {
		targetPath = loadPath
	}

	configPath, configDir, err := resolveConfigPath(loadPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve config path: %v\n", err)
		return 2
	}
	cfg, err := loadConfigOptional(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	code, runErr := withWorkingDir(configDir, func() int {
		recommendedProject, recommendations, preset, findings, starterConfig, starterText, err := buildGuidedInitRecommendation(cfg, *adoptThreshold)
		if err != nil {
			fmt.Fprintf(os.Stderr, "guided init failed: %v\n", err)
			return 2
		}

		if !common.quiet {
			printGuidedSummary(recommendations, preset, findings, starterText, targetPath, *baselineOut, *ciMode, *writeConfig, *writeBaseline)
		}

		if *writeConfig {
			if _, err := os.Stat(targetPath); err == nil && !*force {
				fmt.Fprintf(os.Stderr, "config already exists: %s (use --force to overwrite)\n", targetPath)
				return 2
			}
			if dir := filepath.Dir(targetPath); dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					fmt.Fprintf(os.Stderr, "failed to create config directory: %v\n", err)
					return 2
				}
			}
			if err := os.WriteFile(targetPath, []byte(starterText), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write config: %v\n", err)
				return 2
			}
			fmt.Printf("Created %s\n", targetPath)
		}

		if *writeBaseline {
			if len(findings) == 0 {
				fmt.Println("No findings under the guided starter config. Baseline not written.")
			} else {
				if err := baseline.Write(*baselineOut, findings, nowUTC()); err != nil {
					fmt.Fprintf(os.Stderr, "failed to write baseline: %v\n", err)
					return 2
				}
				fmt.Printf("Created %s (%d finding(s))\n", *baselineOut, len(findings))
			}
		}

		_ = recommendedProject
		_ = starterConfig
		return 0
	})
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "failed to set working directory: %v\n", runErr)
		return 2
	}
	return code
}

type guidedRecommendations struct {
	Language       language.Resolution
	Framework      framework.Resolution
	WorkspaceRoots []string
	Project        config.ProjectSettings
	Notes          []string
}

func buildGuidedInitRecommendation(cfg *config.Config, adoptThreshold string) (config.ProjectSettings, guidedRecommendations, guidedPreset, []model.Finding, *config.Config, string, error) {
	workspaceRoots := append([]string{}, cfg.Project.Roots...)
	recommendedProject := cloneProjectSettings(cfg.Project)
	if recommendedProject.Aliases == nil {
		recommendedProject.Aliases = map[string][]string{}
	}
	initialLanguage := language.Resolve(cfg.Project.Language, cfg.Project.Roots)
	if initialLanguage.Adapter == nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", fmt.Errorf("failed to resolve language adapter")
	}
	initialFramework := framework.Resolve(cfg.Project.Framework, cfg.Project.Roots)
	preset := detectGuidedPreset(recommendedProject, initialLanguage, initialFramework, workspaceRoots)

	if preset.ID != phpBrownfieldPresetID {
		discovered, err := workspace.DiscoverRoots(cfg.Project.Roots)
		if err != nil {
			return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", err
		}
		if len(discovered) > 1 && rootsAreDefault(cfg.Project.Roots) {
			workspaceRoots = discovered
		}
		preset = detectGuidedPreset(recommendedProject, initialLanguage, initialFramework, workspaceRoots)
	}
	if len(workspaceRoots) > 0 {
		recommendedProject.Roots = workspaceRoots
	}

	notes := applyGuidedProjectDefaults(&recommendedProject, preset)

	languageResolution := language.Resolve(recommendedProject.Language, recommendedProject.Roots)
	if languageResolution.Adapter == nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", fmt.Errorf("failed to resolve language adapter")
	}
	frameworkResolution := framework.Resolve(recommendedProject.Framework, recommendedProject.Roots)

	recommendedProject.Language = recommendedLanguage(cfg.Project.Language, languageResolution)
	recommendedProject.Framework = recommendedFramework(cfg.Project.Framework, frameworkResolution)
	if preset.ID != phpBrownfieldPresetID && recommendedProject.Tsconfig == "" && fileExists("tsconfig.json") {
		recommendedProject.Tsconfig = "tsconfig.json"
	}

	result, err := analysis.Run(recommendedProject, languageResolution.Adapter, analysis.Options{})
	if err != nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", err
	}

	normalizedGraph, normalizedFiles, _ := framework.NormalizeMiningInputs(result.Graph, result.Files, frameworkResolution.EffectiveProfile())
	minSupport := 3
	maxPrevalence := 0.02
	if len(normalizedFiles) < 20 {
		minSupport = 1
		maxPrevalence = 0.25
	}
	candidates := miner.Propose(normalizedGraph, normalizedFiles, miner.Options{
		MinSupport:           minSupport,
		MaxPrevalence:        maxPrevalence,
		MaxCandidatesPerKind: 50,
	})
	patterns, err := catalog.LoadBuiltin()
	if err != nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", err
	}
	catalogMatches, err := miner.MatchCatalog(patterns, candidates, result.Files, cfg.Project, miner.CatalogOptions{})
	if err != nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", err
	}
	adopted := miner.AdoptCatalogMatches(catalogMatches, adoptThreshold)
	candidates, adopted, notes = applyPresetDefaults(candidates, adopted, preset, recommendedProject, notes)

	starterConfig := miner.BuildStarterConfigWithCatalog(candidates, adopted, miner.EmitOptions{
		NoCycleSeverity: config.SeverityWarning,
		Project:         &recommendedProject,
	})
	findings, err := policy.Evaluate(starterConfig, result.Imports, result.Files, result.Graph)
	if err != nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", err
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Fingerprint != findings[j].Fingerprint {
			return findings[i].Fingerprint < findings[j].Fingerprint
		}
		return findings[i].FilePath < findings[j].FilePath
	})
	starterText := miner.EmitStarterConfigWithCatalog(candidates, adopted, miner.EmitOptions{
		NoCycleSeverity: config.SeverityWarning,
		Project:         &recommendedProject,
	})
	recommendations := guidedRecommendations{
		Language:       languageResolution,
		Framework:      frameworkResolution,
		WorkspaceRoots: workspaceRoots,
		Project:        recommendedProject,
		Notes:          notes,
	}
	return recommendedProject, recommendations, preset, findings, starterConfig, starterText, nil
}

func printGuidedSummary(recommendations guidedRecommendations, preset guidedPreset, findings []model.Finding, starterText, targetPath, baselinePath, ciMode string, writeConfig, writeBaseline bool) {
	fmt.Println("Guided init summary")
	fmt.Printf("Recommended language: %s (%s)\n", recommendations.Project.Language, recommendations.Language.Reason)
	fmt.Printf("Recommended framework: %s (%s)\n", recommendations.Project.Framework, recommendations.Framework.Reason)
	fmt.Printf("Recommended roots: %v\n", recommendations.Project.Roots)
	if recommendations.Project.Tsconfig != "" {
		fmt.Printf("Recommended tsconfig: %s\n", recommendations.Project.Tsconfig)
	}
	if preset.ID != "" {
		fmt.Printf("Preset: %s - %s\n", preset.Title, preset.Summary)
	}
	if len(recommendations.Notes) > 0 {
		fmt.Println("Notes:")
		for _, note := range recommendations.Notes {
			fmt.Printf("- %s\n", note)
		}
	}
	fmt.Printf("Starter config findings today: %d\n", len(findings))
	fmt.Println()
	fmt.Printf("Config preview for %s:\n", targetPath)
	fmt.Println(starterText)
	if writeConfig || writeBaseline {
		fmt.Println("Write plan:")
		if writeConfig {
			fmt.Printf("- config: %s\n", targetPath)
		}
		if writeBaseline {
			fmt.Printf("- baseline: %s\n", baselinePath)
		}
		fmt.Println()
	}
	fmt.Println("Local commands:")
	fmt.Println("  archguard check --config archguard.yaml --changed-only --format text")
	fmt.Println("  archguard check --config archguard.yaml --format sarif --parse-error-policy error > archguard-results.sarif")
	fmt.Println()
	fmt.Println("GitHub Action:")
	fmt.Println(githubActionSnippet(ciMode))
}

func githubActionSnippet(ciMode string) string {
	enforce := "true"
	if ciMode == "audit" {
		enforce = "false"
	}
	return fmt.Sprintf(`- name: ArchGuard
  uses: honzikec/archguard-action@v1
  with:
    config: archguard.yaml
    format: sarif
    parse-error-policy: error
    severity-threshold: error
    enforce: %s
    upload-sarif: true`, enforce)
}

func recommendedLanguage(current string, resolution language.Resolution) string {
	current = strings.TrimSpace(current)
	if current != "" && current != "auto" {
		return current
	}
	if strings.TrimSpace(resolution.Selected) == "" {
		return "auto"
	}
	return resolution.Selected
}

func recommendedFramework(current string, resolution framework.Resolution) string {
	current = strings.TrimSpace(current)
	if current != "" && current != "generic" {
		return current
	}
	return resolution.EffectiveProfile()
}

func rootsAreDefault(roots []string) bool {
	return len(roots) == 0 || (len(roots) == 1 && strings.TrimSpace(roots[0]) == ".")
}

func cloneProjectSettings(project config.ProjectSettings) config.ProjectSettings {
	out := project
	out.Roots = append([]string{}, project.Roots...)
	out.Include = append([]string{}, project.Include...)
	out.Exclude = append([]string{}, project.Exclude...)
	if project.Aliases != nil {
		out.Aliases = make(map[string][]string, len(project.Aliases))
		for key, value := range project.Aliases {
			out.Aliases[key] = append([]string{}, value...)
		}
	}
	return out
}

const phpBrownfieldPresetID = "php-yii-brownfield"

func detectGuidedPreset(project config.ProjectSettings, languageResolution language.Resolution, frameworkResolution framework.Resolution, workspaceRoots []string) guidedPreset {
	if strings.TrimSpace(languageResolution.Selected) == "php" && fileExists("composer.json") && looksLikePHPBrownfieldRepo() {
		return guidedPreset{
			ID:      phpBrownfieldPresetID,
			Title:   "PHP Brownfield (Yii-style Monolith)",
			Summary: "Bias starter guidance toward PHP app roots, cross-area boundaries, and low-noise onboarding.",
		}
	}
	if frameworkResolution.EffectiveProfile() == "nextjs" || fileExists("app") {
		return guidedPreset{
			ID:      "nextjs-app-router",
			Title:   "Next.js App Router",
			Summary: "Bias starter guidance toward app-router boundaries and staged warning-first adoption.",
		}
	}
	if len(workspaceRoots) > 1 {
		return guidedPreset{
			ID:      "react-shared-packages",
			Title:   "React Shared Packages",
			Summary: "Bias starter guidance toward workspace-aware roots and shared package boundaries.",
		}
	}
	if fileExists("src/domain") || fileExists("src/infra") || fileExists("src/services") {
		return guidedPreset{
			ID:      "layered-node-service",
			Title:   "Layered Node Service",
			Summary: "Bias starter guidance toward domain, infra, and service-layer separation.",
		}
	}
	return guidedPreset{}
}

func applyPresetDefaults(candidates []miner.Candidate, adopted []config.Rule, preset guidedPreset, project config.ProjectSettings, notes []string) ([]miner.Candidate, []config.Rule, []string) {
	switch preset.ID {
	case phpBrownfieldPresetID:
		notes = appendUniqueStrings(notes,
			"PHP-first starter config: JS/tooling areas were intentionally excluded from the first pass.",
			"Guided mining was filtered for low-noise onboarding: trivial file patterns, package-style PHP namespace rules, and broad repo-wide cycles were omitted.",
			"If the starter config feels too light, add rules manually after you validate the narrowed roots on real changes.",
		)
		return refinePHPBrownfieldGuidedCandidates(candidates, project), nil, notes
	case "nextjs-app-router", "react-shared-packages", "layered-node-service":
		for i := range candidates {
			if candidates[i].Confidence != miner.ConfidenceHigh {
				candidates[i].Severity = config.SeverityWarning
			}
		}
		for i := range adopted {
			if adopted[i].Severity == config.SeverityError {
				adopted[i].Severity = config.SeverityWarning
			}
		}
	}
	return candidates, adopted, notes
}

func applyGuidedProjectDefaults(project *config.ProjectSettings, preset guidedPreset) []string {
	if project == nil {
		return nil
	}
	switch preset.ID {
	case phpBrownfieldPresetID:
		project.Language = "php"
		project.Framework = "generic"
		project.Include = []string{"**/*.php", "**/*.phtml"}
		project.Exclude = mergeStringLists(project.Exclude, []string{
			"**/vendor/**",
			"**/runtime/**",
			"**/node_modules/**",
			"**/tests/**",
			"**/docs/**",
			"**/_assets/**",
			"**/_build/**",
			"**/_db/**",
			"**/build/**",
			"**/scripts/**",
			"**/gulp/**",
		})
		project.Tsconfig = ""
		project.Aliases = map[string][]string{}
		if roots := discoverPHPBrownfieldRoots(); len(roots) > 0 {
			project.Roots = roots
		}
		return []string{
			"Recommended roots were narrowed to PHP application areas with supported files.",
			"Generated rules prefer cross-area boundaries and local cycles over template naming or namespace/package noise.",
		}
	}
	return nil
}

func looksLikePHPBrownfieldRepo() bool {
	if fileExists("common") && fileExists("frontend") && fileExists("backend") {
		return true
	}
	for _, versioned := range topLevelVersionedDirs() {
		if fileExists(filepath.Join(versioned, "common")) && fileExists(filepath.Join(versioned, "frontend")) && fileExists(filepath.Join(versioned, "backend")) {
			return true
		}
	}
	return false
}

func discoverPHPBrownfieldRoots() []string {
	preferred := []string{"common", "frontend", "backend", "api", "console"}
	roots := make([]string, 0)
	for _, name := range preferred {
		if hasSupportedPHPFiles(name) {
			roots = append(roots, name)
		}
	}
	for _, versioned := range topLevelVersionedDirs() {
		for _, name := range preferred {
			candidate := filepath.Join(versioned, name)
			if hasSupportedPHPFiles(candidate) {
				roots = append(roots, filepath.ToSlash(candidate))
			}
		}
	}
	return dedupeSortedStrings(roots)
}

func topLevelVersionedDirs() []string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil
	}
	out := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(name), "v") && len(name) > 1 {
			if _, err := strconv.Atoi(name[1:]); err == nil {
				out = append(out, name)
			}
		}
	}
	sort.Strings(out)
	return out
}

func hasSupportedPHPFiles(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == "runtime" || base == "node_modules" || base == "tests" || base == "docs" || base == "_assets" || base == "_build" || base == "_db" || base == "build" || base == "scripts" || base == "gulp" {
				return filepath.SkipDir
			}
			return nil
		}
		lower := strings.ToLower(path)
		if strings.HasSuffix(lower, ".php") || strings.HasSuffix(lower, ".phtml") {
			found = true
		}
		return nil
	})
	return found
}

func mergeStringLists(existing, extra []string) []string {
	return dedupeSortedStrings(append(append([]string{}, existing...), extra...))
}

func dedupeSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(filepath.ToSlash(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func appendUniqueStrings(values []string, extras ...string) []string {
	return mergeStringLists(values, extras)
}

func refinePHPBrownfieldGuidedCandidates(candidates []miner.Candidate, project config.ProjectSettings) []miner.Candidate {
	allowedAreas := phpBrownfieldAllowedAreas(project.Roots)
	if len(allowedAreas) == 0 {
		return nil
	}

	collapsedImports := map[string]miner.Candidate{}
	keptCycles := make([]miner.Candidate, 0)

	for _, candidate := range candidates {
		switch candidate.Kind {
		case config.KindNoImport:
			scopeArea := phpBrownfieldAreaRoot(scopeFromCandidate(candidate), allowedAreas)
			targetArea := phpBrownfieldAreaRoot(targetFromCandidate(candidate), allowedAreas)
			if scopeArea == "" || targetArea == "" || scopeArea == targetArea {
				continue
			}
			collapsed := candidate
			collapsed.Scope = []string{scopeArea + "/**"}
			collapsed.Target = []string{targetArea + "/**"}
			collapsed.Severity = config.SeverityWarning
			collapsed.Evidence = fmt.Sprintf("%d/%d files in %s import %s", candidate.Violations, candidate.Support, scopeArea, targetArea)
			key := scopeArea + "->" + targetArea
			if existing, ok := collapsedImports[key]; !ok || preferPHPBrownfieldCandidate(collapsed, existing) {
				collapsedImports[key] = collapsed
			}
		case config.KindNoCycle:
			if candidate.Violations > 4 {
				continue
			}
			scope := strings.TrimSuffix(scopeFromCandidate(candidate), "/**")
			if pathDepth(scope) < 2 {
				continue
			}
			keptCycles = append(keptCycles, withWarningSeverity(candidate))
		}
	}

	imports := make([]miner.Candidate, 0, len(collapsedImports))
	for _, candidate := range collapsedImports {
		imports = append(imports, candidate)
	}
	sort.Slice(imports, func(i, j int) bool {
		if imports[i].Violations != imports[j].Violations {
			return imports[i].Violations > imports[j].Violations
		}
		if imports[i].Support != imports[j].Support {
			return imports[i].Support > imports[j].Support
		}
		if imports[i].Scope[0] != imports[j].Scope[0] {
			return imports[i].Scope[0] < imports[j].Scope[0]
		}
		return imports[i].Target[0] < imports[j].Target[0]
	})
	sort.Slice(keptCycles, func(i, j int) bool {
		if keptCycles[i].Violations != keptCycles[j].Violations {
			return keptCycles[i].Violations < keptCycles[j].Violations
		}
		if keptCycles[i].Support != keptCycles[j].Support {
			return keptCycles[i].Support > keptCycles[j].Support
		}
		return scopeFromCandidate(keptCycles[i]) < scopeFromCandidate(keptCycles[j])
	})

	out := make([]miner.Candidate, 0, len(imports)+len(keptCycles))
	for _, candidate := range imports {
		out = append(out, candidate)
		if len(out) >= 12 {
			return out[:12]
		}
	}
	for _, candidate := range keptCycles {
		out = append(out, candidate)
		if len(out) >= 12 {
			return out[:12]
		}
	}
	return out
}

func phpBrownfieldAllowedAreas(roots []string) []string {
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(filepath.ToSlash(root))
		root = strings.Trim(root, "/")
		if root == "" || root == "." {
			continue
		}
		out = append(out, root)
	}
	sort.Slice(out, func(i, j int) bool {
		return len(out[i]) > len(out[j])
	})
	return dedupePreserveOrder(out)
}

func phpBrownfieldAreaRoot(pathValue string, allowedAreas []string) string {
	pathValue = strings.TrimSpace(filepath.ToSlash(pathValue))
	pathValue = strings.TrimSuffix(pathValue, "/**")
	pathValue = strings.Trim(pathValue, "/")
	for _, area := range allowedAreas {
		if pathValue == area || strings.HasPrefix(pathValue, area+"/") {
			return area
		}
	}
	return ""
}

func scopeFromCandidate(candidate miner.Candidate) string {
	if len(candidate.Scope) == 0 {
		return ""
	}
	return candidate.Scope[0]
}

func targetFromCandidate(candidate miner.Candidate) string {
	if len(candidate.Target) == 0 {
		return ""
	}
	return candidate.Target[0]
}

func preferPHPBrownfieldCandidate(left, right miner.Candidate) bool {
	if left.Violations != right.Violations {
		return left.Violations > right.Violations
	}
	if left.Support != right.Support {
		return left.Support > right.Support
	}
	return left.Prevalence < right.Prevalence
}

func withWarningSeverity(candidate miner.Candidate) miner.Candidate {
	candidate.Severity = config.SeverityWarning
	return candidate
}

func pathDepth(pathValue string) int {
	pathValue = strings.Trim(strings.TrimSpace(pathValue), "/")
	if pathValue == "" || pathValue == "." {
		return 0
	}
	return strings.Count(pathValue, "/") + 1
}

func dedupePreserveOrder(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
