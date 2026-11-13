package tutor_test

import (
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/tutor"
)

// iState is the note and the vault after the player does one step of the
// English route I: text is what the player has added to the lesson note so
// far, and the rest is what the step's action leaves on disk or in the
// configuration.
type iState struct {
	text    string
	notes   map[string]string // vault-relative note name (no ".md") -> text
	folders []string
	today   string
	changed bool     // the lesson note was saved with changes (:w)
	config  []string // configuration keys changed in :settings
}

// iTable is a Markdown table as /Table inserts it, filled by the player.
func iTable(rows ...string) string { return strings.Join(rows, "\n") + "\n" }

// iStates is, for every English route I step that a text, disk or
// configuration check credits, the state right after the player does it the
// way the lesson says. Steps credited only by an app event are left out:
// their values are language-neutral, or named in the lesson (see
// TestRouteIEnglishActionValuesAreInTheLesson).
func iStates() map[string]iState {
	board := starters(i18n.EN)["board"]
	columns := func(n int) string {
		s := "~~~columns\nOne side"
		for i := 2; i <= n; i++ {
			s += "\n+++\n" + i18n.TFor(i18n.EN, i18n.SlashColumnN, i)
		}
		return s + "\n~~~\n"
	}
	calendar := func(event string) string {
		return "```calendar\nmonth: 2026-11\n2026-11-13: " + event + "\n```\n"
	}
	return map[string]iState{
		"I02#4": {text: "wind and \nlighthouse and wave\n"},
		"I03#2": {text: "beach list\nbeach list\n"},
		"I04#2": {changed: true},
		"I05#1": {text: "<details>\n<summary>Packing</summary>\n\nTowel\n</details>\n"},
		"I05#2": {text: "> [!NOTE]\n> Bring water.\n"},
		"I05#3": {text: "> [!NOTE]\n> Bring water.\n\n> [!TIP]\n> Go early.\n\n> [!WARNING]\n> Strong sun.\n"},
		// /Page and s write the sub-note in the folder named after Trip,
		// inside the route folder.
		"I06#1": {notes: map[string]string{"2 Intermediate/Trip/Beach packing": "x"}},
		"I06#2": {notes: map[string]string{"2 Intermediate/Trip/Beach packing": "x", "2 Intermediate/Trip/Day 1": "x"}},
		"I07#1": {text: iTable("| Day | Column 2 |", "| --- | --- |", "|  |  |")},
		"I08#1": {text: iTable("| Day | Column 2 |", "| --- | --- |", "|  |  |", "|  |  |")},
		"I08#2": {text: iTable("| Day | Column 2 |  |", "| --- | --- | --- |", "|  |  |  |", "|  |  |  |")},
		"I08#3": {text: iTable("| Day | Column 2 |  |", "| --- | --- | --- |", "|  |  |  |")},
		"I08#4": {text: iTable("| Day | Column 2 |", "| --- | --- |", "|  |  |")},
		"I09#1": {text: iTable("| Day       | Plan |", "| --------- | ---- |", "| Saturday  | Sea  |")},
		"I09#2": {text: iTable("| Day       | Plan  |", "| --------- | ----- |", "| Saturday  | Beach |")},
		"I10#1": {text: iTable("| Day | Column 2 |", "| --- | --- |", "| Beach |  |")},
		"I10#2": {text: iTable("| Day | Column 2 |  |", "| --- | --- | --- |", "| Beach |  |  |", "|  |  |  |")},
		"I10#3": {text: iTable("| Day | Cost |  |", "| --- | --- | --- |", "| Beach |  |  |", "|  |  |  |")},
		"I11#1": {text: columns(2)},
		"I11#2": {text: columns(2) + "\n" + columns(3)},
		"I11#3": {text: columns(2) + "\n" + columns(3) + "\n" + columns(4)},
		"I12#1": {text: "```math\na^2 + b^2 = c^2\n```\n"},
		"I12#2": {text: "```math\na^2 + b^2 = c^2\n```\n\n$$\ne^{i\\pi} + 1 = 0\n$$\n"},
		"I13#1": {text: board},
		"I13#2": {text: strings.Replace(board, "## Not started\n- ", "## Not started\n- Card 1\n- Backpack", 1)},
		"I13#3": {text: strings.Replace(board, "## Not started\n- ", "## Not started\n- Card 1\n- Boots", 1)},
		"I13#4": {text: strings.Replace(board, "## Done", "## Finished", 1)},
		"I14#1": {text: calendar("")},
		"I14#2": {text: calendar("Beach")},
		"I14#3": {text: calendar("Museum")},
		"I15#1": {text: starters(i18n.EN)["timeline"]},
		"I15#2": {text: strings.Replace(starters(i18n.EN)["timeline"], "```\n", "Trip: 2026-01-03 -> 2026-01-04\n```\n", 1)},
		"I15#3": {text: strings.Replace(starters(i18n.EN)["timeline"], "```\n", "Coast: 2026-01-03 -> 2026-01-04\n```\n", 1)},
		"I16#1": {text: starters(i18n.EN)["chart"]},
		"I16#2": {text: strings.Replace(starters(i18n.EN)["chart"], "```\n", "Beach: 3\n```\n", 1)},
		"I16#3": {text: strings.Replace(starters(i18n.EN)["chart"], "```\n", "Museum: 4\n```\n", 1)},
		"I17#1": {text: "```\necho hello\n```\n"},
		"I17#2": {text: "```\necho hello\n```\n\n```flow\nStart --> End\n```\n"},
		"I18#1": {text: "![photo](assets/photo.png)\n"},
		"I18#2": {text: "![photo](assets/photo.png)\n\n![clipboard](assets/clipboard.png)\n"},
		// The copy lands in assets/ next to the note, in the route folder.
		"I18#4": {folders: []string{"2 Intermediate/assets"}},
		"I19#1": {text: "[▶ walk](assets/walk.mp4)\n"},
		"I19#2": {text: "[▶ walk](assets/walk.mp4)\n[♪ waves](assets/waves.mp3)\n"},
		"I19#3": {text: "[▶ walk](assets/walk.mp4)\n[♪ waves](assets/waves.mp3)\n[📎 tickets](assets/tickets.pdf)\n"},
		"I20#2": {text: "## Packing\n\nSee [[#Packing]]\n"},
		"I20#4": {
			text:  "## Packing\n\nSee [[#Packing]]\nAlso [[Trip#Packing]]\n",
			notes: map[string]string{"2 Intermediate/Plan": "[[Trip#Packing]]"},
		},
		"I21#1": {text: "We hold the room until Friday. ^booking\n"},
		"I21#2": {text: "We hold the room until Friday. ^booking\n\n[[#^booking]]\n"},
		"I21#3": {
			text:  "We hold the room until Friday. ^booking\n\n[[#^booking]]\n[[Trip#^booking]]\n",
			notes: map[string]string{"2 Intermediate/Trip": "Ferry at nine. ^booking"},
		},
		// A wikilink resolves by the note's name, wherever it lives.
		"I22#1": {text: "[[Trip|my trip]]\n", notes: map[string]string{"Trip": "x"}},
		"I22#2": {config: []string{"ui.link_suggestions"}},
		"I23#1": {text: "[Trip](Trip.md)\n", notes: map[string]string{"2 Intermediate/Trip": "x"}},
		"I23#2": {text: "[Trip](Trip.md)\n[my note](Trip.md)\n", notes: map[string]string{"2 Intermediate/Trip": "x"}},
		"I23#3": {text: "[documentation](https://example.com)\n<https://example.com>\n"},
		"I24#1": {text: "Off to the sea #travel\n"},
		"I24#2": {text: "Off to the sea #travel\nSand #travel/beach\n"},
		"I24#3": {text: "---\ntags: beach, sun\n---\nOff to the sea #travel\nSand #travel/beach\n"},
		"I24#4": {text: "---\ntags: [beach, sun]\n---\nOff to the sea #travel\nSand #travel/beach\n"},
		"I24#5": {text: "---\ntags:\n  - beach\n  - sun\n---\nOff to the sea #travel\nSand #travel/beach\n"},
		// Templates and the daily note live at the vault root, as configured.
		"I25#5": {notes: map[string]string{"Plantillas/daily": "Day plan"}},
		"I27#1": {text: "---\nstage: in progress\n---\n", notes: map[string]string{"Diario/2026-11-13": "x"}, today: "2026-11-13"},
		"I27#2": {
			text:    "---\nstage: in progress\n---\n",
			notes:   map[string]string{"Plantillas/daily": "Day plan\nNo rush"},
			folders: []string{"Plantillas"},
		},
		"I27#3": {text: "---\nstage: in progress\n---\nDay plan\nNo rush\n"},
		"I27#4": {text: "---\nstage: in progress\n---\nDay plan\nNo rush\nDay plan\nNo rush\nDate: 2026-11-13\nNote: I27 Today and templates\n"},
		"I27#5": {text: "---\nstage: in progress\nplace: beach\n---\nDay plan\nNo rush\nDate: 2026-11-13\nNote: I27 Today and templates\n"},
		"I28#2": {config: []string{"ui.accordion"}},
	}
}

// passesWithTheStepBefore are the English route I steps whose check already
// holds on the previous step's work, so the app credits them early. Their
// checks cannot tell the two states apart with the predicates the shape test
// pins to the Spanish route, which has the same gap; the entry names why.
var passesWithTheStepBefore = map[string]string{
	"I08#3": "n-lines-start counts at least three table lines, and the table has four before :delrow",
	"I18#2": "the /Image line of step 1 already starts with ![ and has ](; a pasted image has no fixed name",
}

// env is the vault the state describes, for a lesson of the route folder.
func (s iState) env() tutor.Env {
	notes := map[string]string{}
	for k, v := range s.notes {
		notes[k] = v
	}
	folders := map[string]bool{}
	for _, f := range s.folders {
		folders[f] = true
	}
	env, _ := vaultEnv("", notes, folders)
	env.Today = s.today
	env.Changed = func(string) bool { return s.changed }
	env.ConfigChanged = func(key string) bool {
		for _, k := range s.config {
			if k == key {
				return true
			}
		}
		return false
	}
	return env
}

// passes reports whether step (1-based) of l is credited when the lesson
// note holds its own text plus the state's, the vault is the state's, and
// every app event the step also waits for has happened.
func passes(l tutor.Lesson, lesson string, step int, s iState) bool {
	store := &tutor.Store{}
	store.Recheck(l, lesson+"\n"+s.text, s.env())
	var record func(c tutor.Check)
	record = func(c tutor.Check) {
		switch c.Kind {
		case tutor.KindAll:
			for _, sub := range c.Checks {
				record(sub)
			}
		case tutor.KindAction:
			if len(c.Value) == 0 {
				store.RecordAction(l, c.Event, "")
			}
			for _, v := range c.Value {
				store.RecordAction(l, c.Event, v)
			}
		}
	}
	record(l.Steps[step-1].Check)
	return store.Passed(l, step)
}

// hasStateCheck reports whether c has a leaf that reads the note, the vault
// or the configuration, instead of only app events.
func hasStateCheck(c tutor.Check) bool {
	switch c.Kind {
	case tutor.KindAll:
		for _, sub := range c.Checks {
			if hasStateCheck(sub) {
				return true
			}
		}
	case tutor.KindText, tutor.KindTextRaw, tutor.KindTextFence, tutor.KindDisk:
		return true
	}
	return false
}

func englishRouteI(t *testing.T) (tutor.Route, fs.FS) {
	t.Helper()
	cat, err := tutor.BundledRoutesFor(i18n.EN)
	if err != nil {
		t.Fatalf("the English catalog is invalid: %v", err)
	}
	r, ok := cat.RouteByID("I")
	if !ok {
		t.Fatal("no English route I")
	}
	return r, tutor.BundledFor(i18n.EN)
}

// S5: played in order, every English route I step that reads the note, the
// vault or the configuration passes once the player does it the way the
// lesson says, and not one step earlier: on the state the previous step
// left (the untouched lesson for the first one).
func TestRouteIEnglishStepsPassOnlyWhenDone(t *testing.T) {
	r, bundled := englishRouteI(t)
	states := iStates()
	used := map[string]bool{}
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(bundled, l.Path())
		if err != nil {
			t.Fatalf("%s not embedded: %v", l.Path(), err)
		}
		lesson := string(data)
		before := iState{}
		for i, st := range l.Steps {
			key := l.ID + "#" + strconv.Itoa(i+1)
			if !hasStateCheck(st.Check) {
				continue
			}
			after, ok := states[key]
			if !ok {
				t.Errorf("%s reads the note or the vault and has no state in iStates", key)
				continue
			}
			used[key] = true
			if !passes(l, lesson, i+1, after) {
				t.Errorf("%s (%q) does not pass once done\n%s", key, st.Text, after.text)
			}
			early := passes(l, lesson, i+1, before)
			switch why, known := passesWithTheStepBefore[key]; {
			case early && !known:
				t.Errorf("%s (%q) already passes before the player does it\n%s", key, st.Text, before.text)
			case !early && known:
				t.Errorf("%s no longer passes early: remove it from passesWithTheStepBefore", key)
			case early:
				t.Logf("%s passes with the step before: %s", key, why)
			}
			before = after
		}
	}
	for key := range states {
		if !used[key] {
			t.Errorf("iStates has %s, which reads neither the note nor the vault", key)
		}
	}
}

// S5: the names an English route I action check waits for are the ones its
// lesson asks the player to open, type or create.
func TestRouteIEnglishActionValuesAreInTheLesson(t *testing.T) {
	named := map[string]bool{
		"note_opened": true, "new_draft": true, "links_headings": true,
		"finder_filter": true, "tree_refresh": true, "palette_run": true,
	}
	generic := map[string]bool{"new-tab": true, "tag:": true}
	r, bundled := englishRouteI(t)
	checked := 0
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(bundled, l.Path())
		if err != nil {
			t.Fatalf("%s not embedded: %v", l.Path(), err)
		}
		var walk func(c tutor.Check)
		walk = func(c tutor.Check) {
			for _, sub := range c.Checks {
				walk(sub)
			}
			if c.Kind != tutor.KindAction || !named[c.Event] {
				return
			}
			for _, v := range c.Value {
				if generic[v] {
					continue
				}
				checked++
				if !strings.Contains(string(data), v) {
					t.Errorf("%s waits for %s %q, which the lesson never names", l.ID, c.Event, v)
				}
			}
		}
		for _, st := range l.Steps {
			walk(st.Check)
		}
	}
	if checked < 8 {
		t.Errorf("checked %d named action values, want at least 8", checked)
	}
}

// slashIDs are the "/" menu items, by the ids internal/content gives them.
var slashIDs = []string{
	"format", "h1", "h2", "h3", "h4", "bullet", "numbered", "todo", "toggle",
	"page", "callout", "quote", "table", "equation", "divider", "columns2",
	"columns3", "columns4", "link", "template", "board", "calendar",
	"timeline", "chart", "form", "flowchart", "cover", "image", "paste",
	"video", "audio", "file", "code",
}

var (
	quotedRe = regexp.MustCompile("`([^`]+)`")
	slashRe  = regexp.MustCompile("(^|[\\s`])/([A-Z0-9][^/]*)")
)

// S7: every UI label the English route I quotes, in its lessons and in its
// steps, is the English text the app shows: "/" menu items, the preview's
// buttons and the :settings rows and values.
func TestRouteIEnglishQuotesTheEnglishLabels(t *testing.T) {
	en := func(k i18n.Key) string { return i18n.TFor(i18n.EN, k) }
	var slash []string
	for _, id := range slashIDs {
		slash = append(slash, en(i18n.SlashLabel(id)))
	}
	buttons := map[string]bool{}
	for _, k := range []i18n.Key{
		i18n.PreviewAddRow, i18n.PreviewAddColumn, i18n.PreviewNewPage,
		i18n.PreviewAddValue, i18n.PreviewTimelineNew, i18n.PreviewToday,
	} {
		buttons[en(k)] = true
	}
	settings := []string{
		en(i18n.SettingAccordion), en(i18n.SettingLinkSuggestions),
		en(i18n.LinkSuggestionsInline), en(i18n.LinkSuggestionsFinder),
	}

	// isSlash reports whether s starts with a "/" label, whole words.
	isSlash := func(s string) bool {
		for _, label := range slash {
			rest, ok := strings.CutPrefix(s, label)
			if ok && (rest == "" || !unicode.IsLetter([]rune(rest)[0]) && !unicode.IsDigit([]rune(rest)[0])) {
				return true
			}
		}
		return false
	}
	r, bundled := englishRouteI(t)
	texts := map[string]string{} // where -> text
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(bundled, l.Path())
		if err != nil {
			t.Fatalf("%s not embedded: %v", l.Path(), err)
		}
		texts[l.Path()] = string(data)
		for i, st := range l.Steps {
			texts[l.ID+"#"+strconv.Itoa(i+1)] = st.Text + "\n" + st.Hint
		}
	}
	slashes := 0
	for where, text := range texts {
		for _, line := range strings.Split(text, "\n") {
			for _, m := range slashRe.FindAllStringSubmatch(line, -1) {
				slashes++
				if !isSlash(m[2]) {
					t.Errorf("%s quotes /%s, which is no English \"/\" menu item", where, m[2])
				}
			}
		}
		for _, m := range quotedRe.FindAllStringSubmatch(text, -1) {
			q := m[1]
			if (strings.HasPrefix(q, "+ ") || strings.ContainsAny(q, "‹›⊕")) && !buttons[q] {
				t.Errorf("%s quotes the button `%s`, which the English preview does not show", where, q)
			}
		}
	}
	if slashes < 25 {
		t.Errorf("found %d \"/\" labels, want at least 25", slashes)
	}
	for file, label := range map[string]string{
		"I20": settings[1], "I22": settings[1], "I28": settings[0],
	} {
		for where, text := range texts {
			if strings.Contains(where, "/"+file+" ") && !strings.Contains(text, label) {
				t.Errorf("%s does not name the :settings row %q", where, label)
			}
		}
	}
	for _, v := range settings[2:] {
		if !strings.Contains(texts[r.Dir+"/I22 Aliases and the suggestion mode.md"], v) {
			t.Errorf("I22 does not name the Link suggestions value %q", v)
		}
	}
}
