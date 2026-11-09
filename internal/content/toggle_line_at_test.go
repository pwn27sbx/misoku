package content_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// S1: ToggleLineAt toggles one line as one change u undoes, for the
// preview's clicks.
func TestToggleLineAtTogglesOneLineAsOneChange(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "a.md", Content: "- [ ] one\n- [ ] two"})

	if !c.ToggleLineAt(1) {
		t.Fatal("ToggleLineAt should report the change")
	}
	if got, want := c.Value(), "- [ ] one\n- [x] two"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if !c.ToggleLineAt(1) {
		t.Fatal("a second toggle is a change too")
	}
	if got, want := c.Value(), "- [ ] one\n- [ ] two"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	c = keys(c, "u")
	if got, want := c.Value(), "- [ ] one\n- [x] two"; got != want {
		t.Fatalf("one u should undo one toggle: got %q, want %q", got, want)
	}
}
