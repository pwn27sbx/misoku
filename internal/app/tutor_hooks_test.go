package app_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// V1: a step the quit itself passes is saved before the program ends, so the
// next run reads it from the progress file.
func TestTutorSavesTheQuitStepBeforeQuitting(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B25", Title: "Salir", File: "B25 Salir.md",
			Steps: []tutor.Step{{
				Text:  "Vuelve a abrir y sal con :q!",
				Check: tutor.Check{Kind: tutor.KindAction, Event: "quit", Value: tutor.Values{"discard"}},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "B25 Salir.md", "B25 Salir", true)
	l, _ := cat.Lesson("B25 Salir.md")
	h.key(tea.KeyEsc) // an update makes B25 the current lesson
	h.keys("i")
	h.keys("cambio") // something for :q! to discard
	h.key(tea.KeyEsc)
	h.keys(":")
	h.keys("q!")
	h.key(tea.KeyEnter)
	if !h.quit {
		t.Fatal(":q! should quit")
	}
	if !progress.Passed(l, 1) {
		t.Fatal("the quit step did not pass in memory")
	}
	loaded, err := tutor.Load(h.store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Passed(l, 1) {
		t.Fatal("the quit step did not reach the progress file")
	}
}

// V1: quitting with unsaved changes offers the dialog (the "confirm" value)
// and cancelling it keeps the program open.
func TestTutorQuitDialogIsTheConfirmStep(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B25", Title: "Salir", File: "B25 Salir.md",
			Steps: []tutor.Step{{
				Text:  "Con cambios, pulsa :q y mira la pregunta",
				Check: tutor.Check{Kind: tutor.KindAction, Event: "quit", Value: tutor.Values{"confirm"}},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "B25 Salir.md", "B25 Salir", true)
	l, _ := cat.Lesson("B25 Salir.md")
	h.key(tea.KeyEsc)
	h.keys("i")
	h.keys("cambio")
	h.key(tea.KeyEsc)
	h.keys(":")
	h.keys("q")
	h.key(tea.KeyEnter)
	if h.quit {
		t.Fatal("a dirty quit must ask first")
	}
	if !progress.Passed(l, 1) {
		t.Fatal("showing the quit dialog did not credit the confirm step")
	}
	h.key(tea.KeyEsc) // Cancel
	h.view()
}

// V1: the command line is its own mode, so ":" reports mode_changed=command
// instead of the "normal" it used to report.
func TestTutorCommandLineIsTheCommandMode(t *testing.T) {
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B04", Title: "Línea", File: "B04 Línea.md",
			Steps: []tutor.Step{{
				Text:  "Pulsa dos puntos y abandónala con Esc",
				Check: tutor.Check{Kind: tutor.KindAction, Event: "mode_changed", Value: tutor.Values{"command"}},
			}},
		}},
	}}}
	h, progress := fixtureHarness(t, cat, "B04 Línea.md", "B04 Línea", true)
	l, _ := cat.Lesson("B04 Línea.md")
	h.key(tea.KeyEsc) // an update makes B04 the current lesson
	h.keys(":")
	if !progress.Passed(l, 1) {
		t.Fatal(": did not report the command mode")
	}
}
