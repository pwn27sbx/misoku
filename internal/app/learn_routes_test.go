package app_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// fixtureRoutes is a catalog with two routes, for the home screen tests.
func fixtureRoutes() (tutor.Catalog, *tutor.Store) {
	b1 := tutor.Lesson{ID: "B01", Title: "Uno", File: "B01 Uno.md", Steps: []tutor.Step{{Text: "Paso", Check: tutor.Check{Kind: tutor.KindNone}}}}
	b2 := tutor.Lesson{ID: "B02", Title: "Dos", File: "B02 Dos.md", Steps: []tutor.Step{{Text: "Paso", Check: tutor.Check{Kind: tutor.KindNone}}}}
	i1 := tutor.Lesson{ID: "I01", Title: "Tres", File: "I01 Tres.md", Steps: []tutor.Step{{Text: "Paso", Check: tutor.Check{Kind: tutor.KindNone}}}}
	cat := tutor.Catalog{Routes: []tutor.Route{
		{ID: "B", Name: "Básico", Index: "B00 Básico.md", Lessons: []tutor.Lesson{b1, b2}},
		{ID: "I", Name: "Intermedio", Index: "I00 Intermedio.md", Lessons: []tutor.Lesson{i1}},
	}}
	return cat, &tutor.Store{}
}

// D6: the home screen lists every route with lessons, its progress and a
// check when the route is complete.
func TestLearnEntriesPerRoute(t *testing.T) {
	cat, progress := fixtureRoutes()
	progress.Mark(cat.Routes[0].Lessons[0], 1)
	h := newHarness(t, newTestStore(t), app.WithLearn(progress, cat))
	v := h.view()
	if !strings.Contains(v, "Learn · Básico 1/2") || !strings.Contains(v, "Learn · Intermedio 0/1") {
		t.Fatalf("the home screen should list both routes:\n%s", v)
	}
}

// D6/E10: l opens the first route not complete; an entry carries its route.
func TestLearnKeyOpensTheFirstIncompleteRoute(t *testing.T) {
	cat, progress := fixtureRoutes()
	progress.Mark(cat.Routes[0].Lessons[0], 1)
	progress.Mark(cat.Routes[0].Lessons[1], 1)
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearn(progress, cat), fakeTutorial(&got, nil, nil))
	h.keys("l")
	if got == nil {
		t.Fatal("l did not start the tutorial")
	}
	if got.Args[2] != "--route" || got.Args[3] != "I" {
		t.Fatalf("l ran %q, want --route I", got.Args)
	}
	// With every route complete, it goes back to the first.
	progress.Mark(cat.Routes[1].Lessons[0], 1)
	h.keys("l")
	if got.Args[3] != "B" {
		t.Fatalf("all complete: l ran %q, want --route B", got.Args)
	}
}

// E10: each home entry starts its own route.
func TestLearnEntryStartsItsRoute(t *testing.T) {
	cat, progress := fixtureRoutes()
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearn(progress, cat), fakeTutorial(&got, nil, nil))
	for i := 0; i < 7; i++ { // New Note … Graph, Learn Básico, Learn Intermedio
		h.keys("j")
	}
	h.key(tea.KeyEnter)
	if got == nil {
		t.Fatal("Enter on the Intermedio entry did not start the tutorial")
	}
	if got.Args[2] != "--route" || got.Args[3] != "I" {
		t.Fatalf("the entry ran %q, want --route I", got.Args)
	}
}

// D6/E10: Ya tengo experiencia opens Intermedio, and falls back to Básico
// while Intermedio has no lessons.
func TestExperienceOffersIntermedioOrBasico(t *testing.T) {
	cat, progress := fixtureRoutes()
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearn(progress, cat), app.WithLearnOffer(true),
		app.WithConfigPath(t.TempDir()+"/config.json"), fakeTutorial(&got, nil, nil))
	h.keys("j")
	h.key(tea.KeyEnter)
	if got == nil || got.Args[3] != "I" {
		t.Fatalf("Ya tengo experiencia ran %v, want --route I", got)
	}

	bOnly := tutor.Catalog{Routes: []tutor.Route{cat.Routes[0]}}
	var fallback *exec.Cmd
	h2 := newHarness(t, newTestStore(t), app.WithLearn(progress, bOnly), app.WithLearnOffer(true),
		app.WithConfigPath(t.TempDir()+"/config2.json"), fakeTutorial(&fallback, nil, nil))
	h2.keys("j")
	h2.key(tea.KeyEnter)
	if fallback == nil || fallback.Args[3] != "B" {
		t.Fatalf("with no Intermedio, Ya tengo experiencia ran %v, want --route B", fallback)
	}
}

// D5: finishing a route says so on the bar.
func TestTutorRouteCompleteLine(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B01", Title: "Única", File: "B01 Única.md",
			Steps: []tutor.Step{{Text: "Paso", Check: tutor.Check{Kind: tutor.KindAction, Event: "next_tab"}}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "B01 Única.md", "B01 Única", true)
	l, _ := cat.Lesson("B01 Única.md")
	progress.Mark(l, 1)
	h.key(tea.KeyEsc) // an update; the current lesson was set when the note opened
	if s := statusOf(h.view()); !strings.Contains(s, "🏆 Ruta Básico completa") {
		t.Fatalf("the bar should say the route is complete:\n%s", s)
	}
}

// E10: "Ahora no" goes to the notes even when the home has Learn entries.
func TestLearnNowDoesNotStartTheTutorial(t *testing.T) {
	cat, progress := fixtureRoutes()
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearn(progress, cat), app.WithLearnOffer(true),
		app.WithConfigPath(t.TempDir()+"/config.json"), fakeTutorial(&got, nil, nil))
	h.keys("jj")
	h.key(tea.KeyEnter)
	if got != nil {
		t.Fatalf("Ahora no started %v", got.Args)
	}
}
