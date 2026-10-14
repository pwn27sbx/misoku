package gitsync_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// write creates the file at the slash-separated path name inside dir.
func write(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitVault(t *testing.T, dir string) gitsync.CommitResult {
	t.Helper()
	res, err := gitsync.CommitVault(context.Background(), gitsync.New(dir, 10*time.Second), "mac")
	if err != nil {
		t.Fatalf("CommitVault: %v", err)
	}
	return res
}

// tree lists every path in HEAD's tree, exactly as stored.
func tree(t *testing.T, dir string) []string {
	t.Helper()
	out := git(t, dir, "ls-tree", "-r", "-z", "--name-only", "HEAD")
	var paths []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	slices.Sort(paths)
	return paths
}

func wantTree(t *testing.T, dir string, want ...string) {
	t.Helper()
	slices.Sort(want)
	if got := tree(t, dir); !slices.Equal(got, want) {
		t.Errorf("HEAD holds %q, want %q", got, want)
	}
}

// withLimit lowers the big-file limit for one test.
func withLimit(t *testing.T, n int64) {
	t.Helper()
	old := gitsync.MaxFileSize
	gitsync.MaxFileSize = n
	t.Cleanup(func() { gitsync.MaxFileSize = old })
}

func TestCommitVaultMakesTheFirstCommit(t *testing.T) {
	isolate(t)
	dir := newRepo(t)
	names := []string{"a.md", "img/photo.png", "ñandú con espacio.md", "it's.md"}
	if runtime.GOOS != "windows" {
		names = append(names, `quote"d.md`, "star*.md", ":colon.md")
	}
	for _, n := range names {
		write(t, dir, n, "# "+n+"\n")
	}

	res := commitVault(t, dir)
	if !res.Committed || res.Files != len(names) {
		t.Fatalf("want a commit of %d files, got %+v", len(names), res)
	}
	if head := git(t, dir, "rev-parse", "HEAD"); res.ID != head {
		t.Errorf("ID = %q, HEAD is %q", res.ID, head)
	}
	wantTree(t, dir, names...)
	if subject := git(t, dir, "log", "-1", "--format=%s"); subject != "misoku: "+strconv.Itoa(len(names))+" files changed on mac" {
		t.Errorf("subject = %q", subject)
	}
	if dev := git(t, dir, "log", "-1", "--format=%(trailers:key=Misoku-Device,valueonly)"); dev != "mac" {
		t.Errorf("Misoku-Device trailer = %q, want mac", dev)
	}
	if st := git(t, dir, "status", "--porcelain"); st != "" {
		t.Errorf("everything should be committed, status:\n%s", st)
	}
}

func TestCommitVaultRecordsEveryKindOfChange(t *testing.T) {
	isolate(t)
	dir := newRepo(t)
	write(t, dir, "a.md", "old\n")
	write(t, dir, "b.md", "bye\n")
	write(t, dir, "c.png", "picture bytes that rename detection can follow\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "first")

	write(t, dir, "a.md", "new\n")
	if err := os.Remove(filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "c.png"), filepath.Join(dir, "c2.png")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "notes/e.md", "e\n")

	res := commitVault(t, dir)
	if !res.Committed || res.Files != 4 {
		t.Fatalf("want 4 changes (modified, deleted, renamed, new), got %+v", res)
	}
	wantTree(t, dir, "a.md", "c2.png", "notes/e.md")
	if got := git(t, dir, "show", "HEAD:a.md"); got != "new" {
		t.Errorf("a.md committed as %q", got)
	}
	if subject := git(t, dir, "log", "-1", "--format=%s"); subject != "misoku: 4 files changed on mac" {
		t.Errorf("subject = %q", subject)
	}
}

func TestCommitVaultSaysWhenThereIsNothingToCommit(t *testing.T) {
	isolate(t)
	dir := readyVault(t)
	head := git(t, dir, "rev-parse", "HEAD")

	res := commitVault(t, dir)
	if res.Committed || res.ID != "" || res.Files != 0 {
		t.Errorf("a clean vault must not commit, got %+v", res)
	}
	if now := git(t, dir, "rev-parse", "HEAD"); now != head {
		t.Errorf("HEAD moved from %s to %s", head, now)
	}

	// An empty repository with nothing in it: no commit, no error.
	empty := newRepo(t)
	if res := commitVault(t, empty); res.Committed {
		t.Errorf("an empty vault must not commit, got %+v", res)
	}
}

func TestCommitVaultSaysOneFile(t *testing.T) {
	isolate(t)
	dir := readyVault(t)
	write(t, dir, "a.md", "changed\n")
	commitVault(t, dir)
	if subject := git(t, dir, "log", "-1", "--format=%s"); subject != "misoku: 1 file changed on mac" {
		t.Errorf("subject = %q", subject)
	}
}

func TestCommitVaultNeverCommitsTempFilesOrDSStore(t *testing.T) {
	isolate(t)
	dir := newRepo(t)
	write(t, dir, ".a.md.123456.tmp", "half written")
	write(t, dir, "sub/.b.md.42.tmp", "half written")
	write(t, dir, ".DS_Store", "finder")
	write(t, dir, "sub/.DS_Store", "finder")

	if res := commitVault(t, dir); res.Committed {
		t.Fatalf("only temporary files: nothing to commit, got %+v", res)
	}

	write(t, dir, "sub/n.md", "note\n")
	res := commitVault(t, dir)
	if !res.Committed || res.Files != 1 {
		t.Fatalf("want one note committed, got %+v", res)
	}
	wantTree(t, dir, "sub/n.md")
}

func TestCommitVaultLeavesOutTrackedTempFilesAndDSStore(t *testing.T) {
	isolate(t)
	dir := newRepo(t)
	// Committed by hand before Misoku synced this vault.
	write(t, dir, ".DS_Store", "v1")
	write(t, dir, "n.md", "v1\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "by hand")

	write(t, dir, ".DS_Store", "v2")
	write(t, dir, ".n.md.7.tmp", "half")
	git(t, dir, "add", "-f", ".DS_Store", ".n.md.7.tmp") // even if staged
	write(t, dir, "n.md", "v2\n")

	res := commitVault(t, dir)
	if !res.Committed || res.Files != 1 {
		t.Fatalf("want only n.md committed, got %+v", res)
	}
	wantTree(t, dir, ".DS_Store", "n.md")
	if got := git(t, dir, "show", "HEAD:.DS_Store"); got != "v1" {
		t.Errorf(".DS_Store committed as %q, want the old v1", got)
	}
}

func excludeOf(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCommitVaultWritesItsExclusionsOnce(t *testing.T) {
	isolate(t)
	dir := newRepo(t)
	ignore := "secret.md\r\n"
	write(t, dir, ".gitignore", ignore)
	// The user's own lines, CRLF and no final newline, one of ours already.
	write(t, dir, ".git/info/exclude", "# mine\r\n.DS_Store\r\nprivate/")
	write(t, dir, "n.md", "n\n")

	commitVault(t, dir)
	write(t, dir, "m.md", "m\n")
	commitVault(t, dir)

	ex := excludeOf(t, dir)
	if !strings.HasPrefix(ex, "# mine\r\n.DS_Store\r\nprivate/\n") {
		t.Errorf("the user's lines must stay as they were:\n%q", ex)
	}
	lines := strings.Split(strings.ReplaceAll(ex, "\r\n", "\n"), "\n")
	for _, want := range []string{".*.tmp", ".DS_Store"} {
		if n := slices.Index(lines, want); n < 0 || slices.Contains(lines[n+1:], want) {
			t.Errorf("exclude must hold %q exactly once:\n%s", want, ex)
		}
	}
	if got := git(t, dir, "show", "HEAD:.gitignore"); got != "secret.md" {
		t.Errorf(".gitignore changed: %q", got)
	}
	b, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if string(b) != ignore {
		t.Errorf(".gitignore on disk changed: %q", b)
	}
}

func TestCommitVaultCreatesTheExcludeFile(t *testing.T) {
	isolate(t)
	dir := newRepo(t)
	if err := os.RemoveAll(filepath.Join(dir, ".git", "info")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, ".DS_Store", "finder")
	write(t, dir, "n.md", "n\n")
	commitVault(t, dir)
	wantTree(t, dir, "n.md")
	if ex := excludeOf(t, dir); !strings.Contains(ex, ".*.tmp\n") || !strings.Contains(ex, ".DS_Store\n") {
		t.Errorf("exclude = %q", ex)
	}
}

func TestCommitVaultLeavesOutBigFiles(t *testing.T) {
	isolate(t)
	withLimit(t, 10)
	dir := newRepo(t)
	write(t, dir, "big file.png", "eleven byte")
	write(t, dir, "exact.md", "ten bytes!")
	write(t, dir, "small.md", "small")

	res := commitVault(t, dir)
	if !res.Committed || res.Files != 2 {
		t.Fatalf("want the two small files committed, got %+v", res)
	}
	wantTree(t, dir, "exact.md", "small.md")
	want := []gitsync.BigFile{{Path: "big file.png", Size: 11}}
	if !slices.Equal(res.Skipped, want) {
		t.Errorf("Skipped = %+v, want %+v", res.Skipped, want)
	}
	if st := git(t, dir, "status", "--porcelain", "-z"); st != "?? big file.png\x00" {
		t.Errorf("the big file must stay untracked, status %q", st)
	}

	// Only the big file left: nothing to commit, still reported.
	res = commitVault(t, dir)
	if res.Committed || !slices.Equal(res.Skipped, want) {
		t.Errorf("want no commit and the warning again, got %+v", res)
	}
}

func TestCommitVaultKeepsTheOldVersionOfAFileThatGrew(t *testing.T) {
	isolate(t)
	withLimit(t, 10)
	dir := newRepo(t)
	write(t, dir, "photo.png", "small")
	write(t, dir, "n.md", "v1")
	commitVault(t, dir)

	write(t, dir, "photo.png", "now far too big")
	write(t, dir, "n.md", "v2")
	git(t, dir, "add", "photo.png") // staged by hand: still left out

	res := commitVault(t, dir)
	if !res.Committed || res.Files != 1 {
		t.Fatalf("want only n.md committed, got %+v", res)
	}
	if got := git(t, dir, "show", "HEAD:photo.png"); got != "small" {
		t.Errorf("photo.png committed as %q, want its old version", got)
	}
	if want := []gitsync.BigFile{{Path: "photo.png", Size: 15}}; !slices.Equal(res.Skipped, want) {
		t.Errorf("Skipped = %+v, want %+v", res.Skipped, want)
	}
}

func TestCommitVaultLeavesOutABigFileStagedBeforeTheFirstCommit(t *testing.T) {
	isolate(t)
	withLimit(t, 10)
	dir := newRepo(t)
	write(t, dir, "big.png", "far too big for it")
	write(t, dir, "n.md", "n")
	git(t, dir, "add", "big.png")

	res := commitVault(t, dir)
	if !res.Committed || res.Files != 1 {
		t.Fatalf("want only n.md committed, got %+v", res)
	}
	wantTree(t, dir, "n.md")
}
