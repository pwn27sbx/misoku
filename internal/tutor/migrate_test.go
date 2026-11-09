package tutor_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/tutor"

	"golang.org/x/text/unicode/norm"
)

// tempHome points HOME and XDG_DATA_HOME at a temp folder, so no test ever
// moves anything to the real Trash.
func tempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	return home
}

func mustWrite(t *testing.T, dir, name, text string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// S5/D7: the eight old notes go to the Trash once, with one notice; the
// player's own notes and the renamed lessons are never touched.
func TestInstallRetiresOldNotesOnce(t *testing.T) {
	tempHome(t)
	dir := filepath.Join(t.TempDir(), "tutor")
	mustWrite(t, dir, "00 Empieza aquí.md", "viejo")
	mustWrite(t, dir, "07 Explorador.md", "viejo")
	mustWrite(t, dir, "Mi nota.md", "mía")

	notice, err := tutor.InstallWithNotice(tutor.Bundled(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if notice == "" || !strings.Contains(notice, "Tutorial actualizado") {
		t.Fatalf("notice = %q, want the one-line migration notice", notice)
	}
	want := "papelera"
	if runtime.GOOS == "windows" {
		want = "borraron"
	}
	if !strings.Contains(notice, want) {
		t.Fatalf("notice = %q, want %q", notice, want)
	}
	for _, old := range []string{"00 Empieza aquí.md", "07 Explorador.md"} {
		if _, err := os.Stat(filepath.Join(dir, old)); !os.IsNotExist(err) {
			t.Errorf("%s is still in the tutorial folder", old)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "Mi nota.md")); err != nil {
		t.Errorf("the player's note was touched: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "B01 Títulos.md")); err != nil {
		t.Errorf("the renamed lesson was not installed: %v", err)
	}
	store, err := tutor.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !store.Migrated() {
		t.Fatal("migrated was not recorded")
	}
	if core.HasTrash() {
		trash := map[string]string{
			"darwin": filepath.Join(os.Getenv("HOME"), ".Trash"),
			"linux":  filepath.Join(os.Getenv("XDG_DATA_HOME"), "Trash", "files"),
		}[runtime.GOOS]
		for _, old := range []string{"00 Empieza aquí.md", "07 Explorador.md"} {
			if _, err := os.Stat(filepath.Join(trash, old)); err != nil {
				t.Errorf("%s did not reach the Trash: %v", old, err)
			}
		}
	}

	// Once means once: a note the player creates later with an old name stays.
	mustWrite(t, dir, "00 Empieza aquí.md", "nueva")
	notice, err = tutor.InstallWithNotice(tutor.Bundled(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if notice != "" {
		t.Fatalf("second install reported %q", notice)
	}
	if _, err := os.Stat(filepath.Join(dir, "00 Empieza aquí.md")); err != nil {
		t.Errorf("a later note with an old name was trashed: %v", err)
	}
}

// S5: names are matched by NFC, so a note written on macOS with decomposed
// accents is still an old note.
func TestInstallRetiresDecomposedNames(t *testing.T) {
	tempHome(t)
	dir := filepath.Join(t.TempDir(), "tutor")
	nfd := norm.NFD.String("01 Títulos.md")
	if nfd == "01 Títulos.md" {
		t.Skip("this operating system cannot hold a decomposed name")
	}
	mustWrite(t, dir, nfd, "viejo")

	notice, err := tutor.InstallWithNotice(tutor.Bundled(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if notice == "" {
		t.Fatal("a decomposed old name was not retired")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if norm.NFC.String(e.Name()) == norm.NFC.String("01 Títulos.md") {
			t.Fatalf("%s is still there", e.Name())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "B01 Títulos.md")); err != nil {
		t.Errorf("the new lesson was not installed: %v", err)
	}
}

// S5: a fresh tutorial records migrated, so a later install has nothing to do.
func TestInstallWithNothingToRetire(t *testing.T) {
	tempHome(t)
	dir := filepath.Join(t.TempDir(), "tutor")
	notice, err := tutor.InstallWithNotice(tutor.Bundled(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if notice != "" {
		t.Fatalf("notice = %q, want none", notice)
	}
	store, err := tutor.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !store.Migrated() {
		t.Fatal("migrated was not recorded on a fresh install")
	}
}
