package pkgid

import "testing"

func TestCanonical(t *testing.T) {
	cases := map[string]string{
		"react":                "react",
		"react-dom/client":     "react-dom",
		"lodash/fp":            "lodash",
		"@types/node/fs":       "@types/node",
		"@scope/pkg/sub/path":  "@scope/pkg",
		"node:fs/promises":     "node:fs",
		"node:timers":          "node:timers",
		"yii\\web\\Controller": "yii",
		"Common\\Models\\User": "Common",
		"./local/module":       "",
		"/abs/path/module":     "",
		"../up/module":         "",
		"react-dom/client?x=1": "react-dom",
		"@scope/pkg/sub#hash":  "@scope/pkg",
		"  react-dom/server  ": "react-dom",
	}

	for input, want := range cases {
		if got := Canonical(input); got != want {
			t.Fatalf("canonical(%q): expected %q got %q", input, want, got)
		}
	}
}
