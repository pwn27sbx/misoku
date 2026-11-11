package textarea_test

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pwn27sbx/misoku/internal/textarea"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/rivo/uniseg"
)

// testMarker is a small stand-in for the editor's list detection: the
// textarea only sees the function, so any rule will do.
var testMarker = regexp.MustCompile(`^[ \t]*[-*+][ \t]+(\[[ xX-]\][ \t]+)?`)

func testIndent(line []rune) int {
	return utf8.RuneCountInString(testMarker.FindString(string(line)))
}

// hangEditor returns a bare editor (no prompt, no line numbers, so screen x
// is the text column) of the given width holding text, with the hanging
// indent on unless indent is nil.
func hangEditor(width int, indent func([]rune) int, text string) textarea.Model {
	m := textarea.New()
	m.Prompt = ""
	m.ShowLineNumbers = false
	m.CharLimit = 0
	m.MaxHeight = 0
	m.WrapIndent = indent
	m.SetWidth(width)
	m.SetHeight(12)
	m.SetValue(text)
	m.Focus()
	m.GotoTop()
	return m
}

// hangRows returns the text rows of the view, escapes stripped.
func hangRows(m textarea.Model) []string {
	rows := strings.Split(ansi.Strip(m.View()), "\n")
	var out []string
	for _, r := range rows {
		if strings.TrimSpace(r) == "" || strings.TrimSpace(r) == "~" {
			continue
		}
		out = append(out, r)
	}
	return out
}

// task is the line every cursor test uses. At width 20 with a 6-cell indent
// it wraps as
//
//   - [ ] alpha beta     (runes 0-16)
//     gamma delta    (runes 17-28)
//     epsilon zeta   (runes 29-41)
const task = "- [ ] alpha beta gamma delta epsilon zeta"

func TestHangIndentDrawsContinuationUnderText(t *testing.T) {
	m := hangEditor(20, testIndent, task)
	m.HideCursor = true
	want := []string{
		"- [ ] alpha beta    ",
		"      gamma delta   ",
		"      epsilon zeta  ",
	}
	got := hangRows(m)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("rows:\n got %q\nwant %q", got, want)
	}
	if m.Value() != task {
		t.Errorf("the indent changed the text: %q", m.Value())
	}
}

// Every drawn row is exactly the text width, wherever the cursor is, for list
// lines with long words (V3), wide runes and a cursor past the end (V5).
func TestHangIndentRowsKeepTheWidth(t *testing.T) {
	texts := []string{
		task,
		"- " + strings.Repeat("a", 30),
		"- [x] " + strings.Repeat("b", 40) + " tail words here",
		"* 漢字かな交じりの長い行は幅の広い文字で折り返される 漢字",
		"  - nested item with enough words to wrap a few times over",
		"+ ab cd ef gh ij kl mn op qr st uv wx yz",
	}
	for _, text := range texts {
		for col := 0; col <= utf8.RuneCountInString(text); col++ {
			m := hangEditor(20, testIndent, text)
			m.SetRowCol(0, col)
			for i, row := range hangRows(m) {
				if w := uniseg.StringWidth(row); w != 20 {
					t.Fatalf("%q cursor %d: row %d is %d cells wide: %q", text, col, i, w, row)
				}
			}
		}
	}
}

// A word longer than the narrow width is cut to the continuation rows' width
// and every rune is still drawn once (V3).
func TestHangIndentLongWordFitsTheNarrowRow(t *testing.T) {
	text := "- " + strings.Repeat("a", 30)
	m := hangEditor(20, testIndent, text)
	m.HideCursor = true
	want := []string{
		"-                   ",
		"  " + strings.Repeat("a", 18),
		"  " + strings.Repeat("a", 12) + "      ",
	}
	if got := hangRows(m); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("rows:\n got %q\nwant %q", got, want)
	}
}

func TestHangIndentCursorScreen(t *testing.T) {
	m := hangEditor(20, testIndent, task)
	cases := []struct{ col, x, y int }{
		{12, 12, 0}, // b of beta, first row
		{17, 6, 1},  // g of gamma, first cell after the indent
		{23, 12, 1}, // d of delta
		{37, 14, 2}, // z of zeta
		{41, 18, 2}, // the end of the line
	}
	for _, c := range cases {
		m.SetRowCol(0, c.col)
		x, y, ok := m.CursorScreen()
		if !ok || x != c.x || y != c.y {
			t.Errorf("col %d: CursorScreen = (%d,%d,%v), want (%d,%d)", c.col, x, y, ok, c.x, c.y)
		}
	}
}

// The drawn cursor sits on the same cell CursorScreen reports.
func TestHangIndentDrawnCursor(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	m := hangEditor(20, testIndent, task)
	m.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	m.Cursor.SetMode(cursor.CursorStatic)
	for _, col := range []int{12, 17, 23, 37, 41} {
		m.SetRowCol(0, col)
		x, y, _ := m.CursorScreen()
		row := cells(m.View())[y]
		found := -1
		for i, c := range row {
			if strings.Contains(c.sgr, "38;2;255;0;0") {
				found = i
			}
		}
		if found != x {
			t.Errorf("col %d: cursor drawn at cell %d of row %d, CursorScreen says %d", col, found, y, x)
		}
	}
}

func TestHangIndentUpDownKeepScreenColumn(t *testing.T) {
	m := hangEditor(20, testIndent, task)
	m.SetRowCol(0, 12) // b of beta, screen column 12
	steps := []struct {
		move func()
		col  int
	}{
		{m.CursorDown, 23}, // d of delta
		{m.CursorDown, 35}, // n of epsilon, column 12
		{m.CursorUp, 23},
		{m.CursorUp, 12},
	}
	for i, s := range steps {
		s.move()
		if _, c := m.RowCol(); c != s.col {
			t.Fatalf("step %d: col %d, want %d", i, c, s.col)
		}
	}

	// From the first character after the indent (screen column 6) up lands
	// on screen column 6 of the first row.
	m.SetRowCol(0, 17)
	m.CursorUp()
	if _, c := m.RowCol(); c != 6 {
		t.Errorf("up from g of gamma: col %d, want 6", c)
	}

	// A goal column inside the indent lands on the segment's first
	// character and survives to the next row.
	m.SetRowCol(0, 2)
	m.CursorDown()
	if _, c := m.RowCol(); c != 17 {
		t.Errorf("down from column 2: col %d, want 17", c)
	}
	m.CursorDown()
	if _, c := m.RowCol(); c != 29 {
		t.Errorf("second down from column 2: col %d, want 29", c)
	}
	m.CursorUp()
	m.CursorUp()
	if _, c := m.RowCol(); c != 2 {
		t.Errorf("back up to the first row: col %d, want 2", c)
	}
}

func TestHangIndentClickAt(t *testing.T) {
	m := hangEditor(20, testIndent, task)
	cases := []struct{ x, y, col int }{
		{12, 0, 12}, // b of beta
		{12, 1, 23}, // d of delta
		{6, 1, 17},  // g of gamma
		{3, 1, 17},  // inside the indent: the segment's first character
		{14, 2, 37}, // z of zeta
	}
	for _, c := range cases {
		m.ClickAt(c.x, c.y)
		if _, col := m.RowCol(); col != c.col {
			t.Errorf("click (%d,%d): col %d, want %d", c.x, c.y, col, c.col)
		}
	}
}

// A drag selects the characters under the pointer, and copying them gives
// the note's text without the indent (S3).
func TestHangIndentDragSelection(t *testing.T) {
	m := hangEditor(20, testIndent, task)
	m.ClickAt(6, 1)
	m.StartSelection()
	m.ClickAt(9, 2)
	if got, want := m.SelectedText(), "gamma delta epsi"; got != want {
		t.Errorf("drag selected %q, want %q", got, want)
	}
}

// Scrolling with the wheel drags the cursor along without moving it left:
// the rune offset in the row is kept, not shifted by the indent (V4).
func TestHangIndentScrollDoesNotDrift(t *testing.T) {
	text := strings.TrimSuffix(strings.Repeat(task+"\n", 12), "\n")
	m := hangEditor(20, testIndent, text)
	m.SetHeight(3)
	m.SetRowCol(0, 21) // a of gamma, offset 4 in the second row
	m.EnsureVisible()
	moved := 0
	for i := 0; i < 20; i++ {
		before, _ := m.RowCol()
		m.ScrollBy(1)
		if r, _ := m.RowCol(); r != before {
			moved++
		}
		if off := m.LineInfo().ColumnOffset; off != 4 {
			t.Fatalf("scroll %d: column offset %d, want 4", i, off)
		}
	}
	if moved == 0 {
		t.Fatal("the scroll never dragged the cursor")
	}
}

type cell struct {
	r   rune
	sgr string
}

// cells splits a view into rows of cells, each with the escape that styles it.
func cells(view string) [][]cell {
	var rows [][]cell
	for _, line := range strings.Split(view, "\n") {
		var row []cell
		sgr := ""
		for i := 0; i < len(line); {
			if line[i] == '\x1b' {
				j := strings.IndexByte(line[i:], 'm')
				seq := line[i : i+j+1]
				if seq == "\x1b[0m" || seq == "\x1b[m" {
					sgr = ""
				} else {
					sgr += seq
				}
				i += j + 1
				continue
			}
			r, n := utf8.DecodeRuneInString(line[i:])
			row = append(row, cell{r, sgr})
			i += n
		}
		rows = append(rows, row)
	}
	return rows
}

// styled returns the characters drawn with an escape containing code.
func styled(view, code string) string {
	var b strings.Builder
	for _, row := range cells(view) {
		for _, c := range row {
			if strings.Contains(c.sgr, code) {
				b.WriteRune(c.r)
			}
		}
	}
	return b.String()
}

// Syntax colours and the selection land on the same characters with and
// without the indent.
func TestHangIndentColoursLandOnTheSameCharacters(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	for _, indent := range []func([]rune) int{nil, testIndent} {
		m := hangEditor(20, indent, task)
		m.HideCursor = true
		red := lipgloss.Color("#ff0000")
		colors := make([]lipgloss.Color, utf8.RuneCountInString(task))
		for i := 23; i <= 27; i++ { // delta
			colors[i] = red
		}
		for i := 37; i <= 40; i++ { // zeta
			colors[i] = red
		}
		m.LineColors = [][]lipgloss.Color{colors}
		m.SelectionStyle = lipgloss.NewStyle().Background(lipgloss.Color("#0000ff"))
		m.SetAnchor(0, 29)
		m.SetRowCol(0, 35) // epsilon
		view := m.View()
		if got := styled(view, "38;2;255;0;0"); got != "deltazeta" {
			t.Errorf("indent %v: red drawn on %q, want deltazeta", indent != nil, got)
		}
		if got := styled(view, "48;2;0;0;255"); got != "epsilon" {
			t.Errorf("indent %v: selection drawn on %q, want epsilon", indent != nil, got)
		}
	}
}

// S5: the indent is used only while it is at most half the width.
func TestHangIndentThreshold(t *testing.T) {
	words := "item text here with long words that wrap again and again"
	at := strings.Repeat(" ", 8) + "- " + words   // indent 10 = 20/2
	over := strings.Repeat(" ", 9) + "- " + words // indent 11
	if rows := hangRows(hangEditor(20, testIndent, at)); !strings.HasPrefix(rows[1], strings.Repeat(" ", 10)+"w") &&
		!strings.HasPrefix(rows[1], strings.Repeat(" ", 10)+"h") {
		t.Errorf("indent 10 at width 20 not drawn: %q", rows)
	}
	on, off := hangEditor(20, testIndent, over), hangEditor(20, nil, over)
	on.HideCursor, off.HideCursor = true, true
	if on.View() != off.View() {
		t.Errorf("indent 11 at width 20 changed the view:\n%s\n%s", on.View(), off.View())
	}
}

// S6: lines the function gives no indent to, and an editor without the
// function, draw exactly as before.
func TestHangIndentOffIsUnchanged(t *testing.T) {
	text := "plain words that wrap across the width\n" + task + "\n漢字かな交じりの長い行は幅の広い文字"
	none := func([]rune) int { return 0 }
	for col := 0; col < 30; col++ {
		a, b := hangEditor(20, nil, text), hangEditor(20, none, text)
		a.SetRowCol(1, col)
		b.SetRowCol(1, col)
		if a.View() != b.View() {
			t.Fatalf("col %d: a zero indent changed the view", col)
		}
	}
	rows := hangRows(hangEditor(20, nil, task))
	if strings.HasPrefix(rows[1], " ") {
		t.Errorf("without the option the continuation is indented: %q", rows)
	}
}
