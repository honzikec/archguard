package catalog_test

import (
	"testing"

	"github.com/honzikec/archguard/internal/catalog"
)

func TestLoadBuiltin(t *testing.T) {
	patterns, err := catalog.LoadBuiltin()
	if err != nil {
		t.Fatalf("expected built-in catalog to load: %v", err)
	}
	if len(patterns) < 5 {
		t.Fatalf("expected at least 5 patterns, got %d", len(patterns))
	}
	for _, p := range patterns {
		if p.ID == "" || p.Name == "" || len(p.Sources) == 0 {
			t.Fatalf("invalid loaded pattern: %+v", p)
		}
	}
}
