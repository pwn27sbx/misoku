package gitsync_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// fakeSSH makes git reach origin through a script standing in for ssh: it
// tries the terminal (there is none), prints stderr and fails, as ssh does
// when it cannot get in. origin moves to an ssh URL nobody can reach.
func fakeSSH(t *testing.T, dir, stderr string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake ssh is a shell script")
	}
	unsetenv(t, "GIT_SSH")
	unsetenv(t, "GIT_SSH_VARIANT")
	script := filepath.Join(t.TempDir(), "fake-ssh")
	body := "#!/bin/sh\ncat /dev/tty >/dev/null 2>&1\nprintf '%s\\n' '" + stderr + "' >&2\nexit 255\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	// "sh" is no ssh client Misoku knows, so the command is left as it is.
	t.Setenv("GIT_SSH_COMMAND", "sh '"+script+"'")
	git(t, dir, "remote", "set-url", "origin", "ssh://git@example.invalid/notes.git")
}

// wantDenied syncs dir through the fake ssh and checks the outcome says
// access was denied, with hint, and that it took no time (no prompt, no
// hang: each git command may take 20 s).
func wantDenied(t *testing.T, dir, hint string) gitsync.SyncResult {
	t.Helper()
	start := time.Now()
	res := syncVault(t, dir, "laptop", false)
	if took := time.Since(start); took > 15*time.Second {
		t.Errorf("a refused login must fail at once, it took %s", took)
	}
	wantOutcome(t, res, gitsync.SyncDenied)
	if !strings.Contains(res.Err.Error(), hint) {
		t.Errorf("the error must give the hint %q: %v", hint, res.Err)
	}
	return res
}

// S8: an access error leaves the commits in the vault, and the next sync
// tries again (nothing is paused or remembered).
func TestSyncUnknownHostKeyIsDeniedWithAHintAndRetried(t *testing.T) {
	isolate(t)
	remote := newRemote(t)
	dir := clone(t, remote)
	write(t, dir, "a.md", "a\n")
	fakeSSH(t, dir, "Host key verification failed.")
	res := wantDenied(t, dir, "misoku sync init")
	if !strings.Contains(res.Err.Error(), "host key") || res.Pending != 1 {
		t.Errorf("want the host key named and 1 pending, got %+v", res)
	}

	git(t, dir, "remote", "set-url", "origin", remote)
	wantOutcome(t, syncVault(t, dir, "laptop", false), gitsync.Synced)
	wantInSync(t, remote, dir, "main")
}

func TestSyncRefusedSSHKeyIsDenied(t *testing.T) {
	isolate(t)
	dir := clone(t, newRemote(t))
	fakeSSH(t, dir, "git@example.invalid: Permission denied (publickey).")
	res := wantDenied(t, dir, "misoku sync init")
	if !strings.Contains(res.Err.Error(), "SSH key") {
		t.Errorf("the error must say the SSH key was refused: %v", res.Err)
	}
}

func TestSyncMissingHTTPSCredentialsAreDenied(t *testing.T) {
	isolate(t)
	dir := clone(t, newRemote(t))
	fakeSSH(t, dir, "fatal: could not read Username for 'https://github.com': terminal prompts disabled")
	wantDenied(t, dir, "gh auth setup-git")
}

// A changed host key may be an attack: the hint must not be to accept it.
func TestSyncChangedHostKeyIsDeniedWithoutAdvisingToAcceptIt(t *testing.T) {
	isolate(t)
	dir := clone(t, newRemote(t))
	fakeSSH(t, dir, "WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!")
	res := wantDenied(t, dir, "changed")
	if strings.Contains(res.Err.Error(), "accept") {
		t.Errorf("a changed key must not be accepted blindly: %v", res.Err)
	}
}

// Not every failure is a login problem: an unknown host is offline (S7).
func TestSyncUnreachableHostIsOffline(t *testing.T) {
	isolate(t)
	dir := clone(t, newRemote(t))
	fakeSSH(t, dir, "ssh: Could not resolve hostname example.invalid: nodename nor servname provided")
	wantOutcome(t, syncVault(t, dir, "laptop", false), gitsync.SyncOffline)
}

// Credentials in a URL git or a helper prints never reach the user.
func TestSyncErrorsNeverShowCredentials(t *testing.T) {
	isolate(t)
	dir := clone(t, newRemote(t))
	fakeSSH(t, dir, "fatal: Authentication failed for 'https://me:s3cret@example.invalid/notes.git/'")
	res := wantDenied(t, dir, "gh auth setup-git")
	if msg := res.Err.Error(); strings.Contains(msg, "s3cret") || strings.Contains(msg, "me:") {
		t.Fatalf("the error shows the credentials: %v", msg)
	}
	if !strings.Contains(res.Err.Error(), "https://example.invalid/notes.git") {
		t.Errorf("the URL itself stays, without its credentials: %v", res.Err)
	}
}

// A push origin turns down for its own rules (a hook, a protected branch, a
// file too large) is not "another device pushed first": retrying would
// loop forever. It is an error that says what origin answered.
func TestSyncPushRefusedByOriginIsAnError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("the remote's hook is a shell script")
	}
	remote, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	hook := filepath.Join(remote, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho 'File big.bin is 120 MB; this exceeds the limit' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, desk, "d.md", "desk\n")
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncFailed)
	if msg := res.Err.Error(); !strings.Contains(msg, "origin refused the push") || !strings.Contains(msg, "120 MB") {
		t.Errorf("the error must say origin refused it and why: %v", msg)
	}
	if res.Pending != 1 {
		t.Errorf("the commit stays pending, got %+v", res)
	}
}

// A file left out for being too big stays untracked; when origin brings a
// file at the same path, `merge --ff-only` refuses on every sync. That must
// say which file to move instead of retrying forever.
func TestSyncNamesTheBigFileThatBlocksOriginsChanges(t *testing.T) {
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	withLimit(t, 10)
	write(t, desk, "big.png", "desk's copy")
	git(t, desk, "add", "big.png")
	git(t, desk, "commit", "-q", "-m", "by hand")
	git(t, desk, "push", "-q", "origin", "main")
	_ = remote

	write(t, laptop, "big.png", "laptop's big file")
	for range 2 {
		res := syncVault(t, laptop, "laptop", false)
		wantOutcome(t, res, gitsync.SyncFailed)
		if msg := res.Err.Error(); !strings.Contains(msg, "big.png") || !strings.Contains(msg, "move or rename") {
			t.Fatalf("the error must name the big file and what to do: %v", msg)
		}
	}
	wantFile(t, laptop, "big.png", "laptop's big file")
}

// `merge --ff-only` failing for another reason than a fresh save (here a
// lock someone else holds) is not "a file changed while syncing", and the
// lock that is not Misoku's is left alone.
func TestSyncReportsAFailedApplyThatIsNotASave(t *testing.T) {
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "l.md", "laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)

	lock := filepath.Join(desk, ".git", "index.lock")
	defer gitsync.SetSyncHooks(func() {
		if err := os.WriteFile(lock, []byte("theirs"), 0o644); err != nil {
			t.Error(err)
		}
	}, nil)()
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncFailed)
	if strings.Contains(res.Err.Error(), "a file changed") || !strings.Contains(res.Err.Error(), "index.lock") {
		t.Errorf("the error must be git's, not a save during the sync: %v", res.Err)
	}
	if b, err := os.ReadFile(lock); err != nil || string(b) != "theirs" {
		t.Fatalf("a lock Misoku did not leave must stay: %q %v", b, err)
	}

	gitsync.SetSyncHooks(nil, nil)
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "l.md", "laptop\n")
	wantInSync(t, remote, desk, "main")
}
