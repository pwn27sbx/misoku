package tutor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
)

func TestInstall(t *testing.T) {
	fsys := fstest.MapFS{
		"B01 Uno.md": {Data: []byte("original")},
		"B02 Dos.md": {Data: []byte("dos")},
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
	if got := read("B01 Uno.md"); got != "original" {
		t.Fatalf("installed %q", got)
	}
	if got := read("B02 Dos.md"); got != "dos" {
		t.Fatalf("installed %q", got)
	}

	path := filepath.Join(dir, "B01 Uno.md")
	if err := os.WriteFile(path, []byte("editado"), 0o644); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(dir, "Mi nota.md")
	if err := os.WriteFile(extra, []byte("mía"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "B02 Dos.md")); err != nil {
		t.Fatal(err)
	}
	if err := Install(fsys, dir, false); err != nil {
		t.Fatal(err)
	}
	if got := read("B01 Uno.md"); got != "editado" {
		t.Fatalf("second Install overwrote an edited lesson: %q", got)
	}
	if got := read("B02 Dos.md"); got != "dos" {
		t.Fatalf("second Install did not restore a missing lesson: %q", got)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	if err := Install(fsys, dir, true); err != nil {
		t.Fatal(err)
	}
	if got := read("B01 Uno.md"); got != "original" {
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
		for name, want := range map[string]string{"Mi nota.md": "mía", "B01 Uno.md": "editado"} {
			b, err := os.ReadFile(filepath.Join(trash, "tutor", name))
			if err != nil || string(b) != want {
				t.Errorf("reset did not move %s to the Trash: %q, %v", name, b, err)
			}
		}
	}
}

// S1/G1: Install copies the route folders, not only files at the root.
func TestInstallCopiesRouteFolders(t *testing.T) {
	fsys := fstest.MapFS{
		"1 Básico/B01 Uno.md":     {Data: []byte("uno")},
		"2 Intermedio/I01 Dos.md": {Data: []byte("dos")},
		"mapa.png":                {Data: []byte("png")},
	}
	dir := filepath.Join(t.TempDir(), "tutor")
	if err := Install(fsys, dir, false); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		"1 Básico/B01 Uno.md":     "uno",
		"2 Intermedio/I01 Dos.md": "dos",
	} {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil || string(b) != want {
			t.Errorf("%s not installed: %q, %v", path, b, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "mapa.png")); err != nil {
		t.Errorf("the root file was not installed: %v", err)
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
