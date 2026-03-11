package cli

import (
	"fmt"
	"os"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/fileset"
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

	fmt.Println("check command not yet implemented")
}
