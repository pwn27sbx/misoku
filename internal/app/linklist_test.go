package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// linkStore is a vault whose newest note, Deploy Guide, has headings.
func linkStore(t *testing.T) *core.Store {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][3]string{
		{"", "Start", "home"},
		{"tech", "Deploy Guide", "# Deploy Guide\n\n## Steps\n\nbody ^ship"},
	} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

// openDeployGuide opens the newest note and leaves it in NORMAL mode.
func openDeployGuide(t *testing.T, opts ...app.Option) *harness {
	t.Helper()
	h := newHarness(t, linkStore(t), opts...)
	h.keys("1")
	h.key(tea.KeyEsc)
	return h
}

// S3: "[[" opens the inline list, not the finder, and Enter inserts.
func TestDoubleBracketOpensTheInlineList(t *testing.T) {
	h := openDeployGuide(t)
	h.keys("i[[#step")
	v := h.view()
	if strings.Contains(v, "Link to Note") || strings.Contains(v, "Find Note") {
		t.Fatalf("no finder should open:\n%s", v)
	}
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	p, err := h.store.Get("tech/Deploy Guide.md")
	if err != nil || !strings.HasPrefix(p.Content, "[[#Steps]]") {
		t.Fatalf("Enter on a heading should link to it: %q, %v", p.Content, err)
	}
}

// S3: a query that matches nothing keeps the list on screen, empty.
func TestInlineLinkListShowsNoMatches(t *testing.T) {
	h := openDeployGuide(t)
	h.keys("i[[#zzzz")
	if v := h.view(); !strings.Contains(v, "No matches") {
		t.Fatalf("the inline list should say it found nothing:\n%s", v)
	}
}

// S3: Esc closes the list only; the typed "[[" stays in the note.
func TestInlineLinkListEscLeavesTheText(t *testing.T) {
	h := openDeployGuide(t)
	h.keys("i[[#step")
	h.key(tea.KeyEsc)
	if v := h.view(); strings.Contains(v, "No matches") {
		t.Fatalf("Esc should close the list:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	p, err := h.store.Get("tech/Deploy Guide.md")
	if err != nil || !strings.HasPrefix(p.Content, "[[#step") {
		t.Fatalf("Esc should leave the typed text: %q, %v", p.Content, err)
	}
}

// S3: typing "]]" closes the list.
func TestInlineLinkListClosesOnClosedBrackets(t *testing.T) {
	h := openDeployGuide(t)
	h.keys("i[[#zzzz")
	if v := h.view(); !strings.Contains(v, "No matches") {
		t.Fatalf("the list should be open:\n%s", v)
	}
	h.keys("]]")
	if v := h.view(); strings.Contains(v, "No matches") {
		t.Fatalf("]] should close the list:\n%s", v)
	}
}

// A read that arrives after Esc must not reopen the inline list: only a new
// "[[" does.
func TestInlineLinkListStaysClosedAfterALateSource(t *testing.T) {
	h := openDeployGuide(t)
	h.keys("i[[#zzzz")
	if v := h.view(); !strings.Contains(v, "No matches") {
		t.Fatalf("the list should be open:\n%s", v)
	}
	h.key(tea.KeyEsc)
	// The first "[[" of this app is request 1: this is the read it left in
	// flight when the user closed the list.
	h.send(app.LinkSourceRead(1, []core.LinkNote{{
		ID: "tech/Deploy Guide.md", Title: "Deploy Guide",
		Text: "# Deploy Guide\n\n## Steps\n",
	}}))
	if v := h.view(); strings.Contains(v, "No matches") {
		t.Fatalf("a late source read must not reopen the list:\n%s", v)
	}
}

// S4: the finder comes back with ui.link_suggestions = "finder".
func TestLinkSuggestionsFinderSettingRestoresThePicker(t *testing.T) {
	h := openDeployGuide(t, app.WithUI(config.UI{LinkSuggestions: config.LinkSuggestionsFinder}))
	h.keys("i[[")
	if v := h.view(); !strings.Contains(v, "Link to Note") {
		t.Fatalf("the finder should open with the Finder setting:\n%s", v)
	}
}

// S2/A1: in the link picker "#" lists headings and Enter links to one; the
// footer says what "#" and "^" do there.
func TestLinkFinderListsHeadings(t *testing.T) {
	h := openDeployGuide(t, app.WithUI(config.UI{LinkSuggestions: config.LinkSuggestionsFinder}))
	h.keys("i[[#step")
	v := h.view()
	if !strings.Contains(v, "Link to Note") || !strings.Contains(v, "Steps") {
		t.Fatalf("the picker should list the heading:\n%s", v)
	}
	if !strings.Contains(v, "# heading · ^ block · Esc type it yourself") {
		t.Fatalf("the picker should explain # and ^:\n%s", v)
	}
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	p, err := h.store.Get("tech/Deploy Guide.md")
	if err != nil || !strings.HasPrefix(p.Content, "[[#Steps]]") {
		t.Fatalf("Enter on a heading should link to it: %q, %v", p.Content, err)
	}
}

// A1: tag: still narrows the link picker; "#tag" no longer does.
func TestLinkFinderKeepsTheTagQualifier(t *testing.T) {
	h := openStartNote(t, tagStore(t), app.WithUI(config.UI{LinkSuggestions: config.LinkSuggestionsFinder}))
	h.keys("i[[tag:go bravo")
	v := h.view()
	if !strings.Contains(v, "1 results") || !strings.Contains(v, "bravo.md") {
		t.Fatalf("tag: should still narrow the notes:\n%s", v)
	}
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	p, _ := h.store.Get("start.md")
	if !strings.HasPrefix(p.Content, "[[bravo]]") {
		t.Fatalf("want a link to bravo: %q", p.Content)
	}

	h = openStartNote(t, tagStore(t), app.WithUI(config.UI{LinkSuggestions: config.LinkSuggestionsFinder}))
	h.keys("i[[#go")
	if v := h.view(); !strings.Contains(v, "0 results") {
		t.Fatalf("# should mean headings in the link picker, not tags: %s", v)
	}
}
