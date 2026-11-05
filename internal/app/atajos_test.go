package app_test

// The LazyVim companion's Misoku keys (space o …), the three-key leader they
// need, and Enter on a [[link]].

import (
	"context"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// openTodo opens a note holding text, in NORMAL mode.
func openTodo(t *testing.T, text string, opts ...app.Option) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "todo", text); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, opts...)
	h.keys("1")
	h.key(tea.KeyEsc)
	return h
}

func TestSpaceOCTogglesTheLineAndRefreshes(t *testing.T) {
	h := openTodo(t, "- [ ] task\nnext")
	compiles := app.PreviewCompiles(h.m)
	h.keys(" oc")
	if got := h.editor(); got != "- [x] task\nnext" {
		t.Fatalf("space o c should check the box, got %q", got)
	}
	if !h.dirty() {
		t.Fatalf("the modified mark should show at once:\n%s", h.view())
	}
	if app.PreviewCompiles(h.m) == compiles {
		t.Fatal("the preview should follow the toggle at once")
	}
	h.key(tea.KeyCtrlS)
	if got := h.disk("todo.md"); got != "- [x] task\nnext" {
		t.Fatalf("saved %q", got)
	}
	h.keys("u")
	if got := h.editor(); got != "- [ ] task\nnext" {
		t.Fatalf("one u should undo the toggle, got %q", got)
	}
}

func TestSpaceOCOnAVLineRangeIsOneChange(t *testing.T) {
	h := openTodo(t, "- [ ] a\n- [x] b\nc")
	h.keys("Vj oc")
	if got := h.editor(); got != "- [x] a\n- [ ] b\nc" {
		t.Fatalf("space o c on V j should toggle both lines, got %q", got)
	}
	if !strings.Contains(h.view(), "V-LINE") {
		t.Fatalf("the selection should stay:\n%s", h.view())
	}
	// u does nothing in VISUAL: leave it first.
	h.key(tea.KeyEsc)
	h.keys("u")
	if got := h.editor(); got != "- [ ] a\n- [x] b\nc" {
		t.Fatalf("one u should restore both lines, got %q", got)
	}
}

func TestSpaceOShiftCRemovesTheBox(t *testing.T) {
	h := openTodo(t, "- [x] task\nplain")
	h.keys(" oC")
	if got := h.editor(); got != "- task\nplain" {
		t.Fatalf("space o C should remove the box, got %q", got)
	}
	h.keys("j oC")
	if got := h.editor(); got != "- task\nplain" {
		t.Fatalf("a line without a box stays, got %q", got)
	}
	h.keys("gg")
	h.keys("vj oC") // nothing left to remove: a v selection reaches whole lines
	h.key(tea.KeyEsc)
	h.keys("gg oc oc oC")
	if got := h.editor(); got != "- task\nplain" {
		t.Fatalf("toggle twice then remove, got %q", got)
	}
}

// Only the checkbox keys work on a selection, as in the LazyVim companion:
// the other leader keys and the general keys leave it alone.
func TestVisualOnlyTakesTheCheckboxLeaderKeys(t *testing.T) {
	h := openNotes(t, "alpha", "beta")
	before := h.tabline()
	h.keys("V")
	h.keys("]")
	if got := h.tabline(); got != before {
		t.Fatalf("] in V-LINE should not change the tab:\nbefore %q\nafter  %q", before, got)
	}
	h.keys(" x")
	if got := h.tabline(); got != before || !strings.Contains(h.view(), "V-LINE") {
		t.Fatalf("space x in V-LINE should not close the tab:\n%s", h.view())
	}
}

func TestSpaceGStillOpensTheGraph(t *testing.T) {
	h := openTodo(t, "hello")
	h.keys(" g")
	if !strings.Contains(h.tabline(), "Graph") {
		t.Fatalf("space g should open the graph:\n%s", h.view())
	}
}

func TestSpaceOWaitsAndShowsTheMisokuKeys(t *testing.T) {
	h := openTodo(t, "hello")
	h.keys(" ")
	tickGuide(h)
	v := h.view()
	if !guideRow(v, "o", "+misoku") {
		t.Fatalf("the first page should list o as +misoku:\n%s", v)
	}
	if got := guideGroup(t, v, "+misoku"); got != "NOTE" {
		t.Fatalf("+misoku is under %s, want NOTE:\n%s", got, v)
	}
	if strings.Contains(v, "Toggle checkbox") {
		t.Fatalf("the group's keys belong to its own page:\n%s", v)
	}
	h.keys("o")
	v = h.view()
	if !guideOpen(h) || !strings.Contains(v, "misoku") {
		t.Fatalf("space o should keep the guide on the misoku page:\n%s", v)
	}
	if !guideRow(v, "c", "Toggle checkbox") || !guideRow(v, "Shift+C", "Remove checkbox") || !guideRow(v, "f / g", "Find note") {
		t.Fatalf("the misoku page should list c, Shift+C and f / g:\n%s", v)
	}
	if strings.Contains(v, "Themes") {
		t.Fatalf("keys outside the group stay off its page:\n%s", v)
	}
	h.keys("z")
	v = h.view()
	if guideOpen(h) || !strings.Contains(v, "o z: not a leader key") {
		t.Fatalf("space o z should end and say so:\n%s", v)
	}
	if h.editor() != "hello" {
		t.Fatalf("nothing should change, got %q", h.editor())
	}
}

// Before the guide shows, space o still waits; its own timer brings the
// group's page.
func TestSpaceOBeforeTheGuideStartsItsOwnTimer(t *testing.T) {
	h := openTodo(t, "hello")
	h.keys(" ")
	seq := app.LeaderSeq(h.m)
	h.keys("o")
	if app.LeaderSeq(h.m) == seq {
		t.Fatal("space o should schedule the guide again")
	}
	tickGuide(h)
	if !guideRow(h.view(), "c", "Toggle checkbox") {
		t.Fatalf("the guide after space o should be the misoku page:\n%s", h.view())
	}
	h.key(tea.KeyEsc)
	if guideOpen(h) || strings.Contains(h.view(), "not a leader key") {
		t.Fatalf("Esc ends quietly:\n%s", h.view())
	}
}

func TestSpaceOFOpensTheFinder(t *testing.T) {
	h := openTodo(t, "hello")
	h.keys(" of")
	h.keys("todo")
	h.key(tea.KeyEnter)
	if h.editor() != "hello" || strings.Contains(h.view(), "not a leader key") {
		t.Fatalf("space o f should open the finder:\n%s", h.view())
	}
	h.keys(" on")
	if strings.Count(h.tabline(), "Untitled") != 1 {
		t.Fatalf("space o n should start a new note:\n%s", h.view())
	}
}

func TestHomeSpaceOCNeedsANote(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	h.keys(" oc")
	if v := h.view(); !strings.Contains(v, "o c: needs an open note") || !strings.Contains(v, "HOME") {
		t.Fatalf("space o c on home should say it needs an open note:\n%s", v)
	}
}

// space i and space o p paste the picture, as ctrl+v does in INSERT.
func TestSpaceIPastesThePicture(t *testing.T) {
	for _, keys := range []string{" i", " op"} {
		msg := pastedPNG(t)
		h := openTodo(t, "hola\n", app.WithClipboardImage(func(context.Context) (core.ClipboardImageMsg, bool, error) {
			return msg, true, nil
		}))
		h.keys(keys)
		h.key(tea.KeyCtrlS)
		if got := h.disk("todo.md"); !strings.Contains(got, "](assets/pasted-2026-11-01_10-00-00.png)") {
			t.Fatalf("%q should paste the picture, note is %q", keys, got)
		}
	}
}

// Enter on a [[link]] opens the note; a link that names no note says so and
// starts no draft.
func TestEnterFollowsAWikiLink(t *testing.T) {
	h := openStart(t, linkedNotes(t))
	h.key(tea.KeyEsc)
	h.keys("gg0" + strings.Repeat("l", 16)) // onto "[[Docker]]"
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "containers") {
		t.Fatalf("Enter should open Docker.md:\n%s", v)
	}
}

func TestEnterOnAMissingNoteSaysSo(t *testing.T) {
	h := openStart(t, linkedNotes(t))
	h.key(tea.KeyEsc)
	h.keys("Gkk0" + strings.Repeat("l", 13)) // onto "[[Fresh Idea]]"
	h.key(tea.KeyEnter)
	v := h.view()
	if !strings.Contains(v, "No note found for this link") {
		t.Fatalf("Enter on a missing note should say so:\n%s", v)
	}
	if strings.Contains(h.tabline(), "Fresh Idea") || !strings.Contains(v, "Learn about [[Docker]]") {
		t.Fatalf("Enter should start no draft:\n%s", v)
	}
}

// S6: space o ] and space o [ jump between [[links]], wrapping; with none the
// status says so.
func TestSpaceOBracketsJumpBetweenLinks(t *testing.T) {
	h := openTodo(t, "top [[One]]\nmid [[Two]]")
	h.keys(" o]x")
	if got := h.editor(); got != "top [One]]\nmid [[Two]]" {
		t.Fatalf("space o ] should land on the first link's [, got %q", got)
	}
	h.keys("u o[ o[x")
	if got := h.editor(); got != "top [[One]]\nmid [Two]]" {
		t.Fatalf("space o [ twice should wrap to the last link, got %q", got)
	}
}

func TestSpaceOBracketWithoutLinksSaysSo(t *testing.T) {
	h := openTodo(t, "no links\n`[[code]]`")
	h.keys(" o]")
	if !strings.Contains(h.view(), "No links in this note") {
		t.Fatalf("the status should say there are no links:\n%s", h.view())
	}
}
