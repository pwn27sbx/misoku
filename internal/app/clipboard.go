package app

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const toastDuration = 1800 * time.Millisecond

// toastExpiredMsg hides toast number id (newer toasts keep their own timer).
type toastExpiredMsg struct{ id int }

// copyText puts text on the system clipboard (falling back to the OSC 52
// terminal escape, which also works over SSH) and shows a toast.
func (m *Model) copyText(text string) tea.Cmd {
	write := m.writeClipboard
	if write == nil {
		write = clipboard.WriteAll
	}
	if err := write(text); err != nil {
		termenv.NewOutput(os.Stdout).Copy(text)
	}
	n := utf8.RuneCountInString(text)
	lines := strings.Count(strings.TrimSuffix(text, "\n"), "\n") + 1
	msg := fmt.Sprintf("Copied %d characters", n)
	if lines > 1 {
		msg = fmt.Sprintf("Copied %d lines", lines)
	}
	return m.showToast(msg)
}

func (m *Model) showToast(text string) tea.Cmd {
	m.toastSeq++
	m.toast = text
	id := m.toastSeq
	return tea.Tick(toastDuration, func(time.Time) tea.Msg { return toastExpiredMsg{id: id} })
}

// renderToast draws the toast in the bottom-left corner, above the statusline.
func (m *Model) renderToast(screen string) string {
	if m.toast == "" {
		return screen
	}
	th := m.theme
	box := lipgloss.NewStyle().
		BorderStyle(ui.Border()).
		BorderForeground(th.Green).
		BorderBackground(th.DarkerBg).
		Background(th.DarkerBg).
		Render(lipgloss.NewStyle().Background(th.DarkerBg).Foreground(th.Green).Render(" \U000f018f ") +
			lipgloss.NewStyle().Background(th.DarkerBg).Foreground(th.Fg).Render(m.toast+" "))
	h := lipgloss.Height(box)
	return ui.Overlay(screen, box, 1, max(m.height-footerHeight-h, 0))
}
