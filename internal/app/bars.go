package app

import (
	"path/filepath"
	"strings"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Tabufline (top row): buffer tabs and a "+" button.

type tabHit int

const (
	hitNone tabHit = iota
	hitTab
	hitClose
	hitNew
	hitExplorer
)

type tabSpan struct {
	kind       tabHit
	id         string
	start, end int // absolute columns [start, end)
}

const tabMaxName = 20

func (m *Model) renderTab(b *buffer) (string, int) {
	th := m.theme
	active := b.id == m.active
	bg, fg, iconFg := th.Bg2, th.LightGrey, th.GreyFg
	if active {
		bg, fg, iconFg = th.Bg, th.Fg, th.NordBlue
	}
	base := lipgloss.NewStyle().Background(bg)
	name := ui.Truncate(b.fileName(), tabMaxName)

	closeGlyph, closeFg := "󰅖", th.GreyFg
	if active {
		closeFg = th.Red
	}
	if m.isDirty(b) {
		closeGlyph, closeFg = "●", th.Green
		if m.modifiedMark != "" {
			name = m.modifiedMark + " " + name
		}
	}

	left := base.Foreground(iconFg).Render("  ") + base.Foreground(fg).Bold(active).Render(name)
	if b.draft() && !b.isGraph() {
		left += base.Foreground(th.GreyFg).Italic(true).Render(i18n.T(i18n.AppTabNew))
	}
	s := left + base.Render("  ") + base.Foreground(closeFg).Render(closeGlyph) + base.Render(" ")
	if active {
		s = lipgloss.NewStyle().Background(bg).Foreground(th.Blue).Render("┃") + s
	} else {
		s = base.Render(" ") + s
	}
	closeOffset := lipgloss.Width(s) - 2
	return s, closeOffset
}

// tabline renders the top bar and returns clickable spans.
func (m *Model) tabline() (string, []tabSpan) {
	th := m.theme
	fill := lipgloss.NewStyle().Background(th.DarkerBg)
	l := m.layout()
	var spans []tabSpan
	var b strings.Builder
	x := 0

	// The tabs start in the explorer divider's column, so the active tab's
	// marker sits right above the divider instead of one column to its right.
	if l.SidebarW > 0 {
		w := l.SidebarW
		b.WriteString(ui.FitLine("", w, fill))
		spans = append(spans, tabSpan{kind: hitExplorer, start: 0, end: w})
		x = w
	}

	plus := fill.Foreground(th.GreyFg2).Render("  ")
	avail := m.width - x - lipgloss.Width(plus)

	// Lay out tabs, sliding the window so the active tab is visible.
	type rendered struct {
		s        string
		w, close int
		id       string
	}
	tabs := make([]rendered, len(m.buffers))
	activeIdx := 0
	for i, buf := range m.buffers {
		s, c := m.renderTab(buf)
		tabs[i] = rendered{s: s, w: lipgloss.Width(s), close: c, id: buf.id}
		if buf.id == m.active {
			activeIdx = i
		}
	}
	start := 0
	for {
		used := 0
		end := start
		for end < len(tabs) && used+tabs[end].w <= avail {
			used += tabs[end].w
			end++
		}
		if activeIdx < end || start >= activeIdx {
			break
		}
		start++
	}
	if start > 0 {
		b.WriteString(fill.Foreground(th.GreyFg).Render("‹"))
		x++
		avail--
	}
	used := 0
	for i := start; i < len(tabs); i++ {
		t := tabs[i]
		if used+t.w > avail {
			break
		}
		b.WriteString(t.s)
		spans = append(spans,
			tabSpan{kind: hitTab, id: t.id, start: x, end: x + t.w},
			tabSpan{kind: hitClose, id: t.id, start: x + t.close - 1, end: x + t.close + 2},
		)
		x += t.w
		used += t.w
	}
	b.WriteString(plus)
	spans = append(spans, tabSpan{kind: hitNew, start: x, end: x + lipgloss.Width(plus)})
	x += lipgloss.Width(plus)

	return ui.FitLine(b.String(), m.width, fill), spans
}

// tabAt finds what lies under column x of the tab bar. Close buttons win
// over the tab they sit on.
func (m *Model) tabAt(x int) tabSpan {
	_, spans := m.tabline()
	hit := tabSpan{kind: hitNone}
	for _, s := range spans {
		if x >= s.start && x < s.end {
			if s.kind == hitClose {
				return s
			}
			hit = s
		}
	}
	return hit
}

// ---------------------------------------------------------------------------
// Statusline (NvChad "default" style) and command line.

func (m *Model) modeBlock() (string, lipgloss.Color) {
	label, color := m.modeLabel()
	if m.modeIcon != "" {
		// " 󰙅 EXPLORER " → " 󱗞 EXPLORER ": same icon for every mode.
		f := strings.Fields(label)
		label = " " + m.modeIcon + " " + f[len(f)-1] + " "
	}
	return label, color
}

func (m *Model) modeLabel() (string, lipgloss.Color) {
	th := m.theme
	switch {
	case m.viewMode == viewModeDashboard:
		return "  " + i18n.T(i18n.AppModeHome) + " ", th.Blue
	case m.focus == focusSidebar:
		return " 󰙅 " + i18n.T(i18n.AppModeExplorer) + " ", th.Teal
	case m.focus == focusPreview:
		return " 󰈈 " + i18n.T(i18n.AppModePreview) + " ", th.Yellow
	}
	switch m.content.Mode() {
	case content.ModeInsert:
		return "  INSERT ", th.DarkPurple
	case content.ModeVisual:
		return " \U000f0489 VISUAL ", th.Orange
	case content.ModeVisualLine:
		return " \U000f0489 V-LINE ", th.Orange
	case content.ModeCommand:
		return "  COMMAND ", th.Green
	}
	return "  NORMAL ", th.Blue
}

func (m *Model) statusline() string {
	th := m.theme
	fill := lipgloss.NewStyle().Background(th.StatusBg)
	light := lipgloss.NewStyle().Background(th.LightBg)

	b := m.activeBuffer()
	if m.viewMode == viewModeDashboard {
		b = nil
	}
	shown := b
	left := m.statusLeft(shown, statusNameMax, true)

	// Right side, dropped piece by piece when the terminal is narrow.
	var parts []string
	if badge := m.tutorBadge(); badge != "" {
		parts = append(parts, fill.Foreground(th.Green).Render(badge+"  "))
	}
	if b != nil && b.isGraph() {
		b = nil // no text: no word count or file type
	}
	if b != nil {
		words := m.wordCount(b)
		parts = append(parts, fill.Foreground(th.GreyFg2).Render(i18n.N(i18n.AppWords, words, words)))
	}
	if b != nil {
		parts = append(parts, fill.Foreground(th.GreyFg2).Render(" Markdown  "))
	}
	if text, alert := m.syncIndicator(); text != "" {
		fg := th.GreyFg2
		if alert {
			fg = th.Red
		}
		parts = append(parts, fill.Foreground(fg).Render(m.barText(text)+"  "))
	}
	cwd := lipgloss.NewStyle().Background(th.StatusBg).Foreground(th.Red).Render("") +
		lipgloss.NewStyle().Background(th.Red).Foreground(th.DarkFg).Render("󰉋 ") +
		light.Foreground(th.Fg).Render(" "+ui.Truncate(filepath.Base(m.store.Root()), 20)+" ")
	parts = append(parts, cwd)
	pos := "  "
	if b != nil {
		ln, col := m.content.CursorPosition()
		pos = i18n.T(i18n.AppLnCol, ln, col)
	}
	parts = append(parts,
		lipgloss.NewStyle().Background(th.LightBg).Foreground(th.Green).Render("")+
			lipgloss.NewStyle().Background(th.Green).Foreground(th.DarkFg).Render(" ")+
			light.Foreground(th.Green).Render(pos))

	right := strings.Join(parts, "")
	for len(parts) > 1 && lipgloss.Width(left)+lipgloss.Width(right)+1 > m.width {
		parts = parts[1:]
		right = strings.Join(parts, "")
	}
	msg := m.message()
	if compact, ok := m.barNotice(); ok {
		// Only these notices take the parts' room: what arrived, and a stopped
		// sync with the key that answers it, are their whole point. The text
		// alone decides the compact form: an error's hint may be cut.
		fits := func(s string) bool { return lipgloss.Width(left)+lipgloss.Width(right)+lipgloss.Width(s)+1 <= m.width }
		for len(parts) > 1 && !fits(msg) {
			parts = parts[1:]
			right = strings.Join(parts, "")
		}
		if !fits("  " + m.barText(m.status)) {
			msg = m.messageOf(compact)
		}
		left, right, msg = m.fitStopNotice(shown, left, right, msg)
	}
	if msg != "" {
		room := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 1
		if lipgloss.Width(msg) > room {
			msg = ui.FitLine(msg, max(room, 0), fill)
		}
		left += msg
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	line := left + fill.Render(strings.Repeat(" ", max(gap, 0))) + right
	return ui.FitLine(line, m.width, fill)
}

// statusNameMax is the most columns the bar gives a note's name.
const statusNameMax = 32

// statusLeft is the left of the bar: the mode, and with a note open its name and
// folder. A nameMax of zero leaves the note out.
func (m *Model) statusLeft(b *buffer, nameMax int, folder bool) string {
	th := m.theme
	fill := lipgloss.NewStyle().Background(th.StatusBg)
	light := lipgloss.NewStyle().Background(th.LightBg)

	label, color := m.modeBlock()
	left := lipgloss.NewStyle().Background(color).Foreground(th.DarkFg).Bold(true).Render(label)
	if m.viewMode == viewModeDashboard || nameMax <= 0 {
		return left + lipgloss.NewStyle().Background(th.StatusBg).Foreground(color).Render("\ue0b4")
	}
	file := i18n.T(i18n.AppNoFile)
	fileFg := th.GreyFg2
	if b != nil {
		file = b.fileName()
		fileFg = th.Fg
	}
	left += lipgloss.NewStyle().Background(th.LightBg).Foreground(color).Render("")
	left += light.Foreground(th.NordBlue).Render(" 󰈙 ") + light.Foreground(fileFg).Render(ui.Truncate(file, nameMax)+" ")
	if b != nil && m.isDirty(b) {
		left += light.Foreground(th.Green).Render("● ")
	}
	if b != nil && b.draft() && !b.isGraph() {
		left += light.Foreground(th.Yellow).Render(i18n.T(i18n.AppUnsavedMark))
	}
	left += lipgloss.NewStyle().Background(th.StatusBg).Foreground(th.LightBg).Render("")
	// A tutorial session keeps the folder out: every lesson of a route lives in
	// the same one, and its place is the tutorial line's job, which needs the
	// room.
	if folder && b != nil && !b.draft() && m.tutor == nil {
		if dir := filepath.Dir(b.id); dir != "." {
			left += fill.Foreground(th.GreyFg).Render("  󰉋 " + ui.Truncate(dir, 30))
		}
	}
	return left
}

// fitStopNotice keeps a stop's key on the bar from the app's minimum width up,
// since cutting it leaves a message that does not say what to do. The position,
// the folder, the words after the key and the note's name give way, in that
// order and only as far as the message needs. A notice that is no stop keeps
// the bar as it is.
func (m *Model) fitStopNotice(b *buffer, left, right, msg string) (string, string, string) {
	keep, tight := m.sync.noticeStop, m.sync.noticeStopTight
	if keep == "" {
		return left, right, msg
	}
	over := func() int {
		return lipgloss.Width(left) + lipgloss.Width(right) + lipgloss.Width("  "+keep) + 1 - m.width
	}
	if over() > 0 {
		right = ""
	}
	if over() > 0 {
		left = m.statusLeft(b, statusNameMax, false)
	}
	if over() > 0 {
		keep, msg = tight, m.messageOf(tight)
	}
	if n := over(); n > 0 && b != nil {
		left = m.statusLeft(b, max(min(lipgloss.Width(b.fileName()), statusNameMax)-n, 0), false)
	}
	return left, right, msg
}

// bottomBar is the single bottom row: the statusline. The ":" and "/"
// prompts float near the top of the screen (see cmdline.go), so while one is
// open this row stays the statusline with its COMMAND badge.
func (m *Model) bottomBar() string {
	return m.statusline()
}

// message is the transient text shown in the middle of the statusline: the
// status, the leader, the tutorial's step line and the table hint, in that
// order (E8).
func (m *Model) message() string {
	th := m.theme
	switch {
	case m.status != "":
		return m.messageOf(m.status)
	case m.leaderPending:
		return lipgloss.NewStyle().Background(th.StatusBg).Foreground(th.Blue).Bold(true).Render("  <Space>")
	}
	if m.tutor != nil && m.tutor.flash != "" {
		return m.tutorBar(m.tutor.flash, false)
	}
	line := m.tutorLine()
	table := m.tableHint()
	switch {
	case line != "" && table != "":
		return m.tutorBar(line+" · "+table, false)
	case line != "":
		return m.tutorBar(line, false)
	case table != "":
		return m.tutorBar(table, true)
	}
	return ""
}

// tutorBar draws a plain bar message, in the normal or the muted colour.
func (m *Model) tutorBar(text string, muted bool) string {
	fg := m.theme.Fg
	if muted {
		fg = m.theme.GreyFg
	}
	return lipgloss.NewStyle().Background(m.theme.StatusBg).Foreground(fg).Render("  " + m.barText(text))
}

// tableHint is the keys that work in a table, where the caret is, and how to
// add a column without leaving the cell: MISOKU-05's discoverability ask.
func (m *Model) tableHint() string {
	if m.viewMode == viewModeWorkspace && m.focus == focusEditor &&
		m.content.Mode() == content.ModeInsert && m.content.InTable() {
		return i18n.T(i18n.AppTableHint)
	}
	return ""
}

// messageOf draws text as the status: an error in red, with its hint.
func (m *Model) messageOf(text string) string {
	th := m.theme
	fill := lipgloss.NewStyle().Background(th.StatusBg)
	text = m.barText(text)
	if m.statusErr {
		return fill.Foreground(th.Red).Render("  "+text) +
			fill.Foreground(th.GreyFg).Render(i18n.T(i18n.AppErrorHint))
	}
	return fill.Foreground(th.Fg).Render("  " + text)
}
