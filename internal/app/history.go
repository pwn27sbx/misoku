package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"

	tea "github.com/charmbracelet/bubbletea"
)

// historyTimeout limits each git command of ":history": reading the list
// of commits, and reading one version.
const historyTimeout = 30 * time.Second

// historyCacheSize is how many versions the dialog keeps once read, so
// moving back and forth over the list reads each from git once.
const historyCacheSize = 16

// historyDialog is ":history": the commits that changed the open note, with
// a preview of the selected version, which Enter restores into the buffer.
type historyDialog struct {
	id       string // the note, as the tab names it
	versions []gitsync.Version
	sel      int
	offset   int
	// gen bumps on every selection change: only the read started for the
	// current selection is kept, so a slow read for a version scrolled past
	// never lands late; going back to it reads it again.
	gen   uint64
	cache map[string]historyText // by commit hash
}

// historyText is one version read from git, or why it could not be read.
type historyText struct {
	text string
	err  string
}

// historyListedMsg carries the note's commits, read outside Update.
type historyListedMsg struct {
	id       string
	versions []gitsync.Version
	err      error
}

// historyVersionMsg carries one version's text, read outside Update.
type historyVersionMsg struct {
	id   string
	gen  uint64
	hash string
	text historyText
}

// requestHistory starts reading the open note's commits. A draft, a graph or
// no note at all have none, and say so at once.
func (m *Model) requestHistory() tea.Cmd {
	b := m.activeBuffer()
	switch {
	case b == nil || b.isGraph():
		m.setError("History needs an open note")
		return nil
	case b.draft():
		m.setError("This note was never saved, so it has no history yet")
		return nil
	}
	m.setStatus("Reading the history of " + b.id + "…")
	return listHistory(m.store.Root(), b.id)
}

// listHistory runs git log for note id in the vault at root.
func listHistory(root, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), historyTimeout)
		defer cancel()
		versions, err := gitsync.New(root, historyTimeout).FileHistory(ctx, id)
		return historyListedMsg{id: id, versions: versions, err: err}
	}
}

// readVersion runs git show for one version of note id.
func readVersion(root, id string, gen uint64, v gitsync.Version) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), historyTimeout)
		defer cancel()
		out, err := gitsync.New(root, historyTimeout).FileAt(ctx, v.Hash, v.Path)
		t := historyText{text: string(out)}
		if err != nil {
			t = historyText{err: "Could not read this version: " + err.Error()}
		}
		return historyVersionMsg{id: id, gen: gen, hash: v.Hash, text: t}
	}
}

// showHistory opens the dialog once the commits are read, or says why there
// are none. A list for a note that is no longer the open one is dropped.
func (m *Model) showHistory(msg historyListedMsg) tea.Cmd {
	if b := m.activeBuffer(); b == nil || b.id != msg.id {
		return nil
	}
	switch {
	case errors.Is(msg.err, gitsync.ErrNoGit):
		m.setError("History needs git, and git is not installed")
		return nil
	case errors.Is(msg.err, gitsync.ErrNotARepo):
		m.setError("No history: this vault is not a git repository")
		return nil
	case msg.err != nil:
		m.setError("Could not read the history: " + msg.err.Error())
		return nil
	case len(msg.versions) == 0:
		m.setError(msg.id + " is not in git yet: it was never committed")
		return nil
	}
	m.clearStatus()
	d := &historyDialog{id: msg.id, versions: msg.versions, cache: map[string]historyText{}}
	m.history = d
	return d.load(m)
}

// load reads the selected version unless it is cached.
func (d *historyDialog) load(m *Model) tea.Cmd {
	v := d.versions[d.sel]
	if _, ok := d.cache[v.Hash]; ok {
		return nil
	}
	return readVersion(m.store.Root(), d.id, d.gen, v)
}

// applyHistoryVersion stores a version read from git. A read for another
// note, a closed dialog or an older selection (gen) is dropped.
func (m *Model) applyHistoryVersion(msg historyVersionMsg) {
	d := m.history
	if d == nil || d.id != msg.id || d.gen != msg.gen {
		return
	}
	if len(d.cache) >= historyCacheSize {
		clear(d.cache)
	}
	d.cache[msg.hash] = msg.text
}

// selected returns the selected version's text, and whether it was read.
func (d *historyDialog) selected() (historyText, bool) {
	t, ok := d.cache[d.versions[d.sel].Hash]
	return t, ok
}

// move changes the selection by delta, keeping it in view, and reads the
// newly selected version.
func (d *historyDialog) move(m *Model, delta, rows int) tea.Cmd {
	return d.selectAt(m, d.sel+delta, rows)
}

// selectAt selects version i (clamped) and reads it.
func (d *historyDialog) selectAt(m *Model, i, rows int) tea.Cmd {
	i = max(0, min(i, len(d.versions)-1))
	if i == d.sel {
		return nil
	}
	d.sel = i
	d.gen++
	if d.sel < d.offset {
		d.offset = d.sel
	}
	if d.sel >= d.offset+rows {
		d.offset = d.sel - rows + 1
	}
	return d.load(m)
}

// restore puts the selected version in the open buffer as one change (u
// undoes it). Nothing is written: the tab stays modified until saved.
func (d *historyDialog) restore(m *Model) {
	t, ok := d.selected()
	switch {
	case !ok:
		m.setStatus("Still reading this version…")
		return
	case t.err != "":
		m.setError(t.err)
		return
	}
	b := m.activeBuffer()
	if b == nil || b.id != d.id {
		m.history = nil
		return
	}
	m.history = nil
	v := d.versions[d.sel]
	if !m.content.RestoreText(t.text) {
		m.setStatus("The note already holds the version of " + v.Date)
		return
	}
	m.preview.SetContent(m.content.Value())
	m.refreshModified()
	m.setStatus("Restored the version of " + v.Date + " (" + shortHash(v.Hash) + "): u undoes it, save to keep it")
}

// shortHash is the first seven characters of a commit hash, as git shows it.
func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

// previewLines is the selected version's text as the preview shows it.
func (d *historyDialog) previewLines() []string {
	t, ok := d.selected()
	switch {
	case !ok:
		return []string{"Reading…"}
	case t.err != "":
		return []string{t.err}
	}
	return strings.Split(strings.ReplaceAll(t.text, "\t", "    "), "\n")
}
