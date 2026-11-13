package app

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// Stable IDs: the compact arrival notice is its own sentence in every
// language, not the long one with an English suffix cut off.
func TestArrivalNoticeHasACompactFormInSpanish(t *testing.T) {
	i18n.Set(i18n.ES)
	t.Cleanup(func() { i18n.Set(i18n.EN) })
	for _, res := range []gitsync.SyncResult{
		{NewNotes: []string{"a.md"}},
		{NewNotes: []string{"a.md", "b.md"}},
		{UpdatedNotes: []string{"c.md"}},
		{NewNotes: []string{"a.md"}, UpdatedNotes: []string{"c.md", "d.md"}},
	} {
		long, compact := arrivalHead(res), arrivalCompact(res)
		if !strings.Contains(long, "desde otro equipo") {
			t.Errorf("long notice %q should say where the notes came from, in Spanish", long)
		}
		if strings.Contains(compact, "desde otro equipo") || strings.Contains(compact, "another computer") {
			t.Errorf("compact notice %q should leave out where the notes came from", compact)
		}
		if !strings.Contains(compact, "a.md") && !strings.Contains(compact, "c.md") {
			t.Errorf("compact notice %q lacks the names", compact)
		}
	}
	if got := arrivalHead(gitsync.SyncResult{NewNotes: []string{"a.md"}}); got != "1 nota nueva desde otro equipo" {
		t.Errorf("one new note: %q", got)
	}
}

// Choosing Auto (system) in the settings menu runs in Update: it must use
// the system language read at startup, never read it again (on macOS that
// runs `defaults`). Here the environment says English and the cached
// system language Spanish: a fresh read would say English.
func TestAutoUsesTheSystemLanguageReadAtStartup(t *testing.T) {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(name, "")
	}
	t.Setenv("LANGUAGE", "en")
	i18n.SetSystem(i18n.ES)
	t.Cleanup(func() { i18n.SetSystem("") })
	if got := systemLanguage(); got != i18n.ES {
		t.Fatalf("Auto resolved to %q, want the cached es", got)
	}
}
