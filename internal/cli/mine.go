package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/parser"
	"github.com/honzikec/archguard/internal/pathutil"
)

func runMine(args []string) {
	format := "text"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--format" && i+1 < len(args) {
			format = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--format=") {
			format = strings.TrimPrefix(arg, "--format=")
		}
	}
	files, err := fileset.Discover(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering files: %v\n", err)
		os.Exit(1)
	}

	if format == "text" {
		fmt.Fprintf(os.Stderr, "Scanning %d files\n", len(files))
	}

	var allImports []model.ImportRef
	for _, file := range files {
		imports, err := parser.ParseFile(file)
		if err != nil {
			continue
		}
		
		// Resolve paths
		for i := range imports {
			if !imports[i].IsPackageImport {
				imports[i].ResolvedPath = pathutil.ResolveImport(imports[i].SourceFile, imports[i].RawImport)
			}
		}

		allImports = append(allImports, imports...)
	}

	g := graph.Build(allImports, files)

	// Gather candidates from all three sources
	candidates := miner.Propose(g)
	candidates = append(candidates, miner.ProposeCrossAppRules(g)...)
	candidates = append(candidates, miner.ProposeFileConventions(files)...)

	// Detect cycles (always shown regardless of format, as they are violations not candidates)
	cycles := miner.DetectCycles(g)

	switch format {
	case "yaml":
		if len(cycles) > 0 {
			fmt.Println("# WARNING: Circular dependencies detected:")
			for _, cycle := range cycles {
				fmt.Printf("# CYCLE: %s\n", joinCycle(cycle.Chain))
			}
			fmt.Println()
		}
		miner.PrintCandidatesYAML(candidates)
	default:
		if len(cycles) > 0 {
			fmt.Fprintf(os.Stderr, "\n⚠ Circular dependencies detected:\n")
			for _, cycle := range cycles {
				fmt.Fprintf(os.Stderr, "  CYCLE: %s\n", joinCycle(cycle.Chain))
			}
			fmt.Fprintln(os.Stderr)
		}
		miner.PrintCandidates(candidates)
	}
}

func joinCycle(chain []string) string {
	result := ""
	for i, s := range chain {
		if i > 0 {
			result += " → "
		}
		result += s
	}
	return result
}
