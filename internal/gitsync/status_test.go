package gitsync_test

import (
	"context"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// S9: the copies are found on disk, wherever they are and whoever made
// them, not only the ones the last sync created.
func TestFindConflictCopiesSearchesTheDisk(t *testing.T) {
	t.Parallel()
	_, _, desk, res := conflict(t, "notes/n.md", "one\n", "from laptop\n", "from desk\n")
	wantOutcome(t, res, gitsync.Synced)
	for _, name := range []string{
		"a (conflict desk 2026-10-14 1530).md",
		"img/p (conflict 2026-10-14 1530) 2.png",
		".hidden (conflict mac 2026-01-02 0304)",
		"a (conflicting).md",
		"notes (conflict).md",
		"plain.md",
		".git/x (conflict d 2026-10-14 1530).md",
	} {
		write(t, desk, name, "x")
	}
	got, err := gitsync.FindConflictCopies(desk)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		".hidden (conflict mac 2026-01-02 0304)",
		"a (conflict desk 2026-10-14 1530).md",
		"img/p (conflict 2026-10-14 1530) 2.png",
		"notes/n (conflict laptop " + stamp + ").md",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("FindConflictCopies = %q, want %q", got, want)
	}
}

func TestRedactURLRemovesCredentials(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ in, want string }{
		{"https://me:s3cret@example.com/notes.git", "https://example.com/notes.git"},
		{"https://ghp_TOKEN@github.com/me/notes.git", "https://github.com/me/notes.git"},
		{"https://example.com/notes.git?access_token=abc", "https://example.com/notes.git"},
		{"ssh://git:pw@example.com/notes.git", "ssh://git@example.com/notes.git"},
		{"ssh://git@example.com/notes.git", "ssh://git@example.com/notes.git"},
		{"git@github.com:me/notes.git", "git@github.com:me/notes.git"},
		{"/srv/git/notes.git", "/srv/git/notes.git"},
	} {
		if got := gitsync.RedactURL(c.in); got != c.want {
			t.Errorf("RedactURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// S9 `:sync status`: origin without its credentials, the branch, what is
// pending and the conflict copies on disk.
func TestInspectDescribesTheVault(t *testing.T) {
	t.Parallel()
	_, _, desk, res := conflict(t, "n.md", "one\n", "from laptop\n", "from desk\n")
	wantOutcome(t, res, gitsync.Synced)
	write(t, desk, "new.md", "x")
	git(t, desk, "add", "new.md")
	git(t, desk, "commit", "-q", "-m", "local")
	git(t, desk, "remote", "set-url", "origin", "https://me:s3cret@example.invalid/notes.git")

	info := gitsync.Inspect(context.Background(), gitsync.New(desk, 10*time.Second))
	if info.Err != nil {
		t.Fatal(info.Err)
	}
	if info.Remote != "https://example.invalid/notes.git" || info.Branch != "main" || info.Pending != 1 {
		t.Errorf("want origin without credentials, main and 1 pending, got %+v", info)
	}
	if want := []string{"n (conflict laptop " + stamp + ").md"}; !slices.Equal(info.Conflicts, want) {
		t.Errorf("Conflicts = %q, want %q", info.Conflicts, want)
	}
}

// R4: a git that fails for another reason (a slow disk, a timeout) is not
// "not a git repository", "no remote" or "no identity": that advice is
// wrong for a vault that is set up. The error says what git answered.
func TestDetectReportsGitFailuresAsErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake git is a shell script")
	}
	isolate(t)
	dir := readyVault(t)
	for _, c := range []struct{ name, failing, wrongAdvice string }{
		{"top folder", "rev-parse --show-toplevel", "misoku sync init"},
		{"remote", "remote", "git remote add"},
		{"identity", "config --get user.name", "git config --global"},
	} {
		t.Run(c.name, func(t *testing.T) {
			fakeGit(t, `case "$*" in
*version*) echo "git version 2.54.0" ;;
*"`+c.failing+`"*) echo "fatal: unable to read: Input/output error" >&2; exit 2 ;;
*"rev-parse --show-toplevel"*) echo "$2" ;;
*" remote") echo origin ;;
*"config --get"*) echo someone ;;
*) exit 0 ;;
esac`, "")
			st := detect(t, dir)
			if st.Status != gitsync.Paused || st.Reason != gitsync.ReasonGitFailed {
				t.Fatalf("want paused with %q, got %+v", gitsync.ReasonGitFailed, st)
			}
			if !strings.Contains(st.Message, "Input/output error") || strings.Contains(st.Message, c.wrongAdvice) {
				t.Errorf("the message must carry git's error, not %q: %q", c.wrongAdvice, st.Message)
			}
		})
	}
}
