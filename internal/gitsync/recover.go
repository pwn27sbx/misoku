package gitsync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// markerName is the file, in the vault's git folder, that SyncVault keeps
// while one of its git commands runs (S13). It holds that git's pid and
// when the command started. git removes its own index.lock when it exits,
// even on an error; only a git that was killed (its time limit, Misoku
// dying with it, the computer going down) leaves one behind, and then the
// marker is still there to say the lock was Misoku's.
const markerName = "misoku-sync.running"

// lockSlack is how much older than its command a lock may look and still
// be that command's: some file systems keep times to the second (HFS+) or
// to two seconds (FAT).
const lockSlack = 2 * time.Second

// indexLockCommands are the git commands SyncVault runs that take the
// index lock. `status` is not one: Runner.Status runs it with
// --no-optional-locks, so it never holds the lock (it would only while
// refreshing, and a lock seen after it timed out could be anyone's).
var indexLockCommands = map[string]bool{"add": true, "reset": true, "commit": true, "merge": true}

// takesIndexLock reports whether git command name takes the index lock.
func takesIndexLock(name string) bool { return indexLockCommands[name] }

// writeMarker records that git process pid (0 before it starts) runs a
// command that started at start and may run for at most limit.
func writeMarker(path string, pid int, start time.Time, limit time.Duration) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d %d %d\n", pid, start.UnixNano(), int64(limit))), 0o644)
}

// marker is what a marker file says. pid is 0 when unknown.
type marker struct {
	pid   int
	start time.Time
	limit time.Duration
}

// readMarker returns what the marker at path says; ok is false when there
// is none. An unreadable marker gives pid 0 (unknown).
func readMarker(path string) (m marker, ok bool, err error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, false, nil
	}
	if err != nil {
		return m, false, err
	}
	f := strings.Fields(string(b))
	if len(f) != 3 {
		return m, true, nil
	}
	nanos, err1 := strconv.ParseInt(f[1], 10, 64)
	limit, err2 := strconv.ParseInt(f[2], 10, 64)
	if err1 != nil || err2 != nil {
		return m, true, nil
	}
	m.pid, _ = strconv.Atoi(f[0])
	m.start, m.limit = time.Unix(0, nanos), time.Duration(limit)
	return m, true, nil
}

// markerStale is how long past its time limit a marked command may still
// be running. Later, a live process with its pid is taken for another
// program that got the pid (after a restart, say): waiting for it would
// stop every sync for good. What its lock says is still checked.
const markerStale = 10 * time.Minute

// stale reports whether the marked command is long past its time limit.
func (m marker) stale(now time.Time) bool {
	return now.After(m.start.Add(m.limit + markerStale))
}

// removeMarker removes the marker at path, if any.
func removeMarker(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// errGitStillRunning: the git of an interrupted sync still runs, so the
// lock it holds is in use. The next sync looks again.
var errGitStillRunning = errors.New("still running")

// recoverIndexLock cleans up after a git command of a sync that did not
// finish (S13), given its marker. killedNow is true right after Misoku
// itself killed that git at its time limit. The index.lock next to the
// marker is removed only when leftoverIsMisokus says it can only be that
// git's; otherwise it stays, and the sync stops on it so the user can
// remove it (RemoveLeftoverLock). The marker goes once nothing of that git
// is left to check.
func recoverIndexLock(path string, killedNow bool) error {
	m, ok, err := readMarker(path)
	if err != nil || !ok {
		return err
	}
	if m.pid > 0 && processAlive(m.pid) && !m.stale(time.Now()) {
		return fmt.Errorf("git (process %d) of an earlier sync is %w; sync tries again once it is done", m.pid, errGitStillRunning)
	}
	lock := filepath.Join(filepath.Dir(path), "index.lock")
	info, err := os.Lstat(lock)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// Nothing left behind.
	case err != nil:
		return err
	case m.pid <= 0:
		// git's pid was never noted: it may still run. Keep the marker, so
		// the lock is not mistaken for the user's either.
		return &LockedError{Path: lock}
	case leftoverIsMisokus(m, info.ModTime(), killedNow, gitDiesWithMisoku):
		if err := os.Remove(lock); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return removeMarker(path)
}

// leftoverIsMisokus decides whether an index.lock last written at mod is
// what the marked git left, now that it is dead.
//
// Right after Misoku killed it (killedNow), a lock made while the command
// could run is its own: no other git can have taken the lock it held.
// Otherwise Misoku died first, and it depends on what happened to its git:
// where git dies with Misoku (gitDiesWithMisoku: the Windows job object),
// the same window applies. Where it does not (macOS, Linux: git runs in
// its own session), that git went on, finished and removed its own lock,
// and the lock there now can be a git the user runs in a terminal: it is
// never removed automatically.
func leftoverIsMisokus(m marker, mod time.Time, killedNow, gitDiesWithMisoku bool) bool {
	if !killedNow && !gitDiesWithMisoku {
		return false
	}
	return madeDuring(mod, m)
}

// madeDuring reports whether a lock last written at t can be the marked
// command's: not before it started, nor after its time limit.
func madeDuring(t time.Time, m marker) bool {
	return !t.Before(m.start.Add(-lockSlack)) && !t.After(m.start.Add(m.limit+lockSlack))
}

// lockedMessage says what to do about an index.lock that is not Misoku's.
func lockedMessage(lock string) string {
	return fmt.Sprintf("git's index.lock is there: a git command is running in this vault, or one stopped half-way; wait for it to finish, or if no git is running, delete %s", lock)
}

// LockedError is a sync stopped by git's index.lock, which Misoku does not
// remove on its own (S13): the user confirms first (RemoveLeftoverLock).
type LockedError struct {
	// Path is the index.lock file.
	Path string
}

func (e *LockedError) Error() string { return lockedMessage(e.Path) }

// LeftoverLock is an index.lock as the user was shown it.
type LeftoverLock struct {
	Path    string
	Size    int64
	ModTime time.Time
}

// StatLock reads the index.lock at path; ok is false when it is gone.
func StatLock(path string) (l LeftoverLock, ok bool, err error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return l, false, nil
	}
	if err != nil {
		return l, false, err
	}
	return LeftoverLock{Path: path, Size: info.Size(), ModTime: info.ModTime()}, true, nil
}

// ErrLockChanged: the index.lock changed or went away after the user was
// shown it, so it was not removed.
var ErrLockChanged = errors.New("the lock file changed since it was shown, so it was left alone; check again")

// RemoveLeftoverLock removes the index.lock the user confirmed, only if it
// is still the same file (same size and time) as when it was shown: a
// lock that changed belongs to a git that is running.
func RemoveLeftoverLock(l LeftoverLock) error {
	now, ok, err := StatLock(l.Path)
	if err != nil {
		return err
	}
	if !ok || now.Size != l.Size || !now.ModTime.Equal(l.ModTime) || filepath.Base(l.Path) != "index.lock" {
		return ErrLockChanged
	}
	return os.Remove(l.Path)
}
