package gitsync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
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
}

// Check decides whether Misoku may sync folder, the folder it opened. When
// sync is disabled or folder is not listed, it returns Off without running
// git at all; otherwise it runs Detect on the vault, with each git command
// limited to timeout.
func Check(ctx context.Context, vaults []config.Vault, folder string, timeout time.Duration) State {
	if Disabled() {
		return State{Status: Off, Reason: ReasonDisabled, Message: "sync is turned off by MISOKU_SYNC=0"}
	}
	v, ok := Match(vaults, folder)
	if !ok {
		return State{Status: Off, Reason: ReasonNotListed, Message: "this folder is not listed in sync.vaults"}
	}
	st := Detect(ctx, New(folder, timeout))
	st.Vault = v
	return st
}

// opMarkers are what git leaves in the git dir while an operation waits for
// the user, and how to get out of it. Sync never continues or aborts the
// user's own operation: it pauses until the user does.
var opMarkers = []struct{ path, what, cmd string }{
	{"MERGE_HEAD", "a merge", "merge"},
	{"rebase-merge", "a rebase", "rebase"},
	{"rebase-apply", "a rebase", "rebase"},
	{"CHERRY_PICK_HEAD", "a cherry-pick", "cherry-pick"},
	{"REVERT_HEAD", "a revert", "revert"},
}

// Detect inspects the repository r runs in and says whether it can sync.
// It only reads (rev-parse, config, remote, symbolic-ref and file checks):
// it never changes the repository, its index or its config, because a
// paused vault must be left exactly as the user had it.
//
// Force-pushes on the remote are not detected here: that needs a fetch.
func Detect(ctx context.Context, r *Runner) State {
	paused := func(reason Reason, format string, args ...any) State {
		return State{Status: Paused, Reason: reason, Message: fmt.Sprintf(format, args...)}
	}

	version, ok, err := r.Version(ctx)
	if err != nil {
		// A missing git lands here too (exec cannot find it).
		return paused(ReasonGitMissing, "git did not run (%v); install git %s or newer to sync this vault", err, MinVersion)
	}
	if !ok {
		if version == "" {
			version = "of unknown version"
		}
		return paused(ReasonGitTooOld, "git %s is too old; sync needs git %s or newer, update git", version, MinVersion)
	}

	out, err := r.Run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return paused(ReasonNotARepo, "this vault is not a git repository (%v); set it up with `misoku sync init <url>`", err)
	}
	// D10: the repository must be the vault itself. In a subfolder of a
	// bigger repository, `git add -A` would commit the whole repository.
	top := strings.TrimSpace(string(out))
	if !sameDir(top, r.dir) {
		return paused(ReasonNotRepoRoot, "this vault is inside the git repository at %s; sync needs the vault to be the repository's top folder: open %s instead, or give the vault a repository of its own", filepath.FromSlash(top), filepath.FromSlash(top))
	}

	out, err = r.Run(ctx, "remote")
	if err != nil || !hasLine(string(out), "origin") {
		return paused(ReasonNoRemote, "the repository has no remote named origin; add one with `git remote add origin <url>`")
	}

	for _, key := range []string{"user.name", "user.email"} {
		out, err := r.Run(ctx, "config", "--get", key)
		if err != nil || strings.TrimSpace(string(out)) == "" {
			return paused(ReasonNoIdentity, "git does not know who you are (%s is not set); run `git config --global user.name \"Your Name\"` and `git config --global user.email you@example.com`", key)
		}
	}

	// Checked before the detached HEAD: a rebase detaches HEAD, and its
	// fix is to finish or abort it, not to switch branches.
	// --git-path finds each marker wherever git keeps it, including the
	// per-worktree git dir of a linked worktree.
	args := []string{"rev-parse"}
	for _, m := range opMarkers {
		args = append(args, "--git-path", m.path)
	}
	out, err = r.Run(ctx, args...)
	if err != nil {
		return paused(ReasonNotARepo, "git could not read the repository (%v)", err)
	}
	paths := strings.Split(strings.TrimRight(string(out), "\r\n"), "\n")
	for i, m := range opMarkers {
		if i >= len(paths) {
			break
		}
		p := strings.TrimSpace(paths[i])
		if !filepath.IsAbs(p) {
			p = filepath.Join(r.dir, p) // relative to the folder git ran in
		}
		if _, err := os.Lstat(p); err == nil {
			return paused(ReasonInProgress, "%s is in progress in this vault; finish it (`git %s --continue`) or cancel it (`git %s --abort`)", m.what, m.cmd, m.cmd)
		}
	}
	// symbolic-ref fails when HEAD points at a commit instead of a branch
	// (and works on a branch with no commit yet).
	if _, err := r.Run(ctx, "symbolic-ref", "-q", "HEAD"); err != nil {
		return paused(ReasonDetachedHead, "HEAD is detached (no branch is checked out); check out your branch, for example `git switch main`")
	}

	return State{Status: Ready}
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
