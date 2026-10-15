package gitsync

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// conflictCopyRe matches the names conflictName gives conflict copies:
// "<name> (conflict [<device> ]YYYY-MM-DD HHMM)[ <n>][<ext>]".
var conflictCopyRe = regexp.MustCompile(`^.* \(conflict (?:.+ )?\d{4}-\d{2}-\d{2} \d{4}\)(?: \d+)?(?:\.[^.]*)?$`)

// FindConflictCopies lists the conflict copies in the vault at dir (S6,
// S9), slash-separated and sorted: every file named like one, whichever
// sync or device made it, so the list shows what is still to resolve.
// Git's own folders are skipped. It reads the disk only: no git runs.
func FindConflictCopies(dir string) ([]string, error) {
	var copies []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == dir {
				return err
			}
			return nil // a folder that cannot be read holds nothing to list
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if conflictCopyRe.MatchString(d.Name()) {
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			copies = append(copies, filepath.ToSlash(rel))
		}
		return nil
	})
	slices.Sort(copies)
	return copies, err
}

// Info describes a vault's sync for `:sync status` (S9).
type Info struct {
	// Remote is origin's URL without credentials (RedactURL).
	Remote string
	// Branch is the checked-out branch.
	Branch string
	// Pending counts the local commits origin does not have yet, as far as
	// the vault last heard from origin.
	Pending int
	// Conflicts lists the conflict copies on disk (FindConflictCopies).
	Conflicts []string
	// Err is what could not be read, if anything; the other fields hold
	// what could.
	Err error
}

// Inspect reads what `:sync status` shows about the vault r runs in. It
// only reads (git remote, symbolic-ref, rev-list and the disk) and needs
// no lock, so it can run while a sync does.
func Inspect(ctx context.Context, r *Runner) Info {
	var info Info
	var errs []error
	if out, err := r.Run(ctx, "remote", "get-url", "origin"); err != nil {
		errs = append(errs, err)
	} else {
		info.Remote = RedactURL(strings.TrimSpace(string(out)))
	}
	if out, err := r.Run(ctx, "symbolic-ref", "-q", "--short", "HEAD"); err != nil {
		errs = append(errs, err)
	} else {
		info.Branch = strings.TrimSpace(string(out))
	}
	info.Pending = pending(ctx, r, "")
	copies, err := FindConflictCopies(r.dir)
	if err != nil {
		errs = append(errs, err)
	}
	info.Conflicts = copies
	info.Err = redact(errors.Join(errs...))
	return info
}
