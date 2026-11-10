package content

// SetLive turns the pretty view's editor chrome on or off. On, the editor
// hides its line numbers and the end-of-buffer rows; off, it draws exactly
// as it did before.
func (m *Model) SetLive(on bool) {
	m.live = on
	m.textarea.ShowLineNumbers = !on
	m.textarea.EndOfBufferCharacter = '~'
	if on {
		m.textarea.EndOfBufferCharacter = ' '
	}
	// The text width depends on the line-number column: recompute it.
	m.fitText()
}

// Live reports whether the pretty view's editor chrome is on.
func (m Model) Live() bool { return m.live }
