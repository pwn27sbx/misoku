package app_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/app"
)

func TestCalculateLayoutSplit(t *testing.T) {
	def := app.CalculateLayout(140, 40, true, true, app.Split{})
	if def.SidebarW != app.SidebarWidth || def.EditorW != (140-app.SidebarWidth-2)/2 {
		t.Fatalf("default layout changed: %+v", def)
	}

	l := app.CalculateLayout(140, 40, true, true, app.Split{SidebarW: 40, EditorFrac: 0.25})
	if l.SidebarW != 40 || l.EditorX != 41 {
		t.Errorf("sidebar not resized: %+v", l)
	}
	if l.EditorW != 25 || l.PreviewW != 73 || l.PreviewX != 67 {
		t.Errorf("editor/preview split wrong: %+v", l)
	}

	// Drags past the limits are clamped.
	l = app.CalculateLayout(140, 40, true, true, app.Split{SidebarW: 500, EditorFrac: 0.99})
	if l.SidebarW != 60 || l.PreviewW != 20 {
		t.Errorf("limits not applied: %+v", l)
	}
	l = app.CalculateLayout(140, 40, true, true, app.Split{SidebarW: 2, EditorFrac: 0.01})
	if l.SidebarW != 16 || l.EditorW != 20 {
		t.Errorf("limits not applied: %+v", l)
	}
	if w := l.SidebarW + 1 + l.EditorW + 1 + l.PreviewW; w != 140 {
		t.Errorf("panes cover %d columns, want 140", w)
	}
}

// dividers returns the columns of the "│" pane dividers on screen row y.
func dividers(view string, y int) []int {
	var cols []int
	for x, r := range []rune(ansi.Strip(strings.Split(view, "\n")[y])) {
		if r == '│' {
			cols = append(cols, x)
		}
	}
	return cols
}

func TestDragPreviewDivider(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "note", "hello"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")

	before := dividers(h.view(), 5)
	if len(before) == 0 {
		t.Fatalf("no pane divider on screen:\n%s", h.view())
	}
	div := before[len(before)-1]
	h.send(
		tea.MouseMsg{X: div, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: div - 10, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: div - 10, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
	after := dividers(h.view(), 5)
	if len(after) != len(before) || after[len(after)-1] != div-10 {
		t.Fatalf("divider at %v after drag, want last at %d (was %v)", after, div-10, before)
	}

	// Motion after the release no longer moves it.
	h.send(tea.MouseMsg{X: div, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
	if got := dividers(h.view(), 5); got[len(got)-1] != div-10 {
		t.Errorf("divider moved after release: %v", got)
	}
}

func TestDragBlockInPreview(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "note", "First\n\nSecond\n\nThird"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")

	// Find the preview column where "Third" is drawn and its row.
	lines := strings.Split(ansi.Strip(h.view()), "\n")
	tx, ty := -1, -1
	for y, l := range lines {
		if i := strings.LastIndex(l, "Third"); i >= 0 {
			tx, ty = len([]rune(l[:i])), y
		}
	}
	fy := -1
	for y, l := range lines {
		if strings.Contains(l, "First") && strings.LastIndex(l, "First") > strings.LastIndex(l, "│") {
			fy = y
		}
	}
	if tx < 0 || fy < 0 {
		t.Fatalf("note not shown in the preview:\n%s", strings.Join(lines, "\n"))
	}
	grip := tx - 2
	h.send(
		tea.MouseMsg{X: tx, Y: ty, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: grip, Y: ty, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: grip, Y: fy, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: grip, Y: fy, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
	v := ansi.Strip(h.view())
	if a, b := strings.Index(v, "Third"), strings.Index(v, "First"); a < 0 || b < 0 || a > b {
		t.Fatalf("Third should now come before First:\n%s", v)
	}
}
