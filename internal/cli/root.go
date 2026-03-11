package cli

import (
	"fmt"
	"os"
)

func Execute() {
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printHelp()
		return
	}

	command := os.Args[1]
	switch command {
	case "check":
		runCheck(os.Args[2:])
	case "mine":
		runMine(os.Args[2:])
	case "explain":
		runExplain(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("ArchGuard — Architectural Sentinel\n")
	fmt.Println("Commands:")
	fmt.Println("  check")
	fmt.Println("  mine")
	fmt.Println("  explain")
}
