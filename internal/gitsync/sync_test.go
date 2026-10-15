package gitsync_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// syncTime is the local time every test sync runs at, so conflict copy
// names are known in advance.
var syncTime = time.Date(2026, 10, 14, 15, 30, 0, 0, time.Local)

const stamp = "2026-10-14 1530"

// syncVault runs one sync of dir as device, holding a lock the way the app
// will.
func syncVault(t *testing.T, dir, device string, allowUnrelated bool) gitsync.SyncResult {
	t.Helper()
	lock, ok, err := gitsync.TryLock(filepath.Join(t.TempDir(), "sync.lock"))
	if err != nil || !ok {
		t.Fatalf("TryLock: %v %v", ok, err)
	}
	defer lock.Unlock()
	return gitsync.SyncVault(context.Background(), gitsync.New(dir, 20*time.Second), lock, gitsync.SyncOptions{
		Device:                  device,
		AllowUnrelatedHistories: allowUnrelated,
		Now:                     func() time.Time { return syncTime },
	})
}

func wantOutcome(t *testing.T, res gitsync.SyncResult, want gitsync.SyncOutcome) {
	t.Helper()
	if res.Outcome != want {
		t.Fatalf("outcome = %v, want %v (err: %v)", res.Outcome, want, res.Err)
	}
	if want == gitsync.Synced && res.Err != nil {
		t.Errorf("a finished sync carries no error, got %v", res.Err)
	}
	if want != gitsync.Synced && res.Err == nil {
		t.Errorf("outcome %v must carry its cause in Err", want)
	}
}

// newRemote is an empty bare repository standing in for GitHub.
func newRemote(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "notes.git")
	git(t, filepath.Dir(dir), "init", "-q", "--bare", "notes.git")
	return dir
}

// clone makes a device's vault from remote.
func clone(t *testing.T, remote string) string {
	t.Helper()
	parent := t.TempDir()
	git(t, parent, "clone", "-q", remote, "vault")
	return filepath.Join(parent, "vault")
}

// pair is a remote and two devices' vaults ("laptop" and "desk"), both
// holding seed.
func pair(t *testing.T, seed map[string]string) (remote, laptop, desk string) {
	t.Helper()
	isolate(t)
	remote = newRemote(t)
	laptop, desk = clone(t, remote), clone(t, remote)
	for name, content := range seed {
		write(t, laptop, name, content)
	}
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	return remote, laptop, desk
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func wantFile(t *testing.T, dir, name, want string) {
	t.Helper()
	if got := read(t, dir, name); got != want {
		t.Errorf("%s holds %q, want %q", name, got, want)
	}
}

func wantMissing(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(name))); !os.IsNotExist(err) {
		t.Errorf("%s should not exist (err %v)", name, err)
	}
}

// remoteHead is the commit the remote's branch points at, or "" if none.
func remoteHead(t *testing.T, remote, branch string) string {
	t.Helper()
	return git(t, remote, "for-each-ref", "--format=%(objectname)", "refs/heads/"+branch)
}

// wantInSync checks that dir's HEAD is on the remote, its upstream is set
// and nothing is left uncommitted.
func wantInSync(t *testing.T, remote, dir, branch string) {
	t.Helper()
	if head, rh := git(t, dir, "rev-parse", "HEAD"), remoteHead(t, remote, branch); head != rh {
		t.Errorf("HEAD %s is not what the remote's %s holds (%s)", head, branch, rh)
	}
	if up := git(t, dir, "rev-parse", "--abbrev-ref", "@{upstream}"); up != "origin/"+branch {
		t.Errorf("upstream = %q, want origin/%s", up, branch)
	}
	if st := git(t, dir, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the vault should be clean after a sync:\n%s", st)
	}
}

func parents(t *testing.T, dir, rev string) []string {
	t.Helper()
	return strings.Fields(git(t, dir, "log", "-1", "--format=%P", rev))
}

func TestSyncNeedsTheLock(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	res := gitsync.SyncVault(context.Background(), gitsync.New(dir, 10*time.Second), nil, gitsync.SyncOptions{Device: "mac"})
	wantOutcome(t, res, gitsync.SyncFailed)
	if out := git(t, dir, "log", "--oneline", "--all"); out != "" {
		t.Errorf("nothing may run without the lock, log:\n%s", out)
	}
}

func TestSyncPushesToAnEmptyRemote(t *testing.T) {
	t.Parallel()
	isolate(t)
	remote := newRemote(t)
	dir := newRepo(t)
	git(t, dir, "remote", "add", "origin", remote)
	write(t, dir, "a.md", "# a\r\nwindows line\r\n")

	res := syncVault(t, dir, "mac", false)
	wantOutcome(t, res, gitsync.Synced)
	if res.Branch != "main" || !res.Pushed || res.Pending != 0 || !res.Commit.Committed || res.Updated || len(res.Conflicts) != 0 {
		t.Errorf("want main pushed with nothing pending, got %+v", res)
	}
	wantInSync(t, remote, dir, "main")
	wantFile(t, dir, "a.md", "# a\r\nwindows line\r\n")
	if got := git(t, remote, "show", "main:a.md"); got != strings.TrimSpace("# a\r\nwindows line\r\n") {
		t.Errorf("the remote holds %q", got)
	}
}

func TestSyncWithNothingAnywhereDoesNothing(t *testing.T) {
	t.Parallel()
	isolate(t)
	remote := newRemote(t)
	dir := newRepo(t)
	git(t, dir, "remote", "add", "origin", remote)

	res := syncVault(t, dir, "mac", false)
	wantOutcome(t, res, gitsync.Synced)
	if res.Pushed || res.Commit.Committed || res.Pending != 0 {
		t.Errorf("nothing to do, got %+v", res)
	}
	if refs := git(t, remote, "for-each-ref"); refs != "" {
		t.Errorf("the remote should stay empty, has:\n%s", refs)
	}
}

func TestSyncUsesTheRemotesDefaultBranch(t *testing.T) {
	t.Parallel()
	isolate(t)
	remote := newRemote(t)
	laptop := clone(t, remote)
	git(t, laptop, "checkout", "-q", "-b", "notes")
	write(t, laptop, "n.md", "from laptop\n")
	git(t, laptop, "add", "-A")
	git(t, laptop, "commit", "-q", "-m", "n")
	git(t, laptop, "push", "-q", "origin", "notes")
	git(t, remote, "symbolic-ref", "HEAD", "refs/heads/notes")

	desk := newRepo(t) // on main, no commits yet
	git(t, desk, "remote", "add", "origin", remote)
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if res.Branch != "notes" || !res.Updated || res.Pushed {
		t.Errorf("want the notes branch brought in, got %+v", res)
	}
	wantFile(t, desk, "n.md", "from laptop\n")
	wantInSync(t, remote, desk, "notes")

	write(t, desk, "d.md", "from desk\n")
	res = syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if !res.Pushed {
		t.Errorf("want a push, got %+v", res)
	}
	wantInSync(t, remote, desk, "notes")
	if main := remoteHead(t, remote, "main"); main != "" {
		t.Errorf("nothing may be pushed to another branch, main = %s", main)
	}
}

func TestSyncFastForwards(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "seed\n"})
	write(t, laptop, "n.md", "new on laptop\n")
	write(t, laptop, "seed.md", "seed edited\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if !res.Updated || res.Pushed || res.Commit.Committed || len(res.Conflicts) != 0 || res.Pending != 0 {
		t.Errorf("want a plain fast-forward, got %+v", res)
	}
	wantFile(t, desk, "n.md", "new on laptop\n")
	wantFile(t, desk, "seed.md", "seed edited\n")
	if d, l := git(t, desk, "rev-parse", "HEAD"), git(t, laptop, "rev-parse", "HEAD"); d != l {
		t.Errorf("a fast-forward makes no commit of its own: desk %s, laptop %s", d, l)
	}
	wantInSync(t, remote, desk, "main")
}

func TestSyncJoinsChangesFromBothDevices(t *testing.T) {
	t.Parallel()
	seed := "one\ntwo\nthree\nfour\nfive\n"
	remote, laptop, desk := pair(t, map[string]string{"seed.md": seed})
	write(t, laptop, "x.md", "laptop note\n")
	write(t, laptop, "seed.md", "ONE\ntwo\nthree\nfour\nfive\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "y.md", "desk note\n")
	write(t, desk, "seed.md", "one\ntwo\nthree\nfour\nFIVE\n")
	deskCommit := ""

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if !res.Updated || !res.Pushed || !res.Commit.Committed || len(res.Conflicts) != 0 || res.Pending != 0 {
		t.Errorf("want a joined, pushed sync, got %+v", res)
	}
	deskCommit = res.Commit.ID
	wantFile(t, desk, "x.md", "laptop note\n")
	wantFile(t, desk, "y.md", "desk note\n")
	wantFile(t, desk, "seed.md", "ONE\ntwo\nthree\nfour\nFIVE\n")
	wantInSync(t, remote, desk, "main")
	if p := parents(t, desk, "HEAD"); len(p) != 2 || p[0] != deskCommit || p[1] != git(t, laptop, "rev-parse", "HEAD") {
		t.Errorf("the union's parents are %v, want [desk's commit, laptop's]", p)
	}
	if dev := git(t, desk, "log", "-1", "--format=%(trailers:key=Misoku-Device,valueonly)"); dev != "desk" {
		t.Errorf("the union's Misoku-Device = %q", dev)
	}

	res = syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.Synced)
	wantFile(t, laptop, "y.md", "desk note\n")
	wantFile(t, laptop, "seed.md", "ONE\ntwo\nthree\nfour\nFIVE\n")
	wantInSync(t, remote, laptop, "main")
}

// conflict seeds name, changes it differently on both devices, syncs the
// laptop and then the desk, and returns the desk's result.
func conflict(t *testing.T, name, base, onLaptop, onDesk string) (remote, laptop, desk string, res gitsync.SyncResult) {
	t.Helper()
	remote, laptop, desk = pair(t, map[string]string{name: base})
	write(t, laptop, name, onLaptop)
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, name, onDesk)
	return remote, laptop, desk, syncVault(t, desk, "desk", false)
}

func TestSyncKeepsBothVersionsWhenTheSameLinesChanged(t *testing.T) {
	t.Parallel()
	remote, _, desk, res := conflict(t, "notes/n.md", "one\n", "from laptop\n", "from desk\n")
	wantOutcome(t, res, gitsync.Synced)
	copyName := "notes/n (conflict laptop " + stamp + ").md"
	if !slices.Equal(res.Conflicts, []string{copyName}) || !res.Pushed || !res.Updated {
		t.Errorf("want one conflict copy %q, got %+v", copyName, res)
	}
	wantFile(t, desk, "notes/n.md", "from desk\n")
	wantFile(t, desk, copyName, "from laptop\n")
	wantTree(t, desk, "notes/n.md", copyName)
	wantInSync(t, remote, desk, "main")
}

func TestSyncKeepsTheEditedNoteOverTheDeletedOne(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"d1.md": "one\n", "d2.md": "two\n", "keep.md": "k\n"})
	if err := os.Remove(filepath.Join(laptop, "d1.md")); err != nil {
		t.Fatal(err)
	}
	write(t, laptop, "d2.md", "two edited on laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "d1.md", "one edited on desk\n")
	if err := os.Remove(filepath.Join(desk, "d2.md")); err != nil {
		t.Fatal(err)
	}

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if len(res.Conflicts) != 0 {
		t.Errorf("the edited note wins, no copies: %v", res.Conflicts)
	}
	wantFile(t, desk, "d1.md", "one edited on desk\n")
	wantFile(t, desk, "d2.md", "two edited on laptop\n")
	wantTree(t, desk, "d1.md", "d2.md", "keep.md")
	wantInSync(t, remote, desk, "main")
}

func TestSyncKeepsBothNewNotesWithTheSameName(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "idea.md", "laptop idea\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "idea.md", "desk idea\n")

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	copyName := "idea (conflict laptop " + stamp + ").md"
	if !slices.Equal(res.Conflicts, []string{copyName}) {
		t.Errorf("conflicts = %v, want [%s]", res.Conflicts, copyName)
	}
	wantFile(t, desk, "idea.md", "desk idea\n")
	wantFile(t, desk, copyName, "laptop idea\n")
	wantTree(t, desk, "seed.md", "idea.md", copyName)
	wantInSync(t, remote, desk, "main")
}

func TestSyncKeepsBothVersionsOfAnImage(t *testing.T) {
	t.Parallel()
	laptopBytes, deskBytes := "\x89PNG\x00\x01laptop\x00\xff", "\x89PNG\x00\x01desk\x00\xfe"
	remote, _, desk, res := conflict(t, "img/photo.png", "\x89PNG\x00base", laptopBytes, deskBytes)
	wantOutcome(t, res, gitsync.Synced)
	copyName := "img/photo (conflict laptop " + stamp + ").png"
	if !slices.Equal(res.Conflicts, []string{copyName}) {
		t.Errorf("conflicts = %v, want [%s]", res.Conflicts, copyName)
	}
	wantFile(t, desk, "img/photo.png", deskBytes)
	wantFile(t, desk, copyName, laptopBytes)
	wantInSync(t, remote, desk, "main")
}

func TestConflictCopyNamesAreSafeAndNeverReused(t *testing.T) {
	isolate(t)
	remote := newRemote(t)
	laptop, desk := clone(t, remote), clone(t, remote)
	taken := "notas/ñandú (conflict lap-top " + stamp + ").md"
	write(t, laptop, "notas/ñandú.md", "base\n")
	write(t, laptop, taken, "an older copy\n")
	wantOutcome(t, syncVault(t, laptop, "lap:top", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	write(t, laptop, "notas/ñandú.md", "laptop\n")
	wantOutcome(t, syncVault(t, laptop, "lap:top", false), gitsync.Synced)
	write(t, desk, "notas/ñandú.md", "desk\n")
	// An untracked file on disk also takes a name.
	write(t, desk, "notas/ñandú (conflict lap-top "+stamp+") 2.md", "a stray file\n")
	withLimit(t, 10) // keeps the stray file (13 bytes) out of the desk's commit

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	copyName := "notas/ñandú (conflict lap-top " + stamp + ") 3.md"
	if !slices.Equal(res.Conflicts, []string{copyName}) {
		t.Fatalf("conflicts = %v, want [%s]", res.Conflicts, copyName)
	}
	if strings.ContainsAny(copyName, `:<>"|?*\`) {
		t.Errorf("%q is not a valid Windows file name", copyName)
	}
	wantFile(t, desk, copyName, "laptop\n")
	wantFile(t, desk, taken, "an older copy\n")
	wantFile(t, desk, "notas/ñandú ("+"conflict lap-top "+stamp+") 2.md", "a stray file\n")
	wantFile(t, desk, "notas/ñandú.md", "desk\n")
}

func TestSyncNeverOverwritesASaveDuringTheUnion(t *testing.T) {
	remote, laptop, desk := pair(t, map[string]string{"n.md": "base\n", "o.md": "other\n"})
	write(t, laptop, "n.md", "from laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	laptopHead := git(t, laptop, "rev-parse", "HEAD")
	write(t, desk, "o.md", "other from desk\n")

	// The user saves n.md after the desk committed but before the union
	// lands: the union changes n.md, so it must not be applied.
	defer gitsync.SetSyncHooks(func() { write(t, desk, "n.md", "saved during sync\n") }, nil)()
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncRetry)
	if res.Pending != 1 || res.Pushed || res.Updated || !res.Commit.Committed {
		t.Errorf("want the desk's commit kept and pending, got %+v", res)
	}
	wantFile(t, desk, "n.md", "saved during sync\n")
	wantFile(t, desk, "o.md", "other from desk\n")
	if head := git(t, desk, "rev-parse", "HEAD"); head != res.Commit.ID {
		t.Errorf("HEAD moved to %s, want the desk's own commit %s", head, res.Commit.ID)
	}
	if rh := remoteHead(t, remote, "main"); rh != laptopHead {
		t.Errorf("nothing may be pushed: remote %s, laptop %s", rh, laptopHead)
	}

	// Next time the save is committed and the union retried: nothing lost.
	gitsync.SetSyncHooks(nil, nil)
	res = syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	copyName := "n (conflict laptop " + stamp + ").md"
	wantFile(t, desk, "n.md", "saved during sync\n")
	wantFile(t, desk, copyName, "from laptop\n")
	wantFile(t, desk, "o.md", "other from desk\n")
	wantInSync(t, remote, desk, "main")
}

func TestSyncNeverOverwritesASaveDuringAFastForward(t *testing.T) {
	remote, laptop, desk := pair(t, map[string]string{"n.md": "base\n"})
	write(t, laptop, "n.md", "from laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	deskHead := git(t, desk, "rev-parse", "HEAD")

	defer gitsync.SetSyncHooks(func() { write(t, desk, "n.md", "saved during sync\n") }, nil)()
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncRetry)
	if res.Updated || res.Pending != 0 {
		t.Errorf("want nothing applied and nothing pending, got %+v", res)
	}
	wantFile(t, desk, "n.md", "saved during sync\n")
	if head := git(t, desk, "rev-parse", "HEAD"); head != deskHead {
		t.Errorf("HEAD moved to %s", head)
	}

	gitsync.SetSyncHooks(nil, nil)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "n.md", "saved during sync\n")
	wantFile(t, desk, "n (conflict laptop "+stamp+").md", "from laptop\n")
	wantInSync(t, remote, desk, "main")
}

func TestSyncRetriesWhenThePushIsRejected(t *testing.T) {
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, desk, "d.md", "desk\n")
	// The laptop pushes between the desk's fetch and its push.
	defer gitsync.SetSyncHooks(nil, func() {
		gitsync.SetSyncHooks(nil, nil) // once, and not inside the laptop's own sync
		write(t, laptop, "l.md", "laptop\n")
		wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	})()
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncRetry)
	if res.Pushed || res.Pending != 1 {
		t.Errorf("want the desk's commit pending, got %+v", res)
	}
	if rh, lh := remoteHead(t, remote, "main"), git(t, laptop, "rev-parse", "HEAD"); rh != lh {
		t.Errorf("the laptop's push must stay: remote %s, laptop %s", rh, lh)
	}

	gitsync.SetSyncHooks(nil, nil)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "l.md", "laptop\n")
	wantFile(t, desk, "d.md", "desk\n")
	wantInSync(t, remote, desk, "main")
}

func TestSyncOfflineKeepsCommitsAndCountsThem(t *testing.T) {
	t.Parallel()
	remote, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	gone := remote + ".gone"
	if err := os.Rename(remote, gone); err != nil {
		t.Fatal(err)
	}

	write(t, desk, "a.md", "a\n")
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncOffline)
	if res.Pending != 1 || !res.Commit.Committed || res.Pushed {
		t.Errorf("want 1 pending, got %+v", res)
	}
	write(t, desk, "b.md", "b\n")
	res = syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncOffline)
	if res.Pending != 2 {
		t.Errorf("want 2 pending, got %+v", res)
	}
	wantFile(t, desk, "a.md", "a\n")
	wantFile(t, desk, "b.md", "b\n")

	if err := os.Rename(gone, remote); err != nil {
		t.Fatal(err)
	}
	res = syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if res.Pending != 0 || !res.Pushed {
		t.Errorf("want everything pushed, got %+v", res)
	}
	wantInSync(t, remote, desk, "main")
	if got := git(t, remote, "show", "main:b.md"); got != "b" {
		t.Errorf("the remote holds b.md = %q", got)
	}
}

func TestSyncOfflineBeforeTheFirstPushCountsEveryCommit(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	git(t, dir, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
	write(t, dir, "a.md", "a\n")
	res := syncVault(t, dir, "mac", false)
	wantOutcome(t, res, gitsync.SyncOffline)
	if res.Pending != 1 || !res.Commit.Committed {
		t.Errorf("want 1 pending, got %+v", res)
	}
}

// unrelated is a remote created with its own README and a vault with its
// own history and notes.
func unrelated(t *testing.T) (remote, vault string) {
	t.Helper()
	isolate(t)
	remote = newRemote(t)
	other := clone(t, remote)
	write(t, other, "README.md", "remote readme\n")
	git(t, other, "add", "-A")
	git(t, other, "commit", "-q", "-m", "Initial commit")
	git(t, other, "push", "-q", "origin", "main")

	vault = newRepo(t)
	git(t, vault, "remote", "add", "origin", remote)
	write(t, vault, "README.md", "local readme\n")
	write(t, vault, "note.md", "my note\n")
	git(t, vault, "add", "-A")
	git(t, vault, "commit", "-q", "-m", "mine")
	return remote, vault
}

func TestSyncRefusesUnrelatedHistories(t *testing.T) {
	t.Parallel()
	remote, vault := unrelated(t)
	before, remoteBefore := git(t, vault, "rev-parse", "HEAD"), remoteHead(t, remote, "main")

	res := syncVault(t, vault, "mac", false)
	wantOutcome(t, res, gitsync.SyncUnrelated)
	if res.Pushed || res.Updated || len(res.Conflicts) != 0 {
		t.Errorf("nothing may happen, got %+v", res)
	}
	if head := git(t, vault, "rev-parse", "HEAD"); head != before {
		t.Errorf("HEAD moved to %s", head)
	}
	if rh := remoteHead(t, remote, "main"); rh != remoteBefore {
		t.Errorf("the remote moved to %s", rh)
	}
	wantFile(t, vault, "README.md", "local readme\n")
	wantFile(t, vault, "note.md", "my note\n")
	wantTree(t, vault, "README.md", "note.md")
}

func TestSyncJoinsUnrelatedHistoriesWhenAllowed(t *testing.T) {
	t.Parallel()
	remote, vault := unrelated(t)
	mine, theirs := git(t, vault, "rev-parse", "HEAD"), remoteHead(t, remote, "main")

	res := syncVault(t, vault, "mac", true)
	wantOutcome(t, res, gitsync.Synced)
	// The remote's README was not made by Misoku: no device in the name.
	copyName := "README (conflict " + stamp + ").md"
	if !slices.Equal(res.Conflicts, []string{copyName}) || !res.Pushed {
		t.Errorf("want both READMEs kept and pushed, got %+v", res)
	}
	wantFile(t, vault, "README.md", "local readme\n")
	wantFile(t, vault, copyName, "remote readme\n")
	wantFile(t, vault, "note.md", "my note\n")
	wantTree(t, vault, "README.md", copyName, "note.md")
	if p := parents(t, vault, "HEAD"); !slices.Equal(p, []string{mine, theirs}) {
		t.Errorf("parents = %v, want [%s %s]", p, mine, theirs)
	}
	wantInSync(t, remote, vault, "main")
}

func TestSyncStopsWhenTheRemoteRewroteItsHistory(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "x.md", "x\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	tracking := git(t, desk, "rev-parse", "origin/main")

	// The laptop throws x.md's commit away and force-pushes.
	git(t, laptop, "reset", "-q", "--hard", "HEAD~1")
	write(t, laptop, "y.md", "y\n")
	git(t, laptop, "add", "-A")
	git(t, laptop, "commit", "-q", "-m", "rewritten")
	git(t, laptop, "push", "-q", "-f", "origin", "main")
	rewritten := remoteHead(t, remote, "main")

	write(t, desk, "d.md", "d\n")
	for i := 0; i < 2; i++ { // and it keeps stopping until the user acts
		res := syncVault(t, desk, "desk", false)
		wantOutcome(t, res, gitsync.SyncRewritten)
		if res.Pushed || res.Updated {
			t.Errorf("nothing may be applied or pushed, got %+v", res)
		}
		if got := git(t, desk, "rev-parse", "origin/main"); got != tracking {
			t.Errorf("origin/main moved to %s", got)
		}
		if rh := remoteHead(t, remote, "main"); rh != rewritten {
			t.Errorf("the remote moved to %s", rh)
		}
		wantFile(t, desk, "x.md", "x\n")
		wantFile(t, desk, "d.md", "d\n")
		wantMissing(t, desk, "y.md")
	}
}

func TestSyncKeepsBothNamesOfANoteRenamedDifferently(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"n.md": "the note\n"})
	if err := os.Rename(filepath.Join(laptop, "n.md"), filepath.Join(laptop, "laptop name.md")); err != nil {
		t.Fatal(err)
	}
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	if err := os.Rename(filepath.Join(desk, "n.md"), filepath.Join(desk, "desk name.md")); err != nil {
		t.Fatal(err)
	}

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if len(res.Conflicts) != 0 {
		t.Errorf("no copies are needed, got %v", res.Conflicts)
	}
	wantFile(t, desk, "laptop name.md", "the note\n")
	wantFile(t, desk, "desk name.md", "the note\n")
	wantMissing(t, desk, "n.md")
	wantTree(t, desk, "desk name.md", "laptop name.md")
	wantInSync(t, remote, desk, "main")
}

func TestConflictCopyNameSkipsNamesArrivingFromOrigin(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"n.md": "base\n"})
	// The laptop brings a file whose name is the copy's, in another case:
	// it is not on the desk's disk yet, only in the union.
	arriving := "N (CONFLICT laptop " + stamp + ").md"
	write(t, laptop, arriving, "arriving\n")
	write(t, laptop, "n.md", "laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "n.md", "desk\n")

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	copyName := "n (conflict laptop " + stamp + ") 2.md"
	if !slices.Equal(res.Conflicts, []string{copyName}) {
		t.Fatalf("conflicts = %v, want [%s]", res.Conflicts, copyName)
	}
	wantFile(t, desk, arriving, "arriving\n")
	wantFile(t, desk, copyName, "laptop\n")
	wantFile(t, desk, "n.md", "desk\n")
	wantTree(t, desk, "n.md", arriving, copyName)
	wantInSync(t, remote, desk, "main")
}

// vaultHashes hashes every file in the vault outside .git.
func vaultHashes(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
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
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		m[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func sameSnapshot(t *testing.T, what string, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Errorf("%s: %d files became %d (%v -> %v)", what, len(before), len(after), before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Errorf("%s: %s changed or vanished", what, k)
		}
	}
}

// contents returns the bytes of every file outside .git as a set, and
// fails on any conflict marker.
func contents(t *testing.T, dir string) map[string]bool {
	t.Helper()
	set := map[string]bool{}
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
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "<<<<<<<") || strings.Contains(string(b), ">>>>>>>") {
			t.Errorf("conflict marker on disk in %s: %q", p, b)
		}
		set[string(b)] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return set
}

// wantAllVersions checks that every version is still in some file of dir,
// byte for byte, and that no file holds a conflict marker.
func wantAllVersions(t *testing.T, dir string, versions ...string) {
	t.Helper()
	have := contents(t, dir)
	for _, v := range versions {
		if !have[v] {
			t.Errorf("version %q lost from %s (have %v)", v, dir, have)
		}
	}
}

// wantNoMarkers checks that no file in rev of the repository at dir holds
// a conflict marker.
func wantNoMarkers(t *testing.T, dir, rev string) {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "grep", "-l", "-e", "<<<<<<<", "-e", ">>>>>>>", rev).CombinedOutput()
	if err == nil {
		t.Errorf("conflict markers committed in %s:\n%s", rev, out)
		return
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("git grep: %v %s", err, out)
	}
}

func rename(t *testing.T, dir, from, to string) {
	t.Helper()
	dst := filepath.Join(dir, filepath.FromSlash(to))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, filepath.FromSlash(from)), dst); err != nil {
		t.Fatal(err)
	}
}

func remove(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Remove(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
		t.Fatal(err)
	}
}

func TestSyncRenamedDifferentlyAndEditedKeepsBothCleanVersions(t *testing.T) {
	t.Parallel()
	body := "l2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\n"
	remote, laptop, desk := pair(t, map[string]string{"n.md": "l1\n" + body})
	rename(t, laptop, "n.md", "lap.md")
	write(t, laptop, "lap.md", "LAP\n"+body)
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	rename(t, desk, "n.md", "desk.md")
	write(t, desk, "desk.md", "DESK\n"+body)

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	wantFile(t, desk, "desk.md", "DESK\n"+body)
	wantFile(t, desk, "lap.md", "LAP\n"+body)
	wantMissing(t, desk, "n.md")
	wantAllVersions(t, desk, "LAP\n"+body, "DESK\n"+body)
	wantInSync(t, remote, desk, "main")
	wantNoMarkers(t, remote, "main")

	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantFile(t, laptop, "desk.md", "DESK\n"+body)
	wantFile(t, laptop, "lap.md", "LAP\n"+body)
	wantMissing(t, laptop, "n.md")
}

func TestSyncKeepsBothNotesWhoseNamesDifferOnlyInCase(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "Note.md", "laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "note.md", "desk\n")

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	copyName := "Note (conflict laptop " + stamp + ").md"
	if !slices.Equal(res.Conflicts, []string{copyName}) {
		t.Errorf("conflicts = %v, want [%s]", res.Conflicts, copyName)
	}
	wantFile(t, desk, "note.md", "desk\n")
	wantFile(t, desk, copyName, "laptop\n")
	wantTree(t, desk, "seed.md", "note.md", copyName)
	wantAllVersions(t, desk, "laptop\n", "desk\n", "s\n")
	wantInSync(t, remote, desk, "main")

	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantAllVersions(t, laptop, "laptop\n", "desk\n", "s\n")
	wantInSync(t, remote, laptop, "main")
}

func TestSyncFetchesMainWhenTheRemotesHeadIsUnborn(t *testing.T) {
	t.Parallel()
	isolate(t)
	parent := t.TempDir()
	// A plain `git init --bare` whose HEAD names a branch nobody pushes.
	git(t, parent, "init", "-q", "--bare", "--initial-branch=trunk", "r.git")
	remote := filepath.Join(parent, "r.git")

	first := newRepo(t)
	git(t, first, "remote", "add", "origin", remote)
	write(t, first, "a.md", "a\n")
	res := syncVault(t, first, "first", false)
	wantOutcome(t, res, gitsync.Synced)
	if res.Branch != "main" || remoteHead(t, remote, "main") == "" {
		t.Fatalf("an empty remote gets main, got %+v", res)
	}

	second := newRepo(t)
	git(t, second, "remote", "add", "origin", remote)
	write(t, second, "b.md", "b\n")
	res = syncVault(t, second, "second", false)
	wantOutcome(t, res, gitsync.SyncUnrelated)
	if res.Branch != "main" {
		t.Errorf("branch = %q, want main", res.Branch)
	}
	res = syncVault(t, second, "second", true)
	wantOutcome(t, res, gitsync.Synced)
	wantFile(t, second, "a.md", "a\n")
	wantFile(t, second, "b.md", "b\n")
	wantInSync(t, remote, second, "main")

	// A third device with no notes just takes them.
	third := newRepo(t)
	git(t, third, "remote", "add", "origin", remote)
	wantOutcome(t, syncVault(t, third, "third", false), gitsync.Synced)
	wantFile(t, third, "a.md", "a\n")
	wantFile(t, third, "b.md", "b\n")
	wantInSync(t, remote, third, "main")
}

func TestSyncKeepsAFileAndAFolderOfTheSameName(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "Projects", "laptop file\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "Projects/plan.md", "desk plan\n")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantAllVersions(t, desk, "laptop file\n", "desk plan\n", "s\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantAllVersions(t, laptop, "laptop file\n", "desk plan\n", "s\n")
}

func TestSyncKeepsBothVersionsOfANoteRenamedOnOneSide(t *testing.T) {
	t.Parallel()
	body := "l2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\n"
	_, laptop, desk := pair(t, map[string]string{"n.md": "l1\n" + body})
	rename(t, laptop, "n.md", "new.md")
	write(t, laptop, "new.md", "LAP\n"+body)
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "n.md", "DESK\n"+body)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantAllVersions(t, desk, "LAP\n"+body, "DESK\n"+body)
}

func TestSyncJoinsDifferentPartsOfOneNote(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"n.md": "a\nb\nc\nd\ne\n"})
	write(t, laptop, "n.md", "A\nb\nc\nd\ne\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "n.md", "a\nb\nc\nd\nE\n")
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if len(res.Conflicts) != 0 {
		t.Errorf("conflicts %v", res.Conflicts)
	}
	wantFile(t, desk, "n.md", "A\nb\nc\nd\nE\n")
	contents(t, desk)
}

func TestSyncKeepsAMovedNoteDeletedElsewhere(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"n.md": "keep me\n"})
	rename(t, laptop, "n.md", "moved/n.md")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	remove(t, desk, "n.md")
	write(t, desk, "other.md", "o\n")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantAllVersions(t, desk, "keep me\n", "o\n")
}

func TestSyncEmptyVaultTakesOriginsNotes(t *testing.T) {
	t.Parallel()
	isolate(t)
	remote := newRemote(t)
	other := clone(t, remote)
	write(t, other, "a.md", "a\n")
	wantOutcome(t, syncVault(t, other, "other", false), gitsync.Synced)
	v := newRepo(t)
	git(t, v, "remote", "add", "origin", remote)
	wantOutcome(t, syncVault(t, v, "v", false), gitsync.Synced)
	wantFile(t, v, "a.md", "a\n")
	wantInSync(t, remote, v, "main")
}

func TestSyncOfflineTouchesNoFileOrCommit(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "l.md", "l\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "d.md", "d\n")
	commitVault(t, desk)
	write(t, desk, "e.md", "e\n")
	rename(t, filepath.Dir(remote), filepath.Base(remote), filepath.Base(remote)+".x")
	before := vaultHashes(t, desk)
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncOffline)
	head := git(t, desk, "rev-parse", "HEAD")
	if res.Pending != 2 {
		t.Errorf("pending = %d, want 2", res.Pending)
	}
	sameSnapshot(t, "offline", before, vaultHashes(t, desk))
	rename(t, filepath.Dir(remote), filepath.Base(remote)+".x", filepath.Base(remote))
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	if p := parents(t, desk, "HEAD"); len(p) != 2 || p[0] != head {
		t.Errorf("parents %v, the first should be %s", p, head)
	}
	wantAllVersions(t, desk, "l\n", "d\n", "e\n", "s\n")
}

func TestSyncKeepsANewNoteSavedDuringSyncAtAnArrivingPath(t *testing.T) {
	_, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "new.md", "laptop new\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "d.md", "d\n")
	var before map[string]string
	var head string
	defer gitsync.SetSyncHooks(func() {
		write(t, desk, "new.md", "desk new during sync\n")
		before = vaultHashes(t, desk)
		head = git(t, desk, "rev-parse", "HEAD")
	}, nil)()
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.SyncRetry)
	sameSnapshot(t, "retry", before, vaultHashes(t, desk))
	if h := git(t, desk, "rev-parse", "HEAD"); h != head {
		t.Errorf("HEAD moved to %s", h)
	}
	gitsync.SetSyncHooks(nil, nil)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantAllVersions(t, desk, "laptop new\n", "desk new during sync\n", "d\n")
}

func TestSyncUnrelatedRefusedTouchesNothingAndAllowedKeepsEverything(t *testing.T) {
	t.Parallel()
	remote, vault := unrelated(t)
	before := vaultHashes(t, vault)
	head := git(t, vault, "rev-parse", "HEAD")
	wantOutcome(t, syncVault(t, vault, "mac", false), gitsync.SyncUnrelated)
	sameSnapshot(t, "unrelated refused", before, vaultHashes(t, vault))
	if h := git(t, vault, "rev-parse", "HEAD"); h != head {
		t.Errorf("HEAD moved to %s", h)
	}
	wantOutcome(t, syncVault(t, vault, "mac", true), gitsync.Synced)
	wantAllVersions(t, vault, "local readme\n", "remote readme\n", "my note\n")
	wantInSync(t, remote, vault, "main")
}

func TestSyncRewrittenRemoteTouchesNoFileOrCommit(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "x.md", "x\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	git(t, laptop, "reset", "-q", "--hard", "HEAD~1")
	git(t, laptop, "push", "-q", "-f", "origin", "main")
	write(t, desk, "d.md", "d\n")
	commitVault(t, desk)
	before := vaultHashes(t, desk)
	head := git(t, desk, "rev-parse", "HEAD")
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncRewritten)
	sameSnapshot(t, "rewritten", before, vaultHashes(t, desk))
	if h := git(t, desk, "rev-parse", "HEAD"); h != head {
		t.Errorf("HEAD moved to %s", h)
	}
	if res.Pending != 1 {
		t.Errorf("pending = %d, want 1", res.Pending)
	}
}

func TestSyncKeepsEveryVersionInManyConflictsAtOnce(t *testing.T) {
	t.Parallel()
	seed := map[string]string{"a.md": "a\n", "b/c.md": "c\n", "img.png": "\x00base\xff", "del.md": "del\n"}
	_, laptop, desk := pair(t, seed)
	write(t, laptop, "a.md", "a lap\n")
	write(t, laptop, "b/c.md", "c lap\n")
	write(t, laptop, "img.png", "\x00lap\xff")
	remove(t, laptop, "del.md")
	write(t, laptop, "same.md", "same lap\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "a.md", "a desk\n")
	write(t, desk, "b/c.md", "c desk\n")
	write(t, desk, "img.png", "\x00desk\xff")
	write(t, desk, "del.md", "del desk\n")
	write(t, desk, "same.md", "same desk\n")
	all := []string{"a lap\n", "a desk\n", "c lap\n", "c desk\n", "\x00lap\xff", "\x00desk\xff", "del desk\n", "same lap\n", "same desk\n"}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantAllVersions(t, desk, all...)
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantAllVersions(t, laptop, all...)
}

func TestSyncKeepsANoteAddedToAFolderRenamedElsewhere(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"old/a.md": "a\n", "old/b.md": "b\n"})
	rename(t, laptop, "old", "new")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "old/c.md", "c desk\n")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantAllVersions(t, desk, "a\n", "b\n", "c desk\n")
}
