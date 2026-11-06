package app

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// The tutorial is a vault of its own (tutor.Dir), so it cannot open in this
// window: the home screen starts `misoku tutor` as a program of its own,
// outside Misoku's screen, and takes its progress back when it ends. The
// parent's tabs and unsaved text stay in its memory meanwhile.

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
	return m.execOutside(exec.Command(exe, "tutor"), func(err error) tea.Msg {
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
