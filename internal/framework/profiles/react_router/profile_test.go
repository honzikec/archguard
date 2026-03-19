package react_router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeReactRouterFile(t *testing.T) {
	p := Profile{}
	got := p.NormalizeFile("src/routes/teams.$teamId.tsx")
	want := "src/routes/teams.[param].tsx"
	if got != want {
		t.Fatalf("expected %q got %q", want, got)
	}
}

func TestDetectFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	content := `{"dependencies":{"react-router":"7.0.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p := Profile{}
	d := p.Detect([]string{dir})
	if !d.Matched {
		t.Fatalf("expected react_router profile detection, got %+v", d)
	}
}
