package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/honzikec/archguard/internal/config"
)

func runExplain(args []string) {
	ruleID := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--rule" && i+1 < len(args) {
			ruleID = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--rule=") {
			ruleID = strings.TrimPrefix(args[i], "--rule=")
		}
	}

	if ruleID == "" {
		fmt.Println("Usage: archguard explain --rule <rule-id>")
		os.Exit(1)
	}

	cfg, err := config.Load("archguard.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	for _, rule := range cfg.Rules {
		if rule.ID == ruleID {
			fmt.Printf("Rule: %s\n", rule.ID)
			fmt.Printf("Kind: %s\n", rule.Kind)
			fmt.Printf("Severity: %s\n", rule.Severity)
			fmt.Printf("Rationale: %s\n", rule.Rationale)
			fmt.Printf("Conditions:\n")
			if len(rule.Conditions.FromPaths) > 0 {
				fmt.Printf("  from_paths: %v\n", rule.Conditions.FromPaths)
			}
			if len(rule.Conditions.ForbiddenPaths) > 0 {
				fmt.Printf("  forbidden_paths: %v\n", rule.Conditions.ForbiddenPaths)
			}
			if len(rule.Conditions.ForbiddenPackages) > 0 {
				fmt.Printf("  forbidden_packages: %v\n", rule.Conditions.ForbiddenPackages)
			}
			if len(rule.Conditions.PathPatterns) > 0 {
				fmt.Printf("  path_patterns: %v\n", rule.Conditions.PathPatterns)
			}
			if rule.Conditions.FilenameRegex != "" {
				fmt.Printf("  filename_regex: %s\n", rule.Conditions.FilenameRegex)
			}
			return
		}
	}

	fmt.Printf("Rule %s not found\n", ruleID)
	os.Exit(1)
}
