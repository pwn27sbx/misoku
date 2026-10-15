//go:build !windows

package gitsync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// When git cannot read the user's ssh setting, Misoku leaves
// GIT_SSH_COMMAND alone (it would override core.sshCommand). git must
// still be unable to prompt: no controlling terminal, stdin empty, no
// askpass and no terminal prompts.
func TestFinalSSHLookupFailureStillCannotPrompt(t *testing.T) {
	isolate(t)
	remote, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	_ = remote
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	unsetenv(t, "GIT_SSH_COMMAND")
	unsetenv(t, "GIT_SSH")
	unsetenv(t, "GIT_SSH_VARIANT")
	bin := t.TempDir()
	log := filepath.Join(bin, "ssh.log")
	// git that fails only the ssh-setting lookup.
	gitShim := "#!/bin/sh\nfor a; do [ \"$a\" = --get-regexp ] && { echo 'error: simulated' >&2; exit 3; }; done\nexec '" + realGit + "' \"$@\"\n"
	// ssh that records what it gets and tries the terminal, then serves
	// the remote locally.
	sshShim := "#!/bin/sh\n" +
		"echo \"args=$*\" >> '" + log + "'\n" +
		"echo \"askpass=${SSH_ASKPASS_REQUIRE-unset} gsc=${GIT_SSH_COMMAND-unset} prompt=${GIT_TERMINAL_PROMPT-unset}\" >> '" + log + "'\n" +
		"if ( exec 3</dev/tty ) 2>/dev/null; then echo tty=open >> '" + log + "'; else echo tty=none >> '" + log + "'; fi\n" +
		"for last; do :; done\nexec sh -c \"$last\"\n"
	for name, body := range map[string]string{"git": gitShim, "ssh": sshShim} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	git(t, desk, "remote", "set-url", "origin", "ssh://git@fakehost"+remote)

	write(t, desk, "d.md", "desk\n")
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	lock, ok, err := gitsync.TryLock(filepath.Join(t.TempDir(), "sync.lock"))
	if err != nil || !ok {
		t.Fatalf("TryLock: %v %v", ok, err)
	}
	defer lock.Unlock()
	res := gitsync.SyncVault(ctx, gitsync.New(desk, 20*time.Second), lock, gitsync.SyncOptions{Device: "desk"})
	wantOutcome(t, res, gitsync.Synced)
	if time.Since(start) > 15*time.Second {
		t.Errorf("sync took %v", time.Since(start))
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("ssh never ran: %v", err)
	}
	got := string(data)
	for _, want := range []string{"askpass=never", "gsc=unset", "prompt=0", "tty=none"} {
		if !strings.Contains(got, want) {
			t.Errorf("ssh log lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "tty=open") {
		t.Errorf("ssh could open the terminal:\n%s", got)
	}
}
