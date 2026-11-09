package app

import "github.com/pwn27sbx/misoku/internal/preview"

// Clicking a box in the preview

// previewCheckbox toggles the to-do line a click on its box names, as one
// change u undoes, and refreshes the preview.
func (m *Model) previewCheckbox(h preview.Hit) {
	// While a graph is browsed the preview shows another note: its boxes are
	// not this note's lines.
	if m.preview.PageID() != m.currentNoteID() {
		return
	}
	if !m.content.ToggleLineAt(h.Line) {
		return
	}
	m.tableEdited() // the preview and the modified mark follow the edit
}
