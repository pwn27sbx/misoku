package gitsync

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// lockName is the vault's sync lock file, inside its git folder.
const lockName = "misoku-sync.lock"

// LockVault takes the sync lock of r's vault without waiting (S11). ok is
// false when another Misoku window already holds it.
//
// The lock lives in the vault's git folder, found by asking git, so every
// window on the same vault meets on the same file however it named the
// folder (a symlink, another letter case), a `.git` file pointing elsewhere
// is followed, and the lock is never a change to commit.
func LockVault(ctx context.Context, r *Runner) (l *Lock, ok bool, err error) {
	out, err := r.Run(ctx, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return nil, false, fmt.Errorf("could not find the git folder: %w", err)
	}
	return TryLock(filepath.Join(filepath.FromSlash(strings.TrimSpace(string(out))), lockName))
}
