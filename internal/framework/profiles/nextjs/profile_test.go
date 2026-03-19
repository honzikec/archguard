package nextjs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeSegment(t *testing.T) {
	cases := map[string]string{
		"(marketing)": "(group)",
		"[slug]":      "[param]",
		"[[...slug]]": "[param]",
		"@modal":      "@slot",
		"components":  "components",
	}
	for input, expected := range cases {
		if got := normalizeSegment(input); got != expected {
			t.Fatalf("segment %q expected %q got %q", input, expected, got)
		}
	}
}

func TestDetectFromNextConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "next.config.js"), []byte("module.exports={}"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := Profile{}
	d := p.Detect([]string{dir})
	if !d.Matched {
		t.Fatalf("expected nextjs profile detection, got %+v", d)
	}
}
