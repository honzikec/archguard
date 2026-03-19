package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func runInit(args []string) int {
	if len(args) > 0 && args[0] == "profile" {
		return runInitProfile(args[1:])
	}

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	setFlagSetOutput(fs)
	common := bindCommonFlags(fs, commonFlags{configPath: "archguard.yaml", format: "text"})
	force := fs.Bool("force", false, "Overwrite existing config file")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if _, err := os.Stat(common.configPath); err == nil && !*force {
		fmt.Fprintf(os.Stderr, "config already exists: %s (use --force to overwrite)\n", common.configPath)
		return 2
	}

	if err := os.WriteFile(common.configPath, []byte(defaultConfigTemplate()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write config: %v\n", err)
		return 2
	}

	if !common.quiet {
		fmt.Printf("Created %s\n", common.configPath)
	}
	return 0
}

func runInitProfile(args []string) int {
	fs := flag.NewFlagSet("init profile", flag.ContinueOnError)
	setFlagSetOutput(fs)
	common := bindCommonFlags(fs, commonFlags{configPath: "archguard.yaml", format: "text"})
	name := fs.String("name", "", "Framework profile id (for example react_router)")
	dir := fs.String("dir", "internal/framework/profiles", "Target base directory for generated scaffold")
	force := fs.Bool("force", false, "Overwrite generated files if they already exist")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	profileID := normalizeProfileID(*name)
	if profileID == "" {
		fmt.Fprintln(os.Stderr, "profile name is required (use --name)")
		return 2
	}
	packageName := normalizePackageName(profileID)

	targetDir := filepath.Join(*dir, packageName)
	profilePath := filepath.Join(targetDir, "profile.go")
	testPath := filepath.Join(targetDir, "profile_test.go")
	if !*force {
		if _, err := os.Stat(profilePath); err == nil {
			fmt.Fprintf(os.Stderr, "profile scaffold already exists: %s (use --force to overwrite)\n", profilePath)
			return 2
		}
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create scaffold directory: %v\n", err)
		return 2
	}

	if err := os.WriteFile(profilePath, []byte(profileTemplate(profileID, packageName)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", profilePath, err)
		return 2
	}
	if err := os.WriteFile(testPath, []byte(profileTestTemplate(packageName)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", testPath, err)
		return 2
	}

	if !common.quiet {
		fmt.Printf("Created profile scaffold %s (id=%s)\n", targetDir, profileID)
	}
	return 0
}

func normalizeProfileID(input string) string {
	trimmed := strings.ToLower(strings.TrimSpace(input))
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	trimmed = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(trimmed, "")
	trimmed = strings.Trim(trimmed, "_")
	return trimmed
}

func normalizePackageName(profileID string) string {
	name := regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(strings.ToLower(profileID), "")
	if name == "" {
		return "custom"
	}
	if name[0] >= '0' && name[0] <= '9' {
		return "p_" + name
	}
	return name
}

func profileTemplate(profileID, packageName string) string {
	return fmt.Sprintf(`package %s

import (
	"path"
	"strings"

	"github.com/honzikec/archguard/internal/framework/contracts"
	"github.com/honzikec/archguard/internal/framework/common"
)

type Profile struct{}

func New() contracts.Profile {
	return Profile{}
}

func (Profile) ID() string {
	return %q
}

func (Profile) Detect(roots []string) contracts.Detection {
	_ = common.RootsOrDefault(roots)
	// TODO: add framework-specific detection logic.
	return contracts.Detection{}
}

func (Profile) NormalizeSubtree(subtree string) string {
	subtree = path.Clean(strings.TrimSpace(subtree))
	if subtree == "" {
		return "."
	}
	return subtree
}

func (p Profile) NormalizeFile(file string) string {
	dir := p.NormalizeSubtree(path.Dir(file))
	return path.Join(dir, path.Base(file))
}
`, packageName, profileID)
}

func profileTestTemplate(packageName string) string {
	return fmt.Sprintf(`package %s

import "testing"

func TestNormalizeIdempotent(t *testing.T) {
	p := Profile{}
	input := "src/example/$id.ts"
	if got := p.NormalizeFile(p.NormalizeFile(input)); got != p.NormalizeFile(input) {
		t.Fatalf("expected idempotent normalization, got %%q", got)
	}
}
`, packageName)
}

func defaultConfigTemplate() string {
	return `version: 1
project:
  roots:
    - "."
  include:
    - "**/*.ts"
    - "**/*.tsx"
    - "**/*.js"
    - "**/*.jsx"
    - "**/*.mjs"
    - "**/*.cjs"
  exclude:
    - "**/node_modules/**"
    - "**/dist/**"
    - "**/build/**"
    - "**/.next/**"
    - "**/coverage/**"
    - "**/.git/**"
  framework: generic
  aliases: {}
rules:
  - id: AG-NO-INFRA-IN-DOMAIN
    kind: no_import
    severity: error
    scope:
      - "src/domain/**"
    target:
      - "src/infra/**"
    message: "Domain modules must not import infrastructure modules."

  - id: AG-NO-HTTP-IN-DOMAIN
    kind: no_package
    severity: warning
    scope:
      - "src/domain/**"
    target:
      - "axios"
    message: "Domain modules should stay transport-agnostic."

  - id: AG-SERVICE-FILES
    kind: file_pattern
    severity: warning
    scope:
      - "src/services/**"
    target:
      - "^.*\\.service\\.(ts|js)$"

  - id: AG-NO-CYCLES
    kind: no_cycle
    severity: error
    scope:
      - "src/**"
`
}
