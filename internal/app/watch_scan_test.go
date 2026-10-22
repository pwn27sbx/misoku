package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// collect runs c like the runtime would but feeds nothing back: it returns
// the messages c produced and how many clock commands it held. A test uses
// it to hold a workspace scan's result and deliver it later.
func collect(c tea.Cmd) (msgs []tea.Msg, timers int) {
	if c == nil {
		return nil, 0
	}
	if isTimer(c) {
		return nil, 1
	}
	switch out := c().(type) {
	case nil:
	case tea.BatchMsg:
		for _, sub := range out {
			m, n := collect(sub)
			msgs = append(msgs, m...)
			timers += n
		}
	default:
		msgs = append(msgs, out)
	}
	return msgs, timers
}

// startCheck sends a workspace check straight to the model, without running
// what it returns, and returns the scan's result message.
func startCheck(t *testing.T, h *harness) tea.Msg {
	t.Helper()
	var cmd tea.Cmd
	h.m, cmd = h.m.Update(app.CheckWorkspaceMsg{})
	msgs, timers := collect(cmd)
	if timers != 0 {
		t.Fatalf("the next check must wait for this one's result, got %d timers", timers)
	}
	if len(msgs) != 1 {
		t.Fatalf("a check should produce exactly one result message, got %d", len(msgs))
	}
	return msgs[0]
}

func TestCheckWorkspaceAppliesChangesThroughItsResult(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "first version"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.key(tea.KeyEsc)

	writeOutside(t, store.Root(), "doc.md", "second version from outside")
	writeOutside(t, store.Root(), "Outside.md", "y")
	result := startCheck(t, h)
	if v := h.view(); strings.Contains(v, "second version") {
		t.Fatalf("the check must not read the disk inside Update:\n%s", v)
	}
	h.send(result)
	if v := h.view(); !strings.Contains(v, "second version from outside") {
		t.Fatalf("the result should reload the unmodified note:\n%s", v)
	}
}

func TestStaleScanDoesNotUndoASaveMadeWhileItRan(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.keys("A mine")
	h.key(tea.KeyEsc)

	result := startCheck(t, h) // reads "base" from disk
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	h.send(result)

	v := h.view()
	if !strings.Contains(v, "base mine") {
		t.Fatalf("the saved text must stay in the editor:\n%s", v)
	}
	for _, bad := range []string{"reloaded", "changed on disk"} {
		if strings.Contains(v, bad) {
			t.Fatalf("an old scan must not report %q:\n%s", bad, v)
		}
	}
	if strings.Contains(v, "●") {
		t.Fatalf("the saved note must not look modified:\n%s", v)
	}
	if got := readNote(t, store.Root(), "doc.md"); got != "base mine" {
		t.Fatalf("the file must keep the saved text, got %q", got)
	}
	// Further edits still save on top of the saved text.
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got := readNote(t, store.Root(), "doc.md"); got != "base mine!" {
		t.Fatalf("the next save must build on the saved text, got %q", got)
	}
}

func TestTypingWhileAScanRunsKeepsTheTextAndWarns(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.key(tea.KeyEsc)

	writeOutside(t, store.Root(), "doc.md", "theirs")
	result := startCheck(t, h)
	h.keys("A mine")
	h.key(tea.KeyEsc)
	h.send(result)

	v := h.view()
	if !strings.Contains(v, "base mine") {
		t.Fatalf("text typed during the scan must stay:\n%s", v)
	}
	if !strings.Contains(v, "changed on disk") {
		t.Fatalf("the user should be told the file changed on disk:\n%s", v)
	}
}

func TestOnlyOneWorkspaceScanRunsAtATime(t *testing.T) {
	store := newTestStore(t)
	h := newHarness(t, store)

	result := startCheck(t, h)
	var cmd tea.Cmd
	h.m, cmd = h.m.Update(app.CheckWorkspaceMsg{})
	if msgs, timers := collect(cmd); len(msgs) != 0 || timers != 0 {
		t.Fatalf("a check while one runs must start nothing, got %d messages and %d timers", len(msgs), timers)
	}

	// The result schedules the next check; after it a new scan may start.
	h.m, cmd = h.m.Update(result)
	if msgs, timers := collect(cmd); timers != 1 || len(msgs) != 0 {
		t.Fatalf("the result should schedule exactly the next check, got %d timers and %d messages", timers, len(msgs))
	}
	startCheck(t, h)
}

func TestDeletedNoteIsDetectedThroughTheScanResult(t *testing.T) {
	h := openDoc(t, false)
	removeOutside(t, h.store.Root(), "doc.md")
	result := startCheck(t, h)
	if v := h.view(); strings.Contains(v, goneWarning) {
		t.Fatalf("the check must not stat notes inside Update:\n%s", v)
	}
	h.send(result)
	if v := h.view(); !strings.Contains(v, goneWarning) {
		t.Fatalf("the tab should warn that its note is gone:\n%s", v)
	}
}

func TestRecreatedNoteIgnoresAnOldGoneResult(t *testing.T) {
	h := openDoc(t, true)
	removeOutside(t, h.store.Root(), "doc.md")
	h.send(app.CheckWorkspaceMsg{}) // the tab learns its note is gone
	result := startCheck(t, h)      // still gone on disk
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	h.key(tea.KeyEnter) // Recreate
	if got := readNote(t, h.store.Root(), "doc.md"); got != "base mine" {
		t.Fatalf("setup: the note should be recreated, got %q", got)
	}
	h.send(result)
	if v := h.view(); strings.Contains(v, goneWarning) {
		t.Fatalf("an old scan must not mark a recreated note gone:\n%s", v)
	}
}

func TestUserThemeReloadsThroughTheScanResult(t *testing.T) {
	dir := t.TempDir()
	theme.SetUserThemesPath(dir)
	t.Cleanup(func() { theme.SetUserThemesPath("") })
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	path := filepath.Join(dir, "mine.json")
	start := time.Now().Add(-time.Hour)
	writePalette(t, path, "#aa0000", start)
	h := newHarness(t, newTestStore(t), app.WithTheme("mine"))

	writePalette(t, path, "#00bb00", start.Add(time.Minute))
	result := startCheck(t, h)
	if strings.Contains(h.m.View(), sgr("#00bb00")) {
		t.Fatal("the check must not reload themes inside Update")
	}
	h.send(result)
	if v := h.m.View(); !strings.Contains(v, sgr("#00bb00")) {
		t.Fatal("the result should apply the edited palette")
	}
}

func TestOldScanDoesNotDropANoteSavedWhileItRan(t *testing.T) {
	store := newTestStore(t)
	if err := os.MkdirAll(filepath.Join(store.Root(), "journal"), 0o755); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	result := startCheck(t, h) // the empty journal folder is hidden

	h.keys("n")
	h.keys("hello")
	h.key(tea.KeyEsc)
	h.keys(":w")
	h.key(tea.KeyEnter)
	v := h.view()
	_, top := find(t, v, "Folder")
	x, y := find(t, strings.Join(strings.Split(v, "\n")[top:], "\n"), "journal")
	h.click(x, y+top)
	h.key(tea.KeyEnter)
	if got := readNote(t, store.Root(), "journal/Untitled-1.md"); got != "hello" {
		t.Fatalf("setup: the note should be saved, got %q", got)
	}

	h.send(result)
	v = h.view()
	listed := false
	for _, l := range strings.Split(v, "\n") {
		listed = listed || strings.Contains(l, "└") && strings.Contains(l, "Untitled-1.md")
	}
	if !listed {
		t.Fatalf("an old scan must not drop a note saved meanwhile from the explorer:\n%s", v)
	}
}
