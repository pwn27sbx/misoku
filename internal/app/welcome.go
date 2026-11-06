package app

import (
	"fmt"
	"strings"

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

// welcomeOptions lists the answers in order; the first starts the tutorial.
// The level count comes from the tutorial's catalogue (an empty Progress
// counts its levels).
func welcomeOptions() []welcomeOption {
	_, levels := (&tutor.Progress{}).Count()
	return []welcomeOption{
		{"n", "Soy nuevo", fmt.Sprintf("Tutorial interactivo · %d niveles · ~10 min", levels)},
		{"e", "Ya tengo experiencia", "Vas directo a tus notas"},
		{"a", "Ahora no", "Te espera en «Learn» (tecla l) cuando quieras"},
	}
}

// The banner's rows, and the two cats: the full design's, drawn at the
// banner's right, and the compact variant's, by the question.
var (
	welcomeBanner  = []string{"█▀▄▀█ █ █▀ █▀█ █▄▀ █ █", "█ ▀ █ █ ▄█ █▄█ █ █ █▄█"}
	welcomeCat     = []string{` /\_/\ `, `( ^.^ )`, ` > ♥ < `}
	welcomeCatMini = []string{` /\_/\  `, `( o.o ) `, ` > ^ <  `}
)

const (
	welcomeIndent = 3 // columns before the content
	welcomeHint   = "j/k mover · enter elegir"
)

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
	return welcomeIndent + 5 + w + 1
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
		question := bg.Foreground(th.Fg).Bold(true).Render("  ¿Es tu primera vez en Misoku?")
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
			pad+muted.Render("Tus notas en Markdown, en la terminal,"),
			pad+muted.Render("al estilo de Vim."),
			"",
			pad+bg.Foreground(th.Fg).Bold(true).Render("¿Es tu primera vez aquí?"),
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
	hint := muted.Render(welcomeHint)
	left := (inner - lipgloss.Width(welcomeHint)) / 2
	rows = append(rows, "", bg.Render(strings.Repeat(" ", left))+hint)
	return panel(th, rows, inner), x, y
}

// update takes keys and clicks; it returns done=true once the dialog closed.
func (d *welcomeDialog) update(m *Model, msg tea.Msg) (tea.Cmd, bool) {
	options := welcomeOptions()
	choose := func(i int) (tea.Cmd, bool) {
		m.confirm = nil
		return m.learnChosen(i == 0), true
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
