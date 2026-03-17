package testutil

import (
	"path/filepath"
	"testing"
)

// GetFixturePath returns the absolute path to a named fixture directory
func GetFixturePath(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "fixtures", name))
	if err != nil {
		t.Fatalf("failed to resolve fixture path: %v", err)
	}
	return abs
}
