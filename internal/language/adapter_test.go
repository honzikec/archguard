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
	if len(first) < 2 {
		t.Fatalf("expected javascript and php adapters, got %+v", first)
	}
}

func TestResolveExplicitLanguage(t *testing.T) {
	res := Resolve("php", nil)
	if res.Selected != "php" || res.Reason != "explicit" || res.Adapter == nil {
		t.Fatalf("expected explicit php adapter, got %+v", res)
	}
}

func TestResolveAutoDetectsPHP(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app", "index.php")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("<?php include './db.php';"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Resolve("", []string{dir})
	if res.Selected != "php" || res.Reason != "auto_detected" {
		t.Fatalf("expected php autodetection, got %+v", res)
	}
}

func TestResolveJavaScriptAdapterDeterministicParse(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.ts")
	content := "import x from 'a'\nexport { y } from 'b'\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Resolve("javascript", []string{dir})
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
