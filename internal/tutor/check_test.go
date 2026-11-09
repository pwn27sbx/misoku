package tutor_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

func none(string) bool { return false }

func TestPassed(t *testing.T) {
	onlyMiNota := func(name string) bool { return name == "Mi nota" }
	cases := []struct {
		name   string
		check  string
		text   string
		exists func(string) bool
		want   bool
	}{
		{"subheading ok", "subheading", "## Viaje\n\n### Día 1\n", none, true},
		{"subheading h4 without h2", "subheading", "#### Día 1\n", none, false},
		{"subheading h3 before h2", "subheading", "### Día 1\n\n## Viaje\n", none, false},
		{"subheading in fence", "subheading", "```\n## A\n### B\n```\n", none, false},

		{"checkbox ticked", "checkbox", "- [x] hecho\n", none, true},
		{"checkbox X star indented", "checkbox", "  * [X] hecho\n", none, true},
		{"checkbox plus", "checkbox", "+ [x] hecho\n", none, true},
		{"checkbox unticked", "checkbox", "- [ ] x\n", none, false},
		{"checkbox no space", "checkbox", "- [] x\n", none, false},
		{"checkbox inline code", "checkbox", "Escribe `- [x] hecho` aquí\n", none, false},
		{"checkbox tilde fence", "checkbox", "~~~\n- [x] hecho\n~~~\n", none, false},

		{"table ok", "table", "| a | b |\n| --- | :---: |\n| 1 | 2 |\n", none, true},
		{"table no pipes divider", "table", "texto\n---\n", none, false},
		{"table in fence", "table", "```\n| a |\n| --- |\n```\n", none, false},

		{"wikilink ok", "wikilink", "Ver [[Otra nota]].\n", none, true},
		{"wikilink alias", "wikilink", "Ver [[Otra|texto]].\n", none, true},
		{"wikilink heading only", "wikilink", "Ver [[#Título]].\n", none, false},
		{"wikilink empty", "wikilink", "Ver [[ ]].\n", none, false},
		{"wikilink inline code", "wikilink", "Ver `[[Otra nota]]`.\n", none, false},
		{"wikilink double backtick code", "wikilink", "Ver ``a ` [[Otra]]``.\n", none, false},

		{"headinglink ok", "headinglink", "## Mi Título\n\nVer [[#mi título ]].\n", none, true},
		{"headinglink alias", "headinglink", "## Viaje\n\nVer [[#Viaje|ir]].\n", none, true},
		{"headinglink no such heading", "headinglink", "## Viaje\n\nVer [[#Nope]].\n", none, false},
		{"headinglink heading only in code", "headinglink", "```\n## Viaje\n```\nVer [[#Viaje]].\n", none, false},
		{"headinglink other note", "headinglink", "## Viaje\n\nVer [[Otra#Viaje]].\n", none, false},

		{"blockref ok", "blockref", "Idea clave ^idea\n\nVer [[#^idea]].\n", none, true},
		{"blockref alias", "blockref", "Idea clave ^idea\n\nVer [[#^idea|la idea]].\n", none, true},
		{"blockref id without link", "blockref", "Idea clave ^idea\n", none, false},
		{"blockref link without id", "blockref", "Ver [[#^idea]].\n", none, false},
		{"blockref different id", "blockref", "Idea clave ^idea\n\nVer [[#^otra]].\n", none, false},
		{"blockref in code", "blockref", "`Idea ^idea` y `[[#^idea]]`\n", none, false},

		{"newnote ok", "newnote", "Ver [[Mi nota]].\n", onlyMiNota, true},
		{"newnote missing", "newnote", "Ver [[Mi nota]].\n", none, false},
		{"newnote other link", "newnote", "Ver [[Otra]].\n", func(string) bool { return true }, false},

		{"no check", "", "- [x] hecho\n", none, false},
		{"unknown check", "bogus", "- [x] hecho\n", none, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := tutor.Passed(c.check, c.text, c.exists); got != c.want {
				t.Errorf("Passed(%q, %q) = %v, want %v", c.check, c.text, got, c.want)
			}
		})
	}
}

func TestHint(t *testing.T) {
	const want = "- [] no es una tarea: falta el espacio → - [ ]"
	cases := []struct {
		file, text, want string
	}{
		{"B02 Casillas.md", "- [] comprar pan\n", want},
		{"B02 Casillas.md", "  * [] comprar pan\n", want},
		{"B02 Casillas.md", "- [ ] comprar pan\n", ""},
		{"B02 Casillas.md", "Escribe `- []` mal\n", ""},
		{"B01 Títulos.md", "- [] comprar pan\n", ""},
	}
	for _, c := range cases {
		if got := tutor.Hint(c.file, c.text); got != c.want {
			t.Errorf("Hint(%q, %q) = %q, want %q", c.file, c.text, got, c.want)
		}
	}
}
