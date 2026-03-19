package react

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFromPackageAndSrc(t *testing.T) {
	dir := t.TempDir()
	content := `{"dependencies":{"react":"19.0.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}

	p := Profile{}
	d := p.Detect([]string{dir})
	if !d.Matched {
		t.Fatalf("expected react profile detection, got %+v", d)
	}
}

func TestDetectSkipsWhenReactRouterPresent(t *testing.T) {
	dir := t.TempDir()
	content := `{"dependencies":{"react":"19.0.0","react-router-dom":"7.0.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}

	p := Profile{}
	d := p.Detect([]string{dir})
	if d.Matched {
		t.Fatalf("expected react profile to skip router projects, got %+v", d)
	}
}

func TestNormalizeReactFile(t *testing.T) {
	p := Profile{}
	got := p.NormalizeFile("src/__tests__/service/user-card.stories.tsx")
	want := "src/tests/service/user-card.tsx"
	if got != want {
		t.Fatalf("expected %q got %q", want, got)
	}
}
