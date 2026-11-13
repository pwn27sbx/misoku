package tutor_test

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/tutor"
)

// enAStep is what the player does for one step of the English route A, the
// way its lesson asks: the lines typed into the lesson note, the notes the
// action leaves in the vault (vault-relative, no ".md") and the events the
// app records while doing it.
type enAStep struct {
	text   string
	notes  map[string]string
	events [][2]string
	today  string
}

// typing is the finder_filter events of typing q: the app reports the raw
// query on every change, so every prefix is an event.
func typing(q string) [][2]string {
	var ev [][2]string
	for i := 1; i <= len(q); i++ {
		ev = append(ev, [2]string{"finder_filter", q[:i]})
	}
	return ev
}

func events(event string, values ...string) [][2]string {
	var ev [][2]string
	for _, v := range values {
		ev = append(ev, [2]string{event, v})
	}
	return ev
}

// formStarter is the block the "/Form" item inserts in English.
func formStarter() string { return starters(i18n.EN)["form"] }

// enRouteA is the action of every checked step of the English route A.
// Steps with no check (A19's sync steps, A21) are credited by a save and
// are not listed.
var enRouteA = map[string]enAStep{
	"A01#1": {text: "##### Budget\n###### Detail"},
	"A01#2": {text: "> Plan\n>> Inside the plan"},
	"A01#3": {text: "<!-- to be confirmed -->"},
	"A02#1": {text: "---\ntrip: coast\nstage: planned\n---"},
	"A02#2": {events: events("preview_rendered", "properties")},
	"A02#3": {events: events("preview_rendered", "properties")},
	"A03#1": {text: "---\ntrip: mountains\n---"},
	"A03#2": {text: "---\ntags: [beach, sun, hiking]\n---"},
	"A04#1": {text: "---\nstatus: active\n---"},
	"A04#2": {events: events("status_glyph", "active", "onhold", "completed", "dropped")},
	"A04#3": {events: events("status_glyph", "none")},
	"A05#1": {events: events("finder_filter", "status:")}, // :status opens the finder with it
	"A05#2": {events: typing("status:active")},
	"A05#3": {events: append(typing("folder:Travel"), typing(`folder:"My notes"`)...)},
	"A05#4": {events: typing("status:active folder:Travel beach")},
	"A06#1": {events: events("history_open", "commits")},
	"A06#2": {events: events("history_preview", "")},
	"A06#3": {text: "---\nbudget: 900\n---"},
	"A06#4": {events: events("history_error", "")},
	"A07#1": {events: events("backlinks", "")},
	"A07#2": {events: events("mention_open", "")},
	// The [ link ] button rewrites the mention inside Idea, which the
	// player created in the route folder (space n with the lesson open).
	"A07#3": {notes: map[string]string{"3 Advanced/Idea": "An idea for [[A07 Mentions and backlinks]]"}},
	"A07#4": {events: events("backlink_open", "")},
	// r renames Suitcase and rewrites the link the lesson note has.
	"A08#1": {notes: map[string]string{"3 Advanced/A08 Renaming without breaking links": "See [[Luggage]]"}},
	"A08#2": {text: "[[Home/Suitcase]]"},
	"A09#1": {text: "![[Trip]]"},
	"A09#2": {text: "![[Trip#Luggage]]"},
	"A09#3": {text: "![[Trip#^booking]]"},
	"A09#4": {text: "![[mapa.png]]"},
	"A10#1": {text: "[menu](menu.pdf)"},
	"A10#2": {events: events("link_confirm", "Cancel")},
	"A11#1": {events: events("graph_open", "local")},
	"A11#2": {events: events("graph_move", "hl", "jk", "enter", "zoom", "center")},
	"A11#3": {events: events("graph_mouse", "")},
	"A11#4": {events: events("graph_refresh", "")},
	"A12#1": {events: events("graph_open", "global")},
	// A08 left two notes called Suitcase, so the hollow node is a new name.
	"A12#2": {events: events("new_draft", "Backpack")},
	// :export and space P write the PDF next to the note, in the route folder.
	"A13#1": {notes: map[string]string{"3 Advanced/A13 Export to PDF.pdf": "pdf"}},
	"A13#2": {notes: map[string]string{"3 Advanced/report.pdf": "pdf"}},
	"A13#3": {notes: map[string]string{"3 Advanced/A00 Advanced.pdf": "pdf"}},
	"A14#1": {text: "---\ncover: cover.png\n---"},
	"A14#2": {events: events("preview_rendered", "cover")},
	"A15#1": {text: formStarter()},
	"A15#2": {text: strings.Replace(formStarter(), "? Question 1\n", "? Question 1\n= 3 days\n", 1)},
	"A15#3": {text: strings.Replace(formStarter(), "? Question 2 (choice): Option 1 | Option 2 | Option 3\n",
		"? Question 2 (date)\n= 2026-11-13\n", 1), today: "2026-11-13"},
	"A16#1": {text: "```form\ntitle: Trip\n? Where do you sleep? (choice): Cabin | Hotel | Apartment\n= Cabin\n```"},
	"A16#2": {text: "```form\ntitle: Trip\n? How do you get there? (multi): Bike | Train | Car\n= Bike, Train\n```"},
	"A16#3": {text: "```form\ntitle: Trip\n? How was it? (rating)\n= 3\n```"},
	"A17#1": {text: "```form\ntitle: Trip\n? Luggage (long) *\n```"},
	"A17#2": {text: "```form\ntitle: Trip\n? Transfer (date)\n```"},
	"A17#3": {text: "```form\ntitle: Trip to the coast\ndescription: Back to the village\n```"},
	"A18#1": {text: "Total: $900"},
	"A18#2": {text: "Dinner: $45"},
	"A18#3": {events: events("block_handles", "visible")},
	"A19#2": {events: events("external_change", "confirm")},
	"A20#1": {events: events("dirbuf_open", "")},
	// The folder editor (m) and mini.files (M) create in the selected
	// folder, the lesson's route folder.
	"A20#2": {notes: map[string]string{"3 Advanced/Notes/luggage": "x"}},
	"A20#3": {events: events("dirbuf_open", "i", "a", "o")},
	"A20#4": {notes: map[string]string{"3 Advanced/practice-mini": "x"}},
}

// sameGesture are later steps that one step's action also passes, because
// both ask for the same gesture: A02's table shows the properties without
// cover and icon in one render.
var sameGesture = map[string]bool{"A02#2>A02#3": true}

// englishVault is the installed English tutorial: every embedded file, notes
// without ".md", as the vault holds them before the player does anything.
func englishVault(t *testing.T) map[string]string {
	t.Helper()
	vault := map[string]string{}
	bundled := tutor.BundledFor(i18n.EN)
	err := fs.WalkDir(bundled, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(bundled, p)
		if err != nil {
			return err
		}
		vault[strings.TrimSuffix(p, ".md")] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return vault
}

func englishRouteA(t *testing.T) (tutor.Route, fs.FS) {
	t.Helper()
	cat, err := tutor.BundledRoutesFor(i18n.EN)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range cat.Routes {
		if r.ID == "A" {
			return r, tutor.BundledFor(i18n.EN)
		}
	}
	t.Fatal("no English route A")
	return tutor.Route{}, nil
}

// play runs actions on lesson l of a fresh store, the way the app does: the
// lesson note's text and the vault are rechecked, then the events recorded.
func play(t *testing.T, l tutor.Lesson, lessonText string, vault map[string]string, dir string, actions ...enAStep) *tutor.Store {
	t.Helper()
	notes := map[string]string{}
	for k, v := range vault {
		notes[k] = v
	}
	text, today := lessonText, ""
	for _, a := range actions {
		if a.text != "" {
			text += "\n" + a.text + "\n"
		}
		for k, v := range a.notes {
			notes[k] = v
		}
		if a.today != "" {
			today = a.today
		}
	}
	env, _ := vaultEnv(dir, notes, nil)
	env.Today = today
	s := &tutor.Store{}
	s.Recheck(l, text, env)
	for _, a := range actions {
		for _, e := range a.events {
			s.RecordAction(l, e[0], e[1])
		}
	}
	return s
}

// S5: every checked step of the English route A starts unpassed in the
// installed vault, passes when the player does what its lesson says, and
// doing it passes no later step; doing every step completes the lesson.
func TestEnglishRouteAStepsPassOnlyWhenDone(t *testing.T) {
	r, bundled := englishRouteA(t)
	vault := englishVault(t)
	used := map[string]bool{}
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(bundled, l.Path())
		if err != nil {
			t.Fatalf("%s: %v", l.Path(), err)
		}
		lessonText := string(data)

		untouched := play(t, l, lessonText, vault, r.Dir)
		var all []enAStep
		for i, st := range l.Steps {
			key := l.ID + "#" + strconv.Itoa(i+1)
			if untouched.Passed(l, i+1) {
				t.Errorf("%s starts passed in the installed vault", key)
			}
			if st.Check.Kind == tutor.KindNone {
				continue
			}
			a, ok := enRouteA[key]
			if !ok {
				t.Errorf("%s (%+v) has no action in the table", key, st.Check)
				continue
			}
			used[key] = true
			all = append(all, a)
			done := play(t, l, lessonText, vault, r.Dir, a)
			if !done.Passed(l, i+1) {
				t.Errorf("%s does not pass after its action (check %+v)", key, st.Check)
			}
			for j := i + 1; j < len(l.Steps); j++ {
				later := l.ID + "#" + strconv.Itoa(j+1)
				if done.Passed(l, j+1) && !sameGesture[key+">"+later] {
					t.Errorf("doing %s already passes %s", key, later)
				}
			}
		}
		every := play(t, l, lessonText, vault, r.Dir, all...)
		for i, st := range l.Steps {
			if st.Check.Kind != tutor.KindNone && !every.Passed(l, i+1) {
				t.Errorf("%s#%d does not pass once every step is done", l.ID, i+1)
			}
		}
	}
	for key := range enRouteA {
		if !used[key] {
			t.Errorf("the table has %s, which is not a checked step of the English route A", key)
		}
	}
}

// S7: the UI texts the English route A quotes are the English interface's
// own, word for word.
func TestEnglishRouteAQuotesTheEnglishUI(t *testing.T) {
	r, bundled := englishRouteA(t)
	en := func(k i18n.Key, args ...any) string { return i18n.TFor(i18n.EN, k, args...) }
	quotes := []struct {
		lesson, quote, ui string
	}{
		{"A07", "`[ link ]`", "`" + en(i18n.PreviewMentionButton) + "`"},
		{"A08", "`Update N links in M notes?`", "`" + en(i18n.AppRelinkTitle, en(i18n.AppLinksInNotes, "N links", "M notes")) + "`"},
		{"A10", "with Cancel already selected", "with " + en(i18n.AppCancel) + " already selected"},
		{"A14", "choose `Cover`", "choose `" + en(i18n.SlashLabel("cover")) + "`"},
		{"A14", "`/Cover`", "`/" + en(i18n.SlashLabel("cover")) + "`"},
		{"A15", "choose `Form`", "choose `" + en(i18n.SlashLabel("form")) + "`"},
		{"A16", "`/Form`", "`/" + en(i18n.SlashLabel("form")) + "`"},
		{"A17", "`⊕ add question`", "`" + en(i18n.PreviewFormAddQuestion) + "`"},
		{"A19", "`Both kept:`", "`" + strings.TrimSuffix(en(i18n.AppBothKept, ""), " ") + "`"},
	}
	texts := map[string]string{}
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(bundled, l.Path())
		if err != nil {
			t.Fatal(err)
		}
		texts[l.ID] = string(data)
	}
	for _, q := range quotes {
		if q.quote != q.ui {
			t.Errorf("%s quotes %s, the English UI says %s", q.lesson, q.quote, q.ui)
		}
		if !strings.Contains(texts[q.lesson], q.quote) {
			t.Errorf("%s no longer quotes %s", q.lesson, q.quote)
		}
	}
}
