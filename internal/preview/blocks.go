package preview

import (
	"strconv"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// now is replaceable in tests.
var now = time.Now

const dateLayout = "2006-01-02"

// Hit is a clickable region of a rendered view block. Inside a renderer Row
// is relative to the block's first output line and Line to the block body;
// the compiler turns both into absolute positions (preview row, document
// line).
type Hit struct {
	Row, H int // first row and height (rows)
	X0, X1 int // columns [X0, X1)
	Kind   string
	Line   int // source line of the element (-1 if none)
	Index  int
	Arg    string
	Block  int // document line of the opening fence
	End    int // document line of the closing fence
}

func addHit(hs *[]Hit, h Hit) {
	if h.H == 0 {
		h.H = 1
	}
	*hs = append(*hs, h)
}

// HitCheckbox is the box of a to-do line: clicking it toggles the line of
// Line. The text of the line is not part of the hit.
const HitCheckbox = "checkbox"

// CalendarView shifts every calendar block while browsing the preview.
type CalendarView struct {
	Shift int  // months forward (negative = back)
	Today bool // start from the current month instead of the block's
	// Folded holds the keys of collapsed headings / code blocks.
	Folded map[string]bool
	// FoldAll collapses every heading and code block (zM).
	FoldAll bool
	// Notes resolves [[links]] and draws ![[embeds]]; nil treats every link
	// as resolved and every embed as missing.
	Notes NoteLookup
}

// calendarNav is the clickable header on every calendar, in the interface
// language.
func calendarNav() string { return "‹  " + i18n.T(i18n.PreviewToday) + "  ›" }

// monthName is m's name in the interface language: "January", or "Jan" when
// short.
func monthName(m time.Month, short bool) string {
	key := i18n.PreviewMonths
	if short {
		key = i18n.PreviewMonthsShort
	}
	if names := strings.Split(i18n.T(key), ","); int(m) >= 1 && int(m) <= len(names) {
		return names[m-1]
	}
	return m.String()
}

// monthYear names t's month and year: "January 2026", or "Jan 2026" when
// short.
func monthYear(t time.Time, short bool) string {
	return monthName(t.Month(), short) + " " + strconv.Itoa(t.Year())
}

// renderBlock draws the special fenced blocks (board, calendar, timeline,
// chart, form, flow, math). ok is false for ordinary code.
func renderBlock(lang string, body []string, th theme.Theme, width int, cal CalendarView) ([]string, []Hit, bool) {
	var hs []Hit
	var out []string
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "board", "kanban":
		out = renderBoard(body, th, width, &hs)
	case "calendar":
		out = renderCalendar(body, th, width, cal, &hs)
	case "timeline", "gantt":
		out = renderTimeline(body, th, width, &hs)
	case "chart":
		out = renderChart(body, th, width, &hs)
	case "form":
		out = renderForm(body, th, width, &hs)
	case "flow", "flowchart":
		out = renderFlow(body, th, width, &hs)
	case "math", "latex", "tex", "katex":
		out = renderMath(body, th, width, &hs)
	case "mermaid":
		if !isMermaidFlow(body) {
			return nil, nil, false
		}
		out = renderFlow(body, th, width, &hs)
	default:
		return nil, nil, false
	}
	return out, hs, true
}

// keyValues parses "key: value" lines (keys lowercased) and returns the
// remaining lines too.
func keyValues(body []string, keys ...string) (map[string]string, []string) {
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	meta := map[string]string{}
	var rest []string
	for _, l := range body {
		if k, v, ok := strings.Cut(l, ":"); ok && want[strings.ToLower(strings.TrimSpace(k))] {
			meta[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
			continue
		}
		rest = append(rest, l)
	}
	return meta, rest
}

func palette(th theme.Theme) []lipgloss.Color {
	return []lipgloss.Color{th.Blue, th.Green, th.Purple, th.Yellow, th.Orange, th.Cyan, th.Red, th.Pink, th.Teal}
}

func pad(s string, w int) string {
	if d := w - ansi.StringWidth(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

func center(s string, w int) string {
	d := w - ansi.StringWidth(s)
	if d <= 0 {
		return s
	}
	return strings.Repeat(" ", d/2) + s + strings.Repeat(" ", d-d/2)
}
