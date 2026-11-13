package app

import (
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// inLang runs f with l as the interface language and puts English back.
func inLang(t *testing.T, l i18n.Lang, f func()) {
	t.Helper()
	i18n.Set(l)
	defer i18n.Set(i18n.EN)
	f()
}

// The start screen leaves the sync lines' commands out, which it finds in
// the words of the language in use.
func TestHomeLeavesSyncCommandsOutInEveryLanguage(t *testing.T) {
	home := &Model{viewMode: viewModeDashboard}
	names := []string{"a.md", "b.md", "c.md", "d.md", "e.md"}
	for _, l := range i18n.Langs {
		inLang(t, l, func() {
			for _, text := range []string{
				i18n.T(i18n.AppSyncOffline, 2, ":sync"),
				i18n.T(i18n.AppSyncPausedKey, i18n.T(i18n.AppSyncKeySignsIn, ":sync")),
				i18n.T(i18n.AppSyncPausedWhyKey, lockedText(), i18n.T(i18n.AppSyncKeyOpensIt, ":sync")),
				arrivalLine(arrivalHead(gitsync.SyncResult{NewNotes: names}), names),
			} {
				if got := home.barText(text); strings.Contains(got, ":sync") {
					t.Errorf("%s: the start screen kept the command in %q (from %q)", l, got, text)
				}
			}
		})
	}
}

// A stop's words are read when shown, so they follow the language.
func TestSyncStopFollowsTheLanguage(t *testing.T) {
	msg := syncDoneMsg{res: gitsync.SyncResult{Outcome: gitsync.SyncRewritten}}
	for _, c := range []struct {
		lang i18n.Lang
		want string
	}{
		{i18n.EN, "Sync paused: this vault and origin no longer match · K"},
		{i18n.ES, "Sincronización en pausa: este vault y origin ya no coinciden · K"},
	} {
		inLang(t, c.lang, func() {
			if got, _ := syncSummary(msg, "K"); got != c.want {
				t.Errorf("%s: %q, want %q", c.lang, got, c.want)
			}
		})
	}
}

// The date picker names days and months in the language in use; English
// reads as Go's own format did.
func TestDatesFollowTheLanguage(t *testing.T) {
	day := time.Date(2026, time.November, 13, 0, 0, 0, 0, time.UTC)
	inLang(t, i18n.EN, func() {
		if got, want := longDate(day), day.Format("Mon, 2 Jan 2006"); got != want {
			t.Errorf("English: %q, want %q", got, want)
		}
	})
	inLang(t, i18n.ES, func() {
		if got := longDate(day); got != "vie, 13 nov 2026" {
			t.Errorf("Spanish: %q", got)
		}
		if got := ago(3 * time.Hour); got != "hace 3 h" {
			t.Errorf("Spanish ago: %q", got)
		}
	})
}

// The Windows file dialog's script carries the title and the filter in the
// language in use, quoted so an apostrophe cannot end PowerShell's string.
func TestWindowsDialogScriptFollowsTheLanguage(t *testing.T) {
	for _, c := range []struct {
		lang         i18n.Lang
		title, files string
	}{
		{i18n.EN, "$d.Title = 'Choose an image'", "$d.Filter = 'image|*"},
		{i18n.ES, "$d.Title = 'Elige una imagen'", "$d.Filter = 'imagen|*"},
	} {
		inLang(t, c.lang, func() {
			ps := windowsDialogScript(mediaKinds["image"])
			if !strings.Contains(ps, c.title) || !strings.Contains(ps, c.files) {
				t.Errorf("%s: script lacks %q or %q:\n%s", c.lang, c.title, c.files, ps)
			}
		})
	}
	inLang(t, i18n.EN, func() {
		if ps := windowsDialogScript(mediaKinds["file"]); !strings.Contains(ps, "$d.Filter = 'All files|*.*'") {
			t.Errorf("any file: %s", ps)
		}
	})
	if got := psQuote("l'année ñ"); got != "'l''année ñ'" {
		t.Errorf("psQuote = %q", got)
	}
}
