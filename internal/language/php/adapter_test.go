package php

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseFileExtractsUseAndInclude(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "src", "index.php")
	content := `<?php
use App\Services\UserService;
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
	if refs[0].RawImport != "App\\Services\\UserService" {
		t.Fatalf("expected normalized namespace import, got %+v", refs[0])
	}
	if refs[1].Kind != "php_include" || refs[2].Kind != "php_include" {
		t.Fatalf("expected php_include refs, got %+v %+v", refs[1], refs[2])
	}
}

func TestParseFileExtractsGroupedAliasedAndFunctionConstUse(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "src", "index.php")
	content := `<?php
use App\Infra\Db as MyDb;
use App\Support\{Arr as A, Str};
use function App\Support\helpers as h;
use const App\Support\MY_CONST;
class User {
    use HasFactory, Notifiable;
}
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

	imports := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Kind != "php_use" {
			continue
		}
		imports = append(imports, ref.RawImport)
	}

	want := []string{
		"App\\Infra\\Db",
		"App\\Support\\Arr",
		"App\\Support\\Str",
		"App\\Support\\helpers",
		"App\\Support\\MY_CONST",
	}
	if !reflect.DeepEqual(imports, want) {
		t.Fatalf("unexpected namespace imports\nwant: %#v\ngot:  %#v\nall refs: %+v", want, imports, refs)
	}
}

func TestParseFileIncludesOnlyStaticStringIncludes(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "src", "index.php")
	content := `<?php
require './a.php';
require_once('./b.php');
include "./c.php";
include_once("./d.php");
require getPath();
include $x;
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

	got := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Kind != "php_include" {
			continue
		}
		got = append(got, ref.RawImport)
	}
	want := []string{"./a.php", "./b.php", "./c.php", "./d.php"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected include extraction\nwant: %#v\ngot:  %#v\nall refs: %+v", want, got, refs)
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
