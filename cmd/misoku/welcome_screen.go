package main

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"
)

// The welcome screen shows the welcome's choices full screen, in the
// theme, before the app starts: arrows or a number pick one, Enter takes
// it, Esc (or q at the choices) leaves without changing anything. It only
// decides; creating the folder is a command, and remembering it, or
// bringing the notes, happens once the screen closed (finishWelcome).

// Smallest terminal the screen is drawn in; a smaller one, or one that
// cannot draw (TERM=dumb), gets the text menu.
const (
	screenMinWidth  = 60
	screenMinHeight = 12
)

// screenFits reports whether a terminal called term, w×h cells, shows
// the welcome screen instead of the text menu.
func screenFits(term string, w, h int) bool {
	return term != "dumb" && w >= screenMinWidth && h >= screenMinHeight
}

// screenOutcome is what the welcome screen decided: choice is one of
// choiceStart, choiceBring and choiceUseThis, or 0 when the user left.
// For choiceStart, written and abs are the folder, already created, as
// checkFolder gives them.
type screenOutcome struct {
	choice       int
	written, abs string
}

// folderMade answers makeFolderCmd: err is why the folder could not be
// created, nil when it is there.
type folderMade struct {
	written, abs string
	err          error
}

// makeFolderCmd creates the folder away from Update (it touches the
// disk).
func makeFolderCmd(written, abs string) tea.Cmd {
	return func() tea.Msg { return folderMade{written, abs, makeFolder(abs)} }
}

// welcomeScreen is the Bubble Tea model of the welcome screen.
type welcomeScreen struct {
	o      welcomeOptions
	th     theme.Theme
	w, h   int
	cursor int
	// asking: the folder field of Start my notes is open.
	asking  bool
	input   textinput.Model
	problem string
	outcome screenOutcome
}

// newWelcomeScreen is the screen for the welcome o, drawn in th.
func newWelcomeScreen(o welcomeOptions, th theme.Theme) welcomeScreen {
	in := textinput.New()
	in.Prompt = ""
	in.Cursor.SetMode(cursor.CursorStatic) // no blink ticks
	in.SetValue("~/" + defaultNotes)
	return welcomeScreen{o: o, th: th, input: in}
}

// choices are the rows offered, in order: Use this folder only when
// welcomeFor found a folder that can be used.
func (m welcomeScreen) choices() []string {
	c := []string{"Start my notes", "Bring my notes from another computer"}
	if m.o.Cwd != "" {
		c = append(c, "Use this folder: "+m.o.Cwd)
	}
	return c
}

// Init starts with nothing to do.
func (m welcomeScreen) Init() tea.Cmd { return nil }

// Update reacts to the terminal's size and keys, and to the folder made.
func (m welcomeScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case folderMade:
		if msg.err != nil {
			m.problem = msg.err.Error() + "."
			return m, nil
		}
		m.outcome = screenOutcome{choice: choiceStart, written: msg.written, abs: msg.abs}
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC {
			m.outcome = screenOutcome{}
			return m, tea.Quit
		}
		if m.asking {
			return m.updateFolder(msg)
		}
		return m.updateChoices(msg)
	}
	return m, nil
}

// updateChoices moves between the choices and takes one.
func (m welcomeScreen) updateChoices(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	last := len(m.choices()) - 1
	switch msg.String() {
	case "up", "k":
		m.cursor = max(m.cursor-1, 0)
	case "down", "j":
		m.cursor = min(m.cursor+1, last)
	case "q":
		m.outcome = screenOutcome{}
		return m, tea.Quit
	case "enter":
		return m.choose(m.cursor + 1)
	default:
		if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= last+1 {
			m.cursor = n - 1
			return m.choose(n)
		}
	}
	return m, nil
}

// choose takes the choice numbered n: Start my notes opens the folder
// field, the others close the screen.
func (m welcomeScreen) choose(n int) (tea.Model, tea.Cmd) {
	if n == choiceStart {
		m.asking = true
		m.input.CursorEnd()
		return m, m.input.Focus()
	}
	m.outcome = screenOutcome{choice: n}
	return m, tea.Quit
}

// updateFolder edits the folder field; Enter checks the folder and has
// it created.
func (m welcomeScreen) updateFolder(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		written, abs, problem := checkFolder(strings.TrimSpace(m.input.Value()), "~/"+defaultNotes, m.o.Home)
		m.problem = problem
		if problem != "" {
			return m, nil
		}
		return m, makeFolderCmd(written, abs)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View draws the welcome in a panel centred on the theme's background.
func (m welcomeScreen) View() string {
	t := m.th
	bg := lipgloss.NewStyle().Background(t.Bg)
	text := bg.Foreground(t.Fg)
	muted := bg.Foreground(t.GreyFg2)
	missing := ""
	if m.o.Missing != "" {
		missing = "I can't find your notes at " + m.o.Missing + "."
	}
	rows := make([]string, 0, 3)
	for i, c := range m.choices() {
		rows = append(rows, fmt.Sprintf("%d  %s", i+1, c))
	}
	// As wide as the longest line, so a long folder shows whole when the
	// terminal is wide enough; else it wraps.
	inner := 46
	for _, l := range append(rows, missing, m.problem, "  "+m.input.Value()) {
		inner = max(inner, lipgloss.Width(l)+2)
	}
	inner = max(min(inner, m.w-10), 20)
	lines := []string{bg.Foreground(t.Blue).Bold(true).Render("Welcome to Misoku.")}
	if missing != "" {
		lines = append(lines, bg.Foreground(t.Yellow).Width(inner).Render(missing))
	}
	lines = append(lines, "", text.Render("Where do your notes live?"), "")
	for i, row := range rows {
		mark, style := "  ", text
		if i == m.cursor {
			mark, style = "› ", bg.Foreground(t.Green).Bold(true)
		}
		lines = append(lines, style.Width(inner).Render(mark+row))
	}
	hint := "↑↓ or a number · Enter choose · Esc or q leave"
	if m.asking {
		m.input.Width = inner - 2
		lines = append(lines, "", text.Render("Folder for your notes:"),
			bg.Foreground(t.Fg).Render("  ")+m.input.View())
		if m.problem != "" {
			lines = append(lines, bg.Foreground(t.Red).Width(inner).Render(m.problem))
		}
		hint = "Enter create and open · Esc leave"
	}
	lines = append(lines, "", muted.Render(hint))
	panel := lipgloss.NewStyle().Border(ui.Border()).BorderForeground(t.Border).
		BorderBackground(t.Bg).Background(t.Bg).Padding(1, 3).Width(inner + 6).
		Render(strings.Join(lines, "\n"))
	screen := lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, panel,
		lipgloss.WithWhitespaceBackground(t.Bg))
	return ui.Fit(screen, m.w, m.h, bg)
}

// runWelcomeScreen shows the welcome screen on the terminal of o and
// then ends the welcome as the screen decided (finishWelcome).
func runWelcomeScreen(ctx context.Context, o welcomeOptions, th theme.Theme) (string, int) {
	restore := classicKeyboard(o.Out, o.ClassicKeys)
	p := tea.NewProgram(newWelcomeScreen(o, th), tea.WithAltScreen(), tea.WithInput(o.Stdin), tea.WithOutput(o.Out))
	final, err := p.Run()
	restore()
	if err != nil {
		fmt.Fprintf(o.Err, "Could not show the welcome (%v).\n", err)
		return "", 1
	}
	return finishWelcome(ctx, o, final.(welcomeScreen).outcome)
}

// finishWelcome does what the screen decided and returns the folder to
// open, or "" and the exit code, as runWelcome does: Start my notes and
// Use this folder remember the folder; Bring my notes goes on in the
// terminal with the setup assistant, as the text menu's choice does.
func finishWelcome(ctx context.Context, o welcomeOptions, out screenOutcome) (string, int) {
	switch out.choice {
	case choiceStart:
		return remember(o, out.written, out.abs), 0
	case choiceUseThis:
		return remember(o, o.Cwd, o.Cwd), 0
	case choiceBring:
		defer classicKeyboard(o.Out, o.ClassicKeys)()
		fmt.Fprintln(o.Out, "Bring my notes from another computer")
		t := &setupTerminal{in: bufio.NewReader(o.Stdin), stdin: o.Stdin, out: o.Out, err: o.Err, interrupt: o.Interrupt}
		return bringNotes(ctx, t, o)
	}
	return stopped(o.Out)
}
