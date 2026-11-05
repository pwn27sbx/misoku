package tutor_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

// answers is a correct reply for each lesson's "Tu turno", appended to the
// shipped template.
var answers = map[string]struct {
	text   string
	exists func(string) bool
}{
	"01 Títulos.md":              {"\n## Viaje\n\n### Día 1\n", none},
	"02 Casillas.md":             {"\n- [x] Aprender casillas\n", none},
	"03 Menú de bloques.md":      {"\n| Fruta | Precio |\n| --- | --- |\n| Mango | 2 |\n", none},
	"04 Links.md":                {"\n[[05 Links a títulos]]\n", none},
	"05 Links a títulos.md":      {"\n[[#Tu turno]]\n", none},
	"06 Referencias a bloque.md": {"\nMi frase favorita ^favorita\n\n[[#^favorita]]\n", none},
	"07 Explorador.md":           {"\n[[Mi nota]]\n", func(n string) bool { return n == "Mi nota" }},
}

func TestLessons(t *testing.T) {
	if len(tutor.Lessons) != 8 {
		t.Fatalf("len(Lessons) = %d, want 8", len(tutor.Lessons))
	}
	bundled := tutor.Bundled()
	for _, l := range tutor.Lessons {
		data, err := fs.ReadFile(bundled, l.File)
		if err != nil {
			t.Errorf("%s not embedded: %v", l.File, err)
			continue
		}
		text := string(data)
		if !strings.Contains(text, "## ") && l.Check != "" {
			t.Errorf("%s has no sections", l.File)
		}
		if tutor.Passed(l.Check, text, func(string) bool { return false }) {
			t.Errorf("%s starts passed", l.File)
		}
		if l.Check == "" {
			continue
		}
		a, ok := answers[l.File]
		if !ok {
			t.Errorf("no fixture answer for %s", l.File)
			continue
		}
		if !tutor.Passed(l.Check, text+a.text, a.exists) {
			t.Errorf("%s does not pass with its answer", l.File)
		}
		if !strings.Contains(text, "## Tu turno") {
			t.Errorf("%s has no ## Tu turno", l.File)
		}
	}
}

func TestLessonsQuoteSpec(t *testing.T) {
	bundled := tutor.Bundled()
	musts := map[string][]string{
		"01 Títulos.md":  {"`##` → `###` para meter secciones dentro de otra (no hay numeración 2.1 automática)"},
		"02 Casillas.md": {"`- [ ]` lleva espacio entre los corchetes; `- []` no es una tarea"},
		"06 Referencias a bloque.md": {
			"marcar una línea con `^nombre` al final y enlazarla con `[[#^nombre]]` (misma nota), `[[Nota#^nombre]]` (otra nota) o `[[#^nombre|texto]]`. Seguir el link con `gd`.",
		},
		"05 Links a títulos.md": {"`[[#Título]]`"},
	}
	for file, ss := range musts {
		data, err := fs.ReadFile(bundled, file)
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

func TestIndexLinksEveryLevel(t *testing.T) {
	data, err := fs.ReadFile(tutor.Bundled(), "00 Empieza aquí.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range tutor.Lessons[1:] {
		link := "[[" + strings.TrimSuffix(l.File, ".md") + "]]"
		if !strings.Contains(string(data), link) {
			t.Errorf("index does not link %s", link)
		}
	}
}
