package app

import (
	"sort"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// palette is the command palette, which is also the help (?, SPC h): every
// key and command, from the keymap in use, the editor's command catalog and
// the fixed keys (palette_keys.go), with a search field. Everything typed
// goes to the search, ? q and Space included; only Esc closes. Enter runs
// the selected entry when it is a command, and the bottom explains it.
type palette struct {
	input   textinput.Model
	entries []paletteEntry
	// shown: the indexes of the entries listed, best match first; sel is
	// a position in it.
	shown []int
	sel   int
	// offset is the first line of the list in view: the list scrolls by
	// lines, so no row is ever cut to make it fit.
	offset int
}

// openPalette opens the palette with an empty search.
func (m *Model) openPalette() tea.Cmd {
	th := m.theme
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "Type to search keys and commands"
	in.CharLimit = 80
	field := lipgloss.NewStyle().Background(th.Bg2)
	in.TextStyle = field.Foreground(th.Fg)
	in.PlaceholderStyle = field.Foreground(th.GreyFg)
	in.Cursor.Style = lipgloss.NewStyle().Foreground(th.Blue)
	in.Cursor.TextStyle = field.Foreground(th.Fg)

	p := &palette{input: in, entries: m.paletteEntries()}
	p.refilter()
	m.palette = p
	m.leaderPending = false
	return p.input.Focus()
}

// refilter lists the entries matching the search, fuzzily as the note
// finder does, best first; an empty search lists them all in order.
func (p *palette) refilter() {
	q := strings.TrimSpace(p.input.Value())
	p.shown = p.shown[:0]
	scores := map[int]int{}
	for i, e := range p.entries {
		best, ok := 0, false
		for _, target := range []string{e.title, e.key} {
			if s, _, hit := fuzzyScore(q, target); hit && (!ok || s > best) {
				best, ok = s, true
			}
		}
		if ok {
			p.shown = append(p.shown, i)
			scores[i] = best
		}
	}
	if q != "" {
		sort.SliceStable(p.shown, func(a, b int) bool { return scores[p.shown[a]] > scores[p.shown[b]] })
	}
	p.sel, p.offset = 0, 0
}

// selected is the highlighted entry, or nil when nothing matches.
func (p *palette) selected() *paletteEntry {
	if p.sel < len(p.shown) {
		return &p.entries[p.shown[p.sel]]
	}
	return nil
}

func (p *palette) update(m *Model, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.palette = nil
			return nil
		case "ctrl+c":
			return m.requestQuit(false)
		case "enter":
			e := p.selected()
			if e == nil || e.run == nil {
				return nil // a key, not a command: the bottom explains it
			}
			m.palette = nil
			return e.run(m)
		case "down", "ctrl+n", "ctrl+j":
			p.move(m, 1)
			return nil
		case "up", "ctrl+p", "ctrl+k":
			p.move(m, -1)
			return nil
		case "pgdown":
			p.move(m, p.layout(m).listRows/2)
			return nil
		case "pgup":
			p.move(m, -p.layout(m).listRows/2)
			return nil
		}
		before := p.input.Value()
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		if p.input.Value() != before {
			p.refilter()
		}
		return cmd
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				p.move(m, -1)
			case tea.MouseButtonWheelDown:
				p.move(m, 1)
			}
		}
		return nil
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return cmd
}

// move moves the highlight by delta, without wrapping, and scrolls it into
// view.
func (p *palette) move(m *Model, delta int) {
	if len(p.shown) == 0 {
		return
	}
	p.sel = min(max(p.sel+delta, 0), len(p.shown)-1)
	p.scroll(p.layout(m))
}

// paletteLine is one line of the list: an entry's (entry >= 0) or a
// heading's or a blank's (entry -1).
type paletteLine struct {
	text  string
	entry int  // position in shown
	key   bool // all key: a key too wide for its column, on its own line
}

// paletteLayout is the palette's size for the current terminal.
type paletteLayout struct {
	inner    int // width inside the borders
	keyW     int // width of the key column
	listRows int // lines of the list in view
	long     []string
}

// paletteMaxWidth bounds the palette's inner width on wide terminals.
const paletteMaxWidth = 96

func (p *palette) layout(m *Model) paletteLayout {
	inner := max(min(m.width-4, paletteMaxWidth), 20)
	// The key column fits the longest key, up to half the width; a longer
	// key gets a line of its own instead of being cut.
	keyW := 0
	for _, e := range p.entries {
		keyW = max(keyW, lipgloss.Width(e.key))
	}
	keyW = min(keyW, (inner-6)/2)
	l := paletteLayout{inner: inner, keyW: keyW, long: p.longLines(inner)}
	// Rows: title, search, blank, the list, blank, the explanation, the
	// footer. The list keeps the height of the whole list (or of the
	// screen), so the box does not jump while typing.
	room := max(m.height-2-6-len(l.long), 3)
	l.listRows = min(room, len(p.lines(l, true)))
	return l
}

// longLines is the selected entry's explanation, wrapped.
func (p *palette) longLines(inner int) []string {
	text := "Nothing matches the search."
	if e := p.selected(); e != nil {
		text = e.long
	}
	return strings.Split(ansi.Wrap(text, inner-2, ""), "\n")
}

// lines lays out the list: under headings when the search is empty (or
// all is set, to measure the full list), best match first otherwise.
func (p *palette) lines(l paletteLayout, all bool) []paletteLine {
	shown := p.shown
	headings := all || strings.TrimSpace(p.input.Value()) == ""
	if all {
		shown = make([]int, len(p.entries))
		for i := range shown {
			shown[i] = i
		}
	}
	var out []paletteLine
	group := ""
	for pos, i := range shown {
		e := p.entries[i]
		if headings && e.group != group {
			if group != "" {
				out = append(out, paletteLine{entry: -1})
			}
			group = e.group
			out = append(out, paletteLine{text: e.group, entry: -1})
		}
		keyLines, rest := entryLines(e, l)
		for _, t := range keyLines {
			out = append(out, paletteLine{text: t, entry: pos, key: true})
		}
		for _, t := range rest {
			out = append(out, paletteLine{text: t, entry: pos})
		}
	}
	return out
}

// entryLines renders one entry: the key padded to the key column, two
// spaces and the title, wrapped under itself. A key wider than the column
// takes lines of its own, returned apart.
func entryLines(e paletteEntry, l paletteLayout) (keyLines, lines []string) {
	titleW := max(l.inner-4-l.keyW-2, 8)
	indent := strings.Repeat(" ", l.keyW+2)
	key := e.key
	if lipgloss.Width(key) > l.keyW {
		keyLines = strings.Split(ansi.Wrap(key, l.inner-4, ""), "\n")
		key = ""
	}
	for i, t := range strings.Split(ansi.Wrap(e.title, titleW, ""), "\n") {
		if i == 0 {
			lines = append(lines, padRight(key, l.keyW)+"  "+t)
			continue
		}
		lines = append(lines, indent+t)
	}
	return keyLines, lines
}

// scroll keeps every line of the highlighted entry in view, and its heading
// too when it is the first of the list.
func (p *palette) scroll(l paletteLayout) {
	lines := p.lines(l, false)
	first, last := -1, -1
	for i, ln := range lines {
		if ln.entry == p.sel {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return
	}
	if p.sel == 0 {
		first = 0
	}
	if first < p.offset {
		p.offset = first
	}
	if last >= p.offset+l.listRows {
		p.offset = last - l.listRows + 1
	}
	p.offset = max(min(p.offset, len(lines)-l.listRows), 0)
}

func (p *palette) view(m *Model) (string, int, int) {
	th := m.theme
	l := p.layout(m)
	p.scroll(l) // the window may have shrunk since the last move
	inner := l.inner
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	headStyle := bg.Foreground(th.Purple).Bold(true)
	keyStyle := bg.Foreground(th.Blue).Bold(true)
	textStyle := bg.Foreground(th.Fg)
	selStyle := lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Blue).Bold(true)

	rows := []string{bg.Foreground(th.Blue).Render(" ") + bg.Foreground(th.Fg).Bold(true).Render("MISOKU KEYMAPS & COMMANDS")}
	field := lipgloss.NewStyle().Background(th.Bg2)
	rows = append(rows, bg.Render(" ")+ui.FitLine(field.Foreground(th.Blue).Bold(true).Render("  ")+p.input.View(), inner-2, field)+bg.Render(" "))
	rows = append(rows, "")

	lines := p.lines(l, false)
	for r := 0; r < l.listRows; r++ {
		i := p.offset + r
		switch {
		case i >= len(lines):
			if i == 0 {
				rows = append(rows, bg.Foreground(th.GreyFg).Render("   Nothing matches"))
				continue
			}
			rows = append(rows, "")
		case lines[i].entry < 0:
			rows = append(rows, headStyle.Render(" "+lines[i].text))
		case lines[i].entry == p.sel:
			rows = append(rows, ui.FitLine(selStyle.Render("▌ "+lines[i].text), inner, selStyle))
		case lines[i].key:
			rows = append(rows, keyStyle.Render("  "+lines[i].text))
		default:
			key, rest := ansiCut(lines[i].text, l.keyW)
			rows = append(rows, keyStyle.Render("  "+key)+textStyle.Render(rest))
		}
	}

	rows = append(rows, "")
	for _, ln := range l.long {
		rows = append(rows, bg.Foreground(th.GreyFg2).Render(" "+ln))
	}
	hint := " Enter run · ↑↓ move · Esc close"
	switch e := p.selected(); {
	case e != nil && e.needsNote:
		hint = " Needs an open note · Esc close"
	case e == nil || e.run == nil:
		hint = " Not a command: use the key · Esc close"
	}
	if n := len(p.shown); n > 0 {
		count := strconv.Itoa(p.sel+1) + "/" + strconv.Itoa(n) + " "
		if gap := inner - lipgloss.Width(hint) - lipgloss.Width(count); gap > 0 {
			hint += strings.Repeat(" ", gap) + count
		}
	}
	rows = append(rows, bg.Foreground(th.GreyFg).Render(hint))

	box := panel(th, rows, inner)
	x, y := ui.Center(m.width, m.height, lipgloss.Width(box), lipgloss.Height(box))
	return box, x, y
}

// ansiCut splits plain text s after w cells.
func ansiCut(s string, w int) (string, string) {
	return ansi.Cut(s, 0, w), ansi.Cut(s, w, lipgloss.Width(s))
}

func padRight(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}
