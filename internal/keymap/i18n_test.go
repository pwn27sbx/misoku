package keymap_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/keymap"
)

// useLang switches the interface language for one test and puts the old
// one back. Tests that call it must not run in parallel.
func useLang(t *testing.T, l i18n.Lang) {
	t.Helper()
	old := i18n.Current()
	i18n.Set(l)
	t.Cleanup(func() { i18n.Set(old) })
}

// S1: the help of every action exists in both languages.
func TestEveryActionIsTranslated(t *testing.T) {
	for _, a := range keymap.Actions {
		for _, l := range i18n.Langs {
			for _, key := range []i18n.Key{i18n.KeymapHelp(a.Action), i18n.KeymapLong(a.Action)} {
				if got := i18n.TFor(l, key); got == string(key) || got == "" {
					t.Errorf("%s: no %s text for %q", a.Action, l, key)
				}
			}
		}
	}
}

// S1: the help screen, the palette and the leader guide read the action's
// help in the interface language, at the time they draw.
func TestHelpFollowsTheInterfaceLanguage(t *testing.T) {
	useLang(t, i18n.ES)
	if got := keymap.HelpText(keymap.NewNote); got != "Nueva nota (pestaña sin guardar)" {
		t.Errorf("HelpText(new_note) in Spanish = %q", got)
	}
	if got := keymap.LongText(keymap.Save); !strings.HasPrefix(got, "Escribe en el disco la nota") {
		t.Errorf("LongText(save) in Spanish = %q", got)
	}
	i18n.Set(i18n.EN)
	if got := keymap.HelpText(keymap.NewNote); got != "New note (unsaved tab)" {
		t.Errorf("HelpText(new_note) in English = %q", got)
	}
	if got := keymap.LongText(keymap.Save); !strings.HasPrefix(got, "Writes the note in this tab") {
		t.Errorf("LongText(save) in English = %q", got)
	}
}

// S1: the warnings about the "keys" section speak the interface language;
// the config key and the action names stay as written.
func TestKeyWarningsFollowTheInterfaceLanguage(t *testing.T) {
	useLang(t, i18n.ES)
	_, warnings := keymap.New(map[string][]string{"nope": {"x"}, "graph": {"space o"}, "themes": {"a b"}})
	joined := strings.Join(warnings, "\n")
	for _, want := range []string{
		`keys: acción desconocida "nope"`,
		`"a b": un atajo de varias teclas debe empezar con space`,
		`así que toggle_checkbox nunca se puede teclear`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("Spanish warnings miss %q:\n%s", want, joined)
		}
	}
}

// S1: a leader group without a name of its own is "more" / "más".
func TestAnUnnamedLeaderGroupFollowsTheInterfaceLanguage(t *testing.T) {
	m, _ := keymap.New(map[string][]string{"graph": {"space z g"}})
	useLang(t, i18n.ES)
	if got := m.LeaderSubgroups([]string{keymap.Graph}, nil); !reflect.DeepEqual(got, []keymap.Subgroup{{Key: "z", Name: "más"}}) {
		t.Errorf("Spanish subgroups = %v", got)
	}
	i18n.Set(i18n.EN)
	if got := m.LeaderSubgroups([]string{keymap.Graph}, nil); !reflect.DeepEqual(got, []keymap.Subgroup{{Key: "z", Name: "more"}}) {
		t.Errorf("English subgroups = %v", got)
	}
}
