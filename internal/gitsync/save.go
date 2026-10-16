package gitsync

import (
	"context"
	"fmt"
	"time"
)

// CommitLocal is the commit Misoku makes after notes are saved (S3): it
// commits the vault's changes (CommitVault) and counts the commits origin
// does not have yet, and nothing else. It never fetches, applies or pushes,
// so it needs no network and cannot change a note on disk; the next sync
// pushes the commit.
//
// held must be the vault's Lock, held by the caller (S11). Like a sync, it
// stops on a git lock file Misoku did not leave, and cleans up one it did
// (S13).
func CommitLocal(ctx context.Context, r *Runner, held *Lock, opts SyncOptions) SyncResult {
	var res SyncResult
	stop := res.stopper(ctx)
	if o, err := begin(ctx, r, held); err != nil {
		return stop(o, err)
	}
	defer func() { r.marker = "" }()

	var err error
	if res.Commit, err = CommitVault(ctx, r, opts.Device); err != nil {
		return stop(SyncFailed, fmt.Errorf("commit: %w", err))
	}
	// Ahead of the upstream every sync sets: local refs only.
	res.Pending = pending(ctx, r, "")
	res.Outcome = Synced
	return res
}

// ConflictCopyName returns a free name for a conflict copy of p (slash
// separated, relative to the vault dir), named as a sync names them (S6):
// "<name> (conflict <device> YYYY-MM-DD HHMM)<ext>" in p's folder, device
// left out when empty, and " 2", " 3"... appended when the name is taken
// on disk. The app uses it to keep the version on disk when a note that
// changed there is saved over.
func ConflictCopyName(dir, p, device string, now time.Time) (string, error) {
	return conflictName(dir, p, device, now, map[string]bool{})
}
