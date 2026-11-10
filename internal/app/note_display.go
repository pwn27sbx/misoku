package app

import (
	"sort"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/preview"
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
	m.unfoldCursor()
	d := m.preview.Display()
	m.content.ShowRendered(content.Rendered{
		Generation: d.Generation,
		PageID:     d.PageID,
		Failed:     d.Failed,
		Current:    !m.pv.pending,
		Frozen:     m.preview.Dragging(),
		Footer:     d.Footer,
		FooterRow:  d.FooterRow,
		Blocks: func() []content.RenderedBlock {
			out := make([]content.RenderedBlock, len(d.Blocks))
			for i, b := range d.Blocks {
				out[i] = content.RenderedBlock{Line: b.Line, End: b.End, Row: b.Row, Rows: b.Rows, Folded: b.Folded}
			}
			return out
		},
	}, m.focus != focusEditor) // no split here: only the focus dims the note
}

// unfoldCursor opens the folded sections whose hidden body holds the cursor,
// so a search, a jump or a goto into one lands on a line that shows. j and k
// step over hidden lines and never get here.
func (m *Model) unfoldCursor() {
	if m.pv.pending {
		return // the compile's folds belong to an older text
	}
	line, _ := m.content.CursorPosition()
	for range 8 { // a section folded inside another opens on the next round
		d := m.preview.Display()
		if d.Failed || d.PageID != m.activeBuffer().id {
			return
		}
		key, ok := foldOver(&m.preview, d.Blocks, line-1)
		if !ok {
			return
		}
		m.preview.ToggleFold(key)
	}
}

// foldOver returns the fold key of the folded block whose hidden body holds
// line: the fold hit on one of the block's rows.
func foldOver(p *preview.Model, blocks []preview.DisplayBlock, line int) (string, bool) {
	i := sort.Search(len(blocks), func(i int) bool { return blocks[i].End > line })
	if i == len(blocks) || !blocks[i].Folded || line <= blocks[i].Line {
		return "", false
	}
	b := blocks[i]
	for r := b.Row; r < b.Row+len(b.Rows); r++ {
		if h, ok := p.HitAtRow(r, 0); ok && h.Kind == "fold" {
			return h.Arg, true
		}
	}
	return "", false
}
