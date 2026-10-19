package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/gitsync"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"
)

// The welcome screen shows the welcome's choices full screen, in the
// theme, before the app starts: arrows or a number pick one, Enter takes
// it, Esc (or q at the choices) leaves without changing anything. Start
// my notes and Bring my notes first ask for the folder. Bring my notes
// then runs the setup assistant in the app's dialogs over the screen
// (app.SetupDialogs); leaving it comes back to the choices. Creating the
// folder and the assistant's steps are commands; remembering the folder
// happens once the screen closed (finishWelcome).

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
// For choiceStart and choiceBring, written and abs are the folder,
// already created (or brought), as checkFolder gives them.
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
	// asking: the folder field is open, for choiceStart or choiceBring
	// (askFor).
	asking  bool
	askFor  int
	input   textinput.Model
	problem string
	// bringDef is the free folder Bring my notes proposes (freeFolder).
	bringDef string
	// bring: the setup assistant is asking, for the folder bringTo.
	bring   *app.SetupDialogs
	bringTo screenOutcome
	// notice: how the last Bring my notes ended, when it came back here.
	notice  string
	outcome screenOutcome
	// exec runs gh's sign-in outside the screen; nil is tea.ExecProcess.
	exec func(*exec.Cmd, tea.ExecCallback) tea.Cmd
}

// newWelcomeScreen is the screen for the welcome o, drawn in th.
func newWelcomeScreen(o welcomeOptions, th theme.Theme) welcomeScreen {
	in := textinput.New()
	in.Prompt = ""
	in.Cursor.SetMode(cursor.CursorStatic) // no blink ticks
	return welcomeScreen{o: o, th: th, input: in, bringDef: freeFolder(o.Home, defaultNotes)}
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
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.w, m.h = size.Width, size.Height
	}
	if m.bring != nil {
		return m.updateBring(msg)
	}
	switch msg := msg.(type) {
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

// choose takes the choice numbered n: Start my notes and Bring my notes
// open the folder field, Use this folder closes the screen. How the last
// Bring my notes ended is said no more: another choice was taken.
func (m welcomeScreen) choose(n int) (tea.Model, tea.Cmd) {
	m.notice = ""
	switch n {
	case choiceStart:
		return m.askFolder(choiceStart, "~/"+defaultNotes)
	case choiceBring:
		return m.askFolder(choiceBring, m.bringDef)
	}
	m.outcome = screenOutcome{choice: n}
	return m, tea.Quit
}

// askFolder opens the folder field for choice, offering def.
func (m welcomeScreen) askFolder(choice int, def string) (tea.Model, tea.Cmd) {
	m.asking, m.askFor, m.problem = true, choice, ""
	m.input.SetValue(def)
	m.input.CursorEnd()
	return m, m.input.Focus()
}

// updateFolder edits the folder field; Enter checks the folder and has
// it created, or brings the notes into it.
func (m welcomeScreen) updateFolder(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		def := "~/" + defaultNotes
		if m.askFor == choiceBring {
			def = m.bringDef
		}
		written, abs, problem := checkFolder(strings.TrimSpace(m.input.Value()), def, m.o.Home)
		m.problem = problem
		switch {
		case problem != "":
			return m, nil
		case m.askFor == choiceBring:
			return m.startBring(written, abs)
		}
		return m, makeFolderCmd(written, abs)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// startBring runs the setup assistant in its Bring mode for the folder
// abs: it brings the notes of a repository there, in the app's dialogs.
// Its git never asks in the terminal, which the screen holds: gh gave git
// the access (gh auth setup-git).
func (m welcomeScreen) startBring(written, abs string) (tea.Model, tea.Cmd) {
	s := gitsync.NewSetup(gitsync.SetupOptions{
		Dir: abs, ConfigPath: m.o.ConfigPath, Timeout: setupTimeout, InitTimeout: initTimeout, Bring: true,
	})
	var cmd tea.Cmd
	m.bring, cmd = app.NewSetupDialogs(s, m.th, m.w, m.h, m.exec)
	m.bringTo, m.notice = screenOutcome{choice: choiceBring, written: written, abs: abs}, ""
	return m, cmd
}

// updateBring hands everything to the assistant's dialogs until it ends:
// with the notes brought the screen closes; with no notes elsewhere it
// goes on with Start my notes; else it comes back to the choices, saying
// how it ended.
func (m welcomeScreen) updateBring(msg tea.Msg) (tea.Model, tea.Cmd) {
	end, ok := msg.(app.SetupEndMsg)
	if !ok {
		return m, m.bring.Update(msg)
	}
	m.bring, m.asking = nil, false
	if n := len(end.Step.Lines); n > 0 {
		m.notice = end.Step.Lines[n-1]
	}
	switch end.Step.Outcome {
	case gitsync.SetupDone, gitsync.SetupAlready:
		m.outcome = m.bringTo
		return m, tea.Quit
	case gitsync.SetupStartNotes:
		m.cursor = choiceStart - 1
		return m.askFolder(choiceStart, "~/"+defaultNotes)
	}
	return m, nil
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
	if m.notice != "" {
		lines = append(lines, "", muted.Width(inner).Render(m.notice))
	}
	hint := "↑↓ or a number · Enter choose · Esc or q leave"
	switch {
	case m.bring != nil:
		status, isErr := m.bring.Status()
		style := muted
		if isErr {
			style = bg.Foreground(t.Red)
		}
		lines = append(lines, "", text.Render("Folder for your notes: "+m.bringTo.written))
		if status != "" {
			lines = append(lines, style.Width(inner).Render(status))
		}
		hint = "Esc cancel"
	case m.asking:
		m.input.Width = inner - 2
		lines = append(lines, "", text.Render("Folder for your notes:"),
			bg.Foreground(t.Fg).Render("  ")+m.input.View())
		if m.problem != "" {
			lines = append(lines, bg.Foreground(t.Red).Width(inner).Render(m.problem))
		}
		hint = "Enter create and open · Esc leave"
		if m.askFor == choiceBring {
			hint = "Enter bring the notes here · Esc leave"
		}
	}
	lines = append(lines, "", muted.Render(hint))
	panel := lipgloss.NewStyle().Border(ui.Border()).BorderForeground(t.Border).
		BorderBackground(t.Bg).Background(t.Bg).Padding(1, 3).Width(inner + 6).
		Render(strings.Join(lines, "\n"))
	screen := ui.Fit(lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, panel,
		lipgloss.WithWhitespaceBackground(t.Bg)), m.w, m.h, bg)
	if m.bring != nil {
		screen = m.bring.View(screen)
	}
	return screen
}

// runWelcomeScreen shows the welcome screen on the terminal of o and
// then ends the welcome as the screen decided (finishWelcome).
func runWelcomeScreen(o welcomeOptions, th theme.Theme) (string, int) {
	restore := classicKeyboard(o.Out, o.ClassicKeys)
	p := tea.NewProgram(newWelcomeScreen(o, th), tea.WithAltScreen(), tea.WithInput(o.Stdin), tea.WithOutput(o.Out))
	final, err := p.Run()
	restore()
	if err != nil {
		fmt.Fprintf(o.Err, "Could not show the welcome (%v).\n", err)
		return "", 1
	}
	return finishWelcome(o, final.(welcomeScreen).outcome)
}

// finishWelcome does what the screen decided and returns the folder to
// open, or "" and the exit code, as runWelcome does: the folder started,
// brought or used is remembered.
func finishWelcome(o welcomeOptions, out screenOutcome) (string, int) {
	switch out.choice {
	case choiceStart, choiceBring:
		return remember(o, out.written, out.abs), 0
	case choiceUseThis:
		return remember(o, o.Cwd, o.Cwd), 0
	}
	return stopped(o.Out)
}
