package miner

import (
	"path"
	"strings"

	"github.com/honzikec/archguard/internal/graph"
)

func normalizeMiningInputs(g *graph.Graph, allFiles []string, framework string) (*graph.Graph, []string) {
	framework = strings.ToLower(strings.TrimSpace(framework))
	if framework == "" || framework == "generic" {
		return g, allFiles
	}
	if framework != "nextjs" || g == nil {
		return g, allFiles
	}

	normalized := &graph.Graph{
		Nodes:        map[string]int{},
		Edges:        map[string]map[string]int{},
		PackageEdges: map[string]map[string]int{},
		FileEdges:    map[string]map[string]struct{}{},
	}

	for subtree, count := range g.Nodes {
		key := normalizeSubtreeForFramework(subtree, framework)
		normalized.Nodes[key] += count
	}

	for source, targets := range g.Edges {
		ns := normalizeSubtreeForFramework(source, framework)
		if _, ok := normalized.Edges[ns]; !ok {
			normalized.Edges[ns] = map[string]int{}
		}
		for target, count := range targets {
			nt := normalizeSubtreeForFramework(target, framework)
			if ns == nt {
				continue
			}
			normalized.Edges[ns][nt] += count
		}
	}

	for source, packages := range g.PackageEdges {
		ns := normalizeSubtreeForFramework(source, framework)
		if _, ok := normalized.PackageEdges[ns]; !ok {
			normalized.PackageEdges[ns] = map[string]int{}
		}
		for pkg, count := range packages {
			normalized.PackageEdges[ns][pkg] += count
		}
	}

	normalizedFiles := make([]string, 0, len(allFiles))
	for _, file := range allFiles {
		dir := normalizeSubtreeForFramework(path.Dir(file), framework)
		normalizedFiles = append(normalizedFiles, path.Join(dir, path.Base(file)))
	}

	return normalized, normalizedFiles
}

func normalizeSubtreeForFramework(subtree, framework string) string {
	framework = strings.ToLower(strings.TrimSpace(framework))
	if framework != "nextjs" {
		return subtree
	}
	return normalizeNextJSSubtree(subtree)
}

func normalizeNextJSSubtree(subtree string) string {
	subtree = path.Clean(strings.TrimSpace(subtree))
	if subtree == "." || subtree == "" {
		return "."
	}

	parts := strings.Split(subtree, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, normalizeNextJSSegment(part))
	}
	return path.Clean(strings.Join(out, "/"))
}

func normalizeNextJSSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return segment
	}

	if strings.HasPrefix(segment, "@") && len(segment) > 1 {
		return "@slot"
	}

	if strings.HasPrefix(segment, "(") && strings.HasSuffix(segment, ")") {
		return "(group)"
	}

	if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
		return "[param]"
	}

	return segment
}
