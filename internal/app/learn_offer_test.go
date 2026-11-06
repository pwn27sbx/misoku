package app_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// S1: the first open offers the three answers, with the cat and no letter
// shortcuts of its own.
func TestFirstRunDialogOffersTheThreeAnswers(t *testing.T) {
	store := newTestStore(t)
	h := newHarness(t, store, app.WithLearnOffer(true), app.WithConfigPath(filepath.Join(t.TempDir(), "config.json")))
	v := h.view()
	for _, want := range []string{"Soy nuevo", "Ya tengo experiencia", "Ahora no", "^.^"} {
		if !strings.Contains(v, want) {
			t.Errorf("the first-run dialog should show %q:\n%s", want, v)
		}
	}
	// deliberate: a letter picks nothing, so S, Y and A never answer for the
	// user.
	h.keys("s")
	if v := h.view(); !strings.Contains(v, "Soy nuevo") {
		t.Fatalf("s closed the dialog:\n%s", v)
	}
}

// S2: each answer is remembered in the config and closes the dialog once.
func TestFirstRunDialogRemembersEachAnswer(t *testing.T) {
	answers := []struct {
		name     string
		keys     func(*harness)
		tutorial bool
	}{
		{"Soy nuevo", func(h *harness) { h.key(tea.KeyEnter) }, true},
		{"Ya tengo experiencia", func(h *harness) { h.keys("j"); h.key(tea.KeyEnter) }, false},
		{"Ahora no", func(h *harness) { h.keys("jj"); h.key(tea.KeyEnter) }, false},
		{"n", func(h *harness) { h.keys("n") }, true},
		{"e", func(h *harness) { h.keys("e") }, false},
		{"a", func(h *harness) { h.keys("a") }, false},
		{"Esc", func(h *harness) { h.key(tea.KeyEsc) }, false},
		{"down then Enter", func(h *harness) { h.key(tea.KeyDown); h.key(tea.KeyEnter) }, false},
		{"down, up then Enter", func(h *harness) { h.key(tea.KeyDown); h.key(tea.KeyUp); h.key(tea.KeyEnter) }, true},
	}
	for _, a := range answers {
		t.Run(a.name, func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), "config.json")
			var got *exec.Cmd
			h := newHarness(t, newTestStore(t), app.WithLearnOffer(true), app.WithConfigPath(cfgPath),
				fakeTutorial(&got, nil, nil))
			a.keys(h)
			if v := h.view(); strings.Contains(v, "Soy nuevo") {
				t.Fatalf("%s should close the dialog:\n%s", a.name, v)
			}
			if h.quit {
				t.Fatalf("%s quit Misoku", a.name)
			}
			if !learnOffered(t, cfgPath) {
				t.Fatalf("%s was not remembered in the config", a.name)
			}
			if started := got != nil; started != a.tutorial {
				t.Errorf("%s started the tutorial: %v, want %v", a.name, started, a.tutorial)
			}
		})
	}
}

// S1: Ctrl+C quits without recording anything, so the dialog is offered
// again next time.
func TestFirstRunDialogCtrlCLeavesTheAnswerUnwritten(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	h := newHarness(t, newTestStore(t), app.WithLearnOffer(true), app.WithConfigPath(cfgPath),
		fakeTutorial(new(*exec.Cmd), nil, nil))
	h.key(tea.KeyCtrlC)
	if !h.quit {
		t.Fatal("Ctrl+C should quit Misoku")
	}
	if _, err := os.Stat(cfgPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Ctrl+C wrote the config: %v", err)
	}
}

// S8: below 60x12 the screen says the terminal is too small, and a blind
// Enter must not reach the dialog.
func TestFirstRunDialogWaitsForARoomierTerminal(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	store := newTestStore(t)
	h := newHarnessSize(t, 50, 10, store, app.WithLearnOffer(true), app.WithConfigPath(cfgPath),
		fakeTutorial(new(*exec.Cmd), nil, nil))
	if v := h.screen(); !strings.Contains(v, "Terminal too small") {
		t.Fatalf("a small terminal says so:\n%s", v)
	}
	h.key(tea.KeyEnter)
	if v := h.screen(); strings.Contains(v, "Soy nuevo") {
		t.Fatalf("the dialog took a blind Enter:\n%s", v)
	}
	if h.quit {
		t.Fatalf("a blind Enter quit Misoku")
	}
	if _, err := os.Stat(cfgPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a blind Enter recorded an answer: %v", err)
	}
}

// S8: a resize that stops the dialog fitting closes it and keeps it pending.
func TestFirstRunDialogClosesWhenAResizeStopsItFitting(t *testing.T) {
	store := newTestStore(t)
	h := newHarness(t, store, app.WithLearnOffer(true), app.WithConfigPath(filepath.Join(t.TempDir(), "config.json")),
		fakeTutorial(new(*exec.Cmd), nil, nil))
	if v := h.view(); !strings.Contains(v, "Soy nuevo") {
		t.Fatalf("setup: the dialog should be open:\n%s", v)
	}
	h.send(tea.WindowSizeMsg{Width: 50, Height: 10})
	if v := h.screen(); strings.Contains(v, "Soy nuevo") {
		t.Fatalf("the dialog stayed open on a terminal it does not fit:\n%s", v)
	}
	h.send(tea.WindowSizeMsg{Width: 140, Height: 40})
	if v := h.screen(); !strings.Contains(v, "Soy nuevo") {
		t.Fatalf("the dialog should come back, still unanswered:\n%s", v)
	}
}

// S2: a failed write is reported in the bar and the answer still counts for
// this run: the tutorial starts, and the dialog asks again next time.
func TestFirstRunDialogReportsAFailedWrite(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearnOffer(true),
		app.WithConfigPath(filepath.Join(blocker, "config.json")), fakeTutorial(&got, nil, nil))
	h.key(tea.KeyEnter)
	if s := app.StatusText(h.m); !strings.Contains(s, "Could not remember") {
		t.Fatalf("the failed write should be reported, status %q", s)
	}
	if got == nil {
		t.Fatal("a failed write should not stop the tutorial")
	}
}

// S8: the dialog belongs to the home screen: it never opens over a note.
func TestFirstRunDialogStaysOffTheNoteScreen(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Create("nota", ""); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithLearnOffer(true), app.WithOpenNote("nota.md"),
		app.WithConfigPath(filepath.Join(t.TempDir(), "config.json")))
	if v := h.view(); strings.Contains(v, "Soy nuevo") {
		t.Fatalf("the dialog opened over the note:\n%s", v)
	}
}

// learnOffered reports whether the config at path remembers the answer.
func learnOffered(t *testing.T, path string) bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), `"learn_offered": true`)
}
