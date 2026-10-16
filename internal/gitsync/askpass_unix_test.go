//go:build !windows

package gitsync_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// Only password helpers are skipped: an ssh remote still syncs in the
// background through the user's ssh, which keeps the ssh-agent it is
// given (SSH_AUTH_SOCK) and needs no askpass.
func TestBackgroundGitStillUsesTheUsersSSH(t *testing.T) {
	isolate(t)
	parent := t.TempDir()
	if out, err := exec.Command("git", "-C", parent, "init", "-q", "--bare", "notes.git").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	seen := filepath.Join(t.TempDir(), "agent")
	// A stand-in for ssh: it notes the agent it was given, then runs the
	// git command it was asked to run on the "server" (here, locally).
	ssh := filepath.Join(t.TempDir(), "ssh")
	body := "#!/bin/sh\necho \"$SSH_AUTH_SOCK\" > '" + seen + "'\nfor a; do last=$a; done\nexec sh -c \"$last\"\n"
	if err := os.WriteFile(ssh, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SSH_COMMAND", ssh)
	t.Setenv("SSH_AUTH_SOCK", "/tmp/agent.sock")
	t.Setenv("GIT_ASKPASS", "/bin/false")
	dir := newRepo(t)
	if out, err := exec.Command("git", "-C", dir, "remote", "add", "origin", "ssh://example.invalid"+filepath.ToSlash(filepath.Join(parent, "notes.git"))).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := gitsync.New(dir, 20*time.Second).Run(ctx, "ls-remote", "origin"); err != nil {
		t.Fatalf("background ls-remote over ssh: %v", err)
	}
	if data, err := os.ReadFile(seen); err != nil || string(data) != "/tmp/agent.sock\n" {
		t.Errorf("ssh did not get the user's agent: %q %v", data, err)
	}
}

// When git saves the name but not the email, the assistant says only the
// name was saved, never that nothing changed. A git shim on PATH fails
// `git config --global user.email` alone; the fake gh is a shell script,
// hence Unix only.
func TestSetupSaysOnlyTheNameWasSavedWhenTheEmailFails(t *testing.T) {
	isolate(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	bin := t.TempDir()
	shim := "#!/bin/sh\ncase \"$*\" in *\"--global user.email\"*) echo 'error: could not lock config file' >&2; exit 255 ;; esac\nexec '" + realGit + "' \"$@\"\n"
	for name, body := range map[string]string{"git": shim, "gh": "#!/bin/sh\necho 'gh version 2.62.0'\n"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	global := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(global, []byte("[init]\n\tdefaultBranch = main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	ctx := context.Background()
	s := gitsync.NewSetup(gitsync.SetupOptions{Dir: t.TempDir(), ConfigPath: filepath.Join(t.TempDir(), "config.json"), Timeout: 20 * time.Second})
	s.Begin(ctx)
	if step := s.Next(ctx, gitsync.SetupAnswer{OK: true}); step.Ask != gitsync.AskIdentity {
		t.Fatalf("without an identity setup must ask for it: %+v", step)
	}
	step := s.Next(ctx, gitsync.SetupAnswer{OK: true, Name: "Ana Ruiz", Email: "ana@example.com"})
	text := strings.Join(step.Lines, "\n")
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(text, "Your git name was saved") || !strings.Contains(text, "Nothing else changed") {
		t.Errorf("the message must say only the name was saved:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "nothing was changed") || strings.Contains(text, "name and email were saved") {
		t.Errorf("the message misstates what changed:\n%s", text)
	}
	data, err := os.ReadFile(global)
	if err != nil {
		t.Fatal(err)
	}
	if cfg := string(data); !strings.Contains(cfg, "Ana Ruiz") || strings.Contains(cfg, "ana@example.com") {
		t.Errorf("the global config must hold the name only:\n%s", cfg)
	}
}
