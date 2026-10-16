//go:build !windows

// The fake gh below is a shell script: these tests run on macOS and Linux.
// Windows has no sh to run it, and the assistant's Windows-specific parts
// (the install hint, the paths) are covered by tests that need no gh.

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// fakeToken is what the fake gh says it holds: it must never show up.
const fakeToken = "gho_FAKEtoken1234567890abcdef"

// fakeGHScript answers like gh, from the files in $FAKE_GH_DIR, and logs
// every call (one line of arguments each) in $FAKE_GH_DIR/calls.
//
//   - loggedin: present when signed in; `auth login` creates it when the
//     file login says ok, and fails like a cancelled sign-in otherwise.
//   - repos.json: what `repo list` prints; listfail makes it fail.
//   - user.json: what `api user` prints; without it, it fails.
//   - remotes/ana/<name>.git: the repositories; `repo create` makes one
//     (bare) and fails like GitHub when it exists.
const fakeGHScript = `#!/bin/sh
d="$FAKE_GH_DIR"
printf '%s\n' "$*" >> "$d/calls"
case "$1 $2" in
"--version ")
	echo "gh version 2.62.0 (2024-11-14)" ;;
"auth status")
	if [ -f "$d/loggedin" ]; then
		echo "github.com"
		echo "  Logged in to github.com account ana (keyring)"
		echo "  Token: ` + fakeToken + `"
		exit 0
	fi
	echo "You are not logged into any GitHub hosts. To log in, run: gh auth login" >&2
	exit 1 ;;
"auth login")
	echo "! First copy your one-time code: 4F2A-91KD"
	if [ "$(cat "$d/login" 2>/dev/null)" = ok ]; then
		touch "$d/loggedin"
		echo "Logged in as ana"
		exit 0
	fi
	echo "error: authentication cancelled" >&2
	exit 1 ;;
"auth setup-git")
	if [ -f "$d/setupgitfail" ]; then
		echo "could not write the git config" >&2
		exit 1
	fi
	git config --global "credential.https://github.com.helper" "!gh auth git-credential" || exit 1 ;;
"api user")
	if [ ! -f "$d/user.json" ]; then
		echo "HTTP 401: Bad credentials (https://api.github.com/user)" >&2
		exit 1
	fi
	cat "$d/user.json" ;;
"repo list")
	if [ -f "$d/listfail" ]; then
		echo "HTTP 502: bad gateway for https://ana:` + fakeToken + `@api.github.com/graphql" >&2
		exit 1
	fi
	cat "$d/repos.json" ;;
"repo create")
	if [ -f "$d/createfail" ]; then
		echo "HTTP 403: forbidden" >&2
		exit 1
	fi
	if [ -e "$d/remotes/ana/$3.git" ]; then
		echo "GraphQL: Name already exists on this account (createRepository)" >&2
		exit 1
	fi
	git init -q --bare "$d/remotes/ana/$3.git" || exit 1
	echo "https://github.com/ana/$3" ;;
*)
	echo "fake gh: unknown command $*" >&2
	exit 2 ;;
esac
`

// fakeGH is a gh on PATH that stands in for GitHub, with a git global
// config of the test's own in which https://github.com/ names the local
// bare repositories under remotes. The test changes the environment, so
// it cannot run in parallel.
type fakeGH struct {
	dir, remotes, global string
}

func newFakeGH(t *testing.T, signedIn bool) *fakeGH {
	t.Helper()
	needGit(t)
	f := &fakeGH{dir: t.TempDir()}
	f.remotes = filepath.Join(f.dir, "remotes")
	bin := filepath.Join(f.dir, "bin")
	for _, d := range []string{bin, filepath.Join(f.remotes, "ana")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(fakeGHScript), 0o755); err != nil {
		t.Fatal(err)
	}
	f.global = filepath.Join(f.dir, "gitconfig")
	f.writeGlobal(t, true)
	f.setRepos(t, "[]")
	if signedIn {
		writeFile(t, f.dir, "loggedin", "")
	}
	t.Setenv("FAKE_GH_DIR", f.dir)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_CONFIG_GLOBAL", f.global)
	return f
}

// writeGlobal writes the test's git global config, with or without an
// identity.
func (f *fakeGH) writeGlobal(t *testing.T, identity bool) {
	t.Helper()
	cfg := "[init]\n\tdefaultBranch = main\n[url \"file://" + filepath.ToSlash(f.remotes) + "/\"]\n\tinsteadOf = https://github.com/\n"
	if identity {
		cfg = "[user]\n\tname = Test\n\temail = test@example.com\n" + cfg
	}
	if err := os.WriteFile(f.global, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f *fakeGH) setRepos(t *testing.T, json string) {
	t.Helper()
	writeFile(t, f.dir, "repos.json", json)
}

// remote makes the bare repository ana/name, as if it were on GitHub.
func (f *fakeGH) remote(t *testing.T, name string) string {
	t.Helper()
	git(t, filepath.Join(f.remotes, "ana"), "init", "-q", "--bare", name+".git")
	return filepath.Join(f.remotes, "ana", name+".git")
}

// calls lists the gh commands run so far, one string of arguments each.
func (f *fakeGH) calls(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.dir, "calls"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}

// called reports whether gh ran with arguments starting with prefix.
func (f *fakeGH) called(t *testing.T, prefix string) bool {
	t.Helper()
	for _, c := range f.calls(t) {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// setup runs `misoku sync setup --dir dir` with input as what the user
// types, and returns its exit code and everything it printed.
func setup(t *testing.T, dir, cfg, input string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	code := syncSetup(context.Background(), []string{"--dir", dir}, strings.NewReader(input), &out, &out, cfg)
	return code, out.String()
}

// fileHash hashes the file at p, or says it is absent.
func fileHash(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return "absent"
	}
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// snapshot is everything setup must leave as it was when it changes
// nothing: the vault (.git included), Misoku's config and git's global
// config.
func (f *fakeGH) snapshot(t *testing.T, dir, cfg string) string {
	t.Helper()
	return treeHash(t, dir) + " " + fileHash(t, cfg) + " " + fileHash(t, f.global)
}

// notesVault is a folder with two notes and no git.
func notesVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "hello.md", "# hello\n")
	writeFile(t, dir, "sub/deep.md", "deep\n")
	return dir
}

func TestSyncSetupWithoutGHSaysHowToInstallItAndChangesNothing(t *testing.T) {
	f := newFakeGH(t, true)
	// A PATH with git and nothing else: no gh.
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	bin := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	before := f.snapshot(t, dir, cfg)
	code, out := setup(t, dir, cfg, "\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	want := map[string]string{"darwin": "brew install gh", "linux": "apt install gh"}[runtime.GOOS]
	wantOutput(t, out, "gh", "is not installed", want, "Nothing was changed")
	if after := f.snapshot(t, dir, cfg); after != before {
		t.Error("setup without gh changed the vault or a config")
	}
}

func TestSyncSetupSignsInInTheForegroundThenContinues(t *testing.T) {
	f := newFakeGH(t, false)
	writeFile(t, f.dir, "login", "ok")
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	// Start, sign in, create a new repository (the only choice), with the
	// name it offers.
	code, out := setup(t, dir, cfg, "\n\n1\n\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if !f.called(t, "auth login --web --git-protocol https --hostname github.com") {
		t.Errorf("gh auth login was not run as expected; calls:\n%s", strings.Join(f.calls(t), "\n"))
	}
	// gh printed to the user's own terminal: the code reached the output.
	wantOutput(t, out, "4F2A-91KD")
	if !f.called(t, "repo create") {
		t.Error("after signing in, setup must go on to the repository")
	}
}

func TestSyncSetupStopsWhenSignInIsCancelled(t *testing.T) {
	f := newFakeGH(t, false)
	writeFile(t, f.dir, "login", "cancel")
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	before := f.snapshot(t, dir, cfg)
	code, out := setup(t, dir, cfg, "\n\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "not signed in to GitHub", "Nothing was changed")
	if f.called(t, "repo") {
		t.Error("no repository may be listed or made without signing in")
	}
	if after := f.snapshot(t, dir, cfg); after != before {
		t.Error("a cancelled sign-in changed the vault or a config")
	}
}

func TestSyncSetupAsksForAMissingIdentityAndSetsItOnlyOnConfirm(t *testing.T) {
	f := newFakeGH(t, true)
	f.writeGlobal(t, false)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	before := f.snapshot(t, dir, cfg)
	// Start, name, email, then no to saving them.
	code, out := setup(t, dir, cfg, "\nAna Ruiz\nana@example.com\nn\n")
	if code != 1 {
		t.Errorf("declined: exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "git does not know your name", "nothing was changed")
	if after := f.snapshot(t, dir, cfg); after != before {
		t.Fatal("declining to save the identity changed something")
	}
	// Yes this time, then cancel at the repository: the identity stays.
	code, out = setup(t, dir, cfg, "\nAna Ruiz\nana@example.com\ny\n\n")
	if code != 1 {
		t.Errorf("exit %d, want 1 (cancelled at the repository):\n%s", code, out)
	}
	if got := git(t, dir, "config", "--global", "user.name"); got != "Ana Ruiz" {
		t.Errorf("user.name = %q, want Ana Ruiz", got)
	}
	if got := git(t, dir, "config", "--global", "user.email"); got != "ana@example.com" {
		t.Errorf("user.email = %q, want ana@example.com", got)
	}
}

func TestSyncSetupCreatesAPrivateRepositoryAndPushesTheNotes(t *testing.T) {
	f := newFakeGH(t, true)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n1\nnotes\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if !f.called(t, "repo create notes --private") {
		t.Errorf("the repository must be made private; calls:\n%s", strings.Join(f.calls(t), "\n"))
	}
	remote := filepath.Join(f.remotes, "ana", "notes.git")
	files := remoteFiles(t, remote)
	if files["hello.md"] != "# hello\n" || files["sub/deep.md"] != "deep\n" {
		t.Errorf("the notes did not reach the new repository: %v", files)
	}
	if got := git(t, dir, "config", "--get", "remote.origin.url"); got != "https://github.com/ana/notes.git" {
		t.Errorf("origin = %q, want the repository's HTTPS URL", got)
	}
	data, err := os.ReadFile(cfg)
	if err != nil || !strings.Contains(string(data), "vaults") {
		t.Errorf("the vault must be listed in the config: %v\n%s", err, data)
	}
	wantOutput(t, out, "every 5m", "when you quit", "misoku sync setup", "ana/notes")
}

func TestSyncSetupBringsTheNotesOfAnExistingRepository(t *testing.T) {
	f := newFakeGH(t, true)
	remote := f.remote(t, "notes")
	seedRemote(t, remote, map[string]string{"from-laptop.md": "written on the laptop\n"})
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
	dir := filepath.Join(t.TempDir(), "Notes") // a new computer: no folder yet
	cfg := filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n1\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if got := vaultFiles(t, dir)["from-laptop.md"]; got != "written on the laptop\n" {
		t.Errorf("from-laptop.md = %q, want the repository's note", got)
	}
	if f.called(t, "repo create") {
		t.Error("picking a repository must not create one")
	}
}

func TestSyncSetupRefusesAPublicRepository(t *testing.T) {
	f := newFakeGH(t, true)
	f.remote(t, "site")
	f.setRepos(t, `[{"nameWithOwner":"ana/site","visibility":"PUBLIC","url":"https://github.com/ana/site"}]`)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	before := f.snapshot(t, dir, cfg)
	// Pick the public one by number, then by name, then give up.
	code, out := setup(t, dir, cfg, "\n1\nana/site\n\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	if n := strings.Count(out, "ana/site is public"); n != 2 {
		t.Errorf("each pick of the public repository must be refused (%d refusals):\n%s", n, out)
	}
	wantOutput(t, out, "public: not allowed")
	if after := f.snapshot(t, dir, cfg); after != before {
		t.Error("refusing a public repository changed something")
	}
}

func TestSyncSetupSaysWhenTheNameIsTakenAndLeavesNothingHalfDone(t *testing.T) {
	f := newFakeGH(t, true)
	f.remote(t, "notes") // on GitHub, though the list does not show it
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	before := f.snapshot(t, dir, cfg)
	beforeRemotes := treeHash(t, f.remotes)
	code, out := setup(t, dir, cfg, "\n1\nnotes\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "notes already exists", "choose another name")
	// setup-git ran before the creation was tried: the global config may
	// differ, but neither the vault nor Misoku's config may.
	if treeHash(t, dir)+fileHash(t, cfg) != strings.Join(strings.Fields(before)[:2], "") {
		t.Error("a failed creation changed the vault or Misoku's config")
	}
	if treeHash(t, f.remotes) != beforeRemotes {
		t.Error("a failed creation changed the repositories")
	}
}

func TestSyncSetupCancelledAtAnyStepChangesNothing(t *testing.T) {
	for _, c := range []struct {
		name, input string
		signedIn    bool
	}{
		{"at the start", "n\n", true},
		{"at the start, end of input", "", true},
		{"at the sign-in", "\nn\n", false},
		{"at the repository", "\n\n", true},
		{"at the new repository's name", "\n1\n", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFakeGH(t, c.signedIn)
			dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
			before := f.snapshot(t, dir, cfg)
			code, out := setup(t, dir, cfg, c.input)
			if code != 1 {
				t.Errorf("exit %d, want 1:\n%s", code, out)
			}
			wantOutput(t, out, "cancelled; nothing was changed")
			if after := f.snapshot(t, dir, cfg); after != before {
				t.Error("cancelling changed the vault or a config")
			}
			if f.called(t, "repo create") || f.called(t, "auth setup-git") {
				t.Errorf("cancelling ran gh: %v", f.calls(t))
			}
		})
	}
}

func TestSyncSetupOnAConfiguredVaultSaysSoAndChangesNothing(t *testing.T) {
	f := newFakeGH(t, true)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	if code, out := setup(t, dir, cfg, "\n1\nnotes\n"); code != 0 {
		t.Fatalf("first setup: exit %d:\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(f.dir, "calls")); err != nil {
		t.Fatal(err)
	}
	before := f.snapshot(t, dir, cfg)
	code, out := setup(t, dir, cfg, "")
	if code != 0 {
		t.Errorf("exit %d, want 0:\n%s", code, out)
	}
	wantOutput(t, out, "already set up", "https://github.com/ana/notes.git", "nothing changed")
	if after := f.snapshot(t, dir, cfg); after != before {
		t.Error("setup on a configured vault changed something")
	}
	if calls := f.calls(t); len(calls) > 0 {
		t.Errorf("setup on a configured vault ran gh: %v", calls)
	}
}

func TestSyncSetupNeverShowsOrStoresAToken(t *testing.T) {
	f := newFakeGH(t, false)
	writeFile(t, f.dir, "login", "cancel")
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	_, cancelled := setup(t, dir, cfg, "\n\n")
	// Signed in now, but listing the repositories fails with a message
	// that carries the token in a URL.
	writeFile(t, f.dir, "loggedin", "")
	writeFile(t, f.dir, "listfail", "")
	_, failed := setup(t, dir, cfg, "\n")
	if err := os.Remove(filepath.Join(f.dir, "listfail")); err != nil {
		t.Fatal(err)
	}
	_, done := setup(t, dir, cfg, "\n1\nnotes\n")
	wantOutput(t, failed, "could not list your repositories")
	for name, text := range map[string]string{
		"output (cancelled sign-in)": cancelled,
		"output (failed list)":       failed,
		"output (done)":              done,
		"Misoku's config":            readOr(t, cfg),
		"git's global config":        readOr(t, f.global),
		"the vault's git config":     readOr(t, filepath.Join(dir, ".git", "config")),
		"gh's arguments":             strings.Join(f.calls(t), "\n"),
	} {
		if strings.Contains(text, fakeToken) {
			t.Errorf("%s shows the token:\n%s", name, text)
		}
	}
}

func readOr(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

func TestBackgroundSyncAfterSetupNeverPrompts(t *testing.T) {
	f := newFakeGH(t, true)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	if code, out := setup(t, dir, cfg, "\n1\nnotes\n"); code != 0 {
		t.Fatalf("setup: exit %d:\n%s", code, out)
	}
	if !f.called(t, "auth setup-git --hostname github.com") {
		t.Fatal("setup must make git use gh's credentials (gh auth setup-git)")
	}
	if got := git(t, dir, "config", "--global", "credential.https://github.com.helper"); !strings.Contains(got, "gh auth git-credential") {
		t.Errorf("git's credential helper = %q, want gh's", got)
	}
	// The background runner: no terminal, prompts off, a time limit.
	writeFile(t, dir, "later.md", "written after setup\n")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	r := gitsync.New(dir, 30*time.Second)
	if st := gitsync.Detect(ctx, r); st.Status != gitsync.Ready {
		t.Fatalf("after setup the vault must be ready: %s", st.Message)
	}
	lock, ok, err := gitsync.LockVault(ctx, r)
	if err != nil || !ok {
		t.Fatalf("lock: %v %v", ok, err)
	}
	defer lock.Unlock()
	res := gitsync.SyncVault(ctx, r, lock, gitsync.SyncOptions{Device: "test"})
	if res.Outcome != gitsync.Synced {
		t.Fatalf("background sync: %v", res.Err)
	}
	if remoteFiles(t, filepath.Join(f.remotes, "ana", "notes.git"))["later.md"] == "" {
		t.Error("the background sync did not push the new note")
	}
}

// After the user saved their identity, stopping says so: it is not true
// that nothing changed.
func TestSyncSetupStoppedAfterSavingTheIdentitySaysWhatChanged(t *testing.T) {
	f := newFakeGH(t, true)
	f.writeGlobal(t, false)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\nAna Ruiz\nana@example.com\ny\n\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "Your git name and email were saved", "Nothing else changed")
	if strings.Contains(strings.ToLower(out), "nothing was changed") {
		t.Errorf("the identity was saved, yet setup says nothing changed:\n%s", out)
	}
}

// After gh set up git's access, a failure says so.
func TestSyncSetupFailedAfterSetupGitSaysWhatChanged(t *testing.T) {
	f := newFakeGH(t, true)
	writeFile(t, f.dir, "createfail", "")
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n1\nnotes\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "GitHub did not create the repository", "GitHub access for git was set up with gh", "Nothing else changed")
	if strings.Contains(strings.ToLower(out), "nothing was changed") {
		t.Errorf("git's access was set up, yet setup says nothing changed:\n%s", out)
	}
}

// gh is checked before the identity is asked for: a missing gh never
// leaves a saved identity behind.
func TestSyncSetupChecksGHBeforeAskingForTheIdentity(t *testing.T) {
	f := newFakeGH(t, true)
	f.writeGlobal(t, false)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	bin := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	before := f.snapshot(t, dir, cfg)
	code, out := setup(t, dir, cfg, "\nAna Ruiz\nana@example.com\ny\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "is not installed")
	if strings.Contains(out, "git does not know your name") {
		t.Errorf("without gh, setup must stop before asking for the identity:\n%s", out)
	}
	if after := f.snapshot(t, dir, cfg); after != before {
		t.Error("setup without gh changed something")
	}
}

// A repository whose name is a number beyond the list is read as a name.
func TestSyncSetupReadsANumberBeyondTheListAsAName(t *testing.T) {
	f := newFakeGH(t, true)
	f.remote(t, "2024")
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"},{"nameWithOwner":"ana/2024","visibility":"PRIVATE","url":"https://github.com/ana/2024"}]`)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n2024\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if got := git(t, dir, "config", "--get", "remote.origin.url"); got != "https://github.com/ana/2024.git" {
		t.Errorf("origin = %q, want ana/2024", got)
	}
}

// A user name or password in a listed URL never reaches origin.
func TestSyncSetupStoresOriginWithoutCredentials(t *testing.T) {
	f := newFakeGH(t, true)
	f.remote(t, "notes")
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://ana:secret@github.com/ana/notes"}]`)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n1\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	if got := git(t, dir, "config", "--get", "remote.origin.url"); got != "https://github.com/ana/notes.git" {
		t.Errorf("origin = %q, want it without credentials", got)
	}
}

// A failed gh auth setup-git is one sentence, ending with its period.
func TestSyncSetupReportsAFailedSetupGitAsASentence(t *testing.T) {
	f := newFakeGH(t, true)
	writeFile(t, f.dir, "setupgitfail", "")
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n1\nnotes\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "gh could not give git the access to GitHub (could not write the git config).\n", "Nothing was changed.")
}

// Inside another repository that does not track it, Enter gives the
// notes a repository of their own and the assistant syncs them as usual;
// the outer repository is left exactly as it was.
func TestSyncSetupInsideAnotherRepositorySyncsWithARepositoryOfItsOwn(t *testing.T) {
	f := newFakeGH(t, true)
	outer := t.TempDir()
	git(t, outer, "init", "-q", "-b", "main")
	writeFile(t, outer, "readme.txt", "outer\n")
	git(t, outer, "add", "readme.txt")
	git(t, outer, "commit", "-q", "-m", "outer")
	dir := filepath.Join(outer, "notes")
	writeFile(t, dir, "hello.md", "# hello\n")
	outerState := func() string {
		return git(t, outer, "status", "--porcelain") + git(t, outer, "rev-parse", "HEAD") + git(t, outer, "ls-files", "--stage")
	}
	before := outerState()
	cfg := filepath.Join(t.TempDir(), "config.json")
	// Create the repository of its own, start, create a new GitHub
	// repository (the only choice), with the name it offers.
	code, out := setup(t, dir, cfg, "\n\n1\n\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	wantOutput(t, out, "Done:")
	remote := filepath.Join(f.remotes, "ana", "notes.git")
	wantFiles(t, "origin", remoteFiles(t, remote), map[string]string{"hello.md": "# hello\n", ".gitattributes": gitsync.Attributes})
	if after := outerState(); after != before {
		t.Errorf("the outer repository changed:\nbefore %s\nafter  %s", before, after)
	}
}

// anaUser is the fake GitHub account, as `gh api user` prints it.
const anaUser = `{"login":"ana","id":583231,"name":"Ana Ruiz","email":null}`

// Without an identity, the sign-in comes first, then the identity, filled
// in from the GitHub account: Enter confirms it as it is.
func TestSyncSetupSignsInThenTakesTheIdentityFromGitHub(t *testing.T) {
	f := newFakeGH(t, false)
	f.writeGlobal(t, false)
	writeFile(t, f.dir, "login", "ok")
	writeFile(t, f.dir, "user.json", anaUser)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	// Start, sign in, the name and the email as offered, save, create a
	// new repository with the name it offers.
	code, out := setup(t, dir, cfg, "\n\n\n\ny\n1\n\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	signIn, identity := strings.Index(out, "4F2A-91KD"), strings.Index(out, "git does not know your name")
	if signIn < 0 || identity < 0 || signIn > identity {
		t.Errorf("the sign-in (at %d) must come before the identity (at %d):\n%s", signIn, identity, out)
	}
	wantOutput(t, out, "from your GitHub account", "Your name [Ana Ruiz]:", "Your email [583231+ana@users.noreply.github.com]:")
	if got := git(t, dir, "config", "--global", "user.name"); got != "Ana Ruiz" {
		t.Errorf("user.name = %q, want Ana Ruiz", got)
	}
	if got := git(t, dir, "config", "--global", "user.email"); got != "583231+ana@users.noreply.github.com" {
		t.Errorf("user.email = %q, want GitHub's private address", got)
	}
}

// The identity GitHub suggests can be changed before it is saved.
func TestSyncSetupLetsTheUserEditTheIdentityFromGitHub(t *testing.T) {
	f := newFakeGH(t, true)
	f.writeGlobal(t, false)
	writeFile(t, f.dir, "user.json", anaUser)
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	// Start, another name, the email as offered, save, cancel at the
	// repository.
	code, out := setup(t, dir, cfg, "\nBea\n\ny\n\n")
	if code != 1 {
		t.Errorf("exit %d, want 1 (cancelled at the repository):\n%s", code, out)
	}
	if got := git(t, dir, "config", "--global", "user.name"); got != "Bea" {
		t.Errorf("user.name = %q, want the edited Bea", got)
	}
	if got := git(t, dir, "config", "--global", "user.email"); got != "583231+ana@users.noreply.github.com" {
		t.Errorf("user.email = %q, want GitHub's private address", got)
	}
}
