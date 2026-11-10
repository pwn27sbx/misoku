package app

import (
	"github.com/pwn27sbx/misoku/internal/content"
)

// syncNoteDisplay runs after every event, because the cursor and the compile
// change in many places: in the reading view it hands the editor the latest
// compile, which redraws only when something it depends on changed. Anywhere
// else the editor draws its Markdown.
func (m *Model) syncNoteDisplay() {
	if m.noteView != viewReading || m.graphActive() || m.activeBuffer() == nil {
		m.content.ClearRendered()
		return
	}
	d := m.preview.Display()
	m.content.ShowRendered(content.Rendered{
		Generation: d.Generation,
		PageID:     d.PageID,
		Failed:     d.Failed,
		Current:    !m.pv.pending,
		Footer:     d.Footer,
		Blocks: func() []content.RenderedBlock {
			out := make([]content.RenderedBlock, len(d.Blocks))
			for i, b := range d.Blocks {
				out[i] = content.RenderedBlock{Line: b.Line, End: b.End, Rows: b.Rows, Folded: b.Folded}
			}
			return out
		},
	}, m.focus != focusEditor) // no split here: only the focus dims the note
}
