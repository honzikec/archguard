package miner

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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

// ProposeFileConventions infers file naming conventions from observed suffixes.
// If ≥80% of files in a directory share a common double-extension (e.g. .service.ts),
// it proposes a file_convention rule for that directory.
func ProposeFileConventions(allFiles []string) []Candidate {
	var candidates []Candidate

	// Group files by directory
	byDir := make(map[string][]string)
	for _, f := range allFiles {
		dir := filepath.Dir(f)
		byDir[dir] = append(byDir[dir], filepath.Base(f))
	}

	for dir, files := range byDir {
		if len(files) < 5 {
			continue
		}

		// Count double-extension suffixes (e.g. ".service.ts", ".controller.ts")
		suffixCount := make(map[string]int)
		for _, name := range files {
			parts := strings.SplitN(name, ".", 2)
			if len(parts) == 2 {
				suffix := "." + parts[1]
				suffixCount[suffix]++
			}
		}

		// Find dominant suffix
		type suffixEntry struct {
			suffix string
			count  int
		}
		var entries []suffixEntry
		for s, c := range suffixCount {
			entries = append(entries, suffixEntry{s, c})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].count > entries[j].count
		})

		if len(entries) == 0 {
			continue
		}

		top := entries[0]
		prevalence := float64(top.count) / float64(len(files))
		if prevalence >= 0.80 {
			// Build a regex from the suffix, e.g. ".service.ts" -> ".*\.service\.ts$"
			escaped := regexp.QuoteMeta(top.suffix)
			regexStr := ".*" + escaped + "$"
			candidates = append(candidates, Candidate{
				Kind:           "file_convention",
				FromPaths:      dir + "/**",
				ForbiddenPaths: regexStr, // repurposed field to carry the regex
				Confidence:     "HIGH",
				Support:        len(files),
				Violations:     len(files) - top.count,
			})
		}
	}

	return candidates
}

// ProposeCrossAppRules detects when apps/* imports directly from another apps/* subtree.
func ProposeCrossAppRules(g *graph.Graph) []Candidate {
	var candidates []Candidate

	for subtreeA, edges := range g.Edges {
		partsA := strings.SplitN(subtreeA, "/", 3)
		if len(partsA) < 2 || partsA[0] != "apps" {
			continue
		}
		appA := partsA[0] + "/" + partsA[1]

		for subtreeB, count := range edges {
			if count == 0 {
				continue
			}
			partsB := strings.SplitN(subtreeB, "/", 3)
			if len(partsB) < 2 || partsB[0] != "apps" {
				continue
			}
			appB := partsB[0] + "/" + partsB[1]
			if appA == appB {
				continue // same app, fine
			}

			totalFiles := g.Nodes[subtreeA]
			candidates = append(candidates, Candidate{
				Kind:           "import_boundary",
				FromPaths:      appA + "/**",
				ForbiddenPaths: appB + "/**",
				Confidence:     "HIGH",
				Support:        totalFiles,
				Violations:     count,
			})
		}
	}

	// Deduplicate by (FromPaths, ForbiddenPaths)
	seen := make(map[string]bool)
	var deduped []Candidate
	for _, c := range candidates {
		key := c.FromPaths + "→" + c.ForbiddenPaths
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, c)
		}
	}
	return deduped
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
		switch c.Kind {
		case "banned_package":
			fmt.Printf("%s should not import package %s\n\n", c.FromPaths, c.ForbiddenPaths)
		case "file_convention":
			fmt.Printf("Files in %s should match pattern: %s\n\n", c.FromPaths, c.ForbiddenPaths)
		default:
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

		switch c.Kind {
		case "banned_package":
			fmt.Printf("    rationale: \"Mined invariant: %s should not import package %s\"\n", c.FromPaths, c.ForbiddenPaths)
			fmt.Printf("    conditions:\n")
			fmt.Printf("      from_paths:\n")
			fmt.Printf("        - \"%s\"\n", c.FromPaths)
			fmt.Printf("      forbidden_packages:\n")
			fmt.Printf("        - \"%s\"\n", c.ForbiddenPaths)
		case "file_convention":
			fmt.Printf("    rationale: \"Mined convention: files in %s should match %s\"\n", c.FromPaths, c.ForbiddenPaths)
			fmt.Printf("    conditions:\n")
			fmt.Printf("      path_patterns:\n")
			fmt.Printf("        - \"%s\"\n", c.FromPaths)
			fmt.Printf("      filename_regex: \"%s\"\n", c.ForbiddenPaths)
		default:
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
