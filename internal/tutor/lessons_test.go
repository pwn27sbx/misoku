package tutor

import (
	"io/fs"
	"strings"
	"testing"
)

// answers is a correct reply for each lesson's "Tu turno", appended to the
// shipped lesson, with the environment that check needs.
var answers = map[string]struct {
	text string
	env  Env
}{
	"B01 Títulos.md":              {"\n## Viaje\n\n### Día 1\n", Env{}},
	"B02 Casillas.md":             {"\n- [x] Aprender casillas\n", Env{}},
	"B03 Menú de bloques.md":      {"\n| Fruta | Precio |\n| --- | --- |\n| Mango | 2 |\n", Env{}},
	"B04 Links.md":                {"\n[[B05 Links a títulos]]\n", Env{Exists: func(string) bool { return true }}},
	"B05 Links a títulos.md":      {"\n[[#Un título de ejemplo]]\n", Env{}},
	"B06 Referencias a bloque.md": {"\nMi frase favorita ^favorita\n\n[[#^favorita]]\n", Env{}},
	"B07 Explorador.md":           {"\n[[Mi nota]]\n", Env{Exists: func(n string) bool { return n == "Mi nota" }}},
}

// S1/S7: every lesson of the catalog is embedded, starts unpassed, and its
// answer passes every step.
func TestLessons(t *testing.T) {
	cat := Default()
	if len(cat.Routes) == 0 {
		t.Fatal("the catalog is empty")
	}
	bundled := Bundled()
	for _, r := range cat.Routes {
		if _, err := fs.ReadFile(bundled, r.Index); err != nil {
			t.Errorf("route %s index %s not embedded: %v", r.ID, r.Index, err)
		}
		for _, l := range r.Lessons {
			data, err := fs.ReadFile(bundled, l.File)
			if err != nil {
				t.Errorf("%s not embedded: %v", l.File, err)
				continue
			}
			text := string(data)
			if !strings.Contains(text, "## ") {
				t.Errorf("%s has no sections", l.File)
			}
			// L4: the untouched lesson never passes, not even on its own text.
			untouched := &Store{}
			untouched.Recheck(l, text, Env{})
			if got := untouched.NextStep(l); got != 1 {
				t.Errorf("%s starts passed (next step %d)", l.File, got)
			}
			a, ok := answers[l.File]
			if !ok {
				t.Errorf("no fixture answer for %s", l.File)
				continue
			}
			played := &Store{}
			played.Recheck(l, text+a.text, a.env)
			if !played.LessonDone(l) {
				t.Errorf("%s does not pass with its answer (next step %d)", l.File, played.NextStep(l))
			}
		}
	}
}

func TestLessonsQuoteSpec(t *testing.T) {
	musts := map[string][]string{
		"B01 Títulos.md":  {"`##` → `###` para meter secciones dentro de otra (no hay numeración 2.1 automática)"},
		"B02 Casillas.md": {"`- [ ]` lleva espacio entre los corchetes; `- []` no es una tarea"},
		"B06 Referencias a bloque.md": {
			"marcar una línea con `^nombre` al final y enlazarla con `[[#^nombre]]` (misma nota), `[[Nota#^nombre]]` (otra nota) o `[[#^nombre|texto]]`. Seguir el link con `gd`.",
		},
		"B05 Links a títulos.md": {"`[[#Título]]`"},
	}
	for file, ss := range musts {
		data, err := fs.ReadFile(Bundled(), file)
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		for _, s := range ss {
			if !strings.Contains(string(data), s) {
				t.Errorf("%s does not quote %q", file, s)
			}
		}
	}
}

// The index links every lesson of its route (and the final reference note from
// the last one).
func TestIndexLinksEveryLesson(t *testing.T) {
	cat := Default()
	for _, r := range cat.Routes {
		data, err := fs.ReadFile(Bundled(), r.Index)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range r.Lessons {
			link := "[[" + strings.TrimSuffix(l.File, ".md") + "]]"
			if !strings.Contains(string(data), link) {
				t.Errorf("%s does not link %s", r.Index, link)
			}
		}
	}
}

// E14: the tutorial ships one small original image, embedded with the notes.
func TestBundledImage(t *testing.T) {
	info, err := fs.Stat(Bundled(), "mapa.png")
	if err != nil {
		t.Fatalf("mapa.png is not embedded: %v", err)
	}
	if info.Size() > 50*1024 {
		t.Fatalf("mapa.png is %d bytes, want under 50 KB", info.Size())
	}
}

func TestLastLessonLinksTheReferenceNote(t *testing.T) {
	cat := Default()
	for _, r := range cat.Routes {
		if len(r.Lessons) == 0 {
			continue
		}
		last := r.Lessons[len(r.Lessons)-1]
		data, err := fs.ReadFile(Bundled(), last.File)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "[["+strings.TrimSuffix(ReferenceNote, ".md")+"]]") {
			t.Errorf("%s does not link %s", last.File, ReferenceNote)
		}
	}
}
