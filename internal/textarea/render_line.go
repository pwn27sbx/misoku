package textarea

// misoku: the per-line render hook behind Model.RenderLine. A hooked line
// keeps the rows its raw text wraps to, so scrolling, cursor motion and mouse
// rows work exactly as they do without the hook.

// renderedRows returns the hook's rows for line l fitted to exactly n rows:
// rows past n are dropped and missing rows are blank. View pads each row to
// the editor's width after the prompt and line-number cell. The hook is
// handed the editor's text width; its rows must not be wider than that.
func (m Model) renderedRows(l, n int) []string {
	rows := m.RenderLine(l, m.value[l], m.width)
	out := make([]string, n)
	copy(out, rows)
	return out
}
