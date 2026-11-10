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
func tutorHarness(t *testing.T, lesson string) (*harness, *tutor.Store, tutor.Catalog) {
	t.Helper()
	store := newTestStore(t)
	if err := tutor.Install(tutor.Bundled(), store.Root(), false); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	cat := tutor.Default()
	progress, err := tutor.Load(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	progress.ScanDir(cat, store.Root(), tutor.DirEnv(store.Root()))
	h := newHarness(t, store, app.WithTutor(progress, cat, store.Root(), true))
	m, cmd := app.OpenInCurrentBuffer(h.m, lesson)
	h.m = m
	h.send(cmdMsg(cmd)...)
	h.keys("G")
	h.keys("o")
	return h, progress, cat
}

// fixtureHarness opens note in a test vault with the given catalog and
// progress, played with or without git.
func fixtureHarness(t *testing.T, cat tutor.Catalog, lesson, note string, hasGit bool) (*harness, *tutor.Store) {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.Create(note, ""); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	progress := &tutor.Store{}
	h := newHarness(t, store, app.WithTutor(progress, cat, store.Root(), hasGit))
	m, cmd := app.OpenInCurrentBuffer(h.m, lesson)
	h.m = m
	h.send(cmdMsg(cmd)...)
	return h, progress
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

// S7/G9: starting on a lesson expands its route's folder in the explorer, so
// the lesson's row, and its siblings, are already in view.
func TestTutorExpandsTheRouteFolderAtStart(t *testing.T) {
	h, _, _ := tutorHarness(t, "1 Básico/B13 Títulos.md")
	v := h.view()
	if !strings.Contains(v, "1 Básico") {
		t.Fatalf("the route folder is not in the explorer:\n%s", v)
	}
	if !strings.Contains(v, "B26 Desafío Básico.md") {
		t.Fatalf("the route folder is not expanded:\n%s", v)
	}
}

// D5: a passed step is written to its note and its progress at once, so a
// quit that discards everything else keeps both.
func TestTutorSavesAPassedLevelAtOnce(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B01", Title: "Uno", File: "B01 Uno.md",
			Steps: []tutor.Step{{
				Text:  "Escribe la frase",
				Check: tutor.Check{Kind: tutor.KindText, Pred: "line-contains", Args: []string{"Hoy empiezo mi diario de viaje"}},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "B01 Uno.md", "B01 Uno", true)
	h.key(tea.KeyEsc) // an update makes B01 the current lesson
	h.keys("i")
	h.keys("Hoy empiezo mi diario de viaje")
	l, _ := cat.Lesson("B01 Uno.md")
	if !progress.Passed(l, 1) {
		t.Fatal("the progress store did not record the step")
	}
	root := h.store.Root()
	lesson := filepath.Join(root, "B01 Uno.md")
	if data, err := os.ReadFile(lesson); err != nil || !strings.Contains(string(data), "Hoy empiezo mi diario de viaje") {
		t.Fatalf("the passed lesson was not saved at once (%v):\n%s", err, data)
	}
	loaded, err := tutor.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Passed(l, 1) {
		t.Fatal("the progress file did not keep the step")
	}

	// An edit after the lesson is nobody's business but the discard's.
	h.keys(" y algo más")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	if h.quit {
		t.Fatal("unsaved changes must ask before quitting")
	}
	h.keys("d") // Discard
	if !h.quit {
		t.Fatal("Discard should quit")
	}
	loaded, err = tutor.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Passed(l, 1) {
		t.Fatal("Discard lost the progress")
	}
	if data, _ := os.ReadFile(lesson); strings.Contains(string(data), "y algo más") {
		t.Fatalf("Discard kept the later edit:\n%s", data)
	}
}

// D5: the step line is persistent, the success line replaces it, and the
// badge counts the route.
func TestTutorShowsTheStepAndThenTheSuccess(t *testing.T) {
	h, _, _ := tutorHarness(t, "1 Básico/B13 Títulos.md")
	if s := statusOf(h.view()); !strings.Contains(s, "B13 · Paso 1/4") {
		t.Fatalf("the statusline should show the step in play:\n%s", s)
	}
	for _, line := range []string{"# Viaje", "## Equipaje", "### Día 1", "#### Detalle"} {
		h.keys(line)
		h.key(tea.KeyEnter)
	}
	h.m = app.ExpireToast(app.ExpireTutorFlash(h.m))
	h.send(tea.KeyMsg{Type: tea.KeyEsc}) // clears the save message
	v := h.view()
	if !strings.Contains(statusOf(v), "✓ B13 superado · siguiente: Listas") {
		t.Fatalf("the success line should persist after a toast:\n%s", v)
	}
	if !strings.Contains(statusOf(v), "Tutor 1/26") {
		t.Fatalf("the badge should count the route:\n%s", v)
	}
}

// E5: the hint of the current step shows once while the lesson is
// current.
func TestTutorHintsTheMissingSpace(t *testing.T) {
	h, _, _ := tutorHarness(t, "1 Básico/B15 Casillas, marcar.md")
	h.keys("x")
	hint := "queda - [x] y deja de estar pendiente"
	if v := h.view(); !strings.Contains(v, hint) {
		t.Fatalf("writing - [] should toast the hint:\n%s", v)
	}
	h.m = app.ExpireToast(h.m)
	h.keys("y")
	if v := h.view(); strings.Contains(v, hint) {
		t.Fatalf("the hint came back on the next edit:\n%s", v)
	}
}

// E14: without git, a lesson that waits for :history is skipped, not
// failed, and its route can complete.
func TestTutorSkipsTheHistoryLessonWithoutGit(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "A", Name: "Avanzado", Index: "A00 Avanzado.md",
		Lessons: []tutor.Lesson{{
			ID: "A07", Title: "Historia", File: "A07 Historia.md",
			Steps: []tutor.Step{{
				Text:  "Abre :history",
				Check: tutor.Check{Kind: tutor.KindAction, Event: "history_open"},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "A07 Historia.md", "A07 Historia", false)
	l, _ := cat.Lesson("A07 Historia.md")
	if !progress.Skipped(l, 1) || !progress.LessonDone(l) {
		t.Fatal("the history step should be skipped without git")
	}
	if s := statusOf(h.view()); !strings.Contains(s, "Necesitas git para esta lección") {
		t.Fatalf("the bar should say git is needed:\n%s", s)
	}
}

// E14: with git available, the skip is forgotten and the step is in play.
func TestTutorUnskipsTheHistoryLessonWithGit(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "A", Name: "Avanzado", Index: "A00 Avanzado.md",
		Lessons: []tutor.Lesson{{
			ID: "A07", Title: "Historia", File: "A07 Historia.md",
			Steps: []tutor.Step{{
				Text:  "Abre :history",
				Check: tutor.Check{Kind: tutor.KindAction, Event: "history_open"},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "A07 Historia.md", "A07 Historia", true)
	l, _ := cat.Lesson("A07 Historia.md")
	if progress.Skipped(l, 1) {
		t.Fatal("git is available: the step must not be skipped")
	}
	if s := statusOf(h.view()); !strings.Contains(s, "A07 · Paso 1/1") {
		t.Fatalf("the bar should show the step:\n%s", s)
	}
}

// E9: leaving INSERT is a mode_changed event, credited to the current
// lesson.
func TestTutorModeChangedEvent(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B01", Title: "Modos", File: "B01 Modos.md",
			Steps: []tutor.Step{{
				Text:  "Entra en INSERT",
				Check: tutor.Check{Kind: tutor.KindAction, Event: "mode_changed", Value: tutor.Values{"insert"}},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "B01 Modos.md", "B01 Modos", true)
	l, _ := cat.Lesson("B01 Modos.md")
	h.keys("i")
	if !progress.Passed(l, 1) {
		t.Fatal("entering INSERT did not credit the current lesson")
	}
}

// E14: without git, only the steps that wait for :history are skipped, not
// the ordinary steps of the same lesson.
func TestTutorSkipsOnlyTheHistoryStep(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "A", Name: "Avanzado", Index: "A00 Avanzado.md",
		Lessons: []tutor.Lesson{{
			ID: "A07", Title: "Historia", File: "A07 Historia.md",
			Steps: []tutor.Step{
				{Text: "Abre :history", Check: tutor.Check{Kind: tutor.KindAction, Event: "history_open"}},
				{Text: "Escribe hola", Check: tutor.Check{Kind: tutor.KindText, Pred: "line-contains", Args: []string{"hola"}}},
			},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "A07 Historia.md", "A07 Historia", false)
	l, _ := cat.Lesson("A07 Historia.md")
	if !progress.Skipped(l, 1) {
		t.Fatal("the history step should be skipped without git")
	}
	if progress.Skipped(l, 2) || progress.Passed(l, 2) {
		t.Fatal("the ordinary step must not be skipped")
	}
	if got := progress.NextStep(l); got != 2 {
		t.Fatalf("NextStep = %d, want 2 (the text step)", got)
	}
	if s := statusOf(h.view()); !strings.Contains(s, "A07 · Paso 2/2") {
		t.Fatalf("the bar should show the ordinary step:\n%s", s)
	}
}

// E7: saving another note after the lesson is current re-runs its disk
// checks, so a note created and saved completes the exists step.
func TestTutorDiskCheckAfterSavingAnotherNote(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B07", Title: "Nota", File: "B07 Nota.md",
			Steps: []tutor.Step{{
				Text:  "Crea Mi nota",
				Check: tutor.Check{Kind: tutor.KindDisk, Pred: "exists", Args: []string{"Mi nota"}},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "B07 Nota.md", "B07 Nota", true)
	h.key(tea.KeyEsc) // an update makes B07 the current lesson
	// Create Mi nota through the app, as the player would, and save it.
	h.keys(" n")
	h.keys("hecha")
	h.key(tea.KeyEsc)
	h.keys(":w")
	h.key(tea.KeyEnter)
	for range "Untitled-1" {
		h.key(tea.KeyBackspace)
	}
	h.keys("Mi nota")
	h.key(tea.KeyEnter)
	l, _ := cat.Lesson("B07 Nota.md")
	if !progress.Passed(l, 1) {
		t.Fatal("saving another note did not run the lesson's disk check")
	}
}

// E2/E9: a tab switch credits the current lesson even when another tab is
// now active.
func TestTutorNextTabCreditsTheCurrentLesson(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B07", Title: "Pestañas", File: "B07 Pestañas.md",
			Steps: []tutor.Step{{
				Text:  "Cambia de pestaña",
				Check: tutor.Check{Kind: tutor.KindAction, Event: "next_tab"},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "B07 Pestañas.md", "B07 Pestañas", true)
	h.key(tea.KeyEsc) // an update makes B07 the current lesson
	h.keys(" n")      // a second tab, a draft; the lesson stays current
	h.key(tea.KeyEsc)
	l, _ := cat.Lesson("B07 Pestañas.md")
	h.keys("L")
	if !progress.Passed(l, 1) {
		t.Fatal("the tab switch did not credit the current lesson")
	}
}

// E7: the disk checks run on opening a lesson, off the update loop.
func TestTutorDiskCheckOnOpen(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B07", Title: "Nota", File: "B07 Nota.md",
			Steps: []tutor.Step{{
				Text:  "Crea Mi nota",
				Check: tutor.Check{Kind: tutor.KindDisk, Pred: "exists", Args: []string{"Mi nota"}},
			}},
		}},
	}}}
	store := newTestStore(t)
	if _, err := store.Create("B07 Nota", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create("Mi nota", ""); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	progress := &tutor.Store{}
	h := newHarness(t, store, app.WithTutor(progress, cat, store.Root(), true))
	m, cmd := app.OpenInCurrentBuffer(h.m, "B07 Nota.md")
	h.m = m
	h.send(cmdMsg(cmd)...)
	l, _ := cat.Lesson("B07 Nota.md")
	if !progress.Passed(l, 1) {
		t.Fatal("the disk check did not run on open")
	}
}

// S7: without WithTutor the statusline is the one Misoku drew before the
// tutorial existed.
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
	m, cmd := app.OpenInCurrentBuffer(h.m, "1 Básico/B01 Escribe en 30 segundos.md")
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
