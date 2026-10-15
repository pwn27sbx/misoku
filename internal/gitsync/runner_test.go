package gitsync_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// testIsolationEnv carries the folder TestMain made to a test binary that a
// test starts again as a helper, so the helper reuses it instead of making
// (and, when it is killed, leaving behind) one of its own.
const testIsolationEnv = "MISOKU_GITSYNC_TEST_ISOLATION"

// TestMain gives git an empty global config and no system config, so the
// user's (or the CI image's) settings never leak into a test. It is the same
// for every test, so it is set once here with os.Setenv: t.Setenv would
// forbid t.Parallel, and the tests run in parallel.
func TestMain(m *testing.M) {
	os.Exit(runIsolated(m))
}

func runIsolated(m *testing.M) int {
	if os.Getenv(testIsolationEnv) == "" {
		dir, err := os.MkdirTemp("", "misoku-gitsync-test-")
		if err != nil {
			fmt.Fprintln(os.Stderr, "TestMain:", err)
			return 1
		}
		defer os.RemoveAll(dir)
		global := filepath.Join(dir, "gitconfig")
		cfg := "[user]\n\tname = Test\n\temail = test@example.com\n[init]\n\tdefaultBranch = main\n"
		if err := os.WriteFile(global, []byte(cfg), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "TestMain:", err)
			return 1
		}
		os.Setenv(testIsolationEnv, dir)
		os.Setenv("GIT_CONFIG_GLOBAL", global)
		os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	}
	return m.Run()
}

// isolate skips a test that needs git when git is not installed. The
// configuration git runs with is set for the whole package in TestMain.
func isolate(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// sshCommandOf runs git through r and returns the GIT_SSH_COMMAND it saw,
// and whether it was set at all.
func sshCommandOf(t *testing.T, r *gitsync.Runner) (string, bool) {
	t.Helper()
	out, err := r.Run(context.Background(), "-c", `alias.env=!env`, "env")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimRight(l, "\r"), "GIT_SSH_COMMAND="); ok {
			return v, true
		}
	}
	return "", false
}

// unsetenv removes k for the rest of the test and restores it afterwards.
func unsetenv(t *testing.T, k string) {
	t.Setenv(k, "")
	os.Unsetenv(k)
}

// git prefers GIT_SSH_COMMAND over GIT_SSH, so setting the first must not
// silently throw away a program the user configured with the second, and
// the no-prompt flag must be one that program understands.
func TestBatchModeKeepsTheUsersSSHSetting(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "my tools")
	for _, c := range []struct {
		name, gitSSH, gitSSHCommand, coreSSHCommand string
		want                                        string // "" means GIT_SSH_COMMAND must be left alone
	}{
		{name: "plain ssh", want: "ssh -o BatchMode=yes"},
		{name: "GIT_SSH with OpenSSH", gitSSH: bin + "/ssh", want: "'" + bin + "/ssh' -o BatchMode=yes"},
		{name: "GIT_SSH with plink", gitSSH: bin + "/plink.exe", want: "'" + bin + "/plink.exe' -batch"},
		{name: "GIT_SSH with TortoisePlink", gitSSH: bin + "/TortoisePlink.exe", want: "'" + bin + "/TortoisePlink.exe'"},
		{name: "BatchMode=no in the user's command", gitSSHCommand: "ssh -o BatchMode=no -p 2222", want: "ssh -o BatchMode=yes -o BatchMode=no -p 2222"},
		{name: "quoted program with spaces", coreSSHCommand: `"` + bin + `/ssh" -i key`, want: `"` + bin + `/ssh" -o BatchMode=yes -i key`},
		{name: "core.sshCommand beats GIT_SSH", gitSSH: bin + "/plink", coreSSHCommand: "ssh -i key", want: "ssh -o BatchMode=yes -i key"},
		{name: "wrapper script", coreSSHCommand: bin + "/ssh-wrapper --flag", want: ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			isolate(t)
			unsetenv(t, "GIT_SSH")
			unsetenv(t, "GIT_SSH_COMMAND")
			unsetenv(t, "GIT_SSH_VARIANT")
			if c.gitSSH != "" {
				t.Setenv("GIT_SSH", c.gitSSH)
			}
			if c.gitSSHCommand != "" {
				t.Setenv("GIT_SSH_COMMAND", c.gitSSHCommand)
			}
			repo := newRepo(t)
			if c.coreSSHCommand != "" {
				if out, err := exec.Command("git", "-C", repo, "config", "core.sshCommand", c.coreSSHCommand).CombinedOutput(); err != nil {
					t.Fatalf("%v %s", err, out)
				}
			}
			got, set := sshCommandOf(t, gitsync.New(repo, 5*time.Second))
			if c.want == "" {
				if set && strings.Contains(got, "BatchMode") {
					t.Fatalf("a wrapper of unknown kind must not get ssh options, got %q", got)
				}
				return
			}
			if got != c.want {
				t.Fatalf("GIT_SSH_COMMAND = %q, want %q", got, c.want)
			}
		})
	}
}

// ssh.variant tells git (and Misoku) what kind of client a wrapper is.
func TestSSHVariantConfigDecidesTheFlag(t *testing.T) {
	isolate(t)
	unsetenv(t, "GIT_SSH")
	unsetenv(t, "GIT_SSH_COMMAND")
	unsetenv(t, "GIT_SSH_VARIANT")
	repo := newRepo(t)
	for _, kv := range [][2]string{{"core.sshCommand", "my-ssh -i key"}, {"ssh.variant", "ssh"}} {
		if out, err := exec.Command("git", "-C", repo, "config", kv[0], kv[1]).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	if got, _ := sshCommandOf(t, gitsync.New(repo, 5*time.Second)); got != "my-ssh -o BatchMode=yes -i key" {
		t.Fatalf("GIT_SSH_COMMAND = %q", got)
	}
}

// A failed read of the user's ssh setting (git could not run, a time
// limit on a loaded machine) is not taken as "nothing configured": that
// would set GIT_SSH_COMMAND to plain ssh, which overrides core.sshCommand,
// and keep it for the Runner's whole life. The command runs with the
// user's setting left alone, and the next one reads it again.
func TestFailedSSHLookupIsNotRemembered(t *testing.T) {
	isolate(t)
	unsetenv(t, "GIT_SSH")
	unsetenv(t, "GIT_SSH_COMMAND")
	unsetenv(t, "GIT_SSH_VARIANT")
	dir := filepath.Join(t.TempDir(), "later")
	r := gitsync.New(dir, 5*time.Second)
	if got, ok := envValue(r.EnvFor(context.Background(), "fetch"), "GIT_SSH_COMMAND"); ok {
		t.Fatalf("a failed lookup must leave GIT_SSH_COMMAND unset, got %q", got)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"config", "core.sshCommand", "ssh -i key"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	if got, _ := envValue(r.EnvFor(context.Background(), "fetch"), "GIT_SSH_COMMAND"); got != "ssh -o BatchMode=yes -i key" {
		t.Fatalf("the next command must read the setting again, got %q", got)
	}
}

// envValue returns the value of k in env.
func envValue(env []string, k string) (string, bool) {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, k+"="); ok {
			return v, true
		}
	}
	return "", false
}

// A rename is two fields in -z output: the new path, then the original.
// The original must not come back as an entry of its own, and the entries
// after it must still parse.
func TestStatusReportsRenames(t *testing.T) {
	t.Parallel()
	isolate(t)
	repo := newRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "old name.md"), []byte("same content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "init"}, {"mv", "old name.md", "new name.md"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "zz.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	changes, err := gitsync.New(repo, 5*time.Second).Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []gitsync.Change{{Code: "R ", Path: "new name.md", From: "old name.md"}, {Code: "??", Path: "zz.md"}}
	if len(changes) != len(want) {
		t.Fatalf("status = %q, want %q", changes, want)
	}
	for i := range want {
		if changes[i] != want[i] {
			t.Fatalf("status = %q, want %q", changes, want)
		}
	}
}

// What git printed before hanging (an ssh error, a host-key message) is
// the only clue why it hung, so the timeout error keeps it.
func TestTimeoutErrorKeepsGitsStderr(t *testing.T) {
	t.Parallel()
	isolate(t)
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep from a POSIX shell")
	}
	r := gitsync.New(newRepo(t), time.Second)
	_, err := r.Run(context.Background(), "-c", "alias.hang=!echo ssh: connect to host example: stuck >&2; sleep 30", "hang")
	if err == nil {
		t.Fatal("expected a timeout")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no answer after 1s") || !strings.Contains(msg, "ssh: connect to host example: stuck") {
		t.Fatalf("timeout error must name the limit and keep git's stderr, got %q", msg)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a timeout must match context.DeadlineExceeded, got %v", err)
	}
}

// When the caller gives up (it cancelled, or its own shorter deadline
// passed), the error must say so instead of blaming the Runner's limit.
func TestCallerStopIsNotReportedAsOurTimeout(t *testing.T) {
	t.Parallel()
	isolate(t)
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep from a POSIX shell")
	}
	r := gitsync.New(newRepo(t), time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	_, err := r.Run(ctx, "-c", "alias.hang=!sleep 30", "hang")
	if err == nil || !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "cancelled") || strings.Contains(err.Error(), "no answer") {
		t.Fatalf("a cancelled caller must be reported as cancellation, got %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err = r.Run(ctx, "-c", "alias.hang=!sleep 30", "hang")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "1m0s") {
		t.Fatalf("the caller's own deadline must not be reported as the Runner's 1m limit, got %v", err)
	}
}
