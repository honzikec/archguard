package framework

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honzikec/archguard/internal/conformance"
)

func TestFrameworkProfilesConformance(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"dependencies":{"next":"15.0.0","react":"19.0.0","react-router-dom":"7.0.0","react-native":"0.76.0","@angular/core":"19.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "next.config.js"), []byte("module.exports={}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "angular.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src", "routes"), 0o755); err != nil {
		t.Fatal(err)
	}

	subtrees := []string{
		"src/app/(marketing)/blog/[slug]",
		"src/routes/:id",
		"src/routes/$teamId",
		"src/features/account",
		`src\legacy\route\:id`,
	}
	files := []string{
		"src/app/(marketing)/blog/[slug]/page.tsx",
		"src/routes/teams.$teamId.tsx",
		"src/screens/Home.ios.tsx",
		"src/app/users-routing.module.ts",
		`src\features\account\index.spec.ts`,
	}

	profiles := RegisteredProfiles()
	if err := conformance.ValidateFrameworkProfiles(profiles, []string{root}, subtrees, files); err != nil {
		t.Fatalf("framework conformance failed: %v", err)
	}
}
