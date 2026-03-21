package brief

type Brief struct {
	Version  int            `yaml:"version"`
	Project  Project        `yaml:"project,omitempty"`
	Layers   []Layer        `yaml:"layers,omitempty"`
	Policies []PolicyIntent `yaml:"policies"`
}

type Project struct {
	Roots     []string            `yaml:"roots,omitempty"`
	Include   []string            `yaml:"include,omitempty"`
	Exclude   []string            `yaml:"exclude,omitempty"`
	Framework string              `yaml:"framework,omitempty"`
	Language  string              `yaml:"language,omitempty"`
	Tsconfig  string              `yaml:"tsconfig,omitempty"`
	Aliases   map[string][]string `yaml:"aliases,omitempty"`
}

type Layer struct {
	ID    string   `yaml:"id"`
	Paths []string `yaml:"paths"`
}

type PolicyIntent struct {
	ID               string   `yaml:"id,omitempty"`
	Type             string   `yaml:"type"`
	Severity         string   `yaml:"severity,omitempty"`
	Message          string   `yaml:"message,omitempty"`
	Except           []string `yaml:"except,omitempty"`
	From             []string `yaml:"from,omitempty"`
	To               []string `yaml:"to,omitempty"`
	Scope            []string `yaml:"scope,omitempty"`
	Packages         []string `yaml:"packages,omitempty"`
	Pattern          string   `yaml:"pattern,omitempty"`
	Services         []string `yaml:"services,omitempty"`
	AllowIn          []string `yaml:"allow_in,omitempty"`
	ServiceNameRegex string   `yaml:"service_name_regex,omitempty"`
}
