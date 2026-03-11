package config

type Config struct {
	Version int    `yaml:"version"`
	Rules   []Rule `yaml:"rules"`
}

type Rule struct {
	ID         string     `yaml:"id"`
	Kind       string     `yaml:"kind"`
	Severity   string     `yaml:"severity"`
	Rationale  string     `yaml:"rationale"`
	Conditions Conditions `yaml:"conditions"`
}

type Conditions struct {
	FromPaths         []string `yaml:"from_paths,omitempty"`
	ForbiddenPaths    []string `yaml:"forbidden_paths,omitempty"`
	ForbiddenPackages []string `yaml:"forbidden_packages,omitempty"`
	PathPatterns      []string `yaml:"path_patterns,omitempty"`
	FilenameRegex     string   `yaml:"filename_regex,omitempty"`
}
