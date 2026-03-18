package catalog

type Pattern struct {
	ID           string       `yaml:"id"`
	Name         string       `yaml:"name"`
	Category     string       `yaml:"category"`
	Description  string       `yaml:"description"`
	Sources      []Source     `yaml:"sources"`
	Detection    Detection    `yaml:"detection"`
	RuleTemplate RuleTemplate `yaml:"rule_template"`
}

type Source struct {
	Title   string `yaml:"title"`
	URL     string `yaml:"url"`
	License string `yaml:"license"`
}

type Detection struct {
	RequiredFacts []string  `yaml:"required_facts"`
	Heuristic     Heuristic `yaml:"heuristic"`
}

type Heuristic struct {
	Type   string         `yaml:"type"`
	Params map[string]any `yaml:"params"`
}

type RuleTemplate struct {
	Kind     string      `yaml:"kind"`
	Template string      `yaml:"template"`
	Defaults RuleDefault `yaml:"defaults"`
}

type RuleDefault struct {
	Severity string            `yaml:"severity"`
	Params   map[string]string `yaml:"params"`
}
