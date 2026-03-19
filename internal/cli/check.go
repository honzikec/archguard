package cli

import (
	"flag"
	"fmt"
	"os"
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

	started := time.Now()
	cfg, err := config.Load(common.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	languageResolution := language.Resolve(cfg.Project.Language, cfg.Project.Roots)
	if languageResolution.Adapter == nil {
		fmt.Fprintf(os.Stderr, "failed to resolve language adapter\n")
		return 2
	}
	if common.debug {
		fmt.Fprintf(os.Stderr, "language adapter: %s (%s)\n", languageResolution.Selected, languageResolution.Reason)
	}

	files, err := fileset.DiscoverWithAdapter(cfg.Project, languageResolution.Adapter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to discover files: %v\n", err)
		return 2
	}
	if *changedOnly {
		files, err = filterChangedFiles(files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 2
		}
	}

	resolver, err := pathutil.NewResolver(".", cfg.Project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize resolver: %v\n", err)
		return 2
	}

	allImports := make([]model.ImportRef, 0)
	for _, file := range files {
		imports, err := languageResolution.Adapter.ParseFile(file)
		if err != nil {
			if common.debug {
				fmt.Fprintf(os.Stderr, "parse error %s: %v\n", file, err)
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

	summary := buildSummary(findings, len(files), len(allImports), int(time.Since(started).Milliseconds()))

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
}

func buildSummary(findings []model.Finding, filesScanned, importsScanned, durationMS int) report.Summary {
	summary := report.Summary{
		FilesScanned:   filesScanned,
		ImportsScanned: importsScanned,
		FindingsTotal:  len(findings),
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
