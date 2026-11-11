package content_test

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

var hangWord = regexp.MustCompile(`w[0-9]{2}`)

// hangColumns returns, for each drawn row holding one of the item's words
// (w00, w01, …), the screen column of its first word.
func hangColumns(view string) []int {
	var cols []int
	for _, row := range strings.Split(ansi.Strip(view), "\n") {
		if loc := hangWord.FindStringIndex(row); loc != nil {
			cols = append(cols, uniseg.StringWidth(row[:loc[0]]))
		}
	}
	return cols
}

func hangWords() string {
	var b strings.Builder
	for i := 0; i < 30; i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("w" + string(rune('0'+i/10)) + string(rune('0'+i%10)))
	}
	return b.String()
}

// S1/S2: in the editor, every continuation row of a list item, a nested item,
// a quote, a list in a quote and an item indented with a tab starts under the
// item's text, and the note's text is unchanged (S3).
func TestEditorHangsWrappedItems(t *testing.T) {
	for _, marker := range []string{"- [ ] ", "- ", "1. ", "  - ", "> ", "> - [ ] ", "\t- "} {
		text := "intro\n" + marker + hangWords() + "\nafter"
		c := content.New(theme.DefaultTheme())
		c.SetSize(40, 20)
		c.SetPage(core.Page{ID: "n.md", Title: "n", Content: text})
		cols := hangColumns(c.View())
		if len(cols) < 3 {
			t.Fatalf("%q: the item did not wrap: %v", marker, cols)
		}
		for i, col := range cols {
			if col != cols[0] {
				t.Errorf("%q: row %d starts at column %d, the item's text at %d", marker, i, col, cols[0])
			}
		}
		// The textarea turns a tab into spaces on load, before any indent.
		if !strings.Contains(marker, "\t") && c.Value() != text {
			t.Errorf("%q: the indent changed the note: %q", marker, c.Value())
		}
	}
}

// S6: a plain line wraps to the left edge as before.
func TestEditorDoesNotHangPlainLines(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(40, 20)
	c.SetPage(core.Page{ID: "n.md", Title: "n", Content: "intro " + hangWords()})
	cols := hangColumns(c.View())
	if len(cols) < 3 || cols[1] >= cols[0] {
		t.Errorf("a plain line's continuation is not at the left edge: %v", cols)
	}
}

// S2: in the reading view, the raw item INSERT reveals hangs too.
func TestLiveInsertHangsTheRawItem(t *testing.T) {
	text := "- [ ] " + hangWords() + "\nafter"
	c := liveEditor(t, 40, text)
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	cols := hangColumns(c.View())
	if len(cols) < 3 {
		t.Fatalf("the raw item did not wrap: %v\n%s", cols, ansi.Strip(c.View()))
	}
	for i, col := range cols {
		if col != cols[0] {
			t.Errorf("row %d starts at column %d, the item's text at %d", i, col, cols[0])
		}
	}
}
