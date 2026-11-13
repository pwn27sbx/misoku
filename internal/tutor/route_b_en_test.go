package tutor_test

import (
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/tutor"
)

// noteEdit turns a lesson's text into what the player's note holds.
type noteEdit func(string) string

// addLines writes lines in the practice area, before the lesson's "Next:"
// line, as the player does on new lines.
func addLines(lines ...string) noteEdit {
	return func(text string) string {
		i := strings.LastIndex(text, "\nNext: ")
		if i < 0 {
			return text + "\n" + strings.Join(lines, "\n") + "\n"
		}
		return text[:i] + "\n" + strings.Join(lines, "\n") + "\n" + text[i:]
	}
}

// replaceLine rewrites the line that is exactly old; an empty new deletes it.
func replaceLine(old, new string) noteEdit {
	return func(text string) string {
		lines := strings.Split(text, "\n")
		for i, l := range lines {
			if l == old {
				if new == "" {
					return strings.Join(append(lines[:i:i], lines[i+1:]...), "\n")
				}
				lines[i] = new
				return strings.Join(lines, "\n")
			}
		}
		return text + "\n<no line " + strconv.Quote(old) + ">"
	}
}

func applyEdits(text string, edits []noteEdit) string {
	for _, e := range edits {
		text = e(text)
	}
	return text
}

// textStep is one text step of the English Basic route: the note before
// the player does it (untouched when nil) and the note once done.
type textStep struct {
	before, done []noteEdit
}

// diskStep is one disk path step: the vault before and after the action.
type diskStep struct {
	before, done diskPlace
}

// englishRouteB is the English Basic route and its embedded notes.
func englishRouteB(t *testing.T) (tutor.Route, fs.FS) {
	t.Helper()
	cat, err := tutor.BundledRoutesFor(i18n.EN)
	if err != nil {
		t.Fatalf("the English catalog is invalid: %v", err)
	}
	r, ok := cat.RouteByID("B")
	if !ok || r.Dir != "1 Basic" {
		t.Fatalf("no English route B in 1 Basic (got %q)", r.Dir)
	}
	return r, tutor.BundledFor(i18n.EN)
}

// T6/S5: every text and disk path step of the English Basic route passes
// once the player does what the lesson asks, and not before. Each entry is
// the action the step's text describes.
func TestEnglishRouteBStepsPassOnlyWhenDone(t *testing.T) {
	texts := map[string]textStep{
		"B01#1": {done: []noteEdit{addLines("Today I start my travel journal")}},
		"B02#1": {done: []noteEdit{replaceLine("cat", "cats")}},
		"B02#2": {done: []noteEdit{replaceLine("cat", "cats · suitcase")}},
		"B02#3": {done: []noteEdit{replaceLine("cat", "01 · cats")}},
		"B02#4": {done: []noteEdit{addLines("packing list")}},
		"B02#5": {done: []noteEdit{addLines("— flight —")}},
		"B03#1": {done: []noteEdit{replaceLine("hottel", "hotel")}},
		"B03#2": {
			before: []noteEdit{replaceLine("hottel", "hotel")},
			done:   []noteEdit{replaceLine("hottel", "hotel"), replaceLine("draft", "")},
		},
		// u after dd brings the line back: the step is the deletion.
		"B05#1": {done: []noteEdit{replaceLine("delete me", "")}},
		"B05#2": {done: []noteEdit{addLines("oops")}},
		"B12#2": {done: []noteEdit{addLines("> a quote")}},
		"B13#1": {done: []noteEdit{addLines("# Trip")}},
		"B13#2": {
			before: []noteEdit{addLines("# Trip")},
			done:   []noteEdit{addLines("# Trip", "## Luggage")},
		},
		"B13#3": {
			before: []noteEdit{addLines("# Trip", "## Luggage")},
			done:   []noteEdit{addLines("# Trip", "## Luggage", "### Day 1")},
		},
		"B13#4": {
			before: []noteEdit{addLines("# Trip", "## Luggage", "### Day 1")},
			done:   []noteEdit{addLines("# Trip", "## Luggage", "### Day 1", "#### Detail")},
		},
		"B14#1": {done: []noteEdit{addLines("- milk")}},
		"B14#2": {done: []noteEdit{addLines("1. passport")}},
		"B14#3": {done: []noteEdit{addLines("- [ ] tickets")}},
		"B14#4": {
			before: []noteEdit{addLines("- milk")},
			done:   []noteEdit{addLines("  - milk")},
		},
		"B15#1": {done: []noteEdit{replaceLine("- Buy bread", "- [x] Buy bread")}},
		"B15#2": {done: []noteEdit{replaceLine("- [ ] Call the hotel", "- [x] Call the hotel")}},
		"B15#3": {
			before: []noteEdit{
				replaceLine("- [ ] Book the museum", "- [x] Book the museum"),
				replaceLine("- [ ] Pack the suitcase", "- [x] Pack the suitcase"),
			},
			done: []noteEdit{
				replaceLine("- [ ] Book the museum", "- [x] Book the museum"),
				replaceLine("- [ ] Pack the suitcase", "- [x] Pack the suitcase"),
				replaceLine("- [ ] Charge the phone", "- [x] Charge the phone"),
			},
		},
		"B15#4": {done: []noteEdit{replaceLine("- [ ] Water the plants", "- [x] Water the plants")}},
		"B16#1": {done: []noteEdit{replaceLine("- [x] Book the museum", "- Book the museum")}},
		"B16#2": {done: []noteEdit{replaceLine("- [ ] Water the plants", "- [x] Water the plants")}},
		"B17#1": {done: []noteEdit{addLines("**book hotel**")}},
		"B17#2": {done: []noteEdit{addLines("*beach*")}},
		"B17#3": {done: []noteEdit{addLines("`misoku`")}},
		"B17#4": {done: []noteEdit{addLines("~~cancelled~~")}},
		"B18#1": {done: []noteEdit{addLines("> a quote")}},
		"B18#2": {done: []noteEdit{addLines("one", "---", "two")}},
		"B19#1": {done: []noteEdit{addLines("[[Weekly shopping]]")}},
		"B19#2": {done: []noteEdit{addLines("[[Market]]")}},
	}
	disks := map[string]diskStep{
		// :w travel-list saves at the vault root.
		"B06#1": {done: diskPlace{notes: map[string]string{"travel-list": "x"}}},
		// n in the explorer creates inside the lesson's route folder.
		"B09#2": {done: diskPlace{notes: map[string]string{"1 Basic/groceries": "x"}}},
		"B09#3": {done: diskPlace{folders: []string{"1 Basic/ideas"}}},
		"B10#1": {
			before: diskPlace{notes: map[string]string{"1 Basic/groceries": "x"}},
			done:   diskPlace{notes: map[string]string{"1 Basic/market": "x"}},
		},
		"B10#2": {
			before: diskPlace{notes: map[string]string{"1 Basic/market": "x"}, folders: []string{"1 Basic/ideas"}},
			done:   diskPlace{folders: []string{"1 Basic/ideas"}},
		},
	}

	r, notes := englishRouteB(t)
	used := map[string]bool{}
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(notes, l.Path())
		if err != nil {
			t.Fatalf("%s: %v", l.Path(), err)
		}
		lesson := string(data)
		for i, st := range l.Steps {
			step := i + 1
			key := tutor.StepKey(l.ID, step)
			passes := func(text string, env tutor.Env) bool {
				s := &tutor.Store{}
				s.Recheck(l, text, env)
				return s.Passed(l, step)
			}
			switch kind := stepKind(st.Check); kind {
			case "text":
				c, ok := texts[key]
				if !ok {
					t.Errorf("%s is a text step with no action in the table", key)
					continue
				}
				used[key] = true
				if passes(applyEdits(lesson, c.before), tutor.Env{}) {
					t.Errorf("%s passes before the player does it", key)
				}
				if !passes(applyEdits(lesson, c.done), tutor.Env{}) {
					t.Errorf("%s does not pass once done:\n%s", key, applyEdits(lesson, c.done))
				}
			case "disk":
				c, ok := disks[key]
				if !ok {
					t.Errorf("%s is a disk path step with no place in the table", key)
					continue
				}
				used[key] = true
				if passes("", placeEnv(c.before)) {
					t.Errorf("%s passes before the player does it", key)
				}
				if !passes("", placeEnv(c.done)) {
					t.Errorf("%s does not pass where the action puts it", key)
				}
			}
		}
	}
	for key := range texts {
		if !used[key] {
			t.Errorf("the table has %s, which is not a text step", key)
		}
	}
	for key := range disks {
		if !used[key] {
			t.Errorf("the table has %s, which is not a disk path step", key)
		}
	}
}

// stepKind is "text" for a step made only of text checks, "disk" for one
// made only of disk path checks, and "" for the rest (actions, saves, none).
func stepKind(c tutor.Check) string {
	var kinds []string
	var walk func(tutor.Check)
	walk = func(c tutor.Check) {
		switch c.Kind {
		case tutor.KindAll:
			for _, sub := range c.Checks {
				walk(sub)
			}
		case tutor.KindText, tutor.KindTextRaw:
			kinds = append(kinds, "text")
		case tutor.KindDisk:
			if c.Pred == "file-changed" {
				kinds = append(kinds, "")
			} else {
				kinds = append(kinds, "disk")
			}
		default:
			kinds = append(kinds, "")
		}
	}
	walk(c)
	for _, k := range kinds[1:] {
		if k != kinds[0] {
			return ""
		}
	}
	return kinds[0]
}

func placeEnv(p diskPlace) tutor.Env {
	folders := map[string]bool{}
	for _, f := range p.folders {
		folders[f] = true
	}
	notes := p.notes
	if notes == nil {
		notes = map[string]string{}
	}
	env, _ := vaultEnv("", notes, folders)
	return env
}

var inlineCodeRe = regexp.MustCompile("`[^`]*`")

// practiceArea is the lesson's practice text outside inline code: what the
// player's own search runs into, rather than the examples.
func practiceArea(lesson string) string {
	_, area, _ := strings.Cut(lesson, "## Your practice area")
	area, _, _ = strings.Cut(area, "\nNext: ")
	return inlineCodeRe.ReplaceAllString(area, "")
}

// T6/S5: an action step whose value is something the player types (a search,
// a link's target) is typed as the lesson shows it, and a search finds its
// match in the practice area itself, with Misoku's smartcase: a query with
// an upper-case letter is case-sensitive.
func TestEnglishRouteBTypedValuesAreInTheLesson(t *testing.T) {
	r, notes := englishRouteB(t)
	checked := 0
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(notes, l.Path())
		if err != nil {
			t.Fatalf("%s: %v", l.Path(), err)
		}
		lesson := string(data)
		for i, st := range l.Steps {
			c := st.Check
			if c.Kind != tutor.KindAction || len(c.Value) == 0 {
				continue
			}
			key := tutor.StepKey(l.ID, i+1)
			switch c.Event {
			case "new_draft":
				checked++
				if !strings.Contains(lesson, "`[["+c.Value[0]+"]]`") {
					t.Errorf("%s waits for a draft of %q, which the lesson never asks to type", key, c.Value[0])
				}
			case "search_hit":
				checked++
				q := c.Value[0]
				if !strings.Contains(lesson, "`"+q+"`") {
					t.Errorf("%s waits for a search of %q, which the lesson never asks to type", key, q)
				}
				area := practiceArea(lesson)
				if strings.ToLower(q) == q {
					area, q = strings.ToLower(area), strings.ToLower(q)
				}
				if !strings.Contains(area, q) {
					t.Errorf("%s: a search of %q finds nothing in the practice area, only in the examples", key, c.Value[0])
				}
			}
		}
	}
	if checked != 3 {
		t.Errorf("checked %d typed values, want 3 (B20#5, B22#1 and B22#3)", checked)
	}
}

// slashIDs are the ids of the "/" menu's items (internal/content/slash.go).
var slashIDs = []string{
	"format", "h1", "h2", "h3", "h4", "bullet", "numbered", "todo", "toggle", "page",
	"callout", "quote", "table", "equation", "divider", "columns2", "columns3",
	"columns4", "link", "template", "board", "calendar", "timeline", "chart",
	"form", "flowchart", "cover", "image", "paste", "video", "audio", "file", "code",
}

var slashQuoteRe = regexp.MustCompile("`/([A-Z][^`]*)`|(?:^|\\s)/([A-Z][A-Za-z0-9 -]*[A-Za-z0-9])")

// T6/S7: the English Basic route quotes the English interface: every "/"
// label is a label of the English "/" menu, and the bar, dialog and search
// texts the lessons quote are the English catalog's.
func TestEnglishRouteBQuotesTheEnglishInterface(t *testing.T) {
	labels := map[string]bool{}
	for _, id := range slashIDs {
		labels[i18n.TFor(i18n.EN, i18n.SlashLabel(id))] = true
	}
	r, notes := englishRouteB(t)
	texts := map[string]string{}
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(notes, l.Path())
		if err != nil {
			t.Fatalf("%s: %v", l.Path(), err)
		}
		texts[l.ID] = string(data)
		for i, st := range l.Steps {
			texts[tutor.StepKey(l.ID, i+1)] = st.Text + "\n" + st.Hint
		}
	}
	index, err := fs.ReadFile(notes, r.IndexPath())
	if err != nil {
		t.Fatal(err)
	}
	texts["B00"] = string(index)

	quoted := 0
	for where, text := range texts {
		for _, m := range slashQuoteRe.FindAllStringSubmatch(text, -1) {
			label := m[1] + m[2]
			quoted++
			if !labels[label] {
				t.Errorf("%s quotes /%s, which is not an English / menu label", where, label)
			}
		}
	}
	if quoted == 0 {
		t.Error("no / label found: the pattern is broken")
	}

	en := func(key i18n.Key, args ...any) string { return i18n.TFor(i18n.EN, key, args...) }
	passed, _, _ := strings.Cut(en(i18n.TutorLessonPassed, "B03", "B04"), " · ")
	notFound, _, _ := strings.Cut(en(i18n.EditorPatternNotFound, "zzz"), ": zzz")
	want := map[string][]string{
		"B00": {
			"`" + strings.TrimSuffix(en(i18n.TutorStepLine, "B03", 2, 4, "…"), "…") + "…`",
			"`" + en(i18n.TutorStepFlash, 2) + "`",
			"`" + passed + "`",
			"`" + en(i18n.TutorRouteDone, r.Name) + "`",
			"**" + en(i18n.WelcomeNew) + "**",
			"**" + en(i18n.WelcomeExperienced) + "**",
			"**" + en(i18n.WelcomeLater) + "**",
			"`" + en(i18n.TutorAfterTutorial) + ".md`",
		},
		"B22": {"`" + notFound + "`", "`" + en(i18n.EditorSearchWrappedTop) + "`"},
		"B25": {"`" + en(i18n.AppSaveAll) + "`", "`" + en(i18n.AppDiscard) + "`", "`" + en(i18n.AppCancel) + "`"},
	}
	for id, quotes := range want {
		for _, q := range quotes {
			if !strings.Contains(strings.ToLower(texts[id]), strings.ToLower(q)) {
				t.Errorf("%s does not quote the English interface's %q", id, q)
			}
		}
	}
}
