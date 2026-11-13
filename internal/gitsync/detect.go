package gitsync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// Status says whether Misoku syncs the folder it opened.
type Status int

const (
	// Off: the folder is not a listed vault, or MISOKU_SYNC=0. No git runs.
	Off Status = iota
	// Ready: the vault can sync.
	Ready
	// Paused: the vault is listed but something must be fixed first; the
	// message says what. Nothing was changed.
	Paused
)

// Reason is a stable code for why sync is off or paused, for code that
// must tell the cases apart without reading the message.
type Reason string

// Why sync is off or paused.
const (
	ReasonDisabled     Reason = "disabled"      // MISOKU_SYNC=0
	ReasonNotListed    Reason = "not-listed"    // folder not in sync.vaults
	ReasonGitMissing   Reason = "git-missing"   // no git, or it does not run
	ReasonGitTooOld    Reason = "git-too-old"   // older than MinVersion
	ReasonNotARepo     Reason = "not-a-repo"    // the vault is not a git repository
	ReasonNotRepoRoot  Reason = "not-repo-root" // the vault is a subfolder of a repository
	ReasonNoRemote     Reason = "no-remote"     // no remote named origin
	ReasonNoIdentity   Reason = "no-identity"   // user.name or user.email unset
	ReasonDetachedHead Reason = "detached-head"
	ReasonInProgress   Reason = "in-progress" // the user's merge, rebase, cherry-pick or revert
	ReasonLocked       Reason = "locked"      // git's index.lock is held, and not by Misoku
	ReasonGitFailed    Reason = "git-failed"  // git failed for another reason (a timeout, the disk); the next sync checks again
)

// State is the result of Check and Detect. Message is empty when Ready;
// otherwise it says, in English, what is wrong and how to fix it.
type State struct {
	Status  Status
	Reason  Reason
	Message string
	// Vault holds the matched vault's settings (set by Check when the
	// folder is listed).
	Vault config.Vault
	// Lock is the index.lock that paused the vault (ReasonLocked), for the
	// user to remove once sure no git runs (RemoveLeftoverLock).
	Lock string
}

// Check decides whether Misoku may sync folder, the folder it opened. When
// sync is disabled or folder is not listed, it returns Off without running
// git at all; otherwise it runs Detect on the vault, with each git command
// limited to timeout.
func Check(ctx context.Context, vaults []config.Vault, folder string, timeout time.Duration) State {
	if Disabled() {
		return State{Status: Off, Reason: ReasonDisabled, Message: i18n.T(i18n.GitOff)}
	}
	v, ok := Match(vaults, folder)
	if !ok {
		return State{Status: Off, Reason: ReasonNotListed, Message: i18n.T(i18n.GitNotListed)}
	}
	st := Detect(ctx, New(folder, timeout))
	st.Vault = v
	return st
}

// opMarkers are what git leaves in the git dir while an operation waits for
// the user, and how to get out of it. Sync never continues or aborts the
// user's own operation: it pauses until the user does.
var opMarkers = []struct {
	path string
	what i18n.Key
	cmd  string
}{
	{"MERGE_HEAD", i18n.GitOpMerge, "merge"},
	{"rebase-merge", i18n.GitOpRebase, "rebase"},
	{"rebase-apply", i18n.GitOpRebase, "rebase"},
	{"CHERRY_PICK_HEAD", i18n.GitOpCherryPick, "cherry-pick"},
	{"REVERT_HEAD", i18n.GitOpRevert, "revert"},
}

// Detect inspects the repository r runs in and says whether it can sync.
// It only reads (rev-parse, config, remote, symbolic-ref and file checks):
// it never changes the repository, its index or its config, because a
// paused vault must be left exactly as the user had it.
//
// Force-pushes on the remote are not detected here: that needs a fetch.
func Detect(ctx context.Context, r *Runner) State { return detect(ctx, r, true) }

// detect is Detect; needOrigin false skips the check for origin, for
// `misoku sync init`, which adds it.
func detect(ctx context.Context, r *Runner, needOrigin bool) State {
	paused := func(reason Reason, key i18n.Key, args ...any) State {
		return State{Status: Paused, Reason: reason, Message: i18n.T(key, args...)}
	}

	version, ok, err := r.Version(ctx)
	if err != nil {
		// A missing git lands here too (exec cannot find it).
		return paused(ReasonGitMissing, i18n.GitMissing, err, MinVersion)
	}
	if !ok {
		if version == "" {
			version = i18n.T(i18n.GitUnknownVersion)
		}
		return paused(ReasonGitTooOld, i18n.GitTooOld, version, MinVersion)
	}

	// Only git's own answers say how the vault is set up. Any other
	// failure (a timeout, a slow or failing disk) says nothing about it, so
	// it is reported as it is, without repair advice that would be wrong.
	failed := func(err error) State {
		return paused(ReasonGitFailed, i18n.GitCheckFailed, redact(err))
	}

	out, err := r.Run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		if exitCode(err) != 128 || !strings.Contains(err.Error(), "not a git repository") {
			return failed(err)
		}
		return paused(ReasonNotARepo, i18n.GitNotARepoHint, err)
	}
	// D10: the repository must be the vault itself. In a subfolder of a
	// bigger repository, `git add -A` would commit the whole repository.
	top := strings.TrimSpace(string(out))
	if !sameDir(top, r.dir) {
		return paused(ReasonNotRepoRoot, i18n.GitNotRepoRoot, filepath.FromSlash(top))
	}

	if needOrigin {
		out, err = r.Run(ctx, "remote")
		if err != nil {
			return failed(err)
		}
		if !hasLine(string(out), "origin") {
			return paused(ReasonNoRemote, i18n.GitNoRemote)
		}
	}

	if key, err := missingIdentity(ctx, r); err != nil {
		return failed(err)
	} else if key != "" {
		return paused(ReasonNoIdentity, i18n.GitNoIdentity, key)
	}

	// Checked before the detached HEAD: a rebase detaches HEAD, and its
	// fix is to finish or abort it, not to switch branches.
	// --git-path finds each marker wherever git keeps it, including the
	// per-worktree git dir of a linked worktree.
	args := []string{"rev-parse"}
	for _, m := range opMarkers {
		args = append(args, "--git-path", m.path)
	}
	args = append(args, "--git-path", "index.lock", "--git-path", markerName)
	out, err = r.Run(ctx, args...)
	if err != nil {
		return failed(err)
	}
	paths := strings.Split(strings.TrimRight(string(out), "\r\n"), "\n")
	at := func(i int) string {
		if i >= len(paths) {
			return ""
		}
		p := strings.TrimSpace(paths[i])
		if p != "" && !filepath.IsAbs(p) {
			p = filepath.Join(r.dir, p) // relative to the folder git ran in
		}
		return p
	}
	exists := func(p string) bool {
		_, err := os.Lstat(p)
		return p != "" && err == nil
	}
	for i, m := range opMarkers {
		if exists(at(i)) {
			return paused(ReasonInProgress, i18n.GitInProgress, i18n.T(m.what), m.cmd)
		}
	}
	// S13: a lock left by an interrupted sync of Misoku's own has its
	// marker next to it, and SyncVault cleans it up. Any other lock is a
	// git command of the user's, running or crashed: left alone (S2).
	if lock := at(len(opMarkers)); exists(lock) && !exists(at(len(opMarkers)+1)) {
		st := paused(ReasonLocked, i18n.GitLocked, lock)
		st.Lock = lock
		return st
	}
	// symbolic-ref fails when HEAD points at a commit instead of a branch
	// (and works on a branch with no commit yet).
	if _, err := r.Run(ctx, "symbolic-ref", "-q", "HEAD"); err != nil {
		return paused(ReasonDetachedHead, i18n.GitDetached)
	}

	return State{Status: Ready}
}

// missingIdentity returns the first of user.name and user.email git has no
// value for, or "" when both are set.
func missingIdentity(ctx context.Context, r *Runner) (string, error) {
	for _, key := range []string{"user.name", "user.email"} {
		out, err := r.Run(ctx, "config", "--get", key)
		if err != nil && exitCode(err) != 1 { // 1: the key is not set
			return "", err
		}
		if strings.TrimSpace(string(out)) == "" {
			return key, nil
		}
	}
	return "", nil
}

// identityMessage says that key is not set and how to set it.
func identityMessage(key string) string {
	return i18n.T(i18n.GitNoIdentity, key)
}

// hasLine reports whether text has a line equal to want.
func hasLine(text, want string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}
