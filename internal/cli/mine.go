package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/catalog"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/framework"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/language"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pathutil"
	"github.com/honzikec/archguard/internal/workspace"
)

func runMine(args []string) int {
	fs := flag.NewFlagSet("mine", flag.ContinueOnError)
	setFlagSetOutput(fs)
	common := bindCommonFlags(fs, commonFlags{configPath: "archguard.yaml", format: "text"})
	minSupport := fs.Int("min-support", 20, "Minimum files per scope to propose candidate")
	maxPrevalence := fs.Float64("max-prevalence", 0.02, "Maximum prevalence to consider as invariant")
	maxCandidatesPerKind := fs.Int("max-candidates-per-kind", 200, "Maximum mined candidates per kind (0 = unlimited)")
	emitConfig := fs.Bool("emit-config", false, "Emit a starter config from mined candidates")
	emitNoCycleSeverity := fs.String("emit-no-cycle-severity", "warning", "Severity for emitted no_cycle rules: warning|error")
	workspaceMode := fs.String("workspace-mode", "auto", "Workspace mining mode: auto|off")
	catalogMode := fs.String("catalog", "builtin", "Catalog mode: builtin|off")
	catalogFormat := fs.String("catalog-format", "", "Catalog output format: text|json (default follows --format)")
	adoptCatalog := fs.Bool("adopt-catalog", false, "Include adopted catalog rules when used with --emit-config")
	adoptThreshold := fs.String("adopt-threshold", "high", "Catalog adoption threshold: high|medium")
	showLowConfidence := fs.Bool("show-low-confidence", false, "Include low-confidence catalog matches")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if common.format != "text" && common.format != "yaml" && common.format != "json" {
		fmt.Fprintf(os.Stderr, "unsupported format: %s\n", common.format)
		return 2
	}
	if *catalogMode != "builtin" && *catalogMode != "off" {
		fmt.Fprintf(os.Stderr, "unsupported catalog mode: %s\n", *catalogMode)
		return 2
	}
	if *catalogFormat == "" {
		if common.format == "json" {
			*catalogFormat = "json"
		} else {
			*catalogFormat = "text"
		}
	}
	if *catalogFormat != "text" && *catalogFormat != "json" {
		fmt.Fprintf(os.Stderr, "unsupported catalog-format: %s\n", *catalogFormat)
		return 2
	}
	if *adoptThreshold != "high" && *adoptThreshold != "medium" {
		fmt.Fprintf(os.Stderr, "unsupported adopt-threshold: %s\n", *adoptThreshold)
		return 2
	}
	if *emitNoCycleSeverity != "warning" && *emitNoCycleSeverity != "error" {
		fmt.Fprintf(os.Stderr, "unsupported emit-no-cycle-severity: %s\n", *emitNoCycleSeverity)
		return 2
	}
	if *workspaceMode != "auto" && *workspaceMode != "off" {
		fmt.Fprintf(os.Stderr, "unsupported workspace-mode: %s\n", *workspaceMode)
		return 2
	}

	cfg, err := loadConfigOptional(common.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	languageResolution := language.Resolve(cfg.Project.Language, cfg.Project.Roots)
	if languageResolution.Adapter == nil {
		fmt.Fprintln(os.Stderr, "failed to resolve language adapter")
		return 2
	}

	files, err := fileset.DiscoverWithAdapter(cfg.Project, languageResolution.Adapter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to discover files: %v\n", err)
		return 2
	}
	resolver, err := pathutil.NewResolver(".", cfg.Project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize resolver: %v\n", err)
		return 2
	}

	imports := make([]model.ImportRef, 0)
	for _, file := range files {
		parsed, err := languageResolution.Adapter.ParseFile(file)
		if err != nil {
			if common.debug {
				fmt.Fprintf(os.Stderr, "parse error %s: %v\n", file, err)
			}
			continue
		}
		for i := range parsed {
			resolved, isPackage := resolver.Resolve(parsed[i].SourceFile, parsed[i].RawImport)
			parsed[i].ResolvedPath = resolved
			parsed[i].IsPackageImport = isPackage
		}
		imports = append(imports, parsed...)
	}

	workspaceRoots := append([]string{}, cfg.Project.Roots...)
	workspaceReason := "off"
	if *workspaceMode == "auto" {
		discovered, err := workspace.DiscoverRoots(cfg.Project.Roots)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to discover workspaces: %v\n", err)
			return 2
		}
		if len(discovered) > 0 {
			workspaceRoots = discovered
		}
		if len(workspaceRoots) > 1 {
			workspaceReason = "auto_workspaces"
		} else {
			workspaceReason = "auto_single"
		}
	}
	if len(workspaceRoots) == 0 {
		workspaceRoots = []string{"."}
	}

	rootFrameworkResolution := framework.Resolve(cfg.Project.Framework, cfg.Project.Roots)
	frameworkResolution := rootFrameworkResolution
	candidateBuckets := make([]miner.Candidate, 0)
	totalNormalization := miner.MineNormalizationStats{}
	var debugStats *miner.DebugStats
	if common.debug {
		debugStats = miner.NewDebugStats()
	}

	for _, wsRoot := range workspaceRoots {
		wsFiles := filterFilesByWorkspace(files, wsRoot)
		if len(wsFiles) == 0 {
			continue
		}
		wsImports := filterImportsByWorkspace(imports, wsRoot)
		wsGraph := graph.Build(wsImports, wsFiles)
		wsFramework := framework.Resolve(cfg.Project.Framework, []string{wsRoot})
		if len(workspaceRoots) == 1 {
			frameworkResolution = wsFramework
		}
		normalizedGraph, normalizedFiles, stats := framework.NormalizeMiningInputs(wsGraph, wsFiles, wsFramework.Selected)
		wsCandidates := miner.Propose(normalizedGraph, normalizedFiles, miner.Options{
			MinSupport:           *minSupport,
			MaxPrevalence:        *maxPrevalence,
			MaxCandidatesPerKind: *maxCandidatesPerKind,
			DebugStats:           debugStats,
		})
		candidateBuckets = append(candidateBuckets, wsCandidates...)
		totalNormalization.OriginalNodes += stats.OriginalNodes
		totalNormalization.NormalizedNodes += stats.NormalizedNodes
		totalNormalization.OriginalFiles += stats.OriginalFiles
		totalNormalization.NormalizedFiles += stats.NormalizedFiles
	}
	candidates := dedupeCandidates(candidateBuckets, *maxCandidatesPerKind)
	catalogMatches := make([]miner.PatternMatch, 0)

	if *catalogMode == "builtin" {
		patterns, err := catalog.LoadBuiltin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load built-in catalog: %v\n", err)
			return 2
		}
		catalogMatches, err = miner.MatchCatalog(patterns, candidates, files, cfg.Project, miner.CatalogOptions{
			ShowLowConfidence: *showLowConfidence,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to match catalog patterns: %v\n", err)
			return 2
		}
	}

	metadata := miner.MineMetadata{
		FrameworkProfile: frameworkResolution.EffectiveProfile(),
		FrameworkReason:  frameworkResolution.Reason,
		FrameworkMatched: append([]string{}, frameworkResolution.Matched...),
		LanguageAdapter:  languageResolution.Selected,
		LanguageReason:   languageResolution.Reason,
		Normalization: miner.MineNormalizationStats{
			OriginalNodes:   totalNormalization.OriginalNodes,
			NormalizedNodes: totalNormalization.NormalizedNodes,
			OriginalFiles:   totalNormalization.OriginalFiles,
			NormalizedFiles: totalNormalization.NormalizedFiles,
		},
	}

	if common.debug {
		fmt.Fprintf(os.Stderr, "mine framework profile: %s (%s)\n", metadata.FrameworkProfile, metadata.FrameworkReason)
		if len(metadata.FrameworkMatched) > 0 {
			sorted := append([]string{}, metadata.FrameworkMatched...)
			sort.Strings(sorted)
			fmt.Fprintf(os.Stderr, "mine framework matches: %s\n", strings.Join(sorted, ", "))
		}
		fmt.Fprintf(os.Stderr, "mine language adapter: %s (%s)\n", metadata.LanguageAdapter, metadata.LanguageReason)
		fmt.Fprintf(os.Stderr, "mine workspaces: %d (%s)\n", len(workspaceRoots), workspaceReason)
		fmt.Fprintf(os.Stderr, "mine normalization: nodes %d->%d files %d->%d\n",
			metadata.Normalization.OriginalNodes,
			metadata.Normalization.NormalizedNodes,
			metadata.Normalization.OriginalFiles,
			metadata.Normalization.NormalizedFiles,
		)
		if debugStats != nil && len(debugStats.Dropped) > 0 {
			keys := make([]string, 0, len(debugStats.Dropped))
			for key := range debugStats.Dropped {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(os.Stderr, "mine dropped %s=%d\n", key, debugStats.Dropped[key])
			}
		}
	}

	if *emitConfig {
		adopted := []config.Rule{}
		if *adoptCatalog {
			adopted = miner.AdoptCatalogMatches(catalogMatches, *adoptThreshold)
		}
		fmt.Print(miner.EmitStarterConfigWithCatalog(candidates, adopted, miner.EmitOptions{
			NoCycleSeverity: *emitNoCycleSeverity,
		}))
		return 0
	}

	if len(candidates) == 0 && len(catalogMatches) == 0 && common.format == "text" {
		fmt.Printf("No candidates discovered (min_support=%d, max_prevalence=%.4f).\n", *minSupport, *maxPrevalence)
		return 0
	}

	switch common.format {
	case "yaml":
		miner.PrintYAML(candidates)
	case "json":
		miner.PrintMineJSON(candidates, catalogMatches, metadata)
	default:
		miner.PrintMineText(candidates, catalogMatches, *catalogFormat, common.debug, metadata)
	}
	return 0
}

func filterFilesByWorkspace(files []string, wsRoot string) []string {
	wsRoot = strings.TrimSuffix(pathutil.Normalize(strings.TrimSpace(wsRoot)), "/")
	if wsRoot == "" || wsRoot == "." {
		return append([]string{}, files...)
	}
	prefix := wsRoot + "/"
	out := make([]string, 0)
	for _, f := range files {
		if strings.HasPrefix(pathutil.Normalize(f), prefix) {
			out = append(out, f)
		}
	}
	return out
}

func filterImportsByWorkspace(imports []model.ImportRef, wsRoot string) []model.ImportRef {
	wsRoot = strings.TrimSuffix(pathutil.Normalize(strings.TrimSpace(wsRoot)), "/")
	if wsRoot == "" || wsRoot == "." {
		return append([]model.ImportRef{}, imports...)
	}
	prefix := wsRoot + "/"
	out := make([]model.ImportRef, 0)
	for _, imp := range imports {
		if strings.HasPrefix(pathutil.Normalize(imp.SourceFile), prefix) {
			out = append(out, imp)
		}
	}
	return out
}

func dedupeCandidates(in []miner.Candidate, maxPerKind int) []miner.Candidate {
	type entry struct {
		candidate miner.Candidate
	}
	seen := map[string]entry{}
	for _, c := range in {
		key := strings.Join([]string{
			c.Kind,
			strings.Join(c.Scope, ","),
			strings.Join(c.Target, ","),
			c.Severity,
		}, "|")
		current, ok := seen[key]
		if !ok || c.Support > current.candidate.Support {
			seen[key] = entry{candidate: c}
		}
	}
	merged := make([]miner.Candidate, 0, len(seen))
	for _, e := range seen {
		merged = append(merged, e.candidate)
	}

	grouped := map[string][]miner.Candidate{}
	for _, c := range merged {
		grouped[c.Kind] = append(grouped[c.Kind], c)
	}

	out := make([]miner.Candidate, 0, len(merged))
	for kind := range grouped {
		candidates := grouped[kind]
		sort.Slice(candidates, func(i, j int) bool {
			if confidenceRank(candidates[i].Confidence) != confidenceRank(candidates[j].Confidence) {
				return confidenceRank(candidates[i].Confidence) > confidenceRank(candidates[j].Confidence)
			}
			if candidates[i].Support != candidates[j].Support {
				return candidates[i].Support > candidates[j].Support
			}
			if candidates[i].Prevalence != candidates[j].Prevalence {
				return candidates[i].Prevalence < candidates[j].Prevalence
			}
			if candidates[i].Violations != candidates[j].Violations {
				return candidates[i].Violations < candidates[j].Violations
			}
			left, right := "", ""
			if len(candidates[i].Scope) > 0 {
				left = candidates[i].Scope[0]
			}
			if len(candidates[j].Scope) > 0 {
				right = candidates[j].Scope[0]
			}
			return left < right
		})
		if maxPerKind > 0 && len(candidates) > maxPerKind {
			candidates = candidates[:maxPerKind]
		}
		out = append(out, candidates...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if len(out[i].Scope) == 0 || len(out[j].Scope) == 0 {
			return len(out[i].Scope) < len(out[j].Scope)
		}
		return out[i].Scope[0] < out[j].Scope[0]
	})
	return out
}

func confidenceRank(confidence string) int {
	switch strings.ToUpper(strings.TrimSpace(confidence)) {
	case "HIGH":
		return 3
	case "MEDIUM":
		return 2
	case "LOW":
		return 1
	default:
		return 0
	}
}

func loadConfigOptional(path string) (*config.Config, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return &config.Config{Version: 1, Project: config.DefaultProjectSettings(), Rules: []config.Rule{}}, nil
		}
		return nil, err
	}
	return config.Load(path)
}
