package content_test

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func dimmableEditor(th theme.Theme) content.Model {
	c := content.New(th)
	c.SetSize(40, 8)
	c.SetPage(core.Page{ID: "n.md", Title: "n", Content: "# Heading\n\nsome **bold** text and `code`\n- [ ] task"})
	return c
}

func TestContentSetDimmedGreysText(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	c := dimmableEditor(th)
	lit := c.View()
	c.SetDimmed(true)
	assertGreyed(t, th, lit, c.View())

	empty := content.New(th)
	empty.SetSize(40, 8)
	litEmpty := empty.View()
	empty.SetDimmed(true)
	assertGreyed(t, th, litEmpty, empty.View())
}

// Dimming does not highlight the note again: the syntax colours cached for
// the lit editor stay, so giving it the focus back costs nothing.
func TestContentDimmingKeepsHighlightCache(t *testing.T) {
	th := theme.DefaultTheme()
	c := dimmableEditor(th)
	_ = c.View()
	c.SetDimmed(true)
	_ = c.View()
	if c.HighlightTheme() != th {
		t.Fatal("dimming recomputed the syntax colours")
	}
}

// Undimmed, the editor renders exactly as it did before dimming existed.
func TestContentUndimmedRendersAsBefore(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	c := dimmableEditor(th)
	before := c.View()
	c.SetDimmed(true)
	_ = c.View()
	c.SetDimmed(false)
	if c.View() != before {
		t.Error("SetDimmed(false) does not restore the lit rendering")
	}
}

// trueColour renders in 24-bit colour for the rest of the test.
func trueColour(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

var sgrRe = regexp.MustCompile("\x1b\\[([0-9;]*)m")

// colours lists the 24-bit foregrounds and backgrounds s draws, as "r;g;b".
func colours(s string) (fg, bg map[string]bool) {
	fg, bg = map[string]bool{}, map[string]bool{}
	for _, m := range sgrRe.FindAllStringSubmatch(s, -1) {
		p := strings.Split(m[1], ";")
		for k := 0; k+4 < len(p); k++ {
			if p[k+1] == "2" && (p[k] == "38" || p[k] == "48") {
				c := strings.Join(p[k+2:k+5], ";")
				if p[k] == "38" {
					fg[c] = true
				} else {
					bg[c] = true
				}
				k += 4
			}
		}
	}
	return fg, bg
}

// rgb is a theme colour as the renderer writes it.
func rgb(c lipgloss.Color) string {
	fg, _ := colours(lipgloss.NewStyle().Foreground(c).Render("x"))
	for k := range fg {
		return k
	}
	return ""
}

// assertGreyed checks a dimmed rendering keeps the text and the backgrounds
// of the lit one, and draws every character grey (or in the separator
// colour, already the dimmest).
func assertGreyed(t *testing.T, th theme.Theme, lit, dim string) {
	t.Helper()
	if ansi.Strip(dim) != ansi.Strip(lit) {
		t.Fatalf("dimming changed the text:\n%s\n---\n%s", ansi.Strip(lit), ansi.Strip(dim))
	}
	litFg, litBg := colours(lit)
	fg, bg := colours(dim)
	litText := false
	for c := range litFg {
		litText = litText || (c != rgb(th.Grey) && c != rgb(th.Line))
	}
	if !litText {
		t.Fatalf("the lit rendering is already grey: %v", litFg)
	}
	for c := range fg {
		if c != rgb(th.Grey) && c != rgb(th.Line) {
			t.Errorf("dimmed rendering still draws text in %s (grey is %s)", c, rgb(th.Grey))
		}
	}
	if !reflect.DeepEqual(bg, litBg) {
		t.Errorf("dimming changed the backgrounds: %v, want %v", bg, litBg)
	}
}
