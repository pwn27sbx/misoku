//go:build !windows

// The sign-in offer needs the fake git and gh shell scripts the sync tests
// use, so this event's test runs on macOS and Linux only.

package app_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// E15.3: a refused origin offers gh's sign-in, and the offer is the event.
func TestTutorSyncAuthOfferEvent(t *testing.T) {
	d := newDeniedVault(t, githubOrigin, true)
	writeOutside(t, d.vault, eventLessonFile, "# A01\n\ncuerpo\n")
	cat := eventLesson("sync_auth_offer")
	progress := &tutor.Store{}
	h := wideWindow(t, d.vault, d.cfg, app.WithTutor(progress, cat, d.vault, true), d.exec)
	m, cmd := app.OpenInCurrentBuffer(h.m, eventLessonFile)
	h.m = m
	sendCmd(t, h, cmd)
	h.key(tea.KeyEsc)
	l, _ := cat.Lesson(eventLessonFile)
	h.finish(h.update(app.SyncNowMsg{})) // denied: the stop and the offer are known
	pressSync(h)
	passed(t, progress, l, 1)
}
