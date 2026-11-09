package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// tagStore is a vault where only Zentro/alpha passes "status:active
// tag:go folder:Zentro docker".
func tagStore(t *testing.T) *core.Store {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][3]string{
		{"Zentro", "alpha", "---\nstatus: active\n---\nRun docker with #go/web"},
		{"Zentro", "bravo", "---\nstatus: onhold\n---\nRun docker with #go"},
		{"ZentroOld", "charlie", "---\nstatus: active\n---\nRun docker with #go"},
		{"Zentro", "delta", "---\nstatus: active\ntags: [python]\n---\nRun docker"},
		{"", "start", "home"},
	} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

// openStartNote opens start.md in normal mode.
func openStartNote(t *testing.T, store *core.Store, opts ...app.Option) *harness {
	t.Helper()
	h := newHarness(t, store, opts...)
	h.keys("f")
	h.keys("start")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	return h
}

// S2: the qualifiers typed in the finder AND together with the words.
func TestFinderQualifiersInTheFinder(t *testing.T) {
	h := newHarness(t, tagStore(t))
	h.keys("f")
	h.keys("status:active tag:go folder:Zentro docker")
	v := h.view()
	if !strings.Contains(v, "1 results") || !strings.Contains(v, "Zentro/alpha.md:4") {
		t.Fatalf("want one hit, alpha's docker line:\n%s", v)
	}
	h.key(tea.KeyEnter)
	if tabs := h.tabline(); !strings.Contains(tabs, "alpha.md") {
		t.Fatalf("Enter should open alpha.md: %q", tabs)
	}
}

// S2: the link picker is the same finder, qualifiers included.
func TestLinkPickerHonoursQualifiers(t *testing.T) {
	store := tagStore(t)
	h := openStartNote(t, store, app.WithUI(config.UI{LinkSuggestions: config.LinkSuggestionsFinder}))
	h.keys("Go")
	h.keys("See [[")
	h.keys("status:onhold")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	p, _ := store.Get("start.md")
	if !strings.HasSuffix(p.Content, "See [[bravo]]") {
		t.Fatalf("want a link to bravo, the only note on hold: %q", p.Content)
	}
}

// S4: ":tags" lists every tag with its note count; picking one opens the
// finder on it, nested tags included.
func TestTagsCommandPicksATagIntoTheFinder(t *testing.T) {
	h := openStartNote(t, tagStore(t))
	h.keys(":tags")
	h.key(tea.KeyEnter)
	v := h.view()
	for _, want := range []string{"#go · 2 notes", "#go/web · 1 note", "#python · 1 note"} {
		if !strings.Contains(v, want) {
			t.Fatalf("the tag list should show %q:\n%s", want, v)
		}
	}
	h.key(tea.KeyEnter) // #go, the first
	v = h.view()
	if !strings.Contains(v, "Find Note") || !strings.Contains(v, "tag:go") || !strings.Contains(v, "3 results") {
		t.Fatalf("want the finder on tag:go with alpha, bravo and charlie:\n%s", v)
	}
	for _, name := range []string{"alpha.md", "bravo.md", "charlie.md"} {
		if !strings.Contains(v, name) {
			t.Fatalf("tag:go should list %s:\n%s", name, v)
		}
	}
}

// S1: ":status value" opens the finder on that status; alone it lists the
// four; an unknown value is an error.
func TestStatusCommandOpensTheFinder(t *testing.T) {
	h := openStartNote(t, tagStore(t))
	h.keys(":status on-hold")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "status:onhold") || !strings.Contains(v, "1 results") || !strings.Contains(v, "bravo.md") {
		t.Fatalf("want the finder on status:onhold with bravo:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.keys(":status")
	h.key(tea.KeyEnter)
	v := h.view()
	if !strings.Contains(v, "status:") || !strings.Contains(v, "active · onhold · completed · dropped") {
		t.Fatalf("want the finder on status: and the four values:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.keys(":status nosuch")
	h.key(tea.KeyEnter)
	wantBar(t, h, "E474: Invalid argument: nosuch")
}
