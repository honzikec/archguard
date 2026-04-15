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
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/language"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/policy"
	"github.com/honzikec/archguard/internal/report"
)

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	setFlagSetOutput(fs)
	common := bindCommonFlags(fs, commonFlags{configPath: "archguard.yaml", format: "text"})
	changedOnly := fs.Bool("changed-only", false, "Analyze only changed files from git working tree")
	changedAgainst := fs.String("changed-against", "", "Analyze only files changed against a git ref (for example origin/main)")
	parseErrorPolicy := fs.String("parse-error-policy", "warn", "Parse/read error policy: warn|error")
	severityThreshold := fs.String("severity-threshold", "error", "Blocking threshold: warning|error")
	maxFindings := fs.Int("max-findings", 0, "Maximum findings to emit (0 = unlimited)")
	baselinePath := fs.String("baseline", "", "Suppress findings present in a baseline file")
	writeBaselinePath := fs.String("write-baseline", "", "Write current findings to a baseline file and exit successfully")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if common.format != "text" && common.format != "json" && common.format != "sarif" {
		fmt.Fprintf(os.Stderr, "unsupported format: %s\n", common.format)
		return 2
	}
	if *severityThreshold != "warning" && *severityThreshold != "error" {
		fmt.Fprintf(os.Stderr, "unsupported severity threshold: %s\n", *severityThreshold)
		return 2
	}
	if *parseErrorPolicy != "warn" && *parseErrorPolicy != "error" {
		fmt.Fprintf(os.Stderr, "unsupported parse-error-policy: %s\n", *parseErrorPolicy)
		return 2
	}
	if strings.TrimSpace(*baselinePath) != "" && strings.TrimSpace(*writeBaselinePath) != "" {
		fmt.Fprintln(os.Stderr, "--baseline cannot be combined with --write-baseline")
		return 2
	}

	started := time.Now()
	configPath, configDir, err := resolveConfigPath(common.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve config path: %v\n", err)
		return 2
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	effectiveRoots := resolveEffectiveRoots(configDir, cfg.Project.Roots)

	code, runErr := withWorkingDir(configDir, func() int {
		languageResolution := language.Resolve(cfg.Project.Language, cfg.Project.Roots)
		if languageResolution.Adapter == nil {
			fmt.Fprintf(os.Stderr, "failed to resolve language adapter\n")
			return 2
		}
		if common.debug {
			fmt.Fprintf(os.Stderr, "config dir: %s\n", filepath.ToSlash(filepath.Clean(configDir)))
			fmt.Fprintf(os.Stderr, "effective roots: %s\n", strings.Join(effectiveRoots, ", "))
			fmt.Fprintf(os.Stderr, "language adapter: %s (%s)\n", languageResolution.Selected, languageResolution.Reason)
		}

		var traceImports *os.File
		if common.debug {
			traceImports = os.Stderr
		}
		result, err := analysis.Run(cfg.Project, languageResolution.Adapter, analysis.Options{
			FileFilter: func(files []string) ([]string, error) {
				return filterChangedFiles(files, *changedOnly, *changedAgainst)
			},
			TraceImports: traceImports,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 2
		}
		if common.debug && result.Diagnostics.NonLiteralDynamicImports > 0 {
			fmt.Fprintf(os.Stderr, "ignored non-literal dynamic imports: %d\n", result.Diagnostics.NonLiteralDynamicImports)
		}
		if common.debug && result.Diagnostics.WorkspacePackageImports > 0 {
			fmt.Fprintf(os.Stderr, "workspace package imports resolved: %d\n", result.Diagnostics.WorkspacePackageImports)
		}
		if common.debug && result.Diagnostics.UnresolvedLocalImports > 0 {
			fmt.Fprintf(os.Stderr, "unresolved local-like imports: %d\n", result.Diagnostics.UnresolvedLocalImports)
		}
		if common.debug && result.Diagnostics.IgnoredResolutionCases > 0 {
			fmt.Fprintf(os.Stderr, "ignored resolution cases: %d\n", result.Diagnostics.IgnoredResolutionCases)
		}

		findings, err := policy.Evaluate(cfg, result.Imports, result.Files, result.Graph)
		if err != nil {
			fmt.Fprintf(os.Stderr, "policy evaluation failed: %v\n", err)
			return 2
		}

		suppressedFindings := 0
		if path := strings.TrimSpace(*baselinePath); path != "" {
			entries, err := baseline.Load(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load baseline: %v\n", err)
				return 2
			}
			findings, suppressedFindings = baseline.Filter(findings, entries)
		}
		if path := strings.TrimSpace(*writeBaselinePath); path != "" {
			if err := baseline.Write(path, findings, time.Now()); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write baseline: %v\n", err)
				return 2
			}
			suppressedFindings = len(findings)
			fmt.Fprintf(os.Stderr, "baseline written: %s (%d finding(s))\n", path, suppressedFindings)
			findings = nil
		}

		if *maxFindings > 0 && len(findings) > *maxFindings {
			findings = findings[:*maxFindings]
		}

		summary := buildSummary(findings, len(result.Files), len(result.Imports), result.Diagnostics, suppressedFindings, int(time.Since(started).Milliseconds()), configDir, effectiveRoots)

		var reportErr error
		switch common.format {
		case "json":
			reportErr = report.PrintJSON(findings, summary)
		case "sarif":
			reportErr = report.PrintSARIF(findings, summary)
		default:
			if !common.quiet {
				fmt.Printf("Scanned %d files\n", len(result.Files))
			}
			reportErr = report.PrintText(findings, summary)
		}
		if reportErr != nil {
			fmt.Fprintf(os.Stderr, "failed to write report: %v\n", reportErr)
			return 2
		}

		if result.Diagnostics.ParseErrors > 0 && *parseErrorPolicy == "error" {
			fmt.Fprintf(os.Stderr, "parse/read errors encountered: %d file(s) skipped; failing due to --parse-error-policy=error\n", result.Diagnostics.FilesSkipped)
			return 2
		}

		blocking := false
		for _, f := range findings {
			if severityMeetsThreshold(strings.ToLower(f.Severity), *severityThreshold) {
				blocking = true
				break
			}
		}
		if blocking {
			return 1
		}
		return 0
	})
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "failed to set working directory: %v\n", runErr)
		return 2
	}
	return code
}

func buildSummary(findings []model.Finding, filesScanned, importsScanned int, diagnostics analysis.Diagnostics, suppressedFindings, durationMS int, configDir string, effectiveRoots []string) report.Summary {
	summary := report.Summary{
		FilesScanned:            filesScanned,
		ImportsScanned:          importsScanned,
		FindingsTotal:           len(findings),
		SuppressedFindings:      suppressedFindings,
		ParseErrors:             diagnostics.ParseErrors,
		FilesSkipped:            diagnostics.FilesSkipped,
		WorkspacePackageImports: diagnostics.WorkspacePackageImports,
		UnresolvedLocalImports:  diagnostics.UnresolvedLocalImports,
		IgnoredResolutionCases:  diagnostics.IgnoredResolutionCases,
		ConfigDir:               filepath.ToSlash(filepath.Clean(configDir)),
		EffectiveRoots:          append([]string{}, effectiveRoots...),
		DurationMS:              durationMS,
	}
	for _, f := range findings {
		switch f.Severity {
		case "error":
			summary.FindingsError++
		case "warning":
			summary.FindingsWarning++
		}
	}
	return summary
}

func resolveEffectiveRoots(configDir string, roots []string) []string {
	if len(roots) == 0 {
		return []string{filepath.ToSlash(filepath.Clean(configDir))}
	}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		if filepath.IsAbs(trimmed) {
			out = append(out, filepath.ToSlash(filepath.Clean(trimmed)))
			continue
		}
		out = append(out, filepath.ToSlash(filepath.Clean(filepath.Join(configDir, trimmed))))
	}
	sort.Strings(out)
	return out
}
