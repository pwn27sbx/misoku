// Package dirbuf is the editable folder model of the explorer's mini.files-style
// mode: one folder shown as a list of named lines, edited with a Vim subset.
// Each row keeps the ID of the entry it came from outside its text, so renames,
// moves (cut + paste) and copies (yank + paste) stay traceable without hidden
// markers. It does no disk I/O; Rows, Original and Edited feed the planner.
package dirbuf

import (
	"slices"
	"sort"

	"github.com/mattn/go-runewidth"

	"github.com/charmbracelet/bubbles/cursor"
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
	// NavOpen is Enter in normal mode: with pending edits the host asks to
	// apply them, otherwise it opens the line under the cursor (enter a
	// folder, open a note); see Current.
	NavOpen
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

// Rows returns a copy of the current folder's lines as shown, the one being
// typed included as it was when typing started.
func (b *Buffer) Rows() []Row { return slices.Clone(b.f().rows) }

// Edited returns a copy of the edited rows of a visited folder, as if
// typing ended now: the line being typed has the typed name, and an empty
// new one is left out. The host plans from it even while a name is typed.
func (b *Buffer) Edited(folderID string) []Row {
	f, ok := b.folders[folderID]
	if !ok {
		return nil
	}
	rows := slices.Clone(f.rows)
	if b.editing && folderID == b.cur {
		rows[f.cursor].Name = b.input.Value()
		if rows[f.cursor].Name == "" && rows[f.cursor].OrigID == "" {
			rows = slices.Delete(rows, f.cursor, f.cursor+1)
		}
	}
	return rows
}

// Line is the name on the cursor's line as it stands now: the one being
// typed, or the row's.
func (b *Buffer) Line() string {
	if b.editing {
		return b.input.Value()
	}
	return b.name()
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

// Select moves the cursor to the row that came from entry origID and
// reports whether the current folder has one.
func (b *Buffer) Select(origID string) bool {
	f := b.f()
	for i, r := range f.rows {
		if origID != "" && r.OrigID == origID {
			f.cursor = i
			return true
		}
	}
	return false
}

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

// SetRealCursor hides the input's drawn cursor cell when the terminal's own
// cursor stands in for it (see InputCursorX).
func (b *Buffer) SetRealCursor(on bool) {
	mode := cursor.CursorBlink
	if on {
		mode = cursor.CursorHide
	}
	b.input.Cursor.SetMode(mode)
}

// InputCursorX is the column, within InputView, where the next typed
// character goes.
func (b *Buffer) InputCursorX() int {
	return runewidth.StringWidth(string([]rune(b.input.Value())[:b.input.Position()]))
}

func (b *Buffer) f() *folder { return b.folders[b.cur] }

// Update handles one key. It returns a navigation request for h and l.
func (b *Buffer) Update(msg tea.Msg) (Nav, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return NavNone, nil
	}
	if b.editing {
		return b.updateInput(k)
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
	case "enter":
		return NavOpen, nil
	case "o", "O":
		return NavNone, b.openLine(key == "o")
	case "i", "I":
		if len(f.rows) == 0 {
			return NavNone, b.openLine(true)
		}
		return NavNone, b.edit(b.name(), true)
	case "a", "A":
		if len(f.rows) == 0 {
			return NavNone, b.openLine(true)
		}
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

// openLine inserts an empty line below the cursor (o) or above it (O) and
// starts typing its name.
func (b *Buffer) openLine(below bool) tea.Cmd {
	f := b.f()
	b.before = slices.Clone(f.rows)
	at := f.cursor
	if below && len(f.rows) > 0 {
		at++
	}
	f.rows = slices.Insert(f.rows, at, Row{})
	f.cursor = at
	return b.start("", true)
}

// updateInput handles a key while a name is typed. Esc commits the line and
// returns to normal. Enter, as in Oil, only opens an empty line below and
// goes on typing there: it never creates, opens or applies anything, and on
// an empty new line it keeps typing on it. Up and Down keep the typed name
// and go on typing at the end of the line above or below. Backspace at the
// start of the line joins it into the line above and Delete at its end
// joins the line below into it, as in Vim: a line joined away is gone from
// the folder, which plans its delete like dd.
func (b *Buffer) updateInput(k tea.KeyMsg) (Nav, tea.Cmd) {
	switch k.Type {
	case tea.KeyUp:
		return NavNone, b.moveInput(-1)
	case tea.KeyDown:
		return NavNone, b.moveInput(1)
	case tea.KeyEnter:
		return NavNone, b.splitLine()
	case tea.KeyBackspace:
		if b.input.Position() == 0 {
			return NavNone, b.joinUp()
		}
	case tea.KeyDelete:
		if b.input.Position() == len([]rune(b.input.Value())) {
			return NavNone, b.joinDown()
		}
	case tea.KeyEsc:
		b.commit()
		return NavNone, nil
	}
	var cmd tea.Cmd
	b.input, cmd = b.input.Update(k)
	return NavNone, cmd
}

// splitLine keeps the typed name in its line and goes on typing on a new
// empty line below, within the same undo step. An empty new line stays as
// it is.
func (b *Buffer) splitLine() tea.Cmd {
	f := b.f()
	f.rows[f.cursor].Name = b.input.Value()
	if f.rows[f.cursor].Name == "" && f.rows[f.cursor].OrigID == "" {
		return nil
	}
	f.cursor++
	f.rows = slices.Insert(f.rows, f.cursor, Row{})
	return b.start("", true)
}

// joinUp joins the line being typed onto the end of the line above and
// goes on typing there, at the join. On the first line nothing happens.
func (b *Buffer) joinUp() tea.Cmd {
	f := b.f()
	if f.cursor == 0 {
		return nil
	}
	rest := b.input.Value()
	f.rows = slices.Delete(f.rows, f.cursor, f.cursor+1)
	f.cursor--
	at := len([]rune(f.rows[f.cursor].Name))
	cmd := b.start(f.rows[f.cursor].Name+rest, false)
	b.input.SetCursor(at)
	return cmd
}

// joinDown joins the line below onto the end of the line being typed,
// keeping the typing position. On the last line nothing happens.
func (b *Buffer) joinDown() tea.Cmd {
	f := b.f()
	if f.cursor+1 >= len(f.rows) {
		return nil
	}
	value, at := b.input.Value(), b.input.Position()
	b.input.SetValue(value + f.rows[f.cursor+1].Name)
	b.input.SetCursor(at)
	f.rows = slices.Delete(f.rows, f.cursor+1, f.cursor+2)
	return nil
}

// Reload restarts the buffer on its folder with rows, as read from disk
// after its edits were applied: every visited folder's edits and history
// are forgotten and the cursor goes to row at. A name being typed goes on
// being typed, at the end of that row, or on a new empty line at at when
// the typed line was empty.
func (b *Buffer) Reload(rows []Row, at int) tea.Cmd {
	typing, blank := b.editing, b.editing && b.input.Value() == ""
	b.editing, b.before = false, nil
	b.input.Blur()
	b.folders = map[string]*folder{}
	b.Open(b.cur, rows)
	f := b.f()
	f.cursor = at
	b.clamp()
	switch {
	case blank:
		b.before = slices.Clone(f.rows)
		f.cursor = min(max(at, 0), len(f.rows))
		f.rows = slices.Insert(f.rows, f.cursor, Row{})
		return b.start("", true)
	case typing && len(f.rows) > 0:
		b.before = slices.Clone(f.rows)
		return b.start(f.rows[f.cursor].Name, false)
	}
	return nil
}

// moveInput keeps the typed name in its line (a pending edit, like Esc's) and
// keeps typing at the end of the line delta away. An empty new line is
// dropped. At the first or last line nothing moves. The whole insert stays
// one undo step.
func (b *Buffer) moveInput(delta int) tea.Cmd {
	f := b.f()
	to := f.cursor + delta
	if to < 0 || to >= len(f.rows) {
		return nil
	}
	f.rows[f.cursor].Name = b.input.Value()
	if f.rows[f.cursor].Name == "" && f.rows[f.cursor].OrigID == "" {
		f.rows = slices.Delete(f.rows, f.cursor, f.cursor+1)
		if delta > 0 {
			to--
		}
	}
	f.cursor = to
	return b.start(f.rows[to].Name, false)
}

// commit ends typing: the line takes the typed name, an empty new line is
// dropped (the cursor goes back to the line above it) and a change becomes
// one undo step.
func (b *Buffer) commit() {
	f := b.f()
	b.editing = false
	b.input.Blur()
	f.rows[f.cursor].Name = b.input.Value()
	if f.rows[f.cursor].Name == "" && f.rows[f.cursor].OrigID == "" {
		f.rows = slices.Delete(f.rows, f.cursor, f.cursor+1)
		f.cursor--
		b.clamp()
	}
	if !slices.Equal(b.before, f.rows) {
		f.undo, f.redo = append(f.undo, b.before), nil
	}
	b.before = nil
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
