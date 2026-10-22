package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/gitsync"

	tea "github.com/charmbracelet/bubbletea"
)

// draftIDPrefix marks a buffer that has never been saved, so it has no file
// on disk (and no explorer entry) yet — VSCode's "Untitled-1".
const draftIDPrefix = "draft:"

func isDraftID(id string) bool { return strings.HasPrefix(id, draftIDPrefix) }

// buffer is one open tab. Unsaved edits live here (not on disk) until the
// user saves, so switching tabs never loses work.
type buffer struct {
	id    string // file ID, or draftIDPrefix+N
	title string // file name without ".md", or "Untitled-N"
	saved string // content as last written to disk ("" for drafts)
	text  string // latest text; for the active tab, the editor is the truth
	dir   string // drafts: default folder offered by Save As
	gone  bool   // the file vanished from disk (deleted or renamed outside), see syncOpenBuffers
	// saves counts the changes to saved. Whatever was read from disk before
	// it last moved is out of date: a workspace scan sent earlier is
	// dropped (see applyScannedNote).
	saves uint64
	// textVer is the editor's text version b.text was copied at, 0 when
	// b.text did not come from the editor.
	textVer uint64
	stats   textStats
}

// textStats is what the bars show about a tab's text, kept until the text,
// the tab or what is saved changes, so a frame or a key that only moves the
// cursor neither copies nor scans the note.
type textStats struct {
	key   statsKey
	valid bool
	dirty bool
	words int
}

// statsKey identifies the text textStats were worked out for. version is
// the editor's text version when the text is the editor's (editor true), or
// buffer.textVer otherwise. Saving changes saves but not the text version,
// and without saves the unsaved mark would stay after a save.
type statsKey struct {
	version uint64
	editor  bool
	id      string
	saves   uint64
}

// setSaved records content as what is on disk and in the tab.
func (b *buffer) setSaved(content string) {
	b.saved, b.text = content, content
	b.saves++
}

func (b *buffer) draft() bool { return isDraftID(b.id) }

// graphID is the graph's tab. It counts as a draft (no file behind it)
// that is never dirty and never saved.
const graphID = draftIDPrefix + "graph"

func (b *buffer) isGraph() bool { return b.id == graphID }

func (b *buffer) fileName() string {
	if b.draft() {
		return b.title
	}
	return path.Base(b.id)
}

func (b *buffer) page() core.Page {
	return core.Page{ID: b.id, Title: b.title, Content: b.text}
}

// noteDir is the absolute folder a buffer's note lives (or will live) in.
func (m *Model) noteDir(b *buffer) string {
	rel := b.dir
	if !b.draft() {
		rel = path.Dir(b.id)
	}
	return filepath.Join(m.store.Root(), filepath.FromSlash(rel))
}

func (m *Model) bufferIndex(id string) int {
	for i, b := range m.buffers {
		if b.id == id {
			return i
		}
	}
	return -1
}

func (m *Model) activeBuffer() *buffer {
	if i := m.bufferIndex(m.active); i >= 0 {
		return m.buffers[i]
	}
	return nil
}

// liveText returns a buffer's current text, reading the editor for the active tab.
func (m *Model) liveText(b *buffer) string {
	if b.id == m.active && m.content.HasPage() {
		return m.content.Value()
	}
	return b.text
}

func (m *Model) isDirty(b *buffer) bool { return m.textStats(b).dirty }

// wordCount is the number of words in b's current text.
func (m *Model) wordCount(b *buffer) int { return m.textStats(b).words }

// textStats returns b's unsaved state and word count, reading the text only
// when it changed since they were last worked out.
func (m *Model) textStats(b *buffer) textStats {
	key := statsKey{version: b.textVer, id: b.id, saves: b.saves}
	if b.id == m.active && m.content.HasPage() {
		key.version, key.editor = m.content.Version(), true
	}
	if b.stats.valid && b.stats.key == key {
		return b.stats
	}
	text := m.liveText(b)
	dirty := text != b.saved
	if b.draft() {
		dirty = text != ""
	}
	b.stats = textStats{key: key, valid: true, dirty: dirty, words: countWords(text)}
	return b.stats
}

// countWords is len(strings.Fields(s)) without building the slice.
func countWords(s string) int {
	n, in := 0, false
	for _, r := range s {
		space := unicode.IsSpace(r)
		if !space && !in {
			n++
		}
		in = !space
	}
	return n
}

func (m *Model) dirtyBuffers() []*buffer {
	var out []*buffer
	for _, b := range m.buffers {
		if m.isDirty(b) {
			out = append(out, b)
		}
	}
	return out
}

// stashActive copies the editor's text into the active buffer before the
// editor is pointed at another one.
func (m *Model) stashActive() {
	if b := m.activeBuffer(); b != nil && m.content.HasPage() {
		b.text, b.textVer = m.content.Value(), m.content.Version()
	}
}

// refreshModified pushes the set of dirty saved files to the explorer.
func (m *Model) refreshModified() {
	mod := map[string]bool{}
	for _, b := range m.buffers {
		if !b.draft() && m.isDirty(b) {
			mod[b.id] = true
		}
	}
	m.sidebar.SetModified(mod)
}

// showBuffer points every pane at b without changing focus.
func (m *Model) showBuffer(b *buffer) {
	m.stashActive()
	m.active = b.id
	m.content.SetBuffer(b.page(), b.draft())
	m.preview.SetBaseDir(m.noteDir(b))
	m.preview.SetPage(b.page())
	m.refreshLinks()
	if b.isGraph() {
		m.content.SetBuffer(core.Page{}, false)
		m.previewNote(m.graph.Selected())
	}
	m.sidebar.SetActiveID(b.id)
	if !b.draft() {
		m.sidebar.SetSelectedID(b.id)
	}
	m.viewMode = viewModeWorkspace
}

// clearEditor shows the empty state when no tabs are left.
func (m *Model) clearEditor() {
	m.active = ""
	m.content.SetBuffer(core.Page{}, false)
	m.preview.SetPage(core.Page{})
	m.sidebar.SetActiveID("")
}

// openFile opens id in a tab (reusing an existing tab) and focuses the editor.
func (m *Model) openFile(id string) tea.Cmd {
	if i := m.bufferIndex(id); i >= 0 {
		m.showBuffer(m.buffers[i])
		return m.focusPane(focusEditor)
	}
	p, err := m.store.Get(id)
	if err != nil || p.IsFolder {
		m.setError("Cannot open " + id)
		return nil
	}
	b := &buffer{id: p.ID, title: p.Title, saved: p.Content, text: p.Content}
	m.insertBuffer(b)
	m.showBuffer(b)
	return m.focusPane(focusEditor)
}

// insertBuffer adds a tab just after the active one, like VSCode.
func (m *Model) insertBuffer(b *buffer) {
	i := m.bufferIndex(m.active)
	if i < 0 {
		m.buffers = append(m.buffers, b)
		return
	}
	m.buffers = append(m.buffers[:i+1], append([]*buffer{b}, m.buffers[i+1:]...)...)
}

// newDraft opens an empty "Untitled-N" tab. Nothing is written to disk
// until the user saves and picks a location.
func (m *Model) newDraft(parentID string) tea.Cmd {
	m.draftSeq++
	b := &buffer{
		id:    fmt.Sprintf("%s%d", draftIDPrefix, m.draftSeq),
		title: fmt.Sprintf("Untitled-%d", m.draftSeq),
		dir:   m.store.ChildDir(parentID),
	}
	m.insertBuffer(b)
	m.showBuffer(b)
	cmd := m.focusPane(focusEditor)
	return tea.Batch(cmd, m.content.EnterInsert())
}

// cycleBuffer activates the tab delta positions away (wrapping).
func (m *Model) cycleBuffer(delta int) tea.Cmd {
	if len(m.buffers) < 2 {
		return nil
	}
	i := m.bufferIndex(m.active)
	if i < 0 {
		i = 0
	}
	next := (i + delta + len(m.buffers)) % len(m.buffers)
	m.showBuffer(m.buffers[next])
	return nil
}

// removeBuffer drops a tab without any prompt, activating its neighbour.
func (m *Model) removeBuffer(id string) {
	i := m.bufferIndex(id)
	if i < 0 {
		return
	}
	wasActive := id == m.active
	m.buffers = append(m.buffers[:i], m.buffers[i+1:]...)
	if !wasActive {
		return
	}
	m.active = ""
	if len(m.buffers) == 0 {
		m.clearEditor()
		return
	}
	m.showBuffer(m.buffers[min(i, len(m.buffers)-1)])
}

// closeBuffer closes a tab, asking first when it has unsaved changes.
func (m *Model) closeBuffer(id string, force bool) tea.Cmd {
	i := m.bufferIndex(id)
	if i < 0 {
		return nil
	}
	b := m.buffers[i]
	if force || !m.isDirty(b) {
		m.removeBuffer(id)
		m.refreshModified()
		return nil
	}
	m.confirm = &confirmDialog{
		title:   "Unsaved changes",
		message: []string{fmt.Sprintf("Save changes to %s before closing?", b.fileName()), "Your changes will be lost if you don't save them."},
		buttons: []string{"Save", "Don't Save", "Cancel"},
		onChoose: func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				return m.saveBuffer(b, func(m *Model) tea.Cmd {
					m.removeBuffer(b.id)
					m.refreshModified()
					return nil
				})
			case 1:
				m.removeBuffer(b.id)
				m.refreshModified()
			}
			return nil
		},
	}
	return nil
}

// saveBuffer writes b to disk. A draft has no location yet, so Save As
// opens first; a file that changed on disk since it was opened or last saved
// asks before being overwritten. after runs once the file is safely written.
func (m *Model) saveBuffer(b *buffer, after func(*Model) tea.Cmd) tea.Cmd {
	if b.isGraph() {
		return nil
	}
	if b.draft() {
		m.openSaveAs(b, after)
		return nil
	}
	if m.noteGone(b) {
		// Writing blindly would fail, or after a rename recreate the old name
		// next to the new one and duplicate the note: let the user choose.
		m.confirm = &confirmDialog{
			title:   "Not on disk",
			message: []string{fmt.Sprintf("%s was deleted or renamed outside Misoku.", b.fileName()), "Recreate it at its old path with your version?"},
			buttons: []string{"Recreate", "Cancel"},
			onChoose: func(m *Model, choice int) tea.Cmd {
				if choice == 0 {
					return m.recreateBuffer(b, after)
				}
				return nil
			},
		}
		return nil
	}
	if disk, err := m.store.Get(b.id); err == nil && disk.Content != b.saved && disk.Content != m.liveText(b) {
		if !m.isDirty(b) {
			// Nothing of the user's to write: show the disk's version, as
			// the workspace poll would.
			m.syncOpenBuffers()
			if after != nil {
				return after(m)
			}
			return nil
		}
		return m.keepBothAndWrite(b, disk.Content, after)
	}
	return m.writeBuffer(b, after)
}

// keepBothAndWrite saves b over a file that changed on disk since it was
// opened (another device's edit a sync brought, another app): neither
// version is lost and nothing is asked. The version on disk goes first to a
// conflict copy next to the note, named as a sync names them (S6) but
// without the device, which the app does not know; then the note gets the
// user's version. When the copy cannot be written, nothing is.
func (m *Model) keepBothAndWrite(b *buffer, theirs string, after func(*Model) tea.Cmd) tea.Cmd {
	name, err := gitsync.ConflictCopyName(m.store.Root(), b.id, "", m.clock())
	if err == nil {
		_, err = m.store.SaveAs(path.Dir(name), path.Base(name), theirs)
	}
	if err != nil {
		m.setError("Not saved: the version on disk could not be kept (" + err.Error() + ")")
		m.quitting = false
		return nil
	}
	m.reloadTree()
	// Short, so the status bar shows the copy's name whole.
	return m.writeBufferNote(b, after, "Both kept: "+path.Base(name))
}

// writeBuffer writes b's text to its file, with no conflict check.
func (m *Model) writeBuffer(b *buffer, after func(*Model) tea.Cmd) tea.Cmd {
	return m.writeBufferNote(b, after, "")
}

// writeBufferNote is writeBuffer; a note, when not empty, replaces the
// message that says the file was written and is shown as needing
// attention.
func (m *Model) writeBufferNote(b *buffer, after func(*Model) tea.Cmd, note string) tea.Cmd {
	text := m.liveText(b)
	saved, err := m.store.Update(core.Page{ID: b.id, Title: b.title, Content: text})
	if err != nil {
		m.setError("Save failed: " + err.Error())
		m.quitting = false
		return nil
	}
	b.setSaved(saved.Content)
	m.noteSaved()
	m.refreshGraph()
	if note != "" {
		m.setError(note)
	} else {
		m.setStatus(fmt.Sprintf("\"%s\" %dL written", b.id, strings.Count(text, "\n")+1))
	}
	m.refreshModified()
	if after != nil {
		return after(m)
	}
	return nil
}

// noteGone reports whether b's file is no longer on disk. Only a missing
// path counts: an unreadable file is not gone, and saving it reports why.
func (m *Model) noteGone(b *buffer) bool {
	if b.draft() {
		return false
	}
	return noteGoneAt(m.store.Root(), b.id)
}

// noteGoneAt is noteGone for the note id under root. It reads only the disk,
// so a workspace scan can run it off the update loop.
func noteGoneAt(root, id string) bool {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(id)))
	return errors.Is(err, fs.ErrNotExist)
}

// warnGone tells the user b's note left the disk. Misoku cannot tell a
// delete from a rename; after a rename the explorer shows the new name.
func (m *Model) warnGone(b *buffer) {
	m.setError(fmt.Sprintf("%q was deleted or renamed outside Misoku; saving will ask before recreating it", b.id))
}

// recreateBuffer writes b's text back at its old path after its file
// vanished. SaveAs refuses to overwrite, so a file that reappeared meanwhile
// is never clobbered.
func (m *Model) recreateBuffer(b *buffer, after func(*Model) tea.Cmd) tea.Cmd {
	text := m.liveText(b)
	if _, err := m.store.SaveAs(path.Dir(b.id), path.Base(b.id), text); err != nil {
		m.setError("Save failed: " + err.Error())
		m.quitting = false
		return nil
	}
	b.setSaved(text)
	b.gone = false
	m.noteSaved()
	m.reloadTree()
	m.setStatus(fmt.Sprintf("\"%s\" %dL written", b.id, strings.Count(text, "\n")+1))
	m.refreshModified()
	if after != nil {
		return after(m)
	}
	return nil
}

// saveBufferAs writes b's text to dir/name and turns the tab into that file.
func (m *Model) saveBufferAs(b *buffer, dir, name string) error {
	text := m.liveText(b)
	p, err := m.store.SaveAs(dir, name, text)
	if err != nil {
		return err
	}
	m.noteSaved()
	wasActive := b.id == m.active
	b.id, b.title, b.dir = p.ID, p.Title, ""
	b.setSaved(text)
	if wasActive {
		m.active = b.id
	}
	m.reloadTree()
	if wasActive {
		m.content.SetBuffer(b.page(), false)
		m.preview.SetBaseDir(m.noteDir(b))
		m.preview.SetPage(b.page())
		m.sidebar.SetActiveID(b.id)
		m.sidebar.SetSelectedID(b.id)
	}
	m.setStatus(fmt.Sprintf("\"%s\" written", p.ID))
	m.refreshModified()
	return nil
}

// requestQuit exits, first offering to save any unsaved tabs.
func (m *Model) requestQuit(force bool) tea.Cmd {
	if m.quitHeld() {
		return nil
	}
	dirty := m.dirtyBuffers()
	if force || len(dirty) == 0 {
		return tea.Quit
	}
	names := make([]string, 0, len(dirty))
	for _, b := range dirty {
		names = append(names, b.fileName())
	}
	list := strings.Join(names, ", ")
	if len(names) > 3 {
		list = strings.Join(names[:3], ", ") + fmt.Sprintf(" and %d more", len(names)-3)
	}
	m.confirm = &confirmDialog{
		title:   "Quit Misoku?",
		message: []string{fmt.Sprintf("%d unsaved: %s", len(dirty), list)},
		buttons: []string{"Save All", "Discard", "Cancel"},
		onChoose: func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				m.quitting = true
				m.savedOnQuit = map[*buffer]bool{}
				return m.continueSaveAll()
			case 1:
				return tea.Quit
			}
			return nil
		},
	}
	return nil
}

// continueSaveAll saves every dirty tab, stopping at each draft to ask
// where it should go, then quits.
func (m *Model) continueSaveAll() tea.Cmd {
	for _, b := range m.dirtyBuffers() {
		if m.savedOnQuit[b] {
			continue // already written once; never loop on a tab that stays dirty
		}
		m.savedOnQuit[b] = true
		if b.draft() {
			m.showBuffer(b)
		}
		// One note at a time: a save may open a dialog (Save As for a draft, the
		// changed-on-disk question), and the rest must wait for its answer.
		// Cancelling it clears m.quitting and Save All stops there.
		return m.saveBuffer(b, func(m *Model) tea.Cmd { return m.continueSaveAll() })
	}
	return tea.Quit
}

// remapBuffers renames tabs after a file or folder moved on disk.
func (m *Model) remapBuffers(oldID, newID string) {
	oldPrefix := strings.TrimSuffix(oldID, ".md") + "/"
	newPrefix := strings.TrimSuffix(newID, ".md") + "/"
	remap := func(id string) string {
		switch {
		case id == oldID:
			return newID
		case strings.HasPrefix(id, oldPrefix):
			return newPrefix + strings.TrimPrefix(id, oldPrefix)
		}
		return id
	}
	m.active = remap(m.active)
	for _, b := range m.buffers {
		if nid := remap(b.id); nid != b.id {
			b.id = nid
			b.title = strings.TrimSuffix(path.Base(nid), ".md")
		}
	}
}

// dropBuffersUnder closes (without prompting) tabs for a deleted file or folder.
func (m *Model) dropBuffersUnder(id string) {
	prefix := strings.TrimSuffix(id, ".md") + "/"
	var gone []string
	for _, b := range m.buffers {
		if b.id == id || strings.HasPrefix(b.id, prefix) {
			gone = append(gone, b.id)
		}
	}
	for _, g := range gone {
		m.removeBuffer(g)
	}
}
