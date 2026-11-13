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

// reading opens doc and switches to the reading view (space p).
func reading(t *testing.T, opts ...app.Option) *harness {
	t.Helper()
	h := openNote(t, opts...)
	h.keys(" p")
	if v := h.view(); !inReadingView(v) || mode(v) != "NORMAL" {
		t.Fatalf("setup: space p should open the reading view:\n%s", v)
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

// editorView opens doc and switches to the editor view (space r).
func editorView(t *testing.T, body string) *harness {
	t.Helper()
	h := openNoteWith(t, body)
	h.keys(" r")
	if v := h.view(); mode(v) != "NORMAL" {
		t.Fatalf("setup: space r should keep the vim mode on the statusline:\n%s", v)
	}
	return h
}

// saved saves the open note and returns doc.md's text on disk.
func saved(t *testing.T, h *harness) string {
	t.Helper()
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	got, err := h.store.Get("doc.md")
	if err != nil {
		t.Fatal(err)
	}
	return got.Content
}

// S1, S15: space p shows the note alone, full width, without line numbers,
// and in NORMAL the block holding the cursor rendered too; space p again
// restores the split exactly.
func TestReadingViewShowsOnlyTheRenderedNoteAndRestoresTheSplit(t *testing.T) {
	h := openNoteWith(t, "# Heading\nintro")
	div := dividers(h.view(), bodyRow)
	div0 := div[len(div)-1]
	h.send(
		tea.MouseMsg{X: div0, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: div0 - 12, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: div0 - 12, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
	before := h.view()

	h.keys(" p")
	v := h.view()
	if mode(v) != "NORMAL" {
		t.Fatalf("the statusline should show the real mode:\n%s", v)
	}
	if got := dividers(v, bodyRow); len(got) != 1 || got[0] != app.SidebarWidth {
		t.Fatalf("only the explorer divider should remain, got %v:\n%s", got, v)
	}
	if strings.Contains(v, "# Heading") || !strings.Contains(v, "Heading") {
		t.Fatalf("the block holding the cursor should be rendered in NORMAL:\n%s", v)
	}
	if x, _, paneX := textAt(t, v, "intro"); x == paneX+6 {
		t.Fatalf("the reading view should have no line-number gutter:\n%s", v)
	}

	h.keys(" p")
	if after := h.view(); after != before {
		t.Fatalf("space p again should restore the frame exactly.\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// S1: the keys that edit work in the reading view.
func TestReadingViewIsEditable(t *testing.T) {
	h := openNoteWith(t, "hello\n- [ ] task")
	h.keys(" p")
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.keys("j")
	h.key(tea.KeyEnter) // on the box line: Enter ticks it
	if got := saved(t, h); got != "hello!\n- [x] task" {
		t.Fatalf("the reading view should edit the note, got %q", got)
	}
}

// S1: space o c (checkbox action) edits in the reading view.
func TestReadingViewRunsCheckboxActions(t *testing.T) {
	h := openNoteWith(t, "- [ ] task")
	h.keys(" p")
	h.keys(" oc")
	if got := saved(t, h); got != "- [x] task" {
		t.Fatalf("the checkbox action should edit in the reading view, got %q", got)
	}
}

// S1: Enter and gd still follow a [[link]] in the reading view.
func TestReadingViewFollowsLinks(t *testing.T) {
	for _, follow := range []string{"enter", "gd"} {
		h := openStart(t, linkedNotes(t))
		h.key(tea.KeyEsc)
		h.keys(" p")
		h.keys("gg0" + strings.Repeat("l", 16)) // onto "[[Docker]]"
		if follow == "enter" {
			h.key(tea.KeyEnter)
		} else {
			h.keys("gd")
		}
		if v := h.view(); !strings.Contains(v, "containers") || !inReadingView(v) {
			t.Fatalf("%s should open Docker.md in the reading view:\n%s", follow, v)
		}
	}
}

// S2: space r shows the Markdown editor alone at full width, with line
// numbers and raw markers, editable; space r again restores the split.
func TestEditorViewShowsOnlyTheEditableMarkdown(t *testing.T) {
	h := openNoteWith(t, "# Heading\nintro")
	before := h.view()
	h.keys(" r")
	v := h.view()
	if mode(v) != "NORMAL" {
		t.Fatalf("the editor view keeps the vim mode label:\n%s", v)
	}
	if got := dividers(v, bodyRow); len(got) != 1 || got[0] != app.SidebarWidth {
		t.Fatalf("only the explorer divider should remain, got %v:\n%s", got, v)
	}
	if !strings.Contains(v, "# Heading") {
		t.Fatalf("the editor view shows the raw Markdown:\n%s", v)
	}
	if x, _, paneX := textAt(t, v, "intro"); x != paneX+6 {
		t.Fatalf("editor view text at column %d, want %d (line numbers, no margin):\n%s", x, paneX+6, v)
	}
	h.keys("A!")
	if v := h.view(); mode(v) != "INSERT" {
		t.Fatalf("A in the editor view should edit:\n%s", v)
	}
	h.key(tea.KeyEsc)
	if got := saved(t, h); got != "# Heading!\nintro" {
		t.Fatalf("typing in the editor view should change the note, got %q", got)
	}
	h.keys(" r")
	if got := dividers(h.view(), bodyRow); len(got) != len(dividers(before, bodyRow)) {
		t.Fatalf("space r again should restore the split, dividers %v", got)
	}
}

// S3: space p and space r switch straight between the two views.
func TestReadingAndEditorViewsSwitchDirectly(t *testing.T) {
	h := openNoteWith(t, "intro\n# Heading")
	h.keys(" p")
	h.keys(" r")
	v := h.view()
	if mode(v) != "NORMAL" || !strings.Contains(v, "# Heading") || len(dividers(v, bodyRow)) != 1 {
		t.Fatalf("space r from the reading view should open the editor view:\n%s", v)
	}
	h.keys(" p")
	v = h.view()
	if mode(v) != "NORMAL" || strings.Contains(v, "# Heading") || !inReadingView(v) {
		t.Fatalf("space p from the editor view should open the reading view:\n%s", v)
	}
}

// S4: there is no key left that only hides the preview: both views go back
// to the split, and the split is where the app starts.
func TestBothViewsReturnToTheSplit(t *testing.T) {
	for _, k := range []string{" p", " r"} {
		h := openNote(t)
		h.keys(k)
		h.keys(k)
		if got := dividers(h.view(), bodyRow); len(got) != 2 {
			t.Fatalf("%q twice should end in the split, dividers %v", k, got)
		}
	}
}

func TestReadingModeTabAndCtrlHLMoveBetweenExplorerAndEditor(t *testing.T) {
	h := reading(t)
	steps := []struct {
		key  tea.KeyMsg
		want string
	}{
		{tea.KeyMsg{Type: tea.KeyTab}, "EXPLORER"},
		{tea.KeyMsg{Type: tea.KeyTab}, "NORMAL"},
		{tea.KeyMsg{Type: tea.KeyShiftTab}, "EXPLORER"},
		{tea.KeyMsg{Type: tea.KeyCtrlL}, "NORMAL"},
		{tea.KeyMsg{Type: tea.KeyCtrlL}, "NORMAL"},
		{tea.KeyMsg{Type: tea.KeyCtrlH}, "EXPLORER"},
	}
	for i, s := range steps {
		h.send(s.key)
		if got := mode(h.view()); got != s.want {
			t.Fatalf("step %d (%s): mode %q, want %q", i, s.key, got, s.want)
		}
	}
}

// S3: in the editor view the pane keys move between explorer and editor.
func TestEditorViewTabAndCtrlHLMoveBetweenExplorerAndEditor(t *testing.T) {
	h := editorView(t, "hello")
	steps := []struct {
		key  tea.KeyMsg
		want string
	}{
		{tea.KeyMsg{Type: tea.KeyTab}, "EXPLORER"},
		{tea.KeyMsg{Type: tea.KeyTab}, "NORMAL"},
		{tea.KeyMsg{Type: tea.KeyCtrlH}, "EXPLORER"},
		{tea.KeyMsg{Type: tea.KeyCtrlL}, "NORMAL"},
		{tea.KeyMsg{Type: tea.KeyCtrlL}, "NORMAL"},
		{tea.KeyMsg{Type: tea.KeyCtrlLeft}, "EXPLORER"},
		{tea.KeyMsg{Type: tea.KeyCtrlRight}, "NORMAL"},
	}
	for i, s := range steps {
		h.send(s.key)
		if got := mode(h.view()); got != s.want {
			t.Fatalf("step %d (%s): mode %q, want %q", i, s.key, got, s.want)
		}
	}
}

// S3: the explorer toggle works in both views and leaves the view as it was.
func TestExplorerToggleKeepsTheView(t *testing.T) {
	for _, c := range []struct{ key, mode string }{{" p", "NORMAL"}, {" r", "NORMAL"}} {
		h := openNote(t)
		h.keys(c.key)
		h.send(tea.KeyMsg{Type: tea.KeyCtrlB})
		if v := h.view(); mode(v) != c.mode || len(dividers(v, bodyRow)) != 0 {
			t.Fatalf("%q: ctrl+b should hide the explorer and keep the view %s:\n%s", c.key, c.mode, v)
		}
		h.send(tea.KeyMsg{Type: tea.KeyCtrlB})
		if v := h.view(); mode(v) != "EXPLORER" || len(dividers(v, bodyRow)) != 1 {
			t.Fatalf("%q: ctrl+b should bring the explorer back beside the full-width note:\n%s", c.key, v)
		}
		h.send(tea.KeyMsg{Type: tea.KeyCtrlL})
		if v := h.view(); mode(v) != c.mode {
			t.Fatalf("%q: ctrl+l should go back to the view %s:\n%s", c.key, c.mode, v)
		}
	}
}

func TestReadingModeEscGoesToTheEditor(t *testing.T) {
	h := reading(t)
	h.keys(" e")
	h.key(tea.KeyEsc)
	if v := h.view(); mode(v) != "NORMAL" || !inReadingView(v) {
		t.Fatalf("Esc from the explorer should focus the reading view:\n%s", v)
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
	if !inReadingView(v) || !strings.Contains(v, "second note") {
		t.Fatalf("the found note should open in reading mode:\n%s", v)
	}
}

func TestReadingModeSurvivesTheHomeScreen(t *testing.T) {
	h := reading(t)
	h.keys(" d")
	h.keys("1")
	if v := h.view(); !inReadingView(v) {
		t.Fatalf("opening a note from home should stay in reading mode:\n%s", v)
	}
}

func TestReadingModeGraphLeavesIt(t *testing.T) {
	h := reading(t)
	h.keys(" g")
	v := h.view()
	if inReadingView(v) || !strings.Contains(strings.Split(v, "\n")[0], "Graph") {
		t.Fatalf("space g should leave reading mode and show the graph:\n%s", v)
	}
	if got := dividers(v, bodyRow); len(got) != 2 {
		t.Fatalf("the graph brings the split back, dividers %v:\n%s", got, v)
	}
}

// S13: a new note is typed in the reading view, which stays.
func TestReadingViewNewNoteStaysInTheView(t *testing.T) {
	h := reading(t)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlN})
	h.keys("abc")
	if v := h.view(); mode(v) != "INSERT" || !strings.Contains(v, "abc") || !inReadingView(v) {
		t.Fatalf("a new note from the reading view should be typed in the view:\n%s", v)
	}
}

func TestReadingModeColonOpensTheCommandLine(t *testing.T) {
	h := reading(t)
	h.keys(":")
	v := h.view()
	rows := strings.Split(v, "\n")
	if !strings.Contains(rows[cmdlineRow], " Cmdline ") || !strings.Contains(rows[cmdlineRow+1], "│ › ") ||
		mode(v) != "COMMAND" || !inReadingView(v) {
		t.Fatalf("':' should open the floating command line over the reading view:\n%s", v)
	}
}

// S13: the block menu opens without leaving the view.
func TestReadingModeBlockMenuStays(t *testing.T) {
	h := reading(t)
	h.send(preview.AddBlockMsg{After: 1})
	if v := h.view(); mode(v) != "INSERT" || !inReadingView(v) {
		t.Fatalf("adding a block should stay in the view:\n%s", v)
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

func TestReadingViewKeyIsConfigurable(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"reading_view": {}}))
	h.keys(" p")
	if v := h.view(); inReadingView(v) {
		t.Fatalf("reading_view: [] should unbind space p:\n%s", v)
	}

	h = openNote(t, app.WithKeys(map[string][]string{"reading_view": {"space v"}}))
	h.keys(" p")
	if inReadingView(h.view()) {
		t.Fatal("space p was replaced and must do nothing now")
	}
	h.keys(" v")
	if v := h.view(); !inReadingView(v) {
		t.Fatalf("space v should now open the reading view:\n%s", v)
	}
}

// S4: the old action names still work. reading_mode was the full-width
// rendered view, so it names the reading view; toggle_preview left the
// editor alone, so it names the editor view. The new name wins over the old.
func TestOldViewActionNamesStillWork(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"reading_mode": {"space v"}}))
	h.keys(" v")
	if v := h.view(); !inReadingView(v) || strings.Contains(v, "unknown action") {
		t.Fatalf("reading_mode should bind the reading view without a warning:\n%s", v)
	}

	h = openNote(t, app.WithKeys(map[string][]string{"toggle_preview": {"space v"}}))
	h.keys(" v")
	if v := h.view(); mode(v) != "NORMAL" || len(dividers(v, bodyRow)) != 1 || strings.Contains(v, "unknown action") {
		t.Fatalf("toggle_preview should bind the editor view without a warning:\n%s", v)
	}

	h = openNote(t, app.WithKeys(map[string][]string{"reading_mode": {"space v"}, "reading_view": {"space z"}}))
	h.keys(" z")
	if v := h.view(); !inReadingView(v) {
		t.Fatalf("reading_view should win over reading_mode:\n%s", v)
	}
}

func TestHelpListsBothViews(t *testing.T) {
	h := openNote(t)
	h.keys("?")
	v := h.view()
	if !strings.Contains(v, "SPC p") || !strings.Contains(v, "Reading view") {
		t.Fatalf("the help should list the reading view:\n%s", v)
	}
	if !strings.Contains(v, "SPC r") || !strings.Contains(v, "Editor view") {
		t.Fatalf("the help should list the editor view:\n%s", v)
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
	h.keys(" p")
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
	cx, cy, _ := textAt(t, v, "intro")
	h.click(cx+3, cy)
	h.keys(" r") // the cursor stays where the click put it
	h.keys("iX")
	h.key(tea.KeyEsc)
	if got := saved(t, h); !strings.HasPrefix(got, "intXro\n") {
		t.Fatalf("a click in the reading view should land on the text under it, got %q", got)
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
