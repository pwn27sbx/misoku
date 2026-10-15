package gitsync_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// stage puts content at p in dir's index through plumbing, never the
// disk, so a name that differs from a tracked one only in case can be
// staged on any disk, as a commit made on Linux would hold it.
func stage(t *testing.T, dir, p, content string) {
	t.Helper()
	src := filepath.Join(t.TempDir(), "blob")
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	oid := git(t, dir, "hash-object", "-w", src)
	git(t, dir, "-c", "core.ignorecase=false", "update-index", "--add", "--cacheinfo", "100644,"+oid+","+p)
}

func noticesOf(res gitsync.CommitResult, kind gitsync.NoticeKind) []string {
	var paths []string
	for _, n := range res.Notices {
		if n.Kind == kind {
			paths = append(paths, n.Path)
		}
	}
	return paths
}

// A new path whose name differs from a tracked one, or from another new
// one, only in letter case stays out of the commit on every OS, with a
// notice: macOS and Windows could not hold both.
func TestCommitLeavesOutACaseDuplicateName(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := readyVault(t) // tracks a.md
	// Staged through plumbing and written to disk, so status reads them
	// as added on any disk ("b.md" and "B.md" hold the same bytes, one
	// file on a disk that ignores case).
	for p, content := range map[string]string{"A.md": "other\n", "b.md": "b\n", "B.md": "b\n"} {
		stage(t, dir, p, content)
		write(t, dir, p, content)
	}
	write(t, dir, "c.md", "c\n") // a plain new note still commits
	res := commitVault(t, dir)
	if !res.Committed {
		t.Fatal("c.md was not committed")
	}
	wantTree(t, dir, "B.md", "a.md", "c.md")
	if got, want := noticesOf(res, gitsync.CaseDuplicate), []string{"A.md", "b.md"}; !slices.Equal(got, want) {
		t.Errorf("CaseDuplicate notices %v, want %v", got, want)
	}
	for _, n := range res.Notices {
		if n.Kind == gitsync.CaseDuplicate && !strings.Contains(n.String(), "until you rename") {
			t.Errorf("the notice must say it stays out until renamed: %q", n.String())
		}
	}
}

// A note renamed to a case duplicate keeps syncing under its old name:
// committing the rename's deletion half alone would delete it everywhere.
func TestCommitHoldsARenameToACaseDuplicate(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := readyVault(t) // tracks a.md
	write(t, dir, "n.md", "note\n")
	git(t, dir, "add", "n.md")
	git(t, dir, "commit", "-q", "-m", "n")
	git(t, dir, "rm", "-q", "--cached", "n.md")
	remove(t, dir, "n.md")
	stage(t, dir, "A.md", "note\n")
	write(t, dir, "A.md", "note\n")
	res := commitVault(t, dir)
	wantTree(t, dir, "a.md", "n.md")
	if got := noticesOf(res, gitsync.HeldDeletion); !slices.Equal(got, []string{"n.md"}) {
		t.Errorf("HeldDeletion notices %v, want [n.md]", got)
	}
	if got := noticesOf(res, gitsync.CaseDuplicate); !slices.Equal(got, []string{"A.md"}) {
		t.Errorf("CaseDuplicate notices %v, want [A.md]", got)
	}
}

// Two tracked files differing only in case (from an older commit made on
// Linux): on a case-insensitive vault one shows modified because the
// other's content is on disk. That is not committed: nothing is
// overwritten. Where the disk tells case apart it is a real edit.
func TestCommitHoldsACaseDuplicateOverwrittenOnDisk(t *testing.T) {
	t.Parallel()
	for _, ignoreCase := range []bool{true, false} {
		t.Run("ignorecase="+map[bool]string{true: "true", false: "false"}[ignoreCase], func(t *testing.T) {
			t.Parallel()
			isolate(t)
			dir := readyVault(t) // tracks a.md
			c := plumbCommit(t, dir, git(t, dir, "rev-parse", "HEAD"), map[string]string{"A.md": "upper\n"})
			git(t, dir, "update-ref", "HEAD", c)
			git(t, dir, "read-tree", "HEAD")
			git(t, dir, "config", "core.ignorecase", map[bool]string{true: "true", false: "false"}[ignoreCase])
			// What a case-insensitive disk holds: one file, a.md's content,
			// under both names.
			write(t, dir, "a.md", "# a\n")
			if cs, err := sameFile(dir, "a.md", "A.md"); err == nil && !cs {
				write(t, dir, "A.md", "# a\n")
			}
			res := commitVault(t, dir)
			got := git(t, dir, "show", "HEAD:A.md")
			if ignoreCase {
				if got != "upper" || res.Committed {
					t.Errorf("A.md overwritten with a.md's content: %q (committed %v)", got, res.Committed)
				}
				// A.md held, a.md told about as its twin.
				if n := noticesOf(res, gitsync.CaseClash); !slices.Equal(n, []string{"A.md", "a.md"}) {
					t.Errorf("CaseClash notices %v, want [A.md a.md]", n)
				}
			} else if got != "# a" {
				t.Errorf("a real edit of A.md was not committed: %q", got)
			}
		})
	}
}

// sameFile reports whether a and b in dir are one file on disk (the disk
// ignores case).
func sameFile(dir, a, b string) (bool, error) {
	ia, err := os.Stat(filepath.Join(dir, a))
	if err != nil {
		return false, err
	}
	ib, err := os.Stat(filepath.Join(dir, b))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return os.SameFile(ia, ib), nil
}

// On a real case-insensitive disk: a commit from Linux holding a.md and
// A.md reaches this device by fast-forward; later syncs must leave both
// in origin as they were.
func TestSyncKeepsCaseDuplicatesFromOrigin(t *testing.T) {
	t.Parallel()
	probe := t.TempDir()
	write(t, probe, "x", "x\n")
	if same, err := sameFile(probe, "x", "X"); err != nil || !same {
		t.Skip("this disk tells letter case apart")
	}
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	c := plumbCommit(t, laptop, git(t, laptop, "rev-parse", "HEAD"), map[string]string{"a.md": "lower\n", "A.md": "upper\n"})
	git(t, laptop, "push", "-q", "origin", c+":refs/heads/main")
	want := git(t, remote, "ls-tree", "-r", "main")
	for i := 0; i < 2; i++ {
		wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	}
	write(t, desk, "other.md", "other\n")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	got := git(t, remote, "ls-tree", "-r", "main")
	for _, line := range strings.Split(want, "\n") {
		if !strings.Contains(got, line) {
			t.Errorf("origin lost or changed %q:\n%s", line, got)
		}
	}
	if !strings.Contains(got, "\tother.md") {
		t.Errorf("other.md did not reach origin:\n%s", got)
	}
}

// On a disk that tells case apart, renaming a note to the same name in
// other letters is a plain rename: the old name is gone, so the new one
// is no duplicate.
func TestCommitCaseOnlyRenameOnACaseSensitiveDisk(t *testing.T) {
	t.Parallel()
	probe := t.TempDir()
	write(t, probe, "x", "x\n")
	if same, err := sameFile(probe, "x", "X"); err != nil || same {
		t.Skip("this disk ignores letter case")
	}
	isolate(t)
	dir := readyVault(t) // tracks a.md
	rename(t, dir, "a.md", "A.md")
	res := commitVault(t, dir)
	wantTree(t, dir, "A.md")
	if n := noticesOf(res, gitsync.CaseDuplicate); len(n) != 0 {
		t.Errorf("CaseDuplicate notices %v, want none", n)
	}
}

// Two tracked files differing only in case, both missing from a case-
// insensitive disk after the user renamed the one file it shows: neither
// deletion is committed, the new name is.
func TestCommitHoldsBothDeletionsOfACasePair(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := readyVault(t) // tracks a.md
	c := plumbCommit(t, dir, git(t, dir, "rev-parse", "HEAD"), map[string]string{"A.md": "upper\n"})
	git(t, dir, "update-ref", "HEAD", c)
	git(t, dir, "read-tree", "HEAD")
	git(t, dir, "config", "core.ignorecase", "true")
	remove(t, dir, "a.md")
	write(t, dir, "b.md", "# a\n")
	res := commitVault(t, dir)
	wantTree(t, dir, "A.md", "a.md", "b.md")
	if n := noticesOf(res, gitsync.CaseClash); !slices.Equal(n, []string{"A.md", "a.md"}) {
		t.Errorf("CaseClash notices %v, want [A.md a.md]", n)
	}
}

// A tracked case pair is told about on the first commit on a case-
// insensitive vault, with nothing changed; an ordinary vault, or one
// whose disk tells case apart, gets no such notice.
func TestCommitWarnsAboutATrackedCasePair(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		pair       bool
		ignoreCase string
		want       []string
	}{
		{"pair, case-insensitive", true, "true", []string{"A.md", "a.md"}},
		{"pair, case-sensitive", true, "false", nil},
		{"ordinary vault", false, "true", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			isolate(t)
			dir := readyVault(t) // tracks a.md, "# a\n"
			if c.pair {
				commit := plumbCommit(t, dir, git(t, dir, "rev-parse", "HEAD"), map[string]string{"A.md": "# a\n"})
				git(t, dir, "update-ref", "HEAD", commit)
				git(t, dir, "read-tree", "HEAD")
				write(t, dir, "A.md", "# a\n") // one file or two, same bytes
			}
			git(t, dir, "config", "core.ignorecase", c.ignoreCase)
			res := commitVault(t, dir)
			if res.Committed {
				t.Errorf("nothing changed, yet a commit was made")
			}
			if n := noticesOf(res, gitsync.CaseClash); !slices.Equal(n, c.want) {
				t.Errorf("CaseClash notices %v, want %v", n, c.want)
			}
		})
	}
}
