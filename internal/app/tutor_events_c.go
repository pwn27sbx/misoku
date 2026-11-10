package app

// The events the lessons of routes Avanzado and Experto wait for (F3c). Each
// one is reported where the app confirmed the action (see tutor_hooks.go and
// internal/content/actions.go for the convention); none of them costs
// anything outside `misoku tutor`, and each checks with the tutorial first
// before it builds a value or classifies a key.

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// settingsSavedMsg is a settings write that reached the config file; the
// tutorial credits it as its settings_saved event.
type settingsSavedMsg struct{}

// tutorCWants reports whether the current lesson waits for event. The
// per-key events check it before building a value, so a key the lesson does
// not watch costs one scan of a short step list.
func (m *Model) tutorCWants(event string) bool {
	return m.tutorCWantsAny(event)
}

// tutorCWantsAny is tutorCWants for any of events, in one scan.
func (m *Model) tutorCWantsAny(events ...string) bool {
	g := m.tutor
	if g == nil || !g.hasCurrent {
		return false
	}
	for i := range g.current.Steps {
		if g.store.Passed(g.current, i+1) {
			continue // the step is done: no event is needed for it
		}
		for _, event := range events {
			if tutorCCheckWants(g.current.Steps[i].Check, event) {
				return true
			}
		}
	}
	return false
}

// tutorCCheckWants reports whether c, or a check inside it, waits for event.
func tutorCCheckWants(c tutor.Check, event string) bool {
	if c.Kind == tutor.KindAction && c.Event == event {
		return true
	}
	for i := range c.Checks {
		if tutorCCheckWants(c.Checks[i], event) {
			return true
		}
	}
	return false
}

// tutorCGraphInput credits a key or a mouse gesture the graph pane just
// handled. typing is whether the global graph's filter prompt was open
// before the event, since every key is then text.
func (m *Model) tutorCGraphInput(msg tea.Msg, typing bool) tea.Cmd {
	if !m.tutorCWantsAny("graph_move", "graph_mouse", "graph_filter") {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if typing {
			return nil
		}
		return m.tutorCGraphKey(msg.String())
	case tea.MouseMsg:
		// The local map pans, zooms, drags and opens with the mouse.
		if m.active != graphID || m.graph.Centre() == "" {
			return nil
		}
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown ||
			msg.Action == tea.MouseActionPress {
			return m.tutorEvent("graph_mouse", "")
		}
	}
	return nil
}

// tutorCGraphKey names the graph action key k runs, once the graph has a map
// to act on. The global map accepts the local map's keys plus n, N, esc, o
// and u, and / opens its filter.
func (m *Model) tutorCGraphKey(k string) tea.Cmd {
	if m.active == graphID {
		if m.graph.Centre() == "" {
			return nil
		}
		switch k {
		case "h", "l":
			return m.tutorEvent("graph_move", "hl")
		case "j", "k":
			return m.tutorEvent("graph_move", "jk")
		case "enter":
			return m.tutorEvent("graph_move", "enter")
		case "+", "-":
			return m.tutorEvent("graph_move", "zoom")
		case "0":
			return m.tutorEvent("graph_move", "center")
		}
		return nil
	}
	if len(m.globalGraph.Layout().Nodes) == 0 {
		return nil
	}
	switch k {
	case "/":
		return m.tutorEvent("graph_filter", "")
	case "h", "l", "j", "k", "left", "right", "up", "down",
		"n", "N", "enter", "+", "-", "0", "esc", "o", "u":
		return m.tutorEvent("graph_move", k)
	}
	return nil
}

// tutorCGraphRefreshed credits a graph that just took in the vault's new
// links, written in another tab or on disk.
func (m *Model) tutorCGraphRefreshed() {
	if !m.tutorCWants("graph_refresh") {
		return
	}
	if m.bufferIndex(graphID) < 0 && m.bufferIndex(globalGraphID) < 0 {
		return
	}
	m.tutorDefer("graph_refresh", "")
}

// tutorCHover credits the preview's block handles once the mouse rests on a
// block and the handles are on screen.
func (m *Model) tutorCHover() tea.Cmd {
	if !m.tutorCWants("block_handles") || !m.preview.Hovering() {
		return nil
	}
	return m.tutorEvent("block_handles", "visible")
}

// tutorCBacklinks credits the backlinks the preview is about to list under
// the current lesson's note.
func (m *Model) tutorCBacklinks(back []core.Backlink) {
	g := m.tutor
	if len(back) == 0 || !m.tutorCWants("backlinks") || m.currentNoteID() != g.current.File {
		return
	}
	m.tutorDefer("backlinks", "")
}

// tutorCCaughtUp credits the backlinks a lesson found already on screen
// when its note became current: they were put there before the tutorial was
// watching that note, so the change that drew them is never seen again
// (LNK-22). It runs once per lesson, and only while its step is open.
func (m *Model) tutorCCaughtUp() tea.Cmd {
	g := m.tutor
	if g == nil || !g.hasCurrent || m.tutorCSeen == g.current.ID {
		return nil
	}
	if len(m.preview.Backlinks()) == 0 || !m.tutorCWants("backlinks") || m.currentNoteID() != g.current.File {
		return nil
	}
	m.tutorCSeen = g.current.ID
	return m.tutorEvent("backlinks", "")
}

// tutorCNoteRow tells a click on an unlinked mention's row (it opens the
// note at its mention) from a click on a backlink's row (PRV-31, PRV-33).
func (m *Model) tutorCNoteRow(id string) tea.Cmd {
	if !m.tutorCWantsAny("mention_open", "backlink_open") {
		return nil
	}
	if _, ok := m.mentionRow(id); ok {
		return m.tutorEvent("mention_open", "")
	}
	for _, b := range m.backlinks(m.currentNoteID()) {
		if b.ID == id {
			return m.tutorEvent("backlink_open", "")
		}
	}
	return nil
}

// tutorCStatusGlyphs credits the status glyphs the explorer and the home
// screen just took in, and the "none" of a status the app does not know,
// which draws no glyph (PROP-06, NOTE-24, PROP-08).
func (m *Model) tutorCStatusGlyphs(statuses map[string]core.NoteStatus) {
	if !m.tutorCWants("status_glyph") {
		return
	}
	seen := map[string]bool{}
	for _, st := range statuses {
		if v := string(st); v != "" && !seen[v] {
			seen[v] = true
			m.tutorDefer("status_glyph", v)
		}
	}
	id := m.currentNoteID()
	if id == "" || statuses[id] != core.StatusNone {
		return
	}
	if f, ok := core.ParseFrontMatter(m.content.Value()).Get("status"); ok && strings.TrimSpace(f.Value) != "" {
		m.tutorDefer("status_glyph", "none")
	}
}

// tutorCDirbuf credits the folder editor opening from the explorer: m, or
// the insert key that opened it on the selected note (NOTE-17, NOTE-19).
func (m *Model) tutorCDirbuf(key string, wasEditing bool) tea.Cmd {
	if !m.tutorCWants("dirbuf_open") || wasEditing || !m.sidebar.IsEditing() {
		return nil
	}
	switch key {
	case "i", "a", "o":
		return m.tutorEvent("dirbuf_open", key)
	}
	return m.tutorEvent("dirbuf_open", "")
}

// tutorCCursorShape credits the shape the terminal cursor now draws: a bar
// in INSERT, a block anywhere else (VIEW-11). Only when the real cursor is
// in use and the editor holds it: otherwise there is no shape on screen.
func (m *Model) tutorCCursorShape(mode content.VimMode) tea.Cmd {
	if !m.tutorCWants("cursor_shape") || m.cursor == nil {
		return nil
	}
	if !m.editorHasTheCursor(m.layout()) {
		return nil
	}
	if mode == content.ModeInsert {
		return m.tutorEvent("cursor_shape", "bar")
	}
	return m.tutorEvent("cursor_shape", "block")
}

// tutorCKeyClash credits the warning of a key bound to two actions, kept
// from the keymap the program started with (SRCH-10). The current lesson is
// the one resumed when the tutorial opens.
func (m *Model) tutorCKeyClash() tea.Cmd {
	g := m.tutor
	if g == nil || !g.hasCurrent || !m.keyClash {
		return nil
	}
	m.keyClash = false
	return m.tutorEvent("key_clash", "")
}

// tutorCLeaderGuide credits the leader guide that just appeared, and its
// subpage when the keys typed after Space opened one (SRCH-07, SRCH-08).
func (m *Model) tutorCLeaderGuide() {
	if m.tutor == nil {
		return
	}
	value := ""
	if len(m.leaderKeys) > 0 && m.leaderKeys[0] == "o" {
		value = "subpage"
	}
	m.tutorDefer("leader_guide", value)
}

// tutorCSyncIndicator credits a change of the sync indicator (SYNC-07,
// SYNC-18): prev is what it showed before.
func (m *Model) tutorCSyncIndicator(prev syncKind) {
	if m.tutor == nil || m.sync.kind == prev {
		return
	}
	switch m.sync.kind {
	case syncOK:
		m.tutorDefer("sync_indicator", "synced")
	case syncOffline:
		m.tutorDefer("sync_indicator", "offline")
	case syncPaused, syncError, syncRetry:
		m.tutorDefer("sync_indicator", "paused")
	case syncBusy:
		m.tutorDefer("sync_indicator", "other-window")
	}
}

// tutorCSyncStatus credits the sync status screen as it opens, with its list
// of files that stay out when there is one (SYNC-06, SYNC-17).
func (m *Model) tutorCSyncStatus() {
	if m.tutor == nil {
		return
	}
	m.tutorDefer("sync_status", "")
	if len(m.sync.files) > 0 {
		m.tutorDefer("sync_status", "left-out")
	}
}
