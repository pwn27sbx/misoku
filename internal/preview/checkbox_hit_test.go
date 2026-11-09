package preview_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// checkboxHits returns the box hits of md, in order.
func checkboxHits(md string) (string, []preview.Hit) {
	out, hits := preview.CompileHits(md, theme.DefaultTheme(), 40, "", preview.CalendarView{})
	var boxes []preview.Hit
	for _, h := range hits {
		if h.Kind == preview.HitCheckbox {
			boxes = append(boxes, h)
		}
	}
	return out, boxes
}

// S1: the box of a to-do line is a hit that spans the box cell(s) only and
// names its source line.
func TestCheckboxHitCoversTheBoxOnly(t *testing.T) {
	look := preview.DefaultLook()
	out, boxes := checkboxHits("intro\n\n- [ ] alpha\n  - [x] beta\n")
	if len(boxes) != 2 {
		t.Fatalf("want a hit per box, got %+v in:\n%s", boxes, ansi.Strip(out))
	}
	for i, box := range []string{look.Unchecked, look.Checked} {
		h := boxes[i]
		indent := 2 * i
		if h.X0 != indent || h.X1 != indent+ansi.StringWidth(box) {
			t.Errorf("box %d spans [%d,%d), want [%d,%d)", i, h.X0, h.X1, indent, indent+ansi.StringWidth(box))
		}
		if h.Line != 2+i || h.H != 1 {
			t.Errorf("box %d = line %d height %d, want line %d height 1", i, h.Line, h.H, 2+i)
		}
	}
}

// No box, no hit: "[-]" without a todo glyph is plain text, an ordered item
// is not a preview checkbox, and the text after a box is not part of it.
func TestNoCheckboxHitWithoutABox(t *testing.T) {
	for md, want := range map[string]int{
		"intro\n- [-] doing\n":       0,
		"intro\n1. [ ] numbered\n":   0,
		"- [ ] a\nsome text after\n": 1,
	} {
		_, boxes := checkboxHits(md)
		if len(boxes) != want {
			t.Errorf("%q: got %d box hits, want %d", md, len(boxes), want)
		}
	}
}

// Front matter pushes the document down: the hit still names the note's own
// line.
func TestCheckboxHitLineSkipsFrontMatter(t *testing.T) {
	_, boxes := checkboxHits("---\ntitle: A\n---\n- [ ] alpha\n")
	if len(boxes) != 1 || boxes[0].Line != 3 {
		t.Fatalf("box hits = %+v, want one on line 3", boxes)
	}
}

// A click on the box is a hit; a click on the text of the line is not.
func TestCheckboxClickTargets(t *testing.T) {
	look := preview.DefaultLook()
	m := preview.New(theme.DefaultTheme())
	m.SetSize(60, 20)
	m.SetPage(core.Page{ID: "a.md", Title: "A", Content: "- [ ] alpha\n"})
	v := ansi.Strip(m.View())

	row, x := -1, 0
	for y, l := range strings.Split(v, "\n") {
		if i := strings.Index(l, look.Unchecked); i >= 0 {
			row, x = y, ansi.StringWidth(l[:i])
			break
		}
	}
	if row < 0 {
		t.Fatalf("no unchecked box on screen:\n%s", v)
	}
	h, ok := m.HitAt(x, row)
	if !ok || h.Kind != preview.HitCheckbox || h.Line != 0 {
		t.Fatalf("box hit = %+v ok=%v, want the box of line 0", h, ok)
	}
	if _, ok := m.HitAt(x+ansi.StringWidth(look.Unchecked)+2, row); ok {
		t.Error("the text of the line must not be a hit")
	}
}
