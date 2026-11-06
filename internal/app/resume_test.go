package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
)

// S5: the note misoku tutor opens (the resumed lesson) is already in the
// editor when the screen first shows, like a pick in the finder.
func TestWithOpenNoteOpensTheNoteAtStartup(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Create("02 Casillas", ""); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithOpenNote("02 Casillas.md"))
	if tabs := app.OpenTabs(h.m); len(tabs) != 1 || tabs[0] != "02 Casillas.md" {
		t.Fatalf("open tabs = %v, want the lesson only", tabs)
	}
	if v := h.view(); strings.Contains(v, "New Note") {
		t.Fatalf("the home screen shows instead of the note:\n%s", v)
	}
}

// A note that is not in the vault opens nothing and says nothing: startup
// never fails over it.
func TestWithOpenNoteIgnoresAMissingNote(t *testing.T) {
	store := newTestStore(t)
	h := newHarness(t, store, app.WithOpenNote("Nope.md"))
	if tabs := app.OpenTabs(h.m); len(tabs) != 0 {
		t.Fatalf("open tabs = %v, want none", tabs)
	}
	if v := h.view(); !strings.Contains(v, "New Note") {
		t.Fatalf("the home screen should show:\n%s", v)
	}
}
