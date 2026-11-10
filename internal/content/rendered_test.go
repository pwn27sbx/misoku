package content_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// renderedEditor is an editor on text, in the view's width.
func renderedEditor(t *testing.T, text string) content.Model {
	t.Helper()
	c := content.New(theme.DefaultTheme())
	c.SetSize(60, 20)
	c.SetPage(core.Page{ID: "n.md", Title: "n", Content: text})
	c.SetLive(true)
	return c
}

// compiled is a compile of note n.md made of blocks, numbered gen.
func compiled(gen uint64, blocks ...content.RenderedBlock) content.Rendered {
	return content.Rendered{
		Generation: gen, PageID: "n.md", Current: true,
		Blocks: func() []content.RenderedBlock { return blocks },
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func press(t *testing.T, c content.Model, keys ...string) (content.Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, k := range keys {
		c, cmd = c.Update(key(k))
	}
	return c, cmd
}

// plainView is the editor's real view with its escapes stripped.
func plainView(c content.Model) []string {
	out := strings.Split(c.View(), "\n")
	for i, r := range out {
		out[i] = strings.TrimRight(ansi.Strip(r), " ")
	}
	return out
}

// S11: without a compile, and with a failed one, lines draw raw and edit.
func TestRenderedFailedCompileDrawsRawAndEdits(t *testing.T) {
	c := renderedEditor(t, "# Head\nbody")
	if got := plainView(c); got[0] != "# Head" || got[1] != "body" {
		t.Fatalf("no compile should draw raw, got %q", got[:2])
	}
	c.ShowRendered(content.Rendered{Generation: 1, PageID: "n.md", Current: true, Failed: true}, false)
	if got := plainView(c); got[0] != "# Head" || got[1] != "body" {
		t.Fatalf("a failed compile should draw raw, got %q", got[:2])
	}
	c, _ = press(t, c, "i", "x", "esc")
	if got := c.Value(); got != "x# Head\nbody" {
		t.Fatalf("a failed compile should still edit, got %q", got)
	}
}

// S11: a compile of another note leaves the lines raw.
func TestRenderedAnotherNoteIsNeverApplied(t *testing.T) {
	c := renderedEditor(t, "one\ntwo")
	r := compiled(1,
		content.RenderedBlock{Line: 0, End: 1, Rows: []string{"ONE"}},
		content.RenderedBlock{Line: 1, End: 2, Rows: []string{"THEIRS"}},
	)
	r.PageID = "other.md"
	c.ShowRendered(r, false)
	if got := strings.Join(plainView(c), "\n"); strings.Contains(got, "THEIRS") || !strings.Contains(got, "two") {
		t.Fatalf("another note's rows were applied:\n%s", got)
	}
}

// S3, S8, S15: a folded section hides its body. NORMAL draws every block
// rendered; in INSERT the cursor on the heading shows it raw and keeps the
// body folded, and the cursor on the body shows the whole section raw.
func TestRenderedFoldedSection(t *testing.T) {
	c := renderedEditor(t, "top\n# H\nb1\nb2\nend")
	r := compiled(1,
		content.RenderedBlock{Line: 0, End: 1, Rows: []string{"TOP"}},
		content.RenderedBlock{Line: 1, End: 4, Rows: []string{"FOLD"}, Folded: true},
		content.RenderedBlock{Line: 4, End: 5, Rows: []string{"END"}},
	)
	insert := func(line int) []string {
		t.Helper()
		c.GotoLine(line)
		c.ShowRendered(r, false)
		c, _ = press(t, c, "i")
		c.ShowRendered(r, false)
		got := plainView(c)
		c, _ = press(t, c, "esc")
		c.ShowRendered(r, false)
		return got
	}
	c.ShowRendered(r, false)
	if got := plainView(c); got[0] != "TOP" || got[1] != "FOLD" || got[2] != "END" {
		t.Fatalf("NORMAL on line 0: %q", got[:3])
	}
	if got := insert(1); got[0] != "top" || got[1] != "FOLD" || got[2] != "END" {
		t.Fatalf("INSERT on line 0: %q", got[:3])
	}
	if got := insert(2); got[0] != "TOP" || got[1] != "# H" || got[2] != "END" {
		t.Fatalf("INSERT on the folded heading: %q", got[:3])
	}
	if got := plainView(c); got[1] != "FOLD" {
		t.Fatalf("NORMAL on the folded heading: %q", got[:3])
	}
	if got := insert(3); got[1] != "# H" || got[2] != "b1" || got[3] != "b2" || got[4] != "END" {
		t.Fatalf("INSERT on the folded body: %q", got[:5])
	}
}

// S4: copying a selection over rendered blocks yields the Markdown.
func TestRenderedCopyYieldsRawText(t *testing.T) {
	c := renderedEditor(t, "**one**\n**two**\nend")
	r := compiled(1,
		content.RenderedBlock{Line: 0, End: 1, Rows: []string{"ONE"}},
		content.RenderedBlock{Line: 1, End: 2, Rows: []string{"TWO"}},
		content.RenderedBlock{Line: 2, End: 3, Rows: []string{"END"}},
	)
	c.ShowRendered(r, false)
	c, _ = press(t, c, "V", "j")
	c.ShowRendered(r, false)
	if got := plainView(c); got[0] != "**one**" || got[1] != "**two**" || got[2] != "END" {
		t.Fatalf("the selection should reveal both blocks: %q", got[:3])
	}
	_, cmd := press(t, c, "y")
	msg, ok := cmd().(core.CopyMsg)
	if !ok || msg.Text != "**one**\n**two**" {
		t.Fatalf("copy gave %#v", msg)
	}
}
