package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/testnote"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// The benchmarks run at 120x40, the size the render plan measures at.
const (
	benchW = 120
	benchH = 40
)

// benchSizes are the note lengths every benchmark runs on.
var benchSizes = []struct {
	name  string
	lines int
}{
	{"500", 500},
	{"5k", 5000},
}

// benchApp is a 120x40 app with a generated note open in NORMAL mode, plus
// the screen columns of a point inside the editor and inside the preview.
type benchApp struct {
	*harness
	editorX, previewX int
}

func openBenchNote(b *testing.B, lines int) benchApp {
	b.Helper()
	store := newTestStore(b)
	if _, err := store.SaveAs("", "doc", testnote.Note(lines)); err != nil {
		b.Fatal(err)
	}
	h := newHarnessSize(b, benchW, benchH, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	v := h.m.View()
	if !strings.Contains(ansi.Strip(v), "Chapter 1") {
		b.Fatalf("setup: the note is not open:\n%s", ansi.Strip(v))
	}
	// Sidebar | editor | preview: the editor starts after the first divider
	// and the preview after the last one.
	div := dividers(v, bodyRow)
	if len(div) < 2 {
		b.Fatalf("setup: want the editor and the preview, dividers at %v:\n%s", div, ansi.Strip(v))
	}
	return benchApp{harness: h, editorX: div[0] + 5, previewX: div[len(div)-1] + 5}
}

// step is what the runtime does for one message: Update, then View. The
// returned command is dropped; the runtime runs it in the background.
func (a benchApp) step(msg tea.Msg) {
	a.m, _ = a.m.Update(msg)
	_ = a.m.View()
}

// forth reports whether step i of a back-and-forth walk of span steps goes
// forward, so a long run keeps moving instead of stopping at the end.
func forth(i, span int) bool { return i%(2*span) < span }

// BenchmarkAppView measures one View of the whole app.
func BenchmarkAppView(b *testing.B) {
	for _, s := range benchSizes {
		b.Run(s.name, func(b *testing.B) {
			a := openBenchNote(b, s.lines)
			b.ReportAllocs()
			for b.Loop() {
				_ = a.m.View()
			}
		})
	}
}

// BenchmarkMoveKey measures j (or k on the way back) in NORMAL mode:
// Update + View.
func BenchmarkMoveKey(b *testing.B) {
	j := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	k := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	for _, s := range benchSizes {
		b.Run(s.name, func(b *testing.B) {
			a := openBenchNote(b, s.lines)
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				if forth(i, s.lines-1) {
					a.step(j)
				} else {
					a.step(k)
				}
				i++
			}
		})
	}
}

// BenchmarkTypeKey measures typing one letter in INSERT mode: Update + View.
func BenchmarkTypeKey(b *testing.B) {
	x := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	for _, s := range benchSizes {
		b.Run(s.name, func(b *testing.B) {
			a := openBenchNote(b, s.lines)
			a.keys("i")
			b.ReportAllocs()
			for b.Loop() {
				a.step(x)
			}
		})
	}
}

// BenchmarkWheelScroll measures one wheel step over the editor: Update +
// View.
func BenchmarkWheelScroll(b *testing.B) {
	for _, s := range benchSizes {
		b.Run(s.name, func(b *testing.B) {
			a := openBenchNote(b, s.lines)
			down := tea.MouseMsg{X: a.editorX, Y: bodyRow, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress}
			up := down
			up.Button = tea.MouseButtonWheelUp
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				if forth(i, s.lines/3) {
					a.step(down)
				} else {
					a.step(up)
				}
				i++
			}
		})
	}
}

// motion moves the pointer one cell back and forth around x, no button held.
func motion(a benchApp, x int, b *testing.B) {
	i := 0
	for b.Loop() {
		a.step(tea.MouseMsg{X: x + i%2, Y: bodyRow, Action: tea.MouseActionMotion})
		i++
	}
}

// filteredMotion is motion as the program delivers it: app.MouseFilter
// first, and Update + View only for what it lets through.
func filteredMotion(a benchApp, x int, b *testing.B) {
	i := 0
	for b.Loop() {
		if msg := app.MouseFilter(a.m, tea.MouseMsg{X: x + i%2, Y: bodyRow, Action: tea.MouseActionMotion}); msg != nil {
			a.step(msg)
		}
		i++
	}
}

// BenchmarkMouseMotion measures one pointer move over the editor without a
// drag, which changes nothing on screen: Update + View. The filtered runs go
// through app.MouseFilter first, as the program does, which drops it.
func BenchmarkMouseMotion(b *testing.B) {
	for _, s := range benchSizes {
		b.Run(s.name, func(b *testing.B) {
			a := openBenchNote(b, s.lines)
			b.ReportAllocs()
			motion(a, a.editorX, b)
		})
		b.Run("filtered/"+s.name, func(b *testing.B) {
			a := openBenchNote(b, s.lines)
			b.ReportAllocs()
			filteredMotion(a, a.editorX, b)
		})
	}
}

// BenchmarkPreviewMotion measures one pointer move over the preview, where
// hovering shows the block handles: Update + View. The filtered runs go
// through app.MouseFilter first, which lets it through.
func BenchmarkPreviewMotion(b *testing.B) {
	for _, s := range benchSizes {
		b.Run(s.name, func(b *testing.B) {
			a := openBenchNote(b, s.lines)
			b.ReportAllocs()
			motion(a, a.previewX, b)
		})
		b.Run("filtered/"+s.name, func(b *testing.B) {
			a := openBenchNote(b, s.lines)
			b.ReportAllocs()
			filteredMotion(a, a.previewX, b)
		})
	}
}
