package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
}

func buildGuidedInitRecommendation(cfg *config.Config, adoptThreshold string) (config.ProjectSettings, guidedRecommendations, guidedPreset, []model.Finding, *config.Config, string, error) {
	languageResolution := language.Resolve(cfg.Project.Language, cfg.Project.Roots)
	if languageResolution.Adapter == nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", fmt.Errorf("failed to resolve language adapter")
	}
	frameworkResolution := framework.Resolve(cfg.Project.Framework, cfg.Project.Roots)

	result, err := analysis.Run(cfg.Project, languageResolution.Adapter, analysis.Options{})
	if err != nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", err
	}

	workspaceRoots := append([]string{}, cfg.Project.Roots...)
	discovered, err := workspace.DiscoverRoots(cfg.Project.Roots)
	if err != nil {
		return config.ProjectSettings{}, guidedRecommendations{}, guidedPreset{}, nil, nil, "", err
	}
	if len(discovered) > 1 && rootsAreDefault(cfg.Project.Roots) {
		workspaceRoots = discovered
	}

	recommendedProject := cloneProjectSettings(cfg.Project)
	if len(workspaceRoots) > 0 {
		recommendedProject.Roots = workspaceRoots
	}
	recommendedProject.Language = recommendedLanguage(cfg.Project.Language, languageResolution)
	recommendedProject.Framework = recommendedFramework(cfg.Project.Framework, frameworkResolution)
	if recommendedProject.Aliases == nil {
		recommendedProject.Aliases = map[string][]string{}
	}
	if recommendedProject.Tsconfig == "" && fileExists("tsconfig.json") {
		recommendedProject.Tsconfig = "tsconfig.json"
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
	applyPresetDefaults(&candidates, &adopted, detectGuidedPreset(recommendedProject, frameworkResolution, workspaceRoots))

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
	}
	preset := detectGuidedPreset(recommendedProject, frameworkResolution, workspaceRoots)
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

func detectGuidedPreset(project config.ProjectSettings, frameworkResolution framework.Resolution, workspaceRoots []string) guidedPreset {
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

func applyPresetDefaults(candidates *[]miner.Candidate, adopted *[]config.Rule, preset guidedPreset) {
	switch preset.ID {
	case "nextjs-app-router", "react-shared-packages", "layered-node-service":
		for i := range *candidates {
			if (*candidates)[i].Confidence != miner.ConfidenceHigh {
				(*candidates)[i].Severity = config.SeverityWarning
			}
		}
		for i := range *adopted {
			if (*adopted)[i].Severity == config.SeverityError {
				(*adopted)[i].Severity = config.SeverityWarning
			}
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
