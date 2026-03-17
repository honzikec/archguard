package miner

import (
	"github.com/honzikec/archguard/internal/graph"
)

// CycleFinding represents a detected import cycle between subtrees.
type CycleFinding struct {
	Chain []string // ordered list of subtrees forming the cycle
}

// DetectCycles finds cycles in the dependency graph using DFS.
func DetectCycles(g *graph.Graph) []CycleFinding {
	var findings []CycleFinding
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	stack := []string{}

	var dfs func(node string)
	dfs = func(node string) {
		visited[node] = true
		inStack[node] = true
		stack = append(stack, node)

		for neighbor := range g.Edges[node] {
			if !visited[neighbor] {
				dfs(neighbor)
			} else if inStack[neighbor] {
				// Found a cycle — extract the cycle chain
				cycle := []string{}
				for i := len(stack) - 1; i >= 0; i-- {
					cycle = append([]string{stack[i]}, cycle...)
					if stack[i] == neighbor {
						break
					}
				}
				cycle = append(cycle, neighbor) // close the loop
				findings = append(findings, CycleFinding{Chain: cycle})
			}
		}

		stack = stack[:len(stack)-1]
		inStack[node] = false
	}

	for node := range g.Nodes {
		if !visited[node] {
			dfs(node)
		}
	}

	return findings
}
