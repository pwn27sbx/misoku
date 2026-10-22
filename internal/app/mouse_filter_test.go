package app_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// sendFiltered delivers msgs the way the runtime does with
// tea.WithFilter(app.MouseFilter): what the filter drops never reaches
// Update.
func (h *harness) sendFiltered(msgs ...tea.Msg) {
	h.t.Helper()
	for _, msg := range msgs {
		if out := app.MouseFilter(h.m, msg); out != nil {
			h.send(out)
		}
	}
}

// passes reports whether the filter lets msg through to Update.
func (h *harness) passes(msg tea.Msg) bool {
	h.t.Helper()
	return app.MouseFilter(h.m, msg) != nil
}

func motionAt(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion}
}

func leftAt(x, y int, a tea.MouseAction) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: a}
}

// blocksNote opens a note of three paragraphs and returns the editor's
// first column and the last divider (the preview starts right after it).
func blocksNote(t *testing.T) (h *harness, editorX, previewDiv int) {
	t.Helper()
	h = openWith(t, "First\n\nSecond\n\nThird")
	div := dividers(h.view(), bodyRow)
	if len(div) < 2 {
		t.Fatalf("want sidebar, editor and preview:\n%s", h.view())
	}
	return h, div[0] + 1, div[len(div)-1]
}

// previewBlock returns where "text" is drawn in the preview pane.
func previewBlock(t *testing.T, h *harness, text string) (int, int) {
	t.Helper()
	for y, l := range strings.Split(h.view(), "\n") {
		bar := strings.LastIndex(l, "│")
		if i := strings.LastIndex(l, text); bar >= 0 && i > bar {
			return ansi.StringWidth(l[:i]), y
		}
	}
	t.Fatalf("%q not in the preview:\n%s", text, h.view())
	return 0, 0
}

// handles reports whether the preview draws a block's drag handle.
func handles(h *harness) bool { return strings.Contains(previewPane(h.view()), "⠿") }

func TestMouseFilterDropsIdleMotion(t *testing.T) {
	h, editorX, _ := blocksNote(t)
	for _, c := range []struct {
		name string
		x, y int
	}{
		{"editor", editorX + 3, bodyRow},
		{"sidebar", 2, bodyRow},
		{"tabline", 10, 0},
		{"footer", 10, termH - 1},
	} {
		if h.passes(motionAt(c.x, c.y)) {
			t.Errorf("motion over the %s with nothing to drag or clear should be dropped", c.name)
		}
	}
}

func TestMouseFilterPassesEditorDragSelect(t *testing.T) {
	h, editorX, _ := blocksNote(t)
	h.sendFiltered(leftAt(editorX+2, bodyRow, tea.MouseActionPress))
	if !h.passes(leftAt(editorX+6, bodyRow+1, tea.MouseActionMotion)) {
		t.Fatal("motion while selecting in the editor should pass")
	}
	// The drag keeps going wherever the mouse goes, even over the sidebar.
	if !h.passes(leftAt(2, bodyRow, tea.MouseActionMotion)) {
		t.Fatal("motion while selecting should pass outside the editor too")
	}
	h.sendFiltered(leftAt(editorX+6, bodyRow, tea.MouseActionRelease))
	if h.passes(motionAt(editorX+7, bodyRow)) {
		t.Fatal("motion after the release should be dropped again")
	}
}

func TestMouseFilterPassesDividerDrag(t *testing.T) {
	h, editorX, div := blocksNote(t)
	h.sendFiltered(leftAt(div, bodyRow, tea.MouseActionPress))
	if !h.passes(leftAt(editorX+5, bodyRow, tea.MouseActionMotion)) {
		t.Fatal("motion while dragging a divider should pass")
	}
}

func TestMouseFilterPassesPreviewBlockDrag(t *testing.T) {
	h, editorX, _ := blocksNote(t)
	x, y := previewBlock(t, h, "Third")
	h.sendFiltered(motionAt(x, y), leftAt(x-2, y, tea.MouseActionPress))
	// A block drag keeps going outside the preview.
	if !h.passes(leftAt(editorX+3, bodyRow, tea.MouseActionMotion)) {
		t.Fatal("motion while dragging a preview block should pass")
	}
}

func TestMouseFilterPassesMotionOverPreview(t *testing.T) {
	h, _, div := blocksNote(t)
	// Over the preview the hover follows the mouse, even where no block is.
	for _, y := range []int{bodyRow, termH - 3} {
		if !h.passes(motionAt(div+5, y)) {
			t.Errorf("motion over the preview at row %d should pass", y)
		}
	}
}

func TestMouseFilterClearsHoverOnceWhenLeavingThePreview(t *testing.T) {
	h, editorX, _ := blocksNote(t)
	x, y := previewBlock(t, h, "Second")
	h.sendFiltered(motionAt(x, y))
	if !handles(h) {
		t.Fatalf("hovering a block should show its handles:\n%s", h.view())
	}
	if !h.passes(motionAt(editorX+3, y)) {
		t.Fatal("the first motion leaving the preview should pass to clear the hover")
	}
	h.sendFiltered(motionAt(editorX+3, y))
	if handles(h) {
		t.Fatalf("leaving the preview should hide the handles:\n%s", h.view())
	}
	if h.passes(motionAt(editorX+4, y)) {
		t.Fatal("with the hover cleared, motion over the editor should be dropped")
	}
}

func TestMouseFilterPassesEverythingButMotion(t *testing.T) {
	h, editorX, _ := blocksNote(t)
	type other struct{ n int }
	for _, msg := range []tea.Msg{
		leftAt(editorX+3, bodyRow, tea.MouseActionPress),
		leftAt(editorX+3, bodyRow, tea.MouseActionRelease),
		tea.MouseMsg{X: editorX + 3, Y: bodyRow, Button: tea.MouseButtonRight, Action: tea.MouseActionPress},
		tea.MouseMsg{X: editorX + 3, Y: bodyRow, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress},
		tea.MouseMsg{X: 2, Y: 0, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
		tea.WindowSizeMsg{Width: termW, Height: termH},
		other{7},
		nil,
	} {
		if got := app.MouseFilter(h.m, msg); !reflect.DeepEqual(got, msg) {
			t.Errorf("MouseFilter(%#v) = %#v, want it unchanged", msg, got)
		}
	}
}

func TestMouseFilterLeavesTheModelAlone(t *testing.T) {
	h, editorX, _ := blocksNote(t)
	x, y := previewBlock(t, h, "Second")
	h.sendFiltered(motionAt(x, y))
	before := h.view()
	for _, msg := range []tea.Msg{motionAt(editorX+3, y), motionAt(x, y+2), motionAt(2, 0)} {
		app.MouseFilter(h.m, msg)
	}
	if after := h.view(); after != before {
		t.Fatalf("the filter changed the screen:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// The mouse features that need motion still work when it goes through the
// filter, as it does in the real program.

func TestDragSelectThroughFilterCopies(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "hello world\nsecond line"); err != nil {
		t.Fatal(err)
	}
	var copied []string
	h := newHarness(t, store, app.WithClipboard(func(s string) error { copied = append(copied, s); return nil }))
	h.keys("1")
	x, y := find(t, h.view(), "world")
	h.sendFiltered(
		leftAt(x, y, tea.MouseActionPress),
		leftAt(x+4, y, tea.MouseActionMotion),
		leftAt(x+4, y, tea.MouseActionRelease),
	)
	if len(copied) == 0 || copied[len(copied)-1] != "world" {
		t.Fatalf("drag through the filter copied %q, want \"world\"", copied)
	}
}

func TestDividerDragThroughFilterMoves(t *testing.T) {
	h, _, div := blocksNote(t)
	h.sendFiltered(
		leftAt(div, bodyRow, tea.MouseActionPress),
		leftAt(div-10, bodyRow, tea.MouseActionMotion),
		leftAt(div-10, bodyRow, tea.MouseActionRelease),
	)
	if got := dividers(h.view(), bodyRow); got[len(got)-1] != div-10 {
		t.Fatalf("dividers at %v after the drag, want the last at %d", got, div-10)
	}
}

func TestBlockDragThroughFilterMoves(t *testing.T) {
	h, _, _ := blocksNote(t)
	tx, ty := previewBlock(t, h, "Third")
	_, fy := previewBlock(t, h, "First")
	h.sendFiltered(
		motionAt(tx, ty),
		leftAt(tx-2, ty, tea.MouseActionPress),
		leftAt(tx-2, fy, tea.MouseActionMotion),
		leftAt(tx-2, fy, tea.MouseActionRelease),
	)
	v := h.view()
	if a, b := strings.Index(v, "Third"), strings.Index(v, "First"); a < 0 || b < 0 || a > b {
		t.Fatalf("Third should now come before First:\n%s", v)
	}
}

func TestHoverHandlesThroughFilterComeAndGo(t *testing.T) {
	h, editorX, div := blocksNote(t)
	x, y := previewBlock(t, h, "Second")
	h.sendFiltered(motionAt(x, y))
	if !handles(h) {
		t.Fatalf("hovering a block should show its handles:\n%s", h.view())
	}
	// Leaving through the divider and then over the editor hides them.
	h.sendFiltered(motionAt(div, y), motionAt(editorX+3, y))
	if handles(h) {
		t.Fatalf("leaving the preview should hide the handles:\n%s", h.view())
	}
}

func TestGraphPanThroughFilter(t *testing.T) {
	h := graphWorkspace(t)
	before := editorPane(h.view())
	x := 45 // a blank spot in the graph, above the map
	h.sendFiltered(
		leftAt(x, 3, tea.MouseActionPress),
		leftAt(x+8, 5, tea.MouseActionMotion),
		leftAt(x+8, 5, tea.MouseActionRelease),
	)
	if editorPane(h.view()) == before {
		t.Fatal("dragging through the filter should pan the map")
	}
}

func TestMouseFilterDropsMotionOnTheDashboard(t *testing.T) {
	h := newHarness(t, newTestStore(t)) // a new app opens on the dashboard
	for _, x := range []int{5, termW / 2, termW - 5} {
		if h.passes(motionAt(x, bodyRow)) {
			t.Errorf("motion at column %d of the dashboard should be dropped: it only reacts to clicks", x)
		}
	}
}
