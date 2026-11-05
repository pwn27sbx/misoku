// Package dirbuf is the editable folder model of the explorer's mini.files-style
// mode: one folder shown as a list of named lines, edited with a Vim subset.
// Each row keeps the ID of the entry it came from outside its text, so renames,
// moves (cut + paste) and copies (yank + paste) stay traceable without hidden
// markers. It does no disk I/O; Rows, Original and Edited feed the planner.
package dirbuf

import (
	"slices"
	"sort"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Row is one line of a folder. OrigID is the vault ID of the entry the line
// came from ("" for a new line), Name is the line text as typed (a trailing
// "/" marks a folder) and Yanked marks a line pasted from a yank (a copy).
type Row struct {
	OrigID string
	Name   string
	Yanked bool
}

// Nav is a navigation request the host must resolve, since dirbuf cannot read
// the tree: it answers with Open on the parent or the child folder.
type Nav int

const (
	// NavNone asks for nothing.
	NavNone Nav = iota
	// NavParent asks to open the parent folder (h).
	NavParent
	// NavChild asks to open the folder under the cursor (l); see Current.
	NavChild
)

type folder struct {
	orig, rows []Row
	cursor     int
	undo, redo [][]Row
}

// Buffer holds the pending edits of every folder visited and a register shared
// across them, so a line cut in one folder can be pasted in another.
type Buffer struct {
	cur     string
	folders map[string]*folder
	reg     []Row
	regYank bool
	pending string
	editing bool
	before  []Row
	input   textinput.Model
}

// New returns a buffer opened on folder with its original rows.
func New(folderID string, rows []Row) *Buffer {
	b := &Buffer{folders: map[string]*folder{}, input: textinput.New()}
	b.input.Prompt = ""
	b.Open(folderID, rows)
	return b
}

// Open switches to folderID. The first visit starts from rows; later visits
// keep that folder's pending edits and cursor and ignore rows.
func (b *Buffer) Open(folderID string, rows []Row) {
	b.cur, b.pending = folderID, ""
	if _, ok := b.folders[folderID]; !ok {
		b.folders[folderID] = &folder{orig: slices.Clone(rows), rows: slices.Clone(rows)}
	}
}

// Folder returns the ID of the folder being shown.
func (b *Buffer) Folder() string { return b.cur }

// Rows returns a copy of the current folder's edited rows.
func (b *Buffer) Rows() []Row { return b.Edited(b.cur) }

// Edited returns a copy of the edited rows of a visited folder.
func (b *Buffer) Edited(folderID string) []Row {
	if f, ok := b.folders[folderID]; ok {
		return slices.Clone(f.rows)
	}
	return nil
}

// Original returns a copy of the rows a visited folder was opened with.
func (b *Buffer) Original(folderID string) []Row {
	if f, ok := b.folders[folderID]; ok {
		return slices.Clone(f.orig)
	}
	return nil
}

// Folders returns the IDs of every visited folder, sorted.
func (b *Buffer) Folders() []string {
	out := make([]string, 0, len(b.folders))
	for id := range b.folders {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Dirty reports whether any visited folder differs from its original rows.
func (b *Buffer) Dirty() bool {
	for _, f := range b.folders {
		if !slices.Equal(f.orig, f.rows) {
			return true
		}
	}
	return false
}

// Cursor returns the index of the selected row in the current folder.
func (b *Buffer) Cursor() int { return b.f().cursor }

// Current returns the selected row, false when the folder is empty.
func (b *Buffer) Current() (Row, bool) {
	f := b.f()
	if f.cursor < len(f.rows) {
		return f.rows[f.cursor], true
	}
	return Row{}, false
}

// Editing reports whether a line's name is being typed.
func (b *Buffer) Editing() bool { return b.editing }

// InputView renders the name being typed, for the selected line.
func (b *Buffer) InputView() string { return b.input.View() }

func (b *Buffer) f() *folder { return b.folders[b.cur] }

// Update handles one key. It returns a navigation request for h and l.
func (b *Buffer) Update(msg tea.Msg) (Nav, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return NavNone, nil
	}
	if b.editing {
		return NavNone, b.updateInput(k)
	}
	f := b.f()
	key := k.String()
	if p := b.pending; p != "" {
		b.pending = ""
		switch p + key {
		case "dd":
			if r, ok := b.Current(); ok {
				b.push()
				b.reg, b.regYank = []Row{r}, false
				f.rows = slices.Delete(f.rows, f.cursor, f.cursor+1)
				b.clamp()
			}
		case "yy":
			if r, ok := b.Current(); ok {
				b.reg, b.regYank = []Row{r}, true
			}
		case "gg":
			f.cursor = 0
		case "cw", "cc":
			return NavNone, b.edit("", false)
		}
		return NavNone, nil
	}
	switch key {
	case "d", "y", "g", "c":
		b.pending = key
	case "j", "down":
		f.cursor++
		b.clamp()
	case "k", "up":
		f.cursor--
		b.clamp()
	case "G":
		f.cursor = len(f.rows) - 1
		b.clamp()
	case "h":
		return NavParent, nil
	case "l":
		return NavChild, nil
	case "o", "O":
		b.before = slices.Clone(f.rows)
		at := f.cursor
		if key == "o" && len(f.rows) > 0 {
			at++
		}
		f.rows = slices.Insert(f.rows, at, Row{})
		f.cursor = at
		return NavNone, b.start("", true)
	case "i", "I":
		return NavNone, b.edit(b.name(), true)
	case "a", "A":
		return NavNone, b.edit(b.name(), false)
	case "p", "P":
		b.paste(key == "P")
	case "u":
		b.step(&f.undo, &f.redo)
	case "ctrl+r":
		b.step(&f.redo, &f.undo)
	}
	return NavNone, nil
}

func (b *Buffer) name() string {
	r, _ := b.Current()
	return r.Name
}

func (b *Buffer) edit(value string, atStart bool) tea.Cmd {
	if _, ok := b.Current(); !ok {
		return nil
	}
	b.before = slices.Clone(b.f().rows)
	return b.start(value, atStart)
}

func (b *Buffer) start(value string, atStart bool) tea.Cmd {
	b.editing = true
	b.input.SetValue(value)
	if atStart {
		b.input.CursorStart()
	} else {
		b.input.CursorEnd()
	}
	return b.input.Focus()
}

func (b *Buffer) updateInput(k tea.KeyMsg) tea.Cmd {
	switch k.Type {
	case tea.KeyEsc, tea.KeyEnter:
		f := b.f()
		b.editing = false
		b.input.Blur()
		f.rows[f.cursor].Name = b.input.Value()
		if f.rows[f.cursor].Name == "" && f.rows[f.cursor].OrigID == "" {
			f.rows = slices.Delete(f.rows, f.cursor, f.cursor+1)
			b.clamp()
		}
		if !slices.Equal(b.before, f.rows) {
			f.undo, f.redo = append(f.undo, b.before), nil
		}
		b.before = nil
		return nil
	}
	var cmd tea.Cmd
	b.input, cmd = b.input.Update(k)
	return cmd
}

func (b *Buffer) paste(above bool) {
	if len(b.reg) == 0 {
		return
	}
	f := b.f()
	b.push()
	rows := slices.Clone(b.reg)
	for i := range rows {
		rows[i].Yanked = b.regYank
	}
	at := f.cursor
	if !above && len(f.rows) > 0 {
		at++
	}
	f.rows = slices.Insert(f.rows, at, rows...)
	f.cursor = at
}

// push records the current rows as one undo step and clears the redo stack.
func (b *Buffer) push() {
	f := b.f()
	f.undo, f.redo = append(f.undo, slices.Clone(f.rows)), nil
}

// step pops one snapshot from from, saving the current rows on to.
func (b *Buffer) step(from, to *[][]Row) {
	f := b.f()
	n := len(*from)
	if n == 0 {
		return
	}
	*to = append(*to, slices.Clone(f.rows))
	f.rows = (*from)[n-1]
	*from = (*from)[:n-1]
	b.clamp()
}

func (b *Buffer) clamp() {
	f := b.f()
	f.cursor = max(0, min(f.cursor, len(f.rows)-1))
}
