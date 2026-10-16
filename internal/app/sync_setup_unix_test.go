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
"api user") [ -f "$d/user.json" ] && { cat "$d/user.json"; exit 0; }; echo "HTTP 401" >&2; exit 1 ;;
"repo list") [ -f "$d/listfail" ] && { echo "HTTP 502: bad gateway" >&2; exit 1; }; cat "$d/repos.json" ;;
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

	if !strings.Contains(h.view(), "Sync between computers") {
		t.Fatalf("before setup the home screen must offer sync:\n%s", h.view())
	}
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
	if view := h.view(); !strings.Contains(view, "New Note") || strings.Contains(view, "Sync between computers") {
		t.Errorf("once set up, the home screen must no longer offer sync:\n%s", view)
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
	fake, _ := fakeGitHub(t, "[]", false)
	writeOutside(t, fake, "loggedin", "") // the identity comes after the sign-in
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

// saveIdentityInTheApp starts the assistant, signed in and with no git
// identity, and answers its questions up to the Save button, which it
// returns pressed: the command that saves the identity, not run yet. It
// returns the fake gh's folder too.
func saveIdentityInTheApp(t *testing.T) (*harness, tea.Cmd, string) {
	t.Helper()
	fake, _ := fakeGitHub(t, "[]", false)
	writeOutside(t, fake, "loggedin", "") // the identity comes after the sign-in
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	h.send(core.SyncSetupMsg{})
	h.key(tea.KeyEnter) // Start
	h.keys("Ana Ruiz")
	h.key(tea.KeyEnter)
	h.keys("ana@example.com")
	h.key(tea.KeyEnter)
	if !strings.Contains(h.view(), "Save your git identity") {
		t.Fatalf("the identity must be confirmed:\n%s", h.view())
	}
	return h, h.update(tea.KeyMsg{Type: tea.KeyEnter}), fake // Save
}

// wantChangesSaid checks the bar says the identity was saved, and never
// that nothing changed.
func wantChangesSaid(t *testing.T, h *harness) {
	t.Helper()
	bar := statusBar(h)
	if !strings.Contains(bar, "Your git name and email were saved") || !strings.Contains(bar, "Nothing else changed") {
		t.Errorf("the bar must say what changed:\n%s", bar)
	}
	if strings.Contains(strings.ToLower(bar), "nothing was changed") {
		t.Errorf("the identity was saved, yet the bar says nothing changed:\n%s", bar)
	}
}

// The identity is the last question before the repositories: a setup
// that stops there says the identity was saved.
func TestSyncSetupStoppedInTheAppAfterSavingTheIdentitySaysWhatChanged(t *testing.T) {
	h, save, fake := saveIdentityInTheApp(t)
	writeOutside(t, fake, "listfail", "")
	h.finish(save)
	text := noSpace(dialogText(h.view()))
	for _, want := range []string{"gh could not list your repositories", "Your git name and email were saved", "Nothing else changed"} {
		if !strings.Contains(text, noSpace(want)) {
			t.Errorf("the end must say %q:\n%s", want, h.view())
		}
	}
	if strings.Contains(strings.ToLower(text), noSpace("nothing was changed")) {
		t.Errorf("the identity was saved, yet the end says nothing changed:\n%s", h.view())
	}
}

// After saving the identity, the repositories come next.
func TestSyncSetupInTheAppListsTheRepositoriesAfterTheIdentity(t *testing.T) {
	h, save, _ := saveIdentityInTheApp(t)
	h.finish(save)
	if !strings.Contains(h.view(), "Create a new private repository") {
		t.Fatalf("after the identity, the repositories must be listed:\n%s", h.view())
	}
}

func TestSyncSetupStoppedByAnotherDialogAfterSavingTheIdentitySaysWhatChanged(t *testing.T) {
	h, save, _ := saveIdentityInTheApp(t)
	h.send(core.SyncStatusMsg{}) // another dialog opens while the step runs
	h.finish(save)
	wantChangesSaid(t, h)
}

// Without an identity, the app signs in first, then asks for the
// identity with the GitHub account's name and private address already
// typed: Enter takes them, and they are saved as offered.
func TestSyncSetupInsideTheAppSignsInThenOffersTheIdentityFromGitHub(t *testing.T) {
	fake, _ := fakeGitHub(t, "[]", false)
	writeOutside(t, fake, "user.json", `{"login":"ana","id":583231,"name":"Ana Ruiz"}`)
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"), app.WithExec(func(c *exec.Cmd, done tea.ExecCallback) tea.Cmd {
		return func() tea.Msg { return done(c.Run()) }
	}))
	h.send(core.SyncSetupMsg{})
	h.key(tea.KeyEnter) // Start
	if !strings.Contains(h.view(), "Sign in to GitHub") {
		t.Fatalf("without an identity, the sign-in still comes first:\n%s", h.view())
	}
	h.key(tea.KeyEnter) // Sign in
	view := h.view()
	if !strings.Contains(view, "Your name, for git") || !strings.Contains(view, "Ana Ruiz") || !strings.Contains(view, "From your GitHub account") {
		t.Fatalf("after signing in, the name must come from GitHub:\n%s", view)
	}
	h.key(tea.KeyEnter) // the name as offered
	if !strings.Contains(h.view(), "583231+ana@users.noreply.github.com") {
		t.Fatalf("the email must be GitHub's private address:\n%s", h.view())
	}
	h.key(tea.KeyEnter) // the email as offered
	h.key(tea.KeyEnter) // Save
	global, err := os.ReadFile(filepath.Join(fake, "gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(global), "name = Ana Ruiz") || !strings.Contains(string(global), "email = 583231+ana@users.noreply.github.com") {
		t.Errorf("the identity from GitHub must be saved:\n%s", global)
	}
}
