package cli

import (
	"fmt"
	"os"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/parser"
	"github.com/honzikec/archguard/internal/policy"
)

func runCheck(args []string) {
	cfg, err := config.Load("archguard.yaml")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	files, err := fileset.Discover(".")
	if err != nil {
		fmt.Printf("Error discovering files: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Scanning %d files\n", len(files))

	debug := false
	for _, arg := range args {
		if arg == "--debug" {
			debug = true
			break
		}
	}

	if debug {
		fmt.Println("Detected imports:\n")
	}

	var allImports []model.ImportRef
	for _, file := range files {
		imports, err := parser.ParseFile(file)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", file, err)
			continue
		}

		if debug {
			for _, imp := range imports {
				fmt.Printf("%s -> %s\n", imp.SourceFile, imp.RawImport)
			}
		}
		allImports = append(allImports, imports...)
	}

	findings := policy.Evaluate(cfg, allImports)
	if len(findings) > 0 {
		for _, f := range findings {
			fmt.Printf("%s\n\n%s\nimports\n%s\n\n", f.Message, f.FilePath, f.RawImport)
		}
		os.Exit(1)
	}

	if !debug {
		fmt.Println("No architectural violations found.")
	}
}
