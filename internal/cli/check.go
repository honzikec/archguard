package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/language"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pathutil"
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

		files, err := fileset.DiscoverWithAdapter(cfg.Project, languageResolution.Adapter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to discover files: %v\n", err)
			return 2
		}
		files, err = filterChangedFiles(files, *changedOnly, *changedAgainst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 2
		}

		resolver, err := pathutil.NewResolver(".", cfg.Project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize resolver: %v\n", err)
			return 2
		}

		allImports := make([]model.ImportRef, 0)
		parseErrors := 0
		filesSkipped := 0
		for _, file := range files {
			imports, err := languageResolution.Adapter.ParseFile(file)
			if err != nil {
				parseErrors++
				filesSkipped++
				if common.debug {
					fmt.Fprintf(os.Stderr, "parse/read error %s: %v\n", file, err)
				}
				continue
			}
			for i := range imports {
				resolved, isPackage := resolver.Resolve(imports[i].SourceFile, imports[i].RawImport)
				imports[i].ResolvedPath = resolved
				imports[i].IsPackageImport = isPackage
				if common.debug {
					fmt.Fprintf(os.Stderr, "%s -> %s (resolved=%s package=%t)\n", imports[i].SourceFile, imports[i].RawImport, imports[i].ResolvedPath, imports[i].IsPackageImport)
				}
			}
			allImports = append(allImports, imports...)
		}

		g := graph.Build(allImports, files)
		findings, err := policy.Evaluate(cfg, allImports, files, g)
		if err != nil {
			fmt.Fprintf(os.Stderr, "policy evaluation failed: %v\n", err)
			return 2
		}

		if *maxFindings > 0 && len(findings) > *maxFindings {
			findings = findings[:*maxFindings]
		}

		summary := buildSummary(findings, len(files), len(allImports), parseErrors, filesSkipped, int(time.Since(started).Milliseconds()), configDir, effectiveRoots)

		switch common.format {
		case "json":
			report.PrintJSON(findings, summary)
		case "sarif":
			report.PrintSARIF(findings, summary)
		default:
			if !common.quiet {
				fmt.Printf("Scanned %d files\n", len(files))
			}
			report.PrintText(findings, summary)
		}

		if parseErrors > 0 && *parseErrorPolicy == "error" {
			fmt.Fprintf(os.Stderr, "parse/read errors encountered: %d file(s) skipped; failing due to --parse-error-policy=error\n", filesSkipped)
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

func buildSummary(findings []model.Finding, filesScanned, importsScanned, parseErrors, filesSkipped, durationMS int, configDir string, effectiveRoots []string) report.Summary {
	summary := report.Summary{
		FilesScanned:   filesScanned,
		ImportsScanned: importsScanned,
		FindingsTotal:  len(findings),
		ParseErrors:    parseErrors,
		FilesSkipped:   filesSkipped,
		ConfigDir:      filepath.ToSlash(filepath.Clean(configDir)),
		EffectiveRoots: append([]string{}, effectiveRoots...),
		DurationMS:     durationMS,
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
