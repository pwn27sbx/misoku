package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// tutorHarness opens lesson in a fresh tutorial vault played WithTutor, with
// the editor at the end of the note in INSERT mode on a new line.
func tutorHarness(t *testing.T, lesson string) *harness {
	t.Helper()
	store := newTestStore(t)
	if err := tutor.Install(tutor.Bundled(), store.Root(), false); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithTutor(tutor.Scan(store.Root())))
	m, cmd := app.OpenInCurrentBuffer(h.m, lesson)
	h.m = m
	h.send(cmdMsg(cmd)...)
	h.keys("G")
	h.keys("o")
	return h
}

// cmdMsg runs cmd, if any, and returns its message for harness.send.
func cmdMsg(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	if msg := runCmd(cmd); msg != nil {
		return []tea.Msg{msg}
	}
	return nil
}

// statusOf is the last line of the frame: the statusline.
func statusOf(view string) string {
	lines := strings.Split(view, "\n")
	return lines[len(lines)-1]
}

// S4: a passed level is written to its note at once, so a quit that
// discards everything else keeps the progress the next Scan reads.
func TestTutorSavesAPassedLevelAtOnce(t *testing.T) {
	h := tutorHarness(t, "02 Casillas.md")
	h.keys("- [x] hecho")
	root := h.store.Root()
	lesson := filepath.Join(root, "02 Casillas.md")
	if data, err := os.ReadFile(lesson); err != nil || !strings.Contains(string(data), "- [x]") {
		t.Fatalf("the passed level was not saved at once (%v):\n%s", err, data)
	}
	if done, total := tutor.Scan(root).Count(); done != 1 || total != 7 {
		t.Fatalf("the disk counts %d/%d right after the level passed, want 1/7", done, total)
	}

	// An edit after the level is nobody's business but the discard's.
	h.keys(" y algo más")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	if h.quit {
		t.Fatal("unsaved changes must ask before quitting")
	}
	h.keys("d") // Discard
	if !h.quit {
		t.Fatal("Discard should quit")
	}
	if done, total := tutor.Scan(root).Count(); done != 1 || total != 7 {
		t.Fatalf("after quitting with Discard the progress is %d/%d, want 1/7", done, total)
	}
	if data, _ := os.ReadFile(lesson); strings.Contains(string(data), "y algo más") {
		t.Fatalf("Discard kept the later edit:\n%s", data)
	}
}

// S5, S6: ticking a box in 02 passes level 2 once, and the bar counts it.
func TestTutorAnnouncesALevelOnceAndCountsIt(t *testing.T) {
	h := tutorHarness(t, "02 Casillas.md")
	if s := statusOf(h.view()); !strings.Contains(s, "Tutor 0/7") {
		t.Fatalf("the statusline should show Tutor 0/7 before any level:\n%s", s)
	}
	h.keys("- [x] hecho")
	v := h.view()
	if !strings.Contains(v, "✓ Nivel 2 superado (1/7)") {
		t.Fatalf("passing level 2 should toast it:\n%s", v)
	}
	if s := statusOf(v); !strings.Contains(s, "Tutor 1/7") {
		t.Fatalf("the statusline should show Tutor 1/7:\n%s", s)
	}
	h.m = app.ExpireToast(h.m)
	h.keys(" más")
	if v := h.view(); strings.Contains(v, "Nivel 2 superado") {
		t.Fatalf("a later edit announced level 2 again:\n%s", v)
	}
}

// S5: "- []" in 02 explains the missing space, once while the note is open.
func TestTutorHintsTheMissingSpace(t *testing.T) {
	h := tutorHarness(t, "02 Casillas.md")
	h.keys("- [] x")
	hint := "- [] no es una tarea: falta el espacio → - [ ]"
	if v := h.view(); !strings.Contains(v, hint) {
		t.Fatalf("writing - [] should toast the hint:\n%s", v)
	}
	h.m = app.ExpireToast(h.m)
	h.keys("y")
	if v := h.view(); strings.Contains(v, hint) {
		t.Fatalf("the hint came back on the next edit:\n%s", v)
	}
}

// S7: without WithTutor the statusline is the one Misoku drew before the
// tutorial existed, and editing never toasts a level.
func TestNoTutorLeavesTheStatuslineAsItWas(t *testing.T) {
	store := newTestStore(t)
	if err := tutor.Install(tutor.Bundled(), store.Root(), false); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	if got := statusOf(h.view()); got != baseStatusline {
		t.Fatalf("statusline changed without the tutor:\n got %q\nwant %q", got, baseStatusline)
	}
	m, cmd := app.OpenInCurrentBuffer(h.m, "02 Casillas.md")
	h.m = m
	h.send(cmdMsg(cmd)...)
	h.keys("Go- [x] hecho")
	if v := h.view(); strings.Contains(v, "Nivel 2 superado") || strings.Contains(statusOf(v), "Tutor") {
		t.Fatalf("without the tutor nothing about it shows:\n%s", v)
	}
}

// baseStatusline is the home screen's statusline at base aaa94fb for this
// test's vault (the folder is t.TempDir's second one, "002").
const baseStatusline = " \uf015 HOME \ue0b4" + "                                                                                                                      " +
	"\ue0b6\U000f024b  002 \ue0b6\ue612   "
