package gitsync

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// MaxFileSize is the largest file Misoku commits, in bytes (D7). GitHub
// refuses files over 100 MB, and one such file in a commit would block
// every later push, so bigger files stay out of the commit and the user is
// told about them instead.
var MaxFileSize int64 = 50 << 20

// excludes are the patterns Misoku keeps in .git/info/exclude: the
// temporary files of an atomic save (internal/core writes
// ".<name>.<random>.tmp" next to the note) and the folder data macOS
// leaves everywhere.
var excludes = []string{".*.tmp", ".DS_Store"}

// BigFile is a file left out of a commit because it is larger than
// MaxFileSize. Path is slash-separated, relative to the vault.
type BigFile struct {
	Path string
	Size int64
}

// CommitResult says what CommitVault did.
type CommitResult struct {
	// Committed is false when there was nothing to commit; then ID is
	// empty and Files is zero.
	Committed bool
	// ID is the new commit's full hash.
	ID string
	// Files counts the paths the commit changed; a rename counts once.
	Files int
	// Skipped lists the files left out for being too big, also when
	// nothing was committed. The caller tells the user about them.
	Skipped []BigFile
}

// CommitVault records every change in the vault as one local commit (S3):
// new, modified, deleted and renamed files of any kind, images and other
// attachments included. It never commits Misoku's temporary files,
// .DS_Store or files larger than MaxFileSize; a tracked file that grew past
// the limit keeps its last committed version. With nothing to commit it
// makes no commit. The message names device and carries a
// "Misoku-Device: <device>" trailer (S1).
//
// It only touches the local repository: no fetch, merge or push. The
// caller holds the vault's Lock.
func CommitVault(ctx context.Context, r *Runner, device string) (CommitResult, error) {
	var res CommitResult
	if err := ensureExcludes(ctx, r); err != nil {
		return res, err
	}
	changes, err := r.Status(ctx)
	if err != nil {
		return res, err
	}
	var add, keepOut []string
	for _, c := range changes {
		if ownFile(c.Path) {
			keepOut = append(keepOut, c.Path)
			continue
		}
		if info, err := os.Lstat(filepath.Join(r.dir, filepath.FromSlash(c.Path))); err == nil && info.Mode().IsRegular() && info.Size() > MaxFileSize {
			res.Skipped = append(res.Skipped, BigFile{Path: c.Path, Size: info.Size()})
			keepOut = append(keepOut, c.Path)
			continue
		}
		add = append(add, c.Path)
	}

	// Whatever was staged by hand for a file Misoku leaves out goes back
	// to its committed state, so the commit below cannot carry it. Before
	// the first commit, reset takes it out of the index.
	var staged []string
	for _, c := range changes {
		if c.Code[0] != ' ' && c.Code[0] != '?' && slices.Contains(keepOut, c.Path) {
			staged = append(staged, c.Path)
		}
	}
	if len(staged) > 0 {
		if err := withPathspecs(ctx, r, staged, "reset", "--quiet"); err != nil {
			return res, err
		}
	}
	if len(add) > 0 {
		if err := withPathspecs(ctx, r, add, "add", "--all"); err != nil {
			return res, err
		}
	}

	// On an unborn branch `diff --cached` compares with the empty tree.
	out, err := r.Run(ctx, "diff", "--cached", "--name-only", "-z", "--find-renames", "--no-ext-diff", "--no-color")
	if err != nil {
		return res, err
	}
	files := bytes.Count(out, []byte{0})
	if files == 0 {
		return res, nil
	}

	noun := "files"
	if files == 1 {
		noun = "file"
	}
	msg := fmt.Sprintf("misoku: %d %s changed on %s\n\nMisoku-Device: %s\n", files, noun, device, device)
	if _, err := r.Commit(ctx, msg); err != nil {
		return res, err
	}
	out, err = r.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return res, err
	}
	res.Committed, res.ID, res.Files = true, strings.TrimSpace(string(out)), files
	return res, nil
}

// ownFile reports whether p (slash-separated) is one of the files Misoku
// never commits, wherever it sits in the vault.
func ownFile(p string) bool {
	base := path.Base(p)
	for _, pattern := range excludes {
		if ok, _ := path.Match(pattern, base); ok {
			return true
		}
	}
	return false
}

// withPathspecs runs git args with paths as literal pathspecs. They travel
// in a NUL-separated file, never on a command line or through a shell, so
// any name works, however long the list.
func withPathspecs(ctx context.Context, r *Runner, paths []string, args ...string) error {
	f, err := os.CreateTemp("", "misoku-pathspec-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	var buf bytes.Buffer
	for _, p := range paths {
		buf.WriteString(":(literal)" + p)
		buf.WriteByte(0)
	}
	_, err = f.Write(buf.Bytes())
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	_, err = r.Run(ctx, append(args, "--pathspec-from-file="+f.Name(), "--pathspec-file-nul")...)
	return err
}

// ensureExcludes appends to the repository's info/exclude the patterns of
// excludes it does not hold yet. The user's lines, and their .gitignore,
// stay as they are.
func ensureExcludes(ctx context.Context, r *Runner) error {
	out, err := r.Run(ctx, "rev-parse", "--git-path", "info/exclude")
	if err != nil {
		return err
	}
	p := filepath.FromSlash(strings.TrimSpace(string(out)))
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.dir, p)
	}
	old, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	have := map[string]bool{}
	for _, line := range strings.Split(string(old), "\n") {
		have[strings.TrimSpace(line)] = true
	}
	var add strings.Builder
	for _, pattern := range excludes {
		if !have[pattern] {
			add.WriteString(pattern + "\n")
		}
	}
	if add.Len() == 0 {
		return nil
	}
	text := "# Misoku: temporary files of a save, and macOS folder data\n" + add.String()
	if len(old) > 0 && old[len(old)-1] != '\n' {
		text = "\n" + text
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(text)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
