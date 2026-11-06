package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Ctrl+← and Ctrl+→ move the focus between panes like Ctrl+H and Ctrl+L,
// for whoever does not use the h j k l keys: editor to preview and back,
// then on to the explorer, stopping at the first pane.
func TestFocusMovesWithCtrlArrows(t *testing.T) {
	h := focusHarness(t)
	h.key(tea.KeyCtrlRight)
	assertDividers(t, h.colourView(), false, true)
	h.key(tea.KeyCtrlLeft)
	assertDividers(t, h.colourView(), true, true)
	h.key(tea.KeyCtrlLeft)
	assertDividers(t, h.colourView(), true, false)
	h.key(tea.KeyCtrlLeft) // already the first pane: it stays
	assertDividers(t, h.colourView(), true, false)
	h.key(tea.KeyCtrlRight)
	assertDividers(t, h.colourView(), true, true)
}

// While typing, Ctrl+→ is the editor's, as Ctrl+L is: the focus stays put.
func TestCtrlArrowsKeepTheFocusInInsertMode(t *testing.T) {
	h := focusHarness(t)
	h.keys("i")
	h.key(tea.KeyCtrlRight)
	assertDividers(t, h.colourView(), true, true)
	h.key(tea.KeyCtrlLeft)
	assertDividers(t, h.colourView(), true, true)
}
