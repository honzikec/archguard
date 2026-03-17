package miner

import (
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/graph"
)

type CycleFinding struct {
	Chain []string
}

func DetectCycles(g *graph.Graph) []CycleFinding {
	if g == nil {
		return nil
	}
	visited := map[string]bool{}
	inStack := map[string]int{}
	stack := []string{}
	seen := map[string]struct{}{}
	result := make([]CycleFinding, 0)

	var dfs func(string)
	dfs = func(node string) {
		visited[node] = true
		inStack[node] = len(stack)
		stack = append(stack, node)

		neighbors := make([]string, 0, len(g.Edges[node]))
		for n := range g.Edges[node] {
			neighbors = append(neighbors, n)
		}
		sort.Strings(neighbors)

		for _, neighbor := range neighbors {
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
			key := canonicalCycle(cycle)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, CycleFinding{Chain: cycle})
		}

		delete(inStack, node)
		stack = stack[:len(stack)-1]
	}

	nodes := make([]string, 0, len(g.Nodes))
	for n := range g.Nodes {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)

	for _, node := range nodes {
		if !visited[node] {
			dfs(node)
		}
	}
	return result
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
