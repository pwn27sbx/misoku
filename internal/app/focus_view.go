package app

import "github.com/charmbracelet/lipgloss"

// dimPanes greys out the panes that are not in use (S3). View calls it on its
// own copy of the model, so the dimmed state never outlives a frame.
//
// The editor and the preview side by side are one thing: the note and how it is
// coming out. Both keep their colours while one of them has the focus, so
// watching the preview while writing costs no brightness. The explorer taking
// the focus greys them as it always did, and a graph replaces the editor, so
// there is no split left to keep bright.
func (m *Model) dimPanes(l Layout) {
	split := m.splitHasFocus(l)
	m.sidebar.SetDimmed(m.focus != focusSidebar)
	m.content.SetDimmed(!split && m.focus != focusEditor)
	m.graph.SetDimmed(m.focus != focusEditor)
	m.globalGraph.SetDimmed(m.focus != focusEditor)
	m.preview.SetDimmed(!split && m.focus != focusPreview)
}

// splitHasFocus reports whether the note's split is what the keys go to: the
// editor and the preview both on screen with one of them focused. A graph takes
// the editor's place and the full-width views leave no preview, so neither counts.
func (m *Model) splitHasFocus(l Layout) bool {
	if m.focus != focusEditor && m.focus != focusPreview {
		return false
	}
	return l.EditorW > 0 && l.PreviewW > 0 && !m.graphActive()
}

// dividerFocused reports whether divider d touches the focused pane. With
// no editor column the explorer's divider borders the preview.
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
