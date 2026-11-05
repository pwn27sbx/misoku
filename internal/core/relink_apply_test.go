package core_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

func writeRaw(t *testing.T, s *core.Store, id, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(s.Root(), filepath.FromSlash(id)), []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func readRaw(t *testing.T, s *core.Store, id string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(s.Root(), filepath.FromSlash(id)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// ApplyRelinks writes a note only when its file still holds the text the
// rewrite was planned from: one changed meanwhile is skipped, not clobbered.
func TestApplyRelinksSkipsNotesChangedSincePlanned(t *testing.T) {
	s := newTestStore(t)
	writeRaw(t, s, "a.md", "see [[old]]")
	writeRaw(t, s, "b.md", "edited elsewhere [[old]]")
	writeRaw(t, s, "c.md", "line [[old]]\r\nnext\r\n")
	rs := []core.Relink{
		{ID: "a.md", From: "see [[old]]", To: "see [[new]]", Links: 1},
		{ID: "b.md", From: "also [[old]]", To: "also [[new]]", Links: 1},
		{ID: "c.md", From: "line [[old]]\nnext\n", To: "line [[new]]\nnext\n", Links: 1},
		{ID: "gone.md", From: "x [[old]]", To: "x [[new]]", Links: 1},
	}
	done, skipped, err := s.ApplyRelinks(rs)
	if err != nil {
		t.Fatal(err)
	}
	if want := []core.Relink{rs[0], rs[2]}; !reflect.DeepEqual(done, want) {
		t.Fatalf("done = %+v, want %+v", done, want)
	}
	if want := []string{"b.md", "gone.md"}; !reflect.DeepEqual(skipped, want) {
		t.Fatalf("skipped = %v, want %v", skipped, want)
	}
	if got := readRaw(t, s, "a.md"); got != "see [[new]]" {
		t.Fatalf("a.md = %q", got)
	}
	if got := readRaw(t, s, "b.md"); got != "edited elsewhere [[old]]" {
		t.Fatalf("a note changed meanwhile must be left as it is, got %q", got)
	}
	// A CRLF note is compared as the store reads it and keeps its endings.
	if got := readRaw(t, s, "c.md"); got != "line [[new]]\r\nnext\r\n" {
		t.Fatalf("c.md = %q", got)
	}
	if _, err := os.Stat(filepath.Join(s.Root(), "gone.md")); !os.IsNotExist(err) {
		t.Fatalf("a missing note must not be created: %v", err)
	}
}
