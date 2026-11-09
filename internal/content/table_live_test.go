package content_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// tableFormatted is the note the live-alignment tests type into: every
// column three wide, as :tablefmt leaves it.
const tableFormatted = "| a   | b   |\n| --- | --- |\n| 1   | 2   |"

// insertInCell puts the cursor after the first cell's text of the table's
// only body row and enters INSERT.
func insertInCell(c content.Model) content.Model {
	return keys(c, "jjllli")
}

// closePipe is the rune column of the pipe that closes cell col.
func closePipe(line string, col int) int {
	n := -1
	for i, r := range []rune(line) {
		if r == '|' {
			n++
			if n == col+1 {
				return i
			}
		}
	}
	return -1
}

// S1: while the typed text fits the column, the closing pipe stays put; one
// character more widens the column in every row; deleting gives the width
// back.
func TestTableLiveAlignKeepsPipesWhileTheTextFits(t *testing.T) {
	c := insertInCell(newNote(tableFormatted))
	row, _ := c.CursorPosition()
	pipe := closePipe(strings.Split(c.Value(), "\n")[row-1], 0)

	c = keys(c, "2")
	line := strings.Split(c.Value(), "\n")[row-1]
	if got := closePipe(line, 0); got != pipe {
		t.Fatalf("typing inside the padding moved the pipe from %d to %d:\n%s", pipe, got, c.Value())
	}
	if line != "| 12  | 2   |" {
		t.Fatalf("typing 2: row = %q", line)
	}

	c = keys(c, "3")
	if line := strings.Split(c.Value(), "\n")[row-1]; line != "| 123 | 2   |" {
		t.Fatalf("typing 3: row = %q", line)
	}

	c = keys(c, "4")
	want := "| a    | b   |\n| ---- | --- |\n| 1234 | 2   |"
	if c.Value() != want {
		t.Fatalf("one character over the width should widen every row:\n%s\nwant\n%s", c.Value(), want)
	}

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	want = "| a   | b   |\n| --- | --- |\n| 123 | 2   |"
	if c.Value() != want {
		t.Fatalf("backspace should give the padding back:\n%s\nwant\n%s", c.Value(), want)
	}
}

// S2: after every keystroke the cursor is still in the same cell, at the
// same offset in its text.
func TestTableLiveAlignKeepsTheCursorInItsCell(t *testing.T) {
	c := insertInCell(newNote(tableFormatted))
	for i, r := range "23" {
		c = keys(c, string(r))
		row, col := c.CursorPosition()
		wantRow, wantCol := 3, 4+i+1
		if row != wantRow || col != wantCol {
			t.Fatalf("after typing %q the cursor is at %d:%d, want %d:%d:\n%s",
				string(r), row, col, wantRow, wantCol, c.Value())
		}
	}
}

// A1: splitCells trims, so the cursor is restored into the padding with no
// clamp: the space typed at the end of a cell is not lost, the next rune
// lands after it.
func TestTableLiveAlignKeepsATypedSpace(t *testing.T) {
	c := insertInCell(newNote(tableFormatted))
	c = keys(c, " b")
	line := strings.Split(c.Value(), "\n")[2]
	if line != "| 1 b | 2   |" {
		t.Fatalf("typing a space must survive realignment, row = %q", line)
	}
}

// A4: history is recorded only outside INSERT, and the live path never calls
// edit, record or SetValue, so the whole INSERT session is one undo step.
func TestTableLiveAlignIsOneUndoStep(t *testing.T) {
	c := insertInCell(newNote(tableFormatted))
	c = keys(c, "xyz")
	c = esc(c)
	typed := c.Value()
	if typed == tableFormatted {
		t.Fatal("typing did not change the note")
	}
	c, _ = undo(c)
	if c.Value() != tableFormatted {
		t.Fatalf("one u should restore the pre-session text:\n%s", c.Value())
	}
	c, _ = redo(c)
	if c.Value() != typed {
		t.Fatalf("redo should restore the typed text:\n%s", c.Value())
	}
}

// S5: the separator row and alignment colons survive live realignment.
func TestTableLiveAlignKeepsSeparatorColons(t *testing.T) {
	start := "| a   | b   |\n| :-- | --: |\n| 1   | 2   |"
	c := insertInCell(newNote(start))
	c = keys(c, "xyz")
	want := "| a    | b   |\n| :--- | --: |\n| 1xyz | 2   |"
	if c.Value() != want {
		t.Fatalf("colons lost while widening:\n%s\nwant\n%s", c.Value(), want)
	}
}

// S5, A2: text outside a table is untouched, and a pipe line inside a fence
// is never realigned.
func TestTableLiveAlignLeavesTextAndFencesAlone(t *testing.T) {
	c := newNote("hello")
	c = keys(c, "A")
	c = keys(c, " world")
	if c.Value() != "hello world" {
		t.Fatalf("typing outside a table changed more than the text: %q", c.Value())
	}

	fenced := "```\n| a | b |\n| --- | --- |\n| 1 | 2 |\n```"
	c = newNote(fenced)
	c = keys(c, "jjjllli")
	c = keys(c, "x")
	want := strings.Replace(fenced, "| 1 | 2 |", "| 1x | 2 |", 1)
	if c.Value() != want {
		t.Fatalf("a pipe table inside a fence must stay raw:\n%s\nwant\n%s", c.Value(), want)
	}
}

// S5: wide runes keep lining up by display width.
func TestTableLiveAlignUsesDisplayWidth(t *testing.T) {
	start := "| a   | b   |\n| --- | --- |\n| 日  | 2   |"
	c := newNote(start)
	c = keys(c, "jjllli")
	c = keys(c, "本")
	want := "| a    | b   |\n| ---- | --- |\n| 日本 | 2   |"
	if c.Value() != want {
		t.Fatalf("wide runes:\n%s\nwant\n%s", c.Value(), want)
	}
	for _, line := range strings.Split(c.Value(), "\n") {
		runes := []rune(line)
		if got, want := ansi.StringWidth(string(runes[:closePipe(line, 0)+1])), ansi.StringWidth("| 日本 |"); got != want {
			t.Fatalf("pipes do not line up by display width (%d vs %d):\n%s", got, want, c.Value())
		}
	}
}

// A5: the key that types | does not realign; the next Tab decides the new
// column.
func TestTableLiveAlignSkipsATypedPipe(t *testing.T) {
	c := insertInCell(newNote(tableFormatted))
	c = keys(c, "|")
	want := "| a   | b   |\n| --- | --- |\n| 1|   | 2   |"
	if c.Value() != want {
		t.Fatalf("typing | must not realign:\n%q\nwant\n%q", c.Value(), want)
	}
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := strings.Count(strings.Split(c.Value(), "\n")[0], "|"); got != 4 {
		t.Fatalf("Tab should turn the typed pipe into a column of three:\n%s", c.Value())
	}
}

// A6: with the slash menu open the live path stays out of the way, so
// "Add column right" still works from inside a cell.
func TestTableLiveAlignLeavesTheSlashMenuAlone(t *testing.T) {
	c := newNote(tableFormatted)
	c = keys(c, "jjlli") // start of the first body cell, after a space
	c = keys(c, "/")
	if _, _, _, ok := c.SlashView(); !ok {
		t.Fatal("typing / at the start of a cell should open the slash menu")
	}
	c = keys(c, "add column right")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	header := strings.Split(c.Value(), "\n")[0]
	if !strings.Contains(header, "Column") || strings.Count(header, "|") != 4 {
		t.Fatalf("Add column right did not add a column:\n%s", c.Value())
	}
}

// S3: a long note pays only for its table, not for a whole-document rewrite
// per key. Each iteration types and deletes a rune, so the cell keeps its
// width and the number stays comparable.
func BenchmarkTableLiveAlignLongNote(b *testing.B) {
	var src strings.Builder
	for i := 0; i < 2500; i++ {
		fmt.Fprintf(&src, "line %d of the note\n", i)
	}
	src.WriteString(tableFormatted)
	src.WriteString("\n")
	for i := 0; i < 2500; i++ {
		fmt.Fprintf(&src, "line %d after the table\n", i)
	}
	c := newNote(src.String())
	c = keys(c, strings.Repeat("j", 2502)) // the body row
	c = keys(c, "lllli")
	typeKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	delKey := tea.KeyMsg{Type: tea.KeyBackspace}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c, _ = c.Update(typeKey)
		c, _ = c.Update(delKey)
	}
}
