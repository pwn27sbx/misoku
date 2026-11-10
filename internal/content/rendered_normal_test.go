package content_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// NORMAL in the rendered view (S15): every block rendered, j/k by block, and
// the line commands acting on the cursor's whole block.

// tableNote has a one-line paragraph, a three-line table and a last line.
const tableNote = "intro\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nend"

const rawTable = "| a | b |\n|---|---|\n| 1 | 2 |"

// tableCompile is a compile of tableNote: blank runs own one spacing row.
func tableCompile() content.Rendered {
	return compiled(1,
		content.RenderedBlock{Line: 0, End: 1, Row: 0, Rows: []string{"INTRO"}},
		content.RenderedBlock{Line: 1, End: 2, Row: 1, Rows: []string{""}},
		content.RenderedBlock{Line: 2, End: 5, Row: 2, Rows: []string{"TABLE1", "TABLE2", "TABLE3"}},
		content.RenderedBlock{Line: 5, End: 6, Row: 5, Rows: []string{""}},
		content.RenderedBlock{Line: 6, End: 7, Row: 6, Rows: []string{"END"}},
	)
}

// shown is an editor on tableNote showing tableCompile, the cursor on
// 1-based line.
func shown(t *testing.T, line int) content.Model {
	t.Helper()
	c := renderedEditor(t, tableNote)
	c.GotoLine(line)
	c.ShowRendered(tableCompile(), false)
	return c
}

// keysShown presses keys, handing the compile over after each as the app
// does, and returns the last key's command.
func keysShown(t *testing.T, c content.Model, keys ...string) (content.Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, k := range keys {
		c, cmd = c.Update(key(k))
		c.ShowRendered(tableCompile(), false)
	}
	return c, cmd
}

// copied returns the text cmd copies, or "".
func copied(cmd tea.Cmd) string {
	if cmd == nil {
		return ""
	}
	if msg, ok := cmd().(core.CopyMsg); ok {
		return msg.Text
	}
	return ""
}

func line(c content.Model) int { l, _ := c.CursorPosition(); return l }

// S15: in NORMAL the cursor's block is rendered; INSERT shows it raw and Esc
// renders it again.
func TestRenderedNormalDrawsTheCursorsBlock(t *testing.T) {
	c := shown(t, 3)
	if got := strings.Join(plainView(c), "\n"); !strings.Contains(got, "TABLE1\nTABLE2\nTABLE3") || strings.Contains(got, "|---|") {
		t.Fatalf("NORMAL should draw the cursor's table rendered:\n%s", got)
	}
	c, _ = keysShown(t, c, "i")
	if got := strings.Join(plainView(c), "\n"); !strings.Contains(got, rawTable) || !strings.Contains(got, "INTRO") {
		t.Fatalf("INSERT should draw the cursor's table raw, the rest rendered:\n%s", got)
	}
	c, _ = keysShown(t, c, "esc")
	if got := strings.Join(plainView(c), "\n"); strings.Contains(got, "|---|") {
		t.Fatalf("Esc should render the table again:\n%s", got)
	}
}

// S15: a blank line holding the cursor in NORMAL is not raw either.
func TestRenderedNormalBlankLineIsNotRaw(t *testing.T) {
	c := renderedEditor(t, "**one**\n\n**two**")
	c.GotoLine(2)
	c.ShowRendered(compiled(1,
		content.RenderedBlock{Line: 0, End: 1, Rows: []string{"ONE"}},
		content.RenderedBlock{Line: 1, End: 2, Rows: []string{"GAP"}},
		content.RenderedBlock{Line: 2, End: 3, Rows: []string{"TWO"}},
	), false)
	if got := plainView(c); got[0] != "ONE" || got[1] != "GAP" || got[2] != "TWO" {
		t.Fatalf("NORMAL on the blank line should keep both blocks rendered: %q", got[:3])
	}
}

// S15: j and k go block by block over the blank runs; G lands on the first
// line of the last block.
func TestRenderedNormalMovesBlockByBlock(t *testing.T) {
	c := shown(t, 1)
	want := []struct {
		key  string
		line int
	}{{"j", 3}, {"j", 7}, {"j", 7}, {"k", 3}, {"k", 1}, {"G", 7}}
	for _, w := range want {
		c, _ = keysShown(t, c, w.key)
		if got := line(c); got != w.line {
			t.Fatalf("%s landed on line %d, want %d", w.key, got, w.line)
		}
	}

	c = renderedEditor(t, "intro\n\n"+rawTable+"\n")
	r := compiled(1,
		content.RenderedBlock{Line: 0, End: 1, Rows: []string{"INTRO"}},
		content.RenderedBlock{Line: 1, End: 2, Rows: []string{""}},
		content.RenderedBlock{Line: 2, End: 6, Rows: []string{"T1", "T2", "T3"}},
	)
	c.ShowRendered(r, false)
	c, _ = c.Update(key("G"))
	if got := line(c); got != 3 {
		t.Fatalf("G should land on the last block's first line 3, got %d", got)
	}
	c.ShowRendered(r, false)
	c, _ = c.Update(key("i"))
	c.ShowRendered(r, false)
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := line(c); got != 4 {
		t.Fatalf("INSERT's down should still go line by line, got %d", got)
	}
}

// S15: dd deletes the whole block as one undo step and yanks its Markdown.
func TestRenderedNormalDdDeletesTheBlock(t *testing.T) {
	c := shown(t, 3)
	c, cmd := keysShown(t, c, "d", "d")
	if got := c.Value(); got != "intro\n\n\nend" {
		t.Fatalf("dd on the table left %q", got)
	}
	if got := copied(cmd); got != rawTable+"\n" {
		t.Fatalf("dd should yank the table's lines, got %q", got)
	}
	c, _ = keysShown(t, c, "u")
	if got := c.Value(); got != tableNote {
		t.Fatalf("one u should restore the table, got %q", got)
	}

	c = shown(t, 1)
	c, cmd = keysShown(t, c, "d", "d")
	if got := c.Value(); got != "\n"+rawTable+"\n\nend" || copied(cmd) != "intro\n" {
		t.Fatalf("dd on a one-line paragraph left %q, yanked %q", got, copied(cmd))
	}
}

// S15: yy yanks the whole block's Markdown and changes nothing.
func TestRenderedNormalYyYanksTheBlock(t *testing.T) {
	c, cmd := keysShown(t, shown(t, 3), "y", "y")
	if got := copied(cmd); got != rawTable+"\n" || c.Value() != tableNote {
		t.Fatalf("yy on the table yanked %q, left %q", got, c.Value())
	}
	if _, cmd = keysShown(t, shown(t, 1), "y", "y"); copied(cmd) != "intro\n" {
		t.Fatalf("yy on a one-line paragraph yanked %q", copied(cmd))
	}
}

// S15: o opens a line below the block's last line, O above its first, from
// any line of the block, in INSERT.
func TestRenderedNormalOpensAroundTheBlock(t *testing.T) {
	cases := []struct {
		line int
		keys string
		want string
	}{
		{3, "o", "intro\n\n" + rawTable + "\nnew\n\nend"},
		{4, "O", "intro\n\nnew\n" + rawTable + "\n\nend"},
		{1, "o", "intro\nnew\n\n" + rawTable + "\n\nend"},
		{1, "O", "new\nintro\n\n" + rawTable + "\n\nend"},
	}
	for _, tc := range cases {
		c, _ := keysShown(t, shown(t, tc.line), tc.keys)
		if c.Mode() != content.ModeInsert {
			t.Fatalf("%s on line %d should enter INSERT", tc.keys, tc.line)
		}
		c, _ = keysShown(t, c, "n", "e", "w", "esc")
		if got := c.Value(); got != tc.want {
			t.Fatalf("%s on line %d left %q, want %q", tc.keys, tc.line, got, tc.want)
		}
	}
}

// S15: p and P paste whole lines below and above the block, as one undo
// step, whatever the clipboard ends with.
func TestRenderedNormalPastesAroundTheBlock(t *testing.T) {
	for _, clip := range []string{"X\n", "X"} {
		restore := content.SetClipboard(func() (string, error) { return clip, nil })
		cases := []struct {
			line int
			key  string
			want string
		}{
			{3, "p", "intro\n\n" + rawTable + "\nX\n\nend"},
			{3, "P", "intro\n\nX\n" + rawTable + "\n\nend"},
			{1, "p", "intro\nX\n\n" + rawTable + "\n\nend"},
			{1, "P", "X\nintro\n\n" + rawTable + "\n\nend"},
		}
		for _, tc := range cases {
			c, _ := keysShown(t, shown(t, tc.line), tc.key)
			if got := c.Value(); got != tc.want {
				t.Fatalf("%s of %q on line %d left %q, want %q", tc.key, clip, tc.line, got, tc.want)
			}
			if c, _ = keysShown(t, c, "u"); c.Value() != tableNote {
				t.Fatalf("one u should undo %s, got %q", tc.key, c.Value())
			}
		}
		restore()
	}
}

// S15: >> and << indent and outdent every line of the block, one undo step
// each.
func TestRenderedNormalIndentsTheBlock(t *testing.T) {
	c, _ := keysShown(t, shown(t, 3), ">", ">")
	indented := "intro\n\n  | a | b |\n  |---|---|\n  | 1 | 2 |\n\nend"
	if got := c.Value(); got != indented {
		t.Fatalf(">> on the table left %q", got)
	}
	c, _ = keysShown(t, c, "<", "<")
	if got := c.Value(); got != tableNote {
		t.Fatalf("<< should take the indent back, got %q", got)
	}
	c, _ = keysShown(t, c, "u")
	if got := c.Value(); got != indented {
		t.Fatalf("one u should undo <<, got %q", got)
	}
	c, _ = keysShown(t, shown(t, 1), ">", ">")
	if got := c.Value(); got != "  intro\n\n"+rawTable+"\n\nend" {
		t.Fatalf(">> on a one-line paragraph left %q", got)
	}
}

// S15, assumption: J does nothing on a block of several lines (joining would
// break it); x keeps acting on the source column.
func TestRenderedNormalJoinAndCharacterKeys(t *testing.T) {
	c := renderedEditor(t, "one\ntwo\n\nend")
	r := compiled(1,
		content.RenderedBlock{Line: 0, End: 2, Rows: []string{"ONE TWO"}},
		content.RenderedBlock{Line: 2, End: 3, Rows: []string{""}},
		content.RenderedBlock{Line: 3, End: 4, Rows: []string{"END"}},
	)
	c.ShowRendered(r, false)
	c, _ = c.Update(key("J"))
	if got := c.Value(); got != "one\ntwo\n\nend" {
		t.Fatalf("J on a two-line paragraph left %q", got)
	}
	c.ShowRendered(r, false)
	c, _ = c.Update(key("x"))
	if got := c.Value(); got != "ne\ntwo\n\nend" {
		t.Fatalf("x should delete the source character under the cursor, left %q", got)
	}
}

// S10: without a display, P, >> and << stay what they were: nothing.
func TestRawEditorKeepsItsLineCommands(t *testing.T) {
	restore := content.SetClipboard(func() (string, error) { return "X\n", nil })
	defer restore()
	c := renderedEditor(t, "one\ntwo")
	c, _ = press(t, c, "P", ">", ">", "<", "<")
	if got := c.Value(); got != "one\ntwo" {
		t.Fatalf("the raw editor changed: %q", got)
	}
	c, _ = press(t, c, ">", "j")
	if l, _ := c.CursorPosition(); l != 2 {
		t.Fatalf("> should not wait for a second key in the raw editor: j left line %d", l)
	}
	c, _ = press(t, c, "k")
	c, _ = press(t, c, "j", "d", "d")
	if got := c.Value(); got != "one" {
		t.Fatalf("dd in the raw editor should delete one line, left %q", got)
	}
}

// S15: the cursor on a line of a block without rows of its own (here the
// table's second line, where a click can put it) is on the block's rows.
func TestRenderedNormalRowlessLineIsOnItsBlock(t *testing.T) {
	c := shown(t, 4)
	c.Focus()
	if _, y, ok := c.CursorScreen(); !ok || y != 2 {
		t.Fatalf("the cursor is on row %d (ok %v), want the table's first row 2", y, ok)
	}
}

// S15, assumption: a run of blank lines is no block: dd there deletes the
// cursor's line only.
func TestRenderedNormalBlankRunKeepsLineCommands(t *testing.T) {
	c := renderedEditor(t, "a\n\n\nb")
	c.GotoLine(3)
	c.ShowRendered(compiled(1,
		content.RenderedBlock{Line: 0, End: 1, Rows: []string{"A"}},
		content.RenderedBlock{Line: 1, End: 3, Rows: []string{""}},
		content.RenderedBlock{Line: 3, End: 4, Rows: []string{"B"}},
	), false)
	c, _ = press(t, c, "d", "d")
	if got := c.Value(); got != "a\n\nb" {
		t.Fatalf("dd on a blank line left %q", got)
	}
}
