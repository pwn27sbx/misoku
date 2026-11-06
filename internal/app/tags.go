package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// openTagPicker is ":tags": every tag of the vault with how many notes carry
// it, from the link index; picking one opens the finder on tag:<name>.
func (m *Model) openTagPicker() tea.Cmd {
	if m.index.idx == nil {
		m.setStatus("Indexing notes…")
		return nil
	}
	counts := m.index.idx.Tags()
	if len(counts) == 0 {
		m.setStatus("No tags yet: write #tag in a note, or tags: in its front matter")
		return nil
	}
	names := make([]string, 0, len(counts))
	for t := range counts {
		names = append(names, t)
	}
	sort.Strings(names)
	items := make([]string, len(names))
	for i, t := range names {
		unit := "notes"
		if counts[t] == 1 {
			unit = "note"
		}
		items[i] = fmt.Sprintf("#%s · %d %s", t, counts[t], unit)
	}
	m.searchMenu("Tags", items, 0, func(m *Model, i int) tea.Cmd {
		return m.openFinderWith("tag:" + finderValue(names[i]))
	})
	return nil
}

// openStatusFinder is ":status [value]": the finder on status:<value>, or
// on status: with the four values listed when st is StatusNone.
func (m *Model) openStatusFinder(st core.NoteStatus) tea.Cmd {
	return m.openFinderWith("status:" + string(st))
}

// finderValue quotes a qualifier value with spaces, as a front matter tag
// may have, so the finder reads it whole.
func finderValue(v string) string {
	if strings.ContainsAny(v, " \t") {
		return `"` + v + `"`
	}
	return v
}
