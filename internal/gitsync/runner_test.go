package gitsync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// isolate gives git an empty global config and no system config, so the
// user's (or the CI image's) settings never leak into a test.
func isolate(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	global := filepath.Join(dir, "gitconfig")
	cfg := "[user]\n\tname = Test\n\temail = test@example.com\n[init]\n\tdefaultBranch = main\n"
	if err := os.WriteFile(global, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	return dir
}

func TestVersionCheck(t *testing.T) {
	for _, c := range []struct {
		out  string
		ok   bool
		want string
	}{
		{"git version 2.54.0 (Apple Git-157)", true, "2.54"},
		{"git version 2.38.0", true, "2.38"},
		{"git version 2.38.1.windows.1", true, "2.38"},
		{"git version 2.37.9", false, "2.37"},
		{"git version 1.9.5", false, "1.9"},
		{"nonsense", false, ""},
	} {
		got, ok := gitsync.ParseVersion(c.out)
		if ok != c.ok || got != c.want {
			t.Errorf("ParseVersion(%q) = %q, %v; want %q, %v", c.out, got, ok, c.want, c.ok)
		}
	}
}

func TestRunnerNeverPrompts(t *testing.T) {
	isolate(t)
	r := gitsync.New(newRepo(t), 5*time.Second)
	out, err := r.Run(context.Background(), "-c", `alias.env=!env`, "env")
	if err != nil {
		t.Fatal(err)
	}
	env := string(out)
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never", "BatchMode=yes"} {
		if !strings.Contains(env, want) {
			t.Errorf("git must run with %s; env was:\n%s", want, env)
		}
	}
}

func TestBatchModeKeepsTheUsersSSHCommand(t *testing.T) {
	isolate(t)
	repo := newRepo(t)
	if out, err := exec.Command("git", "-C", repo, "config", "core.sshCommand", "ssh -i ~/.ssh/notes_key").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	r := gitsync.New(repo, 5*time.Second)
	out, err := r.Run(context.Background(), "-c", `alias.env=!env`, "env")
	if err != nil {
		t.Fatal(err)
	}
	line := ""
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "GIT_SSH_COMMAND=") {
			line = l
		}
	}
	if !strings.Contains(line, "-i ~/.ssh/notes_key") || !strings.Contains(line, "BatchMode=yes") {
		t.Fatalf("BatchMode must be added to the user's ssh command, got %q", line)
	}
}

func TestHooksAndSigningDoNotRun(t *testing.T) {
	isolate(t)
	if runtime.GOOS == "windows" {
		t.Skip("hook scripts need a POSIX shell")
	}
	repo := newRepo(t)
	marker := filepath.Join(t.TempDir(), "ran")
	hook := filepath.Join(repo, ".git", "hooks", "post-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\ntouch "+marker+"-hook\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	fakeGPG := filepath.Join(t.TempDir(), "gpg")
	if err := os.WriteFile(fakeGPG, []byte("#!/bin/sh\ntouch "+marker+"-gpg\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, kv := range [][2]string{{"commit.gpgsign", "true"}, {"gpg.program", fakeGPG}} {
		if out, err := exec.Command("git", "-C", repo, "config", kv[0], kv[1]).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := gitsync.New(repo, 5*time.Second)
	if _, err := r.Run(context.Background(), "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Commit(context.Background(), "test"); err != nil {
		t.Fatalf("commit must succeed without signing: %v", err)
	}
	for _, suffix := range []string{"-hook", "-gpg"} {
		if _, err := os.Stat(marker + suffix); err == nil {
			t.Errorf("%s ran, but Misoku must never run hooks or the signing program", strings.TrimPrefix(suffix, "-"))
		}
	}
}

func TestRunnerTimesOutAndKillsTheProcess(t *testing.T) {
	isolate(t)
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep from a POSIX shell")
	}
	r := gitsync.New(newRepo(t), 300*time.Millisecond)
	start := time.Now()
	_, err := r.Run(context.Background(), "-c", "alias.hang=!sleep 30", "hang")
	if err == nil {
		t.Fatal("a command over the time limit must fail")
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("timeout took %v: the hung process was not killed", took)
	}
}

func TestRunnerHasNoTerminal(t *testing.T) {
	isolate(t)
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/tty on Windows")
	}
	if f, err := os.Open("/dev/tty"); err != nil {
		t.Skip("the test itself has no terminal, nothing to detach from")
	} else {
		f.Close()
	}
	r := gitsync.New(newRepo(t), 5*time.Second)
	out, _ := r.Run(context.Background(), "-c", `alias.tty=!sh -c "(exec 3</dev/tty) 2>/dev/null && echo HAS || echo NONE"`, "tty")
	if strings.TrimSpace(string(out)) != "NONE" {
		t.Fatalf("git must not reach the terminal, got %q", out)
	}
}

func TestStatusParsesUnusualNames(t *testing.T) {
	isolate(t)
	repo := newRepo(t)
	for _, name := range []string{"ñandú.md", "with space.md", "quote\"d.md"} {
		if runtime.GOOS == "windows" && strings.Contains(name, `"`) {
			continue
		}
		if err := os.WriteFile(filepath.Join(repo, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := gitsync.New(repo, 5*time.Second)
	changes, err := r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range changes {
		got[c.Path] = true
	}
	for _, want := range []string{"ñandú.md", "with space.md"} {
		if !got[want] {
			t.Errorf("status should list %q as is, got %v", want, changes)
		}
	}
}
