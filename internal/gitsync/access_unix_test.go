//go:build !windows

package gitsync_test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// OpenSSH's real, multi-line messages.
func TestSyncReadsRealOpenSSHMessages(t *testing.T) {
	changed := "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n" +
		"@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @\n" +
		"@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n" +
		"IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!\n" +
		"Host key for example.invalid has changed and you have requested strict checking.\n" +
		"Host key verification failed."
	for name, c := range map[string]struct {
		stderr, want, never string
	}{
		"unknown-batch":  {"No ED25519 host key is known for example.invalid and you have requested strict checking.\nHost key verification failed.", "misoku sync init", ""},
		"changed-full":   {changed, "changed", "accept"},
		"publickey":      {"git@example.invalid: Permission denied (publickey).", "SSH key", ""},
		"gh-https-creds": {"remote: Invalid username or token. Password authentication is not supported for Git operations.\nfatal: Authentication failed for 'https://github.com/o/r.git/'", "gh auth setup-git", ""},
	} {
		t.Run(name, func(t *testing.T) {
			isolate(t)
			dir := clone(t, newRemote(t))
			write(t, dir, "a.md", "a\n")
			fakeSSH(t, dir, c.stderr)
			res := wantDenied(t, dir, c.want)
			if c.never != "" && strings.Contains(res.Err.Error(), c.never) {
				t.Errorf("must not say %q: %v", c.never, res.Err)
			}
			if res.Pending != 1 {
				t.Errorf("the commit stays pending: %+v", res)
			}
		})
	}
}

// the real ssh client, nobody listening: offline, at once.
func TestSyncRealSSHUnreachableIsOffline(t *testing.T) {
	if _, err := exec.LookPath("ssh"); err != nil {
		t.Skip("no ssh")
	}
	isolate(t)
	unsetenv(t, "GIT_SSH")
	unsetenv(t, "GIT_SSH_COMMAND")
	dir := clone(t, newRemote(t))
	write(t, dir, "a.md", "a\n")
	git(t, dir, "remote", "set-url", "origin", "ssh://git@127.0.0.1:1/notes.git")
	start := time.Now()
	res := syncVault(t, dir, "laptop", false)
	t.Logf("%v in %s: %v", res.Outcome, time.Since(start), res.Err)
	wantOutcome(t, res, gitsync.SyncOffline)
	if res.Pending != 1 {
		t.Errorf("pending: %+v", res)
	}
}

// OpenSSH's "Host key for X has changed" line without the
// banner (synthetic: real OpenSSH prints both at the same log level) is
// read as an unknown key, whose hint says to accept it.
func TestSyncChangedKeyWithoutBannerIsChanged(t *testing.T) {
	isolate(t)
	dir := clone(t, newRemote(t))
	fakeSSH(t, dir, "Host key for example.invalid has changed and you have requested strict checking.\nHost key verification failed.")
	res := wantDenied(t, dir, "changed")
	if strings.Contains(res.Err.Error(), "accept") {
		t.Errorf("a changed key must not be accepted blindly: %v", res.Err)
	}
}
