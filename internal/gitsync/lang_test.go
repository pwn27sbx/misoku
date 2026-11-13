package gitsync_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// useLang sets the interface language for one test, which must not run in
// parallel: the language is the whole process's.
func useLang(t *testing.T, l i18n.Lang) {
	t.Helper()
	old := i18n.Current()
	i18n.Set(l)
	t.Cleanup(func() { i18n.Set(old) })
}

// S1, A4: the setup assistant and sync's messages speak the interface
// language; sentinel errors still match.
func TestSetupAndSyncMessagesInEachLanguage(t *testing.T) {
	for _, c := range []struct {
		lang                   i18n.Lang
		hint, stop, notice, no string
	}{
		{i18n.EN, "Install it with `brew install gh` (see https://cli.github.com), then run setup again.",
			"Setup cancelled; nothing was changed.",
			"a is a git submodule: only its commit id syncs, not its files",
			"git is not installed"},
		{i18n.ES, "Instálalo con `brew install gh` (mira https://cli.github.com) y vuelve a ejecutar el setup.",
			"Setup cancelado; no se cambió nada.",
			"a es un submódulo de git: solo se sincroniza el id de su commit, no sus archivos",
			"git no está instalado"},
	} {
		useLang(t, c.lang)
		if got := gitsync.GHInstallHint("darwin"); got != c.hint {
			t.Errorf("%s hint = %q", c.lang, got)
		}
		s := gitsync.NewSetup(gitsync.SetupOptions{Dir: t.TempDir()})
		if got := s.StopText(i18n.T("setup.cancelled")); got != c.stop {
			t.Errorf("%s StopText = %q", c.lang, got)
		}
		if got := (gitsync.Notice{Kind: gitsync.Submodule, Path: "a"}).String(); got != c.notice {
			t.Errorf("%s notice = %q", c.lang, got)
		}
		wrapped := fmt.Errorf("%w (x)", gitsync.ErrNoGit)
		if got := gitsync.ErrNoGit.Error(); got != c.no || !errors.Is(wrapped, gitsync.ErrNoGit) {
			t.Errorf("%s ErrNoGit = %q, Is = %v", c.lang, got, errors.Is(wrapped, gitsync.ErrNoGit))
		}
	}
}
