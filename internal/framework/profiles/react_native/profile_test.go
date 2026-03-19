package react_native

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollapsePlatformSuffix(t *testing.T) {
	cases := map[string]string{
		"Home.ios.tsx":     "Home.platform.tsx",
		"Home.android.tsx": "Home.platform.tsx",
		"Home.native.tsx":  "Home.platform.tsx",
		"Home.web.tsx":     "Home.platform.tsx",
		"Home.tsx":         "Home.tsx",
	}
	for input, expected := range cases {
		if got := collapsePlatformSuffix(input); got != expected {
			t.Fatalf("input %q expected %q got %q", input, expected, got)
		}
	}
}

func TestDetectFromPlatformFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "src", "screens", "Home.ios.tsx")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("export const Home = () => null"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := Profile{}
	d := p.Detect([]string{dir})
	if !d.Matched {
		t.Fatalf("expected react_native profile detection, got %+v", d)
	}
}
