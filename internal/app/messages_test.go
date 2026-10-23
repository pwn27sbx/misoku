package app_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// at1405 is a clock stopped at 14:05, so :messages always shows that time.
func at1405() time.Time { return time.Date(2026, 10, 23, 14, 5, 0, 0, time.Local) }

const unknownCmd = "E492: Not an editor command: nosuch"

// ex types an ex command in NORMAL mode and runs it.
func (h *harness) ex(cmd string) {
	h.t.Helper()
	h.keys(":" + cmd)
	h.key(tea.KeyEnter)
}

func TestErrorStaysThroughKeysUntilEsc(t *testing.T) {
	h := openNote(t)
	h.ex("nosuch")
	if !strings.Contains(h.view(), unknownCmd) {
		t.Fatalf("setup: the error should show:\n%s", h.view())
	}
	h.keys("jkl")
	if !strings.Contains(h.view(), unknownCmd) {
		t.Fatalf("an error must survive keys other than Esc:\n%s", h.view())
	}
	h.key(tea.KeyEsc)
	if strings.Contains(h.view(), unknownCmd) {
		t.Fatalf("Esc must dismiss the error:\n%s", h.view())
	}
}

func TestEscDismissesErrorAndStillLeavesInsert(t *testing.T) {
	h := openNote(t)
	h.ex("nosuch")
	h.keys("i")
	if v := h.view(); !strings.Contains(v, "INSERT") || !strings.Contains(v, unknownCmd) {
		t.Fatalf("setup: INSERT mode with the error still shown:\n%s", v)
	}
	h.key(tea.KeyEsc)
	v := h.view()
	if strings.Contains(v, unknownCmd) {
		t.Fatalf("Esc must dismiss the error:\n%s", v)
	}
	if strings.Contains(v, "INSERT") || !strings.Contains(v, "NORMAL") {
		t.Fatalf("Esc must also leave INSERT as always (D11):\n%s", v)
	}
}

func TestNextMessageReplacesAnError(t *testing.T) {
	h := openNote(t)
	h.ex("nosuch")
	h.send(core.StatusMsg{Text: "Next thing"})
	v := h.view()
	if strings.Contains(v, unknownCmd) || !strings.Contains(v, "Next thing") {
		t.Fatalf("the next message replaces the error:\n%s", v)
	}
}

func TestInfoMessageStillClearsOnAnyKey(t *testing.T) {
	h := openNote(t)
	h.send(core.StatusMsg{Text: "Just so you know"})
	if !strings.Contains(h.view(), "Just so you know") {
		t.Fatalf("setup: the message should show:\n%s", h.view())
	}
	if strings.Contains(h.view(), "Esc dismiss") {
		t.Fatalf("a normal message has nothing to dismiss:\n%s", h.view())
	}
	h.keys("j")
	if strings.Contains(h.view(), "Just so you know") {
		t.Fatalf("a normal message still clears on the next key:\n%s", h.view())
	}
}

func TestErrorBarSaysHowToDismissIt(t *testing.T) {
	h := openNote(t)
	h.ex("nosuch")
	if !strings.Contains(h.view(), "Esc dismiss · :messages") {
		t.Fatalf("the bar should say how to dismiss an error:\n%s", h.view())
	}
}

func TestMessagesListsWhatWasSaidWithTheTime(t *testing.T) {
	h := openNote(t, app.WithClock(at1405))
	h.send(core.StatusMsg{Text: "Just so you know"})
	h.keys("j")
	h.ex("nosuch")
	h.key(tea.KeyEsc)
	h.ex("messages")
	v := h.view()
	for _, want := range []string{"Messages", "14:05  Just so you know", "14:05  " + unknownCmd} {
		if !strings.Contains(v, want) {
			t.Fatalf(":messages should list %q:\n%s", want, v)
		}
	}
	if strings.Index(v, "Just so you know") > strings.Index(v, unknownCmd) {
		t.Fatalf("oldest first:\n%s", v)
	}
	h.key(tea.KeyEsc)
	if strings.Contains(h.view(), "14:05") {
		t.Fatalf("Esc closes :messages:\n%s", h.view())
	}
}

func TestMessagesTravelsAsACoreMessage(t *testing.T) {
	h := openNote(t, app.WithClock(at1405))
	h.send(core.StatusMsg{Text: "Hello there"})
	h.send(core.MessagesMsg{})
	if !strings.Contains(h.view(), "14:05  Hello there") {
		t.Fatalf("core.MessagesMsg should open the list:\n%s", h.view())
	}
}

func TestMessagesKeepsTheLastHundredAndScrolls(t *testing.T) {
	h := openNote(t, app.WithClock(at1405))
	for i := 1; i <= 105; i++ {
		h.send(core.StatusMsg{Text: fmt.Sprintf("msg %03d", i)})
	}
	h.ex("messages")
	if v := h.view(); !strings.Contains(v, "msg 105") || strings.Contains(v, "msg 006") {
		t.Fatalf("the list opens on the newest message:\n%s", v)
	}
	h.key(tea.KeyDown) // wraps to the oldest kept
	v := h.view()
	if !strings.Contains(v, "msg 006") || strings.Contains(v, "msg 005") {
		t.Fatalf("only the last 100 messages are kept:\n%s", v)
	}
}

func TestMessagesKeepsStartupKeyWarnings(t *testing.T) {
	h := openNote(t, app.WithClock(at1405), app.WithKeys(map[string][]string{"no_such_action": {"ctrl+x"}}))
	h.key(tea.KeyEsc)
	h.ex("messages")
	if v := h.view(); !strings.Contains(v, "14:05") || !strings.Contains(v, "no_such_action") {
		t.Fatalf("the key warnings shown at start should be in :messages:\n%s", v)
	}
}

func TestMessagesWithNothingSaidYet(t *testing.T) {
	h := openNote(t)
	h.ex("messages")
	if v := h.view(); !strings.Contains(v, "No messages yet") {
		t.Fatalf("an empty history says so:\n%s", v)
	}
}

// While a step of the sync setup runs, a key still brings the bar back to
// what the setup is doing, even over an error: the error is in :messages.
func TestSetupBarStillReturnsOverAnErrorWhileAStepRuns(t *testing.T) {
	hung := hangingGH(t)
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello\nworld")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	h.keys("1") // open a.md
	h.send(core.SyncSetupMsg{})
	done := startInBackground(t, h.update(tea.KeyMsg{Type: tea.KeyEnter})) // Start: gh hangs
	waitFor(t, hung)
	h.send(core.StatusMsg{Text: "Something broke", Error: true})
	wantBar(t, h, "Something broke")
	h.keys("j")
	wantBar(t, h, "Setting up sync… Esc cancels")
	h.key(tea.KeyEsc)
	h.send(waitStep(t, done))
}

// :messages lists one sync result once: a note the sync reloaded shows in
// the sync's own line, not a second time on its own.
func TestMessagesListsASyncReloadOnce(t *testing.T) {
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault), app.WithClock(at1405))
	h.keys("1")
	h.key(tea.KeyEsc)
	h.finish(h.update(app.SyncNowMsg{}))
	pushFromDesk(t, remote, "doc.md", "desk edit")
	h.finish(h.update(app.SyncNowMsg{}))
	h.key(tea.KeyEsc)
	h.ex("messages")
	v := h.view()
	if n := strings.Count(v, `"doc.md" reloaded`); n != 1 {
		t.Fatalf(":messages lists the reload %d times, want once:\n%s", n, v)
	}
	if !strings.Contains(v, `Synced · "doc.md" reloaded`) {
		t.Fatalf("the reload belongs to the sync's line:\n%s", v)
	}
}

// Progress such as "Syncing…" only shows in the bar: :messages keeps
// results.
func TestMessagesSkipsSyncProgress(t *testing.T) {
	_, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault), app.WithClock(at1405))
	h.keys("1")
	h.key(tea.KeyEsc)
	h.finish(h.update(app.SyncNowMsg{}))
	h.ex("messages")
	v := h.view()
	if strings.Contains(v, "Syncing…") {
		t.Fatalf(":messages must not keep the sync's progress:\n%s", v)
	}
	if !strings.Contains(v, "Synced") {
		t.Fatalf("setup: the sync's result should be listed:\n%s", v)
	}
}
