package preview_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"
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

// S1: the views' controls, placeholders and dates speak the interface
// language; what the note itself says stays as written.
func TestBlocksFollowTheInterfaceLanguage(t *testing.T) {
	th := theme.DefaultTheme()
	cases := []struct {
		src    string
		en, es []string
	}{
		{"```board\n- A\n```", []string{"● Cards", "+ New page"}, []string{"● Tarjetas", "+ Nueva página"}},
		{"```board\n```", []string{"(empty board"}, []string{"(tablero vacío"}},
		{"```calendar\nmonth: 2026-10\n2026-10-15: Launch\n```",
			[]string{"October 2026", "1 events", "Sun", "‹  Today  ›"},
			[]string{"Octubre 2026", "1 evento", "Dom", "Mié", "‹  Hoy  ›"}},
		{"```timeline\nAlpha: 2026-10-01 -> 2026-10-04\n```", []string{"Oct 2026", "+ New"}, []string{"Oct 2026", "+ Nueva"}},
		{"```timeline\nAlpha: 2026-01-01 -> 2026-01-04\n```", []string{"Jan 2026"}, []string{"Ene 2026"}},
		{"```chart\ntype: bar\n```", []string{"(empty chart", "+ Add value"}, []string{"(gráfico vacío", "+ Añadir valor"}},
		{"```form\n? Pick (choice): X | Y\n? When (date)\n```",
			[]string{"Form title", "Description (optional)", "(Respondents can select up to 1)", "+ Add option", "Pick a date (YYYY-MM-DD)", "⊕ add question"},
			[]string{"Título del formulario", "Descripción (opcional)", "(Se pueden elegir hasta 1)", "+ Añadir opción", "Elige una fecha (AAAA-MM-DD)", "⊕ añadir pregunta"}},
		{"```flow\n```", []string{"Empty flowchart"}, []string{"Diagrama de flujo vacío"}},
		{"```math\n```", []string{"Empty equation"}, []string{"Ecuación vacía"}},
		{"```go\nx := 1\n```", []string{"go · 1 line"}, []string{"go · 1 línea"}},
		{"| a |\n| --- |\n| 1 |", []string{"+ Add row   + Add column"}, []string{"+ Añadir fila   + Añadir columna"}},
	}
	for _, c := range cases {
		for _, l := range []struct {
			lang i18n.Lang
			want []string
		}{{i18n.EN, c.en}, {i18n.ES, c.es}} {
			useLang(t, l.lang)
			plain := ansi.Strip(preview.Compile(c.src, th, 60))
			for _, w := range l.want {
				if !strings.Contains(plain, w) {
					t.Errorf("%s %q: missing %q in\n%s", l.lang, c.src, w, plain)
				}
			}
		}
	}
}

// The clickable controls keep working in Spanish, whose labels have other
// widths: each hit covers its label.
func TestControlsAreClickableInSpanish(t *testing.T) {
	useLang(t, i18n.ES)
	th := theme.DefaultTheme()
	for _, c := range []struct{ src, kind, label string }{
		{"| a |\n| --- |\n| 1 |", "table:addcol", "+ Añadir columna"},
		{"| a |\n| --- |\n| 1 |", "table:addrow", "+ Añadir fila"},
		{"```chart\ntype: bar\nA: 1\n```", "chart:add", "+ Añadir valor"},
		{"```timeline\nA: 2026-10-01 -> 2026-10-04\n```", "tl:add", "+ Nueva"},
		{"```calendar\nmonth: 2026-10\n```", "cal:today", "Hoy"},
		{"```calendar\nmonth: 2026-10\n```", "cal:next", "›"},
	} {
		out, hits := preview.CompileHits(c.src, th, 60, "", preview.CalendarView{})
		lines := strings.Split(ansi.Strip(out), "\n")
		found := false
		for _, h := range hits {
			if h.Kind != c.kind {
				continue
			}
			found = true
			line := lines[h.Row]
			i := strings.Index(line, c.label)
			if i < 0 {
				t.Errorf("%s: row %d has no %q: %q", c.kind, h.Row, c.label, line)
				continue
			}
			x0 := ansi.StringWidth(line[:i])
			x1 := x0 + ansi.StringWidth(c.label)
			if h.X0 > x0 || h.X1 < x1 {
				t.Errorf("%s: hit [%d,%d) does not cover %q at [%d,%d)", c.kind, h.X0, h.X1, c.label, x0, x1)
			}
		}
		if !found {
			t.Errorf("%s: no hit", c.kind)
		}
	}
}

// The PDF export leaves out the controls in the language they were drawn in.
func TestPrintedBlocksDropSpanishControls(t *testing.T) {
	useLang(t, i18n.ES)
	lines, _ := preview.PrintBlockLines("board", []string{"## Hoy", "- A"}, theme.DefaultTheme(), 60)
	if joined := ansi.Strip(strings.Join(lines, "\n")); strings.Contains(joined, "Nueva página") || !strings.Contains(joined, "Hoy") {
		t.Errorf("printed Spanish board:\n%s", joined)
	}
	lines, _ = preview.PrintBlockLines("chart", []string{"type: bar"}, theme.DefaultTheme(), 60)
	if joined := ansi.Strip(strings.Join(lines, "\n")); strings.Contains(joined, "vacío") || strings.Contains(joined, "Añadir") {
		t.Errorf("printed empty Spanish chart:\n%s", joined)
	}
}

// S1: the pane's own words: the scroll position and an empty note.
func TestPaneFollowsTheInterfaceLanguage(t *testing.T) {
	useLang(t, i18n.ES)
	p := preview.New(theme.DefaultTheme())
	p.SetSize(60, 20)
	if got := p.ScrollStatus(); got != "Arriba" {
		t.Errorf("ScrollStatus in Spanish = %q", got)
	}
	if v := ansi.Strip(p.View()); !strings.Contains(v, "Nada que mostrar") {
		t.Errorf("empty preview in Spanish:\n%s", v)
	}
}
