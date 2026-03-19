package miner

import (
	"testing"

	"github.com/honzikec/archguard/internal/graph"
)

func TestNormalizeMiningInputsNextJS(t *testing.T) {
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

	ng, nf := normalizeMiningInputs(g, files, "nextjs")
	collapsed := "apps/web/app/(group)/blog/[param]"
	if ng.Nodes[collapsed] != 5 {
		t.Fatalf("expected collapsed nextjs node count 5, got %d", ng.Nodes[collapsed])
	}
	if ng.PackageEdges[collapsed]["next/navigation"] != 3 {
		t.Fatalf("expected package edge aggregation, got %+v", ng.PackageEdges[collapsed])
	}
	if len(ng.Edges[collapsed]) != 1 || ng.Edges[collapsed]["apps/web/lib/domain"] != 5 {
		t.Fatalf("expected edge aggregation into lib/domain, got %+v", ng.Edges[collapsed])
	}
	if nf[0] != "apps/web/app/(group)/blog/[param]/page.tsx" || nf[1] != "apps/web/app/(group)/blog/[param]/page.tsx" {
		t.Fatalf("unexpected normalized files: %+v", nf)
	}
}

func TestNormalizeNextJSSegment(t *testing.T) {
	cases := map[string]string{
		"(marketing)": "(group)",
		"(..)shop":    "(..)shop",
		"[slug]":      "[param]",
		"[[...slug]]": "[param]",
		"@modal":      "@slot",
		"components":  "components",
	}
	for input, expected := range cases {
		got := normalizeNextJSSegment(input)
		if got != expected {
			t.Fatalf("segment %q expected %q got %q", input, expected, got)
		}
	}
}
