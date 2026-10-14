package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Built-in file browser (fallback when no desktop dialog is available)

type fileBrowser struct {
	kind    mediaKind
	dir     string
	entries []os.DirEntry
	input   textinput.Model
	sel     int
	offset  int
	err     string
}

func (m *Model) openFileBrowser(kind string) tea.Cmd {
	k := mediaKinds[kind]
	start, _ := os.UserHomeDir()
	if dl := filepath.Join(start, "Downloads"); start != "" && isDir(dl) {
		start = dl
	}
	if start == "" {
		start = m.store.Root()
	}
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "Filter, or type a path and press Enter"
	in.CharLimit = 512
	fb := &fileBrowser{kind: k, input: in}
	fb.chdir(start)
	m.browser = fb
	return fb.input.Focus()
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func (fb *fileBrowser) chdir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fb.err = err.Error()
		return
	}
	fb.dir, fb.err = dir, ""
	fb.entries = fb.entries[:0]
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() || fb.kind.accepts(e.Name()) {
			fb.entries = append(fb.entries, e)
		}
	}
	sort.SliceStable(fb.entries, func(i, j int) bool {
		di, dj := fb.entries[i].IsDir(), fb.entries[j].IsDir()
		if di != dj {
			return di
		}
		return strings.ToLower(fb.entries[i].Name()) < strings.ToLower(fb.entries[j].Name())
	})
	fb.input.SetValue("")
	fb.sel, fb.offset = 0, 0
}

func (fb *fileBrowser) visible() []os.DirEntry {
	q := strings.ToLower(fb.input.Value())
	if q == "" || strings.ContainsAny(q, "/~") {
		return fb.entries
	}
	var out []os.DirEntry
	for _, e := range fb.entries {
		if strings.Contains(strings.ToLower(e.Name()), q) {
			out = append(out, e)
		}
	}
	return out
}

func browserRows(H int) int { return max(min(H-10, 18), 3) }

func (fb *fileBrowser) move(delta, rows int) {
	n := len(fb.visible())
	if n == 0 {
		return
	}
	fb.sel = (fb.sel + delta + n) % n
	if fb.sel < fb.offset {
		fb.offset = fb.sel
	}
	if fb.sel >= fb.offset+rows {
		fb.offset = fb.sel - rows + 1
	}
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// activate opens the selected folder or picks the selected file.
func (fb *fileBrowser) activate(m *Model) {
	if q := strings.TrimSpace(fb.input.Value()); strings.ContainsAny(q, "/~") {
		p := expandHome(q)
		if !filepath.IsAbs(p) {
			p = filepath.Join(fb.dir, p)
		}
		switch {
		case isDir(p):
			fb.chdir(p)
		case fb.kind.accepts(p):
			if _, err := os.Stat(p); err != nil {
				fb.err = err.Error()
				return
			}
			m.browser = nil
			m.attachMedia(fb.kind.name, p)
		default:
			fb.err = "Not a " + fb.kind.name + ": " + filepath.Base(p)
		}
		return
	}
	list := fb.visible()
	if fb.sel >= len(list) {
		return
	}
	e := list[fb.sel]
	p := filepath.Join(fb.dir, e.Name())
	if e.IsDir() {
		fb.chdir(p)
		return
	}
	m.browser = nil
	m.attachMedia(fb.kind.name, p)
}

func (fb *fileBrowser) update(m *Model, msg tea.Msg) tea.Cmd {
	rows := browserRows(m.height)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		fb.err = ""
		switch msg.String() {
		case "esc", "ctrl+c":
			m.browser = nil
			return nil
		case "enter", "right":
			fb.activate(m)
			return nil
		case "down", "ctrl+n", "ctrl+j":
			fb.move(1, rows)
			return nil
		case "up", "ctrl+p", "ctrl+k":
			fb.move(-1, rows)
			return nil
		case "left":
			fb.chdir(filepath.Dir(fb.dir))
			return nil
		case "backspace":
			if fb.input.Value() == "" {
				fb.chdir(filepath.Dir(fb.dir))
				return nil
			}
		}
		var cmd tea.Cmd
		before := fb.input.Value()
		fb.input, cmd = fb.input.Update(msg)
		if fb.input.Value() != before {
			fb.sel, fb.offset = 0, 0
		}
		return cmd

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			fb.move(-1, rows)
		case tea.MouseButtonWheelDown:
			fb.move(1, rows)
		case tea.MouseButtonLeft:
			x, y, w, h := m.browserGeom()
			if msg.X < x || msg.X >= x+w || msg.Y < y || msg.Y >= y+h {
				m.browser = nil
				return nil
			}
			if i := msg.Y - y - 1 - 4; i >= 0 && i < rows && fb.offset+i < len(fb.visible()) {
				fb.sel = fb.offset + i
				fb.activate(m)
			}
		}
		return nil

	default:
		var cmd tea.Cmd
		fb.input, cmd = fb.input.Update(msg)
		return cmd
	}
}

func (m *Model) browserGeom() (x, y, w, h int) {
	w = min(76, m.width-2)
	h = browserRows(m.height) + 7
	x, y = ui.Center(m.width, m.height, w, h)
	return
}

func (fb *fileBrowser) view(m *Model) (string, int, int) {
	th := m.theme
	x, y, w, _ := m.browserGeom()
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	field := lipgloss.NewStyle().Background(th.Bg2)
	fb.input.TextStyle = field.Foreground(th.Fg)
	fb.input.PlaceholderStyle = field.Foreground(th.GreyFg)
	fb.input.Cursor.Style = lipgloss.NewStyle().Foreground(th.Blue)
	fb.input.Cursor.TextStyle = field.Foreground(th.Fg)
	fb.input.Width = max(inner-8, 1)

	title := bg.Foreground(th.Blue).Render(" \U000f0256 ") + bg.Foreground(th.Fg).Bold(true).Render(fb.kind.title)
	where := bg.Foreground(th.GreyFg2).Render(" " + ui.Truncate(tildePath(fb.dir), inner-2))
	rows := []string{
		title,
		where,
		bg.Render(" ") + ui.FitLine(field.Foreground(th.Blue).Render("  ")+fb.input.View(), inner-2, field) + bg.Render(" "),
		bg.Foreground(th.Line).Render(strings.Repeat("─", inner)),
	}
	list := fb.visible()
	n := browserRows(m.height)
	for r := 0; r < n; r++ {
		i := fb.offset + r
		if i >= len(list) {
			if r == 0 {
				rows = append(rows, bg.Foreground(th.GreyFg).Italic(true).Render("   Nothing here"))
			} else {
				rows = append(rows, "")
			}
			continue
		}
		e := list[i]
		base := bg
		marker := "  "
		if i == fb.sel {
			base = lipgloss.NewStyle().Background(th.OneBg2)
			marker = "▌ "
		}
		icon, fg := " ", th.Fg
		if e.IsDir() {
			icon, fg = " ", th.Folder
		} else if mediaKinds["image"].accepts(e.Name()) {
			icon = "\U000f02e9 "
		}
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		rows = append(rows, base.Foreground(th.Blue).Render(marker)+base.Foreground(fg).Render(icon)+base.Foreground(fg).Bold(i == fb.sel).Render(ui.Truncate(name, inner-6)))
	}
	if fb.err != "" {
		rows = append(rows, bg.Foreground(th.Red).Render(" "+ui.Truncate(fb.err, inner-2)))
	} else {
		rows = append(rows, bg.Foreground(th.GreyFg).Render(" Enter open/choose · ← or Backspace up · ~/path jumps · Esc cancel"))
	}
	return panel(th, rows, inner), x, y
}
