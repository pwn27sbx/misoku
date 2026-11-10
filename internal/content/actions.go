package content

import (
	"strings"
	"sync"

	"github.com/pwn27sbx/misoku/internal/core"
)

// Action is one activity the player confirmed in the editor, reported while
// the tutorial runs (F3b/F3c): the event's name and its value, "" when the
// event takes none.
type Action struct {
	Name  string
	Value string
}

// actionLog collects the actions of an editor and of every copy of it, so the
// app drains them after each Update (TakeActions) and credits them to the
// current lesson. A pointer, so a copy made by Update shares it.
type actionLog struct {
	mu   sync.Mutex
	on   bool
	list []Action
}

func (l *actionLog) setOn(on bool) {
	l.mu.Lock()
	l.on = on
	l.mu.Unlock()
}

func (l *actionLog) reporting() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.on
}

func (l *actionLog) append(a Action) {
	l.mu.Lock()
	l.list = append(l.list, a)
	l.mu.Unlock()
}

func (l *actionLog) take() []Action {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.list) == 0 {
		return nil
	}
	out := l.list
	l.list = nil
	return out
}

// SetReportActions turns the editor's action reports on or off. The app
// leaves them off, which costs one comparison per reported action, and turns
// them on while the tutorial runs (see internal/app's tutorEvent).
func (m *Model) SetReportActions(on bool) {
	if m.actions == nil {
		return
	}
	m.actions.setOn(on)
}

// TakeActions returns the actions recorded since the last call and forgets
// them: the app calls it after every Update.
func (m Model) TakeActions() []Action {
	if m.actions == nil {
		return nil
	}
	return m.actions.take()
}

// reportAction reports one action the player confirmed; it does nothing unless
// SetReportActions turned the reports on. Call it where the action is done,
// never in View.
func (m Model) reportAction(name, value string) {
	if m.actions == nil || !m.actions.reporting() {
		return
	}
	m.actions.append(Action{Name: name, Value: value})
}

// reportCommandError reports the command line error out names ("E488:
// Trailing characters…"), with the error's code as its value (GRF-07).
// Anything else a command returned is not a command line error.
func (m Model) reportCommandError(out any) {
	msg, ok := out.(core.StatusMsg)
	if !ok || !msg.Error {
		return
	}
	code := msg.Text
	if i := strings.IndexByte(code, ':'); i >= 0 {
		code = code[:i]
	}
	if len(code) < 2 || code[0] != 'E' {
		return
	}
	for _, r := range code[1:] {
		if r < '0' || r > '9' {
			return
		}
	}
	m.reportAction("command_error", code)
}
