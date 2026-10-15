package gitsync

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// gitlinkMode is a submodule entry: a commit pointer, not a file.
const gitlinkMode = "160000"

// stageEntry is one side of a conflicted path in `merge-tree` output.
type stageEntry struct{ mode, oid string }

// union joins ours and theirs without touching the working tree (D4) and
// returns the new commit, whose parents are ours and theirs, with the
// conflict copies it holds.
//
// `merge-tree --write-tree` merges everything it can on its own (other
// notes, other parts of the same note). For each path it could not, the
// result keeps both versions (D3, S6):
//
//   - changed on both sides (the same lines, an image, a note new on both):
//     ours stays at the path and theirs goes next to it as a conflict copy;
//   - deleted on one side and changed on the other: the changed one stays;
//   - renamed differently on both sides: both names stay.
//
// Two paths that differ only in letter case are one file on macOS and
// Windows, where applying both would overwrite one with the other: ours
// keeps its name and theirs becomes a conflict copy, on every OS.
//
// No conflict marker ever reaches a file.
func union(ctx context.Context, r *Runner, ours, theirs string, unrelated bool, device string, now time.Time) (string, []string, error) {
	args := []string{"merge-tree", "--write-tree", "-z", "--no-messages"}
	if unrelated {
		args = append(args, "--allow-unrelated-histories")
	}
	out, err := r.Run(ctx, append(args, ours, theirs)...)
	if err != nil && exitCode(err) != 1 { // 1 means "merged, with conflicts"
		return "", nil, fmt.Errorf("merge-tree: %w", err)
	}
	tree, conflicts, err := parseMergeTree(out)
	if err != nil {
		return "", nil, err
	}
	tree, copies, err := resolveTree(ctx, r, tree, conflicts, ours, theirs, now)
	if err != nil {
		return "", nil, err
	}
	msg := fmt.Sprintf("misoku: join changes from origin on %s\n\nMisoku-Device: %s\n", device, device)
	out, err = r.Run(ctx, "commit-tree", tree, "-p", ours, "-p", theirs, "-m", msg)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(string(out)), copies, nil
}

// parseMergeTree reads `merge-tree --write-tree -z --no-messages`: the
// tree, then one "<mode> <oid> <stage>\t<path>" entry per conflicted
// stage, all NUL-terminated. It returns the stages of each path (index
// 1 base, 2 ours, 3 theirs).
func parseMergeTree(out []byte) (string, map[string]*[4]*stageEntry, error) {
	fields := bytes.Split(out, []byte{0})
	tree := strings.TrimSpace(string(fields[0]))
	if tree == "" {
		return "", nil, fmt.Errorf("merge-tree gave no tree")
	}
	conflicts := map[string]*[4]*stageEntry{}
	for _, f := range fields[1:] {
		if len(f) == 0 {
			continue
		}
		head, p, ok := strings.Cut(string(f), "\t")
		parts := strings.Fields(head)
		if !ok || len(parts) != 3 {
			return "", nil, fmt.Errorf("merge-tree: unexpected entry %q", f)
		}
		stage, err := strconv.Atoi(parts[2])
		if err != nil || stage < 1 || stage > 3 {
			return "", nil, fmt.Errorf("merge-tree: unexpected stage in %q", f)
		}
		if conflicts[p] == nil {
			conflicts[p] = &[4]*stageEntry{}
		}
		conflicts[p][stage] = &stageEntry{mode: parts[0], oid: parts[1]}
	}
	return tree, conflicts, nil
}

// resolution is the union's tree as a list of files, with the edits that
// turn merge-tree's result into the one Misoku keeps.
type resolution struct {
	files   map[string]stageEntry // path -> file in the result
	taken   map[string]bool       // lowercased paths in use
	changes map[string]*stageEntry
	copies  []string
}

func (s *resolution) set(p string, e stageEntry) {
	s.files[p] = e
	s.taken[strings.ToLower(p)] = true
	s.changes[p] = &e
}

func (s *resolution) remove(p string) {
	delete(s.files, p)
	s.changes[p] = nil
}

// resolveTree turns merge-tree's tree into the one Misoku keeps (see union)
// and writes it in a temporary index, so the vault's own index and files
// are never touched. It returns the tree and the conflict copies.
func resolveTree(ctx context.Context, r *Runner, tree string, conflicts map[string]*[4]*stageEntry, ours, theirs string, now time.Time) (string, []string, error) {
	files, err := lsTree(ctx, r, tree)
	if err != nil {
		return "", nil, err
	}
	// Each side's own file, never a stage of merge-tree: when both sides
	// renamed and edited a note, merge-tree puts its content merge, with
	// conflict markers, at both stages.
	oursFiles, err := lsTree(ctx, r, ours)
	if err != nil {
		return "", nil, err
	}
	theirsFiles, err := lsTree(ctx, r, theirs)
	if err != nil {
		return "", nil, err
	}
	s := &resolution{files: files, taken: map[string]bool{}, changes: map[string]*stageEntry{}}
	for p := range files {
		s.taken[strings.ToLower(p)] = true
	}
	side := func(own map[string]stageEntry, p string, stage *stageEntry) *stageEntry {
		if stage == nil {
			return nil
		}
		if e, ok := own[p]; ok {
			return &e
		}
		return stage // the path is new to this side (a rename or a moved file)
	}
	// copyFrom keeps e, the version of p from the side whose commit is
	// from, as a conflict copy named for the device that changed it there
	// (other being the other side's commit).
	copyFrom := func(p string, e stageEntry, from, other string) error {
		device, err := deviceOf(ctx, r, other, from, p)
		if err != nil {
			return err
		}
		name, err := conflictName(r.dir, p, device, now, s.taken)
		if err != nil {
			return err
		}
		s.set(name, e)
		s.copies = append(s.copies, name)
		return nil
	}
	copyOf := func(p string, e stageEntry) error { return copyFrom(p, e, theirs, ours) }

	for _, p := range sortedKeys(conflicts) {
		st := conflicts[p]
		// A path with only its base left (renamed differently on both
		// sides) needs nothing: merge-tree's tree already drops the old
		// name and holds both new ones.
		mine, other := side(oursFiles, p, st[2]), side(theirsFiles, p, st[3])
		if orig, e, from, ok := movedAside(p, st, s.files, oursFiles, theirsFiles, ours, theirs); ok {
			// A note on one side, a folder of that name on the other:
			// merge-tree keeps the note as "<name>~<commit>". It becomes a
			// conflict copy named for the device it came from, so the
			// user is told (S6, S9).
			s.remove(p)
			if err := copyFrom(orig, e, from, otherSide(from, ours, theirs)); err != nil {
				return "", nil, err
			}
			continue
		}
		switch {
		case mine != nil && other != nil:
			s.set(p, *mine)
			if other.oid == mine.oid || other.mode == gitlinkMode {
				continue // nothing of theirs to lose
			}
			if err := copyOf(p, *other); err != nil {
				return "", nil, err
			}
		case mine != nil:
			s.set(p, *mine)
		case other != nil:
			s.set(p, *other)
		}
	}

	// Names that differ only in case: the one ours has stays, the others
	// become copies. With none from ours, the first in byte order stays.
	groups := map[string][]string{}
	for p := range s.files {
		groups[strings.ToLower(p)] = append(groups[strings.ToLower(p)], p)
	}
	for _, key := range sortedKeys(groups) {
		g := groups[key]
		if len(g) < 2 {
			continue
		}
		sort.Strings(g)
		keep := g[0]
		for _, p := range g {
			if _, ok := oursFiles[p]; ok {
				keep = p
				break
			}
		}
		for _, p := range g {
			if _, mine := oursFiles[p]; p == keep || mine {
				continue
			}
			e := s.files[p]
			s.remove(p)
			if err := copyOf(p, e); err != nil {
				return "", nil, err
			}
		}
	}

	if len(s.changes) == 0 {
		return tree, nil, nil
	}
	var info bytes.Buffer
	for _, p := range sortedKeys(s.changes) {
		if e := s.changes[p]; e != nil {
			fmt.Fprintf(&info, "%s %s\t%s\x00", e.mode, e.oid, p)
		} else { // mode 0 takes the path out of the index
			fmt.Fprintf(&info, "0 %s\t%s\x00", strings.Repeat("0", len(tree)), p)
		}
	}

	tmp, err := os.MkdirTemp("", "misoku-union-*")
	if err != nil {
		return "", nil, err
	}
	defer os.RemoveAll(tmp) // best effort: a temporary index
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(tmp, "index")}
	if _, err := r.runWith(ctx, env, nil, "read-tree", tree); err != nil {
		return "", nil, err
	}
	if _, err := r.runWith(ctx, env, info.Bytes(), "update-index", "-z", "--index-info"); err != nil {
		return "", nil, err
	}
	out, err := r.runWith(ctx, env, nil, "write-tree")
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(string(out)), s.copies, nil
}

// movedAside reports whether merge-tree moved the file at orig out of the
// way of a folder of the same name on the other side, keeping it at p
// ("<orig>~<label>"), present in one stage only: orig is a file of that
// side with the same content, and a folder in the merged files. It returns
// orig, the file, and the commit of the side it came from.
func movedAside(p string, st *[4]*stageEntry, merged, oursFiles, theirsFiles map[string]stageEntry, ours, theirs string) (string, stageEntry, string, bool) {
	i := strings.LastIndex(p, "~")
	if i <= 0 || (st[2] == nil) == (st[3] == nil) {
		return "", stageEntry{}, "", false
	}
	orig := p[:i]
	e, own, from := st[2], oursFiles, ours
	if e == nil {
		e, own, from = st[3], theirsFiles, theirs
	}
	if f, ok := own[orig]; !ok || f.oid != e.oid || f.mode != e.mode {
		return "", stageEntry{}, "", false
	}
	for q := range merged {
		if strings.HasPrefix(q, orig+"/") {
			return orig, *e, from, true
		}
	}
	return "", stageEntry{}, "", false
}

// otherSide is theirs when from is ours, and ours otherwise.
func otherSide(from, ours, theirs string) string {
	if from == ours {
		return theirs
	}
	return ours
}

// lsTree lists every file of a tree or commit.
func lsTree(ctx context.Context, r *Runner, treeish string) (map[string]stageEntry, error) {
	out, err := r.Run(ctx, "ls-tree", "-r", "-z", "--full-tree", treeish)
	if err != nil {
		return nil, err
	}
	files := map[string]stageEntry{}
	for _, f := range strings.Split(string(out), "\x00") {
		head, p, ok := strings.Cut(f, "\t")
		parts := strings.Fields(head)
		if !ok || len(parts) != 3 {
			continue
		}
		files[p] = stageEntry{mode: parts[0], oid: parts[2]}
	}
	return files, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// deviceOf returns the device that last changed p on theirs' side (the
// commits of theirs that ours lacks), from
// its Misoku-Device trailer, made safe for a file name; "" when that
// commit was not made by Misoku.
func deviceOf(ctx context.Context, r *Runner, ours, theirs, p string) (string, error) {
	out, err := r.Run(ctx, "log", "-1", "--format=%(trailers:key=Misoku-Device,valueonly)", theirs, "--not", ours, "--", ":(literal)"+p)
	if err != nil {
		return "", err
	}
	first, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	first = strings.Trim(strings.TrimSpace(first), ".")
	return strings.Map(func(c rune) rune {
		if c < 0x20 || strings.ContainsRune(`<>:"/\|?*`, c) {
			return '-'
		}
		return c
	}, first), nil
}

// conflictName returns the name of the conflict copy of p (slash
// separated): "<name> (conflict <device> YYYY-MM-DD HHMM)<ext>" in p's
// folder, local time with no ":" so it is valid on Windows, and " 2",
// " 3"... appended when the name is already taken in the union's tree or
// on disk. Names are compared ignoring case, as on macOS and Windows.
// FindConflictCopies finds these names on disk (conflictCopyRe): keep the
// two in step.
func conflictName(dir, p, device string, now time.Time, taken map[string]bool) (string, error) {
	folder, base := path.Split(p)
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if stem == "" { // ".name": all extension, no stem
		stem, ext = base, ""
	}
	label := "conflict "
	if device != "" {
		label += device + " "
	}
	label += now.Format("2006-01-02 1504")
	for n := 1; n <= 1000; n++ {
		name := folder + stem + " (" + label + ")"
		if n > 1 {
			name += " " + strconv.Itoa(n)
		}
		name += ext
		if taken[strings.ToLower(name)] {
			continue
		}
		if _, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(name))); !os.IsNotExist(err) {
			continue // on disk, or unknown: never write over it
		}
		taken[strings.ToLower(name)] = true
		return name, nil
	}
	return "", fmt.Errorf("no free name for a conflict copy of %s", p)
}
