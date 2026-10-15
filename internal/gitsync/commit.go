package gitsync

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
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
	// Notices lists the files that do not sync as plain files (S15),
	// sorted by path, also when nothing was committed: nested
	// repositories and new files with names Windows cannot use are left
	// out; submodules and links to outside the vault are committed as git
	// stores them. The caller tells the user about them.
	Notices []Notice
}

// CommitVault records every change in the vault as one local commit (S3):
// new, modified, deleted and renamed files of any kind, images and other
// attachments included. It never commits Misoku's temporary files,
// .DS_Store or files larger than MaxFileSize; a tracked file that grew past
// the limit keeps its last committed version. Nor does it commit a nested
// git repository or a new file whose name Windows cannot check out (S15):
// both stay on disk, untouched, and are listed in Notices. Nor, on every
// OS, a new file whose name differs from another's only in letter case
// (CaseDuplicate), which macOS and Windows could not check out. On a disk
// that ignores letter case, a tracked file that another tracked path
// shadows (missing, or showing the other's content) keeps its committed
// version (CaseClash). With nothing to commit it makes no commit. The message names device and carries a
// "Misoku-Device: <device>" trailer (S1).
//
// It only touches the local repository: no fetch, merge or push. The
// caller holds the vault's Lock.
func CommitVault(ctx context.Context, r *Runner, device string) (res CommitResult, err error) {
	if err := ensureExcludes(ctx, r); err != nil {
		return res, err
	}
	changes, err := r.Status(ctx)
	if err != nil {
		return res, err
	}

	// While a new file is left out for its name, no deletion is
	// committed: a note renamed to such a name shows as the deletion of
	// its old path, and committing that alone would delete the note on
	// every other device. The old path keeps syncing until the user
	// renames the file to a name Windows can use.
	// The same holds while a new file is left out as a case duplicate.
	names, err := caseNamesOf(ctx, r, changes)
	if err != nil {
		return res, err
	}
	hold := slices.ContainsFunc(changes, func(c Change) bool {
		return newPath(c) && !nestedRepo(c) && !ownFile(c.Path) && (windowsNameProblem(c.Path) != "" || names.dup[c.Path])
	})
	var add, keepOut, restore []string
	holdBack := func(p string, staged bool) {
		res.Notices = append(res.Notices, Notice{Kind: HeldDeletion, Path: p})
		if staged {
			restore = append(restore, p)
		}
	}
	settled := map[string]bool{} // paths whose notice, if any, is decided here
	for _, c := range changes {
		if ownFile(c.Path) {
			keepOut = append(keepOut, c.Path)
			continue
		}
		if nestedRepo(c) {
			p := strings.TrimSuffix(c.Path, "/")
			res.Notices = append(res.Notices, Notice{Kind: NestedRepo, Path: p, LeftOut: true})
			continue
		}
		if info, err := os.Lstat(filepath.Join(r.dir, filepath.FromSlash(c.Path))); err == nil && info.Mode().IsRegular() && info.Size() > MaxFileSize {
			res.Skipped = append(res.Skipped, BigFile{Path: c.Path, Size: info.Size()})
			keepOut = append(keepOut, c.Path)
			continue
		}
		if c.Code[0] == 'R' && hold {
			holdBack(c.From, true) // the rename's deletion half
			settled[c.From] = true
		}
		if newPath(c) {
			if why := windowsNameProblem(c.Path); why != "" {
				res.Notices = append(res.Notices, Notice{Kind: WindowsName, Path: c.Path, Detail: why, LeftOut: true})
				keepOut = append(keepOut, c.Path)
				settled[c.Path] = true
				continue
			}
			if names.dup[c.Path] {
				res.Notices = append(res.Notices, Notice{Kind: CaseDuplicate, Path: c.Path, LeftOut: true})
				keepOut = append(keepOut, c.Path)
				settled[c.Path] = true
				continue
			}
			if n, ok := linkOutside(r.dir, c.Path); ok {
				res.Notices = append(res.Notices, n)
			}
			settled[c.Path] = true
		}
		if strings.Contains(c.Code, "D") {
			settled[c.Path] = true // on its way out of the vault
			if names.missing(c) {
				res.Notices = append(res.Notices, Notice{Kind: CaseClash, Path: c.Path, LeftOut: true})
				continue
			}
			if hold && (c.Code == " D" || c.Code == "D ") {
				holdBack(c.Path, c.Code[0] == 'D')
				continue
			}
		}
		if names.overwritten(c) {
			res.Notices = append(res.Notices, Notice{Kind: CaseClash, Path: c.Path, LeftOut: true})
			settled[c.Path] = true
			continue
		}
		add = append(add, c.Path)
	}
	for _, p := range names.shadowed {
		if !slices.Contains(res.Notices, Notice{Kind: CaseClash, Path: p, LeftOut: true}) {
			res.Notices = append(res.Notices, Notice{Kind: CaseClash, Path: p})
		}
	}
	// What the vault already tracks is read once the commit is made, so
	// reading it never delays the commit.
	defer res.noteTracked(ctx, r, settled)

	// Whatever was staged by hand for a file Misoku leaves out goes back
	// to its committed state, so the commit below cannot carry it. Before
	// the first commit, reset takes it out of the index.
	// A deletion held back above, staged by hand (`git rm`, `git mv`),
	// goes back to the index the same way.
	staged := restore
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

// caseNames is what CommitVault needs to keep names that differ only in
// letter case from reaching the repository or being lost to it. macOS and
// Windows hold one file for "a.md" and "A.md", and one name for a file
// "ideas" and a folder "Ideas"; Linux holds both. git sets core.ignorecase
// on a disk that ignores case.
type caseNames struct {
	ignoreCase bool
	files      map[string]int  // lowercased tracked file -> how many, gone ones aside
	folders    map[string]bool // lowercased folders of those files
	pairs      map[string]int  // lowercased tracked file -> how many, gone ones too
	shadowed   []string        // tracked files another tracked path shadows, sorted
	dup        map[string]bool // new paths left out as case duplicates
}

// caseNamesOf reads core.ignorecase and the tracked files.
func caseNamesOf(ctx context.Context, r *Runner, changes []Change) (caseNames, error) {
	n := caseNames{files: map[string]int{}, folders: map[string]bool{}, pairs: map[string]int{}, dup: map[string]bool{}}
	gone, fresh := map[string]bool{}, map[string]bool{}
	var news []string
	for _, c := range changes {
		if strings.Contains(c.Code, "D") {
			gone[c.Path] = true
		}
		if c.Code[0] == 'R' {
			gone[c.From] = true
		}
		if newPath(c) && !nestedRepo(c) && !ownFile(c.Path) {
			news = append(news, c.Path)
			fresh[c.Path] = true
		}
	}
	out, err := r.Run(ctx, "config", "--bool", "core.ignorecase")
	if err != nil && exitCode(err) != 1 { // 1 means unset
		return n, err
	}
	n.ignoreCase = strings.TrimSpace(string(out)) == "true"
	modes, err := tracked(ctx, r)
	if err != nil {
		return n, err
	}
	for p := range modes {
		if !fresh[p] {
			n.pairs[strings.ToLower(p)]++
		}
		if gone[p] || fresh[p] {
			continue
		}
		n.files[strings.ToLower(p)]++
		for d := path.Dir(p); d != "." && d != "/"; d = path.Dir(d) {
			n.folders[strings.ToLower(d)] = true
		}
	}
	// A new name taken by a tracked file, or by a new one earlier in
	// byte order, is a duplicate, on every OS.
	sort.Strings(news)
	seen := map[string]bool{}
	for _, p := range news {
		lp := strings.ToLower(p)
		n.dup[p] = n.files[lp] > 0 || seen[lp]
		seen[lp] = true
	}
	// On a disk that ignores case, the tracked files it cannot hold apart
	// are told about on every commit, edited or not, until one of them is
	// renamed.
	if n.ignoreCase {
		for p := range modes {
			lp := strings.ToLower(p)
			if !fresh[p] && n.pairs[lp] > 1 {
				n.shadowed = append(n.shadowed, p)
			}
		}
		sort.Strings(n.shadowed)
	}
	return n, nil
}

// missing reports whether c is a tracked file missing from disk only
// because the disk ignores case and another tracked path takes its name: a
// folder ("ideas" next to "Ideas/a.md") or a file where one of its folders
// should be. It also holds a tracked file whose name differs from
// another's only in case ("a.md" next to "A.md"): the disk holds one file
// for both, and renaming or deleting it shows both missing, though at
// most one of them was ever there. Committing the deletion would delete it
// on every device.
func (n caseNames) missing(c Change) bool {
	if !n.ignoreCase || c.Code != " D" {
		return false
	}
	lp := strings.ToLower(c.Path)
	clash := n.folders[lp] || n.pairs[lp] > 1
	for d := path.Dir(lp); !clash && d != "." && d != "/"; d = path.Dir(d) {
		clash = n.files[d] > 0
	}
	return clash
}

// overwritten reports whether c is a tracked file shown modified on a
// disk that ignores case while another tracked file has its name ignoring
// case: the disk holds one file for both, so the change may be the other's
// content, and committing it would overwrite this one.
func (n caseNames) overwritten(c Change) bool {
	return n.ignoreCase && c.Code == " M" && n.files[strings.ToLower(c.Path)] > 1
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

// noteTracked adds the notices about what the index already holds
// (trackedNotices), except the paths this commit settled, and sorts them.
// They are warnings only: when the index cannot be read (the sync was
// cancelled, say), they are left out rather than failing a commit that
// is already made; the next commit reads them again.
func (res *CommitResult) noteTracked(ctx context.Context, r *Runner, settled map[string]bool) {
	if modes, err := tracked(ctx, r); err == nil {
		for _, n := range trackedNotices(r.dir, modes) {
			if !settled[n.Path] {
				res.Notices = append(res.Notices, n)
			}
		}
	}
	slices.SortFunc(res.Notices, func(a, b Notice) int { return strings.Compare(a.Path, b.Path) })
}
