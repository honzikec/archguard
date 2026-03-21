package pathutil_test

import (
	"testing"

	"github.com/honzikec/archguard/internal/pathutil"
)

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern string
		input   string
		match   bool
	}{
		{pattern: "src/**", input: "src/a/b.ts", match: true},
		{pattern: "src/*.ts", input: "src/a/b.ts", match: false},
		{pattern: "**/test/**", input: "packages/core/test/a.ts", match: true},
		{pattern: "src/**/index.ts", input: "src\\infra\\index.ts", match: true},
		{pattern: "", input: "src/a.ts", match: false},
		{pattern: "[", input: "src/a.ts", match: false},
	}

	for _, tc := range cases {
		got := pathutil.MatchGlob(tc.pattern, tc.input)
		if got != tc.match {
			t.Fatalf("MatchGlob(%q, %q): expected %t, got %t", tc.pattern, tc.input, tc.match, got)
		}
	}
}

func TestMatchAny(t *testing.T) {
	patterns := []string{"src/**", "packages/**"}
	if !pathutil.MatchAny(patterns, "packages/core/a.ts") {
		t.Fatalf("expected MatchAny to match packages/core/a.ts")
	}
	if pathutil.MatchAny(patterns, "vendor/a.ts") {
		t.Fatalf("expected MatchAny to not match vendor/a.ts")
	}
}
