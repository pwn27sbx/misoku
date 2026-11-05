package textarea_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/textarea"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// viewRows splits a View into its screen rows with the escape codes removed.
func viewRows(m textarea.Model) []string {
	return strings.Split(ansi.Strip(m.View()), "\n")
}

// tagHook renders every line it is given as one row "R<line>:<text>".
func tagHook(line int, runes []rune, _ int) []string {
	return []string{fmt.Sprintf("R%d:%s", line, string(runes))}
}

func TestRenderLineReplacesOtherLines(t *testing.T) {
	m := frameEditor(20, 5, false, "one\ntwo\nthree")
	var widths []int
	m.RenderLine = func(line int, runes []rune, width int) []string {
		widths = append(widths, width)
		return tagHook(line, runes, width)
	}
	rows := viewRows(m)
	if got := strings.TrimRight(rows[0], " "); got != "one" {
		t.Errorf("cursor row = %q, want the raw %q", got, "one")
	}
	if got := strings.TrimRight(rows[1], " "); got != "R1:two" {
		t.Errorf("row 1 = %q, want the hook's %q", got, "R1:two")
	}
	if got := strings.TrimRight(rows[2], " "); got != "R2:three" {
		t.Errorf("row 2 = %q, want the hook's %q", got, "R2:three")
	}
	for _, w := range widths {
		if w != 20 {
			t.Errorf("hook got width %d, want the editor's 20", w)
		}
	}
}

func TestRenderLineCursorLineStaysRaw(t *testing.T) {
	m := frameEditor(20, 5, false, "one\ntwo\nthree")
	m.CursorDown()
	m.RenderLine = tagHook
	rows := viewRows(m)
	if got := strings.TrimRight(rows[0], " "); got != "R0:one" {
		t.Errorf("row 0 = %q, want the hook's %q", got, "R0:one")
	}
	if got := strings.TrimRight(rows[1], " "); got != "two" {
		t.Errorf("cursor row = %q, want the raw %q", got, "two")
	}
}

// sameShape fails unless got has want's row count and every row is as wide.
func sameShape(t *testing.T, got, want string) {
	t.Helper()
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	if len(g) != len(w) {
		t.Fatalf("view has %d rows, want %d", len(g), len(w))
	}
	for i := range w {
		if lipgloss.Width(g[i]) != lipgloss.Width(w[i]) {
			t.Errorf("row %d is %d wide, want %d: %q", i, lipgloss.Width(g[i]), lipgloss.Width(w[i]), g[i])
		}
	}
}

func TestRenderLineFewerRowsLeavesBlankRows(t *testing.T) {
	text := "top\naaaa bbbb cccc dddd eeee ffff gggg\nend"
	m := frameEditor(16, 10, true, text)
	want := m.View()
	before := strings.Split(ansi.Strip(want), "\n")
	endRow := -1
	for i, r := range before {
		if strings.Contains(r, "end") {
			endRow = i
		}
	}
	if endRow < 3 {
		t.Fatalf("line 1 must wrap to at least 2 rows; view:\n%s", ansi.Strip(want))
	}
	m.RenderLine = func(line int, _ []rune, _ int) []string { return []string{fmt.Sprintf("L%d", line)} }
	got := m.View()
	sameShape(t, got, want)
	rows := strings.Split(ansi.Strip(got), "\n")
	if !strings.Contains(rows[1], "L1") {
		t.Errorf("row 1 = %q, want the hook's row", rows[1])
	}
	for i := 2; i < endRow; i++ {
		if strings.TrimSpace(rows[i]) != "" {
			t.Errorf("row %d = %q, want a blank row", i, rows[i])
		}
	}
	if !strings.Contains(rows[endRow], "L2") {
		t.Errorf("row %d = %q, want the next line where it sat before", endRow, rows[endRow])
	}
}

func TestRenderLineMoreRowsAreTruncated(t *testing.T) {
	m := frameEditor(20, 6, true, "one\ntwo\nthree")
	want := m.View()
	m.RenderLine = func(line int, runes []rune, _ int) []string {
		return []string{fmt.Sprintf("R%d:%s", line, string(runes)), "extra-a", "extra-b"}
	}
	got := m.View()
	sameShape(t, got, want)
	if strings.Contains(got, "extra") {
		t.Errorf("rows past the line's own were drawn:\n%s", ansi.Strip(got))
	}
	rows := strings.Split(ansi.Strip(got), "\n")
	if !strings.Contains(rows[2], "R2:three") {
		t.Errorf("row 2 = %q, want line 2 in its usual row", rows[2])
	}
}

// TestRenderLinePadsInTheLineStyle pins that a hooked row is padded to the
// text width in the line's own style, as a raw row is, so a background colour
// reaches the right edge instead of stopping where the hook's text ends.
func TestRenderLinePadsInTheLineStyle(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	m := frameEditor(20, 4, false, "one\ntwo")
	m.FocusedStyle.Text = m.FocusedStyle.Text.Background(lipgloss.Color("#101010"))
	m.Focus() // reapply the focused style
	m.RenderLine = func(int, []rune, int) []string { return []string{"L1"} }
	row := strings.Split(m.View(), "\n")[1]
	pad := m.FocusedStyle.Text.Render(strings.Repeat(" ", 20-len("L1")))
	if !strings.Contains(row, pad) {
		t.Errorf("hooked row %q lacks the styled padding %q", row, pad)
	}
}

// TestRenderLineNilIsTodaysView pins that an unset hook draws exactly what the
// editor drew before it existed (the frame goldens pin the same for every
// case), also after a hook was set and cleared.
func TestRenderLineNilIsTodaysView(t *testing.T) {
	m := frameEditor(40, 10, true, wideText())
	m.SetRowCol(3, 2)
	m.EnsureVisible()
	want := m.View()
	m.RenderLine = tagHook
	if m.View() == want {
		t.Fatal("hook had no effect; the test proves nothing")
	}
	m.RenderLine = nil
	if got := m.View(); got != want {
		t.Errorf("nil hook changed the view:\n got %q\nwant %q", got, want)
	}
}
