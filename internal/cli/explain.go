package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/honzikec/archguard/internal/config"
)

func runExplain(args []string) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	setFlagSetOutput(fs)
	common := bindCommonFlags(fs, commonFlags{configPath: "archguard.yaml", format: "text"})
	ruleID := fs.String("rule", "", "Rule ID to explain")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *ruleID == "" {
		fmt.Fprintln(os.Stderr, "--rule is required")
		return 2
	}
	if common.format != "text" && common.format != "json" {
		fmt.Fprintf(os.Stderr, "unsupported format: %s\n", common.format)
		return 2
	}

	cfg, err := config.Load(common.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	for _, rule := range cfg.Rules {
		if rule.ID != *ruleID {
			continue
		}
		if common.format == "json" {
			data, _ := json.MarshalIndent(rule, "", "  ")
			fmt.Println(string(data))
			return 0
		}
		fmt.Printf("Rule: %s\n", rule.ID)
		fmt.Printf("Kind: %s\n", rule.Kind)
		fmt.Printf("Severity: %s\n", rule.Severity)
		fmt.Printf("Scope: %v\n", rule.Scope)
		if len(rule.Target) > 0 {
			fmt.Printf("Target: %v\n", rule.Target)
		}
		if len(rule.Except) > 0 {
			fmt.Printf("Except: %v\n", rule.Except)
		}
		if rule.Message != "" {
			fmt.Printf("Message: %s\n", rule.Message)
		}
		return 0
	}

	fmt.Fprintf(os.Stderr, "rule not found: %s\n", *ruleID)
	return 2
}
