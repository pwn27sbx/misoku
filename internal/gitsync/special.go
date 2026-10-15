package gitsync

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// NoticeKind says why a file does not sync like a plain note (S15).
type NoticeKind int

// The files that need the user's attention. None of them stops a sync:
// each is either left out of the commit or committed as git stores it,
// and the user is told which.
const (
	// NestedRepo is a folder with its own .git inside the vault. git
	// would only record which commit it is on, never its files, so it is
	// left out of the commit.
	NestedRepo NoticeKind = iota + 1
	// Submodule is a nested repository the vault already tracks (a
	// gitlink): only its commit id syncs, never its files.
	Submodule
	// LinkOutside is a symbolic link whose target is outside the vault.
	// git stores the link itself, so the target does not sync.
	LinkOutside
	// WindowsName is a name Windows cannot check out (a character such as
	// ":" or "*", a trailing dot or space, a reserved name such as CON or
	// NUL). A new file with such a name is left out of the commit, since
	// it would stop every sync on a Windows computer; one already tracked
	// keeps syncing, and the user is asked to rename it.
	WindowsName
	// HeldDeletion is a tracked file deleted or renamed while a new file
	// is left out for its name (WindowsName, CaseDuplicate): the deletion is not
	// committed, since it may be the old name of that file, so the file
	// keeps syncing under this name until the other is renamed.
	HeldDeletion
	// CaseClash is a tracked file deleted on a disk that ignores letter
	// case (macOS, Windows) while another tracked path takes its name
	// ignoring case ("ideas" next to "Ideas/a.md", made on Linux): the
	// disk can hold only one of them, so the deletion is not committed
	// and the file keeps syncing until the user renames the one shown. Two
	// tracked files that differ only in case are told about on every
	// commit on such a disk, changed or not, and neither one's deletion
	// or change is committed there.
	CaseClash
	// CaseDuplicate is a new file whose name differs from a tracked or
	// another new one only in letter case ("A.md" next to "a.md"),
	// possible on Linux: macOS and Windows cannot hold both, so it is left
	// out of the commit until the user renames it.
	CaseDuplicate
)

// Notice is a file CommitVault left out or committed with a warning (S15).
type Notice struct {
	Kind NoticeKind
	// Path is slash-separated and relative to the vault; a folder's has
	// no trailing slash.
	Path string
	// Detail is the link's target (LinkOutside) or what is wrong with the
	// name (WindowsName).
	Detail string
	// LeftOut is true when the file stays out of the commit.
	LeftOut bool
}

// String tells the user, in one line, what happens to the file and what
// to do about it.
func (n Notice) String() string {
	switch n.Kind {
	case NestedRepo:
		return n.Path + " is a separate git repository, so its files do not sync: move it out of the vault, or ignore it"
	case Submodule:
		return n.Path + " is a git submodule: only its commit id syncs, not its files"
	case LinkOutside:
		return n.Path + " links to " + n.Detail + ", outside the vault: only the link syncs, not what it points to"
	case WindowsName:
		if n.LeftOut {
			return n.Path + " has a name Windows cannot use (" + n.Detail + "), so it stays out of sync until you rename it to a valid name; a note renamed to it keeps syncing under its old name"
		}
		return n.Path + " has a name Windows cannot use (" + n.Detail + "): rename it, or a Windows computer cannot sync this vault"
	case CaseClash:
		return n.Path + " and another note or folder have names that differ only in letter case, so this computer holds only one of them: rename the file shown here and both sync again on every device"
	case CaseDuplicate:
		return n.Path + " has the name of another file, ignoring letter case, which macOS and Windows cannot hold together, so it stays out of sync until you rename it"
	case HeldDeletion:
		return n.Path + " was deleted or renamed, but it keeps syncing under this name until the files left out for their names are renamed"
	}
	return n.Path
}

// nestedRepo reports whether c is a folder git will not look into because
// it holds its own repository: with --untracked-files=all, status lists
// such a folder, with a trailing slash, instead of its files.
func nestedRepo(c Change) bool {
	return c.Code == "??" && strings.HasSuffix(c.Path, "/")
}

// newPath reports whether c is a path the repository does not hold yet:
// untracked, or added, renamed or copied in the index by hand.
func newPath(c Change) bool {
	return strings.ContainsRune("?ARC", rune(c.Code[0]))
}

// tracked lists the paths in the index with their git file mode
// ("100644", "120000" for a link, "160000" for a submodule).
func tracked(ctx context.Context, r *Runner) (map[string]string, error) {
	out, err := r.Run(ctx, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	modes := map[string]string{}
	for _, e := range bytes.Split(out, []byte{0}) {
		// "<mode> <object> <stage>\t<path>"
		meta, p, ok := bytes.Cut(e, []byte{'\t'})
		if !ok {
			continue
		}
		mode, _, _ := bytes.Cut(meta, []byte{' '})
		modes[string(p)] = string(mode)
	}
	return modes, nil
}

// trackedNotices warns about what the vault already tracks and does not
// sync as plain files: submodules, links that point outside the vault and
// names Windows cannot use. They are checked on every commit, not only
// when they change, so the warning stays until the user acts on it.
func trackedNotices(dir string, modes map[string]string) []Notice {
	var notices []Notice
	for p, mode := range modes {
		switch mode {
		case "160000":
			notices = append(notices, Notice{Kind: Submodule, Path: p})
			continue
		case "120000":
			if n, ok := linkOutside(dir, p); ok {
				notices = append(notices, n)
			}
		}
		if why := windowsNameProblem(p); why != "" {
			notices = append(notices, Notice{Kind: WindowsName, Path: p, Detail: why})
		}
	}
	return notices
}

// linkOutside reports whether the file at p is a symbolic link whose
// target is outside the vault: an absolute target (which other computers
// may not have) or one that climbs out of it.
func linkOutside(dir, p string) (Notice, bool) {
	target, err := os.Readlink(filepath.Join(dir, filepath.FromSlash(p)))
	if err != nil {
		return Notice{}, false // not a link on this disk (Windows may check one out as a file)
	}
	slash := filepath.ToSlash(target)
	inside := path.Clean(path.Join(path.Dir(p), slash))
	if filepath.IsAbs(target) || path.IsAbs(slash) || inside == ".." || strings.HasPrefix(inside, "../") {
		return Notice{Kind: LinkOutside, Path: p, Detail: target}, true
	}
	return Notice{}, false
}

// windowsReserved are the device names Windows reserves in any folder,
// with or without an extension.
var windowsReserved = map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}

func init() {
	for i := '1'; i <= '9'; i++ {
		windowsReserved["COM"+string(i)] = true
		windowsReserved["LPT"+string(i)] = true
	}
}

// windowsNameProblem says why Windows cannot check out the slash-separated
// path p, or returns "" when it can.
func windowsNameProblem(p string) string {
	for _, part := range strings.Split(p, "/") {
		if part == "" {
			continue
		}
		for _, c := range part {
			if c < 0x20 {
				return "a control character"
			}
			if strings.ContainsRune(`<>:"\|?*`, c) {
				return fmt.Sprintf("%q", string(c))
			}
		}
		if strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return "it ends with a dot or a space"
		}
		stem, _, _ := strings.Cut(part, ".")
		if stem = strings.TrimRight(stem, " "); windowsReserved[strings.ToUpper(stem)] {
			return stem + " is reserved on Windows"
		}
	}
	return ""
}
