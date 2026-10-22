package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// MouseFilter is meant for tea.WithFilter. With tea.WithMouseAllMotion the
// terminal reports every cell the pointer crosses, and most of that motion
// changes nothing on screen yet would still cost an Update and a full View.
// MouseFilter drops it before the event loop calls either.
//
// Motion passes only while it can change something: a drag in the editor
// (text selection, panning the graph), of a divider or of a preview block;
// over the preview, where the block handles follow the mouse; or when the
// handles are still showing and the next move must hide them. Every other
// message passes unchanged.
//
// Bubble Tea calls the filter on its event loop goroutine with the current
// model, so reading it is safe; MouseFilter only reads it.
func MouseFilter(tm tea.Model, msg tea.Msg) tea.Msg {
	mm, ok := msg.(tea.MouseMsg)
	if !ok || mm.Action != tea.MouseActionMotion {
		return msg
	}
	var m *Model
	switch v := tm.(type) {
	case Model:
		m = &v
	case *Model:
		m = v
	default:
		return msg
	}
	if m.motionMatters(mm) {
		return msg
	}
	return nil
}

// motionMatters reports whether handleMouse would change anything for this
// motion. It must stay read-only: the filter runs it before Update.
func (m *Model) motionMatters(msg tea.MouseMsg) bool {
	if m.viewMode == viewModeDashboard {
		return false // the dashboard only reacts to clicks
	}
	return m.resizing != dividerNone || m.dragging || m.preview.Dragging() ||
		m.preview.Hovering() || m.layout().overPreview(msg.X, msg.Y)
}
