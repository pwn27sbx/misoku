//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Inside init the user is already in a terminal running init: the access
// hints say what to fix, not to run init in a terminal.
func TestSyncInitDeniedHintFitsInit(t *testing.T) {
	needGit(t)
	for _, stderrText := range []string{
		"Host key verification failed.",
		"git@example.invalid: Permission denied (publickey).",
	} {
		ssh := filepath.Join(t.TempDir(), "ssh")
		body := "#!/bin/sh\necho '" + stderrText + "' >&2\nexit 255\n"
		if err := os.WriteFile(ssh, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_SSH_COMMAND", ssh)
		dir := t.TempDir()
		code, _, stderr := initVault(t, dir, "ssh://git@example.invalid/notes.git", filepath.Join(t.TempDir(), "c.json"))
		if code != 1 {
			t.Fatalf("%s: exit %d, want 1\n%s", stderrText, code, stderr)
		}
		if strings.Contains(stderr, "in a terminal") {
			t.Errorf("%s: the hint must fit init:\n%s", stderrText, stderr)
		}
		wantOutput(t, stderr, "then run init again")
	}
}
