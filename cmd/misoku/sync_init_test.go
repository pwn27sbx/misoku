package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// TestMain gives git an empty global config (with an identity and main as
// the default branch) and no system config, so the user's settings never
// leak into a test. Set once with os.Setenv so the tests can run in
// parallel.
//
// With mainArgsEnv set, the test binary runs main() with those arguments
// instead, so a test can check what the real command line does.
func TestMain(m *testing.M) {
	if args, ok := os.LookupEnv(mainArgsEnv); ok {
		os.Args = []string{"misoku"}
		if args != "" { // empty: misoku with no arguments
			os.Args = append(os.Args, strings.Split(args, "\n")...)
		}
		main()
		os.Exit(0)
	}
	os.Exit(runIsolated(m))
}

const mainArgsEnv = "MISOKU_CMD_TEST_MAIN_ARGS"

// runMain runs misoku's main() with args in a child process and returns
// its exit code and output.
func runMain(t *testing.T, args ...string) (int, string) {
	t.Helper()
	return runMainIn(t, t.TempDir(), args...)
}

// runMainIn is runMain with home as the home, cache and config folder,
// so a test can write a config there first (see configIn).
// $MISOKU_WORKSPACE is cleared: the user's own never leaks in.
func runMainIn(t *testing.T, home string, args ...string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), mainArgsEnv+"="+strings.Join(args, "\n"), "MISOKU_WORKSPACE=",
		"HOME="+home, "USERPROFILE="+home, "XDG_CACHE_HOME="+home, "XDG_CONFIG_HOME="+home, "LOCALAPPDATA="+home, "APPDATA="+home)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, string(out)
}

func TestMainRunsSyncAndShowsItInHelp(t *testing.T) {
	t.Parallel()
	code, out := runMain(t, "sync", "nonsense")
	if code != 2 {
		t.Errorf("misoku sync nonsense: exit %d, want 2:\n%s", code, out)
	}
	wantOutput(t, out, "Usage: misoku sync init <url>")
	code, out = runMain(t, "--help")
	if code != 0 {
		t.Errorf("misoku --help: exit %d:\n%s", code, out)
	}
	wantOutput(t, out, "misoku sync init <url> [--dir <folder>]", "-list-themes")
}

func runIsolated(m *testing.M) int {
	dir, err := os.MkdirTemp("", "misoku-cmd-test-")
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
	os.Setenv("GIT_CONFIG_GLOBAL", global)
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	os.Unsetenv("MISOKU_SYNC")
	return m.Run()
}

const wantAttributes = "# Misoku: keep every file byte for byte (no line-ending conversion)\n* -text\n"

func needGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

// git runs a setup command in dir and fails the test if it fails.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRemote is an empty bare repository standing in for GitHub.
func newRemote(t *testing.T) string {
	t.Helper()
	parent := t.TempDir()
	git(t, parent, "init", "-q", "--bare", "notes.git")
	return filepath.Join(parent, "notes.git")
}

// seedRemote pushes files to remote's main from a throwaway clone, the
// way another tool (or GitHub's "add a README") would.
func seedRemote(t *testing.T, remote string, files map[string]string) {
	t.Helper()
	work := t.TempDir()
	git(t, work, "init", "-q", "-b", "main")
	for name, content := range files {
		writeFile(t, work, name, content)
	}
	git(t, work, "add", "-A")
	git(t, work, "commit", "-q", "-m", "seed")
	git(t, work, "push", "-q", remote, "main")
}

// remoteFiles reads every file on remote's main, byte for byte.
func remoteFiles(t *testing.T, remote string) map[string]string {
	t.Helper()
	files := map[string]string{}
	for _, name := range strings.Split(git(t, remote, "ls-tree", "-r", "--name-only", "-z", "main"), "\x00") {
		if name == "" {
			continue
		}
		out, err := exec.Command("git", "-C", remote, "cat-file", "blob", "main:"+name).Output()
		if err != nil {
			t.Fatalf("cat-file %s: %v", name, err)
		}
		files[name] = string(out)
	}
	return files
}

// vaultFiles reads every file in dir outside .git, byte for byte.
func vaultFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// treeHash hashes every file under dir, .git included, so a refused init
// can be shown to have changed nothing at all. .git/FETCH_HEAD is left
// out: it is git's own note of the last fetch, which every sync rewrites.
func treeHash(t *testing.T, dir string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if filepath.ToSlash(rel) == ".git/FETCH_HEAD" {
			return nil
		}
		fmt.Fprintf(h, "%s %v\n", filepath.ToSlash(rel), d.IsDir())
		if !d.IsDir() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			h.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func wantFiles(t *testing.T, what string, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got files %v, want %v", what, sortedKeys(got), sortedKeys(want))
	}
	for name, content := range want {
		if g, ok := got[name]; !ok || g != content {
			t.Errorf("%s: %s = %q (present %v), want %q", what, name, g, ok, content)
		}
	}
}

func sortedKeys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// initVault runs `misoku sync init url --dir dir` with the config at cfg.
func initVault(t *testing.T, dir, url, cfg string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = syncInit(context.Background(), []string{url, "--dir", dir}, &out, &errOut, cfg)
	return code, out.String(), errOut.String()
}

// wantSynced checks that dir and remote hold the same commit on main, that
// dir's main tracks origin/main and that Misoku's exclusions are in place.
func wantSynced(t *testing.T, dir, remote string) {
	t.Helper()
	if got := git(t, dir, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Errorf("branch = %q, want main", got)
	}
	if got := git(t, dir, "rev-parse", "--abbrev-ref", "@{upstream}"); got != "origin/main" {
		t.Errorf("upstream = %q, want origin/main", got)
	}
	if local, theirs := git(t, dir, "rev-parse", "HEAD"), git(t, remote, "rev-parse", "main"); local != theirs {
		t.Errorf("HEAD %s, origin's main %s: want the same commit", local, theirs)
	}
	if got := git(t, dir, "status", "--porcelain", "--untracked-files=all"); got != "" {
		t.Errorf("nothing may be left uncommitted, got:\n%s", got)
	}
	exclude, err := os.ReadFile(filepath.Join(dir, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []string{".*.tmp", ".DS_Store"} {
		if !slices.Contains(strings.Split(string(exclude), "\n"), pattern) {
			t.Errorf("info/exclude lacks %s:\n%s", pattern, exclude)
		}
	}
}

// listedVaults reads sync.vaults from the config file at cfg.
func listedVaults(t *testing.T, cfg string) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Sync struct {
			Vaults map[string]json.RawMessage `json:"vaults"`
		} `json:"sync"`
	}
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("config is not valid JSON: %v\n%s", err, data)
	}
	return c.Sync.Vaults
}

func wantOutput(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("output lacks %q:\n%s", w, out)
		}
	}
}

func TestSyncInitFirstDevice(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, dir := newRemote(t), t.TempDir()
	notes := map[string]string{
		"a.md":          "# a\r\nwindows line ends stay\r\n",
		"sub/b.md":      "b\n",
		"img/pic.png":   "\x89PNG\r\n\x1a\nbinary",
		"Tildes é.md":   "ñ\n",
		".hidden.md":    "kept\n",
		"notes/c.md":    "c\n",
		"notes/c.md~":   "backup\n",
		"another/d.txt": "d\n",
	}
	for name, content := range notes {
		writeFile(t, dir, name, content)
	}
	writeFile(t, dir, ".DS_Store", "mac")
	writeFile(t, dir, ".a.md.123.tmp", "half a save")
	cfg := filepath.Join(t.TempDir(), "misoku", "config.json")
	before := `{"theme": "nord", "future_setting": {"x": [1, 2]}, "keys": {"save": ["<C-s>"]}}`
	writeFile(t, filepath.Dir(cfg), "config.json", before)

	code, stdout, stderr := initVault(t, dir, remote, cfg)
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	want := map[string]string{".gitattributes": wantAttributes}
	for name, content := range notes {
		want[name] = content
	}
	wantFiles(t, "origin", remoteFiles(t, remote), want)
	want[".DS_Store"], want[".a.md.123.tmp"] = "mac", "half a save"
	wantFiles(t, "vault", vaultFiles(t, dir), want)
	wantSynced(t, dir, remote)
	if got := git(t, remote, "symbolic-ref", "HEAD"); got != "refs/heads/main" {
		t.Errorf("origin's HEAD = %q", got)
	}
	if n := git(t, remote, "rev-list", "--count", "main"); n != "1" {
		t.Errorf("one first commit, got %s", n)
	}

	vaults := listedVaults(t, cfg)
	abs, _ := filepath.Abs(dir)
	if raw, ok := vaults[abs]; !ok || string(raw) != "{}" || len(vaults) != 1 {
		t.Errorf("sync.vaults = %v, want only %q: {}", vaults, abs)
	}
	var all map[string]any
	data, _ := os.ReadFile(cfg)
	if err := json.Unmarshal(data, &all); err != nil {
		t.Fatal(err)
	}
	if all["theme"] != "nord" || fmt.Sprint(all["future_setting"]) != "map[x:[1 2]]" || fmt.Sprint(all["keys"]) != "map[save:[<C-s>]]" {
		t.Errorf("other settings must stay as they were, config is now:\n%s", data)
	}
	if !strings.Contains(string(data), `"<C-s>"`) {
		t.Errorf("text must stay readable (no \\u003c escapes):\n%s", data)
	}

	wantOutput(t, stdout,
		"Setting up sync of "+dir+" with "+remote,
		"unencrypted",
		"Created a git repository (branch main)",
		"Added origin "+remote,
		"Wrote .gitattributes",
		"Committed 9 files",
		"Pushed to origin (branch main)",
		"Added "+abs+" to sync.vaults in "+cfg,
		"Done",
	)
	if stderr != "" {
		t.Errorf("nothing on stderr, got:\n%s", stderr)
	}
}

func TestSyncInitAnotherDevice(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, first := newRemote(t), t.TempDir()
	writeFile(t, first, "a.md", "# a\r\n")
	writeFile(t, first, "img/p.png", "\x00\x01png")
	if code, out, errOut := initVault(t, first, remote, filepath.Join(t.TempDir(), "c.json")); code != 0 {
		t.Fatalf("first device: exit %d\n%s\n%s", code, out, errOut)
	}
	pushed := git(t, remote, "rev-parse", "main")

	dir := filepath.Join(t.TempDir(), "empty vault")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Listed already, by hand: the config stays byte for byte.
	cfg := filepath.Join(t.TempDir(), "config.json")
	key, _ := json.Marshal(dir)
	listed := `{"sync": {"vaults": {` + string(key) + `: {"device": "desk"}}}}`
	writeFile(t, filepath.Dir(cfg), "config.json", listed)
	code, stdout, stderr := initVault(t, dir, remote, cfg)
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantFiles(t, "vault", vaultFiles(t, dir), map[string]string{
		"a.md": "# a\r\n", "img/p.png": "\x00\x01png", ".gitattributes": wantAttributes,
	})
	wantSynced(t, dir, remote)
	if got := git(t, remote, "rev-parse", "main"); got != pushed {
		t.Errorf("nothing new to push: origin moved from %s to %s", pushed, got)
	}
	if data, _ := os.ReadFile(cfg); string(data) != listed {
		t.Errorf("a listed vault leaves the config as it was, got:\n%s", data)
	}
	wantOutput(t, stdout, "Brought origin's notes", "Done")
	for _, not := range []string{"Committed", "Pushed", "Wrote .gitattributes", "to sync.vaults", "Already set up"} {
		if strings.Contains(stdout, not) {
			t.Errorf("nothing was %q here:\n%s", not, stdout)
		}
	}
}

func TestSyncInitBothWithNotes(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, first := newRemote(t), t.TempDir()
	writeFile(t, first, "theirs.md", "from the laptop\n")
	writeFile(t, first, "same.md", "laptop version\n")
	if code, out, errOut := initVault(t, first, remote, filepath.Join(t.TempDir(), "c.json")); code != 0 {
		t.Fatalf("first device: exit %d\n%s\n%s", code, out, errOut)
	}

	dir := t.TempDir()
	writeFile(t, dir, "mine.md", "from the desk\n")
	writeFile(t, dir, "same.md", "desk version\n")
	code, stdout, stderr := initVault(t, dir, remote, filepath.Join(t.TempDir(), "c.json"))
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	copies, err := gitsync.FindConflictCopies(dir)
	if err != nil || len(copies) != 1 {
		t.Fatalf("one conflict copy for same.md, got %v (%v)", copies, err)
	}
	if !strings.HasPrefix(copies[0], "same (conflict ") {
		t.Errorf("conflict copy name %q", copies[0])
	}
	want := map[string]string{
		"theirs.md": "from the laptop\n", "mine.md": "from the desk\n",
		"same.md": "desk version\n", copies[0]: "laptop version\n",
		".gitattributes": wantAttributes,
	}
	wantFiles(t, "vault", vaultFiles(t, dir), want)
	wantFiles(t, "origin", remoteFiles(t, remote), want)
	wantSynced(t, dir, remote)
	wantOutput(t, stdout, "Kept both versions", copies[0], "Pushed to origin (branch main)")
}

func TestSyncInitRemoteWithOnlyAReadme(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote := newRemote(t)
	seedRemote(t, remote, map[string]string{"README.md": "# notes\n"})
	dir := t.TempDir()
	writeFile(t, dir, "a.md", "a\n")

	code, stdout, stderr := initVault(t, dir, remote, filepath.Join(t.TempDir(), "c.json"))
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	want := map[string]string{"README.md": "# notes\n", "a.md": "a\n", ".gitattributes": wantAttributes}
	wantFiles(t, "vault", vaultFiles(t, dir), want)
	wantFiles(t, "origin", remoteFiles(t, remote), want)
	wantSynced(t, dir, remote)
	// The README's commit stays in the history: nothing was rewritten.
	readme := git(t, remote, "rev-list", "--max-parents=0", "main")
	if !strings.Contains(readme, git(t, remote, "rev-list", "--reverse", "main", "--", "README.md")) {
		t.Errorf("the README's commit must stay a root of main: %s", readme)
	}
	if copies, _ := gitsync.FindConflictCopies(dir); len(copies) != 0 {
		t.Errorf("no conflict copies, got %v", copies)
	}
	wantOutput(t, stdout, "Brought origin's notes", "Wrote .gitattributes", "Pushed to origin (branch main)")
}

func TestSyncInitTwiceChangesNothing(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, dir := newRemote(t), t.TempDir()
	writeFile(t, dir, "a.md", "a\n")
	cfg := filepath.Join(t.TempDir(), "config.json")
	if code, out, errOut := initVault(t, dir, remote, cfg); code != 0 {
		t.Fatalf("first init: exit %d\n%s\n%s", code, out, errOut)
	}
	cfgBefore, _ := os.ReadFile(cfg)
	vaultBefore, remoteBefore := treeHash(t, dir), treeHash(t, remote)

	code, stdout, stderr := initVault(t, dir, remote, cfg)
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantOutput(t, stdout, "Already set up", "nothing changed")
	for _, not := range []string{"Created", "Added", "Committed", "Pushed", "Wrote"} {
		if strings.Contains(stdout, not) {
			t.Errorf("an init with nothing to do must not say %q:\n%s", not, stdout)
		}
	}
	if cfgAfter, _ := os.ReadFile(cfg); !bytes.Equal(cfgAfter, cfgBefore) {
		t.Errorf("config changed:\n%s\nwas:\n%s", cfgAfter, cfgBefore)
	}
	if treeHash(t, remote) != remoteBefore {
		t.Error("origin changed")
	}
	if treeHash(t, dir) != vaultBefore {
		t.Error("the vault (or its .git) changed")
	}

	// A note changed since: init again syncs it, and says what it did.
	writeFile(t, dir, "a.md", "a, edited\n")
	code, stdout, stderr = initVault(t, dir, remote, cfg)
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantOutput(t, stdout, "Committed 1 file.", "Pushed to origin (branch main)", "Done")
	if strings.Contains(stdout, "Already set up") {
		t.Errorf("a sync that changed something is no no-op:\n%s", stdout)
	}
	wantFiles(t, "origin", remoteFiles(t, remote), map[string]string{"a.md": "a, edited\n", ".gitattributes": wantAttributes})
	wantSynced(t, dir, remote)
}

func TestSyncInitRefusesAnotherOrigin(t *testing.T) {
	t.Parallel()
	needGit(t)
	dir := t.TempDir()
	writeFile(t, dir, "a.md", "a\n")
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "remote", "add", "origin", "https://me:0ld-s3cret@example.com/code.git")
	cfg := filepath.Join(t.TempDir(), "config.json")
	before := treeHash(t, dir)

	code, stdout, stderr := initVault(t, dir, "https://me:n3w-s3cret@example.com/notes.git", cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantOutput(t, stderr, "already has origin https://example.com/code.git", "https://example.com/notes.git", "git remote set-url origin")
	for _, secret := range []string{"0ld-s3cret", "n3w-s3cret", "me:"} {
		if strings.Contains(stdout+stderr, secret) {
			t.Errorf("credentials printed (%q):\n%s%s", secret, stdout, stderr)
		}
	}
	if treeHash(t, dir) != before {
		t.Error("a refused init must change nothing")
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Errorf("a refused init must not write the config: %v", err)
	}
}

func TestSyncInitRefusesAMergeInProgress(t *testing.T) {
	t.Parallel()
	needGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	writeFile(t, dir, "a.md", "base\n")
	git(t, dir, "add", "a.md")
	git(t, dir, "commit", "-q", "-m", "base")
	git(t, dir, "switch", "-q", "-c", "other")
	writeFile(t, dir, "a.md", "other\n")
	git(t, dir, "commit", "-q", "-am", "other")
	git(t, dir, "switch", "-q", "main")
	writeFile(t, dir, "a.md", "main\n")
	git(t, dir, "commit", "-q", "-am", "main")
	if err := exec.Command("git", "-C", dir, "merge", "other").Run(); err == nil {
		t.Fatal("the merge must stop on its conflict")
	}
	remote, cfg := newRemote(t), filepath.Join(t.TempDir(), "config.json")
	before, remoteBefore := treeHash(t, dir), treeHash(t, remote)

	code, stdout, stderr := initVault(t, dir, remote, cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantOutput(t, stderr, "a merge is in progress", "git merge --abort")
	if treeHash(t, dir) != before {
		t.Error("a refused init must change nothing, origin included")
	}
	if treeHash(t, remote) != remoteBefore {
		t.Error("nothing may reach origin")
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Errorf("a refused init must not write the config: %v", err)
	}
}

func TestSyncInitCommitsTheUncommittedNotesOfARepository(t *testing.T) {
	t.Parallel()
	needGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	writeFile(t, dir, "a.md", "committed\n")
	git(t, dir, "add", "a.md")
	git(t, dir, "commit", "-q", "-m", "mine")
	writeFile(t, dir, "a.md", "edited, not committed\n")
	writeFile(t, dir, "new.md", "untracked\n")
	writeFile(t, dir, ".gitattributes", "*.md text\n") // the user's own
	remote := newRemote(t)

	code, stdout, stderr := initVault(t, dir, remote, filepath.Join(t.TempDir(), "c.json"))
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	want := map[string]string{"a.md": "edited, not committed\n", "new.md": "untracked\n", ".gitattributes": "*.md text\n"}
	wantFiles(t, "vault", vaultFiles(t, dir), want)
	wantFiles(t, "origin", remoteFiles(t, remote), want)
	wantSynced(t, dir, remote)
	if n := git(t, remote, "rev-list", "--count", "main"); n != "2" {
		t.Errorf("the user's commit and the init's, got %s commits", n)
	}
	wantOutput(t, stdout, "Kept your .gitattributes", "* -text", "Committed 3 files")
	if strings.Contains(stdout, "Created a git repository") {
		t.Errorf("the repository already existed:\n%s", stdout)
	}
}

func TestSyncInitNeverPrintsCredentials(t *testing.T) {
	t.Parallel()
	needGit(t)
	dir := t.TempDir()
	writeFile(t, dir, "a.md", "a\n")
	// Port 1 on this machine: refused at once, no network and no prompt.
	url := "http://me:t0ken-s3cret@127.0.0.1:1/notes.git?access_token=qu3ry-s3cret"
	cfg := filepath.Join(t.TempDir(), "config.json")

	code, stdout, stderr := initVault(t, dir, url, cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantOutput(t, stdout, "with http://127.0.0.1:1/notes.git", "Created a git repository (branch main).")
	if strings.Contains(stdout, "(branch )") {
		t.Errorf("an empty branch name must never be printed:\n%s", stdout)
	}
	wantOutput(t, stderr, "could not reach origin", "Nothing was lost", "misoku sync init <url>")
	for _, secret := range []string{"t0ken-s3cret", "qu3ry-s3cret", "me:"} {
		if strings.Contains(stdout+stderr, secret) {
			t.Errorf("credentials printed (%q):\nstdout:\n%s\nstderr:\n%s", secret, stdout, stderr)
		}
	}
	wantFiles(t, "vault", vaultFiles(t, dir), map[string]string{"a.md": "a\n"})
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Errorf("an unfinished init must not list the vault: %v", err)
	}
	// origin could not be reached, so it is not kept: init with the right
	// URL works.
	if out := git(t, dir, "remote"); out != "" {
		t.Errorf("an unreachable origin this init added must go again, got %q", out)
	}
	remote := newRemote(t)
	if code, stdout, stderr := initVault(t, dir, remote, cfg); code != 0 {
		t.Fatalf("init with the right URL: exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantFiles(t, "origin", remoteFiles(t, remote), map[string]string{"a.md": "a\n", ".gitattributes": wantAttributes})
	wantSynced(t, dir, remote)
}

func TestSyncInitNeedsOneURL(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{nil, {"a", "b"}, {""}} {
		var out, errOut bytes.Buffer
		if code := syncInit(context.Background(), args, &out, &errOut, ""); code != 2 {
			t.Errorf("%q: exit %d, want 2", args, code)
		}
		wantOutput(t, errOut.String(), "Usage: misoku sync init <url>")
	}
}

func TestUsageListsSyncInit(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	usage(&out)
	wantOutput(t, out.String(), "misoku sync init <url> [--dir <folder>]", "misoku export <note.md>", "-dir")
}

func TestSyncInitUsesOriginsDefaultBranch(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote := newRemote(t)
	work := t.TempDir()
	git(t, work, "init", "-q", "-b", "trunk")
	writeFile(t, work, "a.md", "a\n")
	writeFile(t, work, ".gitattributes", "* -text\n")
	git(t, work, "add", "-A")
	git(t, work, "commit", "-q", "-m", "seed")
	git(t, work, "push", "-q", remote, "trunk")
	git(t, remote, "symbolic-ref", "HEAD", "refs/heads/trunk")

	dir := t.TempDir()
	code, stdout, stderr := initVault(t, dir, remote, filepath.Join(t.TempDir(), "c.json"))
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if got := git(t, dir, "symbolic-ref", "--short", "HEAD"); got != "trunk" {
		t.Errorf("branch = %q, want origin's trunk", got)
	}
	if got := git(t, dir, "rev-parse", "--abbrev-ref", "@{upstream}"); got != "origin/trunk" {
		t.Errorf("upstream = %q", got)
	}
	if git(t, dir, "rev-parse", "HEAD") != git(t, remote, "rev-parse", "trunk") {
		t.Error("the vault must be at origin's trunk")
	}
	wantFiles(t, "vault", vaultFiles(t, dir), map[string]string{"a.md": "a\n", ".gitattributes": "* -text\n"})
	wantOutput(t, stdout, "Created a git repository (branch trunk)")
	if strings.Contains(stdout, "Kept your .gitattributes") {
		t.Errorf("a .gitattributes with * -text needs no advice:\n%s", stdout)
	}
	if strings.Contains(git(t, remote, "for-each-ref"), "refs/heads/main") {
		t.Error("nothing may be pushed to a main origin does not use")
	}
}

func TestSyncInitRefusesAFolderInsideAnotherRepository(t *testing.T) {
	t.Parallel()
	needGit(t)
	outer := t.TempDir()
	git(t, outer, "init", "-q", "-b", "main")
	dir := filepath.Join(outer, "notes")
	writeFile(t, dir, "a.md", "a\n")
	remote, cfg := newRemote(t), filepath.Join(t.TempDir(), "config.json")
	before := treeHash(t, outer)

	code, stdout, stderr := initVault(t, dir, remote, cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantOutput(t, stderr, "inside the git repository at", "git init")
	if strings.Contains(stderr, "open ") {
		t.Errorf("init must not advise opening the outer repository:\n%s", stderr)
	}
	if treeHash(t, outer) != before {
		t.Error("a refused init must change nothing")
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Errorf("a refused init must not write the config: %v", err)
	}
}

func TestSyncInitWaitsForAnotherWindow(t *testing.T) {
	t.Parallel()
	needGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	writeFile(t, dir, "a.md", "a\n")
	lock, ok, err := gitsync.TryLock(filepath.Join(dir, ".git", "misoku-sync.lock"))
	if err != nil || !ok {
		t.Fatalf("TryLock: %v %v", ok, err)
	}
	defer lock.Unlock()
	remote, cfg := newRemote(t), filepath.Join(t.TempDir(), "config.json")

	code, stdout, stderr := initVault(t, dir, remote, cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantOutput(t, stderr, "another Misoku window is syncing this vault")
	if out := git(t, dir, "remote"); out != "" {
		t.Errorf("origin must not be added while another window syncs: %q", out)
	}
	if n := git(t, dir, "rev-list", "--all", "--count"); n != "0" {
		t.Errorf("nothing may be committed, got %s commits", n)
	}
}

// TestSyncInitReportsTheFilesThatDoNotSync checks that init tells, one
// line each and in the app's words, about every file that does not sync
// as a plain file (S15), not only the big ones.
func TestSyncInitReportsTheFilesThatDoNotSync(t *testing.T) {
	t.Parallel()
	notices := []gitsync.Notice{
		{Kind: gitsync.NestedRepo, Path: "code", LeftOut: true},
		{Kind: gitsync.Submodule, Path: "lib"},
		{Kind: gitsync.LinkOutside, Path: "out.md", Detail: "/etc/hosts"},
		{Kind: gitsync.WindowsName, Path: "a:b.md", Detail: `":"`, LeftOut: true},
		{Kind: gitsync.HeldDeletion, Path: "old.md"},
	}
	var res gitsync.InitResult
	res.Sync.Commit.Notices = notices
	var out bytes.Buffer
	report(&out, res, "url")
	lines := strings.Split(out.String(), "\n")
	for _, n := range notices {
		if !slices.Contains(lines, n.String()) {
			t.Errorf("no line %q in:\n%s", n.String(), out.String())
		}
	}
}

// TestSyncInitReportsANestedRepository runs the real init on a vault that
// holds a repository of its own: it is left out, and init says so.
func TestSyncInitReportsANestedRepository(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, dir := newRemote(t), t.TempDir()
	writeFile(t, dir, "a.md", "a\n")
	writeFile(t, dir, "code/main.go", "package main\n")
	git(t, filepath.Join(dir, "code"), "init", "-q")

	code, stdout, stderr := initVault(t, dir, remote, filepath.Join(t.TempDir(), "c.json"))
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	want := gitsync.Notice{Kind: gitsync.NestedRepo, Path: "code", LeftOut: true}.String()
	wantOutput(t, stdout, want)
	if _, ok := remoteFiles(t, remote)["code/main.go"]; ok {
		t.Errorf("the nested repository must stay out of origin")
	}
}
