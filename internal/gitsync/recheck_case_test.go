package gitsync_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// csDir is a folder on a case-sensitive disk (the "Linux" side), from
// MISOKU_CS_DIR; tests that need one skip without it.
func csDir(t *testing.T) string {
	t.Helper()
	root := os.Getenv("MISOKU_CS_DIR")
	if root == "" {
		t.Skip("MISOKU_CS_DIR not set")
	}
	dir, err := os.MkdirTemp(root, "cs-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func cloneIn(t *testing.T, parent, remote string) string {
	t.Helper()
	git(t, parent, "clone", "-q", remote, "vault")
	return filepath.Join(parent, "vault")
}

// originContents is every blob content on origin's main.
func originContents(t *testing.T, remote string) map[string]bool {
	t.Helper()
	set := map[string]bool{}
	for _, name := range strings.Split(git(t, remote, "ls-tree", "-r", "--name-only", "-z", "main"), "\x00") {
		if name != "" {
			set[git(t, remote, "show", "main:"+name)+"\n"] = true
		}
	}
	return set
}

func wantOrigin(t *testing.T, remote string, versions ...string) {
	t.Helper()
	have := originContents(t, remote)
	for _, v := range versions {
		if !have[v] {
			t.Errorf("origin lost %q (tree %s)", v, git(t, remote, "ls-tree", "-r", "--name-only", "main"))
		}
	}
}

func logSync(t *testing.T, dir, name string) gitsync.SyncResult {
	t.Helper()
	res := syncVault(t, dir, name, false)
	var ns []string
	for _, n := range res.Commit.Notices {
		ns = append(ns, n.String())
	}
	t.Logf("%s: %v err=%v conflicts=%v notices=%v status=%q tree=%v", name, res.Outcome, res.Err, res.Conflicts, ns, git(t, dir, "status", "--porcelain"), tree(t, dir))
	return res
}

// The original trace, both on this (case-insensitive) disk: stable, nothing lost.
func TestRecheck2CaseClashStable(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "Ideas/a.md", "in folder\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "ideas", "a file\n")
	for i := 0; i < 4; i++ {
		logSync(t, desk, "desk")
		logSync(t, laptop, "laptop")
	}
	wantOrigin(t, remote, "in folder\n", "a file\n")
	wantAllVersions(t, laptop, "in folder\n", "a file\n")
	wantAllVersions(t, desk, "in folder\n", "a file\n")
	if a, b := tree(t, laptop), tree(t, desk); !slices.Equal(a, b) {
		t.Errorf("trees differ %v %v", a, b)
	}
}

// Same, with the folder made on Linux and the file on macOS, and the
// reverse.
func TestRecheck2CaseClashLinuxMac(t *testing.T) {
	for _, linuxHasFolder := range []bool{true, false} {
		isolate(t)
		remote := newRemote(t)
		linux := cloneIn(t, csDir(t), remote)
		mac := cloneIn(t, t.TempDir(), remote)
		write(t, linux, "seed.md", "s\n")
		wantOutcome(t, syncVault(t, linux, "linux", false), gitsync.Synced)
		wantOutcome(t, syncVault(t, mac, "mac", false), gitsync.Synced)
		if linuxHasFolder {
			write(t, linux, "Ideas/a.md", "in folder\n")
			write(t, mac, "ideas", "a file\n")
		} else {
			write(t, linux, "ideas", "a file\n")
			write(t, mac, "Ideas/a.md", "in folder\n")
		}
		for i := 0; i < 3; i++ {
			logSync(t, linux, "linux")
			logSync(t, mac, "mac")
		}
		wantOrigin(t, remote, "in folder\n", "a file\n")
		wantAllVersions(t, linux, "in folder\n", "a file\n")
		wantAllVersions(t, mac, "in folder\n", "a file\n")
	}
}

// a.md and A.md made on Linux by another tool, arriving on macOS by a
// fast-forward: nothing overwritten on origin; an edit on macOS is held;
// renaming one on Linux resolves it.
func TestRecheck2LinuxDuplicatesReachMac(t *testing.T) {
	isolate(t)
	remote := newRemote(t)
	linux := cloneIn(t, csDir(t), remote)
	mac := cloneIn(t, t.TempDir(), remote)
	write(t, linux, "seed.md", "s\n")
	wantOutcome(t, syncVault(t, linux, "linux", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, mac, "mac", false), gitsync.Synced)
	write(t, linux, "a.md", "lower\n")
	write(t, linux, "A.md", "upper\n")
	git(t, linux, "add", "-A")
	git(t, linux, "commit", "-q", "-m", "other tool")
	git(t, linux, "push", "-q", "origin", "main")
	for i := 0; i < 2; i++ {
		logSync(t, mac, "mac")
	}
	wantOrigin(t, remote, "lower\n", "upper\n")
	// a real edit on macOS to the one file it shows
	b, _ := os.ReadFile(filepath.Join(mac, "a.md"))
	t.Logf("mac shows a.md = %q", b)
	write(t, mac, "a.md", string(b)+"mac edit\n")
	res := logSync(t, mac, "mac")
	wantOrigin(t, remote, "lower\n", "upper\n")
	if len(res.Commit.Notices) == 0 {
		t.Error("the held edit carries no notice")
	}
	wantAllVersions(t, mac, string(b)+"mac edit\n")
	// Linux renames one: resolved
	logSync(t, linux, "linux")
	if err := os.Rename(filepath.Join(linux, "A.md"), filepath.Join(linux, "Upper.md")); err != nil {
		t.Fatal(err)
	}
	logSync(t, linux, "linux")
	res = logSync(t, mac, "mac")
	if res.Outcome != gitsync.Synced {
		// the user follows the message on macOS: renames the file it shows
		if err := os.Rename(filepath.Join(mac, "a.md"), filepath.Join(mac, "b.md")); err != nil {
			t.Fatal(err)
		}
		res = logSync(t, mac, "mac")
	}
	res = logSync(t, mac, "mac")
	logSync(t, linux, "linux")
	wantAllVersions(t, linux, "upper\n", string(b)+"mac edit\n")
	wantAllVersions(t, mac, "upper\n", string(b)+"mac edit\n")
	wantOrigin(t, remote, "upper\n", string(b)+"mac edit\n")
	if len(res.Commit.Notices) != 0 {
		t.Errorf("still held after the rename: %v", res.Commit.Notices)
	}
	wantNoMarkers(t, mac, "HEAD")
}

// Misoku on Linux: a new A.md next to tracked a.md stays out with a
// notice, and a case-only rename foo.md -> Foo.md syncs normally.
func TestRecheck2LinuxCaseOnlyRename(t *testing.T) {
	isolate(t)
	remote := newRemote(t)
	linux := cloneIn(t, csDir(t), remote)
	mac := cloneIn(t, t.TempDir(), remote)
	write(t, linux, "foo.md", "foo\n")
	write(t, linux, "a.md", "a\n")
	wantOutcome(t, syncVault(t, linux, "linux", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, mac, "mac", false), gitsync.Synced)
	if err := os.Rename(filepath.Join(linux, "foo.md"), filepath.Join(linux, "Foo.md")); err != nil {
		t.Fatal(err)
	}
	res := logSync(t, linux, "linux")
	if len(res.Commit.Notices) != 0 {
		t.Errorf("a case-only rename is held: %v", res.Commit.Notices)
	}
	if !slices.Contains(tree(t, linux), "Foo.md") || slices.Contains(tree(t, linux), "foo.md") {
		t.Errorf("rename not committed: %v", tree(t, linux))
	}
	for i := 0; i < 2; i++ {
		logSync(t, mac, "mac")
	}
	wantAllVersions(t, mac, "foo\n", "a\n")
	write(t, mac, "a.md", "a\nmac\n")
	logSync(t, mac, "mac")
	logSync(t, linux, "linux")
	wantAllVersions(t, linux, "foo\n", "a\nmac\n")
	// new duplicate on Linux
	write(t, linux, "A.md", "dup\n")
	res = logSync(t, linux, "linux")
	if len(res.Commit.Notices) == 0 || slices.Contains(tree(t, linux), "A.md") {
		t.Errorf("A.md must stay out with a notice: %v %v", res.Commit.Notices, tree(t, linux))
	}
	wantAllVersions(t, linux, "dup\n", "a\nmac\n")
}

// Unicode case: Ñ.md and ñ.md made on Linux by another tool reach macOS.
func TestRecheck2UnicodeCase(t *testing.T) {
	isolate(t)
	remote := newRemote(t)
	linux := cloneIn(t, csDir(t), remote)
	mac := cloneIn(t, t.TempDir(), remote)
	write(t, linux, "seed.md", "s\n")
	wantOutcome(t, syncVault(t, linux, "linux", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, mac, "mac", false), gitsync.Synced)
	write(t, linux, "Ñandú.md", "upper n\n")
	write(t, linux, "ñandú.md", "lower n\n")
	git(t, linux, "add", "-A")
	git(t, linux, "commit", "-q", "-m", "other tool")
	git(t, linux, "push", "-q", "origin", "main")
	for i := 0; i < 2; i++ {
		logSync(t, mac, "mac")
	}
	wantOrigin(t, remote, "upper n\n", "lower n\n")
	// Misoku on Linux: new É.md next to tracked é.md
	write(t, linux, "é.md", "e\n")
	logSync(t, linux, "linux")
	write(t, linux, "É.md", "E\n")
	res := logSync(t, linux, "linux")
	t.Logf("É notices: %v", res.Commit.Notices)
	logSync(t, mac, "mac")
	wantOrigin(t, remote, "upper n\n", "lower n\n", "e\n")
}

// Linux-made a.md/A.md on macOS; the macOS user renames the one file it
// shows (with or without editing it first). "upper" must survive.
func TestRecheck2MacRenamesTheDuplicate(t *testing.T) {
	for _, edit := range []bool{false, true} {
		isolate(t)
		remote := newRemote(t)
		linux := cloneIn(t, csDir(t), remote)
		mac := cloneIn(t, t.TempDir(), remote)
		write(t, linux, "seed.md", "s\n")
		wantOutcome(t, syncVault(t, linux, "linux", false), gitsync.Synced)
		wantOutcome(t, syncVault(t, mac, "mac", false), gitsync.Synced)
		write(t, linux, "a.md", "lower\n")
		write(t, linux, "A.md", "upper\n")
		git(t, linux, "add", "-A")
		git(t, linux, "commit", "-q", "-m", "other tool")
		git(t, linux, "push", "-q", "origin", "main")
		logSync(t, mac, "mac")
		b, _ := os.ReadFile(filepath.Join(mac, "a.md"))
		if edit {
			write(t, mac, "a.md", string(b)+"mac edit\n")
			b = append(b, "mac edit\n"...)
		}
		if err := os.Rename(filepath.Join(mac, "a.md"), filepath.Join(mac, "b.md")); err != nil {
			t.Fatal(err)
		}
		logSync(t, mac, "mac")
		logSync(t, linux, "linux")
		logSync(t, mac, "mac")
		wantOrigin(t, remote, "lower\n", "upper\n", string(b))
		wantAllVersions(t, linux, "lower\n", "upper\n", string(b))
	}
}
