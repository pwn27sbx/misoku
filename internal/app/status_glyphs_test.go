package app_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// explorerRow returns the stripped row where the explorer lists name.md,
// cut to the explorer's side of the screen; the tab bar on the first row is
// skipped.
func explorerRow(t *testing.T, view, name string) string {
	t.Helper()
	lines := strings.Split(stripped(view), "\n")
	for _, l := range lines[1:] {
		r := []rune(l)
		left := string(r[:min(len(r), 36)])
		if strings.Contains(left, " "+name+".md") {
			return left
		}
	}
	t.Fatalf("explorer does not list %q:\n%s", name, stripped(view))
	return ""
}

func TestSavingAStatusShowsItsGlyphInTheExplorer(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "plan", "hello"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "other", "---\nstatus: on hold\n---\nx"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1") // the newest recent note: other
	h.key(tea.KeyEsc)
	if row := explorerRow(t, h.view(), "other"); !strings.Contains(row, "◐") {
		t.Fatalf("a saved status should show from the start: %q\n%s", row, stripped(h.view()))
	}
	if row := explorerRow(t, h.view(), "plan"); strings.ContainsAny(row, "●◐✓✗") {
		t.Fatalf("a note without status should keep its icon: %q", row)
	}

	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("plan")
	h.key(tea.KeyEnter)
	h.keys("i---")
	h.key(tea.KeyEnter)
	h.keys("status: completed")
	h.key(tea.KeyEnter)
	h.keys("---")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})

	p, err := store.Get("plan.md")
	if err != nil || !strings.HasPrefix(p.Content, "---\nstatus: completed\n---\n") {
		t.Fatalf("the status should be saved, got %q (%v)", p.Content, err)
	}
	if row := explorerRow(t, h.view(), "plan"); !strings.Contains(row, "✓") {
		t.Fatalf("after saving, the explorer should show ✓ for plan: %q", row)
	}
}
