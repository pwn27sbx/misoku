package gitsync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SyncOutcome says how a SyncVault call ended.
type SyncOutcome int

// How a sync ends. Whatever the outcome, no note is lost: local commits
// stay, and nothing the user saved during the sync is overwritten.
const (
	// SyncFailed: something unexpected went wrong; Err says what.
	SyncFailed SyncOutcome = iota
	// Synced: the vault and origin hold the same notes.
	Synced
	// SyncOffline: origin could not be reached (S7). The commits stay in
	// the vault; Pending counts them and the next sync pushes them.
	SyncOffline
	// SyncRetry: something changed during the sync, so it stopped short
	// (S12): a file was saved while the remote's changes were being
	// applied, another device pushed first, or the git of an interrupted
	// sync still runs (S13). The next sync retries.
	SyncRetry
	// SyncUnrelated: the vault and origin share no history. Only
	// `misoku sync init` joins them (S10).
	SyncUnrelated
	// SyncRewritten: origin's branch was rewritten (a force-push), so
	// nothing was applied or pushed (S2).
	SyncRewritten
	// SyncDenied: origin was reached but would not let Misoku in (S8):
	// credentials missing or refused, or an SSH host key not known yet.
	// Err says how to fix it; commits stay pending and the next sync tries
	// again.
	SyncDenied
)

// String names the outcome for logs and tests.
func (o SyncOutcome) String() string {
	switch o {
	case SyncFailed:
		return "failed"
	case Synced:
		return "synced"
	case SyncOffline:
		return "offline"
	case SyncRetry:
		return "retry"
	case SyncUnrelated:
		return "unrelated"
	case SyncRewritten:
		return "rewritten"
	case SyncDenied:
		return "denied"
	}
	return "outcome(" + strconv.Itoa(int(o)) + ")"
}

// SyncOptions are the settings of one SyncVault call.
type SyncOptions struct {
	// Device names this computer in commits (S1).
	Device string
	// AllowUnrelatedHistories joins a vault and a remote that share no
	// history. Only `misoku sync init` sets it, for a folder and a remote
	// that both hold notes (S10); a normal sync stops with SyncUnrelated.
	AllowUnrelatedHistories bool
	// Now gives the local time used in conflict copy names; nil means
	// time.Now.
	Now func() time.Time
}

// SyncResult says what SyncVault did, for the status bar.
type SyncResult struct {
	// Outcome says how the sync ended.
	Outcome SyncOutcome
	// Branch is origin's default branch, the one synced (empty when
	// origin could not be asked).
	Branch string
	// Commit is the local commit of the vault's changes, made first.
	Commit CommitResult
	// Updated is true when changes from origin reached the vault's files.
	Updated bool
	// Pushed is true when commits were pushed to origin.
	Pushed bool
	// Pending counts the local commits origin does not have yet.
	Pending int
	// Conflicts lists the conflict copies this sync created (S6),
	// slash-separated and relative to the vault.
	Conflicts []string
	// Err is the cause when Outcome is not Synced. Credentials in URLs it
	// quotes are removed (RedactURL).
	Err error
}

// Test hooks: a test saves a note or pushes from another device at the
// exact moment SyncVault applies or pushes. Always nil outside tests.
var testHookBeforeApply, testHookBeforePush func()

// SyncVault brings the vault and origin to the same notes (D4, S3, S4,
// S6, S7, S12):
//
//  1. commits the vault's changes (CommitVault);
//  2. fetches origin's default branch (`ls-remote --symref origin HEAD`,
//     main when origin is empty);
//  3. never runs `git pull`: when the branches diverged it computes the
//     union with `merge-tree --write-tree`, which touches no file, keeps
//     both versions of every conflict (see union) and records the result
//     as a commit;
//  4. moves the branch with `merge --ff-only`, which refuses to overwrite
//     a file the user changed since step 1: then nothing is applied, the
//     save goes into the next commit and the next sync retries;
//  5. pushes, and sets the branch's upstream to origin's branch.
//
// held must be the vault's Lock, held by the caller: only one Misoku may
// run git on a vault at a time (S11).
func SyncVault(ctx context.Context, r *Runner, held *Lock, opts SyncOptions) SyncResult {
	var res SyncResult
	if held == nil || held.f == nil {
		res.Err = errors.New("sync needs the vault's lock")
		return res
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	stop := func(o SyncOutcome, err error) SyncResult {
		res.Outcome, res.Err = o, redact(err)
		if ctx.Err() != nil {
			res.Outcome = SyncFailed
		}
		return res
	}

	// S13: a sync that died half-way may have left git's index.lock. It is
	// cleaned up only when it is Misoku's; then this sync simply redoes the
	// work, as every step is safe to repeat.
	out, err := r.Run(ctx, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return stop(SyncFailed, err)
	}
	gitDir := filepath.FromSlash(strings.TrimSpace(string(out)))
	marker := filepath.Join(gitDir, markerName)
	if err := recoverIndexLock(marker, false); err != nil {
		if errors.Is(err, errGitStillRunning) {
			return stop(SyncRetry, err)
		}
		return stop(SyncFailed, err)
	}
	// A lock still there is someone else's (S2): leave it and stop.
	if lock := filepath.Join(gitDir, "index.lock"); fileExists(lock) {
		return stop(SyncFailed, &LockedError{Path: lock})
	}
	r.marker = marker
	defer func() { r.marker = "" }()

	if res.Commit, err = CommitVault(ctx, r, opts.Device); err != nil {
		return stop(SyncFailed, fmt.Errorf("commit: %w", err))
	}

	branch, remoteEmpty, err := defaultBranch(ctx, r)
	if err != nil {
		res.Pending = pending(ctx, r, "")
		var gerr *gitError
		var serr *stoppedError
		if !errors.As(err, &gerr) && !errors.As(err, &serr) {
			return stop(SyncFailed, err) // origin answered, but not as expected
		}
		return stop(remoteFailure("could not reach origin", err))
	}
	res.Branch = branch
	tracking := "refs/remotes/origin/" + branch
	local, err := resolve(ctx, r, "HEAD")
	if err != nil {
		return stop(SyncFailed, err)
	}

	if !remoteEmpty {
		target, err := res.integrate(ctx, r, local, branch, opts, now())
		if err != nil || res.Outcome != Synced {
			if err != nil {
				return stop(SyncFailed, err)
			}
			return stop(res.Outcome, res.Err)
		}
		if target != "" {
			if testHookBeforeApply != nil {
				testHookBeforeApply()
			}
			// --no-autostash: a user's merge.autoStash would stash the
			// fresh save and re-apply it with conflict markers. Paths are
			// not quoted, so applyFailure can find them in the message.
			if _, err := r.Run(ctx, "-c", "core.quotePath=false", "merge", "--ff-only", "--no-autostash", "--quiet", target); err != nil {
				res.Conflicts = nil
				res.Pending = pending(ctx, r, branch)
				return stop(applyFailure(err, res.Commit.Skipped))
			}
			res.Updated = true
			if local, err = resolve(ctx, r, "HEAD"); err != nil {
				return stop(SyncFailed, err)
			}
		}
	}
	if local == "" { // nothing anywhere yet
		res.Outcome = Synced
		return res
	}

	ahead, err := countPending(ctx, r, branch)
	if err != nil {
		return stop(SyncFailed, fmt.Errorf("could not count the commits to push: %w", err))
	}
	if ahead > 0 {
		if testHookBeforePush != nil {
			testHookBeforePush()
		}
		out, err := r.Run(ctx, "push", "--porcelain", "origin", "HEAD:refs/heads/"+branch)
		if err != nil {
			res.Pending = ahead
			switch rejected, refused := pushRefusal(out); {
			case rejected:
				return stop(SyncRetry, fmt.Errorf("another device pushed first; the next sync joins its changes: %w", err))
			case refused != "":
				return stop(SyncFailed, fmt.Errorf("origin refused the push (%s); nothing was lost, fix what origin says and sync again: %w", refused, err))
			}
			return stop(remoteFailure("could not push to origin", err))
		}
		res.Pushed = true
		// Push updates origin/<branch> only through the fetch refspec;
		// set it here so a custom refspec cannot leave the count stale.
		if _, err := r.Run(ctx, "update-ref", tracking, local); err != nil {
			return stop(SyncFailed, err)
		}
	}
	if err := setUpstream(ctx, r, branch); err != nil {
		return stop(SyncFailed, err)
	}
	res.Outcome = Synced
	return res
}

// integrate fetches origin's branch and decides what the vault's branch
// must move to: "" when origin brings nothing new, origin's commit for a
// fast-forward, or a new union commit when the two diverged. It sets
// res.Outcome to Synced when the sync may go on, or to why it stops.
func (res *SyncResult) integrate(ctx context.Context, r *Runner, local, branch string, opts SyncOptions, now time.Time) (string, error) {
	tracking := "refs/remotes/origin/" + branch
	old, err := resolve(ctx, r, tracking)
	if err != nil {
		return "", err
	}
	if _, err := r.Run(ctx, "fetch", "--quiet", "--no-tags", "origin", "+refs/heads/"+branch+":"+tracking); err != nil {
		res.Outcome, res.Err = remoteFailure("could not fetch from origin", err)
		res.Pending = pending(ctx, r, branch)
		return "", nil
	}
	theirs, err := resolve(ctx, r, tracking)
	if err != nil {
		return "", err
	}
	if old != "" {
		kept, err := isAncestor(ctx, r, old, theirs)
		if err != nil {
			return "", err
		}
		if !kept {
			// Put origin/<branch> back, so every later sync stops here
			// too until the user decides what to keep.
			if _, err := r.Run(ctx, "update-ref", tracking, old); err != nil {
				return "", err
			}
			res.Outcome = SyncRewritten
			res.Err = fmt.Errorf("origin's %s was rewritten (a force-push), so sync stopped to lose nothing; compare it with `git fetch origin` and `git log origin/%s`, and join it by hand", branch, branch)
			res.Pending = pending(ctx, r, branch)
			return "", nil
		}
	}

	res.Outcome = Synced
	if local == "" {
		return theirs, nil // a vault with no commit yet takes origin's notes
	}
	if behind, err := isAncestor(ctx, r, theirs, local); err != nil || behind {
		return "", err // origin has nothing new
	}
	if ff, err := isAncestor(ctx, r, local, theirs); err != nil || ff {
		if err != nil {
			return "", err
		}
		return theirs, nil
	}
	related, err := haveCommonHistory(ctx, r, local, theirs)
	if err != nil {
		return "", err
	}
	if !related && !opts.AllowUnrelatedHistories {
		res.Outcome = SyncUnrelated
		res.Err = errors.New("this vault and origin have unrelated histories; join them once with `misoku sync init <url>`")
		res.Pending = pending(ctx, r, branch)
		return "", nil
	}
	commit, copies, err := union(ctx, r, local, theirs, !related, opts.Device, now)
	if err != nil {
		return "", err
	}
	res.Conflicts = copies
	return commit, nil
}

// defaultBranch asks origin for its default branch (S4). empty is true
// when origin has no main and its HEAD names no commit; the branch is then
// main. A HEAD that names a branch nobody pushed (a plain `git init --bare`
// on master or trunk) is not trusted when origin already has main: that is
// where Misoku pushed.
func defaultBranch(ctx context.Context, r *Runner) (branch string, empty bool, err error) {
	out, err := r.Run(ctx, "ls-remote", "--symref", "origin", "HEAD", "refs/heads/main")
	if err != nil {
		return "", false, err
	}
	hasHead, hasMain := false, false
	for _, line := range strings.Split(string(out), "\n") {
		left, name, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok {
			continue
		}
		target, isRef := strings.CutPrefix(left, "ref: ")
		switch {
		case name == "HEAD" && isRef:
			branch = strings.TrimPrefix(target, "refs/heads/")
		case name == "HEAD":
			hasHead = true
		case name == "refs/heads/main" && !isRef:
			hasMain = true
		}
	}
	switch {
	case hasHead && branch == "":
		return "", false, errors.New("origin did not say which branch is its default")
	case hasHead:
		return branch, false, nil
	case hasMain:
		return "main", false, nil
	}
	return "main", true, nil
}

// resolve returns the commit rev names, or "" when it does not exist
// (such as HEAD before the first commit).
func resolve(ctx context.Context, r *Runner, rev string) (string, error) {
	out, err := r.Run(ctx, "rev-parse", "-q", "--verify", rev+"^{commit}")
	if err != nil {
		if exitCode(err) == 1 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// isAncestor reports whether commit a is b or one of b's ancestors.
func isAncestor(ctx context.Context, r *Runner, a, b string) (bool, error) {
	_, err := r.Run(ctx, "merge-base", "--is-ancestor", a, b)
	if err == nil {
		return true, nil
	}
	if exitCode(err) == 1 {
		return false, nil
	}
	return false, err
}

// haveCommonHistory reports whether a and b share any commit.
func haveCommonHistory(ctx context.Context, r *Runner, a, b string) (bool, error) {
	_, err := r.Run(ctx, "merge-base", a, b)
	if err == nil {
		return true, nil
	}
	if exitCode(err) == 1 {
		return false, nil
	}
	return false, err
}

// pending counts the commits origin does not have, for the status bar:
// when git cannot answer it reports 0. See countPending.
func pending(ctx context.Context, r *Runner, branch string) int {
	n, _ := countPending(ctx, r, branch)
	return n
}

// countPending counts the commits origin does not have: those ahead of
// origin/<branch>, or of the branch's upstream when branch is unknown
// (origin unreachable), or every commit when there is neither.
func countPending(ctx context.Context, r *Runner, branch string) (int, error) {
	base := "@{upstream}"
	if branch != "" {
		base = "refs/remotes/origin/" + branch
	}
	rng := "HEAD"
	if have, err := resolve(ctx, r, base); err == nil && have != "" {
		rng = base + "..HEAD"
	}
	out, err := r.Run(ctx, "rev-list", "--count", rng)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

// applyFailure is the outcome of a `merge --ff-only` that would not move
// the branch. A file the user saved during the sync is the expected case
// (S12): the next sync retries. A file left out for being too big (D7) at
// a path origin now has would block every sync the same way, so it is
// named instead. Anything else is git's error.
func applyFailure(err error, skipped []BigFile) (SyncOutcome, error) {
	msg := err.Error()
	if !strings.Contains(msg, "would be overwritten by merge") {
		return SyncFailed, fmt.Errorf("could not apply origin's changes: %w", err)
	}
	for _, line := range strings.Split(msg, "\n") {
		name := strings.TrimSpace(line)
		for _, big := range skipped {
			if name == big.Path {
				return SyncFailed, fmt.Errorf("%s is larger than %s, so it stays out of sync, and origin has a file with the same name: move or rename %s, then sync again: %w", big.Path, sizeLimit(), big.Path, err)
			}
		}
	}
	return SyncRetry, fmt.Errorf("a file changed while syncing, so origin's changes were not applied yet (nothing was lost; the next sync tries again): %w", err)
}

// setUpstream makes origin/<branch> the upstream of the checked-out branch
// (S4), writing the config only when it differs.
func setUpstream(ctx context.Context, r *Runner, branch string) error {
	out, err := r.Run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return err
	}
	local := strings.TrimSpace(string(out))
	for key, want := range map[string]string{
		"branch." + local + ".remote": "origin",
		"branch." + local + ".merge":  "refs/heads/" + branch,
	} {
		out, err := r.Run(ctx, "config", "--get", key)
		if err != nil && exitCode(err) != 1 {
			return err
		}
		if strings.TrimSpace(string(out)) == want {
			continue
		}
		if _, err := r.Run(ctx, "config", key, want); err != nil {
			return err
		}
	}
	return nil
}

// sizeLimit is MaxFileSize in words.
func sizeLimit() string {
	if MaxFileSize >= 1<<20 {
		return strconv.FormatInt(MaxFileSize>>20, 10) + " MB"
	}
	return strconv.FormatInt(MaxFileSize, 10) + " bytes"
}

// fileExists reports whether something is at p.
func fileExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}
