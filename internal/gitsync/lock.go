package gitsync

import "os"

// Lock is held by the one Misoku that runs git for a vault. Other windows on
// the same vault see it taken and leave syncing to the holder.
//
// It is an operating-system lock on an open file (flock on macOS and Linux,
// LockFileEx on Windows), so it disappears when the holder closes it, exits
// or is killed: a crash never leaves a stale lock behind. The file itself may
// stay on disk; only the lock on it matters.
type Lock struct{ f *os.File }

// TryLock takes the lock at path without waiting. ok is false when another
// process (or another Lock in this one) already holds it.
func TryLock(path string) (l *Lock, ok bool, err error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, false, err
	}
	got, err := tryLockFile(f)
	if err != nil || !got {
		f.Close()
		return nil, false, err
	}
	return &Lock{f: f}, true, nil
}

// Unlock releases the lock.
func (l *Lock) Unlock() {
	if l == nil || l.f == nil {
		return
	}
	unlockFile(l.f)
	l.f.Close()
	l.f = nil
}
