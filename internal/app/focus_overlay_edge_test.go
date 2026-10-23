package app_test

import (
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// The narrow explorer's overlay has the focus, so its edge is lit like the
// divider of a docked, focused explorer.
func TestFocusExplorerOverlayEdgeLit(t *testing.T) {
	trueColour(t)
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.send(ctrlB())
	v := h.m.View()
	if !explorerShown(ansi.Strip(v)) {
		t.Fatalf("overlay not open:\n%s", ansi.Strip(v))
	}
	th := theme.DefaultTheme()
	edge := app.CalculateLayout(narrowW, narrowH, true, false, false, app.Split{}).SidebarW
	if c := cellAt(v, edge, 5); c.r != '│' || c.fg != hexOf(th.BorderFocus) {
		t.Errorf("overlay edge is %q %s, want │ %s", c.r, c.fg, hexOf(th.BorderFocus))
	}
}
