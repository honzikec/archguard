package cli

import (
	"fmt"
	"os"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func Execute() {
	code := execute(os.Args[1:])
	os.Exit(code)
}

func execute(args []string) int {
	if len(args) == 0 {
		printHelp()
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp()
		return 0
	case "check":
		return runCheck(args[1:])
	case "mine":
		return runMine(args[1:])
	case "explain":
		return runExplain(args[1:])
	case "init":
		return runInit(args[1:])
	case "version", "--version", "-v":
		return runVersion(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printHelp()
		return 2
	}
}

func printHelp() {
	fmt.Println("ArchGuard v0.2 - Architectural policy checks for TS/JS repositories")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  archguard <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  check    Evaluate rules against repository files")
	fmt.Println("  mine     Discover candidate rules from existing structure")
	fmt.Println("  explain  Print details for a configured rule")
	fmt.Println("  init     Write a starter archguard.yaml or generate profile/adapter scaffolds")
	fmt.Println("  version  Print version/build information")
	fmt.Println()
	fmt.Println("Run `archguard <command> --help` for command-specific flags.")
}
