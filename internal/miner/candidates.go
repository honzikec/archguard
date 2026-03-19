package miner

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/graph"
)

type Options struct {
	MinSupport    int
	MaxPrevalence float64
	Framework     string
}

type Candidate struct {
	Kind       string   `json:"kind" yaml:"kind"`
	Scope      []string `json:"scope" yaml:"scope"`
	Target     []string `json:"target,omitempty" yaml:"target,omitempty"`
	Severity   string   `json:"severity" yaml:"severity"`
	Support    int      `json:"support" yaml:"support"`
	Violations int      `json:"violations" yaml:"violations"`
	Prevalence float64  `json:"prevalence" yaml:"prevalence"`
	Confidence string   `json:"confidence" yaml:"confidence"`
	Evidence   string   `json:"evidence" yaml:"evidence"`
}

func Propose(g *graph.Graph, allFiles []string, opts Options) []Candidate {
	if opts.MinSupport <= 0 {
		opts.MinSupport = 20
	}
	if opts.MaxPrevalence <= 0 {
		opts.MaxPrevalence = 0.02
	}
	g, allFiles = normalizeMiningInputs(g, allFiles, opts.Framework)

	candidates := make([]Candidate, 0)
	candidates = append(candidates, proposeNoImport(g, opts)...)
	candidates = append(candidates, proposeNoPackage(g, opts)...)
	candidates = append(candidates, proposeFilePattern(allFiles)...)
	candidates = append(candidates, proposeNoCycle(g)...)

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Kind != candidates[j].Kind {
			return candidates[i].Kind < candidates[j].Kind
		}
		if len(candidates[i].Scope) == 0 || len(candidates[j].Scope) == 0 {
			return len(candidates[i].Scope) < len(candidates[j].Scope)
		}
		return candidates[i].Scope[0] < candidates[j].Scope[0]
	})
	return candidates
}

func proposeNoImport(g *graph.Graph, opts Options) []Candidate {
	candidates := make([]Candidate, 0)
	for sourceSubtree, totalFiles := range g.Nodes {
		if totalFiles < opts.MinSupport {
			continue
		}
		for targetSubtree := range g.Nodes {
			if sourceSubtree == targetSubtree {
				continue
			}
			violations := 0
			if edges, ok := g.Edges[sourceSubtree]; ok {
				violations = edges[targetSubtree]
			}
			prevalence := float64(violations) / float64(totalFiles)
			if prevalence > opts.MaxPrevalence {
				continue
			}
			candidates = append(candidates, Candidate{
				Kind:       config.KindNoImport,
				Scope:      []string{sourceSubtree + "/**"},
				Target:     []string{targetSubtree + "/**"},
				Severity:   config.SeverityWarning,
				Support:    totalFiles,
				Violations: violations,
				Prevalence: prevalence,
				Confidence: confidence(prevalence, totalFiles),
				Evidence:   fmt.Sprintf("%d/%d files in %s import %s", violations, totalFiles, sourceSubtree, targetSubtree),
			})
		}
	}
	return candidates
}

func proposeNoPackage(g *graph.Graph, opts Options) []Candidate {
	candidates := make([]Candidate, 0)
	allPackages := map[string]struct{}{}
	for _, packages := range g.PackageEdges {
		for pkg := range packages {
			allPackages[pkg] = struct{}{}
		}
	}
	for sourceSubtree, totalFiles := range g.Nodes {
		if totalFiles < opts.MinSupport {
			continue
		}
		for pkg := range allPackages {
			violations := 0
			if edges, ok := g.PackageEdges[sourceSubtree]; ok {
				violations = edges[pkg]
			}
			prevalence := float64(violations) / float64(totalFiles)
			if prevalence > opts.MaxPrevalence {
				continue
			}
			candidates = append(candidates, Candidate{
				Kind:       config.KindNoPackage,
				Scope:      []string{sourceSubtree + "/**"},
				Target:     []string{pkg},
				Severity:   config.SeverityWarning,
				Support:    totalFiles,
				Violations: violations,
				Prevalence: prevalence,
				Confidence: confidence(prevalence, totalFiles),
				Evidence:   fmt.Sprintf("%d/%d files in %s import %s", violations, totalFiles, sourceSubtree, pkg),
			})
		}
	}
	return candidates
}

func proposeFilePattern(allFiles []string) []Candidate {
	byDir := map[string][]string{}
	for _, f := range allFiles {
		dir := path.Dir(f)
		byDir[dir] = append(byDir[dir], path.Base(f))
	}

	candidates := make([]Candidate, 0)
	for dir, files := range byDir {
		if len(files) < 5 {
			continue
		}
		suffixCount := map[string]int{}
		for _, file := range files {
			parts := strings.SplitN(file, ".", 2)
			if len(parts) == 2 {
				suffixCount["."+parts[1]]++
			}
		}
		bestSuffix := ""
		bestCount := 0
		for suffix, c := range suffixCount {
			if c > bestCount {
				bestCount = c
				bestSuffix = suffix
			}
		}
		if bestCount == 0 {
			continue
		}
		prevalence := float64(len(files)-bestCount) / float64(len(files))
		if prevalence > 0.20 {
			continue
		}
		regex := "^.*" + regexp.QuoteMeta(bestSuffix) + "$"
		candidates = append(candidates, Candidate{
			Kind:       config.KindFilePattern,
			Scope:      []string{dir + "/**"},
			Target:     []string{regex},
			Severity:   config.SeverityWarning,
			Support:    len(files),
			Violations: len(files) - bestCount,
			Prevalence: prevalence,
			Confidence: "HIGH",
			Evidence:   fmt.Sprintf("%d/%d files in %s match suffix %s", bestCount, len(files), dir, bestSuffix),
		})
	}
	return candidates
}

func proposeNoCycle(g *graph.Graph) []Candidate {
	cycles := DetectCycles(g)
	candidates := make([]Candidate, 0, len(cycles))
	for _, c := range cycles {
		if len(c.Chain) == 0 {
			continue
		}
		first := c.Chain[0]
		candidates = append(candidates, Candidate{
			Kind:       config.KindNoCycle,
			Scope:      []string{first + "/**"},
			Severity:   config.SeverityError,
			Support:    len(c.Chain) - 1,
			Violations: 1,
			Prevalence: 1.0,
			Confidence: "HIGH",
			Evidence:   strings.Join(c.Chain, " -> "),
		})
	}
	return candidates
}

func confidence(prevalence float64, support int) string {
	if prevalence <= 0.01 && support >= 50 {
		return "HIGH"
	}
	if prevalence <= 0.02 && support >= 20 {
		return "MEDIUM"
	}
	return "LOW"
}

func PrintText(candidates []Candidate) {
	if len(candidates) == 0 {
		fmt.Println("No candidates discovered with current thresholds.")
		return
	}
	for i, c := range candidates {
		if i > 0 {
			fmt.Println("---")
		}
		fmt.Printf("kind: %s\n", c.Kind)
		fmt.Printf("scope: %v\n", c.Scope)
		if len(c.Target) > 0 {
			fmt.Printf("target: %v\n", c.Target)
		}
		fmt.Printf("severity: %s\n", c.Severity)
		fmt.Printf("support: %d\n", c.Support)
		fmt.Printf("violations: %d\n", c.Violations)
		fmt.Printf("prevalence: %.4f\n", c.Prevalence)
		fmt.Printf("confidence: %s\n", c.Confidence)
		fmt.Printf("evidence: %s\n", c.Evidence)
	}
}

func PrintJSON(candidates []Candidate) {
	data, _ := json.MarshalIndent(candidates, "", "  ")
	fmt.Println(string(data))
}

func PrintYAML(candidates []Candidate) {
	fmt.Println("version: 1")
	fmt.Println("rules:")
	for i, c := range candidates {
		fmt.Printf("  - id: MINED-%03d\n", i+1)
		fmt.Printf("    kind: %s\n", c.Kind)
		fmt.Printf("    severity: %s\n", c.Severity)
		fmt.Printf("    scope:\n")
		for _, s := range c.Scope {
			fmt.Printf("      - %q\n", s)
		}
		if len(c.Target) > 0 {
			fmt.Printf("    target:\n")
			for _, t := range c.Target {
				fmt.Printf("      - %q\n", t)
			}
		}
		fmt.Printf("    message: %q\n", c.Evidence)
	}
}

func EmitStarterConfig(candidates []Candidate) string {
	var b strings.Builder
	b.WriteString("version: 1\n")
	b.WriteString("project:\n")
	b.WriteString("  roots: [\".\"]\n")
	b.WriteString("  include: [\"**/*.ts\", \"**/*.tsx\", \"**/*.js\", \"**/*.jsx\", \"**/*.mjs\", \"**/*.cjs\"]\n")
	b.WriteString("  exclude: [\"**/node_modules/**\", \"**/dist/**\", \"**/build/**\", \"**/.next/**\", \"**/coverage/**\", \"**/.git/**\"]\n")
	b.WriteString("rules:\n")
	for i, c := range candidates {
		b.WriteString(fmt.Sprintf("  - id: MINED-%03d\n", i+1))
		b.WriteString(fmt.Sprintf("    kind: %s\n", c.Kind))
		b.WriteString(fmt.Sprintf("    severity: %s\n", c.Severity))
		b.WriteString("    scope:\n")
		for _, s := range c.Scope {
			b.WriteString(fmt.Sprintf("      - %q\n", s))
		}
		if len(c.Target) > 0 {
			b.WriteString("    target:\n")
			for _, t := range c.Target {
				b.WriteString(fmt.Sprintf("      - %q\n", t))
			}
		}
		b.WriteString(fmt.Sprintf("    message: %q\n", c.Evidence))
	}
	return b.String()
}
