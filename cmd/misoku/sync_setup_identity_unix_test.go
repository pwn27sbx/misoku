//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The identity comes after the sign-in, filled in from the
// GitHub account only where git lacks it, and the sign-in neither writes
// git config nor drops the user's own GIT_CONFIG_* settings.

// envProbe makes the fake gh's `auth login` record, before answering, what
// git sees in its environment: the GIT_CONFIG_* variables and the helper
// git reads for github.com.
func envProbe(t *testing.T, f *fakeGH) {
	t.Helper()
	script := strings.Replace(fakeGHScript, `"auth login")`,
		`"auth login")
	env | grep -E '^GIT_CONFIG_(COUNT|KEY_|VALUE_)' | sort > "$d/loginenv"
	git config --get "credential.https://github.com.helper" > "$d/loginhelper"
	git config --get probe.one >> "$d/loginhelper"
	git config --get probe.two >> "$d/loginhelper"`, 1)
	if err := os.WriteFile(filepath.Join(f.dir, "bin", "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readProbe(t *testing.T, f *fakeGH, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// A new user without identity: signs in, accepts what GitHub offers, and
// the first commit carries that identity.
func TestSyncSetupCommitsCarryTheGitHubIdentity(t *testing.T) {
	f := newFakeGH(t, false)
	f.writeGlobal(t, false)
	writeFile(t, f.dir, "login", "ok")
	writeFile(t, f.dir, "user.json", anaUser)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n\n\n\ny\n1\n\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if got := git(t, dir, "log", "-1", "--format=%an <%ae>"); got != "Ana Ruiz <583231+ana@users.noreply.github.com>" {
		t.Errorf("commit identity = %q", got)
	}
}

// An identity already in git's global settings is never touched, and
// GitHub is not asked for one.
func TestSyncSetupGlobalIdentityIsLeftAlone(t *testing.T) {
	f := newFakeGH(t, false)
	writeFile(t, f.dir, "login", "ok")
	writeFile(t, f.dir, "user.json", anaUser)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n\n1\n\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if strings.Contains(out, "Your name") || f.called(t, "api user") {
		t.Errorf("asked for an identity git already has:\n%s", out)
	}
	if got := git(t, dir, "config", "--global", "user.name"); got != "Test" {
		t.Errorf("user.name = %q, want Test", got)
	}
}

// An identity set in the vault's own repository counts as known.
func TestSyncSetupRepoIdentityIsLeftAlone(t *testing.T) {
	f := newFakeGH(t, true)
	f.writeGlobal(t, false)
	writeFile(t, f.dir, "user.json", anaUser)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.name", "Local")
	git(t, dir, "config", "user.email", "local@example.com")
	globalBefore := fileHash(t, f.global)
	code, out := setup(t, dir, cfg, "\n\n")
	if code != 1 {
		t.Errorf("exit %d, want 1 (cancelled at the repository):\n%s", code, out)
	}
	if strings.Contains(out, "Your name") {
		t.Errorf("asked for an identity the repository has:\n%s", out)
	}
	if fileHash(t, f.global) != globalBefore {
		t.Error("git's global config changed")
	}
}

// Only the missing field is filled in from GitHub; the other is kept.
func TestSyncSetupOnlyTheMissingFieldIsFilled(t *testing.T) {
	for _, c := range []struct{ name, key, val, wantName, wantEmail string }{
		{"only name missing", "user.email", "own@example.com", "Ana Ruiz", "own@example.com"},
		{"only email missing", "user.name", "Own Name", "Own Name", "583231+ana@users.noreply.github.com"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFakeGH(t, true)
			f.writeGlobal(t, false)
			writeFile(t, f.dir, "user.json", anaUser)
			dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
			git(t, dir, "config", "--global", c.key, c.val)
			code, out := setup(t, dir, cfg, "\n\n\ny\n\n")
			if code != 1 {
				t.Errorf("exit %d, want 1:\n%s", code, out)
			}
			wantOutput(t, out, "Your name ["+c.wantName+"]:", "Your email ["+c.wantEmail+"]:")
			if got := git(t, dir, "config", "--global", "user.name"); got != c.wantName {
				t.Errorf("user.name = %q, want %q", got, c.wantName)
			}
			if got := git(t, dir, "config", "--global", "user.email"); got != c.wantEmail {
				t.Errorf("user.email = %q, want %q", got, c.wantEmail)
			}
		})
	}
}

// When gh cannot tell (failure, no id, no login, bad JSON), the identity is
// asked empty, as before; a null name falls back to the login.
func TestSyncSetupGitHubIdentityFallbacks(t *testing.T) {
	for _, c := range []struct{ name, user, wantPrompt string }{
		{"api fails", "", "Your name: "},
		{"no id", `{"login":"ana","name":"Ana"}`, "Your name: "},
		{"zero id", `{"login":"ana","id":0,"name":"Ana"}`, "Your name: "},
		{"no login", `{"id":5,"name":"Ana"}`, "Your name: "},
		{"bad json", `<html>`, "Your name: "},
		{"null name", `{"login":"ana","id":5,"name":null}`, "Your name [ana]:"},
		{"blank name", `{"login":"ana","id":5,"name":"  "}`, "Your name [ana]:"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFakeGH(t, true)
			f.writeGlobal(t, false)
			if c.user != "" {
				writeFile(t, f.dir, "user.json", c.user)
			}
			dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
			before := fileHash(t, f.global)
			// Start, then Enter at the name: cancels when empty.
			code, out := setup(t, dir, cfg, "\n\n\n\n")
			if code != 1 {
				t.Errorf("exit %d, want 1:\n%s", code, out)
			}
			wantOutput(t, out, c.wantPrompt)
			if strings.Contains(out, "HTTP 401") {
				t.Errorf("gh's failure leaked into the output:\n%s", out)
			}
			if fileHash(t, f.global) != before {
				t.Error("git's global config changed without a confirmed identity")
			}
		})
	}
}

// Cancelling at the sign-in, or at the identity after it, changes nothing.
func TestSyncSetupCancelChangesNothing(t *testing.T) {
	for _, c := range []struct{ name, login, input string }{
		{"declined sign-in", "", "\nn\n"},
		{"failed sign-in", "no", "\n\n"},
		{"declined save of identity", "ok", "\n\n\n\nn\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFakeGH(t, false)
			f.writeGlobal(t, false)
			if c.login != "" {
				writeFile(t, f.dir, "login", c.login)
			}
			writeFile(t, f.dir, "user.json", anaUser)
			dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
			before := f.snapshot(t, dir, cfg)
			code, out := setup(t, dir, cfg, c.input)
			if code != 1 {
				t.Errorf("exit %d, want 1:\n%s", code, out)
			}
			if after := f.snapshot(t, dir, cfg); after != before {
				t.Errorf("changed the vault or a config:\n%s", out)
			}
		})
	}
}

// The sign-in sees gh's helper for github.com through the environment
// only, after the user's own GIT_CONFIG_* pairs, which still apply; git's
// global config is not written by it, and setup-git still writes the
// helper later.
func TestSyncSetupSignInKeepsTheUsersGitConfigEnv(t *testing.T) {
	f := newFakeGH(t, false)
	envProbe(t, f)
	writeFile(t, f.dir, "login", "ok")
	t.Setenv("GIT_CONFIG_COUNT", "2")
	t.Setenv("GIT_CONFIG_KEY_0", "probe.one")
	t.Setenv("GIT_CONFIG_VALUE_0", "first")
	t.Setenv("GIT_CONFIG_KEY_1", "probe.two")
	t.Setenv("GIT_CONFIG_VALUE_1", "second")
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	globalBefore := fileHash(t, f.global)
	// Start, sign in, cancel at the repository.
	code, out := setup(t, dir, cfg, "\n\n\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantEnv := "GIT_CONFIG_COUNT=3\nGIT_CONFIG_KEY_0=probe.one\nGIT_CONFIG_KEY_1=probe.two\n" +
		"GIT_CONFIG_KEY_2=credential.https://github.com.helper\nGIT_CONFIG_VALUE_0=first\n" +
		"GIT_CONFIG_VALUE_1=second\nGIT_CONFIG_VALUE_2=!gh auth git-credential\n"
	if got := readProbe(t, f, "loginenv"); got != wantEnv {
		t.Errorf("auth login env:\n%s\nwant:\n%s", got, wantEnv)
	}
	if got := readProbe(t, f, "loginhelper"); got != "!gh auth git-credential\nfirst\nsecond\n" {
		t.Errorf("git inside auth login read %q", got)
	}
	if fileHash(t, f.global) != globalBefore {
		t.Error("the sign-in wrote git's global config")
	}
	if os.Getenv("GIT_CONFIG_COUNT") != "2" {
		t.Error("the process's own environment changed")
	}
	// Now all the way: setup-git still configures the helper.
	code, out = setup(t, dir, cfg, "\n1\n\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if got := git(t, dir, "config", "--global", "credential.https://github.com.helper"); got != "!gh auth git-credential" {
		t.Errorf("setup-git helper = %q", got)
	}
}
