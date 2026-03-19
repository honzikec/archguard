package angular

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeAngularFile(t *testing.T) {
	p := Profile{}
	got := p.NormalizeFile("src/app/user-routing.module.ts")
	want := "src/app/user-routes.module.ts"
	if got != want {
		t.Fatalf("expected %q got %q", want, got)
	}
}

func TestDetectFromAngularJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "angular.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := Profile{}
	d := p.Detect([]string{dir})
	if !d.Matched {
		t.Fatalf("expected angular profile detection, got %+v", d)
	}
}
