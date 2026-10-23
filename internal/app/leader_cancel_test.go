package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// S1: "si la segunda tecla llega y después llega el tick, el panel no
// aparece". A global Ctrl shortcut typed after Space is that second key: it
// runs and ends the sequence, so neither the guide nor a swallowed key
// follows.
func TestLeaderEndsOnAGlobalCtrlShortcut(t *testing.T) {
	h := openNote(t)
	h.keys(" ")
	h.key(tea.KeyCtrlS)
	tickGuide(h)
	if guideOpen(h) {
		t.Fatalf("Ctrl+S came before the tick; the guide must not show:\n%s", h.view())
	}
	h.keys("j")
	if strings.Contains(h.view(), "not a leader key") {
		t.Fatalf("the key after Ctrl+S must not be read as a leader key:\n%s", h.view())
	}
}

// A dialog that opens on its own while Space waits (here :messages, as an
// external message) ends the sequence: closing it brings no guide back and
// the next key runs as usual.
func TestLeaderEndsWhenADialogOpens(t *testing.T) {
	h := openNote(t)
	h.ex("nosuch") // something for :messages to list
	h.key(tea.KeyEsc)
	h.keys(" ")
	h.send(core.MessagesMsg{})
	h.key(tea.KeyEsc) // closes the list
	tickGuide(h)
	if guideOpen(h) {
		t.Fatalf("the guide must not come back after the dialog closed:\n%s", h.view())
	}
	h.keys("t") // "space t" would open the themes
	if v := h.view(); themePickerOpen(v) || strings.Contains(v, "not a leader key") {
		t.Fatalf("t after the dialog must not be read as a leader key:\n%s", v)
	}
}

// A mouse click cancels a pending leader, like any other key that is not
// part of the sequence.
func TestLeaderEndsOnAClick(t *testing.T) {
	h := openNote(t)
	h.keys(" ")
	h.click(1, 1)
	tickGuide(h)
	if guideOpen(h) {
		t.Fatalf("a click must cancel the leader and its guide:\n%s", h.view())
	}
	if strings.Contains(h.view(), "<Space>") {
		t.Fatalf("the bar still waits for a leader key after a click:\n%s", h.view())
	}
}

// On the home screen a leader key that needs an open note says so instead
// of doing nothing in silence.
func TestHomeLeaderKeyThatNeedsANoteSaysSo(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	h.keys(" p")
	v := h.view()
	if !strings.Contains(v, "p: needs an open note") || !strings.Contains(v, "HOME") {
		t.Fatalf("space p on home should say it needs an open note:\n%s", v)
	}
	// Truly unbound keys stay as they were.
	h.keys(" z")
	if strings.Contains(h.view(), "z: needs an open note") {
		t.Fatalf("an unbound key is not a note key:\n%s", h.view())
	}
}
