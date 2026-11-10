package content_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"

	tea "github.com/charmbracelet/bubbletea"
)

// typeText sends s one key at a time, as a terminal does: the space bar
// arrives as tea.KeySpace, every other character as tea.KeyRunes.
func typeText(c content.Model, s string) content.Model {
	for _, r := range s {
		k := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		if r == ' ' {
			k.Type = tea.KeySpace
		}
		c, _ = c.Update(k)
	}
	return c
}

// insertAt opens text, moves the cursor right by col and enters INSERT.
func insertAt(text string, col int) content.Model {
	return keys(newNote(text), strings.Repeat("l", col)+"i")
}

// S1: the space that follows "<marker> []" completes the standard task, the
// cursor right after it and the text after the cursor untouched.
func TestTypedSpaceAfterEmptyBracketsMakesATask(t *testing.T) {
	for _, tc := range []struct {
		name, indent, typed, want string
	}{
		{"dash", "", "- [] ", "- [ ] "},
		{"star", "", "* [] ", "* [ ] "},
		{"plus", "", "+ [] ", "+ [ ] "},
		{"ordered dot", "", "1. [] ", "1. [ ] "},
		{"ordered paren", "", "1) [] ", "1) [ ] "},
		{"two digits", "", "12. [] ", "12. [ ] "},
		{"indented", "    ", "- [] ", "- [ ] "},
		{"indented ordered", "  ", "3) [] ", "3) [ ] "},
		{"wide gap after the marker", "", "-   [] ", "-   [ ] "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, tail := range []string{"", "tail text"} {
				c := typeText(insertAt(tc.indent+tail, len(tc.indent)), tc.typed)
				if want := tc.indent + tc.want + tail; c.Value() != want {
					t.Fatalf("after %q with %q after the cursor: line = %q, want %q", tc.typed, tail, c.Value(), want)
				}
				row, col := c.CursorPosition()
				if want := len(tc.indent) + len(tc.want) + 1; row != 1 || col != want {
					t.Fatalf("after %q with %q after the cursor: cursor = %d:%d, want 1:%d", tc.typed, tail, row, col, want)
				}
			}
		})
	}
}

// S2: only the space after a bare "[]" on a list item converts; links,
// embeds, wiki links, plain text and the "]" itself are left alone.
func TestTypedSpaceLeavesOtherBracketsAlone(t *testing.T) {
	for _, typed := range []string{
		"- []",
		"- [](url)",
		"- [](url) ",
		"- [](https://github.com)",
		"- [texto](url)",
		"- [texto](url) y más",
		"- ![](foto.png)",
		"- ![](foto.png) ",
		"- [[nota]]",
		"- [[nota]] ",
		"texto [] ",
		"[] ",
		"-[] ",
		"- x [] ",
		"- [x] ",
		"- [ ] ",
	} {
		t.Run(typed, func(t *testing.T) {
			if c := typeText(insertAt("", 0), typed); c.Value() != typed {
				t.Fatalf("typing %q left %q", typed, c.Value())
			}
		})
	}
}

// S3: inside a fenced code block the characters are the code's own.
func TestTypedSpaceInsideAFenceStaysLiteral(t *testing.T) {
	c := typeText(keys(newNote("```\n\n```"), "ji"), "- [] ")
	if want := "```\n- [] \n```"; c.Value() != want {
		t.Fatalf("value = %q, want %q", c.Value(), want)
	}
}

// S4: a paste is left as it came, whole or ending on the single space.
func TestPastedSpaceDoesNotMakeATask(t *testing.T) {
	paste := func(c content.Model, s string) content.Model {
		c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Paste: true})
		return c
	}
	if c := paste(insertAt("", 0), "- [] "); c.Value() != "- [] " {
		t.Fatalf("pasting the whole line left %q", c.Value())
	}
	if c := paste(typeText(insertAt("", 0), "- []"), " "); c.Value() != "- [] " {
		t.Fatalf("pasting only the space left %q", c.Value())
	}
}

// A space typed anywhere else is an ordinary space.
func TestTypedSpaceElsewhereIsPlain(t *testing.T) {
	const typed = "uno dos  tres - cuatro "
	c := typeText(insertAt("", 0), typed)
	if c.Value() != typed {
		t.Fatalf("value = %q, want %q", c.Value(), typed)
	}
	if row, col := c.CursorPosition(); row != 1 || col != len(typed)+1 {
		t.Fatalf("cursor = %d:%d, want 1:%d", row, col, len(typed)+1)
	}
}
