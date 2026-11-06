package content

// RestoreText puts text in place of the whole note, as when an older version
// of it comes back from git (":history"). It is always one change of its own,
// even typed inside an INSERT session, so a single u brings back what the
// note held just before. The cursor stays on its line, clamped to the new
// text. It reports false, and records nothing, when the note already holds
// text.
func (m *Model) RestoreText(text string) bool {
	if text == m.textarea.Value() {
		return false
	}
	m.record()
	row, col := m.textarea.RowCol()
	m.textarea.SetValue(text)
	m.textarea.SetRowCol(row, col)
	m.textarea.EnsureVisible()
	m.record()
	return true
}
