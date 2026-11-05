package content_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// paint is the SGR that opens a run of text drawn in style st.
func paint(st lipgloss.Style) string { return strings.SplitN(st.Render("x"), "x", 2)[0] }

// TestSlashMenuFollowsTheTransparency: the block menu floats over the
// editor. On a transparent theme it paints no fill and keeps its border and
// text colours; on the opaque theme its fill and border keep PopupBg.
func TestSlashMenuFollowsTheTransparency(t *testing.T) {
	trueColour(t)
	const popupSGR = "48;2;10;11;12"
	for _, clear := range []bool{true, false} {
		th := theme.DefaultTheme()
		th.Bg, th.DarkerBg, th.StatusBg = "#101112", "#0a0b0c", "#131415"
		th.Base, th.PopupBg = th.Bg, th.DarkerBg
		if clear {
			th = th.Transparent()
		}

		c := content.New(th)
		c.SetSize(80, 20)
		c.SetPage(core.Page{ID: "a.md", Content: ""})
		c.SetFocused(true)
		c = keys(c, "i/zzzz") // no block matches: the menu says so on its fill
		box, _, _, ok := c.SlashView()
		if !ok {
			t.Fatal("setup: the slash menu should be open")
		}
		bg := lipgloss.NewStyle().Background(th.PopupBg)
		for _, w := range []string{
			paint(bg.Foreground(th.Line)) + "╭",                           // border
			paint(bg.Foreground(th.GreyFg).Italic(true)) + "  No results", // fill
		} {
			if !strings.Contains(box, w) {
				t.Errorf("transparent=%v: missing %q in:\n%q", clear, w, box)
			}
		}
		if got := strings.Contains(box, popupSGR); got == clear {
			t.Errorf("transparent=%v: PopupBg fill present=%v:\n%q", clear, got, box)
		}
	}
}
