package brief

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Brief, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("brief not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read brief: %w", err)
	}

	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse brief: %w", err)
	}
	if err := validateSchemaDocument(raw); err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var spec Brief
	if err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to parse brief: %w", err)
	}
	if err := validateSpec(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func validateSpec(spec *Brief) error {
	if spec == nil {
		return fmt.Errorf("brief is nil")
	}
	if spec.Version != 1 {
		return fmt.Errorf("unsupported brief version %d, expected 1", spec.Version)
	}
	seenLayer := map[string]struct{}{}
	for i, layer := range spec.Layers {
		id := strings.TrimSpace(layer.ID)
		if id == "" {
			return fmt.Errorf("layers[%d].id is required", i)
		}
		if _, ok := seenLayer[id]; ok {
			return fmt.Errorf("duplicate layer id: %s", id)
		}
		seenLayer[id] = struct{}{}
	}

	seenPolicy := map[string]struct{}{}
	for i, p := range spec.Policies {
		if id := strings.TrimSpace(p.ID); id != "" {
			if _, ok := seenPolicy[id]; ok {
				return fmt.Errorf("duplicate policy id: %s", id)
			}
			seenPolicy[id] = struct{}{}
		}
		if strings.TrimSpace(p.Type) == "" {
			return fmt.Errorf("policies[%d].type is required", i)
		}
	}

	return nil
}
