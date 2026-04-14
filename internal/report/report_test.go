package report

import (
	"os"
	"testing"
)

func TestPrintJSONReturnsWriteError(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	_ = w.Close()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	if err := PrintJSON(nil, Summary{}); err == nil {
		t.Fatal("expected write error from closed stdout")
	}
}
