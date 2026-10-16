//go:build !windows

// The fake gh below is a shell script: these tests run on macOS and Linux.
// They change PATH and git's global config, so none runs in parallel.

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

// engineFakeGH answers like gh from the files in $FAKE_GH_DIR and logs
// every call in $FAKE_GH_DIR/calls.
//
//   - loggedin: present when signed in; `auth login` creates it, and logs
//     the github.com credential helper git shows it in $FAKE_GH_DIR/helper
//     (gh asks its own question unless that helper is its own) and the
//     value of foo.bar, a setting the user's environment may carry.
//   - user.json: what `api user` prints (one line); without it, it fails.
//
// PATH holds only this gh and git, so the script uses no other program.
const engineFakeGH = `#!/bin/sh
d="$FAKE_GH_DIR"
printf '%s\n' "$*" >> "$d/calls"
case "$1 $2" in
"--version ") echo "gh version 2.97.0 (2026-07-31)" ;;
"auth status") [ -f "$d/loggedin" ] && exit 0; echo "not logged in" >&2; exit 1 ;;
"auth login")
	git config --get "credential.https://github.com.helper" > "$d/helper"
	git config --get foo.bar > "$d/foobar"
	: > "$d/loggedin" ;;
"api user")
	[ -f "$d/user.json" ] || { echo "HTTP 401: Bad credentials" >&2; exit 1; }
	IFS= read -r line < "$d/user.json"; printf '%s\n' "$line" ;;
"repo list") echo "[]" ;;
*) echo "fake gh: unknown command $*" >&2; exit 2 ;;
esac
`

// engineGH is the fake gh's folder.
type engineGH struct{ dir, bin, global string }

// newEngineGH puts the fake gh (and git, nothing else) on PATH, signed in
// or not, with a git global config holding the identity given ("" for
// none: name and email are then both unset).
func newEngineGH(t *testing.T, signedIn bool, identity string) *engineGH {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	f := &engineGH{dir: t.TempDir()}
	f.bin = filepath.Join(f.dir, "bin")
	if err := os.MkdirAll(f.bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.bin, "gh"), []byte(engineFakeGH), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(f.bin, "git")); err != nil {
		t.Fatal(err)
	}
	f.global = filepath.Join(f.dir, "gitconfig")
	if err := os.WriteFile(f.global, []byte("[init]\n\tdefaultBranch = main\n"+identity), 0o644); err != nil {
		t.Fatal(err)
	}
	if signedIn {
		f.write(t, "loggedin", "")
	}
	t.Setenv("FAKE_GH_DIR", f.dir)
	t.Setenv("PATH", f.bin)
	t.Setenv("GIT_CONFIG_GLOBAL", f.global)
	for _, k := range []string{"BROWSER", "GH_BROWSER", "WSL_DISTRO_NAME", "GIT_CONFIG_COUNT"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	return f
}

func (f *engineGH) write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, name), []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func (f *engineGH) read(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.dir, name))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

// engineSetup starts the assistant on a vault with one note, for goos,
// and answers Start.
func engineSetup(t *testing.T, goos string) (*gitsync.Setup, gitsync.SetupStep) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := gitsync.NewSetup(gitsync.SetupOptions{
		Dir: dir, ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Timeout: 30 * time.Second, OS: goos,
	})
	ctx := context.Background()
	if step := s.Begin(ctx); step.Ask != gitsync.AskStart {
		t.Fatalf("begin: %+v", step)
	}
	return s, s.Next(ctx, gitsync.SetupAnswer{OK: true})
}

// signIn answers AskSignIn the way a front end does: it runs the sign-in,
// then goes on.
func signIn(t *testing.T, s *gitsync.Setup, step gitsync.SetupStep) gitsync.SetupStep {
	t.Helper()
	if step.Ask != gitsync.AskSignIn {
		t.Fatalf("want the sign-in, got %+v", step)
	}
	if out, err := s.SignInCommand().CombinedOutput(); err != nil {
		t.Fatalf("sign-in: %v %s", err, out)
	}
	return s.Next(context.Background(), gitsync.SetupAnswer{OK: true})
}

const anaUser = `{"login":"ana","id":583231,"name":"Ana Ruiz","email":null}`

func TestSetupSignsInBeforeAskingForTheIdentity(t *testing.T) {
	f := newEngineGH(t, false, "")
	f.write(t, "user.json", anaUser)
	s, step := engineSetup(t, "darwin")
	if step.Ask != gitsync.AskSignIn {
		t.Fatalf("after Start, without an identity, the sign-in comes first: %+v", step)
	}
	step = signIn(t, s, step)
	if step.Ask != gitsync.AskIdentity {
		t.Fatalf("after signing in, the identity comes next: %+v", step)
	}
	if !strings.Contains(strings.Join(step.Notes, "\n"), "signed in to GitHub") {
		t.Errorf("the identity step must show the sign-in passed: %v", step.Notes)
	}
}

func TestSetupFillsTheMissingIdentityFromGitHub(t *testing.T) {
	for _, c := range []struct {
		name, user, identity, wantName, wantEmail string
	}{
		{"the profile name", anaUser, "", "Ana Ruiz", "583231+ana@users.noreply.github.com"},
		{"no profile name: the login", `{"login":"bea","id":77,"name":""}`, "", "bea", "77+bea@users.noreply.github.com"},
		{"a null profile name: the login", `{"login":"bea","id":77,"name":null}`, "", "bea", "77+bea@users.noreply.github.com"},
		{"git's own name is kept", anaUser, "[user]\n\tname = Mine\n", "Mine", "583231+ana@users.noreply.github.com"},
		{"git's own email is kept", anaUser, "[user]\n\temail = me@example.com\n", "Ana Ruiz", "me@example.com"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newEngineGH(t, true, c.identity)
			f.write(t, "user.json", c.user)
			_, step := engineSetup(t, "darwin")
			if step.Ask != gitsync.AskIdentity {
				t.Fatalf("want the identity, got %+v", step)
			}
			if step.Name != c.wantName || step.Email != c.wantEmail {
				t.Errorf("identity = %q <%s>, want %q <%s>", step.Name, step.Email, c.wantName, c.wantEmail)
			}
			if !step.FromGitHub || !strings.Contains(strings.Join(step.Lines, "\n"), "from your GitHub account") {
				t.Errorf("the step must say the identity comes from GitHub: %+v", step)
			}
		})
	}
}

func TestSetupAsksForTheIdentityAsBeforeWhenGitHubCannotSay(t *testing.T) {
	for name, user := range map[string]string{
		"gh api user fails": "",
		"not JSON":          "<html>",
		"no login":          `{"id":5,"name":"Ana"}`,
		"no id":             `{"login":"ana","name":"Ana"}`,
	} {
		t.Run(name, func(t *testing.T) {
			f := newEngineGH(t, true, "")
			if user != "" {
				f.write(t, "user.json", user)
			}
			_, step := engineSetup(t, "darwin")
			if step.Ask != gitsync.AskIdentity || step.Name != "" || step.Email != "" || step.FromGitHub {
				t.Fatalf("without GitHub's answer the identity is asked empty, as before: %+v", step)
			}
			if text := strings.Join(step.Lines, "\n"); strings.Contains(text, "GitHub account") {
				t.Errorf("nothing came from GitHub, yet the step says so:\n%s", text)
			}
		})
	}
}

func TestSetupSavesTheIdentityTheUserEdited(t *testing.T) {
	f := newEngineGH(t, true, "")
	f.write(t, "user.json", anaUser)
	s, step := engineSetup(t, "darwin")
	if step.Ask != gitsync.AskIdentity {
		t.Fatalf("want the identity, got %+v", step)
	}
	step = s.Next(context.Background(), gitsync.SetupAnswer{OK: true, Name: "Ana R.", Email: "ana@example.com"})
	if step.Ask != gitsync.AskRepo {
		t.Fatalf("after the identity, the repositories: %+v", step)
	}
	cfg, err := os.ReadFile(f.global)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "name = Ana R.") || !strings.Contains(string(cfg), "email = ana@example.com") {
		t.Errorf("the edited identity must be saved:\n%s", cfg)
	}
}

func TestSetupWithAnIdentityGoesStraightToTheRepositories(t *testing.T) {
	f := newEngineGH(t, true, "[user]\n\tname = Test\n\temail = test@example.com\n")
	f.write(t, "user.json", anaUser)
	_, step := engineSetup(t, "darwin")
	if step.Ask != gitsync.AskRepo {
		t.Fatalf("git knows the identity: no identity step: %+v", step)
	}
	if strings.Contains(f.read(t, "calls"), "api user") {
		t.Error("git knows the identity, yet GitHub was asked for one")
	}
}

// gh asks "Authenticate Git with your GitHub credentials?" at sign-in
// unless git's helper for github.com is its own: the sign-in shows it
// gh's, without writing it anywhere, and keeps the user's own settings.
func TestSetupSignInAnswersGHsGitQuestionWithoutChangingGit(t *testing.T) {
	f := newEngineGH(t, false, "[user]\n\tname = Test\n\temail = test@example.com\n")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "foo.bar")
	t.Setenv("GIT_CONFIG_VALUE_0", "kept")
	before, err := os.ReadFile(f.global)
	if err != nil {
		t.Fatal(err)
	}
	s, step := engineSetup(t, "darwin")
	signIn(t, s, step)
	if got := strings.TrimSpace(f.read(t, "helper")); got != "!gh auth git-credential" {
		t.Errorf("gh saw the helper %q, want its own", got)
	}
	if got := strings.TrimSpace(f.read(t, "foobar")); got != "kept" {
		t.Errorf("the user's own GIT_CONFIG_* setting was lost: foo.bar = %q", got)
	}
	if after, err := os.ReadFile(f.global); err != nil || string(after) != string(before) {
		t.Errorf("the sign-in changed git's global config: %v\n%s", err, after)
	}
	if !strings.Contains(f.read(t, "calls"), "auth login --web --git-protocol https --hostname github.com") {
		t.Errorf("the sign-in must answer gh's other questions by its flags: %s", f.read(t, "calls"))
	}
}

func TestSetupSignInSaysWhenNoBrowserOpens(t *testing.T) {
	const hint = "no browser"
	for _, c := range []struct {
		name, goos string
		openers    []string
		env        map[string]string
		want       bool
	}{
		{"Linux without a browser opener", "linux", nil, nil, true},
		{"Linux with xdg-open", "linux", []string{"xdg-open"}, nil, false},
		{"Linux with BROWSER set", "linux", nil, map[string]string{"BROWSER": "lynx"}, false},
		{"Linux with GH_BROWSER set", "linux", nil, map[string]string{"GH_BROWSER": "lynx"}, false},
		{"WSL with only xdg-open", "linux", []string{"xdg-open"}, map[string]string{"WSL_DISTRO_NAME": "Ubuntu"}, true},
		{"WSL with wslview", "linux", []string{"wslview"}, map[string]string{"WSL_DISTRO_NAME": "Ubuntu"}, false},
		{"macOS", "darwin", nil, nil, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newEngineGH(t, false, "")
			for _, o := range c.openers {
				f.write(t, filepath.Join("bin", o), "#!/bin/sh\n")
			}
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			_, step := engineSetup(t, c.goos)
			if step.Ask != gitsync.AskSignIn {
				t.Fatalf("want the sign-in, got %+v", step)
			}
			text := strings.Join(step.Lines, "\n")
			if got := strings.Contains(text, hint); got != c.want {
				t.Errorf("hint shown = %v, want %v:\n%s", got, c.want, text)
			}
			if c.want && !strings.Contains(text, "https://github.com/login/device") {
				t.Errorf("the hint must name the address to open:\n%s", text)
			}
		})
	}
}
