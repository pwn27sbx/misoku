package app_test

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/kitty"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// prettyNote has one block of each shape the tests move the cursor over; its
// last line is plain, so G puts the cursor outside all of them.
const prettyNote = "# Title\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n> [!note] Head\n> body\n\nend"

// prettyAt opens body in the view (space p) with the cursor on its last line.
func prettyAt(t *testing.T, body string) *harness {
	t.Helper()
	h := openNoteWith(t, body)
	h.keys(" p")
	h.keys("G")
	return h
}

// wantOnScreen fails unless every piece is on screen and none of absent is.
func wantOnScreen(t *testing.T, v string, shown, absent []string) {
	t.Helper()
	for _, s := range shown {
		if !strings.Contains(v, s) {
			t.Fatalf("%q should be on screen:\n%s", s, v)
		}
	}
	for _, s := range absent {
		if strings.Contains(v, s) {
			t.Fatalf("%q should not be on screen:\n%s", s, v)
		}
	}
}

// paneRows are the note pane's rows: what is right of the explorer's divider,
// without the tab line and the statusline.
func paneRows(v string) []string {
	rows := strings.Split(v, "\n")
	rows = rows[1 : len(rows)-1]
	for i, row := range rows {
		r := []rune(row)
		rows[i] = string(r[min(app.SidebarWidth+1, len(r)):])
	}
	return rows
}

// hasTildes reports whether the editor's end-of-buffer rows are on screen.
func hasTildes(v string) bool {
	for _, row := range strings.Split(v, "\n") {
		if strings.Contains(row, "│~ ") {
			return true
		}
	}
	return false
}

// inReadingView tells the view from the split (two dividers) and from the
// editor view (end-of-buffer rows, on a note shorter than the screen).
func inReadingView(v string) bool {
	return len(dividers(v, bodyRow)) == 1 && !hasTildes(v)
}

// S1: the view edits, and the statusline says the real mode.
func TestPrettyViewEditsAndShowsTheRealMode(t *testing.T) {
	h := openNoteWith(t, "hello\nworld")
	h.keys(" p")
	if got := mode(h.view()); got != "NORMAL" {
		t.Fatalf("mode %q, want NORMAL", got)
	}
	h.keys("i")
	if got := mode(h.view()); got != "INSERT" {
		t.Fatalf("mode %q after i, want INSERT", got)
	}
	h.keys("abc")
	h.key(tea.KeyEsc)
	if got := mode(h.view()); got != "NORMAL" {
		t.Fatalf("mode %q after esc, want NORMAL", got)
	}
	if got := saved(t, h); got != "abchello\nworld" {
		t.Fatalf("typing should change the note, got %q", got)
	}
}

// S1: dd and o edit; since S15 they act on the cursor's whole block, here
// one-line paragraphs: dd takes the blank line after it and lands on the
// next block, and o opens a block of its own.
func TestPrettyViewDeletesAndOpensLines(t *testing.T) {
	h := openNoteWith(t, "one\n\ntwo\n\nthree")
	h.keys(" p")
	h.keys("dd")
	if got := saved(t, h); got != "two\n\nthree" {
		t.Fatalf("dd left %q", got)
	}
	h.keys("onew")
	h.key(tea.KeyEsc)
	if got := saved(t, h); got != "two\n\nnew\n\nthree" {
		t.Fatalf("o left %q", got)
	}
}

// S1: u restores the text. Since S15 dd takes the cursor's whole block:
// both lines of the paragraph.
func TestPrettyViewUndoRestores(t *testing.T) {
	h := openNoteWith(t, "one\ntwo")
	h.keys(" p")
	h.keys("dd")
	if got := saved(t, h); got != "" {
		t.Fatalf("dd left %q", got)
	}
	h.keys("u")
	if got := saved(t, h); got != "one\ntwo" {
		t.Fatalf("u left %q", got)
	}
}

// S3: with the cursor elsewhere, each block is what the preview compiles at
// the editor's text width, row for row.
func TestPrettyViewDrawsBlocksAsThePreviewDoes(t *testing.T) {
	h := prettyAt(t, prettyNote)
	v := h.view()
	p := preview.New(theme.DefaultTheme())
	p.SetSize(termW-app.SidebarWidth-1, termH)
	p.SetPage(core.Page{ID: "doc.md", Title: "doc", Content: prettyNote})
	d := p.Display()
	for _, b := range d.Blocks[:len(d.Blocks)-1] {
		for _, row := range b.Rows {
			if text := strings.TrimRight(ansi.Strip(row), " "); text != "" && !strings.Contains(v, strings.TrimLeft(text, " ")) {
				t.Fatalf("the compiled row %q (line %d) is not on screen:\n%s", text, b.Line, v)
			}
		}
	}
	wantOnScreen(t, v, []string{"end"}, []string{"# Title", "|---|", "[!note]"})
}

// S3: INSERT in a table draws all of it raw, one row per source line (since
// S15 NORMAL keeps it rendered).
func TestPrettyViewTableRevealsWhole(t *testing.T) {
	h := prettyAt(t, prettyNote)
	h.keys("gg")
	h.keys("j") // block by block: onto the table
	h.keys("i")
	v := h.view()
	wantOnScreen(t, v, []string{"| a | b |", "|---|---|", "| 1 | 2 |", "Head"}, []string{"# Title", "[!note]"})
	_, top, _ := textAt(t, v, "| a | b |")
	_, sep, _ := textAt(t, v, "|---|---|")
	_, last, _ := textAt(t, v, "| 1 | 2 |")
	if sep != top+1 || last != top+2 {
		t.Fatalf("the raw table should fill rows %d, %d, %d; got %d, %d:\n%s", top, top+1, top+2, sep, last, v)
	}
}

// S3: a blank line holding the cursor in INSERT is one raw row and reveals
// nothing else. NORMAL's j steps over blank lines (S15): ↓ in INSERT goes
// there.
func TestPrettyViewBlankLineIsRawAlone(t *testing.T) {
	h := openNoteWith(t, "**one**\n\n**two**\n\n**three**")
	h.keys(" p")
	h.keys("i")
	h.key(tea.KeyDown)
	wantOnScreen(t, h.view(), []string{"one", "two", "three"}, []string{"**one**", "**two**", "**three**"})
	h.keys("x")
	h.key(tea.KeyEsc)
	if got := saved(t, h); got != "**one**\nx\n**two**\n\n**three**" {
		t.Fatalf("the blank line should take the typing, got %q", got)
	}
}

// S4: VISUAL and V-LINE reveal every block the selection touches.
func TestPrettyViewSelectionRevealsTouchedBlocks(t *testing.T) {
	for _, start := range []string{"v", "V"} {
		h := openNoteWith(t, "**one**\n\n**two**\n\n**three**")
		h.keys(" p")
		h.keys(start + "jj")
		wantOnScreen(t, h.view(), []string{"**one**", "**two**", "three"}, []string{"**three**"})
	}
}

// S4: a drag leaves the reveal as it was; the release applies it.
func TestPrettyViewDragFreezesTheReveal(t *testing.T) {
	h := prettyAt(t, "**one**\n\n**two**\n\nend")
	v := h.view()
	x, y, _ := textAt(t, v, "one")
	h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if got := paneRows(h.view()); strings.Join(got, "\n") != strings.Join(paneRows(v), "\n") {
		t.Fatalf("the press moved the rows:\n%s\nwas:\n%s", strings.Join(got, "\n"), v)
	}
	h.send(tea.MouseMsg{X: x + 1, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
	if got := h.view(); strings.Contains(got, "**one**") {
		t.Fatalf("the drag revealed a block before the release:\n%s", got)
	}
	h.send(tea.MouseMsg{X: x + 1, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	wantOnScreen(t, h.view(), []string{"**one**", "two"}, []string{"**two**"})
}

// S11: a compile of another note is never drawn over this one.
func TestPrettyViewIgnoresAnotherNotesCompile(t *testing.T) {
	h := openNoteWith(t, "**mine**\n\nend")
	h.keys(" p")
	h.keys("G")
	h.m = app.ShowInPreview(h.m, core.Page{ID: "other.md", Title: "other", Content: "**theirs**\n\nfoot"})
	h.keys("0")
	wantOnScreen(t, h.view(), []string{"**mine**", "end"}, []string{"theirs", "foot"})
}

// S11: Enter changes the line count: the display follows in the same update
// and the text above the cursor stays where it was.
func TestPrettyViewEnterKeepsTheScreenAndTheDisplay(t *testing.T) {
	var b strings.Builder
	const sections = 25
	for i := range sections {
		fmt.Fprintf(&b, "para %d\n\n| h%d |\n|---|\n| v |\n\n", i, i)
	}
	b.WriteString("end")
	h := prettyAt(t, b.String())
	h.keys(strings.Repeat("k", 6)) // block by block, up three sections, onto a paragraph
	h.keys("A")
	before := paneRows(h.view())
	h.key(tea.KeyEnter)
	v := h.view()
	after := paneRows(v)
	for y := 1; y < 6; y++ {
		if before[y] != after[y] {
			t.Fatalf("row %d moved after Enter:\n%q\nbecame\n%q\n%s", y, before[y], after[y], v)
		}
	}
	wantOnScreen(t, v, nil, []string{"|---|"})
	h.key(tea.KeyEsc)
	if got := saved(t, h); !strings.Contains(got, "para 22\n\n\n| h22 |") {
		t.Fatalf("Enter should split the line, got ...%q", got[len(got)-200:])
	}
}

// S9: on a long note a typed Enter leaves its compile pending: rendered rows
// stay and the entries below the anchor move with the text.
func TestPrettyViewPendingCompileShiftsTheRows(t *testing.T) {
	body := longNote("# Title\n**Bold**")
	h := openNoteWith(t, body)
	h.keys(" p")
	h.keys("A")
	compiles := app.PreviewCompiles(h.m)
	h.key(tea.KeyEnter)
	if got := app.PreviewCompiles(h.m); got != compiles {
		t.Fatalf("a typed Enter should wait for the debounce: %d compiles, was %d", got, compiles)
	}
	v := h.view()
	wantOnScreen(t, v, []string{"# Title", "Bold", "filler 0"}, []string{"**Bold**", "- filler 0"})
	h.keys("x")
	wantOnScreen(t, h.view(), []string{"x", "Bold"}, []string{"**Bold**"})

	// The compile reads "x" and "**Bold**" as one paragraph, the cursor's.
	h.send(app.PreviewTick(app.PreviewGen(h.m)))
	wantOnScreen(t, h.view(), []string{"Title", "x", "**Bold**"}, []string{"# Title"})
}

// S9: undo on a long note compiles at once, so no stale row shows.
func TestPrettyViewUndoCompilesAtOnce(t *testing.T) {
	h := openNoteWith(t, longNote("# Title\n**Bold**"))
	h.keys(" p")
	h.keys("jdd")
	wantOnScreen(t, h.view(), nil, []string{"Bold"})
	compiles := app.PreviewCompiles(h.m)
	h.keys("u")
	if got := app.PreviewCompiles(h.m); got != compiles+1 {
		t.Fatalf("undo should compile at once: %d compiles, was %d", got, compiles)
	}
	wantOnScreen(t, h.view(), []string{"Bold"}, nil)
}

// S9: a cursor move swaps the reveal without a compile.
func TestPrettyViewCursorMoveDoesNotCompile(t *testing.T) {
	h := prettyAt(t, prettyNote)
	compiles := app.PreviewCompiles(h.m)
	h.keys("ggjjjj")
	if got := app.PreviewCompiles(h.m); got != compiles {
		t.Fatalf("moving compiled: %d, was %d", got, compiles)
	}
}

// S11: with the explorer focused the rendered rows are greyed like the
// preview's, and no raw Markdown takes their place.
func TestPrettyViewDimsRenderedRows(t *testing.T) {
	th := theme.DefaultTheme()
	trueColour(t)
	h := prettyAt(t, "# Title\n\nend")
	h.send(tea.KeyMsg{Type: tea.KeyTab})
	if got := mode(h.view()); got != "EXPLORER" {
		t.Fatalf("setup: mode %q", got)
	}
	v := h.colourView()
	wantOnScreen(t, ansi.Strip(v), []string{"Title"}, []string{"# Title"})
	grey := hexOf(th.Grey)
	for _, row := range strings.Split(v, "\n") {
		if !strings.Contains(ansi.Strip(row), "Title") {
			continue
		}
		for _, c := range screenCells(row) {
			if c.r == 'T' && c.fg != grey {
				t.Fatalf("the rendered row is %s, want the grey %s", c.fg, grey)
			}
		}
		return
	}
	t.Fatal("no Title row")
}

// S11: no ~ rows in this view; the editor keeps them.
func TestPrettyViewHasNoEndOfBufferRows(t *testing.T) {
	h := openNoteWith(t, "short")
	h.keys(" p")
	if v := h.view(); hasTildes(v) {
		t.Fatalf("~ rows in the view:\n%s", v)
	}
	h.keys(" r")
	if v := h.view(); !hasTildes(v) {
		t.Fatalf("the editor should keep its ~ rows:\n%s", v)
	}
}

// imageSink records the graphics sequences sent to the terminal.
type imageSink struct {
	mu   sync.Mutex
	seqs []string
}

func (s *imageSink) Send(seqs ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seqs = append(s.seqs, seqs...)
	return nil
}

func (s *imageSink) deletes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, seq := range s.seqs {
		if strings.Contains(seq, "a=d") {
			n++
		}
	}
	return n
}

// S7: a picture stays in the terminal when only the cursor moves.
func TestPrettyViewPictureSurvivesACursorMove(t *testing.T) {
	sink := &imageSink{}
	preview.SetImageMode(kitty.Graphics)
	preview.SetImageSink(sink)
	t.Cleanup(func() {
		preview.SetImageMode(kitty.Blocks)
		preview.SetImageSink(nil)
		preview.ClearImages()
	})

	store := newTestStore(t)
	if err := os.MkdirAll(filepath.Join(store.Root(), "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(store.Root(), "assets", "pic.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 80, 40))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := store.SaveAs("", "doc", "![A picture](assets/pic.png)\n\ntext\n\nmore"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys(" p")
	h.keys("jj")
	placeholder := string(rune(0x10EEEE))
	if v := h.view(); !strings.Contains(v, placeholder) {
		t.Fatalf("the picture should be drawn:\n%s", v)
	}
	deletes := sink.deletes()
	h.keys("jjkk")
	if v := h.view(); !strings.Contains(v, placeholder) {
		t.Fatalf("the picture should stay after moving:\n%s", v)
	}
	if got := sink.deletes(); got != deletes {
		t.Fatalf("moving the cursor released pictures: %d deletes, was %d", got, deletes)
	}
}

// S1: the "/" menu opens under the cursor's raw line, past blocks drawn in
// another height.
func TestPrettyViewSlashMenuFollowsTheCursor(t *testing.T) {
	h := prettyAt(t, prettyNote)
	h.keys("o/")
	v := h.view()
	rows := strings.Split(v, "\n")
	y := -1
	for i, row := range rows {
		if strings.HasSuffix(strings.TrimRight(row, " "), "/") && i > 0 && i < len(rows)-2 && strings.Contains(row, "│") {
			y = i
		}
	}
	if y < 0 {
		t.Fatalf("no raw / line on screen:\n%s", v)
	}
	if !strings.ContainsAny(rows[y+1], "╭┌─") {
		t.Fatalf("the menu should open on the row under the cursor (%d):\n%s", y, v)
	}
}

// S4: a wheel during a text drag ends the drag in the app and in the editor
// alike, keeping the selection as the split does: the reveal is not left
// frozen until the next press.
func TestPrettyViewWheelDuringDragUnfreezesTheReveal(t *testing.T) {
	h := prettyAt(t, "**one**\n\n**two**\n\nend")
	x, y, _ := textAt(t, h.view(), "one")
	h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	h.send(tea.MouseMsg{X: x + 1, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
	h.send(tea.MouseMsg{X: x + 1, Y: y, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	wantOnScreen(t, h.view(), []string{"**one**", "two"}, []string{"**two**"})
}

// goToLine walks the cursor down with j until it reaches line n.
func goToLine(h *harness, n int) {
	h.t.Helper()
	h.keys("gg")
	for i := 0; i < 4*n; i++ {
		if l, _, _ := app.NoteCursor(h.m); l >= n {
			break
		}
		h.keys("j")
	}
	if l, _, _ := app.NoteCursor(h.m); l != n {
		h.t.Fatalf("goto %d landed on %d", n, l)
	}
}

// S8: with the cursor on the last line of a raw code block whose top is above
// the screen, ↓ twice renders the block again above the top; the rows below
// it keep their screen rows. The block is raw in INSERT (S15), where the
// arrows go line by line.
func TestPrettyViewRevealAboveTheTopDoesNotJump(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, "para %d\n\n```go\na%d := 1\nb := 2\nc := 3\nd := 4\n```\nafter %d\n\n", i, i, i)
	}
	b.WriteString("end")
	h := openNoteWith(t, b.String())
	h.keys(" p")
	goToLine(h, 3*10+3) // the fence opening section 3's code block
	h.keys("i")
	for range 4 {
		h.key(tea.KeyDown) // onto its "d := 4"
	}
	for i := 0; i < 60; i++ {
		if y, _ := app.NoteCursorY(h.m); y <= 0 {
			break
		}
		h.m = app.ScrollNoteBy(h.m, 1)
		h.send(nil)
	}
	before := paneRows(h.view())
	h.key(tea.KeyDown) // onto the closing fence: same block, still raw
	h.key(tea.KeyDown) // onto "after 3": the code block renders above the top
	after := paneRows(h.view())
	find := func(rows []string, s string) int {
		for i, r := range rows {
			if strings.Contains(r, s) {
				return i
			}
		}
		return -1
	}
	if fb, fa := find(before, "para 4"), find(after, "para 4"); fb < 0 || fb != fa {
		t.Fatalf("para 4 moved from row %d to %d\nbefore:\n%s\nafter:\n%s", fb, fa,
			strings.Join(before, "\n"), strings.Join(after, "\n"))
	}
	if _, _, ok := app.NoteCursor(h.m); !ok {
		t.Fatalf("the cursor should stay on screen:\n%s", h.view())
	}
}
