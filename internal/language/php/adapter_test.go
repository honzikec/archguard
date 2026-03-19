package php

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileExtractsUseAndInclude(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "src", "index.php")
	content := `<?php
use App\\Services\\UserService;
require_once './bootstrap.php';
include "./infra/db.php";
`
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := Adapter{}
	refs, err := adapter.ParseFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d (%+v)", len(refs), refs)
	}
	if refs[0].Kind != "php_use" {
		t.Fatalf("expected first ref kind php_use, got %+v", refs[0])
	}
	if refs[1].Kind != "php_include" || refs[2].Kind != "php_include" {
		t.Fatalf("expected php_include refs, got %+v %+v", refs[1], refs[2])
	}
}

func TestDetectPHP(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"name":"demo/app"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := Adapter{}
	d := adapter.Detect([]string{dir})
	if !d.Matched {
		t.Fatalf("expected php detection, got %+v", d)
	}
}
