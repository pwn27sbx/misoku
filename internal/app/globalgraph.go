package app

import (
	"path"
	"slices"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/globalgraph"

	tea "github.com/charmbracelet/bubbletea"
)

// The global graph is a second graph tab: every note of the vault and the
// links between them, links to notes that don't exist yet included. It is
// fed from the same link index as the local graph and laid out in a
// command, off the update loop.

// globalGraphID is the global graph's tab. Like graphID it counts as a
// draft that is never dirty and never saved (see buffer.isGraph).
const globalGraphID = draftIDPrefix + "graph-all"

// globalGraphState is what the app keeps to lay the global map out.
//
// Maps are laid out in commands, numbered like the link index builds: only
// the latest one asked for is put in use, so an older map that arrives late
// never replaces a newer one.
type globalGraphState struct {
	gen    uint64            // number of the latest layout started
	wanted bool              // a layout is due; Update starts it, see startGlobalMap
	graph  globalgraph.Graph // the graph of the map in use
	drafts map[string]string // unresolved node ID → the note enter starts, see openGlobalNode
}

// globalMapMsg is a finished layout of the global map. same is set when the
// graph had not changed since the map in use: there is no map to apply.
type globalMapMsg struct {
	gen    uint64
	same   bool
	graph  globalgraph.Graph
	drafts map[string]string
	mp     globalgraph.Map
}

// globalGraphActive reports whether the global graph's tab is showing.
func (m *Model) globalGraphActive() bool { return m.active == globalGraphID }

// openGlobalGraph shows the global graph's tab (":graph all", Space G),
// with the note that was open selected.
func (m *Model) openGlobalGraph() tea.Cmd {
	open := m.currentNoteID()
	m.leaderPending = false
	m.leaveReading() // the graph is drawn in the editor column
	m.liveNote()     // the note whose unsaved links the map shows, see liveNote
	m.globalGraph.SetOpenNote(open)
	m.global.wanted = true
	if i := m.bufferIndex(globalGraphID); i >= 0 {
		m.showBuffer(m.buffers[i])
	} else {
		b := &buffer{id: globalGraphID, title: "Global graph"}
		m.insertBuffer(b)
		m.showBuffer(b)
	}
	return m.focusPane(focusEditor)
}

// startGlobalMap starts the layout asked for, if any and if the tab is
// open. Before the first link index build lands (New asks for it) there is
// nothing to lay out: the request waits, and applyLinkIndex makes it again.
//
// The command gets everything it reads: the index in use, which the update
// loop only ever replaces, the unsaved text of the note liveNote picks, and
// the map in use to keep its positions. It works on its own copy of the
// index when it adds that text.
func (m *Model) startGlobalMap() tea.Cmd {
	if !m.global.wanted || m.bufferIndex(globalGraphID) < 0 || m.index.idx == nil {
		return nil
	}
	m.global.wanted = false
	m.global.gen++
	gen, idx, prev, last := m.global.gen, m.index.idx, m.globalGraph.Layout(), m.global.graph
	id, text := m.liveNote()
	return func() tea.Msg {
		x := idx
		if id != "" {
			x = idx.Clone()
			x.SetContent(id, text)
		}
		g, drafts := globalGraphOf(x.Notes(), x.Edges(), x.Unresolved())
		if slices.Equal(g.Nodes, last.Nodes) && slices.Equal(g.Edges, last.Edges) {
			return globalMapMsg{gen: gen, same: true}
		}
		return globalMapMsg{gen: gen, graph: g, drafts: drafts, mp: globalgraph.NewMap(prev, g)}
	}
}

// applyGlobalMap puts a finished layout in use, unless a newer one was
// started since. The preview follows the selection if the map moved it.
func (m *Model) applyGlobalMap(msg globalMapMsg) {
	if msg.gen != m.global.gen || msg.same {
		return
	}
	before := m.globalGraph.Selected()
	m.global.graph, m.global.drafts = msg.graph, msg.drafts
	m.globalGraph.SetMap(msg.mp)
	if sel := m.globalGraph.Selected(); m.globalGraphActive() && sel != before {
		m.previewNote(sel)
	}
}

// openGlobalNode opens a note picked in the global graph. A link target
// with no note starts that note, as following the link does.
func (m *Model) openGlobalNode(msg globalgraph.OpenMsg) tea.Cmd {
	if !msg.Unresolved {
		return m.openFile(msg.ID)
	}
	target, ok := m.global.drafts[msg.ID]
	if !ok {
		target = msg.ID
	}
	return m.newLinkedDraft(target)
}

// globalGraphOf turns the index's notes, links and unresolved links into
// the map's graph. Each note's group is its top-level folder; a folder
// note ("Foo.md" next to "Foo/") goes with its folder.
//
// Every spelling of one missing note is one node, named by the first one
// in note order; it sits in the group of the folder the note would be
// created in. drafts maps each such node to that note's path, for
// newLinkedDraft: a bare name goes next to the first note linking to it.
func globalGraphOf(notes []core.Page, edges []core.Edge, unresolved []core.UnresolvedLink) (globalgraph.Graph, map[string]string) {
	folders := map[string]bool{}
	ids := make(map[string]bool, len(notes))
	for _, p := range notes {
		ids[p.ID] = true
		if top, _, ok := strings.Cut(p.ID, "/"); ok {
			folders[top] = true
		}
	}
	cluster := func(p string) string {
		if top, _, ok := strings.Cut(p, "/"); ok {
			return top
		}
		if top := strings.TrimSuffix(p, ".md"); folders[top] {
			return top
		}
		return ""
	}

	var g globalgraph.Graph
	for _, p := range notes {
		g.Nodes = append(g.Nodes, globalgraph.Node{ID: p.ID, Cluster: cluster(p.ID)})
	}
	for _, e := range edges {
		g.Edges = append(g.Edges, globalgraph.Edge{From: e.From, To: e.To})
	}

	drafts := map[string]string{}
	byKey := map[string]string{} // Key → node ID
	seen := map[globalgraph.Edge]bool{}
	for _, u := range unresolved {
		id, ok := byKey[u.Key]
		if !ok {
			id = strings.TrimSuffix(strings.Trim(strings.ReplaceAll(strings.TrimSpace(u.Target), `\`, "/"), "/"), ".md")
			if id == "" || ids[id] { // never take a note's place
				continue
			}
			byKey[u.Key] = id
			draft := id
			if !strings.Contains(id, "/") {
				if dir := path.Dir(u.From); dir != "." {
					draft = dir + "/" + id
				}
			}
			drafts[id] = draft
			g.Nodes = append(g.Nodes, globalgraph.Node{ID: id, Cluster: cluster(draft), Unresolved: true})
		}
		if e := (globalgraph.Edge{From: u.From, To: id}); !seen[e] {
			seen[e] = true
			g.Edges = append(g.Edges, e)
		}
	}
	return g, drafts
}
