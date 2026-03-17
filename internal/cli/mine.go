package cli

import (
	"fmt"
	"os"

	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/miner"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/parser"
	"github.com/honzikec/archguard/internal/pathutil"
)

func runMine(args []string) {
	files, err := fileset.Discover(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering files: %v\n", err)
		os.Exit(1)
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
	candidates := miner.Propose(g)
	miner.PrintCandidates(candidates)
}
