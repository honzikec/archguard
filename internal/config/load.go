package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if err := validateSchemaDocument(raw); err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	applyDefaults(&cfg)

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	defaults := DefaultProjectSettings()
	if len(cfg.Project.Roots) == 0 {
		cfg.Project.Roots = defaults.Roots
	}
	if len(cfg.Project.Include) == 0 {
		cfg.Project.Include = defaults.Include
	}
	if len(cfg.Project.Exclude) == 0 {
		cfg.Project.Exclude = defaults.Exclude
	}
	if cfg.Project.Aliases == nil {
		cfg.Project.Aliases = map[string][]string{}
	}
}
