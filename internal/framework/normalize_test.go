package framework

import (
	"testing"

	"github.com/honzikec/archguard/internal/graph"
)

func TestNormalizeMiningInputsNextJSAggregatesRoutes(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]int{
			"apps/web/app/(marketing)/blog/[slug]": 2,
			"apps/web/app/(shop)/blog/[id]":        3,
			"apps/web/lib/domain":                  5,
		},
		Edges: map[string]map[string]int{
			"apps/web/app/(marketing)/blog/[slug]": {
				"apps/web/lib/domain": 2,
			},
			"apps/web/app/(shop)/blog/[id]": {
				"apps/web/lib/domain": 3,
			},
		},
		PackageEdges: map[string]map[string]int{
			"apps/web/app/(marketing)/blog/[slug]": {"next/navigation": 1},
			"apps/web/app/(shop)/blog/[id]":        {"next/navigation": 2},
		},
		FileEdges: map[string]map[string]struct{}{},
	}
	files := []string{
		"apps/web/app/(marketing)/blog/[slug]/page.tsx",
		"apps/web/app/(shop)/blog/[id]/page.tsx",
	}

	normalized, normalizedFiles, stats := NormalizeMiningInputs(g, files, "nextjs")
	collapsed := "apps/web/app/(group)/blog/[param]"
	if normalized.Nodes[collapsed] != 5 {
		t.Fatalf("expected collapsed node count 5, got %d", normalized.Nodes[collapsed])
	}
	if normalized.PackageEdges[collapsed]["next/navigation"] != 3 {
		t.Fatalf("expected package edge aggregation, got %+v", normalized.PackageEdges[collapsed])
	}
	if len(normalized.Edges[collapsed]) != 1 || normalized.Edges[collapsed]["apps/web/lib/domain"] != 5 {
		t.Fatalf("expected edge aggregation into lib/domain, got %+v", normalized.Edges[collapsed])
	}
	if normalizedFiles[0] != "apps/web/app/(group)/blog/[param]/page.tsx" {
		t.Fatalf("unexpected normalized files: %+v", normalizedFiles)
	}
	if stats.OriginalNodes != 3 || stats.NormalizedNodes != 2 {
		t.Fatalf("unexpected normalization stats: %+v", stats)
	}
}

func TestNormalizeMiningInputsReactNativeCollapsesPlatformFiles(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]int{
			"src/screens": 2,
		},
		Edges:        map[string]map[string]int{},
		PackageEdges: map[string]map[string]int{},
		FileEdges:    map[string]map[string]struct{}{},
	}
	files := []string{"src/screens/Home.ios.tsx", "src/screens/Home.android.tsx"}

	_, normalizedFiles, _ := NormalizeMiningInputs(g, files, "react_native")
	if normalizedFiles[0] != "src/screens/Home.platform.tsx" || normalizedFiles[1] != "src/screens/Home.platform.tsx" {
		t.Fatalf("expected collapsed platform files, got %+v", normalizedFiles)
	}
}
