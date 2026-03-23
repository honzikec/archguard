package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/miner"
	"gopkg.in/yaml.v3"
)

type interactiveProposal struct {
	Source     string
	Rule       config.Rule
	Confidence string
	Evidence   string
	Support    int
	Violations int
	Prevalence float64
}

func runMineInteractive(configPath string, base *config.Config, candidates []miner.Candidate, adopted []config.Rule, noCycleSeverity string) error {
	proposals := buildInteractiveProposals(candidates, adopted, noCycleSeverity)
	if len(proposals) == 0 {
		fmt.Println("No mined rules available for interactive selection.")
		return nil
	}

	fmt.Printf("Interactive mine: %d rule proposals\n", len(proposals))
	printInteractiveProposals(proposals)

	reader := bufio.NewReader(os.Stdin)
	selected, err := promptProposalSelection(reader, len(proposals))
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		fmt.Println("No rules selected. No changes written.")
		return nil
	}

	severityMode, err := promptSeverityOverride(reader)
	if err != nil {
		return err
	}

	selectedRules := make([]config.Rule, 0, len(selected))
	for _, idx := range selected {
		rule := deepCopyRule(proposals[idx].Rule)
		if severityMode != "" {
			rule.Severity = severityMode
		}
		selectedRules = append(selectedRules, rule)
	}

	merged, added := mergeSelectedRules(base, selectedRules)
	if len(added) == 0 {
		fmt.Println("All selected rules already exist in config. No changes written.")
		return nil
	}
	if err := config.Validate(merged); err != nil {
		return fmt.Errorf("merged config validation failed: %w", err)
	}
	output, err := yaml.Marshal(merged)
	if err != nil {
		return fmt.Errorf("failed to render merged config: %w", err)
	}

	fmt.Printf("Planned changes: add %d rule(s) to %s\n", len(added), configPath)
	for _, rule := range added {
		target := "-"
		if len(rule.Target) > 0 {
			target = previewList(rule.Target, 2)
		}
		fmt.Printf("  + %s kind=%s severity=%s scope=%s target=%s\n",
			rule.ID, rule.Kind, rule.Severity, previewList(rule.Scope, 2), target)
	}

	ok, err := promptConfirmWrite(reader)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("Canceled. No changes written.")
		return nil
	}

	if dir := filepath.Dir(configPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}
	if err := os.WriteFile(configPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Printf("Updated %s with %d new rule(s).\n", configPath, len(added))
	return nil
}

func buildInteractiveProposals(candidates []miner.Candidate, adopted []config.Rule, noCycleSeverity string) []interactiveProposal {
	out := make([]interactiveProposal, 0, len(candidates)+len(adopted))
	for i, c := range candidates {
		rule := minedCandidateToRule(c, i, noCycleSeverity)
		out = append(out, interactiveProposal{
			Source:     "mined",
			Rule:       rule,
			Confidence: string(c.Confidence),
			Evidence:   c.Evidence,
			Support:    c.Support,
			Violations: c.Violations,
			Prevalence: c.Prevalence,
		})
	}
	for _, rule := range adopted {
		out = append(out, interactiveProposal{
			Source:     "catalog",
			Rule:       deepCopyRule(rule),
			Confidence: "CATALOG",
			Evidence:   rule.Message,
			Support:    0,
			Violations: 0,
			Prevalence: 0,
		})
	}
	return out
}

func minedCandidateToRule(c miner.Candidate, index int, noCycleSeverity string) config.Rule {
	severity := c.Severity
	if c.Kind == config.KindNoCycle {
		override := strings.ToLower(strings.TrimSpace(noCycleSeverity))
		if override == config.SeverityWarning || override == config.SeverityError {
			severity = override
		}
	}
	rule := config.Rule{
		ID:       fmt.Sprintf("MINED-%03d", index+1),
		Kind:     c.Kind,
		Severity: severity,
		Scope:    append([]string{}, c.Scope...),
		Target:   append([]string{}, c.Target...),
		Message:  c.Evidence,
	}
	return rule
}

func printInteractiveProposals(proposals []interactiveProposal) {
	for i, p := range proposals {
		target := "-"
		if len(p.Rule.Target) > 0 {
			target = previewList(p.Rule.Target, 2)
		}
		fmt.Printf("[%d] %s (%s) source=%s severity=%s support=%d violations=%d prevalence=%.4f\n",
			i+1, p.Rule.ID, p.Rule.Kind, p.Source, p.Rule.Severity, p.Support, p.Violations, p.Prevalence)
		fmt.Printf("    scope=%s target=%s\n", previewList(p.Rule.Scope, 2), target)
		if strings.TrimSpace(p.Evidence) != "" {
			fmt.Printf("    evidence=%s\n", p.Evidence)
		}
	}
}

func promptProposalSelection(reader *bufio.Reader, max int) ([]int, error) {
	for {
		line, err := readPrompt(reader, "Select rules [a=all, n=none, list e.g. 1,3-5]: ")
		if err != nil {
			return nil, err
		}
		line = strings.ToLower(strings.TrimSpace(line))
		switch line {
		case "", "n", "none":
			return []int{}, nil
		case "a", "all":
			ids := make([]int, max)
			for i := 0; i < max; i++ {
				ids[i] = i
			}
			return ids, nil
		default:
			parsed, err := parseSelection(line, max)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid selection: %v\n", err)
				continue
			}
			return parsed, nil
		}
	}
}

func parseSelection(input string, max int) ([]int, error) {
	seen := map[int]struct{}{}
	for _, token := range strings.Split(input, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid range %q", token)
			}
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q", token)
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q", token)
			}
			if start < 1 || end < 1 || start > end || end > max {
				return nil, fmt.Errorf("range %q out of bounds 1-%d", token, max)
			}
			for i := start; i <= end; i++ {
				seen[i-1] = struct{}{}
			}
			continue
		}
		n, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("invalid index %q", token)
		}
		if n < 1 || n > max {
			return nil, fmt.Errorf("index %d out of bounds 1-%d", n, max)
		}
		seen[n-1] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for idx := range seen {
		out = append(out, idx)
	}
	sort.Ints(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("selection is empty")
	}
	return out, nil
}

func promptSeverityOverride(reader *bufio.Reader) (string, error) {
	for {
		line, err := readPrompt(reader, "Severity override [k=keep, w=warning, e=error] (default k): ")
		if err != nil {
			return "", err
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "", "k", "keep":
			return "", nil
		case "w", "warning":
			return config.SeverityWarning, nil
		case "e", "error":
			return config.SeverityError, nil
		default:
			fmt.Fprintln(os.Stderr, "invalid severity option")
		}
	}
}

func promptConfirmWrite(reader *bufio.Reader) (bool, error) {
	line, err := readPrompt(reader, "Write changes to config? [y/N]: ")
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func readPrompt(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			return strings.TrimSpace(line), nil
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func mergeSelectedRules(base *config.Config, selected []config.Rule) (*config.Config, []config.Rule) {
	merged := cloneConfig(base)
	if merged.Version == 0 {
		merged.Version = 1
	}

	usedIDs := map[string]struct{}{}
	signatures := map[string]struct{}{}
	for _, rule := range merged.Rules {
		usedIDs[rule.ID] = struct{}{}
		signatures[ruleSignature(rule)] = struct{}{}
	}

	added := make([]config.Rule, 0, len(selected))
	for _, rule := range selected {
		sig := ruleSignature(rule)
		if _, exists := signatures[sig]; exists {
			continue
		}
		rule.ID = uniqueRuleID(rule.ID, usedIDs)
		usedIDs[rule.ID] = struct{}{}
		signatures[ruleSignature(rule)] = struct{}{}
		merged.Rules = append(merged.Rules, rule)
		added = append(added, rule)
	}
	return merged, added
}

func uniqueRuleID(base string, used map[string]struct{}) string {
	id := strings.TrimSpace(base)
	if id == "" {
		id = "MINED"
	}
	if _, exists := used[id]; !exists {
		return id
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

func ruleSignature(rule config.Rule) string {
	type sigRule struct {
		Kind     string            `json:"kind"`
		Severity string            `json:"severity"`
		Scope    []string          `json:"scope,omitempty"`
		Target   []string          `json:"target,omitempty"`
		Except   []string          `json:"except,omitempty"`
		Template string            `json:"template,omitempty"`
		Params   map[string]string `json:"params,omitempty"`
		Message  string            `json:"message,omitempty"`
	}
	payload := sigRule{
		Kind:     rule.Kind,
		Severity: rule.Severity,
		Scope:    append([]string{}, rule.Scope...),
		Target:   append([]string{}, rule.Target...),
		Except:   append([]string{}, rule.Except...),
		Template: rule.Template,
		Params:   cloneStringMap(rule.Params),
		Message:  rule.Message,
	}
	sort.Strings(payload.Scope)
	sort.Strings(payload.Target)
	sort.Strings(payload.Except)
	data, _ := json.Marshal(payload)
	return string(data)
}

func cloneConfig(base *config.Config) *config.Config {
	if base == nil {
		return &config.Config{
			Version: 1,
			Project: config.DefaultProjectSettings(),
			Rules:   []config.Rule{},
		}
	}
	out := &config.Config{
		Version: base.Version,
		Project: config.ProjectSettings{
			Roots:     append([]string{}, base.Project.Roots...),
			Include:   append([]string{}, base.Project.Include...),
			Exclude:   append([]string{}, base.Project.Exclude...),
			Framework: base.Project.Framework,
			Language:  base.Project.Language,
			Tsconfig:  base.Project.Tsconfig,
			Aliases:   map[string][]string{},
		},
		Rules: make([]config.Rule, 0, len(base.Rules)),
	}
	for k, vals := range base.Project.Aliases {
		out.Project.Aliases[k] = append([]string{}, vals...)
	}
	for _, rule := range base.Rules {
		out.Rules = append(out.Rules, deepCopyRule(rule))
	}
	return out
}

func deepCopyRule(rule config.Rule) config.Rule {
	return config.Rule{
		ID:       rule.ID,
		Kind:     rule.Kind,
		Severity: rule.Severity,
		Scope:    append([]string{}, rule.Scope...),
		Target:   append([]string{}, rule.Target...),
		Except:   append([]string{}, rule.Except...),
		Template: rule.Template,
		Params:   cloneStringMap(rule.Params),
		Message:  rule.Message,
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func previewList(values []string, max int) string {
	if len(values) == 0 {
		return "[]"
	}
	if max <= 0 || len(values) <= max {
		return "[" + strings.Join(values, ", ") + "]"
	}
	return "[" + strings.Join(values[:max], ", ") + fmt.Sprintf(", ... +%d]", len(values)-max)
}
