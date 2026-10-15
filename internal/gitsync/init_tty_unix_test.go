//go:build !windows

package gitsync_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// sshSeen runs ls-remote on an ssh:// URL through r with a fake ssh that
// notes whether it could open the terminal and the options it got.
func sshSeen(t *testing.T, r *gitsync.Runner) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	ssh := filepath.Join(dir, "ssh")
	body := "#!/bin/sh\nif (exec 3</dev/tty) 2>/dev/null; then t=yes; else t=no; fi\necho \"tty=$t $*\" >> '" + log + "'\nexit 255\n"
	if err := os.WriteFile(ssh, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SSH_COMMAND", ssh)
	_, _ = r.Run(context.Background(), "ls-remote", "ssh://git@example.invalid/x.git")
	got, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

// Run under a terminal (for example through a pty): the background
// runner's ssh can never reach it and runs in BatchMode; init's can.
func TestOnlyInitsSSHReachesTheTerminal(t *testing.T) {
	isolate(t)
	f, err := os.Open("/dev/tty")
	if err != nil {
		t.Skip("no controlling terminal: run this under a pty")
	}
	f.Close()
	repo := newRepo(t)
	if got := sshSeen(t, gitsync.New(repo, 10*time.Second)); !strings.Contains(got, "tty=no") || !strings.Contains(got, "BatchMode=yes") {
		t.Errorf("background ssh must have no terminal and BatchMode; got:\n%s", got)
	}
	if got := sshSeen(t, gitsync.NewInteractive(repo, 10*time.Second)); !strings.Contains(got, "tty=yes") || strings.Contains(got, "BatchMode=yes") {
		t.Errorf("init's ssh must reach the terminal without BatchMode; got:\n%s", got)
	}
}
