package conformance

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	languagecontracts "github.com/honzikec/archguard/internal/language/contracts"
)

func ValidateLanguageAdapters(
	adapters []languagecontracts.Adapter,
	roots []string,
	sampleFiles []string,
) error {
	if len(adapters) == 0 {
		return fmt.Errorf("no language adapters registered")
	}
	if len(sampleFiles) == 0 {
		return fmt.Errorf("no sample files provided for language adapter conformance")
	}

	ids := make([]string, 0, len(adapters))
	seen := map[string]struct{}{}
	for _, adapter := range adapters {
		id := strings.TrimSpace(adapter.ID())
		if id == "" {
			return fmt.Errorf("language adapter id must be non-empty")
		}
		if !moduleIDPattern.MatchString(id) {
			return fmt.Errorf("language adapter id %q must match %s", id, moduleIDPattern.String())
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate language adapter id: %s", id)
		}
		seen[id] = struct{}{}
		ids = append(ids, id)

		d1 := adapter.Detect(roots)
		d2 := adapter.Detect(roots)
		if !reflect.DeepEqual(d1, d2) {
			return fmt.Errorf("language adapter %s detection must be deterministic: first=%+v second=%+v", id, d1, d2)
		}
		if d1.Matched && strings.TrimSpace(d1.Reason) == "" {
			return fmt.Errorf("language adapter %s matched=true requires non-empty detection reason", id)
		}

		supportedCount := 0
		for _, sampleFile := range sampleFiles {
			s1 := adapter.SupportsFile(sampleFile)
			s2 := adapter.SupportsFile(sampleFile)
			if s1 != s2 {
				return fmt.Errorf("language adapter %s SupportsFile must be deterministic for %s", id, sampleFile)
			}
			if !s1 {
				continue
			}
			supportedCount++

			firstRefs, firstErr := adapter.ParseFile(sampleFile)
			secondRefs, secondErr := adapter.ParseFile(sampleFile)
			if (firstErr == nil) != (secondErr == nil) {
				return fmt.Errorf("language adapter %s ParseFile error stability violated for %s", id, sampleFile)
			}
			if firstErr != nil && secondErr != nil && firstErr.Error() != secondErr.Error() {
				return fmt.Errorf("language adapter %s ParseFile error text changed for %s: %q vs %q", id, sampleFile, firstErr.Error(), secondErr.Error())
			}
			if firstErr == nil && !reflect.DeepEqual(firstRefs, secondRefs) {
				return fmt.Errorf("language adapter %s ParseFile must be deterministic for %s", id, sampleFile)
			}
		}
		if supportedCount == 0 {
			return fmt.Errorf("language adapter %s supports no shared conformance sample file; add a sample in language conformance test", id)
		}

		missing := filepath.Join(filepath.Dir(sampleFiles[0]), "__missing__", id+".missing")
		_, firstErr := adapter.ParseFile(missing)
		_, secondErr := adapter.ParseFile(missing)
		if (firstErr == nil) || (secondErr == nil) {
			return fmt.Errorf("language adapter %s ParseFile on missing file must fail", id)
		}
		if firstErr.Error() != secondErr.Error() {
			return fmt.Errorf("language adapter %s ParseFile missing-file error must be deterministic: %q vs %q", id, firstErr.Error(), secondErr.Error())
		}
	}

	sorted := append([]string{}, ids...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(ids, sorted) {
		return fmt.Errorf("language adapters must be returned in deterministic sorted order: got=%v want=%v", ids, sorted)
	}

	return nil
}
