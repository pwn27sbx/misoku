package preview

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

// formQuestion is one "? ..." line plus its "= answer" line, if any.
type formQuestion struct {
	text, kind string
	required   bool
	options    []string
	line       int
	answer     string
}

func parseForm(body []string) []formQuestion {
	var qs []formQuestion
	for i, l := range body {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "?"):
			q := formQuestion{line: i, kind: "text", required: strings.HasPrefix(t, "?*")}
			t = strings.TrimSpace(strings.TrimLeft(t, "?*"))
			text, opts, _ := strings.Cut(t, ":")
			if j := strings.LastIndex(text, "("); j > 0 && strings.HasSuffix(strings.TrimSpace(text), ")") {
				q.kind = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(text[j+1:]), ")"))
				text = strings.TrimSpace(text[:j])
			}
			q.text = strings.TrimSpace(text)
			for _, o := range strings.Split(opts, "|") {
				if o = strings.TrimSpace(o); o != "" {
					q.options = append(q.options, o)
				}
			}
			qs = append(qs, q)
		case strings.HasPrefix(t, "=") && len(qs) > 0:
			qs[len(qs)-1].answer = strings.TrimSpace(t[1:])
		}
	}
	return qs
}

func renderForm(body []string, th theme.Theme, width int, hs *[]Hit) []string {
	meta, _ := keyValues(body, "title", "description")
	w := min(width, 64)
	text := lipgloss.NewStyle().Foreground(th.Fg)
	muted := lipgloss.NewStyle().Foreground(th.GreyFg2)
	faint := lipgloss.NewStyle().Foreground(th.GreyFg)
	accent := lipgloss.NewStyle().Foreground(th.Blue)

	lineOf := func(key string) int {
		for i, l := range body {
			if k, _, ok := strings.Cut(l, ":"); ok && strings.EqualFold(strings.TrimSpace(k), key) {
				return i
			}
		}
		return -1
	}
	title := meta["title"]
	if title == "" {
		title = "Form title"
	}
	addHit(hs, Hit{Row: 0, X1: w, Kind: "form:title", Line: lineOf("title"), Arg: meta["title"]})
	out := []string{text.Bold(true).Render(strings.ToUpper(title[:1]) + title[1:])}
	desc := meta["description"]
	addHit(hs, Hit{Row: 1, X1: w, Kind: "form:desc", Line: lineOf("description"), Arg: desc})
	if desc == "" {
		out = append(out, faint.Render("Description (optional)"))
	} else {
		out = append(out, muted.Render(desc))
	}
	out = append(out, "")

	card := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(th.Line).
		Width(w-2).
		Padding(0, 1)
	input := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(th.OneBg3).
		Width(w - 6)

	qs := parseForm(body)
	for _, q := range qs {
		var lines []string
		var local []Hit
		mark := func(kind string, h, idx int) {
			local = append(local, Hit{Row: len(lines), H: h, X1: w, Kind: kind, Line: q.line, Index: idx, Arg: q.text})
		}
		head := text.Bold(true).Render(q.text)
		if q.required {
			head += lipgloss.NewStyle().Foreground(th.Red).Render(" *")
		}
		mark("form:question", 1, 0)
		lines = append(lines, head)

		answerBox := func(placeholder string, height int) {
			body := faint.Render(" " + placeholder)
			if q.answer != "" {
				body = text.Render(" " + q.answer)
			}
			st := input
			if height > 1 {
				st = st.Height(height)
			}
			box := strings.Split(st.Render(body), "\n")
			mark("form:text", len(box), 0)
			lines = append(lines, box...)
		}
		chosen := map[string]bool{}
		for _, a := range strings.Split(q.answer, ",") {
			chosen[strings.TrimSpace(a)] = true
		}
		switch q.kind {
		case "choice", "select", "radio":
			lines = append(lines, faint.Render("(Respondents can select up to 1)"))
			for i, o := range q.options {
				mark("form:option", 1, i)
				if chosen[o] {
					lines = append(lines, accent.Render("● ")+text.Bold(true).Render(o))
				} else {
					lines = append(lines, muted.Render("○ ")+text.Render(o))
				}
			}
			mark("form:addopt", 1, 0)
			lines = append(lines, faint.Render("+ Add option"))
		case "multi", "checkbox", "checkboxes":
			lines = append(lines, faint.Render(fmt.Sprintf("(Respondents can select up to %d)", len(q.options))))
			for i, o := range q.options {
				mark("form:multi", 1, i)
				if chosen[o] {
					lines = append(lines, accent.Render("☑ ")+text.Bold(true).Render(o))
				} else {
					lines = append(lines, muted.Render("☐ ")+text.Render(o))
				}
			}
			mark("form:addopt", 1, 0)
			lines = append(lines, faint.Render("+ Add option"))
		case "rating":
			n, _ := strconv.Atoi(q.answer)
			var stars []string
			for i := 1; i <= 5; i++ {
				if i <= n {
					stars = append(stars, "★")
				} else {
					stars = append(stars, "☆")
				}
				local = append(local, Hit{Row: len(lines), H: 1, X0: 2 + (i-1)*2, X1: 4 + (i-1)*2, Kind: "form:rating", Line: q.line, Index: i, Arg: q.text})
			}
			lines = append(lines, lipgloss.NewStyle().Foreground(th.Yellow).Render(strings.Join(stars, " ")))
		case "date":
			answerBox("󰃭 Pick a date (YYYY-MM-DD)", 1)
		case "long", "paragraph":
			answerBox("Long answer", 3)
		case "email":
			answerBox("name@example.com", 1)
		default:
			answerBox("Respondent's answer", 1)
		}
		cardStart := len(out)
		for _, h := range local {
			h.Row += cardStart + 1 // top border
			addHit(hs, h)
		}
		out = append(out, strings.Split(card.Render(strings.Join(lines, "\n")), "\n")...)
		out = append(out, "")
	}
	if len(qs) == 0 {
		out = append(out, faint.Render("(add questions: \"? Question\", \"? Pick one (choice): A | B\")"))
	}
	addHit(hs, Hit{Row: len(out), X1: w, Kind: "form:addq", Line: -1})
	out = append(out, center(accent.Render("⊕ add question"), w))
	return out
}
