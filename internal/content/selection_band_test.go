package content_test

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// selectionEditor is a focused editor in NORMAL, with the cursor on the first
// of three lines.
func selectionEditor(t *testing.T, th theme.Theme, text string) content.Model {
	t.Helper()
	trueColour(t)
	c := content.New(th)
	c.SetSize(40, 8)
	c.SetPage(core.Page{ID: "n.md", Title: "n", Content: text})
	c.Focus()
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	return c
}

const selectionNote = "first line\nsecond line\nthird line"

func hasBackground(view string, c lipgloss.Color) bool {
	_, bg := colours(view)
	return bg[rgb(c)]
}

func assertNoBand(t *testing.T, th theme.Theme, view string) {
	t.Helper()
	if hasBackground(view, th.Bg2) {
		t.Errorf("the cursor-line band is drawn while selecting: %q", view)
	}
}

func assertBand(t *testing.T, th theme.Theme, view string) {
	t.Helper()
	if !hasBackground(view, th.Bg2) {
		t.Errorf("the cursor-line band is missing: %q", view)
	}
}

func mouse(c content.Model, x, y int, action tea.MouseAction) content.Model {
	c, _ = c.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: action})
	return c
}

func TestNoCursorLineBandInVisual(t *testing.T) {
	th := theme.DefaultTheme()
	c := keys(selectionEditor(t, th, selectionNote), "vl")
	if c.Mode() != content.ModeVisual {
		t.Fatalf("mode = %v, want VISUAL", c.Mode())
	}
	assertNoBand(t, th, c.View())
}

func TestNoCursorLineBandInVisualLine(t *testing.T) {
	th := theme.DefaultTheme()
	c := keys(selectionEditor(t, th, selectionNote), "Vj")
	if c.Mode() != content.ModeVisualLine {
		t.Fatalf("mode = %v, want VISUAL-LINE", c.Mode())
	}
	assertNoBand(t, th, c.View())
}

func TestNoCursorLineBandDuringMouseSelection(t *testing.T) {
	th := theme.DefaultTheme()
	c := selectionEditor(t, th, selectionNote)
	c = mouse(c, 8, 0, tea.MouseActionPress)
	assertBand(t, th, c.View()) // a press alone selects nothing yet
	c = mouse(c, 12, 0, tea.MouseActionMotion)
	view := c.View()
	if !hasBackground(view, th.OneBg2) {
		t.Fatalf("the drag selected nothing: %q", view)
	}
	if c.Mode() != content.ModeNormal {
		t.Fatalf("mode = %v, a mouse selection has no VISUAL mode", c.Mode())
	}
	assertNoBand(t, th, view)
}

func TestNoCursorLineBandInVisualOnEmptyNote(t *testing.T) {
	th := theme.DefaultTheme()
	c := selectionEditor(t, th, "")
	assertBand(t, th, c.View())
	c = keys(c, "v")
	assertNoBand(t, th, c.View())
}

func TestCursorLineBandReturnsAfterEsc(t *testing.T) {
	th := theme.DefaultTheme()
	c := selectionEditor(t, th, selectionNote)
	before := c.View()
	c = keys(c, "v")
	assertNoBand(t, th, c.View())
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := c.View(); got != before {
		t.Errorf("Esc does not restore the NORMAL rendering:\n got %q\nwant %q", got, before)
	}
}

// numberRe matches the line number n drawn with the bold foreground fg,
// capturing any background on the same sequence.
func numberRe(fg lipgloss.Color, n string) *regexp.Regexp {
	return regexp.MustCompile(`\x1b\[1;38;2;` + rgb(fg) + `(;48;2;[0-9;]+)?m\s*` + n + `\s*\x1b\[0m`)
}

func TestCursorLineNumberStaysHighlightedWhileSelecting(t *testing.T) {
	th := theme.DefaultTheme()
	c := keys(selectionEditor(t, th, selectionNote), "jv")
	view := c.View()
	match := numberRe(th.Fg, "2").FindStringSubmatch(view)
	if match == nil {
		t.Fatalf("line number 2 lost its highlight: %q", view)
	}
	if match[1] != "" {
		t.Errorf("line number 2 keeps a band background %q while selecting", match[1])
	}
}

func TestSelectionBackgroundIsTheActiveThemesOneBg2(t *testing.T) {
	second, ok := theme.Get("nord")
	if !ok {
		t.Fatal("no nord theme")
	}
	custom := theme.DefaultTheme()
	custom.OneBg2 = lipgloss.Color("#123456")

	assertSelection := func(t *testing.T, th theme.Theme, view string) {
		t.Helper()
		if th.OneBg2 == th.OneBg3 {
			t.Fatal("the theme cannot tell the two tokens apart")
		}
		if !strings.Contains(view, "48;2;"+rgb(th.OneBg2)+"mirst line") {
			t.Errorf("selected text is not on the theme's OneBg2 %s: %q", rgb(th.OneBg2), view)
		}
		if hasBackground(view, th.OneBg3) {
			t.Errorf("OneBg3 is still painted: %q", view)
		}
	}

	first := theme.DefaultTheme()
	c := keys(selectionEditor(t, first, selectionNote), "V")
	assertSelection(t, first, c.View())

	// Live theme changes, a built-in and a custom palette alike.
	for _, th := range []theme.Theme{second, custom} {
		c.SetTheme(th)
		assertSelection(t, th, c.View())
	}
}

func TestSyntaxColourSurvivesInsideSelection(t *testing.T) {
	th := theme.DefaultTheme()
	c := selectionEditor(t, th, "plain\n# Heading")
	syntax := regexp.MustCompile(`38;2;([0-9;]+)m[^\x1b]*Heading`).FindStringSubmatch(c.View())
	if syntax == nil || syntax[1] == rgb(th.Fg) {
		t.Fatalf("the heading has no syntax colour to keep: %v", syntax)
	}
	view := keys(c, "Vj").View()
	want := "38;2;" + syntax[1] + ";48;2;" + rgb(th.OneBg2) + "m"
	if !regexp.MustCompile(regexp.QuoteMeta(want) + `[^\x1b]*Heading`).MatchString(view) {
		t.Errorf("selected heading lost its colour, want %q before the text: %q", want, view)
	}
}

func TestCursorLineBandUnchangedInNormalAndInsert(t *testing.T) {
	th := theme.DefaultTheme()
	c := selectionEditor(t, th, selectionNote)
	check := func(mode string, c content.Model) {
		t.Helper()
		view := c.View()
		assertBand(t, th, view)
		match := numberRe(th.Fg, "1").FindStringSubmatch(view)
		if match == nil || match[1] != ";48;2;"+rgb(th.Bg2) {
			t.Errorf("%s: line number 1 is off the band: %v in %q", mode, match, view)
		}
	}
	check("NORMAL", c)
	c = keys(c, "i")
	if c.Mode() != content.ModeInsert {
		t.Fatalf("mode = %v, want INSERT", c.Mode())
	}
	check("INSERT", c)
}
