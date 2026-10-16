//go:build !windows

// The fake gh is a shell script: this test runs on macOS and Linux only.
// The command-line tests (cmd/misoku) cover each rule of the assistant;
// this one covers what the app adds: dialogs, and the sign-in run outside
// the screen.

package app_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// appFakeGH answers like gh from the files in $FAKE_GH_DIR; see the
// command-line tests for the full one.
const appFakeGH = `#!/bin/sh
d="$FAKE_GH_DIR"
printf '%s\n' "$*" >> "$d/calls"
case "$1 $2" in
"--version ") echo "gh version 2.62.0" ;;
"auth status") [ -f "$d/loggedin" ] && exit 0; echo "not logged in" >&2; exit 1 ;;
"auth login") echo "! First copy your one-time code: 4F2A-91KD"; touch "$d/loggedin" ;;
"auth setup-git") git config --global "credential.https://github.com.helper" "!gh auth git-credential" ;;
"repo list") cat "$d/repos.json" ;;
"repo create") git init -q --bare "$d/remotes/ana/$3.git" && echo "https://github.com/ana/$3" ;;
*) echo "unknown $*" >&2; exit 2 ;;
esac
`

// fakeGitHub puts the fake gh on PATH, with repos as its list and git's
// global config (with or without an identity) pointing https://github.com/
// at the local repositories it returns.
func fakeGitHub(t *testing.T, repos string, identity bool) (fake, remotes string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	fake = t.TempDir()
	remotes = filepath.Join(fake, "remotes")
	bin := filepath.Join(fake, "bin")
	for _, d := range []string{bin, filepath.Join(remotes, "ana")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeOutside(t, bin, "gh", appFakeGH)
	if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOutside(t, fake, "repos.json", repos)
	global := "[init]\n\tdefaultBranch = main\n[url \"file://" + filepath.ToSlash(remotes) + "/\"]\n\tinsteadOf = https://github.com/\n"
	if identity {
		global = "[user]\n\tname = Test\n\temail = test@example.com\n" + global
	}
	writeOutside(t, fake, "gitconfig", global)
	t.Setenv("FAKE_GH_DIR", fake)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(fake, "gitconfig"))
	return fake, remotes
}

func TestSyncSetupInsideTheAppSignsInOutsideTheScreenAndConnects(t *testing.T) {
	_, remotes := fakeGitHub(t, `[{"nameWithOwner":"ana/site","visibility":"PUBLIC","url":"https://github.com/ana/site"}]`, true)

	vault := filepath.Join(t.TempDir(), "notes")
	writeOutside(t, vault, "a.md", "hello")
	cfg := filepath.Join(t.TempDir(), "config.json")
	var outside []string
	h := window(t, vault, cfg, app.WithExec(func(c *exec.Cmd, done tea.ExecCallback) tea.Cmd {
		outside = append(outside, strings.Join(c.Args[1:], " "))
		return func() tea.Msg { return done(c.Run()) }
	}))

	h.send(core.SyncSetupMsg{})
	if !strings.Contains(h.view(), "Sync this vault") {
		t.Fatalf("the assistant must open with what will happen:\n%s", h.view())
	}
	h.key(tea.KeyEnter) // Start
	if !strings.Contains(h.view(), "Sign in to GitHub") {
		t.Fatalf("not signed in: the assistant must offer to sign in:\n%s", h.view())
	}
	h.key(tea.KeyEnter) // Sign in
	if len(outside) != 1 || outside[0] != "auth login --web --git-protocol https --hostname github.com" {
		t.Fatalf("the sign-in must run outside the screen, once: %q", outside)
	}
	view := h.view()
	if !strings.Contains(view, "ana/site") || !strings.Contains(view, "Create a new private repository") {
		t.Fatalf("after signing in, the repositories must be listed:\n%s", view)
	}
	h.key(tea.KeyEnter) // the public one: refused
	wantBar(t, h, "ana/site is public")
	h.key(tea.KeyDown)
	h.key(tea.KeyEnter) // Create a new private repository
	if !strings.Contains(h.view(), "Name of the new private repository") {
		t.Fatalf("creating must ask for the name:\n%s", h.view())
	}
	h.key(tea.KeyEnter) // the folder's name, notes
	text := noSpace(dialogText(h.view()))
	if !strings.Contains(text, noSpace("pick ana/notes")) {
		t.Fatalf("the assistant must end saying what to do on the other computer:\n%s", h.view())
	}
	h.key(tea.KeyEnter) // Close
	if _, err := os.Stat(filepath.Join(remotes, "ana", "notes.git")); err != nil {
		t.Fatalf("the repository was not created: %v", err)
	}
	// The window syncs the vault now, every interval and on :sync.
	writeOutside(t, vault, "b.md", "after setup")
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "Synced")
	if !onRemote(t, filepath.Join(remotes, "ana", "notes.git"), "b.md") {
		t.Error("a sync after setup must push the new note")
	}
}

// The identity questions show their whole hint, not one cut off.
func TestSyncSetupInsideTheAppAsksForTheIdentityWithAWholeHint(t *testing.T) {
	fakeGitHub(t, "[]", false)
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	h.send(core.SyncSetupMsg{})
	h.key(tea.KeyEnter) // Start
	view := h.view()
	if !strings.Contains(view, "Your name, for git") || !strings.Contains(view, "git writes it in your notes' history") {
		t.Fatalf("the name question must show its whole hint:\n%s", view)
	}
}
