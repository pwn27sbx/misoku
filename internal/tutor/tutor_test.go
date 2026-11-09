package tutor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
)

func none(string) bool { return false }

func TestInstall(t *testing.T) {
	fsys := fstest.MapFS{
		"B01 Títulos.md":  {Data: []byte("original")},
		"B02 Casillas.md": {Data: []byte("dos")},
	}
	dir := filepath.Join(t.TempDir(), "tutor")
	if err := Install(fsys, dir, false); err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if got := read("B01 Títulos.md"); got != "original" {
		t.Fatalf("installed %q", got)
	}
	if got := read("B02 Casillas.md"); got != "dos" {
		t.Fatalf("installed %q", got)
	}

	path := filepath.Join(dir, "B01 Títulos.md")
	if err := os.WriteFile(path, []byte("editado"), 0o644); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(dir, "Mi nota.md")
	if err := os.WriteFile(extra, []byte("mía"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "B02 Casillas.md")); err != nil {
		t.Fatal(err)
	}
	if err := Install(fsys, dir, false); err != nil {
		t.Fatal(err)
	}
	if got := read("B01 Títulos.md"); got != "editado" {
		t.Fatalf("second Install overwrote an edited lesson: %q", got)
	}
	if got := read("B02 Casillas.md"); got != "dos" {
		t.Fatalf("second Install did not restore a missing lesson: %q", got)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	if err := Install(fsys, dir, true); err != nil {
		t.Fatal(err)
	}
	if got := read("B01 Títulos.md"); got != "original" {
		t.Fatalf("reset did not restore the lesson: %q", got)
	}
	if _, err := os.Stat(extra); !os.IsNotExist(err) {
		t.Fatalf("reset kept a user note: %v", err)
	}
	trash := map[string]string{
		"darwin": filepath.Join(home, ".Trash"),
		"linux":  filepath.Join(home, ".local", "share", "Trash", "files"),
	}[runtime.GOOS]
	if trash != "" {
		for name, want := range map[string]string{"Mi nota.md": "mía", "B01 Títulos.md": "editado"} {
			b, err := os.ReadFile(filepath.Join(trash, "tutor", name))
			if err != nil || string(b) != want {
				t.Errorf("reset did not move %s to the Trash: %q, %v", name, b, err)
			}
		}
	}
}

func TestInstallRefusesEmptyDir(t *testing.T) {
	if err := Install(fstest.MapFS{}, "", true); err == nil {
		t.Fatal("Install with an empty dir must fail")
	}
}

func TestDir(t *testing.T) {
	d := Dir()
	if d == "" {
		t.Skip("no user config dir")
	}
	if filepath.Base(d) != "tutor" || filepath.Base(filepath.Dir(d)) != "misoku" {
		t.Fatalf("Dir() = %q", d)
	}
}

func TestProgress(t *testing.T) {
	dir := t.TempDir()
	if err := Install(Bundled(), dir, false); err != nil {
		t.Fatal(err)
	}
	passed(t, dir, "B01 Títulos.md", "B07 Explorador.md")
	p := Scan(dir)
	if done, total := p.Count(); done != 2 || total != 7 {
		t.Fatalf("Scan Count = %d/%d, want 2/7", done, total)
	}

	if p.Update("B02 Casillas.md", readNote(t, dir, "B02 Casillas.md"), none) {
		t.Fatal("Update passed an unticked box")
	}
	if !p.Update("B02 Casillas.md", answers["B02 Casillas.md"].text, none) {
		t.Fatal("first pass not reported")
	}
	if done, _ := p.Count(); done != 3 {
		t.Fatalf("Count after pass = %d, want 3", done)
	}
	if p.Update("B02 Casillas.md", answers["B02 Casillas.md"].text, none) {
		t.Fatal("second pass reported again")
	}
	if p.Update("B02 Casillas.md", "- [ ] deshecho\n", none) {
		t.Fatal("un-passing reported as a pass")
	}
	// E4: a passed step is sticky, so the lesson stays passed.
	if done, _ := p.Count(); done != 3 {
		t.Fatalf("Count after un-pass = %d, want 3 (sticky)", done)
	}
	if p.Update("B01 Títulos.md", answers["B01 Títulos.md"].text, none) {
		t.Fatal("a lesson passed on disk was reported again")
	}
	if p.Update("Mi nota.md", "- [x] x\n", none) {
		t.Fatal("a non-lesson note reported a pass")
	}
	if p.Update("B00 Básico.md", "- [x] x\n", none) {
		t.Fatal("the index reported a pass")
	}
}

// readNote reads an installed note.
func readNote(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestLevel(t *testing.T) {
	p := Scan(t.TempDir())
	cases := map[string]int{
		"B00 Básico.md":     0,
		"B01 Títulos.md":    1,
		"B02 Casillas.md":   2,
		"B07 Explorador.md": 7,
		"B07 Explorador":    7,
		"Mi nota.md":        0,
	}
	for file, want := range cases {
		if got := p.Level(file); got != want {
			t.Errorf("Level(%q) = %d, want %d", file, got, want)
		}
	}
}

func TestNilProgress(t *testing.T) {
	var p *Progress
	if p.Update("B02 Casillas.md", "- [x] a\n", none) {
		t.Fatal("nil Update reported a pass")
	}
	if done, total := p.Count(); done != 0 || total != 0 {
		t.Fatalf("nil Count = %d/%d", done, total)
	}
	if got := p.Level("B01 Títulos.md"); got != 0 {
		t.Fatalf("nil Level = %d", got)
	}
}

// E5: the "- []" hint travels with B02's step, not with a file name.
func TestHint(t *testing.T) {
	const want = "- [] no es una tarea: falta el espacio → - [ ]"
	cases := []struct {
		file, text, want string
	}{
		{"B02 Casillas.md", "", want},
		{"B02 Casillas.md", "- [ ] comprar pan\n", want},
		{"B02 Casillas.md", "- [x] comprar pan\n", ""},
		{"B01 Títulos.md", "", ""},
		{"Mi nota.md", "- [] comprar pan\n", ""},
	}
	for _, c := range cases {
		if got := Hint(c.file, c.text); got != c.want {
			t.Errorf("Hint(%q, %q) = %q, want %q", c.file, c.text, got, c.want)
		}
	}
}
