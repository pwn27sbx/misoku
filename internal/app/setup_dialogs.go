package app

import (
	"os/exec"

	"github.com/pwn27sbx/misoku/internal/gitsync"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// SetupDialogs is the setup assistant of `:sync setup`, asked in the same
// dialogs on a screen that is not the app: the welcome's "Bring my notes
// from another computer". It is the app's own code, on a Model with no
// vault open, so both screens ask the same questions the same way.
type SetupDialogs struct{ m *Model }

// SetupEndMsg says the assistant of a SetupDialogs ended; Step is its last
// step, and its Outcome how it ended (SetupStartNotes: the user has no
// notes elsewhere). Everything the run created is removed again unless it
// ended with SetupDone or SetupAlready.
type SetupEndMsg struct{ Step gitsync.SetupStep }

// SetupExec runs a program outside the screen, as tea.ExecProcess does.
type SetupExec func(*exec.Cmd, tea.ExecCallback) tea.Cmd

// NewSetupDialogs prepares the dialogs of the assistant s, drawn in th on
// a screen of w×h cells; exec runs gh's sign-in (nil: tea.ExecProcess).
// The command returned starts the assistant, in the background.
func NewSetupDialogs(s *gitsync.Setup, th theme.Theme, w, h int, exec SetupExec) (*SetupDialogs, tea.Cmd) {
	if exec == nil {
		exec = tea.ExecProcess
	}
	m := &Model{theme: th, width: w, height: h, execProcess: exec}
	m.onSetupEnd = func(step gitsync.SetupStep) tea.Cmd {
		return func() tea.Msg { return SetupEndMsg{Step: step} }
	}
	return &SetupDialogs{m: m}, m.setupWork(s, s.Begin)
}

// Update takes the screen's size, the assistant's steps, and the keys and
// clicks for its open dialog. Ctrl+C leaves a dialog as Esc does, so the
// run removes what it created (in the app, Ctrl+C quits).
func (d *SetupDialogs) Update(msg tea.Msg) tea.Cmd {
	m := d.m
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return nil
	case setupMsg:
		return m.showSetup(msg)
	case setupSignedInMsg:
		return m.setupNext(msg.s, gitsync.SetupAnswer{OK: true})
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		}
		return d.dialog(msg)
	}
	return d.dialog(msg)
}

// dialog hands msg to the open dialog: keys and clicks to any, anything
// else (a cursor blink) to the text field only.
func (d *SetupDialogs) dialog(msg tea.Msg) tea.Cmd {
	m := d.m
	_, input := msg.(tea.KeyMsg)
	if _, click := msg.(tea.MouseMsg); click {
		input = true
	}
	switch {
	case m.promptBox != nil:
		return m.promptBox.update(m, msg)
	case !input:
		return nil
	case m.confirm != nil:
		cmd, _ := m.confirm.update(m, msg)
		return cmd
	case m.menuBox != nil:
		return m.menuBox.update(m, msg)
	}
	return nil // a step runs: there is nothing to answer yet
}

// View draws the open dialog over behind, the screen as wide and tall as
// the terminal; with none open, behind as it is.
func (d *SetupDialogs) View(behind string) string {
	m := d.m
	var box string
	var x, y int
	switch {
	case m.confirm != nil:
		box, x, y = m.confirm.view(m.theme, m.width, m.height)
	case m.promptBox != nil:
		box, x, y = m.promptBox.view(m)
	case m.menuBox != nil:
		box, x, y = m.menuBox.view(m)
	default:
		return behind
	}
	return ui.Overlay(behind, box, x, y)
}

// Status is what the assistant says outside its dialogs, as the app's bar
// would: that a step runs, or why the last answer was not taken
// (isError).
func (d *SetupDialogs) Status() (text string, isError bool) {
	return d.m.status, d.m.statusErr
}
