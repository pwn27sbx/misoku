package app

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/tutor"
	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// welcomeDialog is the first-run dialog: a banner, a cat and a vertical menu
// of three answers. It opens on its own, so only j/k, the arrows, Enter, the
// options' own letters and Esc act on it: a stray key never answers. On a
// terminal too short for the banner it draws a compact variant.
type welcomeDialog struct {
	selected int
}

// welcomeOption is one answer: the letter that picks it, its title and the
// muted line under it.
type welcomeOption struct{ key, title, desc string }

// welcomeOptions lists the answers in order, in the interface language; the
// first two start the tutorial on a route, and the last goes straight to the
// notes. The letters stay the same in every language.
func welcomeOptions() []welcomeOption {
	cat := tutor.Default()
	b, _ := cat.RouteByID("B")
	i, ok := cat.RouteByID("I")
	if !ok {
		i = b // routeOrFirst sends the answer to the first route then
	}
	lessons := len(b.Lessons)
	return []welcomeOption{
		{"n", i18n.T(i18n.WelcomeNew), i18n.N(i18n.WelcomeNewRoute, lessons, b.Name, lessons)},
		{"e", i18n.T(i18n.WelcomeExperienced), i18n.T(i18n.WelcomeExperiencedRoute, i.Name)},
		{"a", i18n.T(i18n.WelcomeLater), i18n.T(i18n.WelcomeLaterDesc)},
	}
}

// The banner's rows, and the two cats: the full design's, drawn at the
// banner's right, and the compact variant's, by the question.
var (
	welcomeBanner  = []string{"█▀▄▀█ █ █▀ █▀█ █▄▀ █ █", "█ ▀ █ █ ▄█ █▄█ █ █ █▄█"}
	welcomeCat     = []string{` /\_/\ `, `( ^.^ )`, ` > ♥ < `}
	welcomeCatMini = []string{` /\_/\  `, `( o.o ) `, ` > ^ <  `}
)

const welcomeIndent = 3 // columns before the content

// welcomeFullRows and welcomeCompactRows are the inner heights of the two
// variants (see rows).
const (
	welcomeFullRows    = 17
	welcomeCompactRows = 12
)

// innerWidth is the box's inner width: room for the widest option line,
// with one column to spare on the right.
func (d *welcomeDialog) innerWidth() int {
	w := 0
	for _, o := range welcomeOptions() {
		w = max(w, lipgloss.Width(o.title), lipgloss.Width(o.desc))
	}
	w = welcomeIndent + 5 + w + 1
	// The full banner and its cat share one row: keep room for both.
	banner := welcomeIndent + lipgloss.Width(welcomeCat[0]) + 4 + lipgloss.Width(welcomeBanner[0])
	tagline := welcomeIndent + max(lipgloss.Width(i18n.T(i18n.WelcomeTagline)), lipgloss.Width(i18n.T(i18n.WelcomeTaglineVim))) + 1
	// The question, the compact variant's question beside its cat, and the
	// hint: translations differ in length, so each one is measured.
	question := welcomeIndent + max(lipgloss.Width(i18n.T(i18n.WelcomeQuestion))+1,
		lipgloss.Width(welcomeCatMini[1])+2+lipgloss.Width(i18n.T(i18n.WelcomeQuestionCompact)),
		lipgloss.Width(i18n.T(i18n.WelcomeHint)))
	return max(w, banner, tagline, question)
}

// compact reports whether a W x H terminal takes the compact variant.
func (d *welcomeDialog) compact(H int) bool { return H < welcomeFullRows+2 }

// fits reports whether either variant fits whole in a W x H terminal.
func (d *welcomeDialog) fits(W, H int) bool {
	return d.innerWidth()+2 <= W && welcomeCompactRows+2 <= H
}

// geom places the box: its top-left corner and size, borders included.
func (d *welcomeDialog) geom(W, H int) (x, y, w, h int) {
	w, h = d.innerWidth()+2, welcomeFullRows+2
	if d.compact(H) {
		h = welcomeCompactRows + 2
	}
	x, y = ui.Center(W, H, w, h)
	return x, y, w, h
}

// optionTop is the inner row of the first option's title; each option
// takes two rows.
func (d *welcomeDialog) optionTop(H int) int {
	if d.compact(H) {
		return 4
	}
	return 9
}

// view draws the dialog centred in a W x H terminal.
func (d *welcomeDialog) view(th theme.Theme, W, H int) (string, int, int) {
	x, y, w, _ := d.geom(W, H)
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.PopupBg)
	pad := bg.Render(strings.Repeat(" ", welcomeIndent))
	muted := bg.Foreground(th.GreyFg2)

	var rows []string
	if d.compact(H) {
		cat := bg.Foreground(th.Purple)
		question := bg.Foreground(th.Fg).Bold(true).Render("  " + i18n.T(i18n.WelcomeQuestionCompact))
		for i, l := range welcomeCatMini {
			row := pad + cat.Render(l)
			if i == 1 {
				row += question
			}
			rows = append(rows, row)
		}
		rows = append(rows, "")
	} else {
		// The cat ends four columns short of the border.
		catCol := inner - welcomeIndent - lipgloss.Width(welcomeCat[0]) - 4
		cat := bg.Foreground(th.Purple)
		banner := bg.Foreground(th.Blue).Bold(true)
		for i, l := range welcomeCat {
			left := ""
			if i > 0 {
				left = banner.Render(welcomeBanner[i-1])
			}
			rows = append(rows, pad+ui.FitLine(left, catCol, bg)+cat.Render(l))
		}
		rows = append(rows,
			pad+muted.Render(strings.Repeat("─", inner-2*welcomeIndent-1)),
			pad+muted.Render(i18n.T(i18n.WelcomeTagline)),
			pad+muted.Render(i18n.T(i18n.WelcomeTaglineVim)),
			"",
			pad+bg.Foreground(th.Fg).Bold(true).Render(i18n.T(i18n.WelcomeQuestion)),
			"",
		)
	}

	sel := lipgloss.NewStyle().Background(th.OneBg2)
	band := inner - welcomeIndent - 1 // the highlight's width: one column short of the border
	for i, o := range welcomeOptions() {
		st, marker, title := bg, "  ", bg.Foreground(th.Fg)
		if i == d.selected {
			st, marker, title = sel, "▸ ", sel.Foreground(th.Fg).Bold(true)
		}
		key := st.Foreground(th.Blue).Bold(true).Render(marker + o.key)
		head := key + st.Render("  ") + title.Render(o.title)
		desc := st.Render(strings.Repeat(" ", 5)) + st.Foreground(th.GreyFg2).Render(o.desc)
		rows = append(rows, pad+ui.FitLine(head, band, st), pad+ui.FitLine(desc, band, st))
	}
	hintText := i18n.T(i18n.WelcomeHint)
	hint := muted.Render(hintText)
	left := (inner - lipgloss.Width(hintText)) / 2
	rows = append(rows, "", bg.Render(strings.Repeat(" ", left))+hint)
	return panel(th, rows, inner), x, y
}

// update takes keys and clicks; it returns done=true once the dialog closed.
func (d *welcomeDialog) update(m *Model, msg tea.Msg) (tea.Cmd, bool) {
	options := welcomeOptions()
	choose := func(i int) (tea.Cmd, bool) {
		route := ""
		switch i {
		case 0:
			route = "B"
		case 1:
			route = "I"
		}
		m.confirm = nil
		return m.learnChosen(route), true
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch key := msg.String(); key {
		case "j", "down":
			d.selected = min(d.selected+1, len(options)-1)
		case "k", "up":
			d.selected = max(d.selected-1, 0)
		case "enter":
			return choose(d.selected)
		case "esc":
			return choose(len(options) - 1)
		case "ctrl+c":
			if m.quitHeld() {
				return nil, false
			}
			return tea.Quit, true
		default:
			for i, o := range options {
				if key == o.key {
					return choose(i)
				}
			}
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
			return nil, false
		}
		x, y, w, _ := d.geom(m.width, m.height)
		if msg.X <= x || msg.X >= x+w-1 {
			return nil, false
		}
		if i := (msg.Y - y - 1 - d.optionTop(m.height)); i >= 0 && i < 2*len(options) {
			return choose(i / 2)
		}
	}
	return nil, false
}
