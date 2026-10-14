package core

import (
	"os"
	"path/filepath"
	"runtime"
)

// writeFileAtomic replaces abs with data in one step: it writes a temporary
// file next to it, flushes it to disk and renames it over the original. A
// crash or a full disk mid-save leaves the old note intact instead of a
// truncated one, and programs that already had the file open keep reading the
// complete old version.
//
// A symlinked note is written at its target, so the link survives, and an
// existing file keeps its permissions. On Windows, where a file open in
// another program cannot be replaced, it falls back to writing in place.
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
		if runtime.GOOS == "windows" {
			// Windows refuses to replace a file another program holds open
			// (a sync client, an editor): write in place, as Misoku always
			// did there, rather than fail the save. The temp file is removed
			// by the deferred cleanup.
			return os.WriteFile(abs, data, perm)
		}
		return err
	}
	ok = true
	return nil
}
