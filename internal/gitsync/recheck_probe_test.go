package gitsync_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

var gitAside = regexp.MustCompile(`~[0-9a-f]{7,40}$`)

// settle syncs desk then laptop then desk again and checks both vaults and
// origin agree, every version survives, no git "~<sha>" names remain and
// no markers were written.
func settle(t *testing.T, remote, laptop, desk string, versions ...string) []string {
	t.Helper()
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	for _, v := range []string{laptop, desk} {
		wantAllVersions(t, v, versions...)
		wantNoMarkers(t, v, "HEAD")
		for _, p := range tree(t, v) {
			if gitAside.MatchString(p) {
				t.Errorf("git's own aside name left in %s: %s", v, p)
			}
		}
	}
	if a, b := tree(t, laptop), tree(t, desk); !slices.Equal(a, b) {
		t.Errorf("trees differ:\nlaptop %v\ndesk   %v", a, b)
	}
	if git(t, laptop, "rev-parse", "HEAD") != remoteHead(t, remote, "main") {
		t.Error("laptop and origin differ")
	}
	copies, _ := gitsync.FindConflictCopies(desk)
	t.Logf("res.Conflicts=%v copies=%v tree=%v", res.Conflicts, copies, tree(t, desk))
	return copies
}

func TestRecheckFolderFirstFileSecond(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "ideas/a.md", "in folder\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "ideas", "a file\n")
	c := settle(t, remote, laptop, desk, "in folder\n", "a file\n")
	if len(c) != 1 {
		t.Errorf("want one copy, got %v", c)
	}
}

func TestRecheckFileFirstFolderSecond(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "ideas", "a file\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "ideas/a.md", "in folder\n")
	c := settle(t, remote, laptop, desk, "in folder\n", "a file\n")
	if len(c) != 1 {
		t.Errorf("want one copy, got %v", c)
	}
}

func TestRecheckNestedFolderOfManyFiles(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	var versions []string
	for i := 0; i < 40; i++ {
		v := fmt.Sprintf("file %d\n", i)
		write(t, laptop, fmt.Sprintf("p/q/ideas/n%02d.md", i), v)
		versions = append(versions, v)
	}
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "p/q/ideas", "nested file\n")
	write(t, desk, "p/q/other.md", "other\n")
	c := settle(t, remote, laptop, desk, append(versions, "nested file\n", "other\n")...)
	if len(c) != 1 {
		t.Errorf("want one copy, got %v", c)
	}
}

// The note existed on both; one device edited it, the other replaced it
// with a folder of the same name.
func TestRecheckEditedNoteVersusFolder(t *testing.T) {
	t.Parallel()
	for _, editorFirst := range []bool{true, false} {
		t.Run(fmt.Sprint("editorFirst=", editorFirst), func(t *testing.T) {
			t.Parallel()
			remote, laptop, desk := pair(t, map[string]string{"ideas": "base\n"})
			editor, folder := laptop, desk
			if !editorFirst {
				editor, folder = desk, laptop
			}
			write(t, editor, "ideas", "base\nedited\n")
			remove(t, folder, "ideas")
			write(t, folder, "ideas/a.md", "in folder\n")
			first, second := laptop, desk
			wantOutcome(t, syncVault(t, first, "laptop", false), gitsync.Synced)
			_ = second
			settle(t, remote, laptop, desk, "base\nedited\n", "in folder\n")
		})
	}
}

// A real name with "~" in a modify/delete conflict, next to a folder whose
// name is its prefix: it must stay as it is (the edit wins), not become a
// conflict copy.
func TestRecheckRealTildeNameIsNotMovedAside(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{
		"Ideas~draft.md": "draft\n",
		"Ideas/a.md":     "a\n",
		"notes~1":        "same\n",
	})
	remove(t, laptop, "Ideas~draft.md")
	remove(t, laptop, "notes~1")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "Ideas~draft.md", "draft\nedited\n")
	write(t, desk, "notes~1", "same\nedited\n")
	c := settle(t, remote, laptop, desk, "draft\nedited\n", "a\n", "same\nedited\n")
	wantFile(t, desk, "Ideas~draft.md", "draft\nedited\n")
	wantFile(t, laptop, "notes~1", "same\nedited\n")
	if len(c) != 0 {
		t.Errorf("no copy expected, got %v", c)
	}
}

// Same clash differing only in letter case ("Ideas/" vs "ideas"): the
// file becomes a conflict copy the user is told about.
func TestRecheckCaseOnlyFileFolderClash(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "Ideas/a.md", "in folder\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "ideas", "a file\n")
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	wantAllVersions(t, desk, "in folder\n", "a file\n")
	wantNoMarkers(t, desk, "HEAD")
	if len(res.Conflicts) != 1 || !strings.HasPrefix(res.Conflicts[0], "ideas (conflict desk ") {
		t.Errorf("want the file as desk's conflict copy, got %v", res.Conflicts)
	}
	if copies, err := gitsync.FindConflictCopies(desk); err != nil || len(copies) != 1 {
		t.Errorf("FindConflictCopies = %v, %v; want one copy", copies, err)
	}
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantAllVersions(t, laptop, "in folder\n", "a file\n")
}

// Later syncs after a case-only file/folder clash, whichever device has
// the file and however deep: every round syncs, nothing is left
// uncommitted, origin's tree never changes after the union, and both
// versions stay in origin and on both disks.
func TestRecheckCaseOnlyClashLaterSyncs(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, first, firstPath, secondPath string }{
		{"folder first", "laptop", "Ideas/a.md", "ideas"},
		{"file first", "laptop", "ideas", "Ideas/a.md"},
		{"nested", "laptop", "P/q/Ideas/a.md", "p/Q/ideas"},
		{"file named like a grandparent", "laptop", "P/q/Ideas/a.md", "p/Q"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
			write(t, laptop, c.firstPath, "first\n")
			wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
			write(t, desk, c.secondPath, "second\n")
			var settled string
			for i := 0; i < 3; i++ {
				for _, v := range []struct{ dir, name string }{{desk, "desk"}, {laptop, "laptop"}} {
					res := syncVault(t, v.dir, v.name, false)
					wantOutcome(t, res, gitsync.Synced)
					if st := git(t, v.dir, "status", "--porcelain"); st != "" {
						t.Errorf("round %d %s: left uncommitted: %q", i, v.name, st)
					}
					wantAllVersions(t, v.dir, "first\n", "second\n")
				}
				now := git(t, remote, "ls-tree", "-r", "main")
				if i > 0 && now != settled {
					t.Errorf("round %d changed origin's tree:\n%s\nwas\n%s", i, now, settled)
				}
				settled = now
			}
			if got := len(strings.Split(settled, "\n")); got != 3 {
				t.Errorf("origin holds %d files, want seed and both versions:\n%s", got, settled)
			}
			for _, v := range []string{"first\n", "second\n"} {
				if !strings.Contains(git(t, remote, "grep", "-c", "-F", strings.TrimSpace(v), "main"), ":1") {
					t.Errorf("origin lost %q", v)
				}
			}
		})
	}
}

// Two devices create folders whose names differ only in letter case.
func TestRecheckCaseOnlyFolderFolder(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "Ideas/a.md", "from laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "ideas/b.md", "from desk\n")
	for i := 0; i < 3; i++ {
		for _, v := range []struct{ dir, name string }{{desk, "desk"}, {laptop, "laptop"}} {
			res := syncVault(t, v.dir, v.name, false)
			t.Logf("round %d %s: %v %v status=%q tree=%v", i, v.name, res.Outcome, res.Err, git(t, v.dir, "status", "--porcelain"), tree(t, v.dir))
		}
	}
	wantAllVersions(t, laptop, "from laptop\n", "from desk\n")
	wantAllVersions(t, desk, "from laptop\n", "from desk\n")
	t.Logf("origin tree: %v", git(t, remote, "ls-tree", "-r", "--name-only", "main"))
}

// plumbCommit writes a commit of parent plus files (path -> content) in the
// repository at dir through git plumbing, never the disk, so it can hold
// paths a case-insensitive disk cannot, as a commit made on Linux can.
func plumbCommit(t *testing.T, dir, parent string, files map[string]string) string {
	t.Helper()
	tmp := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "core.ignorecase=false"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(tmp, "index"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("read-tree", parent)
	for p, content := range files {
		src := filepath.Join(tmp, "blob")
		if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		oid := run("hash-object", "-w", src)
		run("update-index", "--add", "--cacheinfo", "100644,"+oid+","+p)
	}
	return run("commit-tree", run("write-tree"), "-p", parent, "-m", "made on linux\n\nMisoku-Device: linux")
}

// A device whose disk tells case apart (Linux) pushes "ideas" next to
// "Ideas/a.md"; the other device fast-forwards to it, its disk holds only
// one of them, and its next commits must not delete the other from
// origin.
func TestRecheckCaseClashFromOriginIsNotDeleted(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	c := plumbCommit(t, laptop, git(t, laptop, "rev-parse", "HEAD"), map[string]string{"ideas": "a file\n", "Ideas/a.md": "in folder\n"})
	git(t, laptop, "push", "-q", "origin", c+":refs/heads/main")
	want := git(t, remote, "ls-tree", "-r", "main")
	for i := 0; i < 2; i++ {
		wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	}
	write(t, desk, "other.md", "other\n") // a real change still syncs
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	got := git(t, remote, "ls-tree", "-r", "main")
	for _, line := range strings.Split(want, "\n") {
		if !strings.Contains(got, line) {
			t.Errorf("origin lost %q:\n%s", line, got)
		}
	}
	if !strings.Contains(got, "\tother.md") {
		t.Errorf("other.md did not reach origin:\n%s", got)
	}
}

// CommitVault on a case-insensitive vault (core.ignorecase) never commits
// the deletion of a path whose name, ignoring case, is taken by another
// tracked path that is still there; a real deletion still commits. The
// commit is built through plumbing and the disk holds what a case-
// insensitive disk could, so the test holds on any disk.
func TestCommitHoldsCaseClashDeletion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		ignoreCase bool
		onDisk     map[string]string // besides a.md
		want       []string          // HEAD's tree after the commit
		held       string            // the CaseClash notice, if any
	}{
		{"file missing, folder there", true, map[string]string{"Ideas/a.md": "in folder\n"}, []string{"Ideas/a.md", "a.md", "ideas"}, "ideas"},
		{"folder missing, file there", true, map[string]string{"ideas": "a file\n"}, []string{"Ideas/a.md", "a.md", "ideas"}, "Ideas/a.md"},
		{"both really deleted", true, nil, []string{"a.md"}, ""},
		{"disk tells case apart", false, map[string]string{"Ideas/a.md": "in folder\n"}, []string{"Ideas/a.md", "a.md"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			isolate(t)
			dir := readyVault(t) // tracks a.md
			for p, content := range c.onDisk {
				write(t, dir, p, content)
			}
			commit := plumbCommit(t, dir, git(t, dir, "rev-parse", "HEAD"), map[string]string{
				"ideas": "a file\n", "Ideas/a.md": "in folder\n", "gone.md": "g\n",
			})
			git(t, dir, "update-ref", "HEAD", commit)
			git(t, dir, "read-tree", "HEAD")
			git(t, dir, "config", "core.ignorecase", fmt.Sprint(c.ignoreCase))
			res := commitVault(t, dir)
			if !res.Committed {
				t.Fatal("the real deletion of gone.md was not committed")
			}
			wantTree(t, dir, c.want...)
			var held []string
			for _, n := range res.Notices {
				if n.Kind == gitsync.CaseClash {
					held = append(held, n.Path)
				}
			}
			if want := []string{c.held}; c.held == "" && len(held) != 0 || c.held != "" && !slices.Equal(held, want) {
				t.Errorf("CaseClash notices %v, want %q", held, c.held)
			}
		})
	}
}

// A conflict copy is never named like a folder of the union, ignoring
// case: that would be the same clash again.
func TestRecheckCaseClashCopyAvoidsFolderName(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	taken := "Ideas (conflict desk " + syncTime.Format("2006-01-02 1504") + ")"
	write(t, laptop, "Ideas/a.md", "in folder\n")
	write(t, laptop, taken+"/x.md", "x\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "ideas", "a file\n")
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if want := strings.ToLower(taken) + " 2"; len(res.Conflicts) != 1 || res.Conflicts[0] != want {
		t.Errorf("conflict copies %v, want [%q]", res.Conflicts, want)
	}
	wantAllVersions(t, desk, "in folder\n", "a file\n", "x\n")
}
