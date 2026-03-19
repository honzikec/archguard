package cli

import (
	"flag"
	"fmt"
	"os"
)

func runInit(args []string) int {
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
