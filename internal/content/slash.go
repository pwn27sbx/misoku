package content

import (
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"

	tea "github.com/charmbracelet/bubbletea"
)

// slashItem is one entry of the "/" block menu.
type slashItem struct {
	section string // "table", "basic", "views" or "media"; its header is i18n.SlashSection
	icon    string
	label   i18n.Key
	hint    string   // syntax or a term, the same in every language
	hintKey i18n.Key // a hint in words, in the interface language
	// aliases are English search words, always searched; aliasesES are
	// Spanish ones, searched while the interface is in Spanish.
	aliases   string
	aliasesES string
	before    string // inserted before the cursor
	after     string // inserted after the cursor
	// starter, when set, gives before and after in the interface language:
	// the text a view starts with (A6).
	starter func() (before, after string)
	action  string // "page" / "link" / "template" are handled by the app
	block   bool   // starts its own line
}

// text is the item's label in the interface language.
func (it slashItem) text() string { return i18n.T(it.label) }

// hintText is the hint shown at the right of the item.
func (it slashItem) hintText() string {
	if it.hintKey != "" {
		return i18n.T(it.hintKey)
	}
	return it.hint
}

// insert is the text the item puts before and after the cursor.
func (it slashItem) insert() (before, after string) {
	if it.starter != nil {
		return it.starter()
	}
	return it.before, it.after
}

// haystack is what a query is matched against: the label as shown and in
// English, the English aliases and, in Spanish, the Spanish ones.
func (it slashItem) haystack() string {
	s := it.text() + " " + i18n.TFor(i18n.EN, it.label) + " " + it.aliases
	if i18n.Current() == i18n.ES {
		s += " " + it.aliasesES
	}
	return strings.ToLower(s)
}

// columnsStarter opens a columns block of n columns, named in the
// interface language from the second on.
func columnsStarter(n int) func() (string, string) {
	return func() (string, string) {
		var after strings.Builder
		for i := 2; i <= n; i++ {
			after.WriteString("\n+++\n" + i18n.T(i18n.SlashColumnN, i))
		}
		after.WriteString("\n~~~")
		return "~~~columns\n", after.String()
	}
}

// day is today moved by days, as a note writes a date.
func day(days int) string { return time.Now().AddDate(0, 0, days).Format("2006-01-02") }

var slashItems = []slashItem{
	{section: "table", icon: "\U000f0a15", label: i18n.TableAddRowBelow, aliases: "row insert", aliasesES: "fila insertar añadir", action: "table:addrow"},
	{section: "table", icon: "\U000f0a12", label: i18n.TableAddColRight, aliases: "column col insert", aliasesES: "columna insertar añadir", action: "table:addcol"},
	{section: "table", icon: "\U000f0a17", label: i18n.TableDeleteRow, aliases: "row remove", aliasesES: "fila borrar quitar", action: "table:delrow"},
	{section: "table", icon: "\U000f0a14", label: i18n.TableDeleteColumn, aliases: "column col remove", aliasesES: "columna borrar quitar", action: "table:delcol"},
	{section: "table", icon: "\U000f04eb", label: i18n.SlashLabel("format"), aliases: "align tidy", aliasesES: "alinear ordenar formato", action: "table:format"},
	{section: "basic", icon: "\U000f026b", label: i18n.SlashLabel("h1"), hint: "#", aliases: "h1 title", aliasesES: "titulo encabezado", before: "# ", block: true},
	{section: "basic", icon: "\U000f026c", label: i18n.SlashLabel("h2"), hint: "##", aliases: "h2 subtitle", aliasesES: "titulo subtitulo encabezado", before: "## ", block: true},
	{section: "basic", icon: "\U000f026d", label: i18n.SlashLabel("h3"), hint: "###", aliases: "h3", aliasesES: "titulo encabezado", before: "### ", block: true},
	{section: "basic", icon: "\U000f026e", label: i18n.SlashLabel("h4"), hint: "####", aliases: "h4", aliasesES: "titulo encabezado", before: "#### ", block: true},
	{section: "basic", icon: "\U000f0279", label: i18n.SlashLabel("bullet"), hint: "-", aliases: "bullet ul unordered", aliasesES: "viñetas vinetas puntos", before: "- ", block: true},
	{section: "basic", icon: "\U000f027b", label: i18n.SlashLabel("numbered"), hint: "1.", aliases: "ol ordered number", aliasesES: "numerada ordenada numeros", before: "1. ", block: true},
	{section: "basic", icon: "\U000f0135", label: i18n.SlashLabel("todo"), hint: "[]", aliases: "todo task checkbox check", aliasesES: "tareas pendientes casilla", before: "- [ ] ", block: true},
	{section: "basic", icon: "\U000f0142", label: i18n.SlashLabel("toggle"), hint: ">", aliases: "details collapse", aliasesES: "desplegable plegable", before: "<details>\n<summary>", after: "</summary>\n\n</details>", block: true},
	{section: "basic", icon: "\U000f0219", label: i18n.SlashLabel("page"), aliases: "subpage sub-note note new", aliasesES: "pagina subpagina subnota nota nueva", action: "page"},
	{section: "basic", icon: "\U000f02fd", label: i18n.SlashLabel("callout"), aliases: "note tip info admonition", aliasesES: "aviso nota consejo", before: "> [!NOTE]\n> ", block: true},
	{section: "basic", icon: "\U000f027e", label: i18n.SlashLabel("quote"), hint: "\"", aliases: "blockquote", aliasesES: "cita", before: "> ", block: true},
	{section: "basic", icon: "\U000f04eb", label: i18n.SlashLabel("table"), aliases: "grid", aliasesES: "tabla cuadricula", starter: func() (string, string) {
		return "| ", " | " + i18n.T(i18n.SlashColumnN, 2) + " |\n| --- | --- |\n|  |  |"
	}, block: true},
	{section: "basic", icon: "\U000f04a0", label: i18n.SlashLabel("equation"), hint: "$$", aliases: "math latex tex formula katex", aliasesES: "ecuacion matematicas formula", before: "```math\n", after: "\n```", block: true},
	{section: "basic", icon: "\U000f0374", label: i18n.SlashLabel("divider"), hint: "---", aliases: "hr rule line separator", aliasesES: "separador linea", before: "---\n", block: true},
	{section: "basic", icon: "\U000f0bcc", label: i18n.SlashLabel("columns2"), hint: "+++", aliases: "columns cols layout side split", aliasesES: "columnas", starter: columnsStarter(2), block: true},
	{section: "basic", icon: "\U000f056d", label: i18n.SlashLabel("columns3"), hint: "+++", aliases: "columns cols layout side split", aliasesES: "columnas", starter: columnsStarter(3), block: true},
	{section: "basic", icon: "\U000f0728", label: i18n.SlashLabel("columns4"), hint: "+++", aliases: "columns cols layout side split", aliasesES: "columnas", starter: columnsStarter(4), block: true},
	{section: "basic", icon: "\U000f0337", label: i18n.SlashLabel("link"), aliases: "mention reference note", aliasesES: "enlace mencion referencia nota", action: "link"},
	{section: "basic", icon: "\U000f018f", label: i18n.SlashLabel("template"), hint: "{{}}", aliases: "plantilla template", action: "template"},
	{section: "views", icon: "\U000f0564", label: i18n.SlashLabel("board"), hint: "kanban", aliases: "kanban status tasks", aliasesES: "tablero tareas estado", starter: func() (string, string) {
		return i18n.T(i18n.SlashBoardStart), i18n.T(i18n.SlashBoardRest)
	}, block: true},
	{section: "views", icon: "\U000f00ed", label: i18n.SlashLabel("calendar"), aliases: "month events dates", aliasesES: "calendario mes eventos fechas", before: "```calendar\nmonth: " + time.Now().Format("2006-01") + "\n" + time.Now().Format("2006-01-02") + ": ", after: "\n```", block: true},
	{section: "views", icon: "\U000f0b77", label: i18n.SlashLabel("timeline"), hint: "gantt", aliases: "gantt schedule roadmap", aliasesES: "cronograma calendario plan", starter: func() (string, string) {
		return "```timeline\n", i18n.T(i18n.SlashTimelineRest, day(-3), day(2), day(0), day(4))
	}, block: true},
	{section: "views", icon: "\U000f0128", label: i18n.SlashLabel("chart"), hint: "bar line pie", aliases: "graph plot bar line pie donut hbar", aliasesES: "grafico barras lineas torta", starter: func() (string, string) {
		return "```chart\ntype: bar\ntitle: ", i18n.T(i18n.SlashChartRest)
	}, block: true},
	{section: "views", icon: "\U000f0b4b", label: i18n.SlashLabel("form"), aliases: "survey questions questionnaire", aliasesES: "formulario encuesta preguntas cuestionario", starter: func() (string, string) {
		return "```form\ntitle: ", i18n.T(i18n.SlashFormRest)
	}, block: true},
	{section: "views", icon: "\U000f04aa", label: i18n.SlashLabel("flowchart"), hint: "-->", aliases: "flow diagram graph mermaid excalidraw chart process", aliasesES: "diagrama flujo proceso", starter: func() (string, string) {
		return i18n.T(i18n.SlashFlowStart), i18n.T(i18n.SlashFlowRest)
	}, block: true},
	{section: "media", icon: "\U000f0e7c", label: i18n.SlashLabel("cover"), hint: "banner", aliases: "banner header hero", aliasesES: "portada cabecera", action: "media:cover"},
	{section: "media", icon: "\U000f02e9", label: i18n.SlashLabel("image"), aliases: "picture photo img upload", aliasesES: "imagen foto subir", action: "media:image"},
	{section: "media", icon: "\U000f02e9", label: i18n.SlashLabel("paste"), hintKey: i18n.SlashHintClipboard, aliases: "clipboard paste screenshot screen shot", aliasesES: "pegar portapapeles captura pantalla", action: "media:paste"},
	{section: "media", icon: "\U000f0567", label: i18n.SlashLabel("video"), aliases: "movie upload", aliasesES: "pelicula subir", action: "media:video"},
	{section: "media", icon: "\U000f075a", label: i18n.SlashLabel("audio"), aliases: "sound music upload", aliasesES: "sonido musica subir", action: "media:audio"},
	{section: "media", icon: "\U000f0214", label: i18n.SlashLabel("file"), aliases: "attachment pdf upload document", aliasesES: "archivo adjunto documento subir", action: "media:file"},
	{section: "media", icon: "\U000f0169", label: i18n.SlashLabel("code"), hint: "```", aliases: "snippet block pre", aliasesES: "codigo bloque", before: "```\n", after: "\n```", block: true},
}

const (
	slashWidth   = 36
	slashMaxRows = 12
)

type slashMenu struct {
	row, col int // position of the "/" that opened the menu
	sel      int
	offset   int
}

// SlashOpen reports whether the "/" block menu is showing.
func (m Model) SlashOpen() bool { return m.slash != nil }

func (m *Model) slashQuery() (string, bool) {
	row, col := m.textarea.RowCol()
	if m.slash == nil || row != m.slash.row || col <= m.slash.col {
		return "", false
	}
	before := []rune(m.textarea.LineBeforeCursor())
	if m.slash.col >= len(before) || before[m.slash.col] != '/' {
		return "", false
	}
	q := string(before[m.slash.col+1:])
	if strings.Contains(q, "  ") || strings.HasPrefix(q, " ") {
		return "", false
	}
	return q, true
}

func (m *Model) slashMatches() []slashItem {
	q, _ := m.slashQuery()
	q = strings.ToLower(strings.TrimSpace(q))
	inTable := m.InTable()
	var out []slashItem
	for _, it := range slashItems {
		if it.section == "table" && !inTable {
			continue
		}
		if q == "" || strings.Contains(it.haystack(), q) {
			out = append(out, it)
		}
	}
	return out
}

// maybeOpenSlash opens the menu right after a "/" typed at the start of a
// line or after whitespace.
func (m *Model) maybeOpenSlash(k tea.KeyMsg) {
	if k.Type != tea.KeyRunes || string(k.Runes) != "/" {
		return
	}
	before := []rune(m.textarea.LineBeforeCursor())
	n := len(before)
	if n == 0 || before[n-1] != '/' {
		return
	}
	if n >= 2 && before[n-2] != ' ' {
		return
	}
	row, _ := m.textarea.RowCol()
	m.slash = &slashMenu{row: row, col: n - 1}
	if m.InTable() {
		m.reportAction("slash_open", "in-table")
	} else {
		m.reportAction("slash_open", "")
	}
}

// updateSlash handles a key while the menu is open. handled=false lets the
// key reach the editor (typing filters the menu).
func (m *Model) updateSlash(k tea.KeyMsg) (tea.Cmd, bool) {
	items := m.slashMatches()
	switch k.String() {
	case "esc":
		m.slash = nil
		m.reportAction("slash_close", "")
		return nil, true
	case "up", "ctrl+p", "ctrl+k":
		if len(items) > 0 {
			m.slash.sel = (m.slash.sel - 1 + len(items)) % len(items)
			m.scrollSlash()
		}
		return nil, true
	case "down", "ctrl+n", "ctrl+j":
		if len(items) > 0 {
			m.slash.sel = (m.slash.sel + 1) % len(items)
			m.scrollSlash()
		}
		return nil, true
	case "enter", "tab":
		if len(items) == 0 {
			m.slash = nil
			return nil, k.String() == "tab"
		}
		return m.applySlash(items[min(m.slash.sel, len(items)-1)]), true
	}
	return nil, false
}

func (m *Model) scrollSlash() {
	s := m.slash
	if s.sel < s.offset {
		s.offset = s.sel
	}
	if s.sel >= s.offset+slashMaxRows {
		s.offset = s.sel - slashMaxRows + 1
	}
}

// afterSlashKey re-validates the menu once the editor consumed a key.
func (m *Model) afterSlashKey() {
	if m.slash == nil {
		return
	}
	if _, ok := m.slashQuery(); !ok {
		m.slash = nil
		return
	}
	if q, _ := m.slashQuery(); strings.HasSuffix(q, " ") && len(m.slashMatches()) == 0 {
		m.slash = nil
		return
	}
	m.slash.sel, m.slash.offset = 0, 0
}

// applySlash replaces "/query" with the chosen block's Markdown.
func (m *Model) applySlash(it slashItem) tea.Cmd {
	q, _ := m.slashQuery()
	ta := &m.textarea
	ta.DeleteBefore(1 + len([]rune(q)))
	m.slash = nil

	if op, ok := strings.CutPrefix(it.action, "table:"); ok {
		m.tableOp(op)
		return nil
	}
	if it.action != "" {
		action := it.action
		return func() tea.Msg { return core.SlashActionMsg{Action: action} }
	}
	if it.block && strings.TrimSpace(ta.LineBeforeCursor()) != "" {
		ta.InsertString("\n")
	}
	before, after := it.insert()
	ta.InsertString(before)
	row, col := ta.RowCol()
	if after != "" {
		ta.InsertString(after)
		ta.SetRowCol(row, col)
	}
	ta.EnsureVisible()
	return nil
}

// SetFrontMatter sets key: value in the note's front matter through
// core.SetFrontMatterField (creating the block if needed) and keeps the
// cursor on the same text. It is one change that u undoes.
func (m *Model) SetFrontMatter(key, value string) {
	m.edit(func() { m.setFrontMatter(key, value) })
}

func (m *Model) setFrontMatter(key, value string) {
	text := m.textarea.Value()
	row, col := m.textarea.RowCol()
	updated := core.SetFrontMatterField(text, key, value)
	// Lines are only added or removed inside the front matter (or a new
	// block above everything), so text after it moves by the difference.
	delta := strings.Count(updated, "\n") - strings.Count(text, "\n")
	if end := core.ParseFrontMatter(text).EndLine; end >= 0 && row < end {
		delta = 0
	}
	m.textarea.SetValue(updated)
	m.textarea.SetRowCol(row+delta, col)
}

// InsertText types text at the cursor (used for "Link to page"). Outside
// INSERT it is one change that u undoes; inside, part of the session.
func (m *Model) InsertText(s string) {
	m.edit(func() {
		m.textarea.InsertString(s)
		m.textarea.EnsureVisible()
	})
}

// InsertBlock types s on a line of its own, moving whatever shared the cursor's
// line out of the way. A picture needs this: the preview draws one only when
// the image is alone on its line.
func (m *Model) InsertBlock(s string) {
	m.edit(func() {
		ta := &m.textarea
		line := ta.CurrentLine()
		before := ta.LineBeforeCursor()
		head := strings.TrimSpace(line[:min(len(before), len(line))])
		tail := strings.TrimSpace(line[min(len(before), len(line)):])
		if head != "" {
			ta.InsertString("\n") // the link starts on a fresh line
		}
		if tail != "" {
			ta.InsertString("\n") // and the rest moves below it
		}
		ta.InsertString(s)
		if tail != "" {
			ta.InsertString("\n")
		}
		ta.EnsureVisible()
	})
}
