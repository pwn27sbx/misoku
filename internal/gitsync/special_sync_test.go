package gitsync_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

func originTree(t *testing.T, laptop string) string {
	t.Helper()
	git(t, laptop, "fetch", "-q", "origin")
	return git(t, laptop, "ls-tree", "-r", "origin/HEAD")
}

func skipColonOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a name with ':' cannot be created on Windows")
	}
}

// Renaming a synced note to a name Windows cannot use must not push
// a deletion that removes the note from every other device.
func TestSyncRenameToAWindowsNameKeepsTheNoteOnOtherDevices(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	_, laptop, desk := pair(t, map[string]string{"Meeting.md": "agenda\n"})
	if err := os.Rename(filepath.Join(laptop, "Meeting.md"), filepath.Join(laptop, "Meeting 10:30.md")); err != nil {
		t.Fatal(err)
	}
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, laptop, "Meeting 10:30.md", "agenda\n")
	if _, err := os.Stat(filepath.Join(desk, "Meeting.md")); err != nil {
		t.Errorf("desk lost the note that laptop renamed to a Windows-invalid name: %v\norigin tree:\n%s", err, originTree(t, laptop))
	}
}

// A nested repository committed as a gitlink by an older Misoku
// keeps syncing as a gitlink; nothing is deleted on origin or elsewhere,
// and the nested repository's files are untouched.
func TestSyncKeepsAGitlinkAnOlderMisokuCommitted(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"n.md": "n\n"})
	nestedRepo(t, laptop, "code", true)
	git(t, laptop, "add", "code") // what baseline CommitVault did
	git(t, laptop, "commit", "-q", "-m", "old misoku")
	git(t, laptop, "push", "-q", "origin", "HEAD")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	for i := 0; i < 3; i++ {
		res := syncVault(t, laptop, "laptop", false)
		wantOutcome(t, res, gitsync.Synced)
		wantFile(t, laptop, "code/x.md", "inner note\n")
		found := false
		for _, n := range res.Commit.Notices {
			found = found || (n.Kind == gitsync.Submodule && n.Path == "code")
		}
		if !found {
			t.Errorf("sync %d: notices %+v, want a Submodule notice for code", i, res.Commit.Notices)
		}
	}
	// A new commit inside the nested repository.
	write(t, laptop, "code/y.md", "y\n")
	git(t, filepath.Join(laptop, "code"), "add", "-A")
	git(t, filepath.Join(laptop, "code"), "commit", "-q", "-m", "y")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	if !strings.Contains(originTree(t, laptop), "160000 commit") {
		t.Errorf("the gitlink left origin:\n%s", originTree(t, laptop))
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "n.md", "n\n")
	wantFile(t, laptop, "code/y.md", "y\n")
}

// A folder already synced becomes a git repository (git init inside):
// its tracked notes are never deleted on origin.
func TestSyncNeverDeletesNotesOfAFolderThatBecameARepository(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"proj/a.md": "a\n", "n.md": "n\n"})
	git(t, filepath.Join(laptop, "proj"), "init", "-q")
	write(t, laptop, "proj/b.md", "b\n")
	write(t, laptop, "proj/a.md", "a2\n")
	for i := 0; i < 2; i++ {
		res := syncVault(t, laptop, "laptop", false)
		wantOutcome(t, res, gitsync.Synced)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "proj/a.md", "a2\n")
	wantFile(t, laptop, "proj/b.md", "b\n")
}

// A left-out Windows name stays byte-identical over many syncs, never
// reaches another device; renaming it makes it sync and clears the warning.
func TestSyncWindowsNameLeftOutUntilRenamed(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	_, laptop, desk := pair(t, map[string]string{"n.md": "n\n"})
	content := "bytes \x00\x01 ñ\r\n"
	write(t, laptop, "dir/a:b.md", content)
	write(t, laptop, "trail./c.md", "c\n")
	write(t, laptop, "nul.txt", "nul\n")
	for i := 0; i < 3; i++ {
		res := syncVault(t, laptop, "laptop", false)
		wantOutcome(t, res, gitsync.Synced)
		if len(res.Commit.Notices) != 3 {
			t.Errorf("sync %d notices %+v, want 3", i, res.Commit.Notices)
		}
		wantFile(t, laptop, "dir/a:b.md", content)
		wantFile(t, laptop, "trail./c.md", "c\n")
		wantFile(t, laptop, "nul.txt", "nul\n")
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	for _, n := range []string{"dir/a:b.md", "trail./c.md", "nul.txt"} {
		if _, err := os.Lstat(filepath.Join(desk, filepath.FromSlash(n))); err == nil {
			t.Errorf("%s reached desk", n)
		}
	}
	os.Rename(filepath.Join(laptop, "dir/a:b.md"), filepath.Join(laptop, "dir/a-b.md"))
	res := syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.Synced)
	if len(res.Commit.Notices) != 2 {
		t.Errorf("after rename notices %+v, want 2", res.Commit.Notices)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "dir/a-b.md", content)
}

// A tracked Windows name (committed by an older Misoku) that is
// deleted syncs the deletion and the warning goes away.
func TestSyncDeletesATrackedWindowsName(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	_, laptop, desk := pair(t, map[string]string{"n.md": "n\n"})
	write(t, laptop, "x?.md", "x\n")
	git(t, laptop, "add", "-A")
	git(t, laptop, "commit", "-q", "-m", "old")
	git(t, laptop, "push", "-q", "origin", "HEAD")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "x?.md", "x\n")
	res := syncVault(t, desk, "desk", false)
	if len(res.Commit.Notices) != 1 || res.Commit.Notices[0].LeftOut {
		t.Errorf("tracked name notices %+v", res.Commit.Notices)
	}
	os.Remove(filepath.Join(laptop, "x?.md"))
	res = syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.Synced)
	if len(res.Commit.Notices) != 0 {
		t.Errorf("after delete notices %+v", res.Commit.Notices)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	if _, err := os.Lstat(filepath.Join(desk, "x?.md")); err == nil {
		t.Errorf("deletion did not reach desk")
	}
}

// Unicode, NFD and case-only names are never flagged.
func TestCommitVaultNeverFlagsUnicodeOrCaseOnlyNames(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	names := []string{"Reunión.md", "Reunión NFD.md", "ñandú/Árbol.md", "日本語.md", "emoji 🎉.md", "CONSOLE.md", "com10.md", "a.con.md"}
	for _, n := range names {
		write(t, dir, n, n)
	}
	res := commitVault(t, dir)
	if len(res.Notices) != 0 {
		t.Errorf("notices %+v, want none", res.Notices)
	}
	if got := len(tree(t, dir)); got != len(names) {
		t.Errorf("tree %v", tree(t, dir))
	}
	// case-only rename
	git(t, dir, "mv", "Reunión.md", "reunión.md")
	if r := commitVault(t, dir); len(r.Notices) != 0 {
		t.Errorf("case rename notices %+v", r.Notices)
	}
}

// Links are stored as links; the target's content never enters git.
func TestCommitVaultNeverUploadsALinksTarget(t *testing.T) {
	t.Parallel()
	isolate(t)
	outside := t.TempDir()
	secret := "SECRET-OUTSIDE-CONTENT-123"
	os.WriteFile(filepath.Join(outside, "s.md"), []byte(secret), 0o644)
	os.Mkdir(filepath.Join(outside, "d"), 0o755)
	os.WriteFile(filepath.Join(outside, "d", "t.md"), []byte(secret), 0o644)
	dir := newRepo(t)
	write(t, dir, "in.md", "in\n")
	symlink(t, "in.md", filepath.Join(dir, "rel.md"))
	symlink(t, filepath.Join(outside, "s.md"), filepath.Join(dir, "abs.md"))
	symlink(t, filepath.Join(outside, "d"), filepath.Join(dir, "absdir"))
	rel, _ := filepath.Rel(filepath.Join(dir, "sub"), filepath.Join(outside, "s.md"))
	os.Mkdir(filepath.Join(dir, "sub"), 0o755)
	symlink(t, rel, filepath.Join(dir, "sub", "up.md"))
	symlink(t, "../in.md", filepath.Join(dir, "sub", "upin.md"))
	res := commitVault(t, dir)
	var kinds []string
	for _, n := range res.Notices {
		kinds = append(kinds, n.Path)
	}
	if strings.Join(kinds, ",") != "abs.md,absdir,sub/up.md" {
		t.Errorf("notices %+v", res.Notices)
	}
	ls := git(t, dir, "ls-tree", "-r", "HEAD")
	if strings.Count(ls, "120000") != 5 {
		t.Errorf("links not stored as links:\n%s", ls)
	}
	if out, err := exec.Command("git", "-C", dir, "grep", "-c", secret, "HEAD").CombinedOutput(); err == nil {
		t.Errorf("the target's content entered git: %s", out)
	}
	if got := read(t, dir, "abs.md"); got != secret {
		t.Errorf("outside file changed")
	}
}

// A rename staged by hand (git mv) to a name Windows cannot use is held
// back whole: the new name stays out and the old one stays tracked, so no
// device loses the note.
func TestCommitVaultHoldsBackAStagedRenameToAWindowsName(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	isolate(t)
	dir := newRepo(t)
	write(t, dir, "a.md", "a\n")
	commitVault(t, dir)
	git(t, dir, "mv", "a.md", "a:b.md")
	res := commitVault(t, dir)
	if res.Committed {
		t.Errorf("nothing may be committed, got %+v", res)
	}
	wantTree(t, dir, "a.md")
	wantNotices(t, res.Notices,
		gitsync.Notice{Kind: gitsync.HeldDeletion, Path: "a.md"},
		gitsync.Notice{Kind: gitsync.WindowsName, Path: "a:b.md", Detail: `":"`, LeftOut: true})
	wantFile(t, dir, "a:b.md", "a\n")
	if !strings.Contains(res.Notices[1].String(), "keeps syncing under its old name") {
		t.Errorf("the warning must say the note keeps its old name: %s", res.Notices[1])
	}
}

// Origin's file at a path inside a nested repository this vault leaves
// out lands next to that repository's own files, which stay untouched
// (known limit: git cannot tell the two apart).
func TestSyncOriginFileLandsBesideANestedRepositorysFiles(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"n.md": "n\n"})
	nestedRepo(t, laptop, "code", true)
	write(t, desk, "code/other.md", "theirs\n")
	git(t, desk, "add", "-A")
	git(t, desk, "commit", "-q", "-m", "desk")
	git(t, desk, "push", "-q", "origin", "HEAD")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantFile(t, laptop, "code/x.md", "inner note\n")
	wantFile(t, laptop, "code/other.md", "theirs\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantFile(t, laptop, "code/x.md", "inner note\n")
}

// A big file plus a nested repo plus a Windows name do not stop the
// sync of ordinary notes.
func TestSyncOrdinaryNotesBesideSpecialFiles(t *testing.T) {
	// Not parallel: withLimit changes a package variable.
	skipColonOnWindows(t)
	withLimit(t, 10)
	_, laptop, desk := pair(t, map[string]string{"n.md": "n\n"})
	nestedRepo(t, laptop, "code", false)
	write(t, laptop, "big.bin", strings.Repeat("x", 100))
	write(t, laptop, "a|b.md", "pipe\n")
	write(t, laptop, "ok.md", "ok\n")
	res := syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "ok.md", "ok\n")
	wantFile(t, laptop, "big.bin", strings.Repeat("x", 100))
	wantFile(t, laptop, "a|b.md", "pipe\n")
}

// Renaming a synced folder to a name Windows cannot use must not
// delete its notes from every other device.
func TestSyncFolderRenameToAWindowsNameKeepsNotesOnOtherDevices(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	_, laptop, desk := pair(t, map[string]string{"Ideas/a.md": "a\n", "Ideas/b.md": "b\n"})
	if err := os.Rename(filepath.Join(laptop, "Ideas"), filepath.Join(laptop, "Ideas?")); err != nil {
		t.Fatal(err)
	}
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	for _, n := range []string{"Ideas/a.md", "Ideas/b.md"} {
		if _, err := os.Stat(filepath.Join(desk, filepath.FromSlash(n))); err != nil {
			t.Errorf("desk lost %s after laptop renamed its folder: %v", n, err)
		}
	}
}
