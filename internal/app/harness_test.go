package app_test

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
)

type slowMsg struct{}

// A command that does real work is waited for however slow the machine is:
// on a loaded CI runner a click's hit message can take longer than a timer
// tick, and dropping it left the old note open (TestBacklinkOpensLinkingNoteAtTheLink).
func TestHarnessWaitsForSlowCommands(t *testing.T) {
	t.Parallel()
	slow := func() tea.Msg { time.Sleep(200 * time.Millisecond); return slowMsg{} }
	if _, ok := runCmd(slow).(slowMsg); !ok {
		t.Fatal("a slow command's message was dropped")
	}
}

// Commands that only wait for a clock are not run at all, so the harness
// never waits for a cursor blink or a toast to expire.
func TestHarnessSkipsTimers(t *testing.T) {
	t.Parallel()
	blink := cursor.New()
	blink.BlinkSpeed = time.Hour
	start := time.Now()
	for _, c := range []tea.Cmd{
		blink.BlinkCmd(),
		tea.Tick(time.Hour, func(time.Time) tea.Msg { return slowMsg{} }),
		tea.Every(time.Hour, func(time.Time) tea.Msg { return slowMsg{} }),
	} {
		if msg := runCmd(c); msg != nil {
			t.Fatalf("a timer returned %T", msg)
		}
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("the harness waited %v for timers", d)
	}
}
