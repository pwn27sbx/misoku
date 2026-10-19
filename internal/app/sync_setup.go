package app

import (
	"context"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"

	tea "github.com/charmbracelet/bubbletea"
)

// The setup assistant inside the app (`:sync setup`): the same engine as
// `misoku sync setup` (gitsync.Setup), asked in dialogs. Each call of the
// engine runs git and gh, so it runs in a tea.Cmd; gh's sign-in needs the
// terminal, so the screen is set aside while it runs (tea.ExecProcess).

const (
	// setupTimeout bounds each gh command and local git check.
	setupTimeout = time.Minute
	// setupGitTimeout bounds each git command of the init, which pushes
	// or brings the whole vault the first time.
	setupGitTimeout = 5 * time.Minute
	// setupStepLimit bounds one call of the engine.
	setupStepLimit = 15 * time.Minute
	// setupRunning is the bar while a step runs.
	setupRunning = "Setting up sync… Esc cancels"
	// setupCancelling is the bar once Esc stopped a step, until it returns.
	setupCancelling = "Cancelling sync setup…"
)

// setupMsg brings a step of the assistant back to Update; id is the
// step's number in its Model (setupStep).
type setupMsg struct {
	s    *gitsync.Setup
	step gitsync.SetupStep
	id   int
}

// setupSignedInMsg says gh's sign-in, run outside the screen, ended.
type setupSignedInMsg struct{ s *gitsync.Setup }

// startSetup starts the assistant for the open vault.
func (m *Model) startSetup() tea.Cmd {
	if m.setupBusy {
		m.setStatus("Sync setup is already running…")
		return nil
	}
	if m.configPath == "" {
		m.setError("Sync setup needs Misoku's config file, and there is none here")
		return nil
	}
	m.setupNotes = nil
	s := gitsync.NewSetup(gitsync.SetupOptions{
		Dir: m.store.Root(), ConfigPath: m.configPath,
		Timeout: setupTimeout, InitTimeout: setupGitTimeout,
	})
	return m.setupWork(s, s.Begin)
}

// setupWork runs one call of the engine in the background. Esc cancels
// it (cancelSetupStep): its git and gh are stopped.
func (m *Model) setupWork(s *gitsync.Setup, call func(context.Context) gitsync.SetupStep) tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), setupStepLimit)
	m.setupStep++
	id := m.setupStep
	m.setupBusy, m.setupRun, m.setupStop, m.setupStopped = true, s, cancel, false
	m.setStatus(setupRunning)
	return func() tea.Msg {
		defer cancel()
		return setupMsg{s: s, step: call(ctx), id: id}
	}
}

// cancelSetupStep is Esc while a step runs: it stops the step's git and
// gh. The step then returns, and its result ends the assistant as a cancel
// does (showSetup), removing what the run created.
func (m *Model) cancelSetupStep() tea.Cmd {
	if !m.setupBusy || m.setupStopped {
		return nil
	}
	m.setupStopped = true
	m.setupStop()
	m.setStatus(setupCancelling)
	return nil
}

// setupBar is the bar while a step runs: it runs, or Esc is stopping it.
func (m *Model) setupBar() string {
	if m.setupStopped {
		return setupCancelling
	}
	return setupRunning
}

// setupNext answers the current step.
func (m *Model) setupNext(s *gitsync.Setup, ans gitsync.SetupAnswer) tea.Cmd {
	return m.setupWork(s, func(ctx context.Context) gitsync.SetupStep { return s.Next(ctx, ans) })
}

// setupCancelled ends the assistant s from one of its dialogs, saying
// what it changed before, if anything. A repository the run created goes
// through the engine, in the background: it runs git to see whether the
// repository can be removed again.
func (m *Model) setupCancelled(s *gitsync.Setup) tea.Cmd {
	if s.MadeRepo() {
		return m.setupNext(s, gitsync.SetupAnswer{})
	}
	text := s.StopText("Sync setup cancelled")
	m.setStatus(text)
	return m.setupEnded(gitsync.SetupStep{Outcome: gitsync.SetupCancelled, Lines: []string{text}})
}

// setupEnded tells the screen hosting the assistant that it ended with
// step (SetupDialogs); the app itself needs no telling.
func (m *Model) setupEnded(step gitsync.SetupStep) tea.Cmd {
	if m.onSetupEnd == nil {
		return nil
	}
	return m.onSetupEnd(step)
}

// setupCancel is the cancel of a dialog of the assistant s.
func setupCancel(s *gitsync.Setup) func(*Model) tea.Cmd {
	return func(m *Model) tea.Cmd { return m.setupCancelled(s) }
}

// setupPrompt asks for a value of the assistant s, as prompt does;
// leaving it cancels the assistant.
func (m *Model) setupPrompt(s *gitsync.Setup, title, value, hint string, onSubmit func(*Model, string) tea.Cmd) tea.Cmd {
	cmd := m.prompt(title, value, hint, onSubmit)
	m.promptBox.onCancel = setupCancel(s)
	return cmd
}

// showSetup asks what the step needs, in a dialog.
func (m *Model) showSetup(msg setupMsg) tea.Cmd {
	if msg.s != m.setupRun || msg.id != m.setupStep || !m.setupBusy {
		return nil // stale: not the step running
	}
	m.setupBusy, m.setupStop = false, nil
	if m.status == setupRunning {
		m.status, m.statusErr = "", false
	}
	s, step := msg.s, msg.step
	if m.setupStopped && step.Outcome != gitsync.SetupDone {
		// Esc stopped this step: whatever it returned, the run ends as
		// cancelled. One that finished setting up is said as such.
		m.setupStopped = false
		return m.setupCancelled(s)
	}
	m.setupStopped = false
	if step.Outcome == gitsync.SetupCancelled && len(step.Lines) > 0 {
		m.setStatus(step.Lines[len(step.Lines)-1])
		return m.setupEnded(step)
	}
	m.setupNotes = append(m.setupNotes, step.Notes...)
	if m.confirm != nil || m.saveAs != nil || m.promptBox != nil || m.menuBox != nil {
		// Another dialog came up meanwhile (it may be asking about
		// unsaved notes): it is left alone, and the assistant stops.
		if step.Ask == gitsync.SetupFinished && len(step.Lines) > 0 {
			if step.Outcome == gitsync.SetupDone {
				m.vaultSetUp()
			}
			m.setStatus(step.Lines[len(step.Lines)-1])
			return nil
		}
		m.setStatus(s.StopText("Sync setup stopped: another dialog is open"))
		return nil
	}
	if step.Problem != "" {
		m.setError(step.Problem)
	}
	lines := append(append([]string{}, m.setupNotes...), step.Lines...)
	switch step.Ask {
	case gitsync.AskOwnRepo:
		m.setupDialog(s, step.Title, append(lines, "", step.Confirm), []string{"Create", "Cancel"}, func(m *Model) tea.Cmd {
			return m.setupNext(s, gitsync.SetupAnswer{OK: true})
		})
	case gitsync.AskStart:
		m.setupDialog(s, step.Title, lines, []string{"Start", "Cancel"}, func(m *Model) tea.Cmd {
			return m.setupNext(s, gitsync.SetupAnswer{OK: true})
		})
	case gitsync.AskIdentity:
		return m.askIdentity(s, step)
	case gitsync.AskSignIn:
		m.setupDialog(s, step.Title, lines, []string{"Sign in", "Cancel"}, func(m *Model) tea.Cmd {
			return m.execProcess(s.SignInCommand(), func(error) tea.Msg {
				// gh said how it went on the terminal; Next asks gh
				// whether it is signed in now.
				return setupSignedInMsg{s: s}
			})
		})
	case gitsync.AskRepo:
		m.askRepo(s, step)
	case gitsync.AskUnrelated:
		m.setupDialog(s, step.Title, append(lines, "", step.Confirm), []string{"Mix them", "Cancel"}, func(m *Model) tea.Cmd {
			return m.setupNext(s, gitsync.SetupAnswer{OK: true})
		})
		// Cancel is the default, and no letter picks: mixing two sets of
		// notes takes a deliberate choice.
		m.confirm.selected, m.confirm.deliberate = 1, true
	case gitsync.AskRepoName:
		hint := "Misoku creates it private on GitHub"
		if step.Problem != "" {
			hint = step.Problem
		}
		return m.setupPrompt(s, step.Title, step.Default, hint, func(m *Model, name string) tea.Cmd {
			return m.setupNext(s, gitsync.SetupAnswer{OK: true, Text: name})
		})
	default:
		return m.finishSetup(step, lines)
	}
	return nil
}

// quitHeld reports whether quitting must wait: a step of the setup
// assistant is running (an init may be pushing the vault), and quitting
// now would leave it half done. The bar says why and that Esc stops it;
// the step also ends on its own within its time limits, and quitting works
// again.
func (m *Model) quitHeld() bool {
	if !m.setupBusy {
		return false
	}
	m.setError("Sync setup is running: Esc cancels it, then quit")
	return true
}

// setupDialog shows lines of the assistant s with the buttons; the first
// goes on, the last cancels.
func (m *Model) setupDialog(s *gitsync.Setup, title string, lines, buttons []string, onGo func(*Model) tea.Cmd) {
	m.confirm = &confirmDialog{
		title:   title,
		message: m.wrapLines(lines),
		buttons: buttons,
		onChoose: func(m *Model, choice int) tea.Cmd {
			if choice != 0 {
				return m.setupCancelled(s)
			}
			return onGo(m)
		},
	}
}

// askIdentity asks for the name, then the email, then whether to save
// them in git's global settings. Values filled in from the GitHub account
// come already typed: Enter takes them, and they can be edited.
func (m *Model) askIdentity(s *gitsync.Setup, step gitsync.SetupStep) tea.Cmd {
	hint := "git writes it in your notes' history"
	if step.FromGitHub {
		hint = "From your GitHub account; change it if you like"
	}
	if step.Problem != "" {
		hint = step.Problem
	}
	return m.setupPrompt(s, "Your name, for git", step.Name, hint, func(m *Model, name string) tea.Cmd {
		return m.setupPrompt(s, "Your email, for git", step.Email, hint, func(m *Model, email string) tea.Cmd {
			lines := []string{step.Confirm, "", "Name: " + name, "Email: " + email}
			m.setupDialog(s, "Save your git identity", lines, []string{"Save", "Cancel"}, func(m *Model) tea.Cmd {
				return m.setupNext(s, gitsync.SetupAnswer{OK: true, Name: name, Email: email})
			})
			return nil
		})
	})
}

// askRepo lists every repository to pick from, with the new private
// repository pinned first (step.NewFirst) where a long list cannot hide
// it. Typing filters the list ("/" first to filter by digits); in a long
// one a number is picked with Enter. Why the last pick was not taken shows
// above the list.
func (m *Model) askRepo(s *gitsync.Setup, step gitsync.SetupStep) {
	items := make([]string, 0, len(step.Repos)+1)
	for _, r := range step.Repos {
		label := r.Name + " · private"
		if !r.Private() {
			label = r.Name + " · " + strings.ToLower(r.Visibility) + ", not allowed"
		}
		items = append(items, label)
	}
	pinned, create := 0, len(items)
	if step.NewFirst {
		pinned, create = 1, 0
		items = append([]string{step.NewChoice}, items...)
	} else {
		items = append(items, step.NewChoice)
	}
	m.searchMenu(step.Title, items, pinned, func(m *Model, i int) tea.Cmd {
		choice := i - pinned // an index in step.Repos
		if i == create {
			choice = len(step.Repos)
		}
		return m.setupNext(s, gitsync.SetupAnswer{OK: true, Choice: choice})
	})
	// Why the last pick was not taken, whole: the bar cuts it off.
	m.menuBox.notice = step.Problem
	m.menuBox.onCancel = setupCancel(s)
}

// finishSetup shows how the assistant ended; closing it tells the screen
// hosting the assistant, if any. Starting new notes instead (Bring) is
// that screen's to show.
func (m *Model) finishSetup(step gitsync.SetupStep, lines []string) tea.Cmd {
	if step.Outcome == gitsync.SetupStartNotes {
		return m.setupEnded(step)
	}
	if step.Outcome == gitsync.SetupDone {
		m.vaultSetUp()
		m.setStatus("Sync is set up")
	}
	m.confirm = &confirmDialog{
		title:    step.Title,
		message:  m.wrapLines(lines),
		buttons:  []string{"Close"},
		onChoose: func(m *Model, _ int) tea.Cmd { return m.setupEnded(step) },
	}
	return nil
}

// vaultSetUp makes the vault the assistant set up sync in this window from
// now on: every interval, on :sync and on quitting. A screen that is not
// the app (no vault open) has nothing to do.
func (m *Model) vaultSetUp() {
	if m.store == nil {
		return
	}
	m.setupSync()
	m.checkWorkspace() // notes brought from the repository
}
