// Package dirplan turns the edited lines of the explorer's folder editor
// (Stage A, mini.files style) into the file operations they ask for,
// validated and ordered. It is pure: it reads a page list, never the disk.
package dirplan

import (
	"fmt"
	"path"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
)

const mdExt = ".md"

// Row is one line of an edited folder.
type Row struct {
	// OrigID is the ID of the note or folder the line stands for, "" for a
	// typed line. A line cut from one folder and pasted in another keeps it.
	OrigID string
	// Name is the line's text: a name, or for a typed line a path read the
	// way the explorer's n reads it ("test", "test/", "ideas/test").
	Name string
}

// Folder is one folder as edited: ID is "" for the root, a folder ID, or a
// note ID for that note's sub-notes; Rows are all its lines.
type Folder struct {
	ID   string
	Rows []Row
}

// OpKind is what an Op does.
type OpKind int

// The kinds of Op, in the order a Plan applies them.
const (
	OpMkdir  OpKind = iota // create the folder To
	OpRename               // rename From to To in the same folder
	OpMove                 // move From to To in another folder, maybe renamed
	OpCreate               // create the empty note To
	OpDelete               // delete From
)

// String names the kind for the confirm dialog: "mkdir", "rename", ….
func (k OpKind) String() string {
	return [...]string{"mkdir", "rename", "move", "create", "delete"}[k]
}

// Op is one file operation. From and To are IDs as they stand when the op
// runs, after the ops before it in the plan: a folder without "/", a note
// with ".md". A note moves, renames and is deleted with its sub-notes.
type Op struct {
	Kind     OpKind
	From, To string
	Folder   bool // the entry is a plain folder
}

// Plan is the ordered list of ops a set of edited folders asks for.
type Plan struct{ Ops []Op }

// Build compares the edited folders with before (the workspace as
// Store.List returns it) and plans: new folders first, then moves and
// renames, then new notes, deletes last. A line missing from every folder
// is a delete; one found in another folder is a move. Names are cleaned
// by core.SplitEntryPath, the parser of n. A name taken in its folder, a
// line pasted twice, a folder moved into itself, an unsafe path or an
// entry that no longer exists refuses the whole plan, with the reason.
func Build(before []core.Page, folders []Folder) (Plan, error) {
	b := builder{
		byID:  map[string]core.Page{},
		taken: map[string]string{},
		named: map[string]bool{},
		dirs:  map[string]bool{},
		seen:  map[string]bool{},
	}
	for _, p := range before {
		b.byID[p.ID] = p
		b.taken[key(p.ID)] = p.ID
	}
	for _, f := range folders {
		for _, r := range f.Rows {
			if err := b.row(f.ID, r); err != nil {
				return Plan{}, err
			}
		}
	}

	var deletes []core.Page
	for _, f := range folders {
		for _, p := range before {
			if p.ParentID == f.ID && !b.seen[p.ID] {
				deletes = append(deletes, p)
			}
		}
	}

	ops := b.mkdirs
	var done []core.Move
	for _, op := range b.moves {
		op.From, op.To = rebase(done, op.From), rebase(done, op.To)
		done = append(done, core.Move{From: op.From, To: op.To})
		ops = append(ops, op)
	}
	for _, op := range b.creates {
		op.To = rebase(done, op.To)
		ops = append(ops, op)
	}
	for _, p := range deletes {
		if !insideAny(p.ID, deletes) {
			ops = append(ops, Op{Kind: OpDelete, From: rebase(done, p.ID), Folder: p.IsFolder})
		}
	}
	return Plan{Ops: ops}, nil
}

type builder struct {
	byID  map[string]core.Page
	taken map[string]string // key of every name in before → its ID
	named map[string]bool   // keys of the notes and moves the plan names
	dirs  map[string]bool   // keys of the folders the plan creates
	seen  map[string]bool   // IDs on some line

	mkdirs, moves, creates []Op
}

// row plans one line of the folder folderID.
func (b *builder) row(folderID string, r Row) error {
	segs, isDir, err := core.SplitEntryPath(r.Name)
	if err != nil {
		return fmt.Errorf("%q: %w", r.Name, err)
	}
	dir := strings.TrimSuffix(folderID, mdExt)
	if r.OrigID == "" {
		return b.typed(dir, r.Name, segs, isDir)
	}

	p, ok := b.byID[r.OrigID]
	switch {
	case !ok:
		return fmt.Errorf("%q no longer exists", r.OrigID)
	case b.seen[p.ID]:
		return fmt.Errorf("%q is pasted more than once", p.Title)
	case len(segs) > 1:
		return fmt.Errorf("%q: a rename takes a name, not a path", r.Name)
	case isDir && !p.IsFolder:
		return fmt.Errorf("%q: a note can't become a folder", r.Name)
	}
	b.seen[p.ID] = true
	to := path.Join(dir, segs[0])
	if !p.IsFolder {
		to += mdExt
	}
	if to == p.ID {
		return nil
	}
	if src := strings.TrimSuffix(p.ID, mdExt); dir == src || strings.HasPrefix(dir, src+"/") {
		return fmt.Errorf("%q can't move into itself", p.Title)
	}
	if err := b.claim(to, r.Name, p.ID); err != nil {
		return err
	}
	kind := OpMove
	if p.ParentID == folderID {
		kind = OpRename
	}
	b.moves = append(b.moves, Op{Kind: kind, From: p.ID, To: to, Folder: p.IsFolder})
	return nil
}

// typed plans a new line: the folders on its path that don't exist yet,
// then the note, if it names one.
func (b *builder) typed(dir, name string, segs []string, isDir bool) error {
	for i, seg := range segs {
		id := path.Join(dir, seg)
		if i == len(segs)-1 && !isDir {
			if err := b.claim(id+mdExt, name, ""); err != nil {
				return err
			}
			b.creates = append(b.creates, Op{Kind: OpCreate, To: id + mdExt})
			return nil
		}
		dir = id
		k := key(id)
		switch {
		case b.taken[k] != "" && i == len(segs)-1:
			return fmt.Errorf("%q already exists", name)
		case b.taken[k] != "" || b.dirs[k]:
			continue // an existing folder, a note's sub-notes, or already planned
		case b.named[k]:
			return fmt.Errorf("%q is named twice", name)
		}
		b.dirs[k] = true
		b.mkdirs = append(b.mkdirs, Op{Kind: OpMkdir, To: id, Folder: true})
	}
	return nil
}

// claim reserves the name of id for the entry self ("" for a new note):
// no other entry may hold it now, and no other line may give it.
func (b *builder) claim(id, name, self string) error {
	k := key(id)
	if t := b.taken[k]; t != "" && t != self {
		return fmt.Errorf("%q already exists", name)
	}
	if b.named[k] || b.dirs[k] {
		return fmt.Errorf("%q is named twice", name)
	}
	b.named[k] = true
	return nil
}

// key is the name an ID occupies in its folder: a note and a folder of the
// same name are one line, and names differing only in case collide on
// macOS and Windows.
func key(id string) string { return strings.ToLower(strings.TrimSuffix(id, mdExt)) }

// rebase maps id through the moves already done, one at a time.
func rebase(done []core.Move, id string) string {
	for _, m := range done {
		if next := core.ExpandMoves([]core.Page{{ID: id}}, []core.Move{m}); len(next) > 0 {
			id = next[0].To
		}
	}
	return id
}

// insideAny reports that id is under another entry of deletes, which takes
// it along.
func insideAny(id string, deletes []core.Page) bool {
	for _, d := range deletes {
		if strings.HasPrefix(id, strings.TrimSuffix(d.ID, mdExt)+"/") {
			return true
		}
	}
	return false
}
