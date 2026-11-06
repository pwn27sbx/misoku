package app

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// The tutorial is a vault of its own (tutor.Dir), so it cannot open in this
// window: the home screen starts `misoku tutor` as a program of its own,
// outside Misoku's screen, and takes its progress back when it ends. The
// parent's tabs and unsaved text stay in its memory meanwhile.

// WithLearnOffer shows the first-run dialog once the home screen opens: the
// answer is remembered, so it never shows again. It is opt-in, so test
// harnesses and every other way of opening Misoku never see it.
func WithLearnOffer(offered bool) Option {
	return func(m *Model) { m.learnPending = offered }
}

// offerLearn shows the first-run dialog when it still waits to be answered,
// on the home screen, with nothing else open, and a terminal it fits. Below
// 60x12 the frame says the terminal is too small while a dialog would still
// take keys, so a blind Enter can never answer it: that, and a resize that
// stops it fitting, close it and keep it pending.
func (m *Model) offerLearn() {
	if m.learnBox != nil && m.confirm == m.learnBox {
		if !m.learnBox.fits(m.width, m.height) {
			m.confirm = nil // too small to read: it waits for a bigger terminal
		}
		return
	}
	if !m.learnPending || m.viewMode != viewModeDashboard || m.modalOpen() {
		return
	}
	if m.width < MinTermWidth || m.height < MinTermHeight {
		return
	}
	d := &confirmDialog{welcome: &welcomeDialog{}}
	if !d.fits(m.width, m.height) {
		return
	}
	m.learnBox = d
	m.confirm = d
}

// learnChosenMsg is the answer to the first-run dialog, once the config says
// so.
type learnChosenMsg struct {
	err      error
	tutorial bool
}

// learnChosen closes the first-run dialog for good and remembers the answer;
// taking the Learn entry counts as an answer too, since the question no
// longer is one. The config is written in the command, outside the update
// loop, and the tutorial starts only after it, so a write that fails still
// starts what the answer asked for (and the dialog asks again next launch).
func (m *Model) learnChosen(tutorial bool) tea.Cmd {
	m.learnPending = false
	m.learnBox = nil
	path := m.configPath
	prefs := m.prefs()
	return func() tea.Msg {
		if path == "" {
			return learnChosenMsg{tutorial: tutorial}
		}
		// Settings saves rewrite the same file: take turns with them.
		prefs.mu.Lock()
		defer prefs.mu.Unlock()
		return learnChosenMsg{err: config.SetLearnOffered(path), tutorial: tutorial}
	}
}

// WithLearn shows the "Learn k/n" entry on the home screen for the tutorial
// progress p, which main reads from disk before the program starts. A nil p
// hides it, as it stays hidden inside a tutorial session: sessions cannot
// nest.
func WithLearn(p *tutor.Progress) Option {
	return func(m *Model) {
		m.learn = p
		m.dashboard.SetLearn(learnLabel(p))
	}
}

// learnLabel is the home entry's label: the levels passed, and a check once
// every level does. "" hides the entry.
func learnLabel(p *tutor.Progress) string {
	if p == nil {
		return ""
	}
	done, total := p.Count()
	switch {
	case total == 0:
		return "Learn"
	case done == total:
		return fmt.Sprintf("Learn %d/%d ✓", done, total)
	}
	return fmt.Sprintf("Learn %d/%d", done, total)
}

// startTutorial runs `misoku tutor` in place of Misoku's screen, and reads
// the tutorial's progress again when it ends, in the callback (which runs
// outside the update loop, see tutor.Scan). A session-only --theme is not
// passed on: the tutorial opens with the saved theme.
func (m *Model) startTutorial() tea.Cmd {
	exe, err := os.Executable()
	if err != nil {
		m.setError("Could not start the tutorial: " + err.Error())
		return nil
	}
	cmd := exec.Command(exe, "tutor")
	// The terminal itself, not the program's wrapped output: os/exec would
	// give a pipe to a program whose stdout is not an *os.File, and a piped
	// terminal says no size, so the tutorial would never draw (see
	// syncout.Writer).
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return m.execOutside(cmd, func(err error) tea.Msg {
		return tutorialDoneMsg{err: err, progress: scanTutorial()}
	})
}

// scanTutorial reads the tutorial's progress from disk, if it has a folder.
func scanTutorial() *tutor.Progress {
	if dir := tutor.Dir(); dir != "" {
		return tutor.Scan(dir)
	}
	return nil
}

// tutorialDoneMsg ends a tutorial run: err is what `misoku tutor` exited
// with, and progress what its notes hold now.
type tutorialDoneMsg struct {
	err      error
	progress *tutor.Progress
}

// finishTutorial brings the tutorial's progress to the home screen and says
// so when the run failed. Drawing the screen turns mouse motion reports back
// on, which the terminal lost while the tutorial held it.
func (m *Model) finishTutorial(msg tutorialDoneMsg) {
	if msg.progress != nil {
		m.learn = msg.progress
		m.dashboard.SetLearn(learnLabel(msg.progress))
	}
	if msg.err != nil {
		m.setError("Tutorial: " + msg.err.Error())
	}
}
