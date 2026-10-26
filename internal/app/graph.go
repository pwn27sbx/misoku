package app

import (
	"path"
	"path/filepath"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// The graph lives in its own tab. While it is showing, the preview on the
// right shows the note selected in it. The global graph (globalgraph.go)
// is a second such tab.

// graphActive reports whether a graph tab, local or global, is showing.
func (m *Model) graphActive() bool { return m.active == graphID || m.globalGraphActive() }

// graphSelected is the note selected in the graph tab showing.
func (m *Model) graphSelected() string {
	if m.globalGraphActive() {
		return m.globalGraph.Selected()
	}
	return m.graph.Selected()
}

// openGraph shows the graph tab (":graph", Space g), centred on the note
// that was open.
func (m *Model) openGraph() tea.Cmd {
	centre := m.currentNoteID()
	m.leaderPending = false
	m.leaveReading() // the graph is drawn in the editor column
	// The index is kept up to date by saves and workspace checks; reading
	// it again here would drop the unsaved links liveGraph adds now.
	m.graphNotes, m.graphEdges = m.linkGraph()
	m.liveGraph()
	m.graph.SetCentre(centre)
	if i := m.bufferIndex(graphID); i >= 0 {
		m.showBuffer(m.buffers[i])
	} else {
		b := &buffer{id: graphID, title: "Graph"}
		m.insertBuffer(b)
		m.showBuffer(b)
	}
	return m.focusPane(focusEditor)
}

// linkIndexState is the index of every note's links, read from disk off
// the update loop.
//
// Each build is a command that works on its own copy of the index in use
// (core.LinkIndex.Clone), so the update loop keeps reading idx while the
// build runs and no index is ever shared between goroutines. Builds are
// numbered: only the result of the latest one asked for is applied, so a
// slower, older build that arrives late never replaces newer links.
type linkIndexState struct {
	idx     *core.LinkIndex // the index in use; nil until the first build arrives
	gen     uint64          // number of the latest build started
	applied uint64          // number of the build idx came from
	wanted  bool            // a build is due; Update starts it, see startLinkIndex
}

// linkIndexedMsg is a finished build of the link index.
type linkIndexedMsg struct {
	gen uint64
	idx *core.LinkIndex
}

// refreshGraph asks for the links of every note to be read again from
// disk, after the workspace or a note in it changed. The read runs in a
// command once the current message is handled (see Update), and its result
// refreshes the graph, if open, and the preview's backlinks.
func (m *Model) refreshGraph() {
	m.index.wanted = true
}

// startLinkIndex starts the build refreshGraph asked for, if any. The
// command lists and reads the notes through the store, whose methods are
// safe to call from another goroutine, and updates a copy of the index in
// use, so only the notes changed since that one are read again.
func (m *Model) startLinkIndex() tea.Cmd {
	if !m.index.wanted || m.store == nil {
		return nil
	}
	m.index.wanted = false
	m.index.gen++
	gen, store := m.index.gen, m.store
	idx := core.NewLinkIndex(preview.IsImageName)
	if m.index.idx != nil {
		idx = m.index.idx.Clone()
	}
	return func() tea.Msg {
		idx.Update(store.List(), func(id string) (string, error) {
			p, err := store.Get(id)
			return p.Content, err
		})
		return linkIndexedMsg{gen: gen, idx: idx}
	}
}

// applyLinkIndex puts a finished build in use, unless a newer build was
// started since: then it is dropped, and the newer one lands later.
func (m *Model) applyLinkIndex(msg linkIndexedMsg) {
	if msg.gen != m.index.gen {
		return
	}
	m.index.idx, m.index.applied = msg.idx, msg.gen
	if m.bufferIndex(graphID) >= 0 {
		m.graphNotes, m.graphEdges = m.linkGraph()
		m.liveGraph()
	}
	if m.bufferIndex(globalGraphID) >= 0 {
		m.global.wanted = true
	}
	if m.graphActive() {
		m.previewNote(m.graphSelected())
		return
	}
	var back []core.Backlink
	if id := m.currentNoteID(); id != "" {
		back = m.backlinks(id)
	}
	m.preview.SetBacklinks(back)
}

// linkIndexOutOfDate reports whether pages, as a workspace scan listed
// them, have a note the index in use doesn't know, lacks one, or has one
// modified since, or has a note it could not read (one that is unreadable
// for a while keeps its modification time). While a build runs it says no:
// that build may already cover the change, and the next scan compares again
// once it lands.
func (m *Model) linkIndexOutOfDate(pages []core.Page) bool {
	if m.index.gen != m.index.applied {
		return false
	}
	if m.index.idx == nil || m.index.idx.Unread() > 0 {
		return true
	}
	known, i := m.index.idx.Notes(), 0
	for _, p := range pages {
		if p.IsFolder {
			continue
		}
		if i >= len(known) || known[i].ID != p.ID || !known[i].UpdatedAt.Equal(p.UpdatedAt) {
			return true
		}
		i++
	}
	return i != len(known)
}

// linkGraph is every note and the links between them, from the index, or
// read from disk while the first build has not arrived.
func (m *Model) linkGraph() ([]core.Page, []core.Edge) {
	if m.index.idx != nil {
		return m.index.idx.Notes(), m.index.idx.Edges()
	}
	return m.store.LinkGraph()
}

// backlinks lists the notes linking to id, from the index, or read from
// disk while the first build has not arrived.
func (m *Model) backlinks(id string) []core.Backlink {
	if m.index.idx != nil {
		return m.index.idx.Backlinks(id)
	}
	return m.store.Backlinks(id)
}

// liveGraph updates the edited note's links from the editor, so links show
// up in the graph as they are typed. It copies and scans the whole note:
// keys that leave the text as it was don't call it (see editorEdited).
func (m *Model) liveGraph() {
	edges := m.graphEdges
	id, text := m.liveNote()
	if id != "" {
		edges = make([]core.Edge, 0, len(m.graphEdges))
		for _, e := range m.graphEdges {
			if e.From != id {
				edges = append(edges, e)
			}
		}
		m.pv.linkScans++
		for _, to := range core.OutLinks(m.graphNotes, core.Page{ID: id, Content: text}) {
			if to != id {
				edges = append(edges, core.Edge{From: id, To: to})
			}
		}
	}
	m.graph.SetLinks(m.graphNotes, edges)
}

// liveNote is the note whose unsaved text the graphs show links of, and
// that text: the edited note, or, while a graph tab is showing and there
// is no edited note, the note the graph was opened from. So its unsaved
// links stay when a new index, built when another note changes on disk,
// lands.
func (m *Model) liveNote() (id, text string) {
	id = m.currentNoteID()
	switch {
	case id != "" && m.content.HasPage():
		text = m.content.Value()
	case m.graphActive():
		id = ""
		if i := m.bufferIndex(m.graphLiveID); i >= 0 {
			id, text = m.graphLiveID, m.liveText(m.buffers[i])
		}
	default:
		id = ""
	}
	m.graphLiveID = id
	return id, text
}

// previewNote shows a note in the preview without opening it.
func (m *Model) previewNote(id string) {
	p, err := m.store.Get(id)
	if err != nil || id == "" {
		m.preview.SetPage(core.Page{})
		m.preview.SetBacklinks(nil)
		return
	}
	live := map[string]string{}
	for _, b := range m.buffers {
		if !b.draft() {
			live[b.id] = b.text
		}
	}
	if text, ok := live[id]; ok {
		p.Content = text
	}
	m.preview.SetBaseDir(filepath.Join(m.store.Root(), filepath.FromSlash(path.Dir(id))))
	m.preview.SetNotes(preview.StoreNotes(m.store, m.notes, id, live))
	m.preview.SetPage(p)
	m.preview.SetBacklinks(m.backlinks(id))
}

// editorPaneUpdate sends input to whatever the editor pane shows.
func (m *Model) editorPaneUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch {
	case m.globalGraphActive():
		m.globalGraph, cmd = m.globalGraph.Update(msg)
		return cmd
	case m.graphActive():
		m.graph, cmd = m.graph.Update(msg)
		return cmd
	}
	m.content, cmd = m.content.Update(msg)
	return cmd
}
