package app_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// trueColour renders in 24-bit colour for the rest of the test.
func trueColour(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// hexOf is a theme colour as the renderer writes it, "#rrggbb" (lipgloss
// can round a channel by one).
func hexOf(c lipgloss.Color) string {
	return screenCells(lipgloss.NewStyle().Foreground(c).Render("x"))[0].fg
}

// colourView is the frame with its colours, checked like h.view.
func (h *harness) colourView() string {
	h.t.Helper()
	v := h.m.View()
	assertFrame(h.t, v)
	return v
}

// screenCell is one terminal cell: its rune and its colours as "#rrggbb"
// ("" when unset).
type screenCell struct {
	r      rune
	fg, bg string
}

// screenCells splits a rendered line into cells, following the SGR colour
// state the way a terminal does. Wide runes take one entry.
func screenCells(line string) []screenCell {
	var out []screenCell
	var fg, bg string
	for i := 0; i < len(line); {
		if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && (line[j] < 0x40 || line[j] > 0x7e) {
				j++
			}
			if j < len(line) && line[j] == 'm' {
				fg, bg = applySGR(line[i+2:j], fg, bg)
			}
			i = j + 1
			continue
		}
		r, n := utf8.DecodeRuneInString(line[i:])
		out = append(out, screenCell{r, fg, bg})
		i += n
	}
	return out
}

func applySGR(params, fg, bg string) (string, string) {
	p := strings.Split(params, ";")
	for k := 0; k < len(p); k++ {
		switch p[k] {
		case "", "0":
			fg, bg = "", ""
		case "39":
			fg = ""
		case "49":
			bg = ""
		case "38", "48":
			if k+4 < len(p) && p[k+1] == "2" {
				var c [3]int
				for n := range c {
					c[n], _ = strconv.Atoi(p[k+2+n])
				}
				h := fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
				if p[k] == "38" {
					fg = h
				} else {
					bg = h
				}
				k += 4
			}
		}
	}
	return fg, bg
}

// cellAt is the cell in screen column x of row y.
func cellAt(view string, x, y int) screenCell {
	row := screenCells(strings.Split(view, "\n")[y])
	if x >= len(row) {
		return screenCell{}
	}
	return row[x]
}

// paneText returns the colours of the non-blank cells in columns [x0, x1)
// of the body rows.
func paneText(view string, x0, x1 int) []screenCell {
	var out []screenCell
	lines := strings.Split(view, "\n")
	for y := 1; y < len(lines)-1; y++ {
		row := screenCells(lines[y])
		for x := x0; x < x1 && x < len(row); x++ {
			if row[x].r != ' ' {
				out = append(out, row[x])
			}
		}
	}
	return out
}

// assertDimmed checks every visible character of a pane is grey (or the
// separator colour of indent guides).
func assertDimmed(t *testing.T, view, pane string, x0, x1 int) {
	t.Helper()
	th := theme.DefaultTheme()
	cells := paneText(view, x0, x1)
	if len(cells) == 0 {
		t.Fatalf("%s shows no text:\n%s", pane, ansi.Strip(view))
	}
	for _, c := range cells {
		if c.fg != hexOf(th.Grey) && c.fg != hexOf(th.Line) {
			t.Fatalf("%s is not dimmed: %q is %s, want grey %s", pane, c.r, c.fg, hexOf(th.Grey))
		}
	}
}

// assertLit checks a pane keeps its own text colours: some character is
// drawn in the theme's text colour.
func assertLit(t *testing.T, view, pane string, x0, x1 int) {
	t.Helper()
	th := theme.DefaultTheme()
	for _, c := range paneText(view, x0, x1) {
		if c.fg == hexOf(th.Fg) {
			return
		}
	}
	t.Fatalf("%s (the active pane) has no text in %s:\n%s", pane, hexOf(th.Fg), ansi.Strip(view))
}

// focusLayout is the wide layout with explorer, editor and preview.
var focusLayout = app.CalculateLayout(termW, termH, true, true, false, app.Split{})

// assertDividers checks the colour of the explorer/editor and the
// editor/preview dividers.
func assertDividers(t *testing.T, view string, sidebarFocus, previewFocus bool) {
	t.Helper()
	th := theme.DefaultTheme()
	want := func(focused bool) string {
		if focused {
			return hexOf(th.BorderFocus)
		}
		return hexOf(th.Line)
	}
	for _, d := range []struct {
		name    string
		x       int
		focused bool
	}{
		{"explorer/editor divider", focusLayout.SidebarW, sidebarFocus},
		{"editor/preview divider", focusLayout.PreviewX - 1, previewFocus},
	} {
		c := cellAt(view, d.x, 5)
		if c.r != '│' {
			t.Fatalf("no %s at column %d:\n%s", d.name, d.x, ansi.Strip(view))
		}
		if c.fg != want(d.focused) {
			t.Errorf("%s is %s, want %s", d.name, c.fg, want(d.focused))
		}
	}
}

// focusHarness opens a note with explorer, editor and preview on screen;
// the editor has the focus.
func focusHarness(t *testing.T) *harness {
	t.Helper()
	trueColour(t)
	store := newTestStore(t)
	if _, err := store.SaveAs("", "note", "plain words here\n\nmore words"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	return h
}

func TestFocusDividersAroundActiveEditor(t *testing.T) {
	h := focusHarness(t)
	assertDividers(t, h.colourView(), true, true)
}

func TestFocusDividersAroundActiveExplorer(t *testing.T) {
	h := focusHarness(t)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlH})
	assertDividers(t, h.colourView(), true, false)
}

func TestFocusDividersAroundActivePreview(t *testing.T) {
	h := focusHarness(t)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlL})
	assertDividers(t, h.colourView(), false, true)
}

func TestFocusDimsInactivePanes(t *testing.T) {
	h := focusHarness(t)
	l := focusLayout
	v := h.colourView()
	assertLit(t, v, "editor", l.EditorX, l.EditorX+l.EditorW)
	assertDimmed(t, v, "explorer", 0, l.SidebarW)
	assertDimmed(t, v, "preview", l.PreviewX, l.PreviewX+l.PreviewW)

	h.send(tea.KeyMsg{Type: tea.KeyCtrlL})
	v = h.colourView()
	assertLit(t, v, "preview", l.PreviewX, l.PreviewX+l.PreviewW)
	assertDimmed(t, v, "editor", l.EditorX, l.EditorX+l.EditorW)
	assertDimmed(t, v, "explorer", 0, l.SidebarW)
}

func TestFocusMovesWithTab(t *testing.T) {
	h := focusHarness(t)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlH})
	h.key(tea.KeyTab)
	assertDividers(t, h.colourView(), true, true)
	h.key(tea.KeyTab)
	assertDividers(t, h.colourView(), false, true)
}

func TestFocusMovesWithCtrlHL(t *testing.T) {
	h := focusHarness(t)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlL})
	assertDividers(t, h.colourView(), false, true)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlH})
	assertDividers(t, h.colourView(), true, true)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlH})
	assertDividers(t, h.colourView(), true, false)
}

func TestFocusMovesWithClick(t *testing.T) {
	h := focusHarness(t)
	l := focusLayout
	h.click(l.PreviewX+l.PreviewW-2, l.BodyY+l.BodyH-2)
	assertDividers(t, h.colourView(), false, true)
	h.click(l.EditorX+l.EditorW-2, l.BodyY+l.BodyH-2)
	assertDividers(t, h.colourView(), true, true)
}

// The narrow explorer opens over the editor and takes the focus: the editor
// behind it reads as inactive.
func TestFocusExplorerOverlayDimsEditor(t *testing.T) {
	trueColour(t)
	store := newTestStore(t)
	long := strings.Repeat("word ", 9)
	if _, err := store.SaveAs("", "alpha", long+"\n"+long); err != nil {
		t.Fatal(err)
	}
	h := newHarnessSize(t, narrowW, narrowH, store)
	openRecent(h)
	h.send(ctrlB())
	v := h.m.View()
	if !explorerShown(ansi.Strip(v)) {
		t.Fatalf("overlay not open:\n%s", ansi.Strip(v))
	}
	th := theme.DefaultTheme()
	edge := app.CalculateLayout(narrowW, narrowH, true, false, false, app.Split{}).SidebarW
	if c := cellAt(v, narrowEditorPreviewDivider, 5); c.fg != hexOf(th.Line) {
		t.Errorf("editor/preview divider is %s, want %s", c.fg, hexOf(th.Line))
	}
	assertDimmed(t, v, "editor behind the overlay", edge+1, narrowEditorPreviewDivider)
}

// In reading mode the editor is hidden and the explorer's divider borders
// the focused preview, so it is lit.
func TestFocusDividerLitInReadingMode(t *testing.T) {
	h := focusHarness(t)
	h.keys(" r")
	v := h.colourView()
	th := theme.DefaultTheme()
	c := cellAt(v, focusLayout.SidebarW, 5)
	if c.r != '│' || c.fg != hexOf(th.BorderFocus) {
		t.Fatalf("explorer divider in reading mode is %q %s, want │ %s:\n%s", c.r, c.fg, hexOf(th.BorderFocus), ansi.Strip(v))
	}
}

// The graph takes the editor's place: lit with the focus, grey without it.
func TestFocusDimsGraphPane(t *testing.T) {
	trueColour(t)
	h := graphWorkspace(t)
	l := focusLayout
	assertLit(t, h.colourView(), "graph", l.EditorX, l.EditorX+l.EditorW)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlH})
	assertDimmed(t, h.colourView(), "graph", l.EditorX, l.EditorX+l.EditorW)
}

// A divider being dragged stays lit even when it does not touch the focused
// pane, as before focus colours existed.
func TestFocusDraggedDividerStaysLit(t *testing.T) {
	h := focusHarness(t)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlH})
	div := focusLayout.PreviewX - 1
	h.send(tea.MouseMsg{X: div, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	th := theme.DefaultTheme()
	if c := cellAt(h.colourView(), div, 5); c.fg != hexOf(th.Blue) {
		t.Fatalf("dragged divider is %s, want %s", c.fg, hexOf(th.Blue))
	}
}
