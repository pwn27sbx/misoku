package app_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"
)

// narrowW is under the default explorer-hide width, wide enough for the
// preview to stay open next to the editor.
const narrowW, narrowH = 100, 30

// narrowEditorPreviewDivider is the editor/preview divider column when the
// explorer takes no columns at narrowW.
var narrowEditorPreviewDivider = app.CalculateLayout(narrowW, narrowH, false, true, false, app.Split{}).PreviewX - 1

func narrowStore(t *testing.T) *core.Store {
	t.Helper()
	store := newTestStore(t)
	for _, n := range []string{"alpha", "bravo"} {
		if _, err := store.SaveAs("", n, n+" body"); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

// screen returns the plain-text frame at whatever size the harness has.
func (h *harness) screen() string {
	h.t.Helper()
	return ansi.Strip(h.m.View())
}

func explorerShown(v string) bool { return strings.Contains(v, "Find note") }

func hasCol(cols []int, c int) bool {
	for _, x := range cols {
		if x == c {
			return true
		}
	}
	return false
}

func ctrlB() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlB} }

// openRecent opens the most recent note from the home screen.
func openRecent(h *harness) {
	h.t.Helper()
	h.keys("1")
}

func TestNarrowExplorerStartsHidden(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	v := h.screen()
	if explorerShown(v) {
		t.Fatalf("explorer must start hidden under the threshold:\n%s", v)
	}
	if cols := dividers(v, 5); len(cols) != 1 || cols[0] != narrowEditorPreviewDivider {
		t.Fatalf("dividers %v, want only the editor/preview one at %d:\n%s", cols, narrowEditorPreviewDivider, v)
	}
}

func TestNarrowExplorerOpensAsOverlayWithoutShrinkingEditor(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.send(ctrlB())
	v := h.screen()
	if !explorerShown(v) {
		t.Fatalf("toggle_explorer must show the explorer:\n%s", v)
	}
	cols := dividers(v, 5)
	if !hasCol(cols, app.SidebarWidth) {
		t.Errorf("overlay edge not at column %d: %v\n%s", app.SidebarWidth, cols, v)
	}
	if !hasCol(cols, narrowEditorPreviewDivider) {
		t.Errorf("editor shrank: editor/preview divider moved from %d, dividers %v\n%s", narrowEditorPreviewDivider, cols, v)
	}
}

func TestNarrowExplorerUsesEffectiveToggleKey(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t), app.WithKeys(map[string][]string{"toggle_explorer": {"space e"}}))
	openRecent(h)
	h.send(tea.KeyMsg{Type: tea.KeySpace})
	h.keys("e")
	if v := h.screen(); !explorerShown(v) {
		t.Fatalf("space e must open the explorer when configured:\n%s", v)
	}
}

func TestNarrowExplorerClosesOnNoteChoice(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.send(ctrlB())
	h.keys("gg")
	h.key(tea.KeyEnter)
	v := h.screen()
	if explorerShown(v) {
		t.Fatalf("choosing a note must close the overlay:\n%s", v)
	}
	if !strings.Contains(v, "alpha body") {
		t.Errorf("the chosen note did not open:\n%s", v)
	}
}

func TestNarrowExplorerClosesOnEsc(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.send(ctrlB())
	h.key(tea.KeyEsc)
	if v := h.screen(); explorerShown(v) {
		t.Fatalf("Esc must close the overlay:\n%s", v)
	}
}

// From the home screen there is no note to fall back to: Esc still closes.
func TestNarrowExplorerClosesOnEscWithoutNote(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	h.keys("e")
	if v := h.screen(); !explorerShown(v) {
		t.Fatalf("e on the home screen must open the explorer:\n%s", v)
	}
	h.key(tea.KeyEsc)
	if v := h.screen(); explorerShown(v) {
		t.Fatalf("Esc must close the overlay with no note open:\n%s", v)
	}
}

func TestExplorerManualToggleKeptAboveThresholdUntilCrossing(t *testing.T) {
	h := newHarness(t, narrowStore(t))
	openRecent(h)
	h.send(ctrlB()) // hide by hand on a wide window
	if v := h.screen(); explorerShown(v) {
		t.Fatalf("ctrl+b must hide the explorer:\n%s", v)
	}
	h.send(tea.WindowSizeMsg{Width: 130, Height: termH})
	if v := h.screen(); explorerShown(v) {
		t.Fatalf("a resize that stays above the threshold must keep it hidden:\n%s", v)
	}
	h.send(tea.WindowSizeMsg{Width: narrowW, Height: termH})
	h.send(tea.WindowSizeMsg{Width: termW, Height: termH})
	if v := h.screen(); !explorerShown(v) {
		t.Fatalf("crossing the threshold back up must dock the explorer again:\n%s", v)
	}

	// Shown by hand above the threshold, then the window narrows: hidden.
	h.send(tea.WindowSizeMsg{Width: narrowW, Height: termH})
	if v := h.screen(); explorerShown(v) {
		t.Fatalf("crossing the threshold down must hide the explorer:\n%s", v)
	}
}

func TestNarrowExplorerMouseClickOpensNote(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.send(ctrlB())
	v := h.screen()
	x, y := -1, -1
	for row, l := range strings.Split(v, "\n") {
		r := []rune(l)
		if len(r) < app.SidebarWidth {
			continue
		}
		if i := strings.Index(string(r[:app.SidebarWidth]), "alpha"); i >= 0 && row > 0 {
			x, y = ansi.StringWidth(string(r[:app.SidebarWidth])[:i]), row
			break
		}
	}
	if y < 0 {
		t.Fatalf("alpha not in the overlay:\n%s", v)
	}
	h.click(x, y)
	v = h.screen()
	if explorerShown(v) || !strings.Contains(v, "alpha body") {
		t.Fatalf("clicking a note in the overlay must open it and close the overlay:\n%s", v)
	}
}

func TestNarrowExplorerClickOutsideCloses(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.send(ctrlB())
	h.click(app.SidebarWidth+5, 5)
	if v := h.screen(); explorerShown(v) {
		t.Fatalf("a click on the editor must close the overlay:\n%s", v)
	}
}

func dragX(h *harness, from, to, y int) {
	h.t.Helper()
	h.send(
		tea.MouseMsg{X: from, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: to, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: to, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
}

func TestNarrowExplorerDragResizesOverlay(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.send(ctrlB())
	dragX(h, app.SidebarWidth, app.SidebarWidth+8, 5)
	v := h.screen()
	cols := dividers(v, 5)
	if !hasCol(cols, app.SidebarWidth+8) || hasCol(cols, app.SidebarWidth) {
		t.Errorf("overlay edge not dragged to %d: %v\n%s", app.SidebarWidth+8, cols, v)
	}
	if !hasCol(cols, narrowEditorPreviewDivider) {
		t.Errorf("dragging the overlay moved the editor: %v", cols)
	}
	// A drag past the limit is clamped, and moving back moves it at once.
	limit := app.CalculateLayout(narrowW, narrowH, true, false, false, app.Split{SidebarW: 500}).SidebarW
	h.send(
		tea.MouseMsg{X: app.SidebarWidth + 8, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: 95, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
	)
	if cols := dividers(h.screen(), 5); !hasCol(cols, limit) {
		t.Errorf("overlay edge not clamped at %d: %v", limit, cols)
	}
	h.send(
		tea.MouseMsg{X: limit - 2, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: limit - 2, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
	if cols := dividers(h.screen(), 5); !hasCol(cols, limit-2) {
		t.Errorf("overlay edge did not come back to %d: %v", limit-2, cols)
	}
}

func TestWideExplorerDragAndClickUnchanged(t *testing.T) {
	h := newHarness(t, narrowStore(t))
	openRecent(h)
	dragX(h, app.SidebarWidth, app.SidebarWidth+8, 5)
	if cols := dividers(h.view(), 5); !hasCol(cols, app.SidebarWidth+8) {
		t.Fatalf("docked explorer divider not dragged: %v", cols)
	}
	// Clicking the editor leaves the docked explorer where it is.
	h.click(app.SidebarWidth+20, 5)
	if v := h.view(); !explorerShown(v) {
		t.Fatalf("a click on the editor must not hide the docked explorer:\n%s", v)
	}
}

func TestWideLayoutUnchanged(t *testing.T) {
	h := newHarness(t, narrowStore(t))
	openRecent(h)
	v := h.view()
	want := app.CalculateLayout(termW, termH, true, true, false, app.Split{})
	cols := dividers(v, 5)
	if len(cols) != 2 || cols[0] != want.SidebarW || cols[1] != want.PreviewX-1 {
		t.Fatalf("wide dividers %v, want [%d %d]:\n%s", cols, want.SidebarW, want.PreviewX-1, v)
	}
	if !explorerShown(v) {
		t.Fatal("explorer must start docked on a wide window")
	}
}

func TestExplorerHideWidthOption(t *testing.T) {
	// A lower threshold docks the explorer at narrowW.
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t), app.WithExplorerHideWidth(80))
	openRecent(h)
	if v := h.screen(); !explorerShown(v) || !hasCol(dividers(v, 5), app.SidebarWidth) {
		t.Fatalf("threshold 80 must dock the explorer at %d columns:\n%s", narrowW, v)
	}
	// Zero never hides it.
	h = newHarnessSize(t, 70, narrowH, narrowStore(t), app.WithExplorerHideWidth(0))
	openRecent(h)
	if v := h.screen(); !explorerShown(v) {
		t.Fatalf("threshold 0 must never hide the explorer:\n%s", v)
	}
	// A higher threshold hides it on the default test window.
	h = newHarness(t, narrowStore(t), app.WithExplorerHideWidth(termW+1))
	openRecent(h)
	if v := h.view(); explorerShown(v) {
		t.Fatalf("threshold %d must hide the explorer at %d columns:\n%s", termW+1, termW, v)
	}
}

func TestDefaultExplorerHideWidth(t *testing.T) {
	if app.DefaultExplorerHideWidth != 120 {
		t.Fatalf("DefaultExplorerHideWidth = %d, want 120", app.DefaultExplorerHideWidth)
	}
	for _, c := range []struct {
		w    int
		show bool
	}{{119, false}, {120, true}} {
		h := newHarnessSize(t, c.w, narrowH, narrowStore(t))
		openRecent(h)
		if got := explorerShown(h.screen()); got != c.show {
			t.Errorf("width %d: explorer shown %v, want %v", c.w, got, c.show)
		}
	}
}

// Tab cycles from the overlay to the editor, the next pane on screen.
func TestNarrowExplorerTabMovesToEditor(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.send(ctrlB())
	h.key(tea.KeyTab)
	v := h.screen()
	if explorerShown(v) || !strings.Contains(v, "NORMAL") {
		t.Fatalf("Tab from the overlay must close it and focus the editor:\n%s", v)
	}
}

// Settings change the threshold on a running app: the current width is
// judged again at once.
func TestExplorerHideWidthOnRunningModel(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	m := h.m.(app.Model)
	app.WithExplorerHideWidth(80)(&m)
	h.m = m
	if v := h.screen(); !explorerShown(v) || !hasCol(dividers(v, 5), app.SidebarWidth) {
		t.Fatalf("lowering the threshold under the width must dock the explorer:\n%s", v)
	}
}
