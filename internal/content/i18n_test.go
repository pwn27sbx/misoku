package content_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// useLang switches the interface language for one test and puts the old
// one back. Tests that call it must not run in parallel.
func useLang(t *testing.T, l i18n.Lang) {
	t.Helper()
	old := i18n.Current()
	i18n.Set(l)
	t.Cleanup(func() { i18n.Set(old) })
}

// slash types "/" and query in a new note and picks the first match.
func slash(query string) content.Model {
	c := keys(newNote(""), "i/"+query)
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return c
}

// S1: the "/" menu shows its labels and sections in the interface language.
func TestSlashMenuFollowsTheInterfaceLanguage(t *testing.T) {
	for _, c := range []struct {
		lang i18n.Lang
		want []string
	}{
		{i18n.EN, []string{"Heading 1", "Basic blocks", "Toggle list"}},
		{i18n.ES, []string{"Título 1", "Bloques básicos", "Lista desplegable"}},
	} {
		useLang(t, c.lang)
		m := keys(newNote(""), "i/")
		box, _, _, ok := m.SlashView()
		if !ok {
			t.Fatalf("%s: the menu did not open", c.lang)
		}
		for _, w := range c.want {
			if !strings.Contains(box, w) {
				t.Errorf("%s: the menu misses %q", c.lang, w)
			}
		}
		m = keys(m, "zzzz")
		if box, _, _, _ := m.SlashView(); !strings.Contains(box, i18n.TFor(c.lang, i18n.EditorNoResults)) {
			t.Errorf("%s: no-results line missing:\n%s", c.lang, box)
		}
	}
}

// A6: Spanish aliases and labels find items in Spanish, the English names
// still work, and the starter text the item inserts is in the interface
// language.
func TestSlashStarterTextFollowsTheInterfaceLanguage(t *testing.T) {
	for _, c := range []struct {
		lang   i18n.Lang
		query  string
		prefix string
		want   string
	}{
		{i18n.EN, "board", "```board\n## Not started\n- ", "## Done\n- Card 3"},
		{i18n.ES, "tablero", "```board\n## Sin empezar\n- ", "## Hecho\n- Tarjeta 3"},
		{i18n.ES, "board", "```board\n## Sin empezar\n- ", "Las líneas bajo una tarjeta son su descripción"},
		{i18n.EN, "2 columns", "~~~columns\n", "\n+++\nColumn 2\n~~~"},
		{i18n.ES, "2 columnas", "~~~columns\n", "\n+++\nColumna 2\n~~~"},
		{i18n.EN, "table", "| ", " | Column 2 |"},
		{i18n.ES, "tabla", "| ", " | Columna 2 |"},
		{i18n.EN, "survey", "```form\ntitle: ", "? Question 2 (choice): Option 1 | Option 2"},
		{i18n.ES, "encuesta", "```form\ntitle: ", "? Pregunta 2 (choice): Opción 1 | Opción 2"},
		{i18n.EN, "chart", "```chart\ntype: bar\ntitle: ", "\nJan: 12\n"},
		{i18n.ES, "grafico", "```chart\ntype: bar\ntitle: ", "\nEne: 12\n"},
		{i18n.EN, "flowchart", "```flow\nstart([Start]) --> ", "check{Is it working?}"},
		{i18n.ES, "diagrama", "```flow\nstart([Inicio]) --> ", "check{¿Funciona?}"},
		{i18n.EN, "timeline", "```timeline\n", "Card 2: "},
		{i18n.ES, "cronograma", "```timeline\n", "Tarjeta 2: "},
	} {
		useLang(t, c.lang)
		got := slash(c.query).Value()
		if !strings.HasPrefix(got, c.prefix) || !strings.Contains(got, c.want) {
			t.Errorf("%s /%s inserted %q, want it to start with %q and hold %q", c.lang, c.query, got, c.prefix, c.want)
		}
	}
}

// A6: in English the Spanish aliases do not widen the matches.
func TestSpanishAliasesOnlyMatchInSpanish(t *testing.T) {
	useLang(t, i18n.EN)
	m := keys(newNote(""), "i/tablero")
	if box, _, _, _ := m.SlashView(); !strings.Contains(box, "No results") {
		t.Errorf("an English menu should not match a Spanish alias:\n%s", box)
	}
}

// A6: a column a table command adds is named in the interface language.
func TestNewTableColumnFollowsTheInterfaceLanguage(t *testing.T) {
	useLang(t, i18n.ES)
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "a.md", Content: "| a |\n| --- |\n| 1 |"})
	c.SetFocused(true)
	c = keys(c, "jj:addcol")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(c.Value(), "| a   | Columna |") {
		t.Errorf("Spanish new column: %q", c.Value())
	}
}

// S1: the editor's messages and the ex-command help speak the interface
// language; Vim's error numbers stay.
func TestEditorMessagesFollowTheInterfaceLanguage(t *testing.T) {
	useLang(t, i18n.ES)
	if msg, _ := content.RunCommand("nope", "").(core.StatusMsg); msg.Text != "E492: No es una orden del editor: nope" {
		t.Errorf("unknown command in Spanish = %q", msg.Text)
	}
	if msg, _ := content.RunCommand("graph x", "").(core.StatusMsg); msg.Text != "E488: Sobran caracteres: x (usa :graph o :graph all)" {
		t.Errorf("graph x in Spanish = %q", msg.Text)
	}
	if msg, _ := content.RunCommand("daily x", "").(core.StatusMsg); msg.Text != "E488: Sobran caracteres: x (usa :daily)" {
		t.Errorf("daily x in Spanish = %q", msg.Text)
	}
	if got := content.Commands[0].HelpText(); got != "Guardar" {
		t.Errorf(":w help in Spanish = %q", got)
	}
	if got := content.Commands[0].LongText(); !strings.HasPrefix(got, "Escribe la nota en el disco") {
		t.Errorf(":w explanation in Spanish = %q", got)
	}
	empty := content.New(theme.DefaultTheme())
	empty.SetSize(80, 20)
	if v := empty.View(); !strings.Contains(v, "Ninguna nota abierta") || !strings.Contains(v, "nueva nota") {
		t.Errorf("the empty editor in Spanish:\n%s", v)
	}

	i18n.Set(i18n.EN)
	if msg, _ := content.RunCommand("nope", "").(core.StatusMsg); msg.Text != "E492: Not an editor command: nope" {
		t.Errorf("unknown command in English = %q", msg.Text)
	}
	if got := content.Commands[0].HelpText(); got != "Save" {
		t.Errorf(":w help in English = %q", got)
	}
	if v := empty.View(); !strings.Contains(v, "No note open") {
		t.Errorf("the empty editor in English:\n%s", v)
	}
}

// S1: every ex command's help exists in both languages.
func TestEveryCommandIsTranslated(t *testing.T) {
	for _, l := range i18n.Langs {
		useLang(t, l)
		for _, c := range content.Commands {
			if h, long := c.HelpText(), c.LongText(); h == "" || long == "" || strings.HasPrefix(h, "command.") || strings.HasPrefix(long, "command.") {
				t.Errorf("%s %s: help %q, explanation %q", l, c.Usage, h, long)
			}
		}
	}
}
