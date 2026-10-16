//go:build !windows

package gitsync_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// A password helper of the user's (an editor's GIT_ASKPASS, SSH_ASKPASS,
// core.askPass) would open a window or wait for an answer: background git
// never runs it, and fails at once instead.
func TestBackgroundGitNeverRunsTheUsersAskPass(t *testing.T) {
	isolate(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="probe"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()
	asked := filepath.Join(t.TempDir(), "asked")
	helper := func(name string) string {
		p := filepath.Join(t.TempDir(), name)
		body := "#!/bin/sh\necho " + name + " >> '" + asked + "'\necho secret\n"
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	t.Setenv("GIT_ASKPASS", helper("git-askpass"))
	t.Setenv("SSH_ASKPASS", helper("ssh-askpass"))
	dir := newRepo(t)
	for _, args := range [][]string{
		{"config", "core.askPass", helper("core-askpass")},
		{"remote", "add", "origin", srv.URL + "/notes.git"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := gitsync.New(dir, 20*time.Second).Run(ctx, "ls-remote", "origin"); err == nil {
		t.Fatal("ls-remote without access must fail")
	}
	if data, err := os.ReadFile(asked); err == nil {
		t.Errorf("background git ran the user's password helper: %s", data)
	}
}
