// Package tutor is Misoku's built-in tutorial: a vault of lesson notes, each
// one a level whose task is checked live against the note's text.
package tutor

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
)

//go:embed lessons/*.md
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

// Install copies every file of fsys into dir that dir does not already have,
// so a player's edits survive. With reset, dir is first moved to the Trash
// (see core.MoveToTrash) and the lessons start over.
func Install(fsys fs.FS, dir string, reset bool) error {
	if dir == "" {
		return errors.New("tutor: no folder for the tutorial")
	}
	clean := filepath.Clean(dir)
	if home, err := os.UserHomeDir(); clean == string(filepath.Separator) || (err == nil && clean == filepath.Clean(home)) {
		return errors.New("tutor: refusing to use " + clean + " as the tutorial folder")
	}
	if reset {
		// The player's edits and own notes go to the Trash, not away for good.
		if _, err := os.Lstat(clean); err == nil {
			if err := core.MoveToTrash(clean); err != nil {
				return err
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(clean, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
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
}
