package app_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// longNote is head followed by enough filler lines that edits to it refresh
// the preview after a pause instead of on every key.
func longNote(head string) string {
	var b strings.Builder
	b.WriteString(head)
	for i := range 1100 {
		fmt.Fprintf(&b, "\n- filler %d", i) // one block per line
	}
	return b.String()
}

// preview returns the preview pane of the current frame.
func (h *harness) preview() string {
	h.t.Helper()
	return previewPane(h.view())
}

// typeAtTop types s in INSERT mode at the start of the note, then leaves
// INSERT mode.
func (h *harness) typeAtTop(s string) {
	h.t.Helper()
	h.keys("ggi")
	h.keys(s)
	h.key(tea.KeyEsc)
}

// hoverPreview moves the mouse over the preview without pressing a button.
func (h *harness) hoverPreview() {
	h.t.Helper()
	h.send(tea.MouseMsg{X: termW - 5, Y: bodyRow, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
}

func TestPreviewOfShortNoteUpdatesOnEveryKey(t *testing.T) {
	h := openWith(t, "# Title\n\nbody")
	h.typeAtTop("zebra ")
	if p := h.preview(); !strings.Contains(p, "zebra") {
		t.Fatalf("a short note's preview should follow every key:\n%s", p)
	}
}

func TestPreviewOfLongNoteWaitsForTheRefresh(t *testing.T) {
	h := openWith(t, longNote("# Title"))
	before := app.PreviewCompiles(h.m)
	h.typeAtTop("zebra ")
	if p := h.preview(); strings.Contains(p, "zebra") {
		t.Fatalf("a long note's preview should wait for the pause:\n%s", p)
	}
	if got := app.PreviewCompiles(h.m); got != before {
		t.Fatalf("typing should not recompile a long note's preview: %d recompiles", got-before)
	}
	h.send(app.PreviewRefreshMsg{})
	if p := h.preview(); !strings.Contains(p, "zebra") {
		t.Fatalf("the refresh should bring the preview up to date:\n%s", p)
	}
	if got := app.PreviewCompiles(h.m); got != before+1 {
		t.Fatalf("six keys should cost one recompile, got %d", got-before)
	}
}

func TestPreviewRefreshIgnoresAnOlderTimer(t *testing.T) {
	h := openWith(t, longNote("# Title"))
	h.typeAtTop("ab")
	gen := app.PreviewGen(h.m)
	if gen < 2 {
		t.Fatalf("two edits should schedule two refreshes, generation %d", gen)
	}
	h.send(app.PreviewTick(gen - 1))
	if p := h.preview(); strings.Contains(p, "ab# Title") {
		t.Fatalf("a timer from an earlier key should not refresh:\n%s", p)
	}
	h.send(app.PreviewTick(gen))
	if p := h.preview(); !strings.Contains(p, "ab# Title") {
		t.Fatalf("the latest timer should refresh:\n%s", p)
	}
}

func TestMovingTheCursorDoesNotRecompileThePreview(t *testing.T) {
	for _, tc := range []struct{ name, text string }{
		{"short", "one\ntwo\nthree\nfour"},
		{"long", longNote("one\ntwo")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := openWith(t, tc.text)
			before := app.PreviewCompiles(h.m)
			h.keys("jjjkk")
			h.send(tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyUp})
			h.send(app.PreviewRefreshMsg{})
			if got := app.PreviewCompiles(h.m); got != before {
				t.Fatalf("moving the cursor recompiled the preview %d times", got-before)
			}
		})
	}
}

func TestLiveGraphSkipsCursorMovement(t *testing.T) {
	h := graphWorkspace(t)
	h.clickGraph("Untitled-1")
	h.clickGraph("Untitled-1") // opens Untitled-1; the graph tab stays
	before := app.GraphLinkScans(h.m)
	h.keys("jkjk")
	if got := app.GraphLinkScans(h.m); got != before {
		t.Fatalf("moving the cursor re-read the note's links %d times", got-before)
	}
	h.keys("A [o](Other.md)") // "[[" would open the link picker
	h.key(tea.KeyEsc)
	if got := app.GraphLinkScans(h.m); got == before {
		t.Fatal("typing a link should re-read the note's links")
	}
	h.keys(" g") // the graph shows the link typed, before any save
	if ed := editorPane(h.view()); !strings.Contains(ed, "Other") {
		t.Fatalf("the live graph should show the typed link to Other:\n%s", ed)
	}
}

// formNote is a long note that starts with a form with a checkbox question.
var formNote = longNote("```form\ntitle: Survey\n? Pick (checkbox): A | B\n```")

func TestCheckboxRightAfterTypingTicksTheRightLine(t *testing.T) {
	h := openWith(t, formNote)
	h.keys("ggi")
	h.keys("intro")
	h.key(tea.KeyEnter) // every line of the form moves down one
	h.key(tea.KeyEsc)
	h.hoverPreview()
	h.clickText("☐ B")
	if got := h.text(); !strings.Contains(got, "intro\n```form\ntitle: Survey\n? Pick (checkbox): A | B\n  = B\n```") {
		t.Fatalf("ticking B should answer the question it belongs to:\n%s", got[:min(len(got), 200)])
	}
}

func TestWheelOverPreviewShowsTheLatestText(t *testing.T) {
	h := openWith(t, longNote("# Title"))
	h.typeAtTop("zebra ")
	h.send(tea.MouseMsg{X: termW - 5, Y: bodyRow, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	if p := h.preview(); !strings.Contains(p, "zebra") {
		t.Fatalf("the wheel over the preview should show the latest text:\n%s", p)
	}
}

func TestHoverOverPreviewShowsTheLatestText(t *testing.T) {
	h := openWith(t, longNote("# Title"))
	h.typeAtTop("zebra ")
	h.hoverPreview()
	if p := h.preview(); !strings.Contains(p, "zebra") {
		t.Fatalf("hovering the preview should show the latest text:\n%s", p)
	}
}

func TestSaveWithPendingRefreshShowsTheLatestText(t *testing.T) {
	h := openWith(t, longNote("# Title"))
	h.typeAtTop("zebra ")
	if got := h.text(); !strings.HasPrefix(got, "zebra # Title") {
		t.Fatalf("save should write what was typed:\n%s", got[:min(len(got), 80)])
	}
	if p := h.preview(); !strings.Contains(p, "zebra") {
		t.Fatalf("save should bring the preview up to date:\n%s", p)
	}
	h.typeAtTop("yak ")
	h.keys(":w")
	h.key(tea.KeyEnter)
	if p := h.preview(); !strings.Contains(p, "yak") {
		t.Fatalf(":w should bring the preview up to date:\n%s", p)
	}
}

func TestQuitWithPendingRefreshSavesTheLatestText(t *testing.T) {
	h := openWith(t, longNote("# Title"))
	h.typeAtTop("zebra ")
	h.keys(":wq")
	h.key(tea.KeyEnter)
	if !h.quit {
		t.Fatal(":wq should quit")
	}
	p, err := h.store.Get("views.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.Content, "zebra # Title") {
		t.Fatalf(":wq should write what was typed:\n%s", p.Content[:min(len(p.Content), 80)])
	}
}

func TestSwitchingTabsWithPendingRefreshKeepsEachPreview(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "alpha", longNote("# Alpha")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "beta", "# Beta\n\nbeta body"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("alpha")
	h.key(tea.KeyEnter)
	h.typeAtTop("x") // a note with unsaved edits keeps its tab
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("beta")
	h.key(tea.KeyEnter)
	h.clickTab("alpha.md")
	h.typeAtTop("zebra ")

	h.clickTab("beta.md") // a click on the tabs is not over the preview
	before := app.PreviewCompiles(h.m)
	h.send(app.PreviewRefreshMsg{})
	if p := h.preview(); !strings.Contains(p, "beta body") || strings.Contains(p, "zebra") {
		t.Fatalf("a refresh due for alpha should leave beta's preview alone:\n%s", p)
	}
	if got := app.PreviewCompiles(h.m); got != before {
		t.Fatalf("a refresh due for a hidden tab should not recompile, got %d", got-before)
	}
	h.clickTab("alpha.md")
	if p := h.preview(); !strings.Contains(p, "zebra") {
		t.Fatalf("back on alpha, the preview should show what was typed:\n%s", p)
	}

	// The graph tab shows the note selected in the graph, not the editor.
	h.keys(":graph")
	h.key(tea.KeyEnter)
	h.clickTab("alpha.md")
	h.typeAtTop("yak ")
	h.clickTab("Graph")
	h.send(app.PreviewRefreshMsg{})
	if p := h.preview(); !strings.Contains(p, "yak zebra") {
		t.Fatalf("a refresh due for alpha should leave the graph's preview alone:\n%s", p)
	}
}

// clickTab clicks a tab's name in the tabline.
func (h *harness) clickTab(name string) {
	h.t.Helper()
	x, y := find(h.t.(*testing.T), h.view(), name)
	if y != 0 {
		h.t.Fatalf("%q is not a tab:\n%s", name, h.view())
	}
	h.click(x, 0)
}

func TestHeadingJumpAfterTypingUsesTheLatestText(t *testing.T) {
	h := openWith(t, longNote("[[#Zeta]]"))
	h.keys("Go## Zeta")
	h.key(tea.KeyEsc)
	h.keys("ggl")
	h.keys("gd")
	if p := h.preview(); !strings.Contains(p, "Zeta") || strings.Contains(p, "filler 0 ") {
		t.Fatalf("gd should show the heading typed at the end:\n%s", p)
	}
}

func TestGraphTabClickedRightAfterTypingShowsTheLink(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "Big", longNote("big")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "Other", "other"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("big")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.keys(":graph")
	h.key(tea.KeyEnter)
	h.clickTab("Big.md")
	h.typeAtTop("[o](Other.md) ")
	h.clickTab("Graph") // before the debounce fires
	h.send(app.PreviewRefreshMsg{})
	if ed := editorPane(h.view()); !strings.Contains(ed, "1 link") {
		t.Fatalf("the graph should show the link typed before switching:\n%s", ed)
	}
}
