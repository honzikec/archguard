package language

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRegisteredLanguagesDeterministic(t *testing.T) {
	first := RegisteredLanguages()
	second := RegisteredLanguages()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic language list, got %v vs %v", first, second)
	}
	if len(first) == 0 {
		t.Fatal("expected at least one language adapter")
	}
}

func TestJavaScriptAdapterDeterministicParse(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.ts")
	content := "import x from 'a'\nexport { y } from 'b'\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Resolve([]string{dir})
	if res.Adapter == nil {
		t.Fatal("expected resolved adapter")
	}
	first, err := res.Adapter.ParseFile(file)
	if err != nil {
		t.Fatal(err)
	}
	second, err := res.Adapter.ParseFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic parse output, got %+v vs %+v", first, second)
	}
}
