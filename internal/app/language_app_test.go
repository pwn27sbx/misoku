package app_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// S1: the app's own screens and messages speak the interface language.
// Each case opens one screen in English and in Spanish and looks for its
// words; tests that switch the language do not run in parallel.

// wantAll fails the test when view lacks any of want.
func wantAll(t *testing.T, what, view string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(view, w) {
			t.Fatalf("%s lacks %q:\n%s", what, w, view)
		}
	}
}

func TestUnsavedTabDialogFollowsTheLanguage(t *testing.T) {
	for _, c := range []struct {
		lang i18n.Lang
		want []string
	}{
		{i18n.EN, []string{"Unsaved changes", "Save", "Don't Save", "Cancel"}},
		{i18n.ES, []string{"Cambios sin guardar", "Guardar", "No guardar", "Cancelar"}},
	} {
		t.Run(string(c.lang), func(t *testing.T) {
			useLang(t, c.lang)
			h := openNote(t)
			h.keys("A2")
			h.key(tea.KeyEsc)
			h.keys(" x") // leader x closes the tab
			wantAll(t, "the close dialog", h.view(), c.want...)
		})
	}
}

func TestKeymapPaletteFollowsTheLanguage(t *testing.T) {
	for _, c := range []struct {
		lang i18n.Lang
		want []string
	}{
		{i18n.EN, []string{"MISOKU KEYMAPS & COMMANDS", "From the shell", "Enter run"}},
		{i18n.ES, []string{"ATAJOS Y COMANDOS DE MISOKU", "Desde la terminal", "Enter ejecuta"}},
	} {
		t.Run(string(c.lang), func(t *testing.T) {
			useLang(t, c.lang)
			h := openNote(t)
			h.key(tea.KeyEsc)
			h.keys("?")
			wantAll(t, "the keymap palette", h.view(), c.want...)
		})
	}
}

func TestEmptyMessagesFollowsTheLanguage(t *testing.T) {
	for _, c := range []struct {
		lang i18n.Lang
		want string
	}{
		{i18n.EN, "No messages yet"},
		{i18n.ES, "Todavía no hay mensajes"},
	} {
		t.Run(string(c.lang), func(t *testing.T) {
			useLang(t, c.lang)
			h := openNote(t)
			h.ex("messages")
			wantAll(t, "the bar", h.view(), c.want)
		})
	}
}

// A language switched while the app runs redraws text built from
// package-level tables, not only text built afresh.
func TestPaletteRedrawsAfterALanguageSwitch(t *testing.T) {
	useLang(t, i18n.EN)
	h := openNote(t)
	h.key(tea.KeyEsc)
	h.keys("?")
	wantAll(t, "the English palette", h.view(), "From the shell")
	h.key(tea.KeyEsc)
	i18n.Set(i18n.ES)
	h.keys("?")
	wantAll(t, "the palette after the switch", h.view(), "Desde la terminal")
}
