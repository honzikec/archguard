package cli

import (
	"reflect"
	"testing"
)

func TestParseSelection(t *testing.T) {
	got, err := parseSelection("1,3-4,2,4", 5)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	want := []int{0, 1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parsed selection, want=%v got=%v", want, got)
	}
}

func TestParseSelectionRejectsOutOfBounds(t *testing.T) {
	if _, err := parseSelection("1,7", 5); err == nil {
		t.Fatal("expected out-of-bounds parse error")
	}
}
