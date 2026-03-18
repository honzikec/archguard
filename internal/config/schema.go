package config

type Config struct {
	Version int             `yaml:"version"`
	Project ProjectSettings `yaml:"project"`
	Rules   []Rule          `yaml:"rules"`
}

type ProjectSettings struct {
	Roots    []string            `yaml:"roots,omitempty"`
	Include  []string            `yaml:"include,omitempty"`
	Exclude  []string            `yaml:"exclude,omitempty"`
	Tsconfig string              `yaml:"tsconfig,omitempty"`
	Aliases  map[string][]string `yaml:"aliases,omitempty"`
}

type Rule struct {
	ID       string            `yaml:"id"`
	Kind     string            `yaml:"kind"`
	Severity string            `yaml:"severity"`
	Scope    []string          `yaml:"scope"`
	Target   []string          `yaml:"target,omitempty"`
	Except   []string          `yaml:"except,omitempty"`
	Template string            `yaml:"template,omitempty"`
	Params   map[string]string `yaml:"params,omitempty"`
	Message  string            `yaml:"message,omitempty"`
}

const (
	KindNoImport    = "no_import"
	KindNoPackage   = "no_package"
	KindFilePattern = "file_pattern"
	KindNoCycle     = "no_cycle"
	KindPattern     = "pattern"
)

const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

func DefaultProjectSettings() ProjectSettings {
	return ProjectSettings{
		Roots:   []string{"."},
		Include: []string{"**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx", "**/*.mjs", "**/*.cjs"},
		Exclude: []string{"**/node_modules/**", "**/dist/**", "**/build/**", "**/.next/**", "**/coverage/**", "**/.git/**"},
	}
}
