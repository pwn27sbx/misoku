package sidebar_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/sidebar"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// useLang switches the interface language for one test and puts the old
// one back. Tests that call it must not run in parallel.
func useLang(t *testing.T, l i18n.Lang) {
	t.Helper()
	old := i18n.Current()
	i18n.Set(l)
	t.Cleanup(func() { i18n.Set(old) })
}

// S1, S9: the explorer's own words follow the language in use when it
// draws.
func TestExplorerFollowsTheInterfaceLanguage(t *testing.T) {
	useLang(t, i18n.EN)
	sb := sidebar.New(theme.DefaultTheme())
	sb.SetSize(40, 20)
	for _, c := range []struct {
		lang i18n.Lang
		want []string
	}{
		{i18n.EN, []string{"Find note", "Empty workspace — press n"}},
		{i18n.ES, []string{"Buscar nota", "Vault vacío — pulsa n"}},
	} {
		i18n.Set(c.lang)
		v := ansi.Strip(sb.View())
		for _, w := range c.want {
			if !strings.Contains(v, w) {
				t.Errorf("%s: the explorer misses %q:\n%s", c.lang, w, v)
			}
		}
	}
}
