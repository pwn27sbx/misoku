package app_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// S4: while the cursor is in a table in INSERT the statusline says which
// keys work there, and how to add a column; it disappears outside the table
// and in NORMAL, where typing is not implied.
func TestTableHintShowsInInsertInsideATable(t *testing.T) {
	store := newTestStore(t)
	note := "| a   | b   |\n| --- | --- |\n| 1   | 2   |\n\noutside"
	if _, err := store.SaveAs("", "note", note); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1") // open the only note
	if s := statusOf(h.view()); strings.Contains(s, "Tab next cell") {
		t.Fatalf("NORMAL on a table must not show the hint:\n%s", s)
	}

	h.keys("jjlli") // INSERT in the table's body cell
	if s := statusOf(h.view()); !strings.Contains(s, "Tab next cell") {
		t.Fatalf("INSERT in a table should show the hint:\n%s", s)
	}

	h.key(tea.KeyEsc)
	if s := statusOf(h.view()); strings.Contains(s, "Tab next cell") {
		t.Fatalf("leaving INSERT should hide the hint:\n%s", s)
	}

	h.keys("Gi") // INSERT on the blank line below the table
	if s := statusOf(h.view()); strings.Contains(s, "Tab next cell") {
		t.Fatalf("INSERT outside a table must not show the hint:\n%s", s)
	}
}
