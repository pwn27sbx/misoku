package core

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// errNoTrash means this OS has no Trash Misoku knows how to use, so deleting
// stays permanent there (Windows).
var errNoTrash = errors.New("no trash on this system")

// moveToTrash moves abs (a note or a folder) to the user's Trash so a delete
// can be undone from Finder or the file manager:
//
//   - macOS: ~/.Trash
//   - Linux: the freedesktop.org Trash ($XDG_DATA_HOME/Trash, by default
//     ~/.local/share/Trash), with the .trashinfo file that lets the file
//     manager restore it to where it was.
//
// A name already in the Trash gets a numbered suffix, so nothing there is
// overwritten. If the move fails the item is left where it was.
func moveToTrash(abs string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find the Trash: %w", err)
	}
	switch runtime.GOOS {
	case "darwin":
		dir := filepath.Join(home, ".Trash")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		_, err := renameUnique(abs, dir, nil)
		return err
	case "linux":
		data := os.Getenv("XDG_DATA_HOME")
		if data == "" {
			data = filepath.Join(home, ".local", "share")
		}
		files := filepath.Join(data, "Trash", "files")
		info := filepath.Join(data, "Trash", "info")
		for _, d := range []string{files, info} {
			if err := os.MkdirAll(d, 0o700); err != nil {
				return err
			}
		}
		_, err := renameUnique(abs, files, func(name string) error {
			// The .trashinfo is created first and exclusively: it reserves the
			// name, as the spec asks.
			f, err := os.OpenFile(filepath.Join(info, name+".trashinfo"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintf(f, "[Trash Info]\nPath=%s\nDeletionDate=%s\n",
				trashInfoPath(abs), time.Now().Format("2006-01-02T15:04:05"))
			if cerr := f.Close(); werr == nil {
				werr = cerr
			}
			return werr
		})
		return err
	}
	return errNoTrash
}

// renameFile is os.Rename, replaceable in tests to simulate another volume.
var renameFile = os.Rename

// moveFile renames from to to. Between volumes (EXDEV), where a rename is
// impossible, it copies everything and only then removes the original; if the
// copy fails, the partial copy is removed and the original stays untouched.
func moveFile(from, to string) error {
	err := renameFile(from, to)
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	if err := copyTree(from, to); err != nil {
		os.RemoveAll(to)
		return err
	}
	return os.RemoveAll(from)
}

// copyTree copies a file, symlink or folder, keeping permissions.
func copyTree(from, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(from)
		if err != nil {
			return err
		}
		return os.Symlink(target, to)
	case info.IsDir():
		if err := os.Mkdir(to, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(from)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyTree(filepath.Join(from, e.Name()), filepath.Join(to, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// renameUnique moves abs into dir under its own name, or "name 2", "name 3"…
// when taken. reserve, if set, runs first for the chosen name and may refuse
// it (os.ErrExist) to try the next one.
func renameUnique(abs, dir string, reserve func(name string) error) (string, error) {
	base := filepath.Base(abs)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; i < 1000; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s %d%s", stem, i, ext)
		}
		dst := filepath.Join(dir, name)
		if _, err := os.Lstat(dst); err == nil {
			continue
		}
		if reserve != nil {
			if err := reserve(name); errors.Is(err, os.ErrExist) {
				continue
			} else if err != nil {
				return "", err
			}
		}
		if err := moveFile(abs, dst); err != nil {
			if reserve != nil {
				os.Remove(filepath.Join(filepath.Dir(dir), "info", name+".trashinfo"))
			}
			return "", fmt.Errorf("could not move to the Trash: %w", err)
		}
		return dst, nil
	}
	return "", fmt.Errorf("could not move to the Trash: too many items named %s", base)
}

// trashInfoPath percent-encodes an absolute path for a .trashinfo file,
// keeping the slashes.
func trashInfoPath(abs string) string {
	parts := strings.Split(filepath.ToSlash(abs), "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// HasTrash reports whether deleting moves to a Trash on this system (macOS
// and Linux) rather than removing for good.
func HasTrash() bool { return runtime.GOOS == "darwin" || runtime.GOOS == "linux" }

// removePath deletes abs: to the Trash where there is one, permanently
// otherwise.
func removePath(abs string) error {
	err := moveToTrash(abs)
	if errors.Is(err, errNoTrash) {
		return os.RemoveAll(abs)
	}
	return err
}
