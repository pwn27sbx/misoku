package app

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// The events of the Básico and Intermedio routes (F3b). Each helper is called
// where the action is confirmed done, outside View, and costs a comparison
// when the tutorial is not playing: tutorDefer answers nil at once then.

// tutorBWants reports whether the current lesson has a step that waits for
// the event name: per-key events ask this before building a value.
func (m *Model) tutorBWants(name string) bool {
	g := m.tutor
	if g == nil || !g.hasCurrent {
		return false
	}
	return checkWants(g.current.Steps, name)
}

// checkWants reports whether one of the steps waits for the event name.
func checkWants(steps []tutor.Step, name string) bool {
	for i := range steps {
		if checkWantsEvent(steps[i].Check, name) {
			return true
		}
	}
	return false
}

// checkWantsEvent walks a check, "all" included.
func checkWantsEvent(c tutor.Check, name string) bool {
	switch c.Kind {
	case tutor.KindAction:
		return c.Event == name
	case tutor.KindAll:
		for _, sub := range c.Checks {
			if checkWantsEvent(sub, name) {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Opening notes

// tutorBNoteOpened reports a note put in a tab. A step with no value asks
// for any note, which is the one the tree opens.
func (m *Model) tutorBNoteOpened() {
	m.tutorDefer("note_opened", "")
}

// tutorBNewTab reports a result opened in a new tab, and the note it opened.
func (m *Model) tutorBNewTab(id string) {
	if !m.tutorBWants("note_opened") {
		return
	}
	m.tutorDefer("note_opened", "new-tab")
	if id != "" {
		m.tutorDefer("note_opened", strings.TrimSuffix(id, ".md"))
	}
}

// tutorBNewDraft reports a blank note. A draft started from a link also
// reports its target's title, which is what the link named.
func (m *Model) tutorBNewDraft(title string) {
	if title != "" {
		m.tutorDefer("new_draft", title)
		return
	}
	m.tutorDefer("new_draft", "")
}

// tutorBNotePreviewed reports the note the explorer showed with "p".
func (m *Model) tutorBNotePreviewed() {
	m.tutorDefer("note_previewed", "")
}

// tutorBFollowedLink reports a link the player followed, with the way it was
// followed: "gd", "gx", "Enter" or "mouse".
func (m *Model) tutorBFollowedLink(key string) {
	m.tutorDefer("followed_link", key)
}

// ---------------------------------------------------------------------------
// Panes, views and the explorer

// tutorBFocusExplorer reports the explorer being opened or closed with
// space e; the overlay that closes with Esc says so.
func (m *Model) tutorBFocusExplorer(value string) {
	m.tutorDefer("focus_explorer", value)
}

// tutorBView reports the note's layout as setNoteView puts it on screen: the
// reading view also shows the concealed markers, and the split is the three
// panes the explorer lesson asks to look at.
func (m *Model) tutorBView(v noteView) {
	switch v {
	case viewReading:
		m.tutorDefer("reading_view", "")
		m.tutorDefer("reading_view", "concealed")
	case viewEditor:
		m.tutorDefer("editor_view", "")
	case viewSplit:
		if m.layout().PreviewW > 0 {
			m.tutorDefer("split_view", "")
		}
	}
}

// tutorBPaneMove moves the focus a pane away and reports the cycle when it
// actually moved.
func (m *Model) tutorBPaneMove(delta int, wrap bool) tea.Cmd {
	before := m.focus
	cmd := m.movePane(delta, wrap)
	if m.focus != before {
		m.tutorDefer("focus_cycle", "")
	}
	return cmd
}

// tutorBPaneResized reports a divider dragged to a new place.
func (m *Model) tutorBPaneResized() {
	m.tutorDefer("pane_resized", "")
}

// tutorBTreeKey reports what a key did to the tree, comparing the folders
// expanded before and after: "collapsed-all" for W, "accordion" for a folder
// that opened, "" for any other fold.
func (m *Model) tutorBTreeKey(before string) {
	if !m.tutorBWants("tree_state") {
		return
	}
	after := m.tutorBTreeExpanded()
	switch {
	case before == after:
	case after == "" && before != "":
		m.tutorDefer("tree_state", "collapsed-all")
	case strings.Count(after, "\n") >= strings.Count(before, "\n")+1:
		// A folder opened. The tree is an accordion: its siblings close.
		m.tutorDefer("tree_state", "accordion")
	default:
		m.tutorDefer("tree_state", "")
	}
}

// tutorBTreeSnapshot is the tree's expanded folders as one string, "" when
// the lesson in play does not wait for the tree.
func (m *Model) tutorBTreeSnapshot() string {
	if !m.tutorBWants("tree_state") {
		return ""
	}
	return m.tutorBTreeExpanded()
}

// tutorBTreeExpanded lists the expanded folders of the visible tree.
func (m *Model) tutorBTreeExpanded() string {
	var b strings.Builder
	for _, it := range m.sidebar.VisibleItems() {
		if it.Expanded {
			b.WriteString(it.Page.ID)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// tutorBTreeScanned reports a note that appeared in the tree from outside
// Misoku, by title: the workspace poll found it.
func (m *Model) tutorBTreeScanned(pages []core.Page) {
	if !m.tutorBWants("tree_refresh") {
		return
	}
	known := make(map[string]bool, len(m.notes))
	for _, p := range m.notes {
		known[p.ID] = true
	}
	for _, p := range pages {
		if !known[p.ID] && !p.IsFolder {
			m.tutorDefer("tree_refresh", p.Title)
		}
	}
}

// ---------------------------------------------------------------------------
// Dialogs, tabs and messages

// tutorBSaveDialog reports the Save As dialog, which shows the folders as the
// player types.
func (m *Model) tutorBSaveDialog() {
	m.tutorDefer("save_dialog", "")
}

// tutorBTabClosed reports a tab closed, which is not the same as a quit.
func (m *Model) tutorBTabClosed() {
	m.tutorDefer("tab_closed", "")
}

// tutorBMessages reports `:messages` shown with the bar's last messages.
func (m *Model) tutorBMessages() {
	m.tutorDefer("messages_open", "")
}

// tutorBTags reports the tag picker `:tags` opens.
func (m *Model) tutorBTags() {
	m.tutorDefer("tags_list", "")
}

// ---------------------------------------------------------------------------
// Palette, finder and the home screen

// tutorBPaletteOpen reports the palette and what it shows: the EXPORT
// entries first and the keys in use, which are the two things the advanced
// lessons ask the player to look at.
func (m *Model) tutorBPaletteOpen(entries []paletteEntry) {
	m.tutorDefer("palette_open", "")
	if len(entries) > 0 && entries[0].group == groupExport {
		m.tutorDefer("palette_open", "export-first")
	}
	m.tutorDefer("palette_open", "configured-keys")
}

// tutorBPaletteRun reports the command a palette entry ran, by its name
// ("settings" for :settings).
func (m *Model) tutorBPaletteRun(e *paletteEntry) {
	if e == nil || e.run == nil {
		return
	}
	name := strings.TrimPrefix(strings.Fields(e.key + " ")[0], ":")
	// A keymap entry's key is a shortcut, not a command: only the ex
	// commands the palette runs have a name to report.
	if name == "" || strings.ContainsAny(name, " ?<") {
		return
	}
	m.tutorDefer("palette_run", name)
}

// tutorBFinderOpen reports the note finder, with the query it opened with.
func (m *Model) tutorBFinderOpen(query string) {
	m.tutorDefer("finder_open", query)
}

// tutorBFinderFilter reports the raw query of the finder on every change,
// which is what the qualifier lessons ask the player to type.
func (m *Model) tutorBFinderFilter(query string) {
	if !m.tutorBWants("finder_filter") {
		return
	}
	m.tutorDefer("finder_filter", query)
}

// tutorBLinkFooter reports the link picker's footer, only shown in Finder
// mode: the inline list has no footer.
func (m *Model) tutorBLinkFooter() {
	m.tutorDefer("link_footer", "")
}

// tutorBHome reports the home screen opening.
func (m *Model) tutorBHome() {
	m.tutorDefer("home", "")
}

// tutorBHomeOpen reports a home entry opened by its number: "recent" or
// "project".
func (m *Model) tutorBHomeOpen(kind string) {
	m.tutorDefer("home_open", kind)
}

// tutorBHomeKey reports one of the home screen's letters.
func (m *Model) tutorBHomeKey(key string) {
	m.tutorDefer("home_key", key)
}

// tutorBLinkNext reports a jump to the next link of the note.
func (m *Model) tutorBLinkNext() {
	m.tutorDefer("link_next", "")
}

// ---------------------------------------------------------------------------
// Preview

// tutorBPreviewKey reports the preview's own keys, by group.
func (m *Model) tutorBPreviewKey(k tea.KeyMsg) {
	switch k.String() {
	case "j", "down", "k", "up", "d", "ctrl+d", "u", "ctrl+u",
		"pgdown", "ctrl+f", " ", "pgup", "ctrl+b", "g", "home", "G", "end":
		m.tutorDefer("preview_scroll", "")
	case "M":
		if m.preview.FoldPending() {
			m.tutorDefer("preview_fold", "collapse")
		}
	case "R":
		if m.preview.FoldPending() {
			m.tutorDefer("preview_fold", "expand")
		}
	case "<", ">", "H", "L":
		m.tutorDefer("calendar_month", "month-shift")
	case "T":
		m.tutorDefer("calendar_month", "today")
	}
}

// tutorBPreviewClick reports a click on the preview's fold triangle or its
// calendar header: the state it changed is read after the click.
func (m *Model) tutorBPreviewClick(hit preview.Hit, ok bool) {
	if !ok {
		return
	}
	switch hit.Kind {
	case "cal:today":
		m.tutorDefer("calendar_month", "today")
	case "cal:prev", "cal:next":
		m.tutorDefer("calendar_month", "month-shift")
	case "fold":
		if m.preview.Folded(hit.Arg) {
			m.tutorDefer("preview_fold", "collapse")
		} else {
			m.tutorDefer("preview_fold", "expand")
		}
	}
}

// tutorBRendered reports what the preview just put on screen: the blocks the
// lessons ask the player to look at, each by its kind.
func (m *Model) tutorBRendered(text string) {
	if !m.tutorBWants("preview_rendered") {
		return
	}
	for _, kind := range previewKinds(text) {
		m.tutorDefer("preview_rendered", kind)
	}
}

// previewKinds names the preview's blocks the lessons look at, in the order
// the preview draws them.
func previewKinds(text string) []string {
	var kinds []string
	if frontMatterHas(text, "properties") {
		kinds = append(kinds, "properties")
	}
	if frontMatterHas(text, "cover") {
		kinds = append(kinds, "cover")
	}
	if hasMarkdownTable(text) {
		kinds = append(kinds, "table")
	}
	if strings.Contains(text, "~~~columns") {
		kinds = append(kinds, "columns")
	}
	if hasFence(text, "math") || strings.Contains(text, "$$") {
		kinds = append(kinds, "equation")
	}
	if hasFence(text, "flow") || hasFence(text, "mermaid") {
		kinds = append(kinds, "diagram")
	}
	if hasImage(text) {
		kinds = append(kinds, "image")
	}
	return kinds
}

// frontMatterHas reports whether the note's front matter carries the key, or
// a property other than cover and icon when want is "properties": that is
// what the preview's table lists.
func frontMatterHas(text, want string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return false
	}
	for _, l := range lines[1:] {
		line := strings.TrimSpace(l)
		if line == "---" {
			return false
		}
		key, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if want == "properties" {
			if key != "cover" && key != "icon" && key != "" {
				return true
			}
			continue
		}
		if strings.EqualFold(key, want) {
			return true
		}
	}
	return false
}

// hasMarkdownTable reports a header row with its separator underneath.
func hasMarkdownTable(text string) bool {
	lines := strings.Split(text, "\n")
	for i := 0; i+1 < len(lines); i++ {
		if !strings.Contains(lines[i], "|") {
			continue
		}
		for _, cell := range strings.Split(strings.TrimSpace(lines[i+1]), "|") {
			cell = strings.TrimSpace(cell)
			if strings.HasPrefix(cell, "---") || strings.HasPrefix(cell, ":--") {
				return true
			}
		}
	}
	return false
}

// hasFence reports a fenced block with the given language.
func hasFence(text, lang string) bool {
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "```"+lang) {
			return true
		}
	}
	return false
}

// hasImage reports a Markdown image or an embedded picture.
func hasImage(text string) bool {
	for _, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "![") {
			return true
		}
	}
	return false
}
