package gitsync_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// nestedRepo makes a git repository at name inside dir, with one commit
// when committed is true.
func nestedRepo(t *testing.T, dir, name string, committed bool) {
	t.Helper()
	write(t, dir, name+"/x.md", "inner note\n")
	git(t, filepath.Join(dir, filepath.FromSlash(name)), "init", "-q")
	if committed {
		git(t, filepath.Join(dir, filepath.FromSlash(name)), "add", "-A")
		git(t, filepath.Join(dir, filepath.FromSlash(name)), "commit", "-q", "-m", "inner")
	}
}

func wantNotices(t *testing.T, got []gitsync.Notice, want ...gitsync.Notice) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("Notices = %+v,\nwant %+v", got, want)
	}
	for _, n := range got {
		if !strings.Contains(n.String(), n.Path) {
			t.Errorf("the warning %q must name %s", n.String(), n.Path)
		}
	}
}

// symlink makes a link or skips the test where links cannot be made.
// Git for Windows stores links only with core.symlinks, off by default, so
// the test is skipped there too.
func symlink(t *testing.T, target, link string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("git for Windows stores symbolic links only with core.symlinks, off by default")
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot make symbolic links here: %v", err)
	}
}

// S15: a repository inside the vault is left out with a warning: git would
// only record its commit id (or fail outright on one with no commit), so
// committing it would sync nothing of it and could stop every sync.
func TestCommitVaultLeavesOutANestedRepository(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	nestedRepo(t, dir, "code", true)
	nestedRepo(t, dir, "empty", false)
	nestedRepo(t, dir, "deep/inner", true)
	write(t, dir, "deep/plain.md", "plain\n")
	write(t, dir, "n.md", "note\n")

	res := commitVault(t, dir)
	if !res.Committed || res.Files != 2 {
		t.Fatalf("want the two notes committed, got %+v", res)
	}
	wantTree(t, dir, "deep/plain.md", "n.md")
	want := []gitsync.Notice{
		{Kind: gitsync.NestedRepo, Path: "code", LeftOut: true},
		{Kind: gitsync.NestedRepo, Path: "deep/inner", LeftOut: true},
		{Kind: gitsync.NestedRepo, Path: "empty", LeftOut: true},
	}
	wantNotices(t, res.Notices, want...)
	if got := read(t, dir, "code/x.md"); got != "inner note\n" {
		t.Errorf("the nested repository's file must stay as it was, got %q", got)
	}

	// Still there next time: still left out, still said.
	res = commitVault(t, dir)
	if res.Committed {
		t.Errorf("nothing new to commit, got %+v", res)
	}
	wantNotices(t, res.Notices, want...)
}

// S15: a submodule the vault already tracks (a gitlink, made by `git add`
// on a nested repository) keeps its place, with a warning that its files
// do not sync.
func TestCommitVaultWarnsAboutASubmodule(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	nestedRepo(t, dir, "lib", true)
	write(t, dir, "n.md", "v1\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "by hand")

	write(t, dir, "n.md", "v2\n")
	res := commitVault(t, dir)
	if !res.Committed || res.Files != 1 {
		t.Fatalf("want n.md committed, got %+v", res)
	}
	wantNotices(t, res.Notices, gitsync.Notice{Kind: gitsync.Submodule, Path: "lib"})
	if mode := strings.Fields(git(t, dir, "ls-tree", "HEAD", "lib"))[0]; mode != "160000" {
		t.Errorf("lib must stay a submodule, mode %s", mode)
	}
}

// S15: git stores a symbolic link, not what it points to. Links inside the
// vault sync as links with no warning; a link out of the vault (climbing
// out or absolute) syncs too, with a warning that its target does not.
func TestCommitVaultKeepsLinksAndWarnsAboutLinksOutside(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	outside := filepath.Join(t.TempDir(), "secret.md")
	write(t, filepath.Dir(outside), "secret.md", "not in the vault\n")
	write(t, dir, "notes/a.md", "a\n")
	symlink(t, "notes/a.md", filepath.Join(dir, "inside.md"))
	symlink(t, "notes", filepath.Join(dir, "folder"))
	symlink(t, "../outside.md", filepath.Join(dir, "up.md"))
	symlink(t, outside, filepath.Join(dir, "abs.md"))

	res := commitVault(t, dir)
	if !res.Committed {
		t.Fatalf("want a commit, got %+v", res)
	}
	wantTree(t, dir, "abs.md", "folder", "inside.md", "notes/a.md", "up.md")
	for _, link := range []string{"abs.md", "folder", "inside.md", "up.md"} {
		if mode := strings.Fields(git(t, dir, "ls-tree", "HEAD", link))[0]; mode != "120000" {
			t.Errorf("%s must be stored as a link, mode %s", link, mode)
		}
	}
	if got := git(t, dir, "cat-file", "-p", "HEAD:abs.md"); got != outside {
		t.Errorf("the link must hold its target, not the file: %q", got)
	}
	want := []gitsync.Notice{
		{Kind: gitsync.LinkOutside, Path: "abs.md", Detail: outside},
		{Kind: gitsync.LinkOutside, Path: "up.md", Detail: "../outside.md"},
	}
	wantNotices(t, res.Notices, want...)

	// Tracked links are warned about on every commit, not only when new.
	write(t, dir, "notes/a.md", "a2\n")
	res = commitVault(t, dir)
	wantNotices(t, res.Notices, want...)
}

// S15: a new file whose name Windows cannot check out is left out with a
// warning, even if staged by hand: committed, it would stop every sync on
// a Windows computer. The file stays on disk, as it was.
func TestCommitVaultLeavesOutNamesWindowsCannotUse(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("these names cannot be created on Windows")
	}
	isolate(t)
	dir := newRepo(t)
	bad := map[string]string{
		"meeting 10:30.md":  `":"`,
		"star*.md":          `"*"`,
		`quote"d.md`:        `"\""`,
		"why?.md":           `"?"`,
		`back\slash.md`:     `"\\"`,
		"dot.":              "it ends with a dot or a space",
		"space ":            "it ends with a dot or a space",
		"CON.md":            "CON is reserved on Windows",
		"nul":               "nul is reserved on Windows",
		"Com1 .txt":         "Com1 is reserved on Windows",
		"folder:x/inner.md": `":"`,
	}
	for name := range bad {
		write(t, dir, name, "keep me\n")
	}
	write(t, dir, "ok.md", "fine\n")
	git(t, dir, "add", "--", "star*.md") // staged by hand: still left out

	res := commitVault(t, dir)
	if !res.Committed || res.Files != 1 {
		t.Fatalf("want only ok.md committed, got %+v", res)
	}
	wantTree(t, dir, "ok.md")
	var want []gitsync.Notice
	for name, why := range bad {
		want = append(want, gitsync.Notice{Kind: gitsync.WindowsName, Path: name, Detail: why, LeftOut: true})
		if got := read(t, dir, name); got != "keep me\n" {
			t.Errorf("%s must stay on disk as it was, got %q", name, got)
		}
	}
	slices.SortFunc(want, func(a, b gitsync.Notice) int { return strings.Compare(a.Path, b.Path) })
	wantNotices(t, res.Notices, want...)
}

// S15: a file with such a name the vault already tracks keeps syncing
// (leaving it out would freeze it), with a warning to rename it.
func TestCommitVaultWarnsAboutATrackedWindowsName(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("this name cannot be created on Windows")
	}
	isolate(t)
	dir := newRepo(t)
	write(t, dir, "x:y.md", "v1\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "by hand")

	write(t, dir, "x:y.md", "v2\n")
	res := commitVault(t, dir)
	if !res.Committed || res.Files != 1 {
		t.Fatalf("want x:y.md committed, got %+v", res)
	}
	if got := git(t, dir, "show", "HEAD:x:y.md"); got != "v2" {
		t.Errorf("x:y.md committed as %q, want v2", got)
	}
	wantNotices(t, res.Notices, gitsync.Notice{Kind: gitsync.WindowsName, Path: "x:y.md", Detail: `":"`})

	// Once deleted, its deletion syncs and the warning goes.
	if err := os.Remove(filepath.Join(dir, "x:y.md")); err != nil {
		t.Fatal(err)
	}
	res = commitVault(t, dir)
	if !res.Committed {
		t.Fatalf("want the deletion committed, got %+v", res)
	}
	wantNotices(t, res.Notices)
}

// D7: the size limit holds inside a folder git has not seen yet, too.
func TestCommitVaultLeavesOutABigFileInANewFolder(t *testing.T) {
	isolate(t)
	withLimit(t, 10)
	dir := newRepo(t)
	write(t, dir, "new/deep/big.png", "eleven byte")
	write(t, dir, "new/deep/n.md", "n")

	res := commitVault(t, dir)
	wantTree(t, dir, "new/deep/n.md")
	if want := []gitsync.BigFile{{Path: "new/deep/big.png", Size: 11}}; !slices.Equal(res.Skipped, want) {
		t.Errorf("Skipped = %+v, want %+v", res.Skipped, want)
	}
}

// S15: when origin holds a file at the path of one this vault leaves out,
// the sync stops naming it, instead of retrying forever; nothing on disk
// changes.
func TestSyncNamesALeftOutFileOriginAlsoHas(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		mine   func(t *testing.T, dir string)
		theirs string
		named  string
	}{
		{"windows name", func(t *testing.T, dir string) { write(t, dir, "a:b.md", "mine\n") }, "a:b.md", "a:b.md"},
		{"nested repository", func(t *testing.T, dir string) { nestedRepo(t, dir, "code", true) }, "code/x.md", "code"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == "windows" && strings.Contains(c.theirs, ":") {
				t.Skip("a name with ':' cannot be created on Windows")
			}
			_, laptop, desk := pair(t, map[string]string{"n.md": "n\n"})
			c.mine(t, laptop)
			write(t, desk, c.theirs, "theirs\n")
			git(t, desk, "add", "-A")
			git(t, desk, "commit", "-q", "-m", "desk")
			git(t, desk, "push", "-q", "origin", "HEAD")
			before := read(t, laptop, c.theirs)

			res := syncVault(t, laptop, "laptop", false)
			wantOutcome(t, res, gitsync.SyncFailed)
			if !strings.Contains(res.Err.Error(), c.named) || !strings.Contains(res.Err.Error(), "rename") {
				t.Errorf("the error must name %s and the fix: %v", c.named, res.Err)
			}
			if got := read(t, laptop, c.theirs); got != before {
				t.Errorf("the vault's %s changed: %q", c.theirs, got)
			}
		})
	}
}

// S15: while a new file is left out for its name, no deletion is
// committed (it may be that file's old name); once the file has a valid
// name, the held deletions go through.
func TestCommitVaultHoldsBackDeletionsWhileANameIsLeftOut(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("this name cannot be created on Windows")
	}
	isolate(t)
	dir := newRepo(t)
	write(t, dir, "n.md", "n\n")
	write(t, dir, "other.md", "other\n")
	commitVault(t, dir)
	if err := os.Remove(filepath.Join(dir, "other.md")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "x?.md", "x\n")

	res := commitVault(t, dir)
	if res.Committed {
		t.Errorf("nothing may be committed, got %+v", res)
	}
	wantTree(t, dir, "n.md", "other.md")
	wantNotices(t, res.Notices,
		gitsync.Notice{Kind: gitsync.HeldDeletion, Path: "other.md"},
		gitsync.Notice{Kind: gitsync.WindowsName, Path: "x?.md", Detail: `"?"`, LeftOut: true})

	if err := os.Rename(filepath.Join(dir, "x?.md"), filepath.Join(dir, "x.md")); err != nil {
		t.Fatal(err)
	}
	res = commitVault(t, dir)
	wantTree(t, dir, "n.md", "x.md")
	wantNotices(t, res.Notices)
}
