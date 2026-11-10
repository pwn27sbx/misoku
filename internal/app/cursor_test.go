package app_test

import (
	"bytes"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/cursor"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// execDone is what the stubbed program reports when it ends.
type execDone struct{}

// reverseVideo reports an SGR sequence that turns reverse video on: how the
// editor draws its own cursor.
func reverseVideo(s string) bool {
	for i := 0; i+3 < len(s); i++ {
		if s[i] == 0x1b && s[i+1] == '[' && s[i+2] == '7' && (s[i+3] == 'm' || s[i+3] == ';') {
			return true
		}
	}
	return false
}

// cursorOut is the program output with the real terminal cursor, wired the way
// main wires it: the app reports the cell and the shape, the writer turns that
// into the bytes a terminal reads.
type cursorOut struct {
	buf bytes.Buffer
	w   *cursor.Writer
}

func newCursorOut() (*cursorOut, app.Option) {
	c := &cursorOut{}
	c.w = cursor.NewWriter(&c.buf)
	return c, app.WithCursor(c.w)
}

func (c *cursorOut) all() string { return c.buf.String() }

func (c *cursorOut) count(seq string) int { return strings.Count(c.all(), seq) }

var cupPattern = regexp.MustCompile(`\x1b\[(\d+);(\d+)H`)

// placed is the cell of the last CUP the writer emitted.
func (c *cursorOut) placed(t *testing.T) (x, y int) {
	t.Helper()
	m := cupPattern.FindAllStringSubmatch(c.all(), -1)
	if len(m) == 0 {
		t.Fatalf("no cursor placement in %q", c.all())
	}
	last := m[len(m)-1]
	row, _ := strconv.Atoi(last[1])
	col, _ := strconv.Atoi(last[2])
	return col - 1, row - 1
}

// cursorCellAt is the rune the frame draws on cell (x, y), 0-based.
func cursorCellAt(t *testing.T, frame string, x, y int) string {
	t.Helper()
	lines := strings.Split(ansi.Strip(frame), "\n")
	if y < 0 || y >= len(lines) {
		t.Fatalf("row %d is outside the %d-row frame", y, len(lines))
	}
	runes := []rune(lines[y])
	if x < 0 || x >= len(runes) {
		t.Fatalf("column %d is outside the %d-column row %q", x, len(runes), lines[y])
	}
	return string(runes[x])
}

// openWithCursor opens a note with content and leaves the editor focused.
func openWithCursor(t *testing.T, content string, opts ...app.Option) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", content); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, opts...)
	h.keys("1")
	h.key(tea.KeyEsc)
	h.m.View() // the runtime draws a frame after every update
	return h
}

// TestNormalModePutsTheBlockOnTheEditorCell is the offset rule: the placement
// must land on the character the frame draws under the editor cursor, with the
// explorer open and with its columns gone.
func TestNormalModePutsTheBlockOnTheEditorCell(t *testing.T) {
	out, opt := newCursorOut()
	h := openWithCursor(t, "hello\nworld", opt)
	frame := h.m.View()

	if out.count("\x1b[2 q") == 0 {
		t.Fatalf("NORMAL did not ask for a block cursor: %q", out.all())
	}
	x, y := out.placed(t)
	if got := cursorCellAt(t, frame, x, y); got != "h" {
		t.Fatalf("NORMAL cursor on %q, want the h of hello\n%s", got, h.view())
	}

	// Moving the cursor moves the placement, not the shape.
	out.buf.Reset()
	h.keys("l")
	frame = h.m.View()
	if got := out.count("\x1b[2 q"); got != 0 {
		t.Fatalf("moving emitted the shape %d times, want 0: %q", got, out.all())
	}
	x, y = out.placed(t)
	if got := cursorCellAt(t, frame, x, y); got != "e" {
		t.Fatalf("cursor after l on %q, want the e of hello\n%s", got, h.view())
	}
}

func TestExplorerColumnsShiftThePlacement(t *testing.T) {
	out, opt := newCursorOut()
	h := openWithCursor(t, "hello", opt)
	frame := h.m.View()
	xOpen, yOpen := out.placed(t)

	h.send(tea.KeyMsg{Type: tea.KeyCtrlB}) // hide the explorer
	if explorerVisible(h.view()) {
		t.Fatalf("setup: ctrl+b should hide the explorer:\n%s", h.view())
	}
	frame = h.m.View()
	xClosed, yClosed := out.placed(t)

	if yOpen != yClosed {
		t.Fatalf("the tab bar moved the cursor row: %d then %d", yOpen, yClosed)
	}
	if want := xOpen - (app.SidebarWidth + 1); xClosed != want {
		t.Fatalf("closing the explorer moved the cursor to %d, want %d", xClosed, want)
	}
	if got := cursorCellAt(t, frame, xClosed, yClosed); got != "h" {
		t.Fatalf("cursor without the explorer on %q, want h\n%s", got, h.view())
	}
}

// Several tabs change the tab line, never the row the panes start on.
func TestTabsDoNotMoveThePlacement(t *testing.T) {
	out, opt := newCursorOut()
	store := newTestStore(t)
	for _, n := range []string{"one", "two"} {
		if _, err := store.SaveAs("", n, n+" body"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store, opt)
	h.keys("e") // explorer, on the first note
	h.key(tea.KeyEnter)
	h.m.View()
	x1, y1 := out.placed(t)

	h.key(tea.KeyEsc)
	h.keys(" e") // explorer again
	h.keys("j")
	h.key(tea.KeyEnter) // second note: a second tab
	x2, y2 := out.placed(t)

	if x1 != x2 || y1 != y2 {
		t.Fatalf("the second tab moved the cursor from (%d,%d) to (%d,%d)", x1, y1, x2, y2)
	}
}

func TestInsertIsABarAndVisualIsABlock(t *testing.T) {
	out, opt := newCursorOut()
	h := openWithCursor(t, "hello", opt)

	out.buf.Reset()
	h.keys("i")
	h.m.View()
	if got := out.count("\x1b[6 q"); got != 1 {
		t.Fatalf("INSERT emitted the bar %d times, want 1: %q", got, out.all())
	}

	out.buf.Reset()
	h.key(tea.KeyEsc)
	h.m.View()
	if got := out.count("\x1b[2 q"); got != 1 {
		t.Fatalf("back to NORMAL emitted the block %d times, want 1: %q", got, out.all())
	}

	// VISUAL and VISUAL-LINE keep NORMAL's block: no shape sequence at all,
	// and above all no bar.
	for _, key := range []string{"v", "V"} {
		out.buf.Reset()
		h.keys(key)
		h.m.View()
		if got := out.all(); got != "" {
			t.Fatalf("%q changed the cursor %q, want the block kept", key, got)
		}
	}
}

// A frame the renderer skips must not repeat the shape either: the state is
// what changed, not the frame.
func TestTheShapeIsWrittenOnceAcrossFrames(t *testing.T) {
	out, opt := newCursorOut()
	h := openWithCursor(t, "hello", opt)
	h.keys("i")

	out.buf.Reset()
	for range 4 {
		h.m.View()
	}
	if got := out.count("\x1b[6 q"); got != 1 {
		t.Fatalf("the bar was written %d times across four frames, want 1: %q", got, out.all())
	}
}

func TestCommandModeHidesTheRealCursor(t *testing.T) {
	out, opt := newCursorOut()
	h := openWithCursor(t, "hello", opt)
	out.buf.Reset()

	h.keys(":")
	h.m.View()

	if got := out.count("\x1b[?25l"); got != 1 {
		t.Fatalf("COMMAND hid the cursor %d times, want 1: %q", got, out.all())
	}
	if got := cupPattern.FindAllString(out.all(), -1); len(got) != 0 {
		t.Fatalf("COMMAND still placed the cursor: %q", out.all())
	}
}

func TestTheFinderHidesTheRealCursor(t *testing.T) {
	out, opt := newCursorOut()
	h := openWithCursor(t, "hello", opt)
	out.buf.Reset()

	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.m.View()

	if got := out.count("\x1b[?25l"); got != 1 {
		t.Fatalf("the finder hid the cursor %d times, want 1: %q", got, out.all())
	}
	if got := cupPattern.FindAllString(out.all(), -1); len(got) != 0 {
		t.Fatalf("the finder still placed the cursor: %q", out.all())
	}
}

// The fake cursor is drawn only when the terminal cursor is not in charge, and
// the placeholder's own fake cursor goes with it.
func TestTheFakeCursorIsHiddenOnlyWithTheRealCursor(t *testing.T) {
	trueColour(t)
	plain := openWithCursor(t, "hello")
	if !reverseVideo(plain.m.View()) {
		t.Fatal("setup: without the real cursor the editor still draws its own")
	}

	_, opt := newCursorOut()
	real := openWithCursor(t, "hello", opt)
	if got := real.m.View(); reverseVideo(got) {
		t.Fatalf("the real cursor is on and the fake one is still drawn:\n%s", got)
	}
	real.keys("i")
	if got := real.m.View(); reverseVideo(got) {
		t.Fatalf("INSERT still draws the fake cursor:\n%s", got)
	}

	empty := openWithCursor(t, "", opt)
	if got := empty.m.View(); reverseVideo(got) {
		t.Fatalf("the placeholder still draws the fake cursor:\n%s", got)
	}
}

// Handing the terminal to gh: the default shape first, the editor's own back
// after.
func TestRunningAProgramOutsideResetsTheShape(t *testing.T) {
	out, opt := newCursorOut()
	var ran bool
	// The stub runs the callback, as tea.ExecProcess does once the program ends.
	exec := app.WithExec(func(_ *exec.Cmd, cb tea.ExecCallback) tea.Cmd {
		ran = true
		return func() tea.Msg { return cb(nil) }
	})
	h := openWithCursor(t, "hello", opt, exec)
	h.m.View()

	out.buf.Reset()
	if cmd := app.ExecOutside(h.m, nil, func(error) tea.Msg { return execDone{} }); cmd != nil {
		h.send(cmd())
	}
	if !ran {
		t.Fatal("the program never ran")
	}
	if got := out.all(); !strings.HasSuffix(got, "\x1b[0 q\x1b[?25h") {
		t.Fatalf("the shape was not reset before the program: %q", got)
	}

	out.buf.Reset()
	h.m.View()
	// The program may have hidden the cursor or given it a shape: the whole
	// state is said again.
	if got := out.all(); !strings.HasPrefix(got, "\x1b[?25h\x1b[2 q") {
		t.Fatalf("the shape was not applied again after the program: %q", got)
	}
}

// A frame without the editor focused (the start screen) leaves the terminal
// cursor hidden and the drawn cursors alone.
func TestTheStartScreenKeepsTheRealCursorHidden(t *testing.T) {
	out, opt := newCursorOut()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "hello"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, opt)
	h.m.View()

	if got := out.count("\x1b[?25h"); got != 0 {
		t.Fatalf("the start screen showed the real cursor: %q", out.all())
	}
	if got := cupPattern.FindAllString(out.all(), -1); len(got) != 0 {
		t.Fatalf("the start screen placed the real cursor: %q", out.all())
	}
}

// Both full-width views edit, and the block under the cursor is raw, so the
// real cursor shows in each.
func TestReadingModeShowsTheRealCursor(t *testing.T) {
	out, opt := newCursorOut()
	h := openWithCursor(t, "hello", opt)
	h.keys(" p")
	out.buf.Reset()
	h.m.View()
	if got := cupPattern.FindAllString(out.all(), -1); len(got) == 0 {
		t.Fatalf("the reading view did not place the real cursor: %q", out.all())
	}
	h.keys(" r")
	out.buf.Reset()
	h.m.View()
	if got := cupPattern.FindAllString(out.all(), -1); len(got) == 0 {
		t.Fatalf("the editor view did not place the real cursor: %q", out.all())
	}
}

// Typing a name in the folder editor shows the real cursor as INSERT's bar,
// on the cell where the next character goes, after the indentation and the
// icon, with no drawn cell of its own. Its normal mode keeps today's look:
// no real cursor.
func TestFolderEditorNameShowsTheRealCursor(t *testing.T) {
	out, opt := newCursorOut()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "aaa", "body"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, opt)
	h.keys("e")
	h.keys("m")
	if v := h.view(); !strings.Contains(v, "EDIT /") {
		t.Fatalf("setup: m must open the folder editor:\n%s", v)
	}
	h.keys("ix")
	out.buf.Reset()
	frame := h.m.View()
	if out.count("\x1b[6 q") == 0 || out.count("\x1b[?25h") == 0 {
		t.Fatalf("typing a name did not show the bar cursor: %q", out.all())
	}
	x, y := out.placed(t)
	if got := cursorCellAt(t, frame, x-1, y); got != "x" {
		t.Fatalf("cell before the cursor is %q, want the typed x\n%s", got, h.view())
	}
	if got := cursorCellAt(t, frame, x, y); got != "a" {
		t.Fatalf("cursor on %q, want the a of aaa\n%s", got, h.view())
	}
	if line := strings.Split(frame, "\n")[y]; reverseVideo(line) {
		t.Fatalf("the name line still draws its own cursor cell: %q", line)
	}

	h.key(tea.KeyEsc)
	out.buf.Reset()
	h.m.View()
	if got := out.all(); !strings.Contains(got, "\x1b[?25l") || cupPattern.MatchString(got) {
		t.Fatalf("normal mode in the folder editor must hide the real cursor: %q", got)
	}
}

func TestFolderEditorArrowsMoveTheRealCursorWhileTyping(t *testing.T) {
	out, opt := newCursorOut()
	store := newTestStore(t)
	for _, name := range []string{"aaa", "bbb"} {
		if _, err := store.SaveAs("", name, "body"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store, opt)
	h.keys("e")
	h.keys("m")
	h.keys("Ax")
	out.buf.Reset()
	h.m.View()
	_, y0 := out.placed(t)

	h.key(tea.KeyDown)
	out.buf.Reset()
	frame := h.m.View()
	x, y := out.placed(t)
	if y != y0+1 {
		t.Fatalf("down moved the cursor to row %d, want %d\n%s", y, y0+1, h.view())
	}
	if got := cursorCellAt(t, frame, x-1, y); got != "b" {
		t.Fatalf("cell before the cursor is %q, want the end of bbb\n%s", got, h.view())
	}
	if !strings.Contains(h.view(), "aaax") {
		t.Fatalf("the typed x was lost:\n%s", h.view())
	}

	h.key(tea.KeyUp)
	out.buf.Reset()
	frame = h.m.View()
	x, y = out.placed(t)
	if y != y0 || cursorCellAt(t, frame, x-1, y) != "x" {
		t.Fatalf("up did not bring the cursor back after aaax\n%s", h.view())
	}
}
