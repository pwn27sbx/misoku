package app

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// linkFinderHelp is the link picker's footer: what "#" and "^" mean there.
const linkFinderHelp = "# heading · ^ block · Esc type it yourself"

// finderOpenLink finishes a "[[" with the selected suggestion: its text, or
// the typed name when it matches no note, which following starts. A query
// with qualifiers names no note and inserts nothing.
func (m *Model) finderOpenLink(f *finder) tea.Cmd {
	insert := ""
	if it, ok := f.selectedLink(); ok {
		insert = it.Insert
	} else if fq := parseFinderQueryHashes(f.input.Value(), false); !fq.qualified {
		insert = core.LinkCompletion(f.input.Value())
	}
	m.finder = nil
	if insert != "" {
		m.insertWikiText(insert)
	}
	return m.focusPane(focusEditor)
}

// refilterLinks fills the link picker's list from the text typed after "["[.
// A leading "#" means headings here, not a tag; status:, tag: and folder:
// still narrow the notes first.
func (f *finder) refilterLinks() {
	q := f.input.Value()
	fq := parseFinderQueryHashes(q, false)
	notes := f.linkNotes()
	if fq.qualified {
		notes, q = f.filteredLinkNotes(fq), fq.text
	}
	f.links = core.LinkSuggestions(notes, f.current, f.currentText, q)
	f.matches = f.matches[:0]
	f.sel, f.offset = 0, 0
}

// linkNotes is every note as a suggestion source, worked out once per
// picker from the text the finder already holds.
func (f *finder) linkNotes() []core.LinkNote {
	if f.linkSrc != nil {
		return f.linkSrc
	}
	f.linkSrc = make([]core.LinkNote, 0, len(f.notes))
	for _, p := range f.notes {
		f.linkSrc = append(f.linkSrc, core.LinkNote{
			ID: p.ID, Title: p.Title,
			Text: strings.Join(f.lines[p.ID], "\n"),
			// The text comes from the finder's own copy, so headings and
			// block ids are read without touching the disk.
			UpdatedAt: p.UpdatedAt,
		})
	}
	return f.linkSrc
}

// filteredLinkNotes keeps the notes that pass the picker's qualifiers.
func (f *finder) filteredLinkNotes(fq finderQuery) []core.LinkNote {
	keep := make(map[string]bool, len(f.notes))
	for _, p := range f.notes {
		if f.passes(p, fq) {
			keep[p.ID] = true
		}
	}
	var out []core.LinkNote
	for _, n := range f.linkNotes() {
		if keep[n.ID] {
			out = append(out, n)
		}
	}
	return out
}

// linkIcon marks a suggestion's kind.
func linkIcon(kind string) string {
	switch kind {
	case core.LinkKindHeading:
		return "#"
	case core.LinkKindBlock:
		return "^"
	}
	return "\U000f0337"
}

// linkRow renders one suggestion of the link picker.
func (f *finder) linkRow(th theme.Theme, idx, width int) string {
	bg := lipgloss.NewStyle().Background(th.PopupBg)
	if idx >= len(f.links) {
		return ui.FitLine("", width, bg)
	}
	s := f.links[idx]
	rowBg := th.PopupBg
	if idx == f.sel {
		rowBg = th.OneBg2
	}
	base := lipgloss.NewStyle().Background(rowBg)
	marker := base.Render("  ")
	labelFg, iconFg := th.Fg, th.GreyFg2
	if idx == f.sel {
		marker = base.Foreground(th.Blue).Render("▌ ")
		iconFg = th.Blue
	}
	line := marker + base.Foreground(iconFg).Render(linkIcon(s.Kind)+" ") +
		base.Foreground(labelFg).Render(ui.Truncate(s.Label, width-6))
	if s.Detail != "" && lipgloss.Width(line) < width {
		detail := base.Foreground(th.GreyFg2).Render(ui.Truncate(s.Detail, width/2))
		if gap := width - lipgloss.Width(line) - lipgloss.Width(detail); gap > 1 {
			line += base.Render(strings.Repeat(" ", gap)) + detail
		}
	}
	return ui.FitLine(line, width, base)
}
