package gitsync

import (
	"context"
	"path/filepath"
	"time"
)

// SetSyncHooks makes SyncVault call beforeApply right before it moves the
// branch with `merge --ff-only`, and beforePush right before it pushes, so
// a test can save a note or push from another device at that moment. The
// returned function removes the hooks.
func SetSyncHooks(beforeApply, beforePush func()) func() {
	testHookBeforeApply, testHookBeforePush = beforeApply, beforePush
	return func() { testHookBeforeApply, testHookBeforePush = nil, nil }
}

// WriteRunMarker leaves the marker SyncVault keeps while one of its git
// commands that take the index lock runs: git process pid, started at
// start, in the repository whose git folder is gitDir, with the 20-second
// time limit the tests run git with.
func WriteRunMarker(gitDir string, pid int, start time.Time) error {
	return writeMarker(filepath.Join(gitDir, markerName), pid, start, 20*time.Second)
}

// LeftoverIsMisokus is leftoverIsMisokus for a marker of git pid, started
// at start with time limit limit.
func LeftoverIsMisokus(pid int, start time.Time, limit time.Duration, mod time.Time, killedNow, gitDiesWithMisoku bool) bool {
	return leftoverIsMisokus(marker{pid: pid, start: start, limit: limit}, mod, killedNow, gitDiesWithMisoku)
}

// EnvFor is the environment r runs the git subcommand cmd with.
func (r *Runner) EnvFor(ctx context.Context, cmd string) []string {
	return r.envFor(ctx, cmd)
}

// SetExecutable makes the gh lookup take path as Misoku's own executable.
// The returned function restores it.
func SetExecutable(path string) func() {
	old := executable
	executable = func() (string, error) { return path, nil }
	return func() { executable = old }
}

// SetAppleInstallingGit makes the git check on macOS see Apple's
// installer for git running when running returns true. The returned
// function restores it.
func SetAppleInstallingGit(running func() bool) func() {
	old := appleInstallingGit
	appleInstallingGit = func(context.Context) bool { return running() }
	return func() { appleInstallingGit = old }
}

// SetGOOS makes the git lookup behave as on goos. The returned function
// restores it.
func SetGOOS(goos string) func() {
	old := runtimeGOOS
	runtimeGOOS = goos
	return func() { runtimeGOOS = old }
}

// GitProgram is the git program Runner starts.
func GitProgram() string { return gitProgram() }
