package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// dailyDay is the day the daily-note tests run on.
var dailyDay = time.Date(2026, 11, 6, 9, 30, 0, 0, time.Local)

// dailyHarness starts on the home screen of store with the clock on dailyDay.
func dailyHarness(t *testing.T, store *core.Store, opts ...app.Option) *harness {
	t.Helper()
	opts = append([]app.Option{app.WithClock(func() time.Time { return dailyDay })}, opts...)
	return newHarness(t, store, opts...)
}

// S5: today's note already on disk opens, from the home screen.
func TestDailyNoteOpensTodaysNoteFromHome(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("Diario", "2026-11-06", "plan of the day"); err != nil {
		t.Fatal(err)
	}
	h := dailyHarness(t, store)
	h.keys(" od")
	if v := h.view(); !strings.Contains(v, "plan of the day") || !strings.Contains(h.tabline(), "2026-11-06") {
		t.Fatalf("space o d should open Diario/2026-11-06.md:\n%s", v)
	}
}

// S5: without today's note a draft starts with its name and folder, nothing
// is written until it is saved, and saving creates the folder and the note.
func TestDailyNoteStartsADraftAndSavingCreatesIt(t *testing.T) {
	store := newTestStore(t)
	h := dailyHarness(t, store)
	h.keys(" od")
	if !strings.Contains(h.tabline(), "2026-11-06") {
		t.Fatalf("the draft should be named after today:\n%s", h.view())
	}
	if bar := statusBar(h); !strings.Contains(bar, `New daily note "2026-11-06"`) {
		t.Fatalf("the bar should say it is a new daily note to save:\n%s", bar)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "Diario")); !os.IsNotExist(err) {
		t.Fatalf("nothing should be written before saving: %v", err)
	}
	h.keys("today")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if v := h.view(); !strings.Contains(v, "Diario") {
		t.Fatalf("Save As should offer the daily folder:\n%s", v)
	}
	h.key(tea.KeyEnter)
	p, err := store.Get("Diario/2026-11-06.md")
	if err != nil || p.Content != "today" {
		t.Fatalf("Diario/2026-11-06.md = %q, %v", p.Content, err)
	}
}

// S5: a second press focuses today's draft instead of starting another.
func TestDailyNoteTwiceKeepsOneTab(t *testing.T) {
	h := dailyHarness(t, newTestStore(t))
	h.keys(" od")
	h.key(tea.KeyEsc)
	h.keys(" od")
	if n := strings.Count(h.tabline(), "2026-11-06"); n != 1 {
		t.Fatalf("today's draft should show once, got %d tabs:\n%s", n, h.view())
	}
	// From another tab, the key comes back to today's draft.
	h.keys(" n")
	h.key(tea.KeyEsc)
	h.keys(" od")
	if n := strings.Count(h.tabline(), "2026-11-06"); n != 1 {
		t.Fatalf("today's draft should still show once, got %d:\n%s", n, h.view())
	}
	if bar := statusBar(h); !strings.Contains(bar, `New daily note "2026-11-06"`) {
		t.Fatalf("today's draft should be the active tab again:\n%s", bar)
	}
}

// S5: ":daily" does what the key does.
func TestDailyCommand(t *testing.T) {
	h := dailyHarness(t, newTestStore(t))
	h.keys(" n")
	h.key(tea.KeyEsc)
	h.ex("daily")
	if !strings.Contains(h.tabline(), "2026-11-06") {
		t.Fatalf(":daily should start today's note:\n%s", h.view())
	}
}

// S5: daily_dir names the folder; a folder that does not exist yet is kept,
// not replaced by the vault's root.
func TestDailyNoteUsesTheConfiguredFolder(t *testing.T) {
	store := newTestStore(t)
	h := dailyHarness(t, store, app.WithDailyDir("Journal/2026"))
	h.keys(" od")
	h.keys("x")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	h.key(tea.KeyEnter)
	if _, err := store.Get("Journal/2026/2026-11-06.md"); err != nil {
		t.Fatalf("saving should create Journal/2026/2026-11-06.md: %v\n%s", err, h.view())
	}
}

// S5: a daily_dir outside the vault is refused with a message and the
// default folder is used.
func TestDailyNoteRejectsAFolderOutsideTheVault(t *testing.T) {
	store := newTestStore(t)
	h := dailyHarness(t, store, app.WithDailyDir("../elsewhere"))
	if bar := statusBar(h); !strings.Contains(bar, "daily_dir") {
		t.Fatalf("the bar should name the refused setting:\n%s", bar)
	}
	h.keys(" od")
	h.keys("x")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	h.key(tea.KeyEnter)
	if _, err := store.Get("Diario/2026-11-06.md"); err != nil {
		t.Fatalf("the default folder should be used: %v\n%s", err, h.view())
	}
}

// S5: space d stays the home screen.
func TestSpaceDStillGoesHome(t *testing.T) {
	h := openNote(t)
	h.keys(" d")
	if v := h.view(); !strings.Contains(v, "HOME") {
		t.Fatalf("space d should show the home screen:\n%s", v)
	}
}
