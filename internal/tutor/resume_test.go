package tutor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

// passed writes the answer of every level named in files into dir, as the
// player's notes would hold it. Level 7 needs the note it links to exist.
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

// S5: the tutorial opens its index at the start and once every level passes,
// and the first unpassed lesson in between.
func TestResume(t *testing.T) {
	t.Parallel()
	if got := tutor.Resume(nil); got != "00 Empieza aquí.md" {
		t.Errorf("without progress the index should open, got %q", got)
	}

	dir := t.TempDir()
	if err := tutor.Install(tutor.Bundled(), dir, false); err != nil {
		t.Fatal(err)
	}
	if got := tutor.Resume(tutor.Scan(dir)); got != "00 Empieza aquí.md" {
		t.Errorf("at 0/7 the index should open, got %q", got)
	}

	passed(t, dir, "01 Títulos.md", "02 Casillas.md")
	if got := tutor.Resume(tutor.Scan(dir)); got != "03 Menú de bloques.md" {
		t.Errorf("after levels 1 and 2 the next unpassed lesson should open, got %q", got)
	}

	// A gap in the middle is resumed at the gap, not after it.
	passed(t, dir, "04 Links.md", "05 Links a títulos.md", "06 Referencias a bloque.md", "07 Explorador.md")
	if got := tutor.Resume(tutor.Scan(dir)); got != "03 Menú de bloques.md" {
		t.Errorf("the first unpassed lesson should open, got %q", got)
	}

	passed(t, dir, "03 Menú de bloques.md")
	if got := tutor.Resume(tutor.Scan(dir)); got != "00 Empieza aquí.md" {
		t.Errorf("at 7/7 the index should open, got %q", got)
	}
}
