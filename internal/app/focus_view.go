package app

import "github.com/charmbracelet/lipgloss"

// dimPanes greys out every pane but the focused one (S3). View calls it on
// its own copy of the model, so the dimmed state never outlives a frame.
func (m *Model) dimPanes() {
	m.sidebar.SetDimmed(m.focus != focusSidebar)
	m.content.SetDimmed(m.focus != focusEditor)
	m.graph.SetDimmed(m.focus != focusEditor)
	m.globalGraph.SetDimmed(m.focus != focusEditor)
	m.preview.SetDimmed(m.focus != focusPreview)
}

// dividerFocused reports whether divider d touches the focused pane. In
// reading mode the explorer's divider borders the preview.
func (m *Model) dividerFocused(d divider, l Layout) bool {
	switch d {
	case dividerSidebar:
		right := focusEditor
		if l.EditorW == 0 {
			right = focusPreview
		}
		return m.focus == focusSidebar || m.focus == right
	case dividerPreview:
		return m.focus == focusEditor || m.focus == focusPreview
	}
	return false
}

// dividerColour is divider d's colour: lit while it is dragged or while it
// touches the focused pane.
func (m *Model) dividerColour(d divider, l Layout) lipgloss.Color {
	switch {
	case m.resizing == d:
		return m.theme.Blue
	case m.dividerFocused(d, l):
		return m.theme.BorderFocus
	}
	return m.theme.Line
}
