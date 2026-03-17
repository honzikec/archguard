package miner

import (
	"fmt"

	"github.com/honzikec/archguard/internal/graph"
)

type Candidate struct {
	Kind           string
	FromPaths      string
	ForbiddenPaths string
	Confidence     string
	Support        int
	Violations     int
}

func Propose(g *graph.Graph) []Candidate {
	var candidates []Candidate

	// First collect all known packages across the graph
	allPackages := make(map[string]bool)
	for _, packages := range g.PackageEdges {
		for pkg := range packages {
			allPackages[pkg] = true
		}
	}

	// Check all pairs of subtrees
	for subtreeA, totalFiles := range g.Nodes {
		for subtreeB := range g.Nodes {
			if subtreeA == subtreeB {
				continue
			}

			count := 0
			if targets, ok := g.Edges[subtreeA]; ok {
				count = targets[subtreeB]
			}

			prevalence := float64(count) / float64(totalFiles)

			if prevalence <= 0.02 && totalFiles >= 20 {
				confidence := "MEDIUM"
				if prevalence <= 0.01 && totalFiles >= 50 {
					confidence = "HIGH"
				}

				candidates = append(candidates, Candidate{
					Kind:           "import_boundary",
					FromPaths:      subtreeA + "/**",
					ForbiddenPaths: subtreeB + "/**",
					Confidence:     confidence,
					Support:        totalFiles,
					Violations:     count,
				})
			}
		}

		// Check banned packages prevalence for this subtree
		for pkg := range allPackages {
			count := 0
			if pkgs, ok := g.PackageEdges[subtreeA]; ok {
				count = pkgs[pkg]
			}

			prevalence := float64(count) / float64(totalFiles)

			if prevalence <= 0.02 && totalFiles >= 20 {
				confidence := "MEDIUM"
				if prevalence <= 0.01 && totalFiles >= 50 {
					confidence = "HIGH"
				}

				candidates = append(candidates, Candidate{
					Kind:           "banned_package",
					FromPaths:      subtreeA + "/**",
					ForbiddenPaths: pkg,
					Confidence:     confidence,
					Support:        totalFiles,
					Violations:     count,
				})
			}
		}
	}

	return candidates
}

func PrintCandidates(candidates []Candidate) {
	if len(candidates) == 0 {
		return
	}

	for i, c := range candidates {
		if i > 0 {
			fmt.Println("---")
		}
		fmt.Printf("Candidate rule:\n\n")
		if c.Kind == "banned_package" {
			fmt.Printf("%s should not import package %s\n\n", c.FromPaths, c.ForbiddenPaths)
		} else {
			fmt.Printf("%s should not import %s\n\n", c.FromPaths, c.ForbiddenPaths)
		}
		fmt.Printf("Confidence: %s\n", c.Confidence)
		fmt.Printf("Support: %d files\n", c.Support)
		fmt.Printf("Violations: %d\n", c.Violations)
	}
}

func PrintCandidatesYAML(candidates []Candidate) {
	if len(candidates) == 0 {
		return
	}

	fmt.Println("version: 1")
	fmt.Println()
	fmt.Println("rules:")

	for i, c := range candidates {
		fmt.Printf("  - id: MINED-%03d\n", i+1)
		fmt.Printf("    kind: %s\n", c.Kind)
		fmt.Printf("    severity: warning\n")
		
		if c.Kind == "banned_package" {
			fmt.Printf("    rationale: \"Mined invariant: %s should not import package %s\"\n", c.FromPaths, c.ForbiddenPaths)
			fmt.Printf("    conditions:\n")
			fmt.Printf("      from_paths:\n")
			fmt.Printf("        - \"%s\"\n", c.FromPaths)
			fmt.Printf("      forbidden_packages:\n")
			fmt.Printf("        - \"%s\"\n", c.ForbiddenPaths)
		} else {
			fmt.Printf("    rationale: \"Mined invariant: %s should not import %s\"\n", c.FromPaths, c.ForbiddenPaths)
			fmt.Printf("    conditions:\n")
			fmt.Printf("      from_paths:\n")
			fmt.Printf("        - \"%s\"\n", c.FromPaths)
			fmt.Printf("      forbidden_paths:\n")
			fmt.Printf("        - \"%s\"\n", c.ForbiddenPaths)
		}
		fmt.Println()
	}
}
