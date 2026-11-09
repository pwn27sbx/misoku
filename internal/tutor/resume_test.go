package tutor

import (
	"os"
	"path/filepath"
	"testing"
)

// passed writes the answer of every lesson named in files into dir, as the
// player's notes would hold it. B07 needs the note it links to exist.
func passed(t *testing.T, dir string, files ...string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "Mi nota.md"), []byte("una nota\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		a, ok := answers[f]
		if !ok {
			t.Fatalf("no fixture answer for %s", f)
		}
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), append(data, a.text...), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// S4: the tutorial opens its index at the start and once every lesson passes,
// and the first unpassed lesson in between.
func TestResume(t *testing.T) {
	t.Parallel()
	if got := Resume(nil); got != "B00 Básico.md" {
		t.Errorf("without progress the index should open, got %q", got)
	}

	dir := t.TempDir()
	if err := Install(Bundled(), dir, false); err != nil {
		t.Fatal(err)
	}
	if got := Resume(Scan(dir)); got != "B00 Básico.md" {
		t.Errorf("at 0/7 the index should open, got %q", got)
	}

	passed(t, dir, "B01 Títulos.md", "B02 Casillas.md")
	if got := Resume(Scan(dir)); got != "B03 Menú de bloques.md" {
		t.Errorf("after lessons 1 and 2 the next unpassed lesson should open, got %q", got)
	}

	// A gap in the middle is resumed at the gap, not after it.
	passed(t, dir, "B04 Links.md", "B05 Links a títulos.md", "B06 Referencias a bloque.md", "B07 Explorador.md")
	if got := Resume(Scan(dir)); got != "B03 Menú de bloques.md" {
		t.Errorf("the first unpassed lesson should open, got %q", got)
	}

	passed(t, dir, "B03 Menú de bloques.md")
	if got := Resume(Scan(dir)); got != "B00 Básico.md" {
		t.Errorf("at 7/7 the index should open, got %q", got)
	}
}
