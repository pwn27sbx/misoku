package content

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// Clicks on the rendered view. A rendered row is not the Markdown under it,
// so the app asks RenderedCell where the preview's compile has that cell, to
// act on what the preview would act on; a click that hits nothing lands on
// the source text the row shows (clickAt).

// Spot is what a pane row shows, by source line, so the row can be found
// again after a compile moved it.
type Spot struct {
	line, off int
	footer    bool
}

// SpotAt returns what pane row y shows.
func (m Model) SpotAt(y int) Spot {
	line, off, footer := m.textarea.DisplayRowAt(m.textarea.YOffset() + y)
	return Spot{line: line, off: off, footer: footer}
}

// SpotY returns the pane row showing s now, when it is still on screen.
func (m Model) SpotY(s Spot) (int, bool) {
	l := s.line
	if s.footer {
		l = m.LineCount()
	}
	first, n := m.textarea.LineRows(l)
	y := first + s.off - m.textarea.YOffset()
	return y, s.off < n && y >= 0 && y < m.textarea.Height()
}

// RenderedCell returns the compile's row and column under cell (x, y) of the
// pane, when a rendered row or the footer is there and the rows are the
// compile's own, not shifted to a newer text. Raw lines, the margin and an
// open popup report false.
func (m Model) RenderedCell(x, y int) (row, col int, ok bool) {
	s := m.rendered
	col = x - m.margin
	if !s.on || !s.current || col < 0 || m.slash != nil || m.linkList != nil {
		return 0, 0, false
	}
	line, off, footer := m.textarea.DisplayRowAt(m.textarea.YOffset() + y)
	if footer {
		return s.footerRow + off, col, off < len(s.footer)
	}
	b, ok := s.renderedBlock(line, off)
	if !ok {
		return 0, 0, false
	}
	return b.Row + off, col, true
}

// renderedBlock returns the block drawn rendered whose rows start on line,
// when it has a row off.
func (s renderedState) renderedBlock(line, off int) (RenderedBlock, bool) {
	if !s.on || (line >= s.from && line < s.to) {
		return RenderedBlock{}, false
	}
	i := sort.Search(len(s.blocks), func(i int) bool { return s.blocks[i].Line >= line })
	if i == len(s.blocks) || s.blocks[i].Line != line || off < 0 || off >= len(s.blocks[i].Rows) {
		return RenderedBlock{}, false
	}
	return s.blocks[i], true
}

// clickAt puts the cursor under cell (x, y) of the pane. On a rendered row it
// finds the source text the row shows there.
func (m *Model) clickAt(x, y int) {
	m.textarea.ClickAt(x, y)
	line, off, footer := m.textarea.DisplayRowAt(m.textarea.YOffset() + y)
	if footer {
		return
	}
	b, ok := m.rendered.renderedBlock(line, off)
	if !ok {
		return
	}
	end := b.End
	if b.Folded {
		end = b.Line + 1 // the body is hidden: never land in it
	}
	lines := make([]string, 0, end-b.Line)
	for l := b.Line; l < min(end, m.LineCount()); l++ {
		lines = append(lines, m.textarea.RowText(l))
	}
	l, c := sourceAt(lines, b.Rows[:off+1], x-m.margin)
	m.textarea.SetRowCol(b.Line+l, c)
}

// sourceAt maps column col of the last of rows, rendered from lines, to a
// line of lines and a rune column in it. The words of the rows up to the
// column are looked up in the source one after another, so markers the rows
// hide are skipped; the column lands at the same place in the word it is on,
// or after the last word found. Without any, the first line's start.
func sourceAt(lines, rows []string, col int) (line, column int) {
	src := strings.Join(lines, "\n")
	pos, at := 0, 0 // where the next lookup starts; the answer, in bytes
	for k, row := range rows {
		for _, w := range words(ansi.Strip(row)) {
			click := k == len(rows)-1 && col < w.x+w.width
			if click && col < w.x {
				break // between words: after the last one found
			}
			i := strings.Index(src[pos:], w.text)
			if i < 0 {
				continue // drawn, not written: a glyph, a number
			}
			if click {
				at = pos + i + len(string([]rune(w.text)[:w.runeAt(col)]))
				break
			}
			pos = pos + i + len(w.text)
			at = pos
		}
	}
	line = strings.Count(src[:at], "\n")
	start := strings.LastIndex(src[:at], "\n") + 1
	return line, utf8.RuneCountInString(src[start:at])
}

// word is a run of letters and digits of a rendered row, at cell x.
type word struct {
	text     string
	x, width int
}

// runeAt is the rune of w under cell col, clamped into the word.
func (w word) runeAt(col int) int {
	x := w.x
	for i, r := range []rune(w.text) {
		x += ansi.StringWidth(string(r))
		if col < x {
			return i
		}
	}
	return utf8.RuneCountInString(w.text)
}

// words splits row into its runs of letters and digits.
func words(row string) []word {
	var out []word
	x, start := 0, -1
	var b strings.Builder
	flush := func() {
		if start >= 0 {
			out = append(out, word{text: b.String(), x: start, width: x - start})
			b.Reset()
			start = -1
		}
	}
	for _, r := range row {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = x
			}
			b.WriteRune(r)
		} else {
			flush()
		}
		x += ansi.StringWidth(string(r))
	}
	flush()
	return out
}
