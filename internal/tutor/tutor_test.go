package tutor_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

func TestInstall(t *testing.T) {
	fsys := fstest.MapFS{
		"01 Títulos.md":  {Data: []byte("original")},
		"02 Casillas.md": {Data: []byte("dos")},
	}
	dir := filepath.Join(t.TempDir(), "tutor")
	if err := tutor.Install(fsys, dir, false); err != nil {
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
	if got := read("01 Títulos.md"); got != "original" {
		t.Fatalf("installed %q", got)
	}
	if got := read("02 Casillas.md"); got != "dos" {
		t.Fatalf("installed %q", got)
	}

	path := filepath.Join(dir, "01 Títulos.md")
	if err := os.WriteFile(path, []byte("editado"), 0o644); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(dir, "Mi nota.md")
	if err := os.WriteFile(extra, []byte("mía"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "02 Casillas.md")); err != nil {
		t.Fatal(err)
	}
	if err := tutor.Install(fsys, dir, false); err != nil {
		t.Fatal(err)
	}
	if got := read("01 Títulos.md"); got != "editado" {
		t.Fatalf("second Install overwrote an edited lesson: %q", got)
	}
	if got := read("02 Casillas.md"); got != "dos" {
		t.Fatalf("second Install did not restore a missing lesson: %q", got)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	if err := tutor.Install(fsys, dir, true); err != nil {
		t.Fatal(err)
	}
	if got := read("01 Títulos.md"); got != "original" {
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
		for name, want := range map[string]string{"Mi nota.md": "mía", "01 Títulos.md": "editado"} {
			b, err := os.ReadFile(filepath.Join(trash, "tutor", name))
			if err != nil || string(b) != want {
				t.Errorf("reset did not move %s to the Trash: %q, %v", name, b, err)
			}
		}
	}
}

func TestInstallRefusesEmptyDir(t *testing.T) {
	if err := tutor.Install(fstest.MapFS{}, "", true); err == nil {
		t.Fatal("Install with an empty dir must fail")
	}
}

func TestDir(t *testing.T) {
	d := tutor.Dir()
	if d == "" {
		t.Skip("no user config dir")
	}
	if filepath.Base(d) != "tutor" || filepath.Base(filepath.Dir(d)) != "misoku" {
		t.Fatalf("Dir() = %q", d)
	}
}

func writeVault(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestProgress(t *testing.T) {
	dir := writeVault(t, map[string]string{
		"01 Títulos.md":    "## A\n### B\n",
		"02 Casillas.md":   "- [ ] nada\n",
		"07 Explorador.md": "[[Mi nota]]\n",
		"Mi nota.md":       "hola\n",
	})
	p := tutor.Scan(dir)
	if done, total := p.Count(); done != 2 || total != 7 {
		t.Fatalf("Scan Count = %d/%d, want 2/7", done, total)
	}

	if p.Update("02 Casillas.md", "- [ ] nada\n", none) {
		t.Fatal("Update reported a pass for an unticked box")
	}
	if !p.Update("02 Casillas.md", "- [x] hecho\n", none) {
		t.Fatal("first pass not reported")
	}
	if done, _ := p.Count(); done != 3 {
		t.Fatalf("Count after pass = %d, want 3", done)
	}
	if p.Update("02 Casillas.md", "- [x] hecho otra vez\n", none) {
		t.Fatal("second pass reported again")
	}
	if p.Update("01 Títulos.md", "## A\n### B\nmás\n", none) {
		t.Fatal("a level passed on disk was reported again")
	}
	if p.Update("02 Casillas.md", "- [ ] deshecho\n", none) {
		t.Fatal("un-passing reported as a pass")
	}
	if done, _ := p.Count(); done != 2 {
		t.Fatalf("Count after un-pass = %d, want 2", done)
	}
	if p.Update("02 Casillas.md", "- [x] de nuevo\n", none) {
		t.Fatal("re-pass after un-pass reported again")
	}
	if p.Update("Mi nota.md", "- [x] x\n", none) {
		t.Fatal("a non-lesson note reported a pass")
	}
	if p.Update("00 Empieza aquí.md", "- [x] x\n", none) {
		t.Fatal("the index reported a pass")
	}
}

func TestLevel(t *testing.T) {
	p := tutor.Scan(t.TempDir())
	cases := map[string]int{
		"00 Empieza aquí.md": 0,
		"01 Títulos.md":      1,
		"02 Casillas.md":     2,
		"07 Explorador.md":   7,
		"07 Explorador":      7,
		"Mi nota.md":         0,
	}
	for file, want := range cases {
		if got := p.Level(file); got != want {
			t.Errorf("Level(%q) = %d, want %d", file, got, want)
		}
	}
}

func TestNilProgress(t *testing.T) {
	var p *tutor.Progress
	if p.Update("02 Casillas.md", "- [x] a\n", none) {
		t.Fatal("nil Update reported a pass")
	}
	if done, total := p.Count(); done != 0 || total != 0 {
		t.Fatalf("nil Count = %d/%d", done, total)
	}
}
