package preview_test

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// displayNotes resolves the one embed the partition notes use.
func displayNotes(target string) (preview.Note, bool) {
	if target == "other" {
		return preview.Note{ID: "other", Title: "Other", Content: "# Inside\n\n- embedded\n"}, true
	}
	return preview.Note{}, false
}

// displayOf compiles md the way the app does and returns the partition and
// the compiled body it has to add up to.
func displayOf(md string, foldAll bool) (preview.Display, string) {
	const paneWidth, textWidth = 80, 72 // margin 4 on each side
	m := preview.New(theme.DefaultTheme())
	m.SetSize(paneWidth, 24)
	m.SetNotes(displayNotes)
	if foldAll {
		m.FoldAll(true)
	}
	m.SetPage(core.Page{ID: "notes/a.md", Title: "a", Content: md})
	cal := preview.CalendarView{Notes: displayNotes, FoldAll: foldAll}
	if foldAll {
		cal.Folded = map[string]bool{}
	}
	body, _, _ := preview.CompileBlocks(md, theme.DefaultTheme(), preview.DefaultLook(), textWidth, "", cal)
	return m.Display(), body
}

// checkPartition pins the invariant of the contract: blocks are sorted,
// disjoint, cover every source line, and their rows are the compiled body.
func checkPartition(t testing.TB, md string, d preview.Display, body string) {
	t.Helper()
	if d.Failed {
		t.Fatalf("compile failed for %q", md)
	}
	lineCount := strings.Count(md, "\n") + 1
	at, row := 0, 0
	var rows []string
	for i, b := range d.Blocks {
		if b.Line != at || b.End <= b.Line {
			t.Fatalf("block %d covers [%d,%d), want it to start at %d, in %q: %+v", i, b.Line, b.End, at, md, spans(d))
		}
		if b.Row != row {
			t.Fatalf("block %d starts at row %d, want %d, in %q", i, b.Row, row, md)
		}
		for _, r := range b.Rows {
			if strings.Contains(r, "\n") {
				t.Fatalf("block %d has a row with a line break: %q", i, r)
			}
		}
		at, row = b.End, row+len(b.Rows)
		rows = append(rows, b.Rows...)
	}
	if at != lineCount {
		t.Fatalf("blocks end at line %d, want %d, in %q: %+v", at, lineCount, md, spans(d))
	}
	if got := strings.Join(rows, "\n"); got != body {
		t.Fatalf("rows do not add up to the body of %q:\n%q\nwant\n%q", md, got, body)
	}
}

func spans(d preview.Display) [][2]int {
	out := make([][2]int, len(d.Blocks))
	for i, b := range d.Blocks {
		out[i] = [2]int{b.Line, b.End}
	}
	return out
}

func TestDisplayPartition(t *testing.T) {
	cases := []struct {
		name string
		md   string
		fold bool
		want [][2]int
	}{
		{name: "empty note", md: "", want: [][2]int{{0, 1}}},
		{name: "plain paragraphs", md: "a\n\nb", want: [][2]int{{0, 1}, {1, 2}, {2, 3}}},
		{name: "multi-line paragraph", md: "a\nb\nc", want: [][2]int{{0, 3}}},
		{name: "list continuation then text", md: "- a\n  b\nc", want: [][2]int{{0, 3}}},
		{name: "list continuation then blanks", md: "- a\n  b\n\n", want: [][2]int{{0, 2}, {2, 4}}},
		{name: "nested lists", md: "- a\n  - b\n    - c\n- d", want: [][2]int{{0, 3}, {3, 4}}},
		{name: "table", md: "| a | b |\n|---|---|\n| 1 | 2 |\n\nx", want: [][2]int{{0, 3}, {3, 4}, {4, 5}}},
		{name: "callout with body", md: "> [!NOTE]\n> body\n> more\n\nx", want: [][2]int{{0, 3}, {3, 4}, {4, 5}}},
		{name: "two callouts in a row", md: "> [!NOTE]\n> a\n> [!TIP]\n> b", want: [][2]int{{0, 2}, {2, 4}}},
		{name: "plain quote lines stay apart", md: "> a\n> b", want: [][2]int{{0, 1}, {1, 2}}},
		{name: "fenced code", md: "```go\nx\n\ny\n```\nz", want: [][2]int{{0, 5}, {5, 6}}},
		{name: "unclosed fence", md: "```go\nx", want: [][2]int{{0, 2}}},
		{name: "math", md: "$$\na^2\n$$\n\ntext", want: [][2]int{{0, 3}, {3, 4}, {4, 5}}},
		{name: "image line", md: "![alt](missing.png)\nnext", want: [][2]int{{0, 1}, {1, 2}}},
		{name: "embed", md: "![[other]]\nnext", want: [][2]int{{0, 1}, {1, 2}}},
		{name: "board", md: "```board\n## Todo\n- a\n```", want: [][2]int{{0, 4}}},
		{name: "calendar", md: "```calendar\n2026-11-10 x\n```\nt", want: [][2]int{{0, 3}, {3, 4}}},
		{name: "form", md: "```form\nname: text\n```", want: [][2]int{{0, 3}}},
		{name: "columns", md: "```columns\n- a\n\n+++\n# b\n```\nc", want: [][2]int{{0, 6}, {6, 7}}},
		{name: "front matter only", md: "---\ntitle: x\n---", want: [][2]int{{0, 3}}},
		{name: "front matter and a final newline", md: "---\ntitle: x\n---\n", want: [][2]int{{0, 3}, {3, 4}}},
		{name: "front matter plus body", md: "---\ntitle: x\n---\n\nhello", want: [][2]int{{0, 3}, {3, 4}, {4, 5}}},
		{name: "cover", md: "---\ncover: missing.png\n---\ntext", want: [][2]int{{0, 3}, {3, 4}}},
		{name: "heading without a blank line", md: "# H\ntext", want: [][2]int{{0, 1}, {1, 2}}},
		{name: "several blank lines", md: "a\n\n\n\nb", want: [][2]int{{0, 1}, {1, 4}, {4, 5}}},
		{name: "leading blank lines", md: "\n\na", want: [][2]int{{0, 2}, {2, 3}}},
		{name: "comment joins what follows", md: "a\n\n<!-- c -->\nb", want: [][2]int{{0, 1}, {1, 2}, {2, 4}}},
		{name: "comment then blanks", md: "a\n<!-- c -->\n\nb", want: [][2]int{{0, 1}, {1, 3}, {3, 4}}},
		{name: "tag-only line", md: "<div>\nb", want: [][2]int{{0, 2}}},
		{name: "block id line", md: "x\n\n^abc\nb", want: [][2]int{{0, 1}, {1, 2}, {2, 4}}},
		{name: "stripped line at the end", md: "a\n<!-- c -->", want: [][2]int{{0, 2}}},
		{name: "only a comment", md: "<!-- c -->", want: [][2]int{{0, 1}}},
		{name: "folded section with blank lines", md: "# A\nx\n\ny\n\n# B\nz", fold: true, want: [][2]int{{0, 5}, {5, 7}}},
		{name: "folded section to the end", md: "# A\n\n```\n# not a heading\n```\n\n", fold: true, want: [][2]int{{0, 7}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, body := displayOf(c.md, c.fold)
			checkPartition(t, c.md, d, body)
			if got := spans(d); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("lines %v, want %v", got, c.want)
			}
		})
	}
}

// The rows between two blocks belong to the blank source lines between them,
// and to the previous block when the source has no blank line there.
func TestDisplaySpacingRows(t *testing.T) {
	d, _ := displayOf("# H\ntext", false)
	last := d.Blocks[0].Rows[len(d.Blocks[0].Rows)-1]
	if last != "" || len(d.Blocks[1].Rows) != 1 {
		t.Fatalf("spacing should be the heading's tail: %+v", d.Blocks)
	}

	d, _ = displayOf("# H\n\n\ntext", false)
	if rows := d.Blocks[1].Rows; len(d.Blocks) != 3 || len(rows) != 1 || rows[0] != "" {
		t.Fatalf("the blank run should own the spacing row: %+v", d.Blocks)
	}
	if h := d.Blocks[0].Rows; h[len(h)-1] == "" {
		t.Fatalf("the heading should not keep the spacing row: %q", h)
	}
}

// Rows before the first body row belong to the front matter's entry.
func TestDisplayHeaderRowsBelongToTheFrontMatter(t *testing.T) {
	d, _ := displayOf("---\ncover: missing.png\ntitle: x\n---\n# Body", false)
	head := d.Blocks[0]
	plain := ansi.Strip(strings.Join(head.Rows, "\n"))
	if head.Line != 0 || head.End != 4 || !strings.Contains(plain, "cover") || !strings.Contains(plain, "title") {
		t.Fatalf("header rows lost: %+v\n%s", head, plain)
	}
}

func TestDisplayFoldedSection(t *testing.T) {
	md := "intro\n\n# A\nx\n\ny\n\nz\n# B\nw"
	cal := preview.CalendarView{Folded: map[string]bool{"h:#A#1": true}}
	out, _, blocks := preview.CompileBlocks(md, theme.DefaultTheme(), preview.DefaultLook(), 72, "", cal)
	if !strings.Contains(ansi.Strip(out), "… 3 lines") {
		t.Fatalf("the label must keep counting non-blank lines:\n%s", ansi.Strip(out))
	}
	found := false
	for _, b := range blocks {
		if b.Line == 2 {
			found = true
			if b.End != 8 {
				t.Fatalf("folded section ends at %d, want its true end 8", b.End)
			}
		}
	}
	if !found {
		t.Fatalf("no block for the folded heading: %+v", blocks)
	}

	d, _ := displayOf("# A\nx\n\ny\n\n# B\nz", true)
	for i, b := range d.Blocks {
		if !b.Folded {
			t.Errorf("block %d should be folded: %+v", i, b)
		}
	}
	d, _ = displayOf("# A\nx", false)
	for i, b := range d.Blocks {
		if b.Folded {
			t.Errorf("block %d should not be folded: %+v", i, b)
		}
	}
}

func TestDisplayFooterHoldsBacklinksAndMentions(t *testing.T) {
	m := preview.New(theme.DefaultTheme())
	m.SetSize(80, 24)
	m.SetPage(core.Page{ID: "notes/a.md", Content: "hello\n\n- [ ] box"})
	m.SetBacklinks([]core.Backlink{{ID: "b", Title: "Linker", Text: "see a"}})
	m.SetMentions("notes/a.md", []core.UnlinkedMention{{ID: "c", Title: "Mentioner", Text: "a", Count: 1}})

	d := m.Display()
	if d.PageID != "notes/a.md" {
		t.Fatalf("PageID = %q", d.PageID)
	}
	body := 0
	for _, b := range d.Blocks {
		body += len(b.Rows)
	}
	if d.FooterRow != body {
		t.Fatalf("FooterRow = %d, want %d", d.FooterRow, body)
	}
	plain := ansi.Strip(strings.Join(d.Footer, "\n"))
	for _, want := range []string{"Linked from 1 note", "Linker", "Unlinked mentions in 1 note", "Mentioner"} {
		if !strings.Contains(plain, want) {
			t.Errorf("footer misses %q:\n%s", want, plain)
		}
	}
	// A footer row is an absolute compiled row like any other.
	h, ok := m.HitAtRow(d.FooterRow+2, 0)
	if !ok || h.Kind != preview.HitNoteLink || h.Arg != "b" {
		t.Fatalf("footer hit = %+v, %v", h, ok)
	}
}

// The footer's own Compile used to open a second image scope, which released
// the picture the body had just drawn.
func TestDisplayFooterKeepsThePicturesOfTheBody(t *testing.T) {
	dir := t.TempDir()
	md := picture(t, dir, 80, 40)
	s := graphics(t)

	m := preview.New(theme.DefaultTheme())
	m.SetSize(80, 24)
	m.SetBaseDir(dir)
	m.SetPage(core.Page{ID: "notes/a.md", Title: "a", Content: md})
	m.SetBacklinks([]core.Backlink{{ID: "b", Title: "Linker", Text: "see a"}})
	m.SetMentions("notes/a.md", []core.UnlinkedMention{{ID: "c", Title: "Mentioner", Text: "a", Count: 1}})

	for _, seq := range s.all() {
		if strings.Contains(seq, "a=d,d=I,i=") {
			t.Fatalf("the picture was released while still on the page: %q", seq)
		}
	}
}

func TestDisplayFailedCompile(t *testing.T) {
	m := preview.New(theme.DefaultTheme())
	m.SetSize(80, 24)
	m.SetPage(core.Page{ID: "notes/a.md", Content: "![[boom]]"})
	if m.Display().Failed {
		t.Fatal("a healthy compile must not be Failed")
	}
	m.SetNotes(func(string) (preview.Note, bool) { panic("boom") })
	d := m.Display()
	if !d.Failed {
		t.Fatalf("a panic must set Failed: %+v", d)
	}
	if len(d.Blocks) != 0 {
		t.Fatalf("a failed compile has no partition: %+v", d.Blocks)
	}
	m.SetNotes(nil)
	if m.Display().Failed {
		t.Fatal("the next good compile must clear Failed")
	}
}

func TestDisplayGeneration(t *testing.T) {
	a, b := preview.New(theme.DefaultTheme()), preview.New(theme.DefaultTheme())
	a.SetSize(80, 24)
	seen := map[uint64]bool{}
	last := uint64(0)
	step := func(m *preview.Model) {
		t.Helper()
		g := m.Display().Generation
		if g <= last || seen[g] {
			t.Fatalf("generation %d after %d (seen: %v)", g, last, seen[g])
		}
		seen[g], last = true, g
	}
	step(&a)
	b.SetSize(80, 24)
	step(&b)
	a.SetContent("x")
	step(&a)
	a.SetTheme(theme.DefaultTheme())
	step(&a)
	a.FoldAll(true)
	step(&a)
	b.ShiftCalendars(1, false)
	step(&b)
	a.SetBacklinks([]core.Backlink{{ID: "b", Title: "L", Text: "t"}})
	step(&a)
	if again := a.Display().Generation; again != a.Display().Generation {
		t.Fatalf("reading the display must not bump it: %d", again)
	}
}

// tallNote has a to-do far below the first screen; it returns the note and
// the compiled row of its box.
func tallNote(t *testing.T) (preview.Model, int) {
	t.Helper()
	md := strings.Repeat("para\n\n", 30) + "- [ ] box"
	m := preview.New(theme.DefaultTheme())
	m.SetSize(80, 5)
	m.SetPage(core.Page{ID: "notes/a.md", Content: md})
	d := m.Display()
	return m, d.Blocks[len(d.Blocks)-1].Row
}

func TestHitAtRowIgnoresTheScroll(t *testing.T) {
	m, row := tallNote(t)
	before, ok := m.HitAtRow(row, 0)
	if !ok || before.Kind != preview.HitCheckbox {
		t.Fatalf("no box at row %d: %+v", row, before)
	}
	m.ScrollBy(12)
	after, ok := m.HitAtRow(row, 0)
	if !ok || !reflect.DeepEqual(before, after) {
		t.Fatalf("the scroll moved the hit: %+v != %+v (%v)", after, before, ok)
	}
	if h, ok := m.HitAtRow(row-12, 0); ok {
		t.Fatalf("a hit above the box: %+v", h)
	}
}

func TestPressRowCheckboxIsAClick(t *testing.T) {
	m, row := tallNote(t)
	cmd, ok := m.PressRow(row, 0)
	if !ok || cmd == nil {
		t.Fatalf("PressRow = %v, %v", cmd, ok)
	}
	got := cmd()

	m.SetSize(80, row+10) // the whole note on screen; the page margin is 4
	_, clickCmd := m.Update(tea.MouseMsg{X: 4, Y: row, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if clickCmd == nil {
		t.Fatal("the click reported nothing")
	}
	if want := clickCmd(); !reflect.DeepEqual(got, want) {
		t.Fatalf("PressRow sent %+v, a click sends %+v", got, want)
	}
	if _, isHit := got.(preview.HitMsg); !isHit {
		t.Fatalf("want a HitMsg, got %T", got)
	}
}

func TestPressRowFoldToggles(t *testing.T) {
	m := preview.New(theme.DefaultTheme())
	m.SetSize(80, 24)
	m.SetPage(core.Page{ID: "notes/a.md", Content: "# A\nx"})
	cmd, ok := m.PressRow(0, 0)
	if !ok || cmd != nil {
		t.Fatalf("PressRow = %v, %v, want handled here", cmd, ok)
	}
	if !m.Folded("h:#A#1") {
		t.Fatal("the section did not fold")
	}
	if _, ok := m.PressRow(0, 0); !ok || m.Folded("h:#A#1") {
		t.Fatal("the section did not unfold")
	}
}

func TestPressRowOffAHit(t *testing.T) {
	m, row := tallNote(t)
	gen := m.Display().Generation
	for _, p := range [][2]int{{row, 40}, {0, 0}, {row + 100, 0}, {-1, 0}} {
		if cmd, ok := m.PressRow(p[0], p[1]); ok || cmd != nil {
			t.Errorf("PressRow(%d, %d) = %v, %v, want no hit", p[0], p[1], cmd, ok)
		}
	}
	if m.Display().Generation != gen {
		t.Fatal("a press on nothing must not recompile")
	}
}

// displayFragments are the pieces FuzzDisplayPartition glues into notes.
var displayFragments = []string{
	"plain", "two\nlines", "- a\n  b", "- a\n  - b\n    c", "- [ ] box\n  more", "1. one\n   two",
	"# H", "## Sub", "### Deep\ntext", "---", "> q", "> [!NOTE]", "> [!TIP] x\n> y", "> z",
	"| a | b |\n|---|---|\n| 1 | 2 |", "```go\nx\n\n# no\n```", "```", "~~~\n", "$$\na\n$$", "$$ b $$", "$$",
	"![a](missing.png)", "![[other]]", "![[nope]]", "<!-- c -->", "<div>", "</p>", "^abc", "text ^abc",
	"```board\n## T\n- a\n```", "```calendar\n```", "```form\nn: text\n```", "```columns\na\n+++\n# b\n```",
	"```chart\na: 1\n```", "```flow\na -> b\n```", "   ", "  - nested", "    deep", "\t- tab",
}

// FuzzDisplayPartition glues random fragments and front matter into notes:
// each byte picks a fragment, its top bits how it is joined to the next.
func FuzzDisplayPartition(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3}, false)
	f.Add([]byte{6, 0x47, 8, 0x80, 24}, true)
	f.Add([]byte{0xFF, 0x7F, 0x3F, 12, 13, 13, 14}, false)
	f.Fuzz(func(t *testing.T, data []byte, fold bool) {
		var b strings.Builder
		if len(data) > 0 && data[0]%5 == 0 {
			b.WriteString("---\ntitle: x\n---\n")
		}
		for _, p := range data {
			b.WriteString(displayFragments[int(p&0x3F)%len(displayFragments)])
			switch p >> 6 {
			case 0:
				b.WriteString("\n")
			case 1:
				b.WriteString("\n\n")
			case 2:
				b.WriteString("\n\n\n")
			}
		}
		md := b.String()
		d, body := displayOf(md, fold)
		checkPartition(t, md, d, body)
	})
}
