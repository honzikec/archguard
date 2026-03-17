package cli

import (
	"fmt"
	"os"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/fileset"
	"github.com/honzikec/archguard/internal/parser"
)

func runCheck(args []string) {
	_, err := config.Load("archguard.yaml")
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
	}

	if !debug {
		fmt.Println("check command not yet implemented")
	}
}
