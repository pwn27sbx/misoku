package app

import (
	"fmt"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// Unlinked mentions: the notes that name the open note in plain text
// without linking to it, listed under its backlinks, each with a button
// that turns its mentions into [[links]].

// mentionState is the scan of the open note's unlinked mentions. Scans
// read every note, so they run in a command; they are numbered like the
// link index builds (linkIndexState), and only the latest one asked for is
// applied: a slower, older scan that arrives late is dropped.
type mentionState struct {
	gen    uint64 // number of the latest scan started
	wanted bool   // a scan is due; Update starts it, see startMentionScan
	target string // the note the latest scan was asked for
	shown  string // the note rows are for
	rows   []core.UnlinkedMention
}

// mentionsScannedMsg is a finished scan of the notes mentioning target.
type mentionsScannedMsg struct {
	gen    uint64
	target string
	rows   []core.UnlinkedMention
}

// mentionsLinkedMsg carries what linking one note's mentions wrote.
type mentionsLinkedMsg struct {
	row     core.UnlinkedMention
	done    []core.Relink
	skipped []string
	err     error
}

// requestMentions asks for the open note's mentions to be scanned once
// the current message is handled: when another note is open than the one
// last scanned, or always with force (the notes changed on disk).
func (m *Model) requestMentions(force bool) {
	id := m.currentNoteID()
	if id == "" {
		m.mentions.target = ""
		return
	}
	if force || id != m.mentions.target {
		m.mentions.target = id
		m.mentions.wanted = true
	}
}

// startMentionScan starts the scan requestMentions asked for, if any. The
// command reads the notes through the store, which is safe to call from
// another goroutine; templates are no notes of their own and are left out.
func (m *Model) startMentionScan() tea.Cmd {
	if !m.mentions.wanted || m.store == nil {
		return nil
	}
	m.mentions.wanted = false
	b := m.activeBuffer()
	if b == nil || b.draft() || b.id != m.mentions.target {
		return nil
	}
	m.mentions.gen++
	gen, store, notes, dir := m.mentions.gen, m.store, m.notes, m.templatesFolder()
	target := core.Page{ID: b.id, Title: b.title}
	return func() tea.Msg {
		rows := core.FindUnlinkedMentions(notes, target, func(id string) (string, error) {
			p, err := store.Get(id)
			return p.Content, err
		}, func(id string) bool { return inTemplates(id, dir) })
		return mentionsScannedMsg{gen: gen, target: target.ID, rows: rows}
	}
}

// applyMentions shows a finished scan, unless a newer one was started.
func (m *Model) applyMentions(msg mentionsScannedMsg) {
	if msg.gen != m.mentions.gen {
		return
	}
	m.mentions.shown, m.mentions.rows = msg.target, msg.rows
	m.preview.SetMentions(msg.target, msg.rows)
}

// mentionRow is the shown row of note id, if any.
func (m *Model) mentionRow(id string) (core.UnlinkedMention, bool) {
	if m.mentions.shown != m.currentNoteID() {
		return core.UnlinkedMention{}, false
	}
	for _, r := range m.mentions.rows {
		if r.ID == id {
			return r, true
		}
	}
	return core.UnlinkedMention{}, false
}

// refuseDirtyMention reports, and refuses, linking the mentions of a note
// whose open tab has unsaved edits: the rewrite would be written under
// them.
func (m *Model) refuseDirtyMention(id string) bool {
	if i := m.bufferIndex(id); i >= 0 && m.isDirty(m.buffers[i]) {
		m.setError(id + " has unsaved changes: save it before linking its mentions")
		return true
	}
	return false
}

// askLinkMentions is the link button of an unlinked mention: it asks, then
// rewrites every mention in note id as a link to the open note, against
// the text the scan read (ApplyRelinks skips the note if it changed since).
func (m *Model) askLinkMentions(id string) tea.Cmd {
	row, ok := m.mentionRow(id)
	if !ok || m.refuseDirtyMention(id) {
		return nil
	}
	b := m.activeBuffer()
	to, n := core.LinkMentions(row.Content, b.title, m.linkName(core.Page{ID: b.id, Title: b.title}))
	if n == 0 {
		return nil
	}
	rs := []core.Relink{{ID: row.ID, From: row.Content, To: to, Links: n}}
	m.confirm = &confirmDialog{
		title:   fmt.Sprintf("Link %s in %s?", mentionCount(n), row.Title),
		message: []string{"Each one becomes [[" + b.title + "]]; No leaves the note as written."},
		buttons: []string{"Yes", "No"},
		onChoose: func(m *Model, choice int) tea.Cmd {
			if choice != 0 || m.refuseDirtyMention(id) {
				return nil
			}
			store := m.store
			return func() tea.Msg {
				done, skipped, err := store.ApplyRelinks(rs)
				return mentionsLinkedMsg{row: row, done: done, skipped: skipped, err: err}
			}
		},
	}
	return nil
}

// mentionCount says "1 mention" or "n mentions".
func mentionCount(n int) string {
	return fmt.Sprintf("%d %s", n, plural(n, "mention", "mentions"))
}

// finishLinkMentions brings an open clean tab of the rewritten note up to
// date like a rename's rewrite (relinkTab), counts the write as a save for
// sync and reads the links again, so the note moves to the backlinks.
func (m *Model) finishLinkMentions(msg mentionsLinkedMsg) {
	for _, r := range msg.done {
		m.relinkTab(relinkJob{}, r, r.ID)
	}
	if len(msg.done) > 0 {
		m.noteSaved()
		m.refreshGraph()
		m.refreshModified()
	}
	switch {
	case msg.err != nil:
		m.setError(msg.err.Error())
	case len(msg.skipped) > 0:
		m.setError(strings.Join(msg.skipped, ", ") + " changed since it was read; its mentions were left as written")
		m.requestMentions(true)
	case len(msg.done) > 0:
		m.setStatus(fmt.Sprintf("Linked %s in %s", mentionCount(msg.done[0].Links), msg.row.Title))
	}
}
