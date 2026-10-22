package textarea

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/testnote"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/rivo/uniseg"
)

// refView is View as it was before it painted only the visible rows: it
// paints every row of the note and lets the viewport cut out the visible
// ones. It is kept verbatim as the reference the new View must match.
func refView(m Model) string {
	empty := len(m.value) == 0 || (len(m.value) == 1 && len(m.value[0]) == 0)
	if empty && m.row == 0 && m.col == 0 && m.Placeholder != "" {
		return m.placeholderView()
	}
	m.Cursor.TextStyle = m.style.computedCursorLine()

	var (
		s                strings.Builder
		style            lipgloss.Style
		newLines         int
		widestLineNumber int
		lineInfo         = m.LineInfo()
	)

	displayLine := 0
	for l, line := range m.value {
		wrappedLines := m.memoizedWrap(line, m.width)

		if m.row == l {
			style = m.style.computedCursorLine()
		} else {
			style = m.style.computedText()
		}

		segStart := 0 // misoku: rune offset of this wrapped segment in the line
		for wl, wrappedLine := range wrappedLines {
			off := segStart
			segStart += len(wrappedLine)
			prompt := m.getPromptString(displayLine)
			prompt = m.style.computedPrompt().Render(prompt)
			s.WriteString(style.Render(prompt))
			displayLine++

			var ln string
			if m.ShowLineNumbers { //nolint:nestif
				if wl == 0 {
					if m.row == l {
						ln = style.Render(m.style.computedCursorLineNumber().Render(m.formatLineNumber(l + 1)))
						s.WriteString(ln)
					} else {
						ln = style.Render(m.style.computedLineNumber().Render(m.formatLineNumber(l + 1)))
						s.WriteString(ln)
					}
				} else {
					if m.row == l {
						ln = style.Render(m.style.computedCursorLineNumber().Render(m.formatLineNumber(" ")))
						s.WriteString(ln)
					} else {
						ln = style.Render(m.style.computedLineNumber().Render(m.formatLineNumber(" ")))
						s.WriteString(ln)
					}
				}
			}

			// Note the widest line number for padding purposes later.
			lnw := lipgloss.Width(ln)
			if lnw > widestLineNumber {
				widestLineNumber = lnw
			}

			strwidth := uniseg.StringWidth(string(wrappedLine))
			padding := m.width - strwidth
			// If the trailing space causes the line to be wider than the
			// width, we should not draw it to the screen since it will result
			// in an extra space at the end of the line which can look off when
			// the cursor line is showing.
			if strwidth > m.width {
				// The character causing the line to be wider than the width is
				// guaranteed to be a space since any other character would
				// have been wrapped.
				wrappedLine = []rune(strings.TrimSuffix(string(wrappedLine), " "))
				padding -= m.width - strwidth
			}
			if m.row == l && lineInfo.RowOffset == wl {
				// misoku: the cursor may sit just past a segment whose
				// trailing space was trimmed above (wide CJK runes); clamp.
				co := min(lineInfo.ColumnOffset, len(wrappedLine))
				s.WriteString(m.paint(style, l, off, wrappedLine[:co]))
				if co >= len(wrappedLine) || (m.col >= len(line) && lineInfo.CharOffset >= m.width) {
					m.Cursor.SetChar(" ")
					s.WriteString(m.Cursor.View())
				} else {
					m.Cursor.SetChar(string(wrappedLine[lineInfo.ColumnOffset]))
					s.WriteString(style.Render(m.Cursor.View()))
					s.WriteString(m.paint(style, l, off+lineInfo.ColumnOffset+1, wrappedLine[lineInfo.ColumnOffset+1:]))
				}
			} else {
				s.WriteString(m.paint(style, l, off, wrappedLine))
			}
			s.WriteString(style.Render(strings.Repeat(" ", max(0, padding))))
			s.WriteRune('\n')
			newLines++
		}
	}

	// Always show at least `m.Height` lines at all times.
	// To do this we can simply pad out a few extra new lines in the view.
	for i := 0; i < m.height; i++ {
		prompt := m.getPromptString(displayLine)
		prompt = m.style.computedPrompt().Render(prompt)
		s.WriteString(prompt)
		displayLine++

		// Write end of buffer content
		leftGutter := string(m.EndOfBufferCharacter)
		rightGapWidth := m.Width() - lipgloss.Width(leftGutter) + widestLineNumber
		rightGap := strings.Repeat(" ", max(0, rightGapWidth))
		s.WriteString(m.style.computedEndOfBuffer().Render(leftGutter + rightGap))
		s.WriteRune('\n')
	}

	m.viewport.SetContent(s.String())
	return m.style.Base.Render(m.viewport.View())
}

// FuzzVisibleView applies random edits and, after each one, renders the
// editor at several scroll offsets (inside the note, around its end, past
// every row and negative) and checks that View equals refView byte for byte
// and leaves the same offset behind.
func FuzzVisibleView(f *testing.F) {
	f.Add("hello world\nsecond line", []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}, []byte{20, 5, 1})
	f.Add(strings.Repeat("some words that wrap ", 12)+"\n漢字かな😀\nend", []byte{
		1, 0, 1, 2, 1, 4, 1, 6, 7, 3, 2, 9, 4, 11, 8, 7, 20, 30, 40, 5, 15, 25, 35, 45, 55, 65}, []byte{3, 4, 9})
	f.Add("a\nb\nc\nd\ne\nf", []byte{3, 3, 5, 19, 5, 33, 9, 22, 12, 13, 14, 26, 27, 1, 1, 1}, []byte{0, 2, 26})
	f.Add("", []byte{0, 0, 0, 2, 2, 7, 7, 6, 6, 13}, []byte{40, 1, 32})
	f.Add(testnote.Note(40), []byte{11, 25, 39, 53, 67, 81, 95, 109}, []byte{50, 9, 15})
	f.Add(strings.Repeat("漢字かな交じりの長い行 😀 ", 9)+"\nx", []byte{12, 26, 40, 54, 11, 25}, []byte{1, 6, 12})
	f.Add("one\ntwo", []byte{25, 11}, []byte{20, 5, 65})
	f.Add("one\ntwo\nthree", []byte{11, 25}, []byte{20, 5, 65})

	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	f.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	f.Fuzz(func(t *testing.T, seed string, ops, cfg []byte) {
		if len(ops) > 100 {
			ops = ops[:100]
		}
		cfg = append(cfg, 0, 0, 0)
		flags := cfg[2]
		m := New()
		m.Prompt = ""
		m.ShowLineNumbers = flags&1 != 0
		if flags&2 != 0 {
			m.SetPromptFunc(3, func(i int) string { return strconv.Itoa(i%100) + ">" })
		}
		if flags&32 != 0 {
			m.Placeholder = "type something here, it wraps around"
		}
		m.EndOfBufferCharacter = '~'
		m.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#303030"))
		m.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0d0d0"))
		m.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#606060"))
		m.FocusedStyle.CursorLineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true)
		m.FocusedStyle.EndOfBuffer = lipgloss.NewStyle().Foreground(lipgloss.Color("#404040"))
		if flags&64 != 0 {
			// A wider cursor-line number cell changes the end-of-buffer pad.
			m.FocusedStyle.CursorLineNumber = m.FocusedStyle.CursorLineNumber.PaddingRight(2)
		}
		m.SelectionStyle = lipgloss.NewStyle().Background(lipgloss.Color("#3a3a6a"))
		m.SetWidth(10 + int(cfg[0])%60)
		m.SetHeight(1 + int(cfg[1])%15)
		m.SetValue(seed)
		m.Focus()
		if flags&4 != 0 {
			m.Blur()
		}

		check := func(op string, off int) {
			t.Helper()
			m.viewport.YOffset = off
			want := refView(m)
			wantOff := m.viewport.YOffset
			m.viewport.YOffset = off
			got := m.View()
			if got != want {
				t.Fatalf("after %s at offset %d (rows=%d height=%d width=%d):\n got %q\nwant %q",
					op, off, m.visualLineCount(), m.height, m.width, got, want)
			}
			if m.viewport.YOffset != wantOff {
				t.Fatalf("after %s at offset %d: View left offset %d, want %d", op, off, m.viewport.YOffset, wantOff)
			}
		}
		for i, b := range ops {
			op := applyFuzzOp(&m, b, seed)
			if flags&8 != 0 && len(m.value) > 0 {
				m.SelectLinewise = flags&16 != 0
				m.SetAnchor(int(b)%len(m.value), int(b)%9)
			}
			cur := m.viewport.YOffset
			total, h := m.visualLineCount(), m.height
			for _, off := range []int{
				cur, 0, cur - 1, cur + 1, int(b) % (total + 1), total - 1, total, total + h - 1,
				total + h, total + h + 1, total + h + 7, -1, -h - 2,
			} {
				check(fmt.Sprintf("op %d (%s)", i, op), off)
			}
			m.viewport.YOffset = cur
		}
	})
}
