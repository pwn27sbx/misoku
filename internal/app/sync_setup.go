package app

import (
	"context"
	"fmt"
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
)

// setupMsg brings a step of the assistant back to Update.
type setupMsg struct {
	s    *gitsync.Setup
	step gitsync.SetupStep
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

// setupWork runs one call of the engine in the background.
func (m *Model) setupWork(s *gitsync.Setup, call func(context.Context) gitsync.SetupStep) tea.Cmd {
	m.setupBusy = true
	m.setStatus("Setting up sync…")
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), setupStepLimit)
		defer cancel()
		return setupMsg{s: s, step: call(ctx)}
	}
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
	m.setStatus(s.StopText("Sync setup cancelled"))
	return nil
}

// showSetup asks what the step needs, in a dialog.
func (m *Model) showSetup(msg setupMsg) tea.Cmd {
	m.setupBusy = false
	if m.status == "Setting up sync…" {
		m.status, m.statusErr = "", false
	}
	s, step := msg.s, msg.step
	if step.Outcome == gitsync.SetupCancelled && len(step.Lines) > 0 {
		m.setStatus(step.Lines[len(step.Lines)-1])
		return nil
	}
	m.setupNotes = append(m.setupNotes, step.Notes...)
	if m.confirm != nil || m.saveAs != nil || m.promptBox != nil || m.menuBox != nil {
		// Another dialog came up meanwhile (it may be asking about
		// unsaved notes): it is left alone, and the assistant stops.
		if step.Ask == gitsync.SetupFinished && len(step.Lines) > 0 {
			if step.Outcome == gitsync.SetupDone {
				m.setupSync()
				m.checkWorkspace()
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
	case gitsync.AskRepoName:
		hint := "Misoku creates it private on GitHub"
		if step.Problem != "" {
			hint = step.Problem
		}
		return m.prompt(step.Title, step.Default, hint, func(m *Model, name string) tea.Cmd {
			return m.setupNext(s, gitsync.SetupAnswer{OK: true, Text: name})
		})
	default:
		m.finishSetup(step, lines)
	}
	return nil
}

// quitHeld reports whether quitting must wait: a step of the setup
// assistant is running (an init may be pushing the vault), and quitting
// now would leave it half done. The bar says why; the step ends on its own
// within its time limits, and quitting works again.
func (m *Model) quitHeld() bool {
	if !m.setupBusy {
		return false
	}
	m.setError("Sync setup is running: quit once it finishes")
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
// them in git's global settings.
func (m *Model) askIdentity(s *gitsync.Setup, step gitsync.SetupStep) tea.Cmd {
	hint := "git writes it in your notes' history"
	if step.Problem != "" {
		hint = step.Problem
	}
	return m.prompt("Your name, for git", step.Name, hint, func(m *Model, name string) tea.Cmd {
		return m.prompt("Your email, for git", step.Email, hint, func(m *Model, email string) tea.Cmd {
			lines := []string{step.Confirm, "", "Name: " + name, "Email: " + email}
			m.setupDialog(s, "Save your git identity", lines, []string{"Save", "Cancel"}, func(m *Model) tea.Cmd {
				return m.setupNext(s, gitsync.SetupAnswer{OK: true, Name: name, Email: email})
			})
			return nil
		})
	})
}

// askRepo lists the repositories to pick from: as many as fit, then
// "create" and, when some did not fit, a way to type one's name.
func (m *Model) askRepo(s *gitsync.Setup, step gitsync.SetupStep) {
	room := max(m.height-10, 3)
	shown := step.Repos
	if len(shown) > room {
		shown = shown[:room]
	}
	var items []string
	for _, r := range shown {
		label := r.Name + " · private"
		if !r.Private() {
			label = r.Name + " · " + strings.ToLower(r.Visibility) + ", not allowed"
		}
		items = append(items, label)
	}
	create := len(items)
	items = append(items, step.NewChoice)
	typed := -1
	if len(shown) < len(step.Repos) {
		typed = len(items)
		items = append(items, fmt.Sprintf("Another one of your %d repositories (type its name)…", len(step.Repos)))
	}
	m.menu(step.Title, items, func(m *Model, i int) tea.Cmd {
		switch i {
		case create:
			return m.setupNext(s, gitsync.SetupAnswer{OK: true, Choice: len(step.Repos)})
		case typed:
			return m.prompt("Repository name", "", "Such as notes or you/notes", func(m *Model, name string) tea.Cmd {
				if name == "" {
					return m.setupCancelled(s)
				}
				return m.setupNext(s, gitsync.SetupAnswer{OK: true, Text: name})
			})
		}
		return m.setupNext(s, gitsync.SetupAnswer{OK: true, Choice: i})
	})
}

// finishSetup shows how the assistant ended. A vault it set up syncs in
// this window from now on: every interval, on :sync and on quitting.
func (m *Model) finishSetup(step gitsync.SetupStep, lines []string) {
	if step.Outcome == gitsync.SetupDone {
		m.setupSync()
		m.checkWorkspace() // notes brought from the repository
		m.setStatus("Sync is set up")
	}
	m.confirm = &confirmDialog{
		title:    step.Title,
		message:  m.wrapLines(lines),
		buttons:  []string{"Close"},
		onChoose: func(*Model, int) tea.Cmd { return nil },
	}
}
