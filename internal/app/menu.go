package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// menuDialog is a list to pick one item from, by arrows, number or click.
// In a list of ten or more a digit does not pick at once: the digits
// typed so far (number) highlight their item and Enter picks it. A first
// digit picking at once took "5" of "54" in a long repository list, and
// the "4" fell through to the screen behind (issue #7).
type menuDialog struct {
	title    string
	items    []string
	onChoose func(m *Model, i int) tea.Cmd
	// onCancel, when set, runs once the list closed without a pick.
	onCancel func(m *Model) tea.Cmd
	// search: typing filters the items (searchMenu); the first pinned
	// items are always listed, above the matches.
	search bool
	pinned int
	query  string
	// filtering: "/" was typed, so digits go to the query too.
	filtering bool
	// number: the digits typed so far, in a list of ten or more, kept as
	// typed even when the list has no such item (numbered tells).
	number string
	// notice is shown, wrapped, above the list: why the last pick was
	// not taken, readable whole rather than cut off in the status bar.
	notice string
	// shown: the indexes of the items listed, in order; sel is a
	// position in it, offset the first one in view.
	shown       []int
	sel, offset int
}

// menu opens a list to pick one of items from.
func (m *Model) menu(title string, items []string, onChoose func(*Model, int) tea.Cmd) {
	m.menuBox = newMenu(title, items, onChoose)
}

// searchMenu opens a list that typing filters, like the theme picker's;
// its first pinned items stay listed whatever is typed.
func (m *Model) searchMenu(title string, items []string, pinned int, onChoose func(*Model, int) tea.Cmd) {
	d := newMenu(title, items, onChoose)
	d.search, d.pinned = true, pinned
	m.menuBox = d
}

func newMenu(title string, items []string, onChoose func(*Model, int) tea.Cmd) *menuDialog {
	d := &menuDialog{title: title, items: items, onChoose: onChoose}
	d.filter()
	return d
}

// numberedAtOnce is the length below which a digit picks its item at once.
const numberedAtOnce = 10

// filter lists the pinned items, then those containing the query, in any
// letter case.
func (d *menuDialog) filter() {
	q := strings.ToLower(d.query)
	d.shown = d.shown[:0]
	for i, it := range d.items {
		if i < d.pinned || strings.Contains(strings.ToLower(it), q) {
			d.shown = append(d.shown, i)
		}
	}
	d.sel, d.offset = 0, 0
}

// rows is how many items the dialog lists at a time.
func (d *menuDialog) rows(m *Model) int {
	return min(len(d.items), max(m.height-10-len(d.noticeLines(m)), 3))
}

// width is the dialog's width, borders included: room for the items, and
// for a notice in a few lines.
func (d *menuDialog) width(m *Model) int {
	w := 34
	for _, it := range d.items {
		w = max(w, lipgloss.Width(it)+8)
	}
	if d.notice != "" {
		w = max(w, 64)
	}
	return min(w, m.width-4)
}

// noticeLines is the notice wrapped to the dialog's width.
func (d *menuDialog) noticeLines(m *Model) []string {
	if d.notice == "" {
		return nil
	}
	return strings.Split(ansi.Wrap(d.notice, max(d.width(m)-4, 10), ""), "\n")
}

// header is how many rows come before the first item: the title, the
// notice, the search field when there is one, and a blank row.
func (d *menuDialog) header(m *Model) int {
	n := 2 + len(d.noticeLines(m))
	if d.search {
		n++
	}
	return n
}

func (d *menuDialog) geom(m *Model) (x, y, w, h int) {
	w = d.width(m)
	h = d.rows(m) + d.header(m) + 2
	x, y = ui.Center(m.width, m.height, w, h)
	return
}

// numbered reports whether the typed number names an item of the list.
func (d *menuDialog) numbered() bool {
	v, err := strconv.Atoi(d.number)
	return err == nil && d.number[0] != '0' && v <= len(d.shown)
}

// highlightNumber moves the highlight to the typed number's item, when
// the list has it.
func (d *menuDialog) highlightNumber(m *Model) {
	if d.number != "" && d.numbered() {
		v, _ := strconv.Atoi(d.number)
		d.sel = v - 1
		d.scroll(m)
	}
}

// move moves the highlight by delta, wrapping, and drops a typed number.
func (d *menuDialog) move(m *Model, delta int) {
	d.number = ""
	if len(d.shown) == 0 {
		return
	}
	d.sel = (d.sel + delta + len(d.shown)) % len(d.shown)
	d.scroll(m)
}

// scroll keeps the highlighted item in view.
func (d *menuDialog) scroll(m *Model) {
	rows := d.rows(m)
	if d.sel < d.offset {
		d.offset = d.sel
	}
	if d.sel >= d.offset+rows {
		d.offset = d.sel - rows + 1
	}
}

// maxNumber bounds the digits a number keeps.
const maxNumber = 6

// digit takes a typed digit: in a short list it picks its item (ok); in a
// long one it adds to the number, which highlights its item. A number the
// list does not have stays as typed, marked, and Enter picks nothing: a
// digit quietly dropped would leave Enter on another item.
func (d *menuDialog) digit(m *Model, r rune) (pick int, ok bool) {
	n := len(d.shown)
	if n < numberedAtOnce {
		if v := int(r - '0'); v >= 1 && v <= n {
			return d.shown[v-1], true
		}
		return 0, false
	}
	if len(d.number) < maxNumber {
		d.number += string(r)
	}
	d.highlightNumber(m)
	return 0, false
}

func (d *menuDialog) update(m *Model, msg tea.Msg) tea.Cmd {
	choose := func(i int) tea.Cmd {
		m.menuBox = nil
		return d.onChoose(m, i)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch key := msg.String(); key {
		case "esc", "ctrl+c":
			if d.number != "" || d.query != "" || d.filtering {
				// Esc first takes back what was typed.
				d.number, d.query, d.filtering = "", "", false
				d.filter()
				return nil
			}
			return d.cancel(m)
		case "up", "shift+tab":
			d.move(m, -1)
			return nil
		case "down", "tab":
			d.move(m, 1)
			return nil
		case "enter":
			if d.number != "" && !d.numbered() {
				return nil // no such item: Backspace or Esc correct it
			}
			if len(d.shown) > 0 {
				return choose(d.shown[d.sel])
			}
			return nil
		case "backspace":
			d.erase(m)
			return nil
		}
		return d.typed(m, msg, choose)
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			d.move(m, -1)
			return nil
		case tea.MouseButtonWheelDown:
			d.move(m, 1)
			return nil
		}
		x, y, w, h := d.geom(m)
		if msg.X < x || msg.X >= x+w || msg.Y < y || msg.Y >= y+h {
			return d.cancel(m)
		}
		row := msg.Y - y - 1 - d.header(m)
		if i := d.offset + row; msg.Button == tea.MouseButtonLeft && row >= 0 && row < d.rows(m) && i < len(d.shown) {
			return choose(d.shown[i])
		}
	}
	return nil
}

// cancel closes the list without a pick.
func (d *menuDialog) cancel(m *Model) tea.Cmd {
	m.menuBox = nil
	if d.onCancel != nil {
		return d.onCancel(m)
	}
	return nil
}

// erase takes back the last typed digit, else the query's last letter,
// else the "/" that started the filter.
func (d *menuDialog) erase(m *Model) {
	switch {
	case d.number != "":
		d.number = d.number[:len(d.number)-1]
		d.highlightNumber(m)
	case d.query != "":
		r := []rune(d.query)
		d.query = string(r[:len(r)-1])
		d.filter()
	default:
		d.filtering = false
	}
}

// typed takes any other key: digits pick or number an item, letters
// filter a search list, and a plain list keeps its letter keys (q closes,
// j and k move, space picks). In a search list digits are numbers only
// while nothing was typed in the filter: after a letter, or after "/"
// (which starts the filter), they are part of the filter.
// Every key stays in the dialog: none reaches the screen behind.
func (d *menuDialog) typed(m *Model, msg tea.KeyMsg, choose func(int) tea.Cmd) tea.Cmd {
	if msg.Type != tea.KeyRunes && msg.Type != tea.KeySpace {
		return nil
	}
	text := string(msg.Runes)
	if msg.Type == tea.KeySpace {
		text = " "
	}
	if d.search && d.query == "" && !d.filtering && text == "/" {
		d.number, d.filtering = "", true
		return nil
	}
	if !d.search || (d.query == "" && !d.filtering) {
		if digits := strings.Trim(text, "0123456789") == ""; digits {
			for _, r := range text {
				if i, ok := d.digit(m, r); ok {
					return choose(i)
				}
			}
			return nil
		}
	}
	if d.search {
		d.number, d.query = "", d.query+text
		d.filter()
		return nil
	}
	switch text {
	case "q":
		return d.cancel(m)
	case "k":
		d.move(m, -1)
	case "j":
		d.move(m, 1)
	case " ":
		if len(d.shown) > 0 {
			return choose(d.shown[d.sel])
		}
	}
	return nil
}

func (d *menuDialog) view(m *Model) (string, int, int) {
	th := m.theme
	x, y, w, _ := d.geom(m)
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	rows := []string{bg.Foreground(th.Fg).Bold(true).Render(" " + ui.Truncate(d.title, inner-2))}
	for _, l := range d.noticeLines(m) {
		rows = append(rows, bg.Foreground(th.Red).Render(" "+l))
	}
	if d.search {
		field := lipgloss.NewStyle().Background(th.Bg2)
		text := field.Foreground(th.Fg).Render(d.query + "▏")
		if d.query == "" && !d.filtering {
			text = field.Foreground(th.GreyFg).Render("Type to filter (/ for digits)…")
		}
		rows = append(rows, bg.Render(" ")+ui.FitLine(field.Foreground(th.Blue).Bold(true).Render("  ")+text, inner-2, field)+bg.Render(" "))
	}
	rows = append(rows, "")
	for r := 0; r < d.rows(m); r++ {
		pos := d.offset + r
		if pos >= len(d.shown) {
			if r == 0 {
				rows = append(rows, bg.Foreground(th.GreyFg).Render("   Nothing matches"))
				continue
			}
			rows = append(rows, "")
			continue
		}
		st := bg.Foreground(th.Fg)
		marker := "  "
		if pos == d.sel {
			st = lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Blue).Bold(true)
			marker = "▌ "
		}
		rows = append(rows, ui.FitLine(st.Render(marker+fmt.Sprintf("%d  ", pos+1)+d.items[d.shown[pos]]), inner, st))
	}
	hint := " ↑↓ Enter · Esc"
	switch {
	case d.number != "" && !d.numbered():
		hint = " No item " + d.number + " · Backspace or Esc"
	case d.number != "":
		hint = " Number: " + d.number + "_ · Enter picks it"
	case d.search:
		hint = " Type or / to filter · ↑↓ Enter · Esc"
	}
	rows = append(rows, bg.Foreground(th.GreyFg).Render(hint))
	return panel(th, rows, inner), x, y
}
