package core

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const atomicRenameRetries = 5

var (
	atomicRename      = os.Rename
	retryAtomicRename = runtime.GOOS == "windows"
	atomicRetryDelay  = 30 * time.Millisecond
)

// WriteFileAtomic replaces abs with data in one step: it writes a temporary
// file next to it, flushes it to disk and renames it over the original. A
// crash or a full disk mid-save leaves the old note intact instead of a
// truncated one, and programs that already had the file open keep reading the
// complete old version.
//
// A symlinked note is written at its target, so the link survives, and an
// existing file keeps its permissions. On Windows, where antivirus or sync
// clients briefly hold a handle on the target file, it retries the atomic
// rename before falling back to writing in place.
func WriteFileAtomic(abs string, data []byte) error {
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

	attempts := 1
	if retryAtomicRename {
		attempts = atomicRenameRetries
	}
	var renameErr error
	for i := 0; i < attempts; i++ {
		if renameErr = atomicRename(tmpName, abs); renameErr == nil {
			ok = true
			return nil
		}
		if i+1 < attempts && atomicRetryDelay > 0 {
			time.Sleep(atomicRetryDelay)
		}
	}
	if runtime.GOOS == "windows" {
		// Windows refuses to replace a file another program holds open
		// (a sync client, an editor): write in place after retries rather
		// than fail the save. The temp file is removed by the deferred cleanup.
		return os.WriteFile(abs, data, perm)
	}
	return renameErr
}
