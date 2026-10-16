package gitsync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Attributes is the .gitattributes `misoku sync init` writes: git keeps
// every file byte for byte, so a note saved on Windows and on macOS never
// changes for line endings alone.
const Attributes = "# Misoku: keep every file byte for byte (no line-ending conversion)\n* -text\n"

// InitOptions are the settings of InitVault.
type InitOptions struct {
	// URL is origin's address, exactly as the user gave it.
	URL string
	// Device names this computer in commits (S1).
	Device string
	// Timeout bounds each git command; the network ones may wait for the
	// user to type a password, so it should be generous.
	Timeout time.Duration
	// Now gives the local time used in conflict copy names; nil means
	// time.Now.
	Now func() time.Time
	// NoPrompt keeps every git command from asking, network ones
	// included, as in the background (New): for the setup assistant
	// inside the app, where Misoku owns the terminal and gh already gave
	// git the access.
	NoPrompt bool
}

// InitResult says what InitVault did.
type InitResult struct {
	// CreatedRepo: the folder was not a git repository; `git init` made
	// one, on origin's default branch (main when origin is empty).
	CreatedRepo bool
	// Branch is that branch, origin's default.
	Branch string
	// CreatedFolder: the vault's folder did not exist and was made.
	CreatedFolder bool
	// AddedOrigin: origin was added with the URL.
	AddedOrigin bool
	// WroteAttributes: .gitattributes was written (Attributes).
	WroteAttributes bool
	// KeptAttributes: the vault has a .gitattributes of its own without
	// "* -text"; it was left as it is.
	KeptAttributes bool
	// Files counts the files committed, in every commit of the init.
	Files int
	// AlreadySetUp: the vault was set up for origin before and the init
	// changed nothing.
	AlreadySetUp bool
	// Sync is how the sync that joined the vault and origin ended; its
	// Outcome is Synced when the vault is ready.
	Sync SyncResult
}

// InitVault sets up the folder dir to sync with opts.URL, for
// `misoku sync init` (D8, S10). It runs in the foreground: the network
// steps may ask the user for a password or to accept a new SSH host key
// (NewInteractive).
//
//   - First device (notes in dir, origin empty): `git init -b main`,
//     Misoku's exclusions (D7), .gitattributes with "* -text", the first
//     commit, origin, and a push that sets the upstream.
//   - Another device (dir empty, origin with notes): origin's notes.
//   - Both with notes (origin made with a README too): the one sync that
//     joins unrelated histories, with the S6 rules, so nothing is lost.
//
// A folder already set up for that URL gets a plain sync, and
// AlreadySetUp when nothing changed. It refuses, changing nothing, when
// dir is inside another repository (D10), has another origin, or holds
// something of the user's half done (a merge, a rebase, a lock, a
// detached HEAD). Changes that are only not committed yet are notes like
// any other: the init commits them, as every sync does.
//
// The error is a refusal or a failure before the sync; once origin is set
// up the sync's own end is in Sync. Neither ever holds credentials
// (RedactURL).
func InitVault(ctx context.Context, dir string, opts InitOptions) (InitResult, error) {
	res, err := initVault(ctx, dir, opts)
	return res, redact(err)
}

func initVault(ctx context.Context, dir string, opts InitOptions) (InitResult, error) {
	var res InitResult
	made, err := makeFolder(dir)
	if err != nil {
		return res, err
	}
	res.CreatedFolder = made
	defer func() {
		// A folder made for an init that stopped before `git init` goes
		// again; it is still empty.
		if made && !res.CreatedRepo {
			_ = os.Remove(dir) // best effort: an empty folder left is harmless
			res.CreatedFolder = false
		}
	}()
	r := NewInteractive(dir, opts.Timeout)
	if opts.NoPrompt {
		r = New(dir, opts.Timeout)
	}

	version, ok, err := r.Version(ctx)
	if err != nil {
		return res, fmt.Errorf("git did not run (%w); install git %s or newer", err, MinVersion)
	}
	if !ok {
		return res, fmt.Errorf("git %s is too old; sync needs git %s or newer, update git", version, MinVersion)
	}
	if key, err := missingIdentity(ctx, r); err != nil {
		return res, err
	} else if key != "" {
		return res, errors.New(identityMessage(key))
	}

	isRepo, origin, err := inspectFolder(ctx, r, opts.URL)
	if err != nil {
		return res, err
	}
	attrs := filepath.Join(dir, ".gitattributes")
	setUp := false
	if isRepo && origin != "" {
		up, err := upstreamBranch(ctx, r)
		if err != nil {
			return res, err
		}
		setUp = up != "" && fileExists(attrs)
	}

	if !isRepo {
		if _, err := r.Run(ctx, "init", "--quiet", "--initial-branch=main"); err != nil {
			return res, err
		}
		res.CreatedRepo, res.Branch = true, "main"
	}
	lock, ok, err := LockVault(ctx, r)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, errors.New("another Misoku window is syncing this vault; close it, then run `misoku sync init <url>` again")
	}
	defer lock.Unlock()
	if origin == "" {
		if _, err := r.Run(ctx, "remote", "add", "--", "origin", opts.URL); err != nil {
			return res, err
		}
		res.AddedOrigin = true
	}

	branch, empty, o, err := res.Sync.originBranch(ctx, r)
	if err != nil {
		res.Sync.Outcome, res.Sync.Err = o, redact(err)
		// Nothing came from origin yet: an origin this run added goes
		// again, so a mistyped URL can be fixed by running init again.
		if res.AddedOrigin {
			if _, rerr := r.Run(ctx, "remote", "remove", "origin"); rerr != nil {
				res.Sync.Err = errors.Join(res.Sync.Err, redact(fmt.Errorf("could not remove origin again: %w", rerr)))
			} else {
				res.AddedOrigin = false
			}
		}
		return res, nil
	}
	res.Branch = branch
	if res.CreatedRepo && res.Branch != "main" {
		if _, err := r.Run(ctx, "symbolic-ref", "HEAD", "refs/heads/"+res.Branch); err != nil {
			return res, err
		}
	}
	// The attributes go in the vault's first commit, so git's line-ending
	// conversion (core.autocrlf, Git for Windows' default) never touches a
	// note: ours on the first device, origin's own when it has one (the
	// same bytes on both sides join without a conflict copy). An empty
	// folder commits nothing: it takes origin's notes as they are, and
	// ours only after, when origin had none.
	if !fileExists(attrs) {
		content, write := Attributes, empty
		if !empty {
			if write, err = hasNotes(ctx, r); err != nil {
				return res, err
			}
			if write {
				var o SyncOutcome
				if content, o, err = originAttributes(ctx, r, branch); err != nil {
					res.Sync.Outcome, res.Sync.Err = o, redact(err)
					return res, nil
				}
			}
		}
		if write {
			if res.WroteAttributes, err = writeAttributes(attrs, content); err != nil {
				return res, err
			}
		}
	}

	sopts := SyncOptions{Device: opts.Device, AllowUnrelatedHistories: true, Now: opts.Now}
	res.Sync = SyncVault(ctx, r, lock, sopts)
	res.Files = res.Sync.Commit.Files
	if res.Sync.Outcome == Synced && !fileExists(attrs) {
		if res.WroteAttributes, err = writeAttributes(attrs, Attributes); err != nil {
			return res, err
		}
		first := res.Sync
		res.Sync = SyncVault(ctx, r, lock, sopts)
		res.Files += res.Sync.Commit.Files
		res.Sync.Updated = res.Sync.Updated || first.Updated
		res.Sync.Pushed = res.Sync.Pushed || first.Pushed
		res.Sync.Conflicts = append(first.Conflicts, res.Sync.Conflicts...)
		res.Sync.Commit.Skipped = append(first.Commit.Skipped, res.Sync.Commit.Skipped...)
	}
	if !res.WroteAttributes {
		data, err := os.ReadFile(attrs)
		res.KeptAttributes = err == nil && !hasLine(string(data), "* -text")
	}
	res.AlreadySetUp = setUp && res.Sync.Outcome == Synced && res.Files == 0 && !res.Sync.Updated && !res.Sync.Pushed
	return res, nil
}

// inspectFolder says whether the folder r runs in is a repository of its
// own and returns its origin's URL ("" when it has none). It refuses, with
// the reason and the fix, a folder init must not touch (folderState) and
// one whose origin is not url.
func inspectFolder(ctx context.Context, r *Runner, url string) (isRepo bool, origin string, err error) {
	isRepo, origin, err = folderState(ctx, r)
	if err != nil || origin == "" {
		return isRepo, origin, err
	}
	if origin != url {
		return false, "", fmt.Errorf("this repository already has origin %s, not %s; init changed nothing. To sync with %s, run `git remote set-url origin <url>` first, or choose another folder", RedactURL(origin), RedactURL(url), RedactURL(url))
	}
	return true, origin, nil
}

// folderState says whether the folder r runs in is a repository of its
// own and returns its origin's URL as configured ("" when it has none),
// changing nothing. It refuses, with the reason and the fix, a folder
// that must not be set up: one inside another repository (D10), and one
// holding something of the user's half done (see Detect).
func folderState(ctx context.Context, r *Runner) (isRepo bool, origin string, err error) {
	if _, err := r.Run(ctx, "rev-parse", "--show-toplevel"); err != nil {
		if exitCode(err) == 128 && strings.Contains(err.Error(), "not a git repository") {
			return false, "", nil
		}
		return false, "", err
	}
	st := detect(ctx, r, false)
	if st.Reason == ReasonNotRepoRoot {
		// D10, in init's words: Detect tells the app to open the outer
		// repository instead, which is not what setting up a vault needs.
		top, err := r.Run(ctx, "rev-parse", "--show-toplevel")
		if err != nil {
			return false, "", err
		}
		return false, "", fmt.Errorf("this folder is inside the git repository at %s, and syncing it would commit that whole repository; init changed nothing. Give the vault a repository of its own: move it outside %s, or run `git init` in this folder (the outer repository then sees it as a nested one), then run init again", filepath.FromSlash(strings.TrimSpace(string(top))), filepath.FromSlash(strings.TrimSpace(string(top))))
	}
	if st.Status != Ready {
		return false, "", fmt.Errorf("%s; then run `misoku sync init <url>` again", st.Message)
	}
	out, err := r.Run(ctx, "remote")
	if err != nil {
		return false, "", err
	}
	if !hasLine(string(out), "origin") {
		return true, "", nil
	}
	// The URL as configured, not as url.<base>.insteadOf rewrites it
	// (`remote get-url` would): that is what the user typed last time.
	out, err = r.Run(ctx, "config", "--get", "remote.origin.url")
	if err != nil {
		return false, "", err
	}
	return true, strings.TrimSpace(string(out)), nil
}

// makeFolder creates dir when it is missing but its parent folder exists
// (a new device's empty vault), and reports whether it did.
func makeFolder(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return false, fmt.Errorf("%s is not a folder", dir)
		}
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	parent := filepath.Dir(dir)
	if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		return false, fmt.Errorf("%s does not exist, nor does its folder %s; create it first, or choose another folder", dir, parent)
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		return false, err
	}
	return true, nil
}

// hasNotes reports whether the vault has anything to commit or already
// committed: anything but Misoku's own files.
func hasNotes(ctx context.Context, r *Runner) (bool, error) {
	if head, err := resolve(ctx, r, "HEAD"); err != nil || head != "" {
		return head != "", err
	}
	changes, err := r.Status(ctx)
	if err != nil {
		return false, err
	}
	for _, c := range changes {
		if !ownFile(c.Path) {
			return true, nil
		}
	}
	return false, nil
}

// originAttributes fetches origin's branch and returns its .gitattributes,
// or Attributes when it has none. A failed fetch comes with the outcome
// to report.
func originAttributes(ctx context.Context, r *Runner, branch string) (string, SyncOutcome, error) {
	tracking := "refs/remotes/origin/" + branch
	if _, err := r.Run(ctx, "fetch", "--quiet", "--no-tags", "origin", "+refs/heads/"+branch+":"+tracking); err != nil {
		o, err := remoteFailure(r, "could not fetch from origin", err)
		return "", o, err
	}
	out, err := r.Run(ctx, "cat-file", "-e", tracking+":.gitattributes")
	if err != nil {
		if exitCode(err) == 128 || exitCode(err) == 1 {
			return Attributes, Synced, nil
		}
		return "", SyncFailed, err
	}
	if out, err = r.Run(ctx, "cat-file", "blob", tracking+":.gitattributes"); err != nil {
		return "", SyncFailed, err
	}
	return string(out), Synced, nil
}

// writeAttributes creates the file at p with content, unless something
// is there already (a file of the user's is never replaced).
func writeAttributes(p, content string) (bool, error) {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = f.WriteString(content)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err == nil, err
}

// Report says what the init did, one line per step, for
// `misoku sync init` and the setup assistant. url is origin's address
// already without credentials (RedactURL).
func (res InitResult) Report(url string) []string {
	var lines []string
	if res.CreatedRepo {
		lines = append(lines, fmt.Sprintf("Created a git repository (branch %s).", res.Branch))
	}
	if res.AddedOrigin {
		lines = append(lines, fmt.Sprintf("Added origin %s.", url))
	}
	if res.WroteAttributes {
		lines = append(lines, "Wrote .gitattributes (* -text): files stay byte for byte, with no line-ending conversion.")
	}
	if res.KeptAttributes {
		lines = append(lines, "Kept your .gitattributes as it is; add the line `* -text` to it so line endings never change between systems.")
	}
	for _, big := range res.Sync.Commit.Skipped {
		lines = append(lines, fmt.Sprintf("Left out %s (%d MB): files that big do not sync.", big.Path, big.Size>>20))
	}
	// The files that do not sync as plain files (S15), in the words
	// `:sync status` uses.
	for _, n := range res.Sync.Commit.Notices {
		lines = append(lines, n.String())
	}
	if res.Files > 0 {
		noun := "files"
		if res.Files == 1 {
			noun = "file"
		}
		lines = append(lines, fmt.Sprintf("Committed %d %s.", res.Files, noun))
	}
	if res.Sync.Updated {
		lines = append(lines, "Brought origin's notes.")
	}
	if n := len(res.Sync.Conflicts); n > 0 {
		lines = append(lines, fmt.Sprintf("Kept both versions of %d note(s) changed on both sides; the other version is in:", n))
		for _, c := range res.Sync.Conflicts {
			lines = append(lines, "  "+c)
		}
		lines = append(lines, "Resolve each by deleting the copy (or copying what you want into the note).")
	}
	if res.Sync.Pushed {
		lines = append(lines, fmt.Sprintf("Pushed to origin (branch %s).", res.Sync.Branch))
	}
	return lines
}
