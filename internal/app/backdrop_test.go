package app

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// The distinct theme's grey (the dim colour), fg and blue as foreground SGRs.
const (
	greyFG = "38;2;128;129;130"
	fgFG   = "38;2;192;193;194"
	blueFG = "38;2;16;80;224"
)

// frameCell is one terminal cell of a frame: its rune and the foreground and
// background SGRs in effect ("" for the terminal's default).
type frameCell struct {
	r      rune
	fg, bg string
}

// frameCells splits a frame into rows of cells, following the SGR state the
// way a terminal does. Combining marks join the cell before them.
func frameCells(frame string) [][]frameCell {
	var rows [][]frameCell
	for _, line := range strings.Split(frame, "\n") {
		var row []frameCell
		var fg, bg string
		for i := 0; i < len(line); {
			if line[i] == 0x1b {
				n, params, sgr := escapeLen(line[i:])
				if sgr {
					fg, bg = applySGR(params, fg, bg)
				}
				i += n
				continue
			}
			r, n := utf8.DecodeRuneInString(line[i:])
			if ansi.StringWidth(string(r)) > 0 || r == ' ' || r >= 0x10000 {
				row = append(row, frameCell{r, fg, bg})
			}
			i += n
		}
		rows = append(rows, row)
	}
	return rows
}

// escapeLen measures the escape sequence at the start of s and returns an
// SGR's parameters.
func escapeLen(s string) (int, string, bool) {
	if len(s) > 1 && s[1] == '[' {
		for j := 2; j < len(s); j++ {
			if s[j] >= 0x40 && s[j] <= 0x7e {
				return j + 1, s[2:j], s[j] == 'm'
			}
		}
	}
	if len(s) > 1 && (s[1] == ']' || s[1] == '_') {
		if end := strings.Index(s, "\x1b\\"); end > 0 {
			return end + 2, "", false
		}
	}
	return min(2, len(s)), "", false
}

// applySGR updates the foreground and background after SGR params.
func applySGR(params, fg, bg string) (string, string) {
	p := strings.Split(params, ";")
	for k := 0; k < len(p); k++ {
		switch t := p[k]; {
		case t == "" || t == "0":
			fg, bg = "", ""
		case t == "39":
			fg = ""
		case t == "49":
			bg = ""
		case t == "38" || t == "48" || t == "58":
			n := 3
			if k+1 < len(p) && p[k+1] == "2" {
				n = 5
			}
			end := min(k+n, len(p))
			v := strings.Join(p[k:end], ";")
			if t == "38" {
				fg = v
			} else if t == "48" {
				bg = v
			}
			k = end - 1
		case len(t) == 2 && (t[0] == '3' || t[0] == '9') && t[1] <= '7':
			fg = t
		case len(t) == 2 && t[0] == '4' && t[1] <= '7':
			bg = t
		}
	}
	return fg, bg
}

// rect is a box on screen.
type rect struct{ x, y, w, h int }

func (r rect) has(x, y int) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// boxRect is where a box rendered at (x, y) lands.
func boxRect(box string, x, y int) rect {
	return rect{x, y, ansi.StringWidth(strings.SplitN(box, "\n", 2)[0]), strings.Count(box, "\n") + 1}
}

// colouredOutside counts the visible cells outside r drawn in a colour other
// than the grey, and the cells outside r carrying a background other than
// base ("" for none).
func colouredOutside(frame string, r rect, base string) (coloured, painted int) {
	for y, row := range frameCells(frame) {
		for x, c := range row {
			if r.has(x, y) {
				continue
			}
			if c.r != ' ' && c.fg != greyFG {
				coloured++
			}
			if c.bg != base {
				painted++
			}
		}
	}
	return coloured, painted
}

// hasInside reports whether some cell inside r is drawn in foreground fg.
func hasInside(frame string, r rect, fg string) bool {
	for y, row := range frameCells(frame) {
		for x, c := range row {
			if r.has(x, y) && c.fg == fg && c.r != ' ' {
				return true
			}
		}
	}
	return false
}

// press sends key to m.
func press(m *Model, key tea.KeyMsg) {
	tm, _ := m.Update(key)
	*m = tm.(Model)
}

// S1-S4: while a modal box is open, everything behind it is grey text on the
// base background (none with ui.transparent); the box keeps its colours; the
// next frame after closing it has the colours back.
func TestModalOverlaysDimTheBackdrop(t *testing.T) {
	for _, c := range []struct {
		name string
		opts func(t *testing.T) []Option
		// open opens the box and returns where it is drawn.
		open  func(m *Model) rect
		close tea.KeyMsg
	}{
		{"welcome dialog on Home", func(t *testing.T) []Option {
			return []Option{WithLearnOffer(true), WithConfigPath(filepath.Join(t.TempDir(), "config.json"))}
		}, func(m *Model) rect {
			if m.confirm == nil || m.confirm.welcome == nil {
				t.Fatal("setup: the welcome dialog did not open on Home")
			}
			return boxRect(m.confirm.view(m.theme, m.width, m.height))
		}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}},
		{"confirm dialog over a note", func(*testing.T) []Option {
			return []Option{WithOpenNote("doc.md")}
		}, func(m *Model) rect {
			m.confirmDelete("doc.md")
			return boxRect(m.confirm.view(m.theme, m.width, m.height))
		}, tea.KeyMsg{Type: tea.KeyEsc}},
		{"finder over a note", func(*testing.T) []Option {
			return []Option{WithOpenNote("doc.md")}
		}, func(m *Model) rect {
			m.openFinder()
			return boxRect(m.finder.view(m))
		}, tea.KeyMsg{Type: tea.KeyEsc}},
		{"palette over a note", func(*testing.T) []Option {
			return []Option{WithOpenNote("doc.md")}
		}, func(m *Model) rect {
			m.openPalette()
			return boxRect(m.palette.view(m))
		}, tea.KeyMsg{Type: tea.KeyEsc}},
	} {
		for _, clear := range []bool{false, true} {
			name := c.name + "/opaque"
			base := "48;2;16;17;18" // the distinct theme's bg
			if clear {
				name, base = c.name+"/transparent", ""
			}
			t.Run(name, func(t *testing.T) {
				m := distinctModel(t, 120, 40, clear, c.opts(t)...)
				if c.name != "welcome dialog on Home" {
					if before, _ := colouredOutside(m.View(), rect{}, base); before == 0 {
						t.Fatal("setup: the screen has no colour to dim")
					}
				}
				r := c.open(m)
				frame := m.View()
				coloured, painted := colouredOutside(frame, r, base)
				if coloured > 0 {
					t.Errorf("%d cells behind the box keep a colour other than the grey:\n%s", coloured, ansi.Strip(frame))
				}
				if painted > 0 {
					t.Errorf("%d cells behind the box keep a background other than the base %q", painted, base)
				}
				if !hasInside(frame, r, blueFG) {
					t.Errorf("the box lost its blue border:\n%q", frame)
				}
				press(m, c.close)
				if m.modalOpen() {
					t.Fatal("setup: the box did not close")
				}
				if after, _ := colouredOutside(m.View(), rect{}, base); after == 0 {
					t.Error("closing the box left the screen grey")
				}
			})
		}
	}
}

// Inline popups are not modal: the leader guide and the slash menu leave the
// screen in its colours.
func TestInlinePopupsLeaveTheScreenInColour(t *testing.T) {
	m := distinctModel(t, 120, 40, false, WithOpenNote("doc.md"))
	m.leaderPending, m.leaderGuide = true, true
	// The guide sits at the bottom: the top rows (tabs and the note) are
	// what it must leave in colour.
	top := rect{0, 3, m.width, m.height}
	if coloured, _ := colouredOutside(m.View(), top, "48;2;16;17;18"); coloured == 0 {
		t.Error("the leader guide greyed out the screen")
	}
}

// Unfocused panes keep their own dimming: with the explorer focused, the
// note's text is grey while the explorer keeps its colours.
func TestUnfocusedPanesStillDim(t *testing.T) {
	m := distinctModel(t, 120, 40, false, WithOpenNote("doc.md"))
	m.sidebarOpen = true
	m.focus = focusSidebar
	m.updateLayout()
	l := m.layout()
	if l.SidebarW == 0 || l.EditorW == 0 {
		t.Fatal("setup: the explorer and the editor should both show")
	}
	frame := m.View()
	explorer := rect{0, l.BodyY, l.SidebarW, l.BodyH}
	editor := rect{l.EditorX, l.BodyY, l.EditorW, l.BodyH}
	if !hasInside(frame, editor, greyFG) || hasInside(frame, editor, fgFG) {
		t.Errorf("the unfocused editor is not grey:\n%s", ansi.Strip(frame))
	}
	if !hasInside(frame, explorer, blueFG) {
		t.Errorf("the focused explorer lost its colours:\n%s", ansi.Strip(frame))
	}
}
