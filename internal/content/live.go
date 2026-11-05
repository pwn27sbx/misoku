package content

// The live view: every line but the cursor's is drawn the way
// render-markdown.nvim draws it, its Markdown markers hidden, while the
// cursor line stays raw and editable. Colours come from the editor's own
// syntax highlighting (hlCache); this file only hides markers, swaps a few
// for glyphs and adds bold, italic and underline.

import (
	"strings"
	"unicode/utf8"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/highlight"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Glyphs the live view draws in place of Markdown markers. The boxes are the
// preview's default ones (preview.DefaultLook).
const (
	liveBullet    = "•"
	liveQuote     = "▎"
	liveUnchecked = "󰄱"
	liveChecked   = "󰄲"
)

// SetLive turns the live view on or off. On, the editor hides its line
// numbers and draws every line but the cursor's rendered; off, it draws
// exactly as it did before.
func (m *Model) SetLive(on bool) {
	m.live = on
	m.textarea.ShowLineNumbers = !on
	// The text width depends on the line-number column: recompute it.
	m.textarea.SetWidth(max(m.width-1, 10))
}

// Live reports whether the live view is on.
func (m Model) Live() bool { return m.live }

// liveCache memoises, for the last text version, theme and width, which
// lines the live view leaves raw and the rows already rendered. It is
// shared across copies, like hlCache, so View can memoise.
type liveCache struct {
	version uint64
	theme   theme.Theme
	width   int
	valid   bool
	raw     []bool
	rows    map[int][]string
}

// renderLine returns the textarea.Model.RenderLine hook for m, drawing each
// line with the syntax colours cols.
func (c *liveCache) renderLine(m Model, cols highlight.Colors) func(int, []rune, int) []string {
	if !c.valid || c.version != m.Version() || c.theme != m.theme {
		c.version, c.theme, c.valid = m.Version(), m.theme, true
		c.raw = rawLines(m.textarea.RowText, m.LineCount())
		c.rows = map[int][]string{}
	}
	th := m.theme
	return func(line int, runes []rune, width int) []string {
		if width != c.width {
			c.width, c.rows = width, map[int][]string{}
		}
		if rows, ok := c.rows[line]; ok {
			return rows
		}
		var lc []lipgloss.Color
		if line < len(cols) {
			lc = cols[line]
		}
		rows := concealLine(string(runes), lc, line < len(c.raw) && c.raw[line], th, width)
		c.rows[line] = rows
		return rows
	}
}

// rawLines marks the lines the live view draws raw: the front matter, fenced
// code blocks with their fences, and pipe tables. It is the single-pass form
// of inFence, run once per text version instead of once per line.
func rawLines(rowText func(int) string, n int) []bool {
	raw := make([]bool, n)
	start := 0
	if n > 1 && strings.TrimRight(rowText(0), "\r") == "---" {
		for end := 1; end < n; end++ {
			if t := strings.TrimRight(rowText(end), "\r"); t == "---" || t == "..." {
				for i := 0; i <= end; i++ {
					raw[i] = true
				}
				start = end + 1
				break
			}
		}
	}
	var open byte
	openLen := 0
	for r := start; r < n; r++ {
		t := strings.TrimSpace(rowText(r))
		c, k := fenceRun(t)
		switch {
		case open != 0:
			raw[r] = true
			if c == open && k >= openLen && strings.TrimSpace(t[k:]) == "" {
				open = 0
			}
		case c != 0:
			raw[r], open, openLen = true, c, k
		default:
			raw[r] = isTableLine(t)
		}
	}
	return raw
}

// cellStyle is how one drawn rune looks; zero fields keep the text style.
type cellStyle struct {
	fg                      lipgloss.Color
	bold, italic, underline bool
}

// cell is one drawn rune.
type cell struct {
	r  rune
	st cellStyle
}

// concealLine draws line in rows no wider than width: raw, with its syntax
// colours cols, when raw is set; otherwise with its Markdown markers hidden.
func concealLine(line string, cols []lipgloss.Color, raw bool, th theme.Theme, width int) []string {
	c := newConcealer(line, cols)
	if !raw {
		c.block()
		c.inline()
	}
	return wrapCells(c.cells(), th, width)
}

// concealer holds, per byte of a line, whether it is hidden, the style it
// gains, and the glyph drawn in its place.
type concealer struct {
	line  string
	cols  []lipgloss.Color // per rune
	hide  []bool
	st    []cellStyle // fg unused: colours come from cols
	subst map[int][]cell
}

func newConcealer(line string, cols []lipgloss.Color) *concealer {
	return &concealer{
		line:  line,
		cols:  cols,
		hide:  make([]bool, len(line)),
		st:    make([]cellStyle, len(line)),
		subst: map[int][]cell{},
	}
}

// colorAt is the syntax colour of the rune at byte b.
func (c *concealer) colorAt(b int) lipgloss.Color {
	if i := utf8.RuneCountInString(c.line[:b]); i < len(c.cols) {
		return c.cols[i]
	}
	return ""
}

func (c *concealer) hideRange(from, to int) {
	for i := from; i < to; i++ {
		c.hide[i] = true
	}
}

// replace hides bytes [from, to) and draws text at from instead.
func (c *concealer) replace(from, to int, text string, st cellStyle) {
	c.hideRange(from, to)
	var cs []cell
	for _, r := range text {
		cs = append(cs, cell{r, st})
	}
	c.subst[from] = cs
}

func (c *concealer) style(from, to int, f func(*cellStyle)) {
	for i := from; i < to; i++ {
		f(&c.st[i])
	}
}

// block conceals the marker a line opens with: a heading's "#"s, a quote's
// ">", a bullet or a to-do's "- [ ]".
func (c *concealer) block() {
	line := c.line
	if n := strings.IndexFunc(line, func(r rune) bool { return r != '#' }); n >= 1 && n <= 6 && (line[n] == ' ' || line[n] == '\t') {
		end := n + len(line[n:]) - len(strings.TrimLeft(line[n:], " \t"))
		c.hideRange(0, end)
		c.style(0, len(line), func(s *cellStyle) { s.bold = true })
		return
	}
	if strings.HasPrefix(strings.TrimLeft(line, " "), ">") {
		for i := 0; i < len(line) && (line[i] == ' ' || line[i] == '>'); i++ {
			if line[i] == '>' {
				c.replace(i, i+1, liveQuote, cellStyle{fg: c.colorAt(i)})
			}
		}
		return
	}
	if !bulletItem.MatchString(line) {
		return
	}
	head, rest, _ := splitItem(line)
	marker := len(head) - len(strings.TrimLeft(head, " \t"))
	switch boxMark(rest) {
	case ' ':
		c.hideRange(marker, len(head))
		c.replace(len(head), len(head)+3, liveUnchecked, cellStyle{fg: c.colorAt(len(head))})
	case 'x', 'X':
		c.hideRange(marker, len(head))
		c.replace(len(head), len(head)+3, liveChecked, cellStyle{fg: c.colorAt(len(head))})
	default:
		c.replace(marker, marker+1, liveBullet, cellStyle{fg: c.colorAt(marker)})
	}
}

// inline conceals inline markers left to right: `code` delimiters,
// [[wiki]] and [text](url) links down to their text, and the delimiters of
// **bold** and *italic*. Nothing inside a code span is touched.
func (c *concealer) inline() {
	line := c.line
	code := make([]bool, len(line))
	for i := range line {
		code[i] = inInlineCode(line, i)
	}
	for i := 0; i < len(line); {
		switch {
		case c.hide[i]:
			i++
		case code[i]:
			i = c.codeSpan(code, i)
		case line[i] == '[' || line[i] == '!':
			i = c.link(i)
		case strings.HasPrefix(line[i:], "**") || strings.HasPrefix(line[i:], "__"):
			i = c.emphasis(code, i, line[i:i+2], func(s *cellStyle) { s.bold = true })
		case line[i] == '*':
			i = c.emphasis(code, i, "*", func(s *cellStyle) { s.italic = true })
		default:
			i++
		}
	}
}

// codeSpan hides the backtick runs that open and close the code span
// starting at byte i, and returns the byte after it.
func (c *concealer) codeSpan(code []bool, i int) int {
	end := i
	for end < len(code) && code[end] {
		end++
	}
	_, open := backtickRun(c.line, i)
	closing := end
	for closing > i+open && c.line[closing-1] == '`' {
		closing--
	}
	c.hideRange(i, i+open)
	c.hideRange(closing, end)
	return end
}

// link conceals the [[wiki]] or [text](url) link at byte i down to the text
// it shows, underlined, and returns the byte after it (i+1 when there is
// no link there).
func (c *concealer) link(i int) int {
	line := c.line
	if l, n, ok := core.WikiLinkAt(line[i:]); ok {
		inner := i + strings.Index(line[i:], "[[") + 2
		c.replace(i, i+n, l.Label(), cellStyle{fg: c.colorAt(inner), underline: true})
		return i + n
	}
	open := i
	if line[i] == '!' {
		open++
	}
	if open >= len(line) || line[open] != '[' {
		return i + 1
	}
	text := strings.IndexByte(line[open:], ']')
	if text < 0 || !strings.HasPrefix(line[open+text:], "](") {
		return i + 1
	}
	end := strings.IndexByte(line[open+text:], ')')
	if end < 0 {
		return i + 1
	}
	closeAt, after := open+text, open+text+end+1
	c.hideRange(i, open+1)
	c.hideRange(closeAt, after)
	c.style(open+1, closeAt, func(s *cellStyle) { s.underline = true })
	return after
}

// emphasis hides the delimiter delim at byte i and its closing one, and
// styles the text between with f. A delimiter without a closing one, or
// with only spaces next to it, is left as text. It returns the byte to scan
// next: the text between is scanned too, so it can hold links or code.
func (c *concealer) emphasis(code []bool, i int, delim string, f func(*cellStyle)) int {
	line := c.line
	from := i + len(delim)
	if from >= len(line) || line[from] == ' ' {
		return from
	}
	for j := from + 1; j+len(delim) <= len(line); j++ {
		if code[j] || !strings.HasPrefix(line[j:], delim) || line[j-1] == ' ' {
			continue
		}
		if len(delim) == 1 && j+1 < len(line) && line[j+1] == delim[0] {
			j++ // part of a "**": not this delimiter
			continue
		}
		c.hideRange(i, from)
		c.hideRange(j, j+len(delim))
		c.style(from, j, f)
		return from
	}
	return from
}

// cells is the line as drawn: its visible runes with their colours and
// styles, and the glyphs that replace hidden markers.
func (c *concealer) cells() []cell {
	var out []cell
	ri := 0
	for b, r := range c.line {
		out = append(out, c.subst[b]...)
		if !c.hide[b] {
			st := c.st[b]
			if ri < len(c.cols) {
				st.fg = c.cols[ri]
			}
			out = append(out, cell{r, st})
		}
		ri++
	}
	return out
}

// wrapCells word-wraps cells to width with ansi.Wrap and styles each row
// on its own, so a style that spans a break carries on to the next row.
// Uncoloured runes use the theme's text colour.
func wrapCells(cells []cell, th theme.Theme, width int) []string {
	var plain strings.Builder
	for _, c := range cells {
		plain.WriteRune(c.r)
	}
	var out []string
	k := 0
	for _, row := range strings.Split(ansi.Wrap(plain.String(), width, ""), "\n") {
		var rowCells []cell
		for _, r := range row {
			// ansi.Wrap only drops spaces at a break: skip them.
			for k < len(cells) && cells[k].r != r && cells[k].r == ' ' {
				k++
			}
			st := cellStyle{}
			if k < len(cells) {
				st = cells[k].st
				k++
			}
			rowCells = append(rowCells, cell{r, st})
		}
		out = append(out, renderCells(rowCells, th))
	}
	return out
}

// renderCells paints a row, one lipgloss run per stretch of equal style.
func renderCells(cells []cell, th theme.Theme) string {
	var b strings.Builder
	for start := 0; start < len(cells); {
		end := start + 1
		for end < len(cells) && cells[end].st == cells[start].st {
			end++
		}
		st := cells[start].st
		fg := st.fg
		if fg == "" {
			fg = th.Fg
		}
		text := make([]rune, 0, end-start)
		for _, c := range cells[start:end] {
			text = append(text, c.r)
		}
		b.WriteString(lipgloss.NewStyle().Foreground(fg).Bold(st.bold).Italic(st.italic).Underline(st.underline).Render(string(text)))
		start = end
	}
	return b.String()
}
