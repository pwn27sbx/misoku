package dashboard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/i18n"
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

// S1, S9: the home screen draws its entries in the language in use when it
// draws, so a screen built before a switch redraws in the new language.
// The learn entries keep the label the app gave them.
func TestHomeFollowsTheInterfaceLanguage(t *testing.T) {
	useLang(t, i18n.EN)
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetSyncSetup(true)
	db.SetLearn([]dashboard.LearnEntry{{Route: "b", Label: "Aprender · Básico 3/26"}})
	db.SetRecentPages([]core.Page{
		{ID: "a.md", Title: "A", UpdatedAt: time.Now()},
		{ID: "work", Title: "work", IsFolder: true},
	})
	for _, c := range []struct {
		lang i18n.Lang
		want []string
	}{
		{i18n.EN, []string{"New Note", "Find Note", "Open Explorer", "Keymaps", "Settings", "Sync between computers", "notes that live in plain Markdown files", "RECENT", "FOLDERS", "1 note ", "SPC opens every other command"}},
		{i18n.ES, []string{"Nueva nota", "Buscar nota", "Abrir explorador", "Atajos", "Ajustes", "Sincronizar entre computadoras", "notas que viven en archivos Markdown simples", "RECIENTES", "CARPETAS", "1 nota ", "SPC abre los demás comandos"}},
	} {
		i18n.Set(c.lang)
		view := db.View()
		for _, w := range append(c.want, "Aprender · Básico 3/26") {
			if !strings.Contains(view, w) {
				t.Errorf("%s: the home screen misses %q", c.lang, w)
			}
		}
	}
}
