package app_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// needleNote is long enough that its last line starts off screen.
func needleNote() string {
	var b strings.Builder
	for i := range 120 {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	b.WriteString("the needle line\n")
	return b.String()
}

// S2: / searches the open note and scrolls the editor to the match.
func TestSearchMovesTheEditorToTheMatch(t *testing.T) {
	h := openWith(t, needleNote())
	if strings.Contains(editorPane(h.view()), "the needle line") {
		t.Fatal("the last line should start off screen")
	}
	h.keys("/needle")
	h.key(tea.KeyEnter)
	if v := editorPane(h.view()); !strings.Contains(v, "the needle line") {
		t.Fatalf("the editor should show the match:\n%s", v)
	}
}

// S2: the reading view searches too, scrolling the rendered note to it.
func TestReadingViewSearchScrollsToTheMatch(t *testing.T) {
	h := openWith(t, needleNote())
	h.keys(" ")
	h.keys("p") // reading view
	if strings.Contains(h.view(), "the needle line") {
		t.Fatal("the last line should start off screen")
	}
	h.keys("/needle")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "the needle line") {
		t.Fatalf("the reading view should scroll to the match:\n%s", v)
	}
}

// S2: a search that wraps says so in the status bar.
func TestSearchWrapMessage(t *testing.T) {
	h := openWith(t, "foo\nbar foo")
	h.keys("/foo")
	h.key(tea.KeyEnter)
	h.keys("n")
	if got := app.StatusText(h.m); got != "search hit BOTTOM, continuing at TOP" {
		t.Fatalf("status = %q", got)
	}
}

// S2/A6: "/" in the global graph is still its filter, never the note search.
func TestSlashInTheGlobalGraphStillFilters(t *testing.T) {
	h := globalWorkspace(t)
	h.keys("/Orphan")
	if v := globalTitle(h); !strings.Contains(v, `filter "Orphan"`) {
		t.Fatalf("the graph should filter on Orphan:\n%s", v)
	}
	if got := h.editor(); got != "" {
		t.Fatalf("the filter must not type into the note: %q", got)
	}
	h.key(tea.KeyEsc) // clears the filter
	if v := globalTitle(h); strings.Contains(v, "filter") {
		t.Fatalf("Esc should clear the filter:\n%s", v)
	}
}

// S2: ? still opens the help, and / in INSERT still opens the block menu.
func TestSearchKeysDoNotShadowHelpOrTheBlockMenu(t *testing.T) {
	h := openWith(t, "body")
	h.keys("i/")
	if !strings.Contains(h.view(), "Basic blocks") {
		t.Fatalf("/ in INSERT should open the block menu:\n%s", h.view())
	}
	h.key(tea.KeyEsc) // closes the menu only
	h.key(tea.KeyEsc) // leaves INSERT
	h.keys("?")
	if v := h.view(); !strings.Contains(v, paletteTitle) {
		t.Fatalf("? should open the help:\n%s", v)
	}
}
