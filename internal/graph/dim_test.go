package graph_test

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/graph"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func dimmableGraph(th theme.Theme) graph.Model {
	var notes []core.Page
	for _, n := range []string{"Hub", "Docker", "AWS", "Study"} {
		notes = append(notes, core.Page{ID: n + ".md", Title: n})
	}
	edges := []core.Edge{{From: "Hub.md", To: "Docker.md"}, {From: "Hub.md", To: "AWS.md"}, {From: "Study.md", To: "Hub.md"}}
	g := graph.New(th)
	g.SetSize(80, 16)
	g.SetLinks(notes, edges)
	g.SetCentre("Hub.md")
	return g
}

func TestGraphSetDimmedGreysText(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	g := dimmableGraph(th)
	lit := g.View()
	g.SetDimmed(true)
	assertGreyed(t, th, lit, g.View())
}

// Undimmed, the graph renders exactly as it did before dimming existed.
func TestGraphUndimmedRendersAsBefore(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	g := dimmableGraph(th)
	before := g.View()
	g.SetDimmed(true)
	g.SetDimmed(false)
	if g.View() != before {
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
