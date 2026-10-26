package globalgraph

// The filter and the toggles. The filter greys out the nodes whose name and
// path don't hold its text; the toggles hide notes without links (orphans)
// and link targets with no note (unresolved). Neither moves a node: they act
// when the map is drawn and when a node is picked, never on the layout, so
// the map keeps its shape.

import (
	"cmp"
	"math"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// hides reports whether a toggle hides n: an orphan is a note with no link
// in or out, as the title row counts them.
func hides(n Placed, orphans, unresolved bool) bool {
	if n.Unresolved {
		return unresolved
	}
	return orphans && n.Degree == 0
}

// noMatch is matchRank's result for a node the filter doesn't match.
const noMatch = -1

// matchRank tells how well n matches q, already lower case: 0 when its name
// starts with q, 1 when its name holds q, 2 when only its path does, and
// noMatch otherwise. The name is the file name without ".md".
func matchRank(n Placed, q string) int {
	switch name := strings.ToLower(title(n.ID)); {
	case strings.HasPrefix(name, q):
		return 0
	case strings.Contains(name, q):
		return 1
	case strings.Contains(strings.ToLower(n.ID), q):
		return 2
	}
	return noMatch
}

func title(id string) string { return path.Base(strings.TrimSuffix(id, ".md")) }

// selectable reports whether id is a node on the map that no toggle hides.
func (m Model) selectable(id string) bool {
	i, ok := m.l.index[id]
	return ok && id != "" && !hides(m.l.Nodes[i], m.hideOrphans, m.hideUnresolved)
}

// bestMatch is the node enter jumps to: the best matchRank, then the
// shortest name, then the smaller ID; "" when nothing shown matches.
func (m Model) bestMatch() string {
	q := strings.ToLower(m.filter)
	best, bestRank, bestW := "", 0, 0
	for _, n := range m.l.Nodes {
		rank := matchRank(n, q)
		if rank == noMatch || !m.selectable(n.ID) {
			continue
		}
		w := ansi.StringWidth(title(n.ID))
		if best == "" || cmp.Or(cmp.Compare(rank, bestRank), cmp.Compare(w, bestW), cmp.Compare(n.ID, best)) < 0 {
			best, bestRank, bestW = n.ID, rank, w
		}
	}
	return best
}

// promptKey types into the open prompt. Every printable key is text;
// enter jumps to the best match and keeps the filter, esc clears it.
func (m *Model) promptKey(k tea.KeyMsg) {
	switch k.Type {
	case tea.KeyEnter:
		m.typing = false
		if m.filter != "" {
			m.pick(m.bestMatch())
		}
	case tea.KeyEsc:
		m.typing, m.filter = false, ""
	case tea.KeyBackspace:
		if r := []rune(m.filter); len(r) > 0 {
			m.filter = string(r[:len(r)-1])
		}
	case tea.KeySpace:
		m.filter += " "
	case tea.KeyRunes:
		m.filter += string(k.Runes)
	}
}

// toggle hides or shows orphans or unresolved targets, keeping the
// selection on a node that is shown.
func (m *Model) toggle(action string) {
	if action == "orphans" {
		m.hideOrphans = !m.hideOrphans
	} else {
		m.hideUnresolved = !m.hideUnresolved
	}
	m.keepSelectionShown()
}

// keepSelectionShown moves a selection a toggle hides to the open note or,
// when that is hidden too, to the nearest node shown (a row counts as
// cellAspect columns; ties go to the smaller ID). With nothing selected,
// as after a toggle hid every node, or with the selection gone from the
// map, it goes by the open note in the same way; with no open note on the
// map, nothing is selected.
func (m *Model) keepSelectionShown() {
	if m.selectable(m.sel) {
		return
	}
	origin := m.sel
	if !m.exists(origin) {
		origin = m.open
	}
	if !m.exists(origin) {
		m.sel = ""
		return
	}
	from, _ := m.l.Lookup(origin)
	if m.selectable(m.open) {
		m.sel = m.open
	} else {
		best, bestD := "", math.Inf(1)
		for _, n := range m.l.Nodes {
			dx, dy := float64(n.P.X-from.P.X), cellAspect*float64(n.P.Y-from.P.Y)
			if d := math.Hypot(dx, dy); m.selectable(n.ID) && d < bestD {
				best, bestD = n.ID, d
			}
		}
		m.sel = best
	}
	m.ringAt = ""
	m.reveal()
}

// promptLine is the hint row while the prompt is open: the text, its end
// in sight when it is too long, then the keys that close it where they fit.
func (m Model) promptLine() string {
	for _, tail := range []string{"   enter jump · esc clear", "   enter · esc", ""} {
		room := m.width - 3 - 1 - ansi.StringWidth(tail) // " / ", the cursor
		if room < 1 {
			continue
		}
		text := m.filter
		if w := ansi.StringWidth(text); w > room {
			text = "…" + ansi.TruncateLeft(text, w-room+1, "")
		}
		if tail == "" || ansi.StringWidth(m.filter) <= room {
			return " / " + text + "▏" + tail
		}
	}
	return " / ▏"
}
