package content_test

import "testing"

// S9: restoring an older version swaps the whole note in one change of its
// own, which u undoes, even typed right after an INSERT session.
func TestRestoreTextIsOneChangeThatUUndoes(t *testing.T) {
	c := newNote("today\nsecond line\nthird line")
	c = esc(keys(c, "A!"))
	if !c.RestoreText("older") {
		t.Fatal("RestoreText reported no change")
	}
	if c.Value() != "older" {
		t.Fatalf("after restore the note is %q", c.Value())
	}
	c, _ = undo(c)
	if c.Value() != "today!\nsecond line\nthird line" {
		t.Fatalf("u after a restore = %q, want the text before it", c.Value())
	}
	c, _ = undo(c)
	if c.Value() != "today\nsecond line\nthird line" {
		t.Fatalf("a second u = %q, want the text before the typing", c.Value())
	}
}

// S9: restoring the text the note already holds changes nothing.
func TestRestoreTextOfTheSameTextIsNoChange(t *testing.T) {
	c := newNote("same")
	if c.RestoreText("same") {
		t.Fatal("restoring the same text must report no change")
	}
	if _, msg := undo(c); msg != "Already at oldest change" {
		t.Fatalf("u after a no-op restore: %q", msg)
	}
}

// S9: typed while an INSERT session is still open, the restore does not
// swallow the typing: u goes back to the text as typed, not before it.
func TestRestoreTextInsideInsertKeepsTheTypingAsItsOwnChange(t *testing.T) {
	c := newNote("today")
	c = keys(c, "A!")
	c.RestoreText("older")
	c = esc(c)
	c, _ = undo(c)
	if c.Value() != "today!" {
		t.Fatalf("u after a restore inside INSERT = %q, want %q", c.Value(), "today!")
	}
}
