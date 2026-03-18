package catalog

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed patterns/*.yaml
var patternFiles embed.FS

func LoadBuiltin() ([]Pattern, error) {
	entries, err := fs.ReadDir(patternFiles, "patterns")
	if err != nil {
		return nil, fmt.Errorf("failed to read built-in catalog: %w", err)
	}

	patterns := make([]Pattern, 0, len(entries))
	seenIDs := map[string]struct{}{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		b, err := patternFiles.ReadFile("patterns/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed reading catalog file %s: %w", entry.Name(), err)
		}

		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)

		var p Pattern
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("failed parsing catalog file %s: %w", entry.Name(), err)
		}
		if err := validatePattern(p); err != nil {
			return nil, fmt.Errorf("invalid catalog file %s: %w", entry.Name(), err)
		}
		if _, ok := seenIDs[p.ID]; ok {
			return nil, fmt.Errorf("duplicate catalog id: %s", p.ID)
		}
		seenIDs[p.ID] = struct{}{}
		patterns = append(patterns, p)
	}

	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].ID < patterns[j].ID
	})

	return patterns, nil
}
