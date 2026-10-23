package app

import tea "github.com/charmbracelet/bubbletea"

// maxMessages is how many of the bar's messages `:messages` keeps.
const maxMessages = 100

// sentMessage is one message the bar showed, kept for `:messages`.
type sentMessage struct {
	at   string // HH:MM, read from the app's clock
	text string
}

// say shows a message in the bar and keeps it for `:messages`. An error
// stays until Esc or the next message; any other message clears on the
// next key.
func (m *Model) say(text string, isErr bool) {
	m.status, m.statusErr = text, isErr
	if text == "" {
		return
	}
	if len(m.messages) == maxMessages {
		m.messages = append(m.messages[:0], m.messages[1:]...)
	}
	m.messages = append(m.messages, sentMessage{at: m.clock().Format("15:04"), text: text})
}

// showProgress shows text in the bar without keeping it for `:messages`:
// progress such as "Syncing…" is replaced by its result, and only results
// are worth listing.
func (m *Model) showProgress(text string) { m.status, m.statusErr = text, false }

// unsayLast takes the newest message out of `:messages`, for one that is
// about to be said again as part of a longer one.
func (m *Model) unsayLast() {
	if n := len(m.messages); n > 0 {
		m.messages = m.messages[:n-1]
	}
}

// clearStatus empties the bar's message; `:messages` still has it.
func (m *Model) clearStatus() { m.status, m.statusErr = "", false }

// openMessages lists the bar's last messages, oldest first, with the
// newest highlighted (`:messages`). The list scrolls; Esc closes it.
func (m *Model) openMessages() {
	if len(m.messages) == 0 {
		m.setStatus("No messages yet")
		return
	}
	items := make([]string, len(m.messages))
	for i, s := range m.messages {
		items[i] = s.at + "  " + s.text
	}
	m.menu("Messages", items, func(*Model, int) tea.Cmd { return nil })
	d := m.menuBox
	d.sel = len(d.shown) - 1
	d.scroll(m)
}
