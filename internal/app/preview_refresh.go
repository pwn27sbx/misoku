package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// The preview recompiles the whole note (preview.CompileBlocks), which is
// slow on a long note. So:
//   - a key that leaves the text as it was (j, k, arrows) recompiles nothing;
//   - on a note of previewDebounceLines lines or more, an edit schedules a
//     refresh previewDebounce later instead of recompiling on the spot; only
//     the latest scheduled refresh runs, so the preview catches up once the
//     typing pauses;
//   - anything that reads what the preview compiled (its hits and blocks) or
//     shows it on purpose first runs the pending refresh: a mouse event over
//     the preview, a click action from it, focusing it, saving, quitting,
//     exporting, following a link, a theme change and a resize;
//   - switching tabs (by mouse, key or command) first runs the pending
//     refresh of the tab being left, so its preview and the live graph's
//     links are current before the other tab shows.

const (
	previewDebounceLines = 1000
	previewDebounce      = 100 * time.Millisecond
)

// PreviewRefreshMsg recompiles the preview after an edit to a long note. The
// app schedules its own; it is exported so tests and embedders can refresh
// without waiting for the timer. The zero value runs whatever is pending.
type PreviewRefreshMsg struct{ gen int }

// previewSync tracks edits the preview has not been recompiled for yet.
type previewSync struct {
	pending   bool   // an edit is waiting for its refresh
	buf       string // the tab the pending edit was made in
	gen       int    // the latest refresh scheduled; older timers are stale
	compiles  int    // editor-driven preview recompiles, read by tests
	linkScans int    // liveGraph scans of the edited note, read by tests
}

// editorEdited brings the preview up to date after the editor changed the
// text: at once on a short note, after a pause on a long one.
func (m *Model) editorEdited() tea.Cmd {
	if m.content.LineCount() < previewDebounceLines {
		m.pv.pending = false
		m.syncPreview()
		return nil
	}
	m.pv.pending, m.pv.buf = true, m.active
	m.pv.gen++
	gen := m.pv.gen
	return tea.Tick(previewDebounce, func(time.Time) tea.Msg { return PreviewRefreshMsg{gen: gen} })
}

// refreshPreview handles a PreviewRefreshMsg: a timer that a later edit
// superseded does nothing.
func (m *Model) refreshPreview(msg PreviewRefreshMsg) {
	if msg.gen == 0 || msg.gen == m.pv.gen {
		m.flushPreview()
	}
}

// flushPreview runs the pending refresh, if any, now. If the tab it was for
// is no longer showing, showing the other tab already set the preview.
func (m *Model) flushPreview() {
	if !m.pv.pending {
		return
	}
	m.pv.pending = false
	if m.active != m.pv.buf || m.activeBuffer() == nil || m.graphActive() {
		return
	}
	m.syncPreview()
}

// syncPreview recompiles the preview, and the live graph's links of the
// edited note, from the editor's text.
func (m *Model) syncPreview() {
	m.preview.SetContent(m.content.Value())
	m.pv.compiles++
	if m.bufferIndex(graphID) >= 0 {
		m.liveGraph()
	}
}
