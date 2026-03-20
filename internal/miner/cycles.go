package miner

import (
	"sort"
	"strings"

	"github.com/honzikec/archguard/internal/graph"
)

type CycleFinding struct {
	Chain []string
}

type CycleComponent struct {
	Nodes []string
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

func DetectCycleComponents(g *graph.Graph) []CycleComponent {
	if g == nil {
		return nil
	}

	components := stronglyConnectedComponents(g)
	out := make([]CycleComponent, 0, len(components))
	for _, nodes := range components {
		if len(nodes) == 0 {
			continue
		}
		if len(nodes) == 1 {
			only := nodes[0]
			if g.Edges[only][only] <= 0 {
				continue
			}
		}
		out = append(out, CycleComponent{Nodes: nodes})
	}
	return out
}

func stronglyConnectedComponents(g *graph.Graph) [][]string {
	indices := map[string]int{}
	lowlink := map[string]int{}
	onStack := map[string]bool{}
	stack := make([]string, 0)
	components := make([][]string, 0)
	index := 0

	var visit func(string)
	visit = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		neighbors := make([]string, 0, len(g.Edges[v]))
		for n := range g.Edges[v] {
			neighbors = append(neighbors, n)
		}
		sort.Strings(neighbors)

		for _, w := range neighbors {
			if _, seen := indices[w]; !seen {
				visit(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
				continue
			}
			if onStack[w] && indices[w] < lowlink[v] {
				lowlink[v] = indices[w]
			}
		}

		if lowlink[v] != indices[v] {
			return
		}

		component := make([]string, 0)
		for len(stack) > 0 {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			component = append(component, w)
			if w == v {
				break
			}
		}
		sort.Strings(component)
		components = append(components, component)
	}

	nodes := make([]string, 0, len(g.Nodes))
	for n := range g.Nodes {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)

	for _, n := range nodes {
		if _, seen := indices[n]; seen {
			continue
		}
		visit(n)
	}

	sort.Slice(components, func(i, j int) bool {
		if len(components[i]) == 0 || len(components[j]) == 0 {
			return len(components[i]) < len(components[j])
		}
		return components[i][0] < components[j][0]
	})
	return components
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
