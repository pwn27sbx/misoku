package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/tutor"
)

// S5/G5: the app's disk Env and tutor.DirEnv agree. Both look a disk path up
// inside the lesson's route folder first and at the vault root second, so the
// same teaching action passes in the app and in the scan before it.
func TestTutorDiskEnvAgreesWithDirEnv(t *testing.T) {
	root := t.TempDir()
	write := func(rel, text string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("1 Básico/compras.md", "comprado\n")
	write("1 Básico/Plan.md", "[[Viaje#Equipaje]]\n")
	write("Plantillas/diario.md", "Plan del día\n")
	write("Viaje.md", "nota de otra ruta\n")

	store, err := core.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	cat := tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Dir: "1 Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B01", Title: "Fixture", File: "B01 Fixture.md",
			Steps: []tutor.Step{{Text: "Paso", Check: tutor.Check{Kind: tutor.KindNone}}},
		}},
	}}}
	m := New(store, WithTutor(&tutor.Store{}, cat, root, true))

	appEnv := m.tutorDiskEnv(cat.Routes[0].Lessons[0])
	appEnv.LessonDir = "1 Básico"
	dirEnv := tutor.DirEnv(root)
	dirEnv.LessonDir = "1 Básico"

	cases := []struct {
		name  string
		check tutor.Check
		want  bool
	}{
		{"route folder note", tutor.Check{Kind: tutor.KindDisk, Pred: "exists", Args: []string{"compras"}}, true},
		{"missing note", tutor.Check{Kind: tutor.KindDisk, Pred: "exists", Args: []string{"mercado"}}, false},
		{"route folder folder", tutor.Check{Kind: tutor.KindDisk, Pred: "folder-exists", Args: []string{"1 Básico"}}, true},
		{"root folder", tutor.Check{Kind: tutor.KindDisk, Pred: "folder-exists", Args: []string{"Plantillas"}}, true},
		{"missing folder", tutor.Check{Kind: tutor.KindDisk, Pred: "folder-exists", Args: []string{"Notas"}}, false},
		{"another note in the route folder", tutor.Check{Kind: tutor.KindDisk, Pred: "other-note-text", Args: []string{"Plan", "[[Viaje#Equipaje]]"}}, true},
		{"another note at the root", tutor.Check{Kind: tutor.KindDisk, Pred: "other-note-text", Args: []string{"Viaje", "otra ruta"}}, true},
		{"another note's missing text", tutor.Check{Kind: tutor.KindDisk, Pred: "other-note-text", Args: []string{"Plan", "no está"}}, false},
		{"no-file for a gone note", tutor.Check{Kind: tutor.KindDisk, Pred: "no-file", Args: []string{"mercado"}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.check.Pass("", appEnv); got != c.want {
				t.Errorf("app Env = %v, want %v", got, c.want)
			}
			if got := c.check.Pass("", dirEnv); got != c.want {
				t.Errorf("DirEnv = %v, want %v", got, c.want)
			}
		})
	}
}
