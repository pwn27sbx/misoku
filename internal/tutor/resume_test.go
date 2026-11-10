package tutor

import (
	"testing"
)

// D6/S3: an explicit route nobody started opens its index; the last route
// played or the first route resumes at its first unfinished step. Both come
// back as vault-relative paths inside the route's folder.
func TestResumeNote(t *testing.T) {
	b := textLesson("B01", contains("b"))
	b.File = "B01 B.md"
	b.dir = "1 Básico"
	i := textLesson("I01", contains("i"))
	i.File = "I01 I.md"
	i.dir = "2 Intermedio"
	cat := Catalog{Routes: []Route{
		{ID: "B", Name: "Básico", Dir: "1 Básico", Index: "B00 Básico.md", Lessons: []Lesson{b}},
		{ID: "I", Name: "Intermedio", Dir: "2 Intermedio", Index: "I00 Intermedio.md", Lessons: []Lesson{i}},
	}}
	s := &Store{}
	if got := ResumeNote(cat, s, "I"); got != "2 Intermedio/I00 Intermedio.md" {
		t.Fatalf("an explicit fresh route opens its index, got %q", got)
	}
	if got := ResumeNote(cat, s, ""); got != "1 Básico/B01 B.md" {
		t.Fatalf("without progress the first lesson opens, got %q", got)
	}
	s.Mark(b, 1)
	if got := ResumeNote(cat, s, "B"); got != "1 Básico/B00 Básico.md" {
		t.Fatalf("a finished route opens its index, got %q", got)
	}
	s.SetLastRoute("I")
	s.Mark(i, 1)
	// A new unfinished lesson in the last route resumes there.
	i2 := textLesson("I02", contains("i2"))
	i2.File = "I02 I.md"
	i2.dir = "2 Intermedio"
	cat.Routes[1].Lessons = append(cat.Routes[1].Lessons, i2)
	if got := ResumeNote(cat, s, ""); got != "2 Intermedio/I02 I.md" {
		t.Fatalf("the last route resumed at %q, want 2 Intermedio/I02 I.md", got)
	}
}

func TestCatalogHelpers(t *testing.T) {
	need := textLesson("A07", Check{Kind: KindAction, Event: "history_open"})
	if !need.NeedsGit() {
		t.Fatal("a history event should need git")
	}
	if textLesson("B01", contains("x")).NeedsGit() {
		t.Fatal("a text step does not need git")
	}
	cfgj := textLesson("E01",
		Check{Kind: KindDisk, Pred: "config-key", Args: []string{"ui.link_suggestions"}},
		Check{Kind: KindAll, Checks: []Check{{Kind: KindDisk, Pred: "config-changed", Args: []string{"ui.theme"}}}},
	)
	keys := cfgj.ConfigKeys()
	if len(keys) != 2 || keys[0] != "ui.link_suggestions" || keys[1] != "ui.theme" {
		t.Fatalf("ConfigKeys = %v", keys)
	}
}
