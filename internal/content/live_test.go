package content_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

const liveNote = "cursor line **raw**\n" +
	"# Heading\n" +
	"some **bold** and *it* and `code`\n" +
	"see [text](http://x) and [[Note|alias]] and [[Other]]\n" +
	"- [ ] todo\n" +
	"- [x] done\n" +
	"- item\n" +
	"> quote\n" +
	"```\n" +
	"**inside** [[Note]]\n" +
	"```\n" +
	"| a | **b** |"

// liveEditor is a lit editor on text, w columns wide, with the live view on.
func liveEditor(t *testing.T, w int, text string) content.Model {
	t.Helper()
	trueColour(t)
	c := content.New(theme.DefaultTheme())
	c.SetSize(w, 20)
	c.SetPage(core.Page{ID: "n.md", Title: "n", Content: text})
	c.SetLive(true)
	return c
}

func TestLiveHidesLineNumbers(t *testing.T) {
	c := liveEditor(t, 60, liveNote)
	if !c.Live() {
		t.Fatal("Live() = false after SetLive(true)")
	}
	row0 := strings.Split(ansi.Strip(c.View()), "\n")[0]
	if !strings.HasPrefix(row0, "cursor") {
		t.Errorf("row 0 = %q, want no line number", row0)
	}
}

// With the switch off, the editor draws exactly what it drew before the
// live view existed: same rows, same numbers, same escapes.
func TestLiveOffRendersAsBefore(t *testing.T) {
	trueColour(t)
	before := content.New(theme.DefaultTheme())
	before.SetSize(60, 20)
	before.SetPage(core.Page{ID: "n.md", Title: "n", Content: liveNote})
	want := before.View()

	c := liveEditor(t, 60, liveNote)
	_ = c.View()
	c.SetLive(false)
	if c.Live() {
		t.Fatal("Live() = true after SetLive(false)")
	}
	if got := c.View(); got != want {
		t.Errorf("View after SetLive(false) differs from before:\n%q\nwant\n%q", got, want)
	}
}
