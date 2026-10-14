package core

import (
	"os"
	"path/filepath"
)

// writeFileAtomic replaces abs with data in one step: it writes a temporary
// file next to it, flushes it to disk and renames it over the original. A
// crash or a full disk mid-save leaves the old note intact instead of a
// truncated one, and programs that already had the file open keep reading the
// complete old version.
//
// A symlinked note is written at its target, so the link survives, and an
// existing file keeps its permissions.
func writeFileAtomic(abs string, data []byte) error {
	if target, err := filepath.EvalSymlinks(abs); err == nil {
		abs = target
	}
	perm := os.FileMode(0o644)
	if info, err := os.Stat(abs); err == nil {
		perm = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(abs), "."+filepath.Base(abs)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return err
	}
	ok = true
	return nil
}
