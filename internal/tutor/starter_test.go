package tutor_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/tutor"
)

// starters is the text the "/" menu inserts for each view the routes use, in
// l, as internal/content assembles it from the same catalog entries.
func starters(l i18n.Lang) map[string]string {
	columns := func(n int) string {
		s := "~~~columns\n"
		for i := 2; i <= n; i++ {
			s += "\n+++\n" + i18n.TFor(l, i18n.SlashColumnN, i)
		}
		return s + "\n~~~\n"
	}
	return map[string]string{
		"columns3": columns(3),
		"columns4": columns(4),
		"board":    i18n.TFor(l, i18n.SlashBoardStart) + i18n.TFor(l, i18n.SlashBoardRest) + "\n",
		"timeline": "```timeline\n" + i18n.TFor(l, i18n.SlashTimelineRest, "2026-01-01", "2026-01-05", "2026-01-02", "2026-01-06") + "\n",
		"chart":    "```chart\ntype: bar\ntitle: " + i18n.TFor(l, i18n.SlashChartRest) + "\n",
		"form":     "```form\ntitle: " + i18n.TFor(l, i18n.SlashFormRest) + "\n",
	}
}

// A6: every route step that checks the text a "/" menu item inserts passes
// on that language's starter text, and I13's rename step is not met by the
// starter alone. Routes B and E have no such step.
func TestInsertStepsMatchTheInsertedStarterText(t *testing.T) {
	cases := []struct {
		lesson  string
		step    int
		starter string
		want    bool
	}{
		{"I11", 1, "columns3", true},
		{"I11", 2, "columns4", true},
		{"I13", 0, "board", true},
		{"I13", 3, "board", false}, // the player renames a header; the starter must not pass it already
		{"I15", 0, "timeline", true},
		{"I16", 0, "chart", true},
		{"A15", 0, "form", true},
	}
	for _, l := range i18n.Langs {
		cat, err := tutor.BundledRoutesFor(l)
		if err != nil {
			t.Fatalf("%s: %v", l, err)
		}
		texts := starters(l)
		for _, c := range cases {
			lesson, ok := findLesson(cat, c.lesson)
			if !ok {
				t.Fatalf("%s: no lesson %s", l, c.lesson)
			}
			step := lesson.Steps[c.step]
			if got := step.Check.Pass(texts[c.starter], tutor.Env{}); got != c.want {
				t.Errorf("%s %s#%d on the /%s starter: Pass = %v, want %v (check %+v)\n%s",
					l, c.lesson, c.step+1, c.starter, got, c.want, step.Check, texts[c.starter])
			}
		}
	}
}

func findLesson(cat tutor.Catalog, id string) (tutor.Lesson, bool) {
	for _, r := range cat.Routes {
		for _, les := range r.Lessons {
			if les.ID == id {
				return les, true
			}
		}
	}
	return tutor.Lesson{}, false
}
