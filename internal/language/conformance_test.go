package language

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/honzikec/archguard/internal/conformance"
)

func TestLanguageAdaptersConformance(t *testing.T) {
	root := t.TempDir()

	samples := map[string]string{
		"src/sample.ts": `import a from "./a"
export { b } from "./b"
`,
		"src/sample.js": `const dep = require("./dep")
`,
		"src/sample.php": `<?php
use App\Service\UserService;
require_once "./bootstrap.php";
`,
		"src/view.phtml": `<?php include "./header.php"; ?>`,
		"src/readme.txt": `plain text`,
	}

	samplePaths := make([]string, 0, len(samples))
	for rel, content := range samples {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		samplePaths = append(samplePaths, abs)
	}
	sort.Strings(samplePaths)

	if err := conformance.ValidateLanguageAdapters(RegisteredAdapters(), []string{root}, samplePaths); err != nil {
		t.Fatalf("language conformance failed: %v", err)
	}
}
