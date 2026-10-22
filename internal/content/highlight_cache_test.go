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
