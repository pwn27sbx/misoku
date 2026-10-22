package content_test

import (
	"reflect"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/highlight"
	"github.com/pwn27sbx/misoku/internal/theme"
)

const hlNote = "# Title\n\nsome **bold** text and `code`\n\n```go\nfunc main() {}\n```"

func openNote(th theme.Theme, text string) content.Model {
	c := content.New(th)
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "n.md", Title: "n", Content: text})
	return c
}

func TestHighlightFollowsEdits(t *testing.T) {
	th := theme.DefaultTheme()
	c := openNote(th, "plain")
	c.View()
	c = keys(c, "I## ")
	c.View()
	if got, want := c.ViewColors(), highlight.Markdown(c.Value(), th); !reflect.DeepEqual(got, want) {
		t.Fatalf("colours after an edit are stale:\n got %v\nwant %v", got, want)
	}
}

// A user theme can change its colours on disk and keep its name: the new
// colours must show, not the ones cached for the name.
func TestHighlightFollowsAThemeThatKeepsItsName(t *testing.T) {
	first := theme.DefaultTheme()
	var second theme.Theme
	for _, n := range theme.Names() {
		if th, ok := theme.Get(n); ok && n != first.Name && !reflect.DeepEqual(highlight.Markdown(hlNote, th), highlight.Markdown(hlNote, first)) {
			second = th
			break
		}
	}
	if second.Name == "" {
		t.Fatal("no theme colours the note differently")
	}
	second.Name = first.Name
	c := openNote(first, hlNote)
	c.View()
	c.SetTheme(second)
	c.View()
	if got, want := c.ViewColors(), highlight.Markdown(hlNote, second); !reflect.DeepEqual(got, want) {
		t.Fatal("colours after a theme change are the old theme's")
	}
}

// Opening a fence by typing recolours the rest of the note as code, and
// switching to another page with the same editor still matches a full
// highlight (the incremental highlighter diffs against the last text).
func TestHighlightFollowsAFenceTypedAndANewPage(t *testing.T) {
	th := theme.DefaultTheme()
	c := openNote(th, "# Title\n\nplain\n# After")
	c.View()
	c = keys(c, "jI```go")
	if c.Value() != "# Title\n```go\nplain\n# After" {
		t.Fatalf("typing did not open a fence: %q", c.Value())
	}
	c.View()
	if got, want := c.ViewColors(), highlight.Markdown(c.Value(), th); !reflect.DeepEqual(got, want) {
		t.Fatalf("colours after opening a fence are stale:\n got %v\nwant %v", got, want)
	}
	c.SetPage(core.Page{ID: "m.md", Title: "m", Content: hlNote})
	c.View()
	if got, want := c.ViewColors(), highlight.Markdown(hlNote, th); !reflect.DeepEqual(got, want) {
		t.Fatalf("colours after switching page are stale:\n got %v\nwant %v", got, want)
	}
}
