package rich

import (
	"errors"
	"strings"
	"testing"
)

type fakeSender struct {
	sent  []string
	fail  bool
	frees []int
}

func (f *fakeSender) Send(seqs ...string) error {
	if f.fail {
		return errors.New("terminal refused")
	}
	f.sent = append(f.sent, seqs...)
	return nil
}

func testView(t *testing.T, w, h int) (*View, *fakeSender, *Page) {
	t.Helper()
	f := testFonts(t)
	doc := Parse([]byte(strings.Repeat("A paragraph of body text that fills rows.\n\n", 40)))
	p := Layout(doc, LayoutOptions{Width: w, Fonts: f, Palette: testPalette()})
	s := &fakeSender{}
	v := NewView(w, h, f, testPalette(), s)
	v.SetPage(p)
	return v, s, p
}

func TestViewNotReadyWithoutPage(t *testing.T) {
	f := testFonts(t)
	v := NewView(40, 10, f, testPalette(), &fakeSender{})
	if v.Ready() {
		t.Fatal("a view with no page should say it is not ready")
	}
	if got := v.View(); got != strings.TrimRight(strings.Repeat(strings.Repeat(" ", 40)+"\n", 10), "\n") {
		t.Fatalf("empty view should be blanks:\n%q", got)
	}
}

func TestViewPrepareSendsThenShowsPlaceholders(t *testing.T) {
	v, s, p := testView(t, 40, 10)
	if err := v.Prepare(); err != nil {
		t.Fatal(err)
	}
	if len(s.sent) == 0 {
		t.Fatal("Prepare did not transmit any tile")
	}
	if !strings.Contains(s.sent[0], "a=T") || !strings.Contains(s.sent[0], "U=1") {
		t.Fatalf("first sequence is not a transmit: %q", s.sent[0][:min(len(s.sent[0]), 40)])
	}
	view := v.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 10 {
		t.Fatalf("view has %d lines, want 10", len(lines))
	}
	if !strings.Contains(view, "\U0010EEEE") {
		t.Fatal("view has no placeholder cells")
	}
	// Every line keeps the pane width, placeholders or not.
	for _, l := range lines {
		if got := placeholderCells(l); got > 40 {
			t.Fatalf("line has %d placeholder cells, over the pane", got)
		}
	}
	if p.Height < 10 {
		t.Fatalf("test page is only %d rows tall", p.Height)
	}
}

func TestViewScrollAddsTilesAndEvicts(t *testing.T) {
	v, s, p := testView(t, 40, 10)
	_ = v.Prepare()
	first := len(s.sent)
	if first == 0 {
		t.Fatal("no tile sent")
	}
	for i := 0; i < 8; i++ {
		v.Scroll(5)
		if err := v.Prepare(); err != nil {
			t.Fatal(err)
		}
		if len(v.tiles) > maxTiles {
			t.Fatalf("%d tiles alive, over the quota of %d", len(v.tiles), maxTiles)
		}
	}
	if len(s.sent) == first {
		t.Fatal("scrolling never sent another tile")
	}
	if v.ScrollTop() == 0 {
		t.Fatal("scroll did not move")
	}
	if v.ScrollTop() > v.MaxScroll() {
		t.Fatalf("scroll %d over max %d", v.ScrollTop(), v.MaxScroll())
	}
	_ = p
}

func TestViewGoToLineAndSourceLine(t *testing.T) {
	v, _, p := testView(t, 40, 10)
	if err := v.Prepare(); err != nil {
		t.Fatal(err)
	}
	for line := 0; line < 40; line += 4 {
		v.GoToLine(line)
		if got := v.SourceLine(); got < line-1 || got > line+1 {
			t.Fatalf("at line %d the top source line is %d", line, got)
		}
	}
	_ = p
}

func TestViewDegradesWhenTheTerminalRefuses(t *testing.T) {
	v, s, _ := testView(t, 40, 10)
	s.fail = true
	if err := v.Prepare(); !errors.Is(err, ErrDegraded) {
		t.Fatalf("Prepare = %v, want ErrDegraded", err)
	}
	if !v.Degraded() {
		t.Fatal("view should report degraded")
	}
	if len(v.tiles) != 0 {
		t.Fatalf("%d tiles cached after a failure", len(v.tiles))
	}
	// The view still draws a full pane of blank cells.
	if lines := strings.Split(v.View(), "\n"); len(lines) != 10 {
		t.Fatalf("degraded view has %d lines", len(lines))
	}
}

func TestViewCloseDeletesTiles(t *testing.T) {
	v, s, _ := testView(t, 40, 10)
	if err := v.Prepare(); err != nil {
		t.Fatal(err)
	}
	sent := len(s.sent)
	v.Close()
	var deletes int
	for _, seq := range s.sent[sent:] {
		if strings.Contains(seq, "a=d") {
			deletes++
		}
	}
	if deletes == 0 {
		t.Fatal("Close sent no delete sequences")
	}
	if len(v.tiles) != 0 {
		t.Fatalf("%d tiles left after Close", len(v.tiles))
	}
}

func placeholderCells(line string) int {
	n := 0
	for _, r := range line {
		if r == '\U0010EEEE' {
			n++
		}
	}
	return n
}

func TestViewSearchAndOutline(t *testing.T) {
	f := testFonts(t)
	var body strings.Builder
	body.WriteString("# One\n\nalpha beta\n\n")
	for i := 0; i < 30; i++ {
		body.WriteString("filler paragraph that takes rows of the page\n\n")
	}
	body.WriteString("## Two\n\ngamma alpha\n")
	doc := Parse([]byte(body.String()))
	p := Layout(doc, LayoutOptions{Width: 40, Fonts: f, Palette: testPalette()})
	s := &fakeSender{}
	v := NewView(40, 10, f, testPalette(), s)
	v.SetPage(p)

	out := v.Outline()
	if len(out) != 2 || out[0].Text != "One" || out[1].Level != 2 {
		t.Fatalf("outline = %+v", out)
	}
	if n := v.Search("alpha"); n != 2 {
		t.Fatalf("search matched %d rows, want 2", n)
	}
	if len(v.overlay) != 2 {
		t.Fatalf("search marks = %d, want one per row", len(v.overlay))
	}
	first := v.ScrollTop()
	if !v.SearchNext(1) {
		t.Fatal("search next found nothing")
	}
	if v.ScrollTop() == first {
		t.Fatal("search next did not move the view")
	}
	// The marked tiles must be redrawn: Prepare sends them again.
	if err := v.Prepare(); err != nil {
		t.Fatal(err)
	}
	if len(s.sent) == 0 {
		t.Fatal("no tile sent after a search")
	}
}

func TestViewSelectionCopiesRows(t *testing.T) {
	f := testFonts(t)
	doc := Parse([]byte("first line\n\nsecond line\n"))
	p := Layout(doc, LayoutOptions{Width: 40, Fonts: f, Palette: testPalette()})
	v := NewView(40, 10, f, testPalette(), &fakeSender{})
	v.SetPage(p)
	text := v.Select(0, 3)
	if !strings.Contains(text, "first line") || !strings.Contains(text, "second line") {
		t.Fatalf("selection text = %q", text)
	}
	if len(v.overlay) == 0 {
		t.Fatal("selection drew no overlay")
	}
	a, b := v.SelectionRows()
	if a != 0 || b != 3 {
		t.Fatalf("selection rows = %d..%d", a, b)
	}
	v.ClearSelection()
	ra, rb := v.SelectionRows()
	if len(v.overlay) != 0 || ra != -1 || rb != -1 {
		t.Fatal("ClearSelection left state behind")
	}
}
