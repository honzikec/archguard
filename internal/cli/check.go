package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/parser"
	"github.com/honzikec/archguard/internal/policy"
	"github.com/honzikec/archguard/internal/report"
)

func runCheck(args []string) {
	cfg, err := config.Load("archguard.yaml")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	format := "text"
	debug := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--debug" {
			debug = true
		}
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

	if format == "text" || debug {
		fmt.Printf("Scanning %d files\n", len(files))
	}

	if debug {
		fmt.Printf("Detected imports:\n\n")
	}

	var allImports []model.ImportRef
	for _, file := range files {
		imports, err := parser.ParseFile(file)
		if err != nil {
			if format == "text" || debug {
				fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file, err)
			}
			continue
		}

		if debug {
			for _, imp := range imports {
				fmt.Printf("%s -> %s\n", imp.SourceFile, imp.RawImport)
			}
		}
		allImports = append(allImports, imports...)
	}

	findings := policy.Evaluate(cfg, allImports, files)
	if len(findings) > 0 {
		switch format {
		case "json":
			report.PrintJSON(findings)
		case "sarif":
			report.PrintSARIF(findings)
		case "text":
			fallthrough
		default:
			report.PrintText(findings)
		}
		os.Exit(1)
	}

	if !debug {
		if format == "text" {
			fmt.Println("No architectural violations found.")
		} else if format == "json" {
			fmt.Println("[]")
		} else if format == "sarif" {
			report.PrintSARIF(nil)
		}
	}
}
