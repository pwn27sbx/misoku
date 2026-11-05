package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// bodyRow is a screen row inside the panes, used to count dividers.
const bodyRow = 5

// mode returns the mode block of the statusline (the last screen row).
func mode(v string) string {
	rows := strings.Split(v, "\n")
	f := strings.Fields(rows[len(rows)-1])
	if len(f) < 2 {
		return ""
	}
	return f[1]
}

// reading opens doc and switches to reading mode.
func reading(t *testing.T, opts ...app.Option) *harness {
	t.Helper()
	h := openNote(t, opts...)
	h.keys(" r")
	if v := h.view(); mode(v) != "READING" {
		t.Fatalf("setup: space r should enter reading mode:\n%s", v)
	}
	return h
}

// openNoteWith opens a note holding body, the cursor on its first line.
func openNoteWith(t *testing.T, body string) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", body); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	return h
}

func TestCalculateLayoutReadingMode(t *testing.T) {
	cases := []struct {
		name         string
		width        int
		sidebar      bool
		editorX, edW int
	}{
		{"with explorer", 140, true, app.SidebarWidth + 1, 140 - app.SidebarWidth - 1},
		{"without explorer", 140, false, 0, 140},
		{"narrow window", app.MinTermWidth, true, app.SidebarWidth + 1, app.MinTermWidth - app.SidebarWidth - 1},
	}
	for _, c := range cases {
		// Reading mode hides the preview even when it was open.
		l := app.CalculateLayout(c.width, 40, c.sidebar, true, true, app.Split{EditorFrac: 0.3})
		if l.PreviewW != 0 || l.EditorX != c.editorX || l.EditorW != c.edW {
			t.Errorf("%s: %+v, want no preview and the editor at %d, %d wide", c.name, l, c.editorX, c.edW)
		}
	}
}

func TestReadingModeShowsOnlyTheEditorAndRestoresTheSplit(t *testing.T) {
	h := openNoteWith(t, "intro\n# Heading")
	div := dividers(h.view(), bodyRow)
	div0 := div[len(div)-1]
	h.send(
		tea.MouseMsg{X: div0, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: div0 - 12, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: div0 - 12, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
	before := h.view()

	h.keys(" r")
	v := h.view()
	if mode(v) != "READING" {
		t.Fatalf("the statusline should say READING:\n%s", v)
	}
	if got := dividers(v, bodyRow); len(got) != 1 || got[0] != app.SidebarWidth {
		t.Fatalf("only the explorer divider should remain, got %v:\n%s", got, v)
	}

	h.keys(" r")
	if after := h.view(); after != before {
		t.Fatalf("space r again should restore the frame exactly.\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestReadingModeEditsTheNote(t *testing.T) {
	h := reading(t)
	h.keys("A!")
	if v := h.view(); mode(v) != "INSERT" {
		t.Fatalf("i/A in reading mode should edit the note:\n%s", v)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got, _ := h.store.Get("doc.md"); got.Content != "hello!" {
		t.Fatalf("typing in reading mode should change and save the note, got %q", got.Content)
	}
	h.key(tea.KeyEsc)
	if v := h.view(); mode(v) != "READING" {
		t.Fatalf("back in Normal the statusline should say READING:\n%s", v)
	}
}

func TestReadingModeRendersTheMarkdown(t *testing.T) {
	h := openNoteWith(t, "intro\n# Heading")
	if v := h.view(); !strings.Contains(v, "# Heading") {
		t.Fatalf("setup: the split editor shows the raw heading:\n%s", v)
	}
	h.keys(" r")
	v := h.view()
	if strings.Contains(v, "# Heading") || !strings.Contains(v, "Heading") {
		t.Fatalf("reading mode should hide the heading's # off the cursor line:\n%s", v)
	}
}

func TestReadingModeTabAndCtrlHLMoveBetweenExplorerAndEditor(t *testing.T) {
	h := reading(t)
	steps := []struct {
		key  tea.KeyMsg
		want string
	}{
		{tea.KeyMsg{Type: tea.KeyTab}, "EXPLORER"},
		{tea.KeyMsg{Type: tea.KeyTab}, "READING"},
		{tea.KeyMsg{Type: tea.KeyShiftTab}, "EXPLORER"},
		{tea.KeyMsg{Type: tea.KeyCtrlL}, "READING"},
		{tea.KeyMsg{Type: tea.KeyCtrlL}, "READING"},
		{tea.KeyMsg{Type: tea.KeyCtrlH}, "EXPLORER"},
	}
	for i, s := range steps {
		h.send(s.key)
		if got := mode(h.view()); got != s.want {
			t.Fatalf("step %d (%s): mode %q, want %q", i, s.key, got, s.want)
		}
	}
}

func TestReadingModeEscGoesToTheEditor(t *testing.T) {
	h := reading(t)
	h.keys(" e")
	h.key(tea.KeyEsc)
	h.keys("A!")
	if v := h.view(); mode(v) != "INSERT" {
		t.Fatalf("Esc from the explorer should focus the editor:\n%s", v)
	}
}

func TestReadingModeFinderOpensNoteInTheEditor(t *testing.T) {
	h := reading(t)
	if _, err := h.store.SaveAs("", "other", "second note"); err != nil {
		t.Fatal(err)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("other")
	h.key(tea.KeyEnter)
	v := h.view()
	if mode(v) != "READING" || !strings.Contains(v, "second note") {
		t.Fatalf("the found note should open in reading mode:\n%s", v)
	}
}

func TestReadingModeSurvivesTheHomeScreen(t *testing.T) {
	h := reading(t)
	h.keys(" d")
	h.keys("1")
	if v := h.view(); mode(v) != "READING" {
		t.Fatalf("opening a note from home should stay in reading mode:\n%s", v)
	}
}

func TestReadingModeGraphLeavesIt(t *testing.T) {
	h := reading(t)
	h.keys(" g")
	v := h.view()
	if mode(v) == "READING" || !strings.Contains(strings.Split(v, "\n")[0], "Graph") {
		t.Fatalf("space g should leave reading mode and show the graph:\n%s", v)
	}
	if got := dividers(v, bodyRow); len(got) != 2 {
		t.Fatalf("the graph brings the split back, dividers %v:\n%s", got, v)
	}
}

func TestReadingModeSpacePRestoresTheSplit(t *testing.T) {
	h := openNote(t)
	before := dividers(h.view(), bodyRow)
	h.keys(" r")
	h.keys(" p")
	v := h.view()
	if mode(v) == "READING" {
		t.Fatalf("space p should leave reading mode:\n%s", v)
	}
	if got := dividers(v, bodyRow); len(got) != len(before) || got[len(got)-1] != before[len(before)-1] {
		t.Fatalf("space p should restore the split %v, got %v:\n%s", before, got, v)
	}
}

func TestSpacePOutsideReadingModeStillTogglesThePreview(t *testing.T) {
	h := openNote(t)
	h.keys(" p")
	if got := dividers(h.view(), bodyRow); len(got) != 1 {
		t.Fatalf("space p should hide the preview, dividers %v", got)
	}
	h.keys(" p")
	if got := dividers(h.view(), bodyRow); len(got) != 2 {
		t.Fatalf("space p should show the preview again, dividers %v", got)
	}
}

func TestReadingModeColonOpensTheCommandLine(t *testing.T) {
	h := reading(t)
	h.keys(":")
	v := h.view()
	rows := strings.Split(v, "\n")
	if !strings.HasPrefix(rows[len(rows)-1], ":") {
		t.Fatalf("':' should open the command line:\n%s", v)
	}
}

func TestReadingModeBlockMenuLeavesIt(t *testing.T) {
	h := reading(t)
	h.send(preview.AddBlockMsg{After: 1})
	if v := h.view(); mode(v) != "INSERT" {
		t.Fatalf("adding a block needs the editor:\n%s", v)
	}
}

func TestReadingModeExplorerDividerStillDrags(t *testing.T) {
	h := reading(t)
	x := app.SidebarWidth
	h.send(
		tea.MouseMsg{X: x, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: x + 5, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: x + 5, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
	if got := dividers(h.view(), bodyRow); len(got) != 1 || got[0] != x+5 {
		t.Fatalf("the explorer divider should move to %d, got %v", x+5, got)
	}
}

func TestReadingModeKeyIsConfigurable(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"reading_mode": {}}))
	h.keys(" r")
	if v := h.view(); mode(v) == "READING" {
		t.Fatalf("reading_mode: [] should unbind space r:\n%s", v)
	}

	h = openNote(t, app.WithKeys(map[string][]string{"reading_mode": {"space v"}}))
	h.keys(" r")
	if mode(h.view()) == "READING" {
		t.Fatal("space r was replaced and must do nothing now")
	}
	h.keys(" v")
	if v := h.view(); mode(v) != "READING" {
		t.Fatalf("space v should now enter reading mode:\n%s", v)
	}
}

func TestHelpListsReadingMode(t *testing.T) {
	h := openNote(t)
	h.keys("?")
	if v := h.view(); !strings.Contains(v, "SPC r") || !strings.Contains(v, "Reading mode") {
		t.Fatalf("the help should list reading mode:\n%s", v)
	}
}

// textAt returns the screen cell of the leftmost text on screen, and the
// first column of the pane holding it (just past the explorer's divider).
func textAt(t *testing.T, v, text string) (x, y, paneX int) {
	t.Helper()
	for y, row := range strings.Split(v, "\n") {
		r := []rune(row)
		i := strings.Index(row, text)
		if i < 0 {
			continue
		}
		x = len([]rune(row[:i]))
		for c := x - 1; c >= 0; c-- {
			if r[c] == '│' {
				return x, y, c + 1
			}
		}
		return x, y, 0
	}
	t.Fatalf("%q not on screen:\n%s", text, v)
	return 0, 0, 0
}

// clickInsert clicks the fourth cell of the on-screen "intro" and types X
// there, returning the note's text.
func clickInsert(t *testing.T, h *harness) string {
	t.Helper()
	x, y, _ := textAt(t, h.view(), "intro")
	h.click(x+3, y)
	h.keys("iX")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	got, _ := h.store.Get("doc.md")
	return got.Content
}

func TestReadingModeHasThePreviewMargins(t *testing.T) {
	long := strings.Repeat("word ", 60)
	h := openNoteWith(t, "intro\n"+long)
	h.keys(" r")
	v := h.view()
	x, _, paneX := textAt(t, v, "intro")
	paneW := termW - paneX
	margin := preview.PageMargin(paneW)
	if margin == 0 || x != paneX+margin {
		t.Fatalf("reading text starts at column %d, want %d (pane at %d, margin %d):\n%s", x, paneX+margin, paneX, margin, v)
	}
	for _, row := range strings.Split(v, "\n") {
		if !strings.Contains(row, "word word") { // not the statusline's word count
			continue
		}
		r := []rune(strings.TrimRight(row, " "))
		if end := len(r); end > termW-margin {
			t.Fatalf("reading text runs to column %d, past the right margin at %d:\n%s", end, termW-margin, v)
		}
	}
	if got := clickInsert(t, h); !strings.HasPrefix(got, "intXro\n") {
		t.Fatalf("a click in reading mode should land on the text under it, got %q", got)
	}
}

func TestSplitEditorKeepsItsColumns(t *testing.T) {
	h := openNoteWith(t, "intro\n# Heading")
	x, _, paneX := textAt(t, h.view(), "intro")
	if x != paneX+6 { // the line-number gutter, no margin
		t.Fatalf("split editor text at column %d, want %d", x, paneX+6)
	}
	if got := clickInsert(t, h); !strings.HasPrefix(got, "intXro\n") {
		t.Fatalf("a click in the split editor should land on the text under it, got %q", got)
	}
}
