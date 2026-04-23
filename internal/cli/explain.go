package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/honzikec/archguard/internal/analysis"
	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/language"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/policy"
)

func runExplain(args []string) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	setFlagSetOutput(fs)
	common := bindCommonFlags(fs, commonFlags{configPath: "archguard.yaml", format: "text"})
	ruleID := fs.String("rule", "", "Rule ID to explain")
	fingerprint := fs.String("finding", "", "Finding fingerprint to explain from the current repository state")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if common.format != "text" && common.format != "json" {
		fmt.Fprintf(os.Stderr, "unsupported format: %s\n", common.format)
		return 2
	}

	ruleIDValue := strings.TrimSpace(*ruleID)
	fingerprintValue := strings.TrimSpace(*fingerprint)
	if (ruleIDValue == "" && fingerprintValue == "") || (ruleIDValue != "" && fingerprintValue != "") {
		fmt.Fprintln(os.Stderr, "exactly one of --rule or --finding is required")
		return 2
	}

	configPath, configDir, err := resolveConfigPath(common.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve config path: %v\n", err)
		return 2
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	if ruleIDValue != "" {
		return printRuleExplanation(cfg, ruleIDValue, common.format)
	}

	code, runErr := withWorkingDir(configDir, func() int {
		languageResolution := language.Resolve(cfg.Project.Language, cfg.Project.Roots)
		if languageResolution.Adapter == nil {
			fmt.Fprintln(os.Stderr, "failed to resolve language adapter")
			return 2
		}
		result, err := analysis.Run(cfg.Project, languageResolution.Adapter, analysis.Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 2
		}
		findings, err := policy.Evaluate(cfg, result.Imports, result.Files, result.Graph)
		if err != nil {
			fmt.Fprintf(os.Stderr, "policy evaluation failed: %v\n", err)
			return 2
		}
		return printFindingExplanation(cfg, findings, fingerprintValue, common.format)
	})
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "failed to set working directory: %v\n", runErr)
		return 2
	}
	return code
}

func printRuleExplanation(cfg *config.Config, ruleID string, format string) int {
	rule, ok := findRuleByID(cfg, ruleID)
	if !ok {
		fmt.Fprintf(os.Stderr, "rule not found: %s\n", ruleID)
		return 2
	}

	payload := struct {
		Mode        string      `json:"mode"`
		Rule        config.Rule `json:"rule"`
		Why         string      `json:"why"`
		Remediation string      `json:"remediation"`
		Options     string      `json:"options"`
	}{
		Mode:        "rule",
		Rule:        rule,
		Why:         policy.RuleWhy(rule),
		Remediation: policy.RuleRemediation(rule),
		Options:     policy.FindingOptions(rule),
	}

	if format == "json" {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return 0
	}

	fmt.Printf("Rule: %s\n", rule.ID)
	fmt.Printf("Kind: %s\n", rule.Kind)
	fmt.Printf("Severity: %s\n", rule.Severity)
	if rule.Template != "" {
		fmt.Printf("Template: %s\n", rule.Template)
	}
	fmt.Printf("Scope: %v\n", rule.Scope)
	if len(rule.Target) > 0 {
		fmt.Printf("Target: %v\n", rule.Target)
	}
	if len(rule.Except) > 0 {
		fmt.Printf("Except: %v\n", rule.Except)
	}
	if len(rule.Params) > 0 {
		fmt.Printf("Params: %v\n", rule.Params)
	}
	if rule.Message != "" {
		fmt.Printf("Message: %s\n", rule.Message)
	}
	fmt.Printf("Why: %s\n", payload.Why)
	fmt.Printf("How to fix: %s\n", payload.Remediation)
	fmt.Printf("Options: %s\n", payload.Options)
	fmt.Println("Tip: use `archguard explain --finding <fingerprint>` to inspect a specific current violation.")
	return 0
}

func printFindingExplanation(cfg *config.Config, findings []model.Finding, fingerprint string, format string) int {
	for _, finding := range findings {
		if finding.Fingerprint != fingerprint {
			continue
		}
		rule, _ := findRuleByID(cfg, finding.RuleID)
		payload := struct {
			Mode    string        `json:"mode"`
			Finding model.Finding `json:"finding"`
			Rule    *config.Rule  `json:"rule,omitempty"`
			Why     string        `json:"why,omitempty"`
			Options string        `json:"options,omitempty"`
		}{
			Mode:    "finding",
			Finding: finding,
		}
		if rule.ID != "" {
			ruleCopy := rule
			payload.Rule = &ruleCopy
			payload.Why = policy.RuleWhy(rule)
			payload.Options = policy.FindingOptions(rule)
		}

		if format == "json" {
			data, _ := json.MarshalIndent(payload, "", "  ")
			fmt.Println(string(data))
			return 0
		}

		fmt.Printf("Finding: %s\n", finding.Fingerprint)
		fmt.Printf("Rule: %s (%s, %s)\n", finding.RuleID, finding.RuleKind, finding.Severity)
		fmt.Printf("Location: %s:%d:%d\n", finding.FilePath, finding.Line, finding.Column)
		fmt.Printf("Message: %s\n", finding.Message)
		if finding.RawImport != "" {
			fmt.Printf("Trigger: %s\n", finding.RawImport)
		}
		if finding.MatchedScope != "" {
			fmt.Printf("Matched scope: %s\n", finding.MatchedScope)
		}
		if finding.MatchedTarget != "" {
			fmt.Printf("Matched target: %s\n", finding.MatchedTarget)
		}
		if finding.Evidence != "" {
			fmt.Printf("Evidence: %s\n", finding.Evidence)
		}
		if payload.Why != "" {
			fmt.Printf("Why: %s\n", payload.Why)
		}
		if finding.Remediation != "" {
			fmt.Printf("How to fix: %s\n", finding.Remediation)
		}
		if payload.Options != "" {
			fmt.Printf("Options: %s\n", payload.Options)
		}
		return 0
	}

	fmt.Fprintf(os.Stderr, "finding not found: %s\n", fingerprint)
	return 2
}

func findRuleByID(cfg *config.Config, ruleID string) (config.Rule, bool) {
	for _, rule := range cfg.Rules {
		if rule.ID == ruleID {
			return rule, true
		}
	}
	return config.Rule{}, false
}
