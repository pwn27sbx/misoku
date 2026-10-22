package app_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// tabline is the first row of the frame: one segment per open tab.
func (h *harness) tabline() string {
	h.t.Helper()
	return strings.Split(h.view(), "\n")[0]
}

// tabDirty reports whether the tab named name shows the unsaved mark: the
// mark takes the place of the close button, right after the name.
func (h *harness) tabDirty(name string) bool {
	h.t.Helper()
	row := h.tabline()
	switch {
	case strings.Contains(row, name+"  ●"):
		return true
	case strings.Contains(row, name+"  󰅖"):
		return false
	}
	h.t.Fatalf("no tab %q in %q", name, row)
	return false
}

// dirty reports whether the frame shows an unsaved mark anywhere: on a tab,
// in the status line or in the explorer.
func (h *harness) dirty() bool {
	h.t.Helper()
	return strings.Contains(h.view(), "●")
}

// words is the word count the status line shows, as "N words".
func (h *harness) words() string {
	h.t.Helper()
	for _, l := range strings.Split(h.view(), "\n") {
		if i := strings.Index(l, "󰈭 "); i >= 0 {
			f := strings.Fields(l[i+len("󰈭 "):])
			return f[0] + " " + f[1]
		}
	}
	h.t.Fatalf("no word count in the frame:\n%s", h.view())
	return ""
}

// paste sends s as one bracketed paste, as a terminal does.
func (h *harness) paste(s string) {
	h.t.Helper()
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Paste: true})
}

// openTwo opens alpha and then beta, each in its own tab, beta active.
func openTwo(t *testing.T, alpha, beta string) *harness {
	t.Helper()
	store := newTestStore(t)
	for n, text := range map[string]string{"alpha": alpha, "beta": beta} {
		if _, err := store.SaveAs("", n, text); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	for _, n := range []string{"alpha", "beta"} {
		h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
		h.keys(n)
		h.send(tea.KeyMsg{Type: tea.KeyCtrlT}) // in a new tab
	}
	if !strings.Contains(h.tabline(), "alpha.md") || !strings.Contains(h.tabline(), "beta.md") {
		t.Fatalf("expected two tabs: %q", h.tabline())
	}
	return h
}

func TestDirtyMarkAppearsOnEditAndClearsOnCtrlS(t *testing.T) {
	h := openWith(t, "one two")
	if h.dirty() {
		t.Fatalf("a note just opened has nothing unsaved:\n%s", h.view())
	}
	h.keys("A three")
	if !h.dirty() || !h.tabDirty("views.md") {
		t.Fatalf("typing should show the unsaved mark:\n%s", h.view())
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if h.dirty() {
		t.Fatalf("Ctrl+S should clear the unsaved mark at once:\n%s", h.view())
	}
	h.keys("!")
	if !h.dirty() {
		t.Fatalf("the next edit after a save should show the mark again:\n%s", h.view())
	}
}

func TestDirtyMarkClearsOnColonW(t *testing.T) {
	h := openWith(t, "one two")
	h.keys("A three")
	h.key(tea.KeyEsc)
	if !h.dirty() {
		t.Fatalf("typing should show the unsaved mark:\n%s", h.view())
	}
	h.keys(":w")
	h.key(tea.KeyEnter)
	if h.dirty() {
		t.Fatalf(":w should clear the unsaved mark at once:\n%s", h.view())
	}
	if got := readNote(t, h.store.Root(), "views.md"); got != "one two three" {
		t.Fatalf(":w should write the note, got %q", got)
	}
	h.keys("x")
	if !h.dirty() {
		t.Fatalf("the next edit after :w should show the mark again:\n%s", h.view())
	}
}

// Misoku has no undo: deleting what was typed is how the text goes back
// to what is saved, and then nothing is unsaved.
func TestDeletingTheEditBackToTheSavedTextClearsTheMark(t *testing.T) {
	h := openWith(t, "one two")
	h.keys("A!")
	if !h.dirty() {
		t.Fatalf("typing should show the unsaved mark:\n%s", h.view())
	}
	h.key(tea.KeyBackspace)
	if h.dirty() {
		t.Fatalf("back to the saved text, the mark should go:\n%s", h.view())
	}
}

func TestDirtyMarkFollowsEachTabWhenSwitching(t *testing.T) {
	h := openTwo(t, "alpha body", "beta body")
	h.keys("A!") // edit beta
	h.key(tea.KeyEsc)
	h.keys("[") // to alpha
	if h.tabDirty("alpha.md") || !h.tabDirty("beta.md") {
		t.Fatalf("only beta was edited: %q", h.tabline())
	}
	if strings.Contains(h.view(), "alpha.md ●") {
		t.Fatalf("the status line should not mark alpha:\n%s", h.view())
	}
	h.keys("A?") // edit alpha, then save it alone
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	h.key(tea.KeyEsc)
	if h.tabDirty("alpha.md") || !h.tabDirty("beta.md") {
		t.Fatalf("saving alpha should leave beta unsaved: %q", h.tabline())
	}
	h.keys("]") // back to beta
	if !h.tabDirty("beta.md") || h.tabDirty("alpha.md") {
		t.Fatalf("beta still has its edit: %q", h.tabline())
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if h.dirty() {
		t.Fatalf("both saved, nothing is unsaved:\n%s", h.view())
	}
}

// A tab that is not showing shows the state of its latest text, also after
// it was left, edited back and left again.
func TestHiddenTabShowsItsLatestEdit(t *testing.T) {
	h := openTwo(t, "alpha body", "beta body")
	h.keys("A!") // beta edited
	h.key(tea.KeyEsc)
	h.keys("[")
	if !h.tabDirty("beta.md") {
		t.Fatalf("beta left with an edit: %q", h.tabline())
	}
	h.keys("]A")
	h.key(tea.KeyBackspace) // beta back to its saved text
	h.key(tea.KeyEsc)
	h.keys("[")
	if h.tabDirty("beta.md") {
		t.Fatalf("beta left with its saved text: %q", h.tabline())
	}
	h.keys("]A?") // and edited once more
	h.key(tea.KeyEsc)
	h.keys("[")
	if !h.tabDirty("beta.md") {
		t.Fatalf("beta left with a new edit: %q", h.tabline())
	}
}

func TestReloadFromDiskUpdatesMarkAndWordCount(t *testing.T) {
	h := openWith(t, "one two")
	if got := h.words(); got != "2 words" {
		t.Fatalf("word count = %q, want 2 words", got)
	}
	writeOutside(t, h.store.Root(), "views.md", "one two three four")
	h.send(app.CheckWorkspaceMsg{})
	if got := h.words(); got != "4 words" {
		t.Fatalf("after the reload the word count = %q, want 4 words", got)
	}
	if h.dirty() {
		t.Fatalf("a reloaded note has nothing unsaved:\n%s", h.view())
	}
	h.keys("A five")
	if !h.dirty() || h.words() != "5 words" {
		t.Fatalf("an edit after the reload should count and mark:\n%s", h.view())
	}
}

// A note in a tab that is not showing is reloaded too; showing it then
// shows the disk's text, counted, and nothing unsaved.
func TestReloadOfAHiddenTabUpdatesMarkAndWordCount(t *testing.T) {
	h := openTwo(t, "alpha body", "beta body")
	h.keys("[") // alpha
	h.keys("A!")
	h.key(tea.KeyBackspace) // edited, then back to the saved text
	h.key(tea.KeyEsc)
	h.keys("]") // beta
	writeOutside(t, h.store.Root(), "alpha.md", "alpha body from outside now")
	h.send(app.CheckWorkspaceMsg{})
	if h.tabDirty("alpha.md") {
		t.Fatalf("alpha was reloaded, not edited: %q", h.tabline())
	}
	h.keys("[")
	if got := h.words(); got != "5 words" {
		t.Fatalf("alpha's word count = %q, want 5 words", got)
	}
	if h.dirty() {
		t.Fatalf("nothing is unsaved:\n%s", h.view())
	}
}

func TestReopeningFromTheFinderUpdatesMarkAndWordCount(t *testing.T) {
	store := newTestStore(t)
	for n, text := range map[string]string{"alpha": "a b c", "beta": "one two three four five"} {
		if _, err := store.SaveAs("", n, text); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("alpha")
	h.key(tea.KeyEnter)
	if got := h.words(); got != "3 words" {
		t.Fatalf("alpha's word count = %q, want 3 words", got)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP}) // the clean tab is reused
	h.keys("beta")
	h.key(tea.KeyEnter)
	if strings.Contains(h.tabline(), "alpha.md") {
		t.Fatalf("beta should replace alpha in the same tab: %q", h.tabline())
	}
	if got := h.words(); got != "5 words" || h.dirty() {
		t.Fatalf("beta: words %q, want 5 words and nothing unsaved:\n%s", got, h.view())
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("alpha")
	h.key(tea.KeyEnter)
	if got := h.words(); got != "3 words" || h.dirty() {
		t.Fatalf("alpha again: words %q, want 3 words and nothing unsaved:\n%s", got, h.view())
	}
}

func TestWordCountStaysExact(t *testing.T) {
	h := openTwo(t, "alpha body", "one")
	if got := h.words(); got != "1 word" {
		t.Fatalf("word count = %q, want 1 word", got)
	}
	h.keys("A two")
	if got := h.words(); got != "2 words" {
		t.Fatalf("after typing: %q, want 2 words", got)
	}
	h.paste(" three four\nfive\tsix")
	if got := h.words(); got != "6 words" {
		t.Fatalf("after a paste: %q, want 6 words", got)
	}
	for range len("six") {
		h.key(tea.KeyBackspace)
	}
	if got := h.words(); got != "5 words" {
		t.Fatalf("after deleting a word: %q, want 5 words", got)
	}
	h.key(tea.KeyEsc)
	h.keys("[")
	if got := h.words(); got != "2 words" {
		t.Fatalf("alpha's tab: %q, want 2 words", got)
	}
	h.keys("]")
	if got := h.words(); got != "5 words" {
		t.Fatalf("back on beta: %q, want 5 words", got)
	}
}

// Redrawing and keys that only move the cursor or the view leave the
// editor's text alone: none of them copies the whole note.
func TestCursorKeysAndRedrawsDoNotCopyTheText(t *testing.T) {
	var b strings.Builder
	for i := range 5000 {
		fmt.Fprintf(&b, "line %d with some words\n", i)
	}
	h := openWith(t, b.String())
	h.view()
	before := app.TextCopies(h.m)
	for range 5 {
		h.keys("j")
		h.view()
	}
	h.keys("k")
	h.send(tea.MouseMsg{X: 30, Y: bodyRow, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	h.view()
	h.view()
	if got := app.TextCopies(h.m) - before; got != 0 {
		t.Fatalf("cursor keys and redraws copied the note %d times", got)
	}
}

// A draft has no file: it is unsaved as soon as it has any text.
func TestDraftIsUnsavedOnceItHasText(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlN})
	if h.tabDirty("Untitled-1 new") {
		t.Fatalf("an empty draft has nothing unsaved: %q", h.tabline())
	}
	h.keys("hi")
	if !h.tabDirty("Untitled-1 new") {
		t.Fatalf("a draft with text is unsaved: %q", h.tabline())
	}
	h.key(tea.KeyBackspace)
	h.key(tea.KeyBackspace)
	if h.tabDirty("Untitled-1 new") {
		t.Fatalf("an emptied draft has nothing unsaved: %q", h.tabline())
	}
}

// The explorer marks the notes with unsaved edits, and drops the mark on
// save.
func TestExplorerMarksUnsavedNotes(t *testing.T) {
	h := openWith(t, "one two")
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.keys(" e") // the explorer
	if v := h.view(); !strings.Contains(v, "views ●") && !strings.Contains(v, "views.md ●") {
		t.Fatalf("the explorer should mark the edited note:\n%s", v)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if h.dirty() {
		t.Fatalf("after saving nothing is marked:\n%s", h.view())
	}
}

// The count is strings.Fields' count, also for Unicode spaces and bytes that
// are not UTF-8.
func TestCountWordsMatchesFields(t *testing.T) {
	for _, s := range []string{"", " ", "one", " one  two ", "a\tb\nc\r\nd", "x y z", "日本 語", "\xff\xfe a", "\n\n"} {
		if got, want := app.CountWords(s), len(strings.Fields(s)); got != want {
			t.Errorf("CountWords(%q) = %d, want %d", s, got, want)
		}
	}
}

// A draft edited and left with no frame drawn in between is marked once it
// is hidden, and the quit prompt counts it: the text version must be stashed
// with the text, or the draft's cached state from before the edit is reused.
func TestHiddenDraftEditedOffFrameIsMarked(t *testing.T) {
	h := openTwo(t, "alpha body", "beta body")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlN})
	h.key(tea.KeyEsc)
	h.keys("[")
	if strings.Contains(h.tabline(), "●") {
		t.Fatalf("nothing edited: %q", h.tabline())
	}
	h.keys("]ihi") // no frame between here and the next view
	h.key(tea.KeyEsc)
	h.keys("[")
	if !strings.Contains(h.tabline(), "●") {
		t.Fatalf("the hidden draft has text: %q", h.tabline())
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	if h.quit {
		t.Fatal("quit with an unsaved draft must ask first")
	}
	if v := h.view(); !strings.Contains(v, "1 unsaved: Untitled-1") {
		t.Fatalf("the quit prompt should list the draft:\n%s", v)
	}
}
