// Package tutor is Misoku's built-in tutorial: a vault of lesson notes grouped
// in routes, each lesson's steps checked against the note's text, the disk or
// the app's events.
package tutor

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"

	"golang.org/x/text/unicode/norm"
)

// The whole folder, subfolders included: each route keeps its lessons in its
// own folder, so a pattern of files at the root would leave them out.
//
//go:embed lessons
var lessons embed.FS

// Bundled returns the embedded lesson notes, rooted at the lessons folder.
func Bundled() fs.FS {
	sub, err := fs.Sub(lessons, "lessons")
	if err != nil {
		panic(err) // the directory is embedded at build time
	}
	return sub
}

// Dir returns the tutorial vault folder, next to Misoku's config file, or ""
// when the user config directory is unknown.
func Dir() string {
	p := config.DefaultPath()
	if p == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(p), "tutor")
}

// oldLessonFiles are the eight notes of the pre-routes tutorial, in order.
// Install moves them to the Trash once; the renamed B0x notes never match.
var oldLessonFiles = []string{
	"00 Empieza aquí.md",
	"01 Títulos.md",
	"02 Casillas.md",
	"03 Menú de bloques.md",
	"04 Links.md",
	"05 Links a títulos.md",
	"06 Referencias a bloque.md",
	"07 Explorador.md",
}

// Install copies every file of fsys into dir that dir does not already have,
// so a player's edits survive. With reset, dir is first moved to the Trash
// (see core.MoveToTrash) and the lessons start over.
func Install(fsys fs.FS, dir string, reset bool) error {
	_, err := InstallWithNotice(fsys, dir, reset)
	return err
}

// InstallWithNotice is Install, and it also reports the one-line notice for
// the next time the tutorial opens, when the old level notes were retired.
func InstallWithNotice(fsys fs.FS, dir string, reset bool) (string, error) {
	if dir == "" {
		return "", errors.New("tutor: no folder for the tutorial")
	}
	clean := filepath.Clean(dir)
	if home, err := os.UserHomeDir(); clean == string(filepath.Separator) || (err == nil && clean == filepath.Clean(home)) {
		return "", errors.New("tutor: refusing to use " + clean + " as the tutorial folder")
	}
	if reset {
		// The player's edits and own notes go to the Trash, not away for good.
		if _, err := os.Lstat(clean); err == nil {
			if err := core.MoveToTrash(clean); err != nil {
				return "", err
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}
	if err := os.MkdirAll(clean, 0o755); err != nil {
		return "", err
	}
	// Copy the new lessons first: if that fails, the old tutorial is still
	// there to open, and the retirement has not happened yet. Only then move
	// the old notes to the Trash.
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(clean, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if _, err := os.Lstat(target); err == nil {
			return nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return "", err
	}
	return migrateOldLessons(clean)
}

// migrateOldLessons retires the pre-routes notes once. It matches names by
// their NFC form, so a note written on macOS with decomposed accents (NFD) is
// the same note; any other name is never touched. The notice is "" when there
// was nothing to retire or the migration already ran.
func migrateOldLessons(dir string) (string, error) {
	store, err := Load(dir)
	if err != nil {
		return "", err
	}
	if store.Migrated() {
		return "", nil
	}
	moved, err := trashOldLessons(dir)
	if err != nil {
		return "", err
	}
	store.SetMigrated(true)
	if err := store.Save(dir); err != nil {
		return "", err
	}
	if moved == 0 {
		return "", nil
	}
	return migrationNotice(), nil
}

// trashOldLessons moves the old notes it finds in dir to the Trash and
// returns how many it moved.
func trashOldLessons(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	present := map[string]string{} // NFC name -> name on disk
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		present[norm.NFC.String(e.Name())] = e.Name()
	}
	moved := 0
	for _, old := range oldLessonFiles {
		name, ok := present[norm.NFC.String(old)]
		if !ok {
			continue
		}
		if err := core.MoveToTrash(filepath.Join(dir, name)); err != nil {
			return moved, err
		}
		delete(present, norm.NFC.String(old))
		moved++
	}
	return moved, nil
}

// migrationNotice is what the tutorial says the next time it opens after the
// old level notes were retired. Windows has no Trash, so there they are gone.
func migrationNotice() string { return migrationNoticeFor(core.HasTrash()) }

// migrationNoticeFor is migrationNotice with the system's Trash as a
// parameter, so a test can prove both texts everywhere.
func migrationNoticeFor(hasTrash bool) string {
	if hasTrash {
		return "Tutorial actualizado: las notas del tutorial anterior están en la papelera."
	}
	return "Tutorial actualizado: las notas del tutorial anterior se borraron (este sistema no tiene papelera)."
}
