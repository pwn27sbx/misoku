package app_test

// Space e toggles the explorer focus like LazyVim, and H / L switch tabs
// without taking the preview's calendar keys.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// explorerFocused reports whether the mode badge says the explorer has the
// focus.
func explorerFocused(v string) bool { return strings.Contains(v, "EXPLORER") }

// spaceE presses the default focus_explorer shortcut, Space then e.
func spaceE(h *harness) {
	h.t.Helper()
	h.keys(" e")
}

// spaceE toggles: an open explorer with the focus closes and the editor takes
// it back; an open explorer without the focus gets it.
func TestSpaceETogglesTheExplorerFocus(t *testing.T) {
	h := openNote(t)
	if v := h.view(); !explorerShown(v) || explorerFocused(v) {
		t.Fatalf("setup: the explorer should start visible without the focus:\n%s", v)
	}
	spaceE(h)
	if v := h.view(); !explorerShown(v) || !explorerFocused(v) {
		t.Fatalf("space e should focus the visible explorer:\n%s", v)
	}
	spaceE(h)
	v := h.view()
	if explorerShown(v) || explorerFocused(v) || !strings.Contains(v, "NORMAL") {
		t.Fatalf("space e on a focused explorer should close it and focus the editor:\n%s", v)
	}
}

// A hidden explorer is shown and focused, as before.
func TestSpaceEShowsAHiddenExplorer(t *testing.T) {
	h := openNote(t)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlB})
	if explorerShown(h.view()) {
		t.Fatalf("setup: ctrl+b should hide the explorer:\n%s", h.view())
	}
	spaceE(h)
	if v := h.view(); !explorerShown(v) || !explorerFocused(v) {
		t.Fatalf("space e should show the explorer with the focus:\n%s", v)
	}
}

// The narrow explorer is an overlay: Space e opens it and closes it again.
func TestSpaceETogglesTheNarrowExplorerOverlay(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	spaceE(h)
	if v := h.screen(); !explorerShown(v) || !explorerFocused(v) {
		t.Fatalf("space e should open the overlay with the focus:\n%s", v)
	}
	spaceE(h)
	v := h.screen()
	if explorerShown(v) || explorerFocused(v) || !strings.Contains(v, "NORMAL") {
		t.Fatalf("space e again should close the overlay and focus the editor:\n%s", v)
	}
}

// On the home screen it keeps opening the explorer, as it did.
func TestSpaceEFromHomeFocusesTheExplorer(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	spaceE(h)
	if v := h.view(); !explorerShown(v) || !explorerFocused(v) {
		t.Fatalf("space e on home should open and focus the explorer:\n%s", v)
	}
}

// With no note open there is nothing to go back to: the explorer stays.
func TestSpaceEKeepsTheExplorerWithNoNoteOpen(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	spaceE(h) // home: open and focus the explorer
	spaceE(h)
	if v := h.view(); !explorerShown(v) || !explorerFocused(v) {
		t.Fatalf("space e with no note open should keep the explorer open and focused:\n%s", v)
	}
}

// H and L move between tabs from the editor's NORMAL mode, wrapping like the
// bracket keys.
func TestHLKeysSwitchTabs(t *testing.T) {
	h := openTwo(t, "alpha body", "beta body")
	if got := h.editor(); got != "beta body" {
		t.Fatalf("setup: beta should be the active tab, got %q", got)
	}
	h.keys("L")
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("L should show the next tab (wrapping to alpha), got %q", got)
	}
	h.keys("H")
	if got := h.editor(); got != "beta body" {
		t.Fatalf("H should show the previous tab (wrapping to beta), got %q", got)
	}
	h.keys("]")
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("] should still show the next tab, got %q", got)
	}
	h.keys("[")
	if got := h.editor(); got != "beta body" {
		t.Fatalf("[ should still show the previous tab, got %q", got)
	}
	h.keys(" ")
	h.key(tea.KeyTab)
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("space tab should still show the next tab, got %q", got)
	}
}

// In INSERT mode the letters are text.
func TestHLTypeWhileInserting(t *testing.T) {
	h := openNotes(t, "alpha")
	h.keys("A")
	h.keys("LH")
	if got := h.editor(); got != "alpha bodyLH" {
		t.Fatalf("INSERT should type L and H, got %q", got)
	}
}

// The preview keeps its calendar keys: with the preview focused H and L
// shift the calendars and no tab changes.
func TestPreviewKeepsHLForItsCalendars(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "alpha", "plain body"); err != nil {
		t.Fatal(err)
	}
	cal := "```calendar\nmonth: 2026-10\n2026-10-15: Launch\n```"
	if _, err := store.SaveAs("", "zulu", cal); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("e") // explorer, on alpha
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.keys(" e")
	h.keys("j") // zulu, in its own tab
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	if got := h.editor(); got != cal {
		t.Fatalf("setup: the calendar note should be active, got %q", got)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlL}) // focus the preview
	if v := h.view(); !strings.Contains(v, "PREVIEW") {
		t.Fatalf("setup: ctrl+l should focus the preview:\n%s", v)
	}
	h.keys("H")
	if got := h.view(); !strings.Contains(got, "September 2026") {
		t.Fatalf("H with the preview focused should shift the calendar back:\n%s", got)
	}
	if got := h.editor(); got != cal {
		t.Fatalf("H in the preview must not switch tabs, got the note %q", got)
	}
	h.keys("L")
	if got := h.view(); !strings.Contains(got, "October 2026") {
		t.Fatalf("L with the preview focused should shift the calendar forward:\n%s", got)
	}
	if got := h.editor(); got != cal {
		t.Fatalf("L in the preview must not switch tabs, got the note %q", got)
	}
}
