package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/honzikec/archguard/internal/catalog"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/parser"
	"github.com/honzikec/archguard/internal/pathutil"
)

func runMine(args []string) int {
	fs := flag.NewFlagSet("mine", flag.ContinueOnError)
	setFlagSetOutput(fs)
	common := bindCommonFlags(fs, commonFlags{configPath: "archguard.yaml", format: "text"})
	minSupport := fs.Int("min-support", 20, "Minimum files per scope to propose candidate")
	maxPrevalence := fs.Float64("max-prevalence", 0.02, "Maximum prevalence to consider as invariant")
	emitConfig := fs.Bool("emit-config", false, "Emit a starter config from mined candidates")
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

	cfg, err := loadConfigOptional(common.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	files, err := fileset.Discover(cfg.Project)
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
		parsed, err := parser.ParseFile(file)
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

	g := graph.Build(imports, files)
	candidates := miner.Propose(g, files, miner.Options{MinSupport: *minSupport, MaxPrevalence: *maxPrevalence})
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

	if *emitConfig {
		adopted := []config.Rule{}
		if *adoptCatalog {
			adopted = miner.AdoptCatalogMatches(catalogMatches, *adoptThreshold)
		}
		fmt.Print(miner.EmitStarterConfigWithCatalog(candidates, adopted))
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
		miner.PrintMineJSON(candidates, catalogMatches)
	default:
		miner.PrintMineText(candidates, catalogMatches, *catalogFormat)
	}
	return 0
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
