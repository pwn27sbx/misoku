package textarea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// refStarts recomputes the first visual row of every logical line from
// scratch, without any cache; the last element is the total row count.
func refStarts(m Model) []int {
	starts := make([]int, 0, len(m.value)+1)
	acc := 0
	for _, l := range m.value {
		starts = append(starts, acc)
		acc += len(wrap(l, m.width))
	}
	return append(starts, acc)
}

// refMoveToVisual is the linear moveToVisual that predates the row index.
func refMoveToVisual(m *Model, v, col int) {
	v = max(v, 0)
	acc := 0
	for r, l := range m.value {
		segs := wrap(l, m.width)
		if v < acc+len(segs) {
			start := 0
			for i := 0; i < v-acc; i++ {
				start += len(segs[i])
			}
			segLen := len(segs[v-acc])
			if v-acc < len(segs)-1 && segLen > 0 {
				segLen--
			}
			m.row = r
			m.col = min(start+clamp(col, 0, segLen), len(l))
			return
		}
		acc += len(segs)
	}
	m.row = len(m.value) - 1
	m.col = len(m.value[m.row])
}

// checkIndex fails when the cached row index, or anything derived from it,
// disagrees with a full recompute.
func checkIndex(t *testing.T, m Model, op string) {
	t.Helper()
	want := refStarts(m)
	got := m.rows()
	if len(got) != len(want) {
		t.Fatalf("after %s: index has %d entries, want %d", op, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("after %s: index[%d] = %d, want %d (lines=%d width=%d)", op, i, got[i], want[i], len(m.value), m.width)
		}
	}
	total := want[len(want)-1]
	if n := m.visualLineCount(); n != total {
		t.Fatalf("after %s: visualLineCount = %d, want %d", op, n, total)
	}
	if r, w := m.cursorLineNumber(), want[m.row]+m.LineInfo().RowOffset; r != w {
		t.Fatalf("after %s: cursorLineNumber = %d, want %d", op, r, w)
	}
	for _, v := range []int{-1, 0, total / 3, total / 2, total - 1, total, total + 5} {
		for _, col := range []int{0, 7, 1000} {
			a, b := m, m
			a.moveToVisual(v, col)
			refMoveToVisual(&b, v, col)
			if a.row != b.row || a.col != b.col {
				t.Fatalf("after %s: moveToVisual(%d,%d) = %d:%d, want %d:%d", op, v, col, a.row, a.col, b.row, b.col)
			}
		}
	}
}

var fuzzTexts = []string{
	"a", "word ", "\n", "漢字かな", "😀 ", "line one\nline two\nline three",
	"averyveryverylongwordthatcannotbreakanywhere ", "  ", "x\n\n\ny",
	"漢字かな交じりの長い行は幅の広い文字で折り返される漢字かな交じり ",
}

var fuzzKeys = []tea.KeyMsg{
	{Type: tea.KeyBackspace}, {Type: tea.KeyDelete}, {Type: tea.KeyEnter},
	{Type: tea.KeyCtrlK}, {Type: tea.KeyCtrlU}, {Type: tea.KeyCtrlW},
	{Type: tea.KeyRunes, Runes: []rune("d"), Alt: true},
	{Type: tea.KeyCtrlT},
	{Type: tea.KeyRunes, Runes: []rune("u"), Alt: true},
	{Type: tea.KeyRunes, Runes: []rune("l"), Alt: true},
	{Type: tea.KeyRunes, Runes: []rune("c"), Alt: true},
	{Type: tea.KeyDown}, {Type: tea.KeyUp}, {Type: tea.KeyLeft}, {Type: tea.KeyRight},
	{Type: tea.KeyCtrlEnd}, {Type: tea.KeyCtrlHome}, {Type: tea.KeyEnd},
	{Type: tea.KeyRunes, Runes: []rune("z")},
	{Type: tea.KeySpace, Runes: []rune(" ")},
}

// FuzzWrapIndex applies random edits and, after each one, checks that the
// cached row index equals a full recompute and that every change to the text
// moved Version. A write to the text that skips touch() fails here.
func FuzzWrapIndex(f *testing.F) {
	f.Add("hello world\nsecond line", []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13})
	f.Add(strings.Repeat("some words that wrap ", 12)+"\n漢字かな😀\nend", []byte{
		1, 0, 1, 2, 1, 4, 1, 6, 7, 3, 2, 9, 4, 11, 8, 7, 20, 30, 40, 5, 15, 25, 35, 45, 55, 65})
	f.Add("a\nb\nc\nd\ne\nf", []byte{3, 3, 5, 19, 5, 33, 9, 22, 12, 13, 14, 26, 27, 1, 1, 1})
	f.Add("", []byte{0, 0, 0, 2, 2, 7, 7, 6, 6, 13})
	// Word case and transpose keys (alt+u, alt+l, alt+c, ctrl+t) rewrite
	// runes in place without changing the line length.
	f.Add("hello world again", []byte{11, 113, 11, 127, 11, 141, 15, 99})
	// ctrl+u, ctrl+k, ctrl+w, alt+d, delete at line end, indent/outdent,
	// linewise DeleteSelection, DeleteBefore and DeleteLine on a single line.
	f.Add("alpha beta\ngamma delta\nepsilon\nzeta eta\ntheta", []byte{
		25, 57, 25, 43, 71, 53, 85, 15, 4, 18, 19, 14, 36, 13, 0, 3})
	// alt+d at the line start, then enter and a backspace that joins lines.
	f.Add("one two three", []byte{11, 85, 29, 1})
	f.Fuzz(func(t *testing.T, seed string, ops []byte) {
		if len(ops) > 200 {
			ops = ops[:200]
		}
		m := New()
		m.SetWidth(30)
		m.SetValue(seed)
		m.Focus()
		checkIndex(t, m, "SetValue")
		for i, b := range ops {
			prevValue, prevVersion := m.Value(), m.Version()
			op := applyFuzzOp(&m, b, seed)
			if m.Value() != prevValue && m.Version() == prevVersion {
				t.Fatalf("op %d (%s) changed the text without moving Version", i, op)
			}
			checkIndex(t, m, op)
		}
	})
}

func TestVersionMovesOnlyOnEdits(t *testing.T) {
	m := New()
	m.SetValue("one\ntwo\nthree")
	m.Focus()
	v := m.Version()
	for _, k := range []tea.KeyType{tea.KeyDown, tea.KeyUp, tea.KeyRight, tea.KeyEnd, tea.KeyCtrlHome} {
		m, _ = m.Update(tea.KeyMsg{Type: k})
	}
	m.ScrollBy(1)
	m.ClickAt(8, 1)
	_ = m.View()
	if m.Version() != v {
		t.Fatalf("cursor movement changed Version from %d to %d", v, m.Version())
	}
	m.InsertString("x")
	if m.Version() == v {
		t.Fatal("an edit did not change Version")
	}
	// Two models never share a version, so a copy edited apart from the
	// original cannot reuse its index.
	a, b := m, m
	a.InsertString("a")
	b.InsertString("b")
	if a.Version() == b.Version() {
		t.Fatal("independent edits produced the same Version")
	}
}

// TestRepositionWithoutPriorView: Update keeps the cursor on screen even when
// View has not run since the text was set (the view used to be clamped by
// the content of the last View, which was empty here).
func TestRepositionWithoutPriorView(t *testing.T) {
	m := New()
	m.SetWidth(40)
	m.SetHeight(5)
	m.SetValue(strings.Repeat("line\n", 50) + "last")
	m.Focus()
	m.GotoTop()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlEnd})
	if row := m.cursorLineNumber(); row < m.YOffset() || row >= m.YOffset()+m.Height() {
		t.Fatalf("cursor row %d is off screen (yoff %d, height %d)", row, m.YOffset(), m.Height())
	}
}

func TestPlaceholderOnlyForEmptyText(t *testing.T) {
	for _, tc := range []struct {
		text string
		want bool
	}{{"", true}, {"\n", false}, {" ", false}, {"a", false}} {
		m := New()
		m.Placeholder = "PLACEHOLDER"
		m.SetValue(tc.text)
		m.GotoTop()
		if got := strings.Contains(m.View(), "PLACEHOLDER"); got != tc.want {
			t.Errorf("text %q: placeholder shown = %v, want %v", tc.text, got, tc.want)
		}
	}
}

// applyFuzzOp applies the edit encoded by b to m and returns its name. The
// low bits pick the operation and the rest is its argument.
func applyFuzzOp(m *Model, b byte, seed string) string {
	arg := int(b) / 14
	var op string
	switch b % 14 {
	case 0:
		op = "InsertString"
		m.InsertString(fuzzTexts[arg%len(fuzzTexts)])
	case 1:
		op = "Update key"
		*m, _ = m.Update(fuzzKeys[arg%len(fuzzKeys)])
	case 2:
		op = "paste"
		*m, _ = m.Update(pasteMsg(fuzzTexts[arg%len(fuzzTexts)]))
	case 3:
		op = "DeleteLine"
		m.DeleteLine()
	case 4:
		op = "IndentLine/OutdentLine"
		if arg%2 == 0 {
			m.IndentLine(arg%5 + 1)
		} else {
			m.OutdentLine(arg%5 + 1)
		}
	case 5:
		op = "DeleteSelection"
		m.SelectLinewise = arg%2 == 1
		m.SetAnchor(arg%len(m.value), arg%7)
		m.SetRowCol(len(m.value)-1-arg%len(m.value), arg%5)
		m.DeleteSelection()
	case 6:
		op = "SetValue"
		m.SetValue(fuzzTexts[arg%len(fuzzTexts)] + seed)
	case 7:
		op = "SetWidth"
		m.SetWidth(10 + arg*3)
	case 8:
		op = "DeleteBefore"
		m.DeleteBefore(arg % 4)
	case 9:
		op = "DeleteCharForward"
		m.DeleteCharForward()
	case 10:
		op = "OpenLineBelow/Above"
		if arg%2 == 0 {
			m.OpenLineBelow()
		} else {
			m.OpenLineAbove()
		}
	case 11:
		op = "SetRowCol"
		m.SetRowCol(arg%len(m.value), arg*3)
	case 12:
		op = "InsertRune"
		m.InsertRune([]rune("q漢 😀")[arg%4])
	case 13:
		op = "Reset"
		m.Reset()
	}
	return op
}
