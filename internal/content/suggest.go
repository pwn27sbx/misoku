package content

import (
	"slices"
	"strings"

	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SuggestMaxRows caps the rows of the command list shown under ":" (the
// design keeps it shorter than LazyVim's 15).
const SuggestMaxRows = 8

// suggestState is the list of commands that match the ":" line.
type suggestState struct {
	list []int // indexes into Commands, in catalog order
	sel  int   // index into list; -1 while no row is selected
}

// InsertText is what selecting the command writes into the command line:
// its Usage without ":", the part before " / ", and without its
// placeholder, ending in a space when there was one, so the argument
// follows.
func (c Command) InsertText() string {
	s := strings.TrimPrefix(c.Usage, ":")
	if before, _, ok := strings.Cut(s, " / "); ok {
		s = before
	}
	if i := strings.LastIndex(s, " ["); i >= 0 && strings.HasSuffix(s, "]") {
		return s[:i+1]
	}
	if strings.HasSuffix(s, " name") {
		return strings.TrimSuffix(s, "name")
	}
	return s
}

// matchesCommand reports whether c belongs in the list for the typed text:
// one word matches any of its names by prefix; once there is a space the
// whole text must start its insert text.
func matchesCommand(c Command, typed string) bool {
	if strings.Contains(typed, " ") {
		return strings.HasPrefix(c.InsertText(), typed)
	}
	return slices.ContainsFunc(c.Names, func(n string) bool { return strings.HasPrefix(n, typed) })
}

// refilterSuggestions lists the commands matching the ":" line again and
// drops the selection. The "/" prompt and an empty line list nothing.
func (m *Model) refilterSuggestions() {
	m.suggest = suggestState{sel: -1}
	typed := m.cmdInput.Value()
	if m.search || typed == "" {
		return
	}
	for i, c := range Commands {
		if matchesCommand(c, typed) {
			m.suggest.list = append(m.suggest.list, i)
		}
	}
}

// SuggestionsOpen reports whether the ":" line shows its command list.
func (m Model) SuggestionsOpen() bool {
	return m.mode == ModeCommand && !m.search && len(m.suggest.list) > 0
}

// moveSuggestion handles the list keys of the ":" line: Tab/Down select
// the next row, Shift+Tab/Up the previous one, wrapping. The selected row's
// insert text goes into the line and the list stays. The "/" prompt never
// has a list, so there these keys do nothing, as before. handled is false
// for any other key.
func (m *Model) moveSuggestion(k tea.KeyMsg) (handled bool) {
	step := 0
	switch k.String() {
	case "tab", "down":
		step = 1
	case "shift+tab", "up":
		step = -1
	default:
		return false
	}
	n := len(m.suggest.list)
	if n == 0 {
		return true // no list: the keys do nothing
	}
	switch {
	case m.suggest.sel < 0 && step > 0:
		m.suggest.sel = 0
	case m.suggest.sel < 0:
		m.suggest.sel = n - 1
	default:
		m.suggest.sel = (m.suggest.sel + step + n) % n
	}
	m.cmdInput.SetValue(Commands[m.suggest.list[m.suggest.sel]].InsertText())
	m.cmdInput.CursorEnd()
	return true
}

// SuggestionsView draws the ":" command list as a box exactly width
// columns wide, framed in border, with at most maxRows rows (and never more
// than SuggestMaxRows). When the matches do not fit, the rows shown scroll
// to keep the selection visible. ok is false when there is no list or not
// even one row fits.
func (m Model) SuggestionsView(width, maxRows int, border lipgloss.Color) (box string, ok bool) {
	if !m.SuggestionsOpen() || maxRows < 1 || width < 3 {
		return "", false
	}
	list, sel := m.suggest.list, m.suggest.sel
	rows := min(len(list), maxRows, SuggestMaxRows)
	start := 0
	if sel >= rows {
		start = sel - rows + 1
	}
	th := m.theme
	inner := width - 2
	bg := lipgloss.NewStyle().Background(th.PopupBg)
	lines := make([]string, 0, rows)
	for i := start; i < start+rows; i++ {
		c := Commands[list[i]]
		base := bg
		if i == sel {
			base = lipgloss.NewStyle().Background(th.OneBg2)
		}
		line := base.Foreground(th.Fg).Render(" "+c.Usage) +
			base.Foreground(th.GreyFg).Render("  "+c.HelpText())
		lines = append(lines, ui.FitLineEllipsis(line, inner, base))
	}
	return lipgloss.NewStyle().
		BorderStyle(ui.Border()).
		BorderForeground(border).
		BorderBackground(th.PopupBg).
		Render(strings.Join(lines, "\n")), true
}
