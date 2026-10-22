package textarea_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/testnote"
	"github.com/pwn27sbx/misoku/internal/textarea"

	tea "github.com/charmbracelet/bubbletea"
)

// navGolden is the recorded trace of cursor and scroll state for a scripted
// session. It was generated before the wrapped-row index replaced the linear
// scans, so any change in how rows map to positions fails here. Regenerate
// with UPDATE_NAV_GOLDEN=1 only for an intended behavior change.
var navGolden = filepath.Join("testdata", "navigation.golden")

// navEditor returns a narrow editor whose note mixes long soft-wrapped
// lines, wide CJK runes and emoji, so most logical lines span several rows.
func navEditor() textarea.Model {
	m := textarea.New()
	m.SetWidth(40)
	m.SetHeight(10)
	extra := "漢字かな交じりの長い行は幅の広い文字で折り返される 漢字かな交じり 😀 emoji 😀 wide 😀 runes 😀 wrap here too\n"
	m.SetValue(extra + testnote.Note(300) + "\n" + extra + "tail")
	m.Focus()
	m.GotoTop()
	return m
}

func TestNavigationTraceGolden(t *testing.T) {
	m := navEditor()
	var b strings.Builder
	step := func(name string, f func()) {
		f()
		_ = m.View() // the app renders after every message
		row, col := m.RowCol()
		x, y, ok := m.CursorScreen()
		li := m.LineInfo()
		fmt.Fprintf(&b, "%-22s row=%d col=%d yoff=%d screen=(%d,%d,%v) li=%d/%d/%d\n",
			name, row, col, m.YOffset(), x, y, ok, li.RowOffset, li.ColumnOffset, li.Height)
	}
	upd := func(k tea.KeyType) func() {
		return func() { m, _ = m.Update(tea.KeyMsg{Type: k}) }
	}
	half := m.Height() / 2

	step("start", func() {})
	for i := range 25 {
		step(fmt.Sprintf("down%d", i), upd(tea.KeyDown))
	}
	for i := range 8 {
		step(fmt.Sprintf("up%d", i), upd(tea.KeyUp))
	}
	step("G", m.GotoBottom)
	step("down-past-end", upd(tea.KeyDown))
	step("move-past-end", func() { m.MoveCursorBy(5) })
	step("end-of-line", upd(tea.KeyEnd))
	step("up-from-end", upd(tea.KeyUp))
	step("gg", m.GotoTop)
	step("up-past-top", upd(tea.KeyUp))
	step("move-past-top", func() { m.MoveCursorBy(-5) })
	for i := range 6 {
		step(fmt.Sprintf("halfdown%d", i), func() { m.ScrollBy(half); m.MoveCursorBy(half) })
	}
	for i := range 3 {
		step(fmt.Sprintf("halfup%d", i), func() { m.ScrollBy(-half); m.MoveCursorBy(-half) })
	}
	step("wheel+3", func() { m.ScrollBy(3) })
	step("wheel+3b", func() { m.ScrollBy(3) })
	step("wheel-3", func() { m.ScrollBy(-3) })
	step("wheel-huge", func() { m.ScrollBy(100000) })
	step("down-after-wheel", upd(tea.KeyDown))
	step("wheel-huge-up", func() { m.ScrollBy(-100000) })
	for y := range 10 {
		step(fmt.Sprintf("click-y%d", y), func() { m.ClickAt(12+3*y, y) })
	}
	step("click-gutter", func() { m.ClickAt(0, 2) })
	step("click-far-right", func() { m.ClickAt(200, 1) })
	step("click-below", func() { m.ClickAt(5, 50) })
	step("goto-line-150", func() { m.GotoLine(150) })
	step("right", upd(tea.KeyRight))
	step("word-right", upd(tea.KeyCtrlRight))
	step("type", func() { m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("typed 漢字 text")}) })
	step("enter", upd(tea.KeyEnter))
	step("backspace-join", upd(tea.KeyBackspace))
	step("ctrl+end", upd(tea.KeyCtrlEnd))
	step("ctrl+home", upd(tea.KeyCtrlHome))
	step("wheel-to-mid", func() { m.ScrollBy(400) })
	step("resize-narrow", func() { m.SetWidth(25) })
	step("down-narrow", upd(tea.KeyDown))
	step("click-narrow", func() { m.ClickAt(10, 4) })
	step("resize-wide", func() { m.SetWidth(90) })
	step("up-wide", upd(tea.KeyUp))

	got := b.String()
	if os.Getenv("UPDATE_NAV_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(navGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(navGolden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := range min(len(gl), len(wl)) {
			if gl[i] != wl[i] {
				t.Fatalf("trace differs at line %d:\n got: %s\nwant: %s", i+1, gl[i], wl[i])
			}
		}
		t.Fatalf("trace length differs: got %d lines, want %d", len(gl), len(wl))
	}
}
