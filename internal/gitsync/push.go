package gitsync

import (
	"context"
	"fmt"
	"strings"
)

// PushVault is the sync Misoku runs as it quits (S14): it commits the
// vault's changes (CommitVault) and pushes them to origin, and nothing
// else. It neither fetches nor applies origin's changes, so stopped at its
// time limit it cannot leave a note half-updated; when origin moved on,
// the push is turned down (SyncRetry) and the commits wait, counted in
// Pending, for the next sync, which joins both sides.
//
// The branch is the vault's upstream, which every sync sets; before the
// first one, origin is asked for its default branch, as SyncVault does.
// held must be the vault's Lock, held by the caller (S11). ctx bounds the
// whole call: when it ends, git is killed and only what git finished
// stays, as after any stopped sync (S13).
func PushVault(ctx context.Context, r *Runner, held *Lock, opts SyncOptions) SyncResult {
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
	local, err := resolve(ctx, r, "HEAD")
	if err != nil {
		return stop(SyncFailed, err)
	}
	if local == "" { // nothing to push yet
		res.Outcome = Synced
		return res
	}
	branch, err := upstreamBranch(ctx, r)
	if err != nil {
		return stop(SyncFailed, err)
	}
	if branch == "" {
		var o SyncOutcome
		if branch, _, o, err = res.originBranch(ctx, r); err != nil {
			return stop(o, err)
		}
	}
	res.Branch = branch
	if o, err := res.push(ctx, r, branch, local); err != nil {
		return stop(o, err)
	}
	res.Outcome = Synced
	return res
}

// upstreamBranch is the branch of origin the checked-out branch tracks, or
// "" when it tracks none there.
func upstreamBranch(ctx context.Context, r *Runner) (string, error) {
	out, err := r.Run(ctx, "rev-parse", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		if exitCode(err) == 128 { // no upstream set
			return "", nil
		}
		return "", err
	}
	name, ok := strings.CutPrefix(strings.TrimSpace(string(out)), "refs/remotes/origin/")
	if !ok {
		return "", nil
	}
	return name, nil
}
