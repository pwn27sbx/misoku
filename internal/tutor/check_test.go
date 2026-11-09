package tutor_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

func none(string) bool { return false }

// textCheck builds a text predicate.
func textCheck(pred string, args ...string) tutor.Check {
	return tutor.Check{Kind: tutor.KindText, Pred: pred, Args: args}
}

func diskCheck(pred string, args ...string) tutor.Check {
	return tutor.Check{Kind: tutor.KindDisk, Pred: pred, Args: args}
}

func TestCheckPass(t *testing.T) {
	onlyMiNota := func(name string) bool { return name == "Mi nota" }
	cases := []struct {
		name  string
		check tutor.Check
		text  string
		env   tutor.Env
		want  bool
	}{
		{"contains", textCheck("line-contains", "Hoy empiezo"), "una línea\nHoy empiezo el diario\n", tutor.Env{}, true},
		{"contains no", textCheck("line-contains", "Hoy empiezo"), "otra cosa\n", tutor.Env{}, false},
		{"contains ignores inline code", textCheck("line-contains", "Hoy empiezo"), "Escribe `Hoy empiezo` aquí\n", tutor.Env{}, false},
		{"contains ignores fences", textCheck("line-contains", "Hoy empiezo"), "```\nHoy empiezo\n```\n", tutor.Env{}, false},

		{"starts", textCheck("line-starts", "## Viaje"), "texto\n## Viaje\n", tutor.Env{}, true},
		{"starts needs the prefix", textCheck("line-starts", "## Viaje"), "texto ## Viaje\n", tutor.Env{}, false},
		{"starts indented", textCheck("line-starts", "- [x] "), "  - [x] hecho\n", tutor.Env{}, false},

		{"equals", textCheck("line-equals", "borrador"), "texto\n  borrador  \n", tutor.Env{}, true},
		{"equals no", textCheck("line-equals", "borrador"), "borrador largo\n", tutor.Env{}, false},

		{"ends", textCheck("line-ends", " · maleta"), "gatos · maleta   \n", tutor.Env{}, true},
		{"ends no", textCheck("line-ends", " · maleta"), "gatos · maletas\n", tutor.Env{}, false},

		{"n lines start", textCheck("n-lines-start", "2", "- [x] "), "- [x] a\n- [x] b\n", tutor.Env{}, true},
		{"n lines start short", textCheck("n-lines-start", "2", "- [x] "), "- [x] a\n- [ ] b\n", tutor.Env{}, false},

		{"absent", textCheck("absent", "borrador"), "todo menos eso\n", tutor.Env{}, true},
		{"absent fails", textCheck("absent", "borrador"), "el borrador\n", tutor.Env{}, false},

		{"heading level and text", textCheck("heading-exists", "3", "Día 1"), "## Viaje\n### DÍA 1\n", tutor.Env{}, true},
		{"heading wrong level", textCheck("heading-exists", "2", "Día 1"), "### Día 1\n", tutor.Env{}, false},
		{"heading wrong text", textCheck("heading-exists", "2", "Viaje"), "## Otro\n", tutor.Env{}, false},
		{"heading in fence", textCheck("heading-exists", "2", "Viaje"), "```\n## Viaje\n```\n", tutor.Env{}, false},

		{"wikilink resolves", textCheck("wikilink-resolves", "Mi nota"), "Ver [[Mi nota]].\n", tutor.Env{Exists: onlyMiNota}, true},
		{"wikilink alias resolves", textCheck("wikilink-resolves", "Mi nota"), "Ver [[Mi nota|allí]].\n", tutor.Env{Exists: onlyMiNota}, true},
		{"wikilink missing", textCheck("wikilink-resolves", "Mi nota"), "Ver [[Mi nota]].\n", tutor.Env{Exists: none}, false},
		{"wikilink other note", textCheck("wikilink-resolves", "Mi nota"), "Ver [[Otra]].\n", tutor.Env{Exists: func(string) bool { return true }}, false},
		{"wikilink no reader", textCheck("wikilink-resolves", "Mi nota"), "Ver [[Mi nota]].\n", tutor.Env{}, false},

		{"blockref", textCheck("blockref-exists", "favorita"), "Mi frase ^favorita\n\n[[#^favorita]]\n", tutor.Env{}, true},
		{"blockref no link", textCheck("blockref-exists", "favorita"), "Mi frase ^favorita\n", tutor.Env{}, false},
		{"blockref no id", textCheck("blockref-exists", "favorita"), "[[#^favorita]]\n", tutor.Env{}, false},
		{"blockref other id", textCheck("blockref-exists", "favorita"), "Mi frase ^otra\n\n[[#^favorita]]\n", tutor.Env{}, false},

		{"headinglink", textCheck("headinglink-exists", "Un título"), "## Un título\n\n[[#un TÍTULO|ir]]\n", tutor.Env{}, true},
		{"headinglink no heading", textCheck("headinglink-exists", "Un título"), "[[#Un título]]\n", tutor.Env{}, false},
		{"headinglink other note", textCheck("headinglink-exists", "Un título"), "## Un título\n\n[[Otra#Un título]]\n", tutor.Env{}, false},

		{"table shape", textCheck("table-shape", "2"), "| Fruta | Precio |\n| --- | --- |\n| Mango | 2 |\n", tutor.Env{}, true},
		{"table shape cells", textCheck("table-shape", "3"), "| Fruta | Precio |\n| --- | --- |\n", tutor.Env{}, false},
		{"table shape no separator", textCheck("table-shape", "2"), "| Fruta | Precio |\n", tutor.Env{}, false},
		{"table shape in fence", textCheck("table-shape", "2"), "```\n| a | b |\n| --- | --- |\n```\n", tutor.Env{}, false},

		{"raw keeps inline code", tutor.Check{Kind: tutor.KindTextRaw, Pred: "line-contains", Args: []string{"`misoku`"}}, "Escribe `misoku`\n", tutor.Env{}, true},
		{"text strips inline code", textCheck("line-contains", "`misoku`"), "Escribe `misoku`\n", tutor.Env{}, false},

		{"fence", tutor.Check{Kind: tutor.KindTextFence, Fence: "board", Pattern: "## "}, "```board\n## Columna\n- tarjeta\n```\n", tutor.Env{}, true},
		{"fence wrong kind", tutor.Check{Kind: tutor.KindTextFence, Fence: "chart", Pattern: "## "}, "```board\n## Columna\n```\n", tutor.Env{}, false},
		{"plain code fence", tutor.Check{Kind: tutor.KindTextFence, Fence: "code", Pattern: "Inicio --> Fin"}, "```\nInicio --> Fin\n```\n", tutor.Env{}, true},

		{"disk exists", diskCheck("exists", "Mi nota"), "", tutor.Env{Exists: onlyMiNota}, true},
		{"disk exists missing", diskCheck("exists", "Mi nota"), "", tutor.Env{Exists: none}, false},
		{"disk no-file", diskCheck("no-file", "Otra"), "", tutor.Env{Exists: onlyMiNota}, true},
		{"disk folder", diskCheck("folder-exists", "Diario"), "", tutor.Env{FolderExists: func(n string) bool { return n == "Diario" }}, true},
		{"disk file changed", diskCheck("file-changed"), "", tutor.Env{Changed: func(f string) bool { return f == "B01 Títulos.md" }, LessonFile: "B01 Títulos.md"}, true},
		{"disk file changed no reader", diskCheck("file-changed"), "", tutor.Env{LessonFile: "B01 Títulos.md"}, false},
		{"disk other note text", diskCheck("other-note-text", "Otra", "hola"), "", tutor.Env{NoteText: func(n string) (string, bool) { return "hola mundo", n == "Otra" }}, true},
		{"disk other note missing", diskCheck("other-note-text", "Otra", "hola"), "", tutor.Env{NoteText: func(string) (string, bool) { return "", false }}, false},
		{"disk config present", diskCheck("config-key", "ui.transparent"), "", tutor.Env{Config: func(k string) (string, bool) { return "true", k == "ui.transparent" }}, true},
		{"disk config equals", diskCheck("config-key", "ui.transparent", "true"), "", tutor.Env{Config: func(k string) (string, bool) { return "true", k == "ui.transparent" }}, true},
		{"disk config different", diskCheck("config-key", "ui.transparent", "false"), "", tutor.Env{Config: func(k string) (string, bool) { return "true", k == "ui.transparent" }}, false},
		{"disk config changed", diskCheck("config-changed", "ui.transparent"), "", tutor.Env{ConfigChanged: func(k string) bool { return k == "ui.transparent" }}, true},

		{"all passes", tutor.Check{Kind: tutor.KindAll, Checks: []tutor.Check{textCheck("line-contains", "a"), textCheck("line-contains", "b")}}, "a\nb\n", tutor.Env{}, true},
		{"all fails on one", tutor.Check{Kind: tutor.KindAll, Checks: []tutor.Check{textCheck("line-contains", "a"), textCheck("line-contains", "b")}}, "a\n", tutor.Env{}, false},

		{"action never from text", tutor.Check{Kind: tutor.KindAction, Event: "next_tab"}, "cualquier texto", tutor.Env{}, false},
		{"none never from text", tutor.Check{Kind: tutor.KindNone}, "cualquier texto", tutor.Env{}, false},
		{"unknown kind", tutor.Check{Kind: "bogus"}, "texto", tutor.Env{}, false},
		{"unknown pred", textCheck("bogus", "x"), "x\n", tutor.Env{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.check.Pass(c.text, c.env); got != c.want {
				t.Errorf("Pass = %v, want %v", got, c.want)
			}
		})
	}
}

// E14: text checks see the front matter (only the format test skips it).
func TestCheckSeesFrontMatter(t *testing.T) {
	text := "---\nstatus: hecho\ntitle: Viaje\n---\n\n# Nota\n"
	check := textCheck("line-contains", "status: hecho")
	if !check.Pass(text, tutor.Env{}) {
		t.Fatal("a text check must see the front matter")
	}
}

// E14: {today} expands in text and fence patterns, not only in paths.
func TestTodayInTextPatterns(t *testing.T) {
	env := tutor.Env{Today: "2026-11-09"}
	if !textCheck("line-contains", "## Diario {today}").Pass("## Diario 2026-11-09\n", env) {
		t.Fatal("{today} did not expand in a text pattern")
	}
	check := tutor.Check{Kind: tutor.KindTextFence, Fence: "calendar", Pattern: "{today}: playa"}
	text := "```calendar\n2026-11-09: playa\n```\n"
	if !check.Pass(text, env) {
		t.Fatal("{today} did not expand in a fence pattern")
	}
}

// S1b: {today} expands to Env.Today in a disk path.
func TestCheckTodayToken(t *testing.T) {
	var got string
	env := tutor.Env{Today: "2026-11-09", Exists: func(name string) bool { got = name; return true }}
	check := diskCheck("exists", "Diario/{today}.md")
	if !check.Pass("", env) {
		t.Fatal("the expanded path should exist")
	}
	if got != "Diario/2026-11-09.md" {
		t.Fatalf("path = %q, want Diario/2026-11-09.md", got)
	}
}
