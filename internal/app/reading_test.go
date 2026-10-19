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

func TestCalculateLayoutReadingMode(t *testing.T) {
	cases := []struct {
		name          string
		width         int
		sidebar       bool
		previewX, prW int
	}{
		{"with explorer", 140, true, app.SidebarWidth + 1, 140 - app.SidebarWidth - 1},
		{"without explorer", 140, false, 0, 140},
		{"narrow window", app.MinTermWidth, true, app.SidebarWidth + 1, app.MinTermWidth - app.SidebarWidth - 1},
	}
	for _, c := range cases {
		// Reading mode shows the preview even when it was hidden.
		l := app.CalculateLayout(c.width, 40, c.sidebar, false, true, app.Split{EditorFrac: 0.3})
		if l.EditorW != 0 || l.PreviewX != c.previewX || l.PreviewW != c.prW {
			t.Errorf("%s: %+v, want no editor and the preview at %d, %d wide", c.name, l, c.previewX, c.prW)
		}
	}
}

func TestReadingModeShowsOnlyThePreviewAndRestoresTheSplit(t *testing.T) {
	h := openNote(t)
	before := dividers(h.view(), bodyRow)
	div := before[len(before)-1]
	h.send(
		tea.MouseMsg{X: div, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: div - 12, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: div - 12, Y: bodyRow, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
	dragged := dividers(h.view(), bodyRow)

	h.keys(" r")
	v := h.view()
	if mode(v) != "READING" {
		t.Fatalf("the statusline should say READING:\n%s", v)
	}
	if got := dividers(v, bodyRow); len(got) != 1 || got[0] != app.SidebarWidth {
		t.Fatalf("only the explorer divider should remain, got %v:\n%s", got, v)
	}

	h.keys(" r")
	v = h.view()
	if mode(v) == "READING" {
		t.Fatalf("space r again should leave reading mode:\n%s", v)
	}
	if got := dividers(v, bodyRow); len(got) != len(dragged) || got[len(got)-1] != dragged[len(dragged)-1] {
		t.Fatalf("dividers at %v after reading mode, want %v as dragged", got, dragged)
	}
}

func TestReadingModeTabAndCtrlHLSkipTheEditor(t *testing.T) {
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

func TestReadingModeEscGoesToThePreview(t *testing.T) {
	h := reading(t)
	h.keys(" e")
	h.key(tea.KeyEsc)
	if v := h.view(); mode(v) != "READING" {
		t.Fatalf("Esc must not focus the hidden editor:\n%s", v)
	}
}

func TestReadingModeFinderOpensNoteInThePreview(t *testing.T) {
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
		t.Fatalf("the graph needs the editor column, dividers %v:\n%s", got, v)
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
	if !strings.HasPrefix(rows[len(rows)-1], ":") || len(dividers(v, bodyRow)) != 2 {
		t.Fatalf("':' should leave reading mode and open the command line:\n%s", v)
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
