package policy

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/graph"
	"github.com/honzikec/archguard/internal/model"
	"github.com/honzikec/archguard/internal/pathutil"
)

func Evaluate(cfg *config.Config, imports []model.ImportRef, files []string, g *graph.Graph) ([]model.Finding, error) {
	compiled, err := compileRules(cfg.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to compile rules: %w", err)
	}

	findings := make([]model.Finding, 0)
	seen := map[string]struct{}{}

	for _, cr := range compiled {
		rule := cr.Rule
		switch rule.Kind {
		case config.KindNoImport:
			for _, imp := range imports {
				if imp.IsPackageImport || imp.ResolvedPath == "" {
					continue
				}
				if !matchesScope(rule.Scope, imp.SourceFile) {
					continue
				}
				if isExcepted(rule.Except, imp.SourceFile, imp.ResolvedPath) {
					continue
				}
				if !pathutil.MatchAny(rule.Target, imp.ResolvedPath) {
					continue
				}
				f := baseFinding(rule, imp.SourceFile, imp.Line, imp.Column, imp.RawImport)
				f.Message = defaultMessage(rule, fmt.Sprintf("%s imports %s", imp.SourceFile, imp.ResolvedPath))
				appendFinding(&findings, seen, f)
			}
		case config.KindNoPackage:
			for _, imp := range imports {
				if !imp.IsPackageImport {
					continue
				}
				if !matchesScope(rule.Scope, imp.SourceFile) {
					continue
				}
				if isExcepted(rule.Except, imp.SourceFile, imp.RawImport) {
					continue
				}
				if !packageMatches(rule.Target, imp.RawImport) {
					continue
				}
				f := baseFinding(rule, imp.SourceFile, imp.Line, imp.Column, imp.RawImport)
				f.Message = defaultMessage(rule, fmt.Sprintf("%s imports package %s", imp.SourceFile, imp.RawImport))
				appendFinding(&findings, seen, f)
			}
		case config.KindFilePattern:
			for _, file := range files {
				if !matchesScope(rule.Scope, file) {
					continue
				}
				if isExcepted(rule.Except, file, "") {
					continue
				}
				base := path.Base(file)
				matched := false
				for _, tr := range cr.TargetRegexes {
					if tr.Regexp.MatchString(base) {
						matched = true
						break
					}
				}
				if matched {
					continue
				}
				f := baseFinding(rule, file, 1, 1, "")
				f.Message = defaultMessage(rule, fmt.Sprintf("%s does not match required file pattern", file))
				appendFinding(&findings, seen, f)
			}
		case config.KindNoCycle:
			cycles := detectScopedCycles(g, rule.Scope, rule.Except)
			for _, cycle := range cycles {
				if len(cycle) == 0 {
					continue
				}
				chain := strings.Join(cycle, " -> ")
				f := baseFinding(rule, cycle[0], 1, 1, "")
				f.Details = chain
				f.Message = defaultMessage(rule, "dependency cycle detected: "+chain)
				appendFinding(&findings, seen, f)
			}
		case config.KindPattern:
			patternFindings, err := evaluatePatternRule(rule, imports, files)
			if err != nil {
				return nil, err
			}
			for _, finding := range patternFindings {
				appendFinding(&findings, seen, finding)
			}
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity < findings[j].Severity
		}
		if findings[i].RuleID != findings[j].RuleID {
			return findings[i].RuleID < findings[j].RuleID
		}
		if findings[i].FilePath != findings[j].FilePath {
			return findings[i].FilePath < findings[j].FilePath
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Column < findings[j].Column
	})

	return findings, nil
}

func appendFinding(findings *[]model.Finding, seen map[string]struct{}, finding model.Finding) {
	fingerprint := computeFingerprint(finding)
	finding.Fingerprint = fingerprint
	if _, ok := seen[fingerprint]; ok {
		return
	}
	seen[fingerprint] = struct{}{}
	*findings = append(*findings, finding)
}

func computeFingerprint(f model.Finding) string {
	h := sha1.New()
	h.Write([]byte(f.RuleID))
	h.Write([]byte("|"))
	h.Write([]byte(f.FilePath))
	h.Write([]byte("|"))
	h.Write([]byte(fmt.Sprintf("%d:%d", f.Line, f.Column)))
	h.Write([]byte("|"))
	h.Write([]byte(f.RawImport))
	h.Write([]byte("|"))
	h.Write([]byte(f.Details))
	return hex.EncodeToString(h.Sum(nil))
}

func baseFinding(rule config.Rule, file string, line, col int, rawImport string) model.Finding {
	return model.Finding{
		RuleID:    rule.ID,
		RuleKind:  rule.Kind,
		Severity:  rule.Severity,
		FilePath:  file,
		Line:      line,
		Column:    col,
		RawImport: rawImport,
	}
}

func defaultMessage(rule config.Rule, fallback string) string {
	if rule.Message != "" {
		return rule.Message
	}
	return fallback
}

func detectScopedCycles(g *graph.Graph, scope []string, except []string) [][]string {
	if g == nil {
		return nil
	}
	allowed := map[string]struct{}{}
	for node := range g.Nodes {
		if len(scope) > 0 && !pathutil.MatchAny(scope, node) {
			continue
		}
		if isExcepted(except, node, "") {
			continue
		}
		allowed[node] = struct{}{}
	}

	visited := map[string]bool{}
	inStack := map[string]int{}
	stack := []string{}
	canonicalSeen := map[string]struct{}{}
	cycles := [][]string{}

	var dfs func(string)
	dfs = func(node string) {
		visited[node] = true
		inStack[node] = len(stack)
		stack = append(stack, node)

		for neighbor := range g.Edges[node] {
			if _, ok := allowed[neighbor]; !ok {
				continue
			}
			if !visited[neighbor] {
				dfs(neighbor)
				continue
			}
			idx, ok := inStack[neighbor]
			if !ok {
				continue
			}
			cycle := append([]string{}, stack[idx:]...)
			cycle = append(cycle, neighbor)
			canon := canonicalCycle(cycle)
			if _, exists := canonicalSeen[canon]; exists {
				continue
			}
			canonicalSeen[canon] = struct{}{}
			cycles = append(cycles, cycle)
		}

		delete(inStack, node)
		stack = stack[:len(stack)-1]
	}

	nodes := make([]string, 0, len(allowed))
	for n := range allowed {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if !visited[node] {
			dfs(node)
		}
	}
	return cycles
}

func canonicalCycle(cycle []string) string {
	if len(cycle) < 2 {
		return strings.Join(cycle, "->")
	}
	core := cycle[:len(cycle)-1]
	best := ""
	for i := range core {
		rot := append([]string{}, core[i:]...)
		rot = append(rot, core[:i]...)
		key := strings.Join(rot, "->")
		if best == "" || key < best {
			best = key
		}
	}
	return best
}

func evaluatePatternRule(rule config.Rule, imports []model.ImportRef, files []string) ([]model.Finding, error) {
	switch rule.Template {
	case "dependency_constraint":
		return evaluatePatternDependencyConstraint(rule, imports), nil
	case "construction_policy":
		return evaluatePatternConstructionPolicy(rule, files)
	default:
		return nil, fmt.Errorf("unsupported pattern template %q", rule.Template)
	}
}

func evaluatePatternDependencyConstraint(rule config.Rule, imports []model.ImportRef) []model.Finding {
	findings := make([]model.Finding, 0)
	relation := "imports"
	if rule.Params != nil && strings.TrimSpace(rule.Params["relation"]) != "" {
		relation = rule.Params["relation"]
	}

	for _, imp := range imports {
		if !matchesScope(rule.Scope, imp.SourceFile) {
			continue
		}
		if relation == "packages" {
			if !imp.IsPackageImport {
				continue
			}
			if isExcepted(rule.Except, imp.SourceFile, imp.RawImport) {
				continue
			}
			if !packageMatches(rule.Target, imp.RawImport) {
				continue
			}
			f := baseFinding(rule, imp.SourceFile, imp.Line, imp.Column, imp.RawImport)
			f.Message = defaultMessage(rule, fmt.Sprintf("%s imports package %s", imp.SourceFile, imp.RawImport))
			findings = append(findings, f)
			continue
		}

		// Default to import-path relation
		if imp.IsPackageImport || imp.ResolvedPath == "" {
			continue
		}
		if isExcepted(rule.Except, imp.SourceFile, imp.ResolvedPath) {
			continue
		}
		if !pathutil.MatchAny(rule.Target, imp.ResolvedPath) {
			continue
		}
		f := baseFinding(rule, imp.SourceFile, imp.Line, imp.Column, imp.RawImport)
		f.Message = defaultMessage(rule, fmt.Sprintf("%s imports %s", imp.SourceFile, imp.ResolvedPath))
		findings = append(findings, f)
	}
	return findings
}

func evaluatePatternConstructionPolicy(rule config.Rule, files []string) ([]model.Finding, error) {
	serviceNames := collectServiceClassNames(files, rule.Target)
	newExpr := regexp.MustCompile(`\bnew\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)
	serviceNameRegex := regexp.MustCompile(`.*Service$`)
	if rule.Params != nil && strings.TrimSpace(rule.Params["service_name_regex"]) != "" {
		compiled, err := regexp.Compile(rule.Params["service_name_regex"])
		if err != nil {
			return nil, fmt.Errorf("invalid service_name_regex: %w", err)
		}
		serviceNameRegex = compiled
	}

	findings := make([]model.Finding, 0)
	for _, file := range files {
		if !matchesScope(rule.Scope, file) {
			continue
		}
		if isExcepted(rule.Except, file, "") {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		text := string(content)
		idxs := newExpr.FindAllStringSubmatchIndex(text, -1)
		for _, idx := range idxs {
			if len(idx) < 4 {
				continue
			}
			nameStart, nameEnd := idx[2], idx[3]
			if nameStart < 0 || nameEnd <= nameStart {
				continue
			}
			className := text[nameStart:nameEnd]
			if len(serviceNames) > 0 {
				if _, ok := serviceNames[className]; !ok {
					continue
				}
			} else if !serviceNameRegex.MatchString(className) {
				continue
			}
			line, col := lineCol(content, idx[0])
			f := baseFinding(rule, file, line, col, className)
			f.Message = defaultMessage(rule, fmt.Sprintf("direct construction of service %s outside composition root", className))
			findings = append(findings, f)
		}
	}

	return findings, nil
}

func collectServiceClassNames(files []string, serviceGlobs []string) map[string]struct{} {
	serviceNames := map[string]struct{}{}
	classDecl := regexp.MustCompile(`\bclass\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`)
	for _, file := range files {
		if !pathutil.MatchAny(serviceGlobs, file) {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		text := string(content)
		idxs := classDecl.FindAllStringSubmatchIndex(text, -1)
		for _, idx := range idxs {
			if len(idx) < 4 {
				continue
			}
			nameStart, nameEnd := idx[2], idx[3]
			if nameStart < 0 || nameEnd <= nameStart {
				continue
			}
			serviceNames[text[nameStart:nameEnd]] = struct{}{}
		}
	}
	return serviceNames
}

func lineCol(content []byte, index int) (int, int) {
	line := 1
	col := 1
	for i := 0; i < len(content) && i < index; i++ {
		if content[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}
