package testnote_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/testnote"
)

func TestNoteHasExactlyNLines(t *testing.T) {
	for _, n := range []int{1, 2, 3, 7, 500, 5000} {
		if got := len(strings.Split(testnote.Note(n), "\n")); got != n {
			t.Errorf("Note(%d) has %d lines", n, got)
		}
	}
	if got := testnote.Note(0); got != "" {
		t.Errorf("Note(0) = %q, want empty", got)
	}
}

func TestNoteIsDeterministic(t *testing.T) {
	if testnote.Note(500) != testnote.Note(500) {
		t.Fatal("Note(500) differs between calls")
	}
}

func TestNoteMixesMarkdownBlocks(t *testing.T) {
	note := testnote.Note(500)
	for _, want := range []string{"# Chapter", "## Section", "- item", "- [ ] ", "| --- |", "```go", "> A quote", "[[note-"} {
		if !strings.Contains(note, want) {
			t.Errorf("Note(500) has no %q", want)
		}
	}
	long := false
	for _, l := range strings.Split(note, "\n") {
		if len(l) > 200 {
			long = true
		}
	}
	if !long {
		t.Error("Note(500) has no long line that wraps")
	}
}

func TestNoteCodeFencesAreBalanced(t *testing.T) {
	for _, n := range []int{5, 33, 499, 500, 5000} {
		if c := strings.Count(testnote.Note(n), "```"); c%2 != 0 {
			t.Errorf("Note(%d) has %d fences", n, c)
		}
	}
}
