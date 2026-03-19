package fileset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/config"
	"github.com/honzikec/archguard/internal/language/contracts"
	"github.com/honzikec/archguard/internal/model"
)

type fakeAdapter struct{}

func (fakeAdapter) ID() string { return "fake" }
func (fakeAdapter) Detect(_ []string) contracts.Detection {
	return contracts.Detection{Matched: true}
}
func (fakeAdapter) SupportsFile(path string) bool {
	return filepath.Ext(path) == ".foo"
}
func (fakeAdapter) ParseFile(_ string) ([]model.ImportRef, error) { return nil, nil }

func TestDiscoverWithAdapterUsesSupportedFileFilter(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.foo"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.ts"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverWithAdapter(config.ProjectSettings{Roots: []string{"."}, Include: []string{"**"}}, fakeAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "a.foo" {
		t.Fatalf("expected adapter-filtered files, got %+v", files)
	}
}
