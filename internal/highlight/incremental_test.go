package highlight

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/testnote"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// linePool mixes everything that changes the highlighter's state: fences of
// both kinds and several lengths (with and without a language, indented),
// front-matter delimiters, multi-line constructs chroma tokenises across
// lines, CRLF endings and wide or accented runes.
var linePool = []string{
	"", "```", "```go", "````", "~~~", "~~~python", "~~~~", "  ```js", "```c\r",
	"---", "...", "---\r", "# Heading", "## Sub **b**", "- [x] done", "- [ ] open",
	"1. step", "> quote [l](u)", "plain `code` and [[wiki|w]] ^id",
	"title: hi", "tags: [a, b]", "func main() {", "}", "/* open comment",
	"close */", `x = "unterminated`, "'''", "<b>tag</b>", "日本語の行 **太字**",
	"ünïcödé ~~strike~~", "line with CRLF\r", "***", "\t```",
}

func otherTheme(t testing.TB) theme.Theme {
	first := theme.DefaultTheme()
	for _, n := range theme.Names() {
		if th, ok := theme.Get(n); ok && th != first {
			return th
		}
	}
	t.Fatal("no second theme")
	return first
}

// check highlights text incrementally and compares it with a fresh full
// highlight and with the pre-refactor implementation.
func check(t testing.TB, h *Highlighter, text string, th theme.Theme, step string) {
	t.Helper()
	got := h.Markdown(text, th)
	if want := Markdown(text, th); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: incremental differs from a full highlight\ntext: %q\n got %v\nwant %v", step, text, got, want)
	}
	if want := legacyMarkdown(text, th); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: differs from the previous Markdown\ntext: %q\n got %v\nwant %v", step, text, got, want)
	}
}

// edit applies one random edit to lines, driven by three numbers so random
// tests and the fuzzer share it.
func edit(lines []string, kind, pos, pick int, extra string) ([]string, string) {
	at := 0
	if len(lines) > 0 {
		at = pos % (len(lines) + 1)
	}
	line := linePool[pick%len(linePool)]
	switch kind % 9 {
	case 0: // insert one line
		return insertAt(lines, at, line), fmt.Sprintf("insert %q at %d", line, at)
	case 1: // insert several lines (a paste)
		n := pick%4 + 2
		block := make([]string, n)
		for i := range block {
			block[i] = linePool[(pick+i*7)%len(linePool)]
		}
		return insertAt(lines, at, block...), fmt.Sprintf("paste %q at %d", block, at)
	case 2: // delete a range
		if len(lines) == 0 {
			return lines, "delete nothing"
		}
		at %= len(lines)
		end := min(at+pick%4+1, len(lines))
		return append(append([]string{}, lines[:at]...), lines[end:]...), fmt.Sprintf("delete %d..%d", at, end)
	case 3: // replace a line
		if len(lines) == 0 {
			return []string{line}, "replace empty"
		}
		at %= len(lines)
		out := append([]string{}, lines...)
		out[at] = line
		return out, fmt.Sprintf("replace %d with %q", at, line)
	case 4: // type one rune into a line
		if len(lines) == 0 {
			return []string{"x"}, "type into empty"
		}
		at %= len(lines)
		out := append([]string{}, lines...)
		r := []rune(out[at])
		c := pick % (len(r) + 1)
		typed := []rune("`~-x#é日\r")
		ins := typed[pick%len(typed)]
		out[at] = string(r[:c]) + string(ins) + string(r[c:])
		return out, fmt.Sprintf("type %q into line %d", ins, at)
	case 5: // edit the first line (front matter on/off)
		out := append([]string{}, lines...)
		if len(out) == 0 {
			return []string{"---"}, "open front matter"
		}
		if out[0] == "---" {
			out[0] = "x"
		} else {
			out[0] = "---"
		}
		return out, "toggle first line"
	case 6: // edit the last line
		if len(lines) == 0 {
			return []string{line}, "last of empty"
		}
		out := append([]string{}, lines...)
		out[len(out)-1] = line
		return out, fmt.Sprintf("last line = %q", line)
	case 7: // replace the whole text
		return strings.Split(extra, "\n"), "whole replace"
	default: // join two lines (backspace at a line start)
		if len(lines) < 2 {
			return lines, "join nothing"
		}
		at = at%(len(lines)-1) + 1
		out := append([]string{}, lines[:at-1]...)
		out = append(out, lines[at-1]+lines[at])
		return append(out, lines[at+1:]...), fmt.Sprintf("join %d", at)
	}
}

func insertAt(lines []string, at int, add ...string) []string {
	out := make([]string, 0, len(lines)+len(add))
	out = append(out, lines[:at]...)
	out = append(out, add...)
	return append(out, lines[at:]...)
}

// TestIncrementalMatchesFullHighlight runs random edit sequences and checks
// after every edit (and every theme switch) that the incremental colours are
// byte-identical to a full highlight.
func TestIncrementalMatchesFullHighlight(t *testing.T) {
	themes := []theme.Theme{theme.DefaultTheme(), otherTheme(t)}
	seeds := []string{"", "x", testnote.Note(120), "---\ntitle: a\n---\n# h", "```go\nfunc f() {\n```\ntext"}
	for s := 0; s < 40; s++ {
		rng := rand.New(rand.NewSource(int64(s)))
		start := seeds[s%len(seeds)]
		lines := strings.Split(start, "\n")
		h := &Highlighter{}
		th := themes[0]
		check(t, h, start, th, fmt.Sprintf("seed %d start", s))
		for step := 0; step < 50; step++ {
			if rng.Intn(25) == 0 {
				th = themes[rng.Intn(len(themes))]
			}
			var what string
			lines, what = edit(lines, rng.Intn(9), rng.Intn(1<<16), rng.Intn(1<<16), seeds[rng.Intn(len(seeds))])
			check(t, h, strings.Join(lines, "\n"), th, fmt.Sprintf("seed %d step %d: %s", s, step, what))
		}
	}
}

// FuzzIncrementalHighlight drives the same edits from fuzzer bytes: the
// start text is arbitrary, each op is three bytes (kind, position, line).
func FuzzIncrementalHighlight(f *testing.F) {
	f.Add("---\na: b\n---\n```go\nx\n```\n# h", []byte{5, 0, 0, 0, 3, 1, 3, 4, 2})
	f.Add("a\n~~~\nb\n~~~\nc", []byte{2, 1, 0, 0, 2, 4, 4, 3, 7})
	f.Add("", []byte{1, 0, 9, 7, 0, 0})
	f.Add("x\r\ny\r\n```\r\nz", []byte{4, 2, 8, 8, 0, 0})
	th := theme.DefaultTheme()
	f.Fuzz(func(t *testing.T, start string, ops []byte) {
		lines := strings.Split(start, "\n")
		h := &Highlighter{}
		check(t, h, start, th, "start")
		for i := 0; i+2 < len(ops) && i < 90; i += 3 {
			var what string
			lines, what = edit(lines, int(ops[i]), int(ops[i+1]), int(ops[i+2]), start)
			check(t, h, strings.Join(lines, "\n"), th, what)
		}
	})
}

func lineOf(lines []string, prefix string) int {
	for i, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return i
		}
	}
	return -1
}

// Typing one letter into a paragraph of a 5k-line note must recolour only
// that line, not the whole note.
func TestOneLineEditRecolorsOnlyThatLine(t *testing.T) {
	th := theme.DefaultTheme()
	lines := strings.Split(testnote.Note(5000), "\n")
	h := &Highlighter{}
	h.Markdown(strings.Join(lines, "\n"), th)
	at := lineOf(lines[2500:], "Opening paragraph") + 2500
	lines[at] = "x" + lines[at]
	check(t, h, strings.Join(lines, "\n"), th, "type")
	if got := h.Recolored(); got > 2 {
		t.Fatalf("one-line edit recoloured %d lines, want <= 2", got)
	}
}

// Typing inside a fenced block recolours that block (chroma tokenises it as
// a whole) and stops at its closing fence.
func TestEditInsideCodeBlockRecolorsOnlyTheBlock(t *testing.T) {
	th := theme.DefaultTheme()
	lines := strings.Split(testnote.Note(5000), "\n")
	h := &Highlighter{}
	h.Markdown(strings.Join(lines, "\n"), th)
	open := lineOf(lines[2500:], "```go") + 2500
	closing := lineOf(lines[open+1:], "```") + open + 1
	lines[open+1] += " // typed"
	check(t, h, strings.Join(lines, "\n"), th, "type in block")
	if got, want := h.Recolored(), closing-open+1; got != want {
		t.Fatalf("edit in a code block recoloured %d lines, want the block's %d", got, want)
	}
}

// Opening a fence near the top turns the rest of the note into code; closing
// it again restores the original colours. Both must match a full highlight.
func TestOpeningAndClosingAFenceRecolorsToTheEnd(t *testing.T) {
	th := theme.DefaultTheme()
	lines := strings.Split(testnote.Note(2000), "\n")
	h := &Highlighter{}
	h.Markdown(strings.Join(lines, "\n"), th)
	opened := insertAt(lines, 1, "~~~~")
	check(t, h, strings.Join(opened, "\n"), th, "open fence")
	if h.Recolored() < 1000 {
		t.Fatalf("opening a fence recoloured only %d lines", h.Recolored())
	}
	check(t, h, strings.Join(lines, "\n"), th, "close fence")
}

// The fence kind matters: a tilde block is not closed by backticks, so
// changing the opening fence's kind recolours everything after it.
func TestChangingFenceKindRecolors(t *testing.T) {
	th := theme.DefaultTheme()
	doc := []string{"a", "```", "x = 1", "```", "# after", "~~~", "y", "~~~", "tail"}
	h := &Highlighter{}
	check(t, h, strings.Join(doc, "\n"), th, "start")
	doc[1] = "~~~"
	check(t, h, strings.Join(doc, "\n"), th, "backticks to tildes")
	doc[1] = "````"
	check(t, h, strings.Join(doc, "\n"), th, "longer fence")
	doc[3] = "~~~"
	check(t, h, strings.Join(doc, "\n"), th, "mismatched closer")
}

// Front matter only counts with a closing delimiter: adding that closer far
// below the first line recolours from the top.
func TestFrontMatterClosedLaterRecolorsFromTop(t *testing.T) {
	th := theme.DefaultTheme()
	doc := []string{"---", "title: a", "body: b", "# h", "text"}
	h := &Highlighter{}
	check(t, h, strings.Join(doc, "\n"), th, "unclosed")
	doc[2] = "..."
	check(t, h, strings.Join(doc, "\n"), th, "closed with ...")
	doc[0] = "--- "
	check(t, h, strings.Join(doc, "\n"), th, "opener broken")
	doc[0] = "---"
	doc = append(doc[:2], doc[3:]...)
	check(t, h, strings.Join(doc, "\n"), th, "closer deleted")
}

// Existing behaviours that share the segment scanner keep their colours.
func TestMarkdownKeepsPreviousBehaviour(t *testing.T) {
	th := theme.DefaultTheme()
	for _, doc := range []string{
		"",
		"---",
		"---\nkey: v",                       // unclosed front matter is a rule + text
		"---\nkey: v\n...\n# h",             // "..." closes front matter
		"```go\nfunc f() {}",                // unclosed fence runs to the end
		"~~~\n```\nstill code\n~~~\nout",    // backticks do not close tildes
		"````\nx\n```\nafter",               // a longer fence closes on three
		"```\nx\n````\nafter",               // a longer closer closes too
		"```\nx\n``` js\nafter",             // so does a closer with text
		"a\r\n```c\r\nint x;\r\n```\r\n# h", // CRLF
		"   ```py\n  print(1)\n   ```",      // indented fences
		testnote.Note(300),
	} {
		if got, want := Markdown(doc, th), legacyMarkdown(doc, th); !reflect.DeepEqual(got, want) {
			t.Errorf("Markdown(%q) changed:\n got %v\nwant %v", doc, got, want)
		}
	}
}

// A new theme recolours every line even when the text did not change.
func TestThemeChangeRecolorsEverything(t *testing.T) {
	text := testnote.Note(200)
	h := &Highlighter{}
	h.Markdown(text, theme.DefaultTheme())
	check(t, h, text, otherTheme(t), "theme")
	if h.Recolored() != 200 {
		t.Fatalf("theme change recoloured %d lines, want 200", h.Recolored())
	}
	h.Markdown(text, otherTheme(t))
	if h.Recolored() != 0 {
		t.Fatalf("unchanged text and theme recoloured %d lines, want 0", h.Recolored())
	}
}

func BenchmarkIncrementalTypeKey(b *testing.B) {
	th := theme.DefaultTheme()
	lines := strings.Split(testnote.Note(5000), "\n")
	h := &Highlighter{}
	h.Markdown(strings.Join(lines, "\n"), th)
	at := lineOf(lines[2500:], "Opening paragraph") + 2500
	b.ReportAllocs()
	for b.Loop() {
		lines[at] += "x"
		h.Markdown(strings.Join(lines, "\n"), th)
	}
}
