package framework

import (
	"path"

	"github.com/honzikec/archguard/internal/graph"
)

func NormalizeMiningInputs(g *graph.Graph, allFiles []string, profileID string) (*graph.Graph, []string, NormalizationStats) {
	stats := NormalizationStats{
		OriginalFiles: len(allFiles),
	}
	if g != nil {
		stats.OriginalNodes = len(g.Nodes)
	}
	if profileID == "" || profileID == "generic" || g == nil {
		stats.NormalizedNodes = stats.OriginalNodes
		stats.NormalizedFiles = stats.OriginalFiles
		return g, allFiles, stats
	}

	p, ok := FindProfile(profileID)
	if !ok {
		stats.NormalizedNodes = stats.OriginalNodes
		stats.NormalizedFiles = stats.OriginalFiles
		return g, allFiles, stats
	}

	normalized := &graph.Graph{
		Nodes:        map[string]int{},
		Edges:        map[string]map[string]int{},
		PackageEdges: map[string]map[string]int{},
		FileEdges:    map[string]map[string]struct{}{},
	}

	for subtree, count := range g.Nodes {
		key := p.NormalizeSubtree(subtree)
		normalized.Nodes[key] += count
	}

	for source, targets := range g.Edges {
		ns := p.NormalizeSubtree(source)
		if _, exists := normalized.Edges[ns]; !exists {
			normalized.Edges[ns] = map[string]int{}
		}
		for target, count := range targets {
			nt := p.NormalizeSubtree(target)
			if ns == nt {
				continue
			}
			normalized.Edges[ns][nt] += count
		}
	}

	for source, packages := range g.PackageEdges {
		ns := p.NormalizeSubtree(source)
		if _, exists := normalized.PackageEdges[ns]; !exists {
			normalized.PackageEdges[ns] = map[string]int{}
		}
		for pkg, count := range packages {
			normalized.PackageEdges[ns][pkg] += count
		}
	}

	normalizedFiles := make([]string, 0, len(allFiles))
	for _, file := range allFiles {
		nf := p.NormalizeFile(file)
		if nf == "" {
			dir := p.NormalizeSubtree(path.Dir(file))
			nf = path.Join(dir, path.Base(file))
		}
		normalizedFiles = append(normalizedFiles, path.Clean(nf))
	}

	stats.NormalizedNodes = len(normalized.Nodes)
	stats.NormalizedFiles = len(normalizedFiles)
	return normalized, normalizedFiles, stats
}
