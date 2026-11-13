package app

import (
	"os"
	"os/exec"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// The tutorial is a vault of its own (tutor.Dir), so it cannot open in this
// window: the home screen starts `misoku tutor` as a program of its own,
// outside Misoku's screen, and takes its progress back when it ends. The
// parent's tabs and unsaved text stay in its memory meanwhile.

// learnState is the tutorial as the home screen sees it: the routes and the
// stored progress.
type learnState struct {
	store *tutor.Store
	cat   tutor.Catalog
}

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
// so. An empty route means "not now".
type learnChosenMsg struct {
	err   error
	route string
}

// learnChosen closes the first-run dialog for good and remembers the answer;
// taking a Learn entry counts as an answer too, since the question no longer
// is one. The config is written in the command, outside the update loop, and
// the tutorial starts only after it, so a write that fails still starts what
// the answer asked for (and the dialog asks again next launch).
func (m *Model) learnChosen(route string) tea.Cmd {
	if m.learn != nil && route != "" {
		// Only a route answer falls back; "Ahora no" (empty) goes to the notes.
		route = m.learn.routeOrFirst(route)
	}
	m.learnPending = false
	m.learnBox = nil
	path := m.configPath
	prefs := m.prefs()
	return func() tea.Msg {
		if path == "" {
			return learnChosenMsg{route: route}
		}
		// Settings saves rewrite the same file: take turns with them.
		prefs.mu.Lock()
		defer prefs.mu.Unlock()
		return learnChosenMsg{err: config.SetLearnOffered(path), route: route}
	}
}

// WithLearn shows one home entry per route with lessons, with the progress
// store read before the program starts. A nil store hides them, as they stay
// hidden inside a tutorial session: sessions cannot nest.
func WithLearn(store *tutor.Store, cat tutor.Catalog) Option {
	return func(m *Model) {
		if store == nil {
			return
		}
		m.learn = &learnState{store: store, cat: cat}
		m.dashboard.SetLearn(m.learn.entries())
	}
}

// entries is the home screen's list: only routes with lessons, "Learn ·
// Básico 3/26", with a check once the route is complete.
func (l *learnState) entries() []dashboard.LearnEntry {
	if l == nil || l.store == nil {
		return nil
	}
	var out []dashboard.LearnEntry
	for _, r := range l.cat.Routes {
		if len(r.Lessons) == 0 {
			continue
		}
		done, total := l.store.RouteCount(r)
		label := i18n.T(i18n.TutorLearnEntry, r.Name, done, total)
		if done == total {
			label += " ✓"
		}
		out = append(out, dashboard.LearnEntry{Route: r.ID, Label: label})
	}
	return out
}

// firstIncomplete is the route the l key opens: the first one not complete,
// or the first route with lessons when all are.
func (l *learnState) firstIncomplete() string {
	if l == nil || l.store == nil {
		return ""
	}
	first := ""
	for _, r := range l.cat.Routes {
		if len(r.Lessons) == 0 {
			continue
		}
		if first == "" {
			first = r.ID
		}
		if done, total := l.store.RouteCount(r); done < total {
			return r.ID
		}
	}
	return first
}

// routeOrFirst keeps route when it has lessons and falls back to the first
// route otherwise ("Ya tengo experiencia" while Intermedio has no lessons).
func (l *learnState) routeOrFirst(route string) string {
	if l == nil || l.store == nil {
		return route
	}
	if r, ok := l.cat.RouteByID(route); ok && len(r.Lessons) > 0 {
		return route
	}
	return l.firstIncomplete()
}

// routeChooser starts a tutorial run on route, which may be "".
func (m *Model) startTutorial(route string) tea.Cmd {
	exe, err := os.Executable()
	if err != nil {
		m.setError(i18n.T(i18n.TutorStartFailed, err.Error()))
		return nil
	}
	args := []string{"tutor"}
	if route != "" {
		args = append(args, "--route", route)
	}
	cmd := exec.Command(exe, args...)
	// The terminal itself, not the program's wrapped output: os/exec would
	// give a pipe to a program whose stdout is not an *os.File, and a piped
	// terminal says no size, so the tutorial would never draw (see
	// syncout.Writer).
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return m.execOutside(cmd, func(err error) tea.Msg {
		return tutorialDoneMsg{err: err, learn: scanTutorial()}
	})
}

// scanTutorial reads the tutorial's progress from disk, if it has a folder.
func scanTutorial() *learnState {
	dir := tutor.Dir()
	if dir == "" {
		return nil
	}
	cat := tutor.Default()
	store, err := tutor.Load(dir)
	if err != nil {
		return nil
	}
	store.ScanDir(cat, dir, tutor.DirEnv(dir))
	return &learnState{store: store, cat: cat}
}

// reloadLearn reads the home screen's Learn entries again, off the update
// loop: after the interface language changes they belong to the other
// language's tutorial, in its own folder with its own progress (A3, S9).
// Inside a tutorial session there are none, and it does nothing.
func (m *Model) reloadLearn() tea.Cmd {
	if m.learn == nil {
		return nil
	}
	return func() tea.Msg { return tutorialDoneMsg{learn: scanTutorial()} }
}

// tutorialDoneMsg ends a tutorial run: err is what `misoku tutor` exited
// with, and learn what the tutorial holds now.
type tutorialDoneMsg struct {
	err   error
	learn *learnState
}

// finishTutorial brings the tutorial's progress to the home screen and says
// so when the run failed. Drawing the screen turns mouse motion reports back
// on, which the terminal lost while the tutorial held it.
func (m *Model) finishTutorial(msg tutorialDoneMsg) {
	if msg.learn != nil {
		m.learn = msg.learn
		m.dashboard.SetLearn(msg.learn.entries())
	}
	if msg.err != nil {
		m.setError(i18n.T(i18n.TutorFailed, msg.err.Error()))
	}
}
