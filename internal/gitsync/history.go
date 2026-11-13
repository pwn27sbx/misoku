package gitsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// ErrNotARepo is returned by FileHistory when the vault is not inside a git
// repository.
var ErrNotARepo = i18n.Err(i18n.GitNotARepo)

// ErrNoGit is returned by FileHistory when git is not installed or does not
// start.
var ErrNoGit = i18n.Err(i18n.GitNotInstalled)

// Version is one commit that changed a note.
type Version struct {
	// Hash is the commit's full hash; Date its author date, YYYY-MM-DD;
	// Subject the first line of its message.
	Hash, Date, Subject string
	// Path is the note's path in that commit, relative to the repository's
	// top folder, slash-separated: what FileAt reads it by. It differs from
	// today's path in the commits before a rename.
	Path string
}

// FileHistory lists the commits that changed the note at path (relative to
// the folder r runs in), newest first, following it across renames. A note
// git never saw has none. Merges that only carry the note over are left
// out: they show no version of their own.
func (r *Runner) FileHistory(ctx context.Context, path string) ([]Version, error) {
	out, err := r.Run(ctx, "log", "--follow", "--format=%H%x00%ad%x00%s", "--date=short", "--name-only", "-z", "--", path)
	if err != nil {
		return nil, historyError(err)
	}
	return parseHistoryZ(out), nil
}

// FileAt returns the note as it was in commit hash, at path relative to the
// repository's top folder (Version.Path).
func (r *Runner) FileAt(ctx context.Context, hash, path string) ([]byte, error) {
	out, err := r.Run(ctx, "show", hash+":"+path)
	if err != nil {
		return nil, historyError(err)
	}
	return out, nil
}

// historyError names the two failures the user can fix (no git, no
// repository) with their sentinel errors, keeping git's own text.
func historyError(err error) error {
	switch {
	case errors.Is(err, exec.ErrNotFound):
		return fmt.Errorf("%w (%v)", ErrNoGit, err)
	case exitCode(err) == 128 && strings.Contains(err.Error(), "not a git repository"):
		return fmt.Errorf("%w (%v)", ErrNotARepo, err)
	}
	return err
}

// parseHistoryZ reads `git log --format=%H%x00%ad%x00%s --name-only -z`:
// each commit is its hash, date and subject separated by NULs, then a
// NUL-ended "\n<path>" when the commit changed the file.
func parseHistoryZ(out []byte) []Version {
	fields := bytes.Split(out, []byte{0})
	var versions []Version
	for i := 0; i+2 < len(fields); {
		v := Version{Hash: string(fields[i]), Date: string(fields[i+1]), Subject: string(fields[i+2])}
		i += 3
		if i < len(fields) && bytes.HasPrefix(fields[i], []byte{'\n'}) {
			v.Path = string(fields[i][1:])
			i++
		}
		if v.Hash != "" && v.Path != "" {
			versions = append(versions, v)
		}
	}
	return versions
}
