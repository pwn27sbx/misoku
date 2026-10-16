package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A file, a pipe and the null device are not a terminal: misoku run from
// a script never shows the welcome.
func TestScriptsAreNotATerminal(t *testing.T) {
	t.Parallel()
	f, err := os.Create(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	for name, file := range map[string]*os.File{"a file": f, "a pipe": r, "the null device": null} {
		if isTerminal(file) {
			t.Errorf("%s counts as a terminal", name)
		}
	}
}
