package conformance

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	frameworkcontracts "github.com/honzikec/archguard/internal/framework/contracts"
)

var moduleIDPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

func ValidateFrameworkProfiles(
	profiles []frameworkcontracts.Profile,
	roots []string,
	subtrees []string,
	files []string,
) error {
	if len(profiles) == 0 {
		return fmt.Errorf("no framework profiles registered")
	}

	ids := make([]string, 0, len(profiles))
	seen := map[string]struct{}{}
	for _, profile := range profiles {
		id := strings.TrimSpace(profile.ID())
		if id == "" {
			return fmt.Errorf("framework profile id must be non-empty")
		}
		if !moduleIDPattern.MatchString(id) {
			return fmt.Errorf("framework profile id %q must match %s", id, moduleIDPattern.String())
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate framework profile id: %s", id)
		}
		seen[id] = struct{}{}
		ids = append(ids, id)

		d1 := profile.Detect(roots)
		d2 := profile.Detect(roots)
		if !reflect.DeepEqual(d1, d2) {
			return fmt.Errorf("framework profile %s detection must be deterministic: first=%+v second=%+v", id, d1, d2)
		}
		if d1.Matched && strings.TrimSpace(d1.Reason) == "" {
			return fmt.Errorf("framework profile %s matched=true requires non-empty detection reason", id)
		}
		if d1.Score < 0 {
			return fmt.Errorf("framework profile %s detection score must be >= 0", id)
		}

		for _, subtree := range subtrees {
			once := profile.NormalizeSubtree(subtree)
			twice := profile.NormalizeSubtree(once)
			if once != twice {
				return fmt.Errorf("framework profile %s subtree normalization not idempotent: once=%q twice=%q", id, once, twice)
			}
			if strings.TrimSpace(subtree) != "" && strings.TrimSpace(once) == "" {
				return fmt.Errorf("framework profile %s returned empty subtree for non-empty input %q", id, subtree)
			}
		}

		for _, file := range files {
			once := profile.NormalizeFile(file)
			twice := profile.NormalizeFile(once)
			if once != twice {
				return fmt.Errorf("framework profile %s file normalization not idempotent: once=%q twice=%q", id, once, twice)
			}
			if strings.TrimSpace(file) != "" && strings.TrimSpace(once) == "" {
				return fmt.Errorf("framework profile %s returned empty file for non-empty input %q", id, file)
			}
		}
	}

	sorted := append([]string{}, ids...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(ids, sorted) {
		return fmt.Errorf("framework profiles must be returned in deterministic sorted order: got=%v want=%v", ids, sorted)
	}

	return nil
}
