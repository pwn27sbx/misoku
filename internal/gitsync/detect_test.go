package gitsync_test

import (
	"context"
	"crypto/sha256"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// git runs a setup command in dir and fails the test if it fails.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// readyVault is a repository Misoku can sync: an origin remote (never
// contacted), an identity from isolate and a branch checked out.
func readyVault(t *testing.T) string {
	t.Helper()
	dir := newRepo(t)
	git(t, dir, "remote", "add", "origin", "file:///nowhere/notes.git")
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "a.md")
	git(t, dir, "commit", "-q", "-m", "first")
	return dir
}

func detect(t *testing.T, dir string) gitsync.State {
	t.Helper()
	return gitsync.Detect(context.Background(), gitsync.New(dir, 10*time.Second))
}

func wantPaused(t *testing.T, st gitsync.State, reason gitsync.Reason, fix string) {
	t.Helper()
	if st.Status != gitsync.Paused || st.Reason != reason {
		t.Fatalf("want paused with %q, got %+v", reason, st)
	}
	if !strings.Contains(st.Message, fix) {
		t.Errorf("the message must say how to fix it (%q): %q", fix, st.Message)
	}
}

// fakeGit puts a program named git, and nothing else, on PATH. body is a
// shell script on Unix and a batch file on Windows.
func fakeGit(t *testing.T, unixBody, windowsBody string) {
	t.Helper()
	dir := t.TempDir()
	name, body := "git", "#!/bin/sh\n"+unixBody+"\n"
	if runtime.GOOS == "windows" {
		name, body = "git.bat", "@echo off\r\n"+windowsBody+"\r\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestReadyVault(t *testing.T) {
	t.Parallel()
	isolate(t)
	st := detect(t, readyVault(t))
	if st.Status != gitsync.Ready || st.Reason != "" || st.Message != "" {
		t.Fatalf("want ready, got %+v", st)
	}
}

// A brand-new repository with no commit yet is on a branch: it can sync.
func TestReadyBeforeTheFirstCommit(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	git(t, dir, "remote", "add", "origin", "file:///nowhere/notes.git")
	if st := detect(t, dir); st.Status != gitsync.Ready {
		t.Fatalf("want ready, got %+v", st)
	}
}

func TestPausedWithoutGit(t *testing.T) {
	isolate(t)
	dir := readyVault(t)
	t.Setenv("PATH", t.TempDir())
	wantPaused(t, detect(t, dir), gitsync.ReasonGitMissing, "install git")
}

func TestPausedWithOldGit(t *testing.T) {
	isolate(t)
	dir := readyVault(t)
	fakeGit(t, `echo "git version 2.37.1"`, `echo git version 2.37.1`)
	st := detect(t, dir)
	wantPaused(t, st, gitsync.ReasonGitTooOld, "2.38")
	if !strings.Contains(st.Message, "2.37") {
		t.Errorf("the message must name the installed version: %q", st.Message)
	}
}

func TestPausedOutsideARepository(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	// Never let git find a repository above the temporary folder.
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	wantPaused(t, detect(t, dir), gitsync.ReasonNotARepo, "misoku sync init")
}

func TestPausedInASubfolderOfABiggerRepository(t *testing.T) {
	t.Parallel()
	isolate(t)
	repo := readyVault(t)
	sub := filepath.Join(repo, "notes")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	st := detect(t, sub)
	wantPaused(t, st, gitsync.ReasonNotRepoRoot, "top folder")
}

func TestPausedWithoutOrigin(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	wantPaused(t, detect(t, dir), gitsync.ReasonNoRemote, "git remote add origin")

	// Another remote is not enough: sync talks to origin.
	git(t, dir, "remote", "add", "backup", "file:///nowhere/backup.git")
	wantPaused(t, detect(t, dir), gitsync.ReasonNoRemote, "git remote add origin")
}

func TestPausedWithoutIdentity(t *testing.T) {
	isolate(t)
	dir := readyVault(t)
	for _, c := range []struct{ name, cfg string }{
		{"no identity", ""},
		{"no email", "[user]\n\tname = Test\n"},
		{"no name", "[user]\n\temail = test@example.com\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			global := filepath.Join(t.TempDir(), "gitconfig")
			if err := os.WriteFile(global, []byte(c.cfg), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GIT_CONFIG_GLOBAL", global)
			wantPaused(t, detect(t, dir), gitsync.ReasonNoIdentity, "git config --global user.")
		})
	}
}

func TestPausedOnDetachedHead(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := readyVault(t)
	git(t, dir, "checkout", "-q", "--detach")
	wantPaused(t, detect(t, dir), gitsync.ReasonDetachedHead, "git switch")
}

func TestPausedDuringARealMerge(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := readyVault(t)
	git(t, dir, "checkout", "-q", "-b", "other")
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# other\n"), 0o644)
	git(t, dir, "commit", "-q", "-am", "other")
	git(t, dir, "checkout", "-q", "main")
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# main\n"), 0o644)
	git(t, dir, "commit", "-q", "-am", "main")
	if err := exec.Command("git", "-C", dir, "merge", "other").Run(); err == nil {
		t.Fatal("the merge was meant to stop on a conflict")
	}
	wantPaused(t, detect(t, dir), gitsync.ReasonInProgress, "git merge --abort")
}

// A rebase detaches HEAD: the advice must be about the rebase, not about
// switching branches (which would leave the rebase behind).
func TestPausedDuringARealRebase(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := readyVault(t)
	git(t, dir, "checkout", "-q", "-b", "other")
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# other\n"), 0o644)
	git(t, dir, "commit", "-q", "-am", "other")
	git(t, dir, "checkout", "-q", "main")
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# main\n"), 0o644)
	git(t, dir, "commit", "-q", "-am", "main")
	if err := exec.Command("git", "-C", dir, "rebase", "other").Run(); err == nil {
		t.Fatal("the rebase was meant to stop on a conflict")
	}
	wantPaused(t, detect(t, dir), gitsync.ReasonInProgress, "git rebase --abort")
}

// Each operation is recognised by what git leaves in its git dir, wherever
// that is (git rev-parse --git-path), including a linked worktree.
func TestPausedDuringAnyOperation(t *testing.T) {
	t.Parallel()
	isolate(t)
	for _, c := range []struct {
		marker string
		dir    bool
		fix    string
	}{
		{"MERGE_HEAD", false, "git merge --abort"},
		{"rebase-merge", true, "git rebase --abort"},
		{"rebase-apply", true, "git rebase --abort"},
		{"CHERRY_PICK_HEAD", false, "git cherry-pick --abort"},
		{"REVERT_HEAD", false, "git revert --abort"},
	} {
		for _, worktree := range []bool{false, true} {
			name := c.marker
			if worktree {
				name += " in a linked worktree"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				dir := readyVault(t)
				if worktree {
					wt := filepath.Join(t.TempDir(), "wt")
					git(t, dir, "worktree", "add", "-q", "-b", "wt", wt)
					dir = wt
					if st := detect(t, dir); st.Status != gitsync.Ready {
						t.Fatalf("a clean linked worktree must be ready, got %+v", st)
					}
				}
				p := git(t, dir, "rev-parse", "--git-path", c.marker)
				if !filepath.IsAbs(p) {
					p = filepath.Join(dir, p)
				}
				var err error
				if c.dir {
					err = os.Mkdir(p, 0o755)
				} else {
					err = os.WriteFile(p, []byte("0000000000000000000000000000000000000000\n"), 0o644)
				}
				if err != nil {
					t.Fatal(err)
				}
				wantPaused(t, detect(t, dir), gitsync.ReasonInProgress, c.fix)
			})
		}
	}
}

type fileStamp struct {
	size int64
	mode fs.FileMode
	mod  time.Time
	sum  [sha256.Size]byte // contents, for regular files
}

// snapshot records every file and folder under dir, .git included: the
// contents of each file, and the modification time of each folder, which
// changes when anything is created or removed in it (a lock or temporary
// file included, even if it is gone again).
//
// Each entry is read with os.Lstat, not the DirEntry's Info: on Windows
// Info comes from the parent folder's listing, a copy NTFS updates lazily,
// so a folder written during the test setup could show its new time only
// after something (such as git) opened it, and look changed.
func snapshot(t *testing.T, dir string) map[string]fileStamp {
	t.Helper()
	files := map[string]fileStamp{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		st := fileStamp{size: info.Size(), mode: info.Mode(), mod: info.ModTime()}
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			st.sum = sha256.Sum256(data)
		}
		files[p] = st
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestDetectionNeverTouchesTheRepository(t *testing.T) {
	t.Parallel()
	isolate(t)
	ready := readyVault(t)
	merging := readyVault(t)
	os.WriteFile(filepath.Join(merging, ".git", "MERGE_HEAD"), []byte("0000000000000000000000000000000000000000\n"), 0o644)
	os.WriteFile(filepath.Join(merging, "a.md"), []byte("# edited\n"), 0o644)
	for _, dir := range []string{ready, merging} {
		before := snapshot(t, dir)
		detect(t, dir)
		after := snapshot(t, dir)
		if len(before) != len(after) {
			t.Fatalf("detection added or removed files in %s", dir)
		}
		for p, s := range before {
			if after[p] != s {
				t.Errorf("detection changed %s", p)
			}
		}
	}
}

// markerGit puts a fake git on PATH that leaves a file behind each time it
// runs, and returns that file's path.
func markerGit(t *testing.T) string {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "git-ran")
	fakeGit(t, `echo ran >> '`+marker+`'; echo "git version 2.54.0"`,
		`echo ran>> "`+marker+`"`+"\r\necho git version 2.54.0")
	return marker
}

func TestNoGitRunsForAFolderThatIsNotListed(t *testing.T) {
	t.Setenv("MISOKU_SYNC", "")
	marker := markerGit(t)
	listed, opened := t.TempDir(), t.TempDir()

	for _, vaults := range [][]config.Vault{nil, {vaultAt(listed)}} {
		st := gitsync.Check(context.Background(), vaults, opened, 5*time.Second)
		if st.Status != gitsync.Off || st.Reason != gitsync.ReasonNotListed {
			t.Fatalf("want off (not listed), got %+v", st)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("git ran for a folder that is not listed")
	}

	// The probe works: the listed folder does run git.
	st := gitsync.Check(context.Background(), []config.Vault{vaultAt(listed)}, listed, 5*time.Second)
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("the fake git should have run for the listed vault (state %+v)", st)
	}
	if st.Vault.Path != listed {
		t.Errorf("the state must carry the vault's settings, got %+v", st.Vault)
	}
}

func TestNoGitRunsWithMisokuSyncZero(t *testing.T) {
	marker := markerGit(t)
	dir := t.TempDir()
	t.Setenv("MISOKU_SYNC", "0")
	st := gitsync.Check(context.Background(), []config.Vault{vaultAt(dir)}, dir, 5*time.Second)
	if st.Status != gitsync.Off || st.Reason != gitsync.ReasonDisabled {
		t.Fatalf("want off (disabled), got %+v", st)
	}
	if !strings.Contains(st.Message, "MISOKU_SYNC") {
		t.Errorf("the message must say what turned sync off: %q", st.Message)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("git ran with MISOKU_SYNC=0")
	}
}

func TestCheckDetectsAListedVault(t *testing.T) {
	isolate(t)
	t.Setenv("MISOKU_SYNC", "")
	dir := readyVault(t)
	st := gitsync.Check(context.Background(), []config.Vault{vaultAt(dir)}, dir, 10*time.Second)
	if st.Status != gitsync.Ready {
		t.Fatalf("want ready, got %+v", st)
	}
	git(t, dir, "remote", "remove", "origin")
	st = gitsync.Check(context.Background(), []config.Vault{vaultAt(dir)}, dir, 10*time.Second)
	wantPaused(t, st, gitsync.ReasonNoRemote, "git remote add origin")
}
