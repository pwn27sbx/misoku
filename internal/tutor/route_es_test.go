package tutor_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/tutor"
)

// spanishRoutes are the Spanish routes and their embedded notes.
func spanishRoutes(t *testing.T) ([]tutor.Route, fs.FS) {
	t.Helper()
	cat, err := tutor.BundledRoutesFor(i18n.ES)
	if err != nil {
		t.Fatalf("the Spanish catalog is invalid: %v", err)
	}
	if len(cat.Routes) != 4 {
		t.Fatalf("the Spanish catalog has %d routes, want 4", len(cat.Routes))
	}
	return cat.Routes, tutor.BundledFor(i18n.ES)
}

// spanishRoute is the Spanish route id.
func spanishRoute(t *testing.T, id string) (tutor.Route, fs.FS) {
	t.Helper()
	cat, err := tutor.BundledRoutesFor(i18n.ES)
	if err != nil {
		t.Fatalf("the Spanish catalog is invalid: %v", err)
	}
	r, ok := cat.RouteByID(id)
	if !ok {
		t.Fatalf("no Spanish route %s", id)
	}
	return r, tutor.BundledFor(i18n.ES)
}

// spanishTexts is every text of the Spanish tutorial a player reads, by
// where it is: lessons and indexes by their ID ("B05", "B00"), steps by
// their key ("B05#1", text and hint) and the note after the tutorial.
func spanishTexts(t *testing.T) map[string]string {
	t.Helper()
	routes, notes := spanishRoutes(t)
	read := func(p string) string {
		data, err := fs.ReadFile(notes, p)
		if err != nil {
			t.Fatalf("%s not embedded: %v", p, err)
		}
		return string(data)
	}
	texts := map[string]string{}
	for _, r := range routes {
		texts[r.ID+"00"] = read(r.IndexPath())
		for _, l := range r.Lessons {
			texts[l.ID] = read(l.Path())
			for i, st := range l.Steps {
				texts[tutor.StepKey(l.ID, i+1)] = st.Text + "\n" + st.Hint
			}
		}
	}
	texts["after"] = read(tutor.ReferenceNoteFor(i18n.ES))
	return texts
}

var missingArgRe = regexp.MustCompile(`%(!\w\(MISSING\)|[a-z])`)

// uiPhrases are the words of an interface text without its arguments: the
// fragments between the placeholders, trimmed of the sync icon, spaces and
// punctuation, that are long enough to name something.
func uiPhrases(s string) []string {
	var out []string
	for _, f := range missingArgRe.Split(s, -1) {
		f = strings.TrimFunc(strings.TrimPrefix(strings.TrimSpace(f), "󰓦 "), func(r rune) bool {
			return unicode.IsSpace(r) || strings.ContainsRune(":·()?¿", r)
		})
		if len([]rune(f)) >= 4 {
			out = append(out, f)
		}
	}
	return out
}

// containsWord reports whether text holds phrase as whole words: not inside
// a longer word, as "Form" is inside "Formulario".
func containsWord(text, phrase string) bool {
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	for from := 0; ; {
		i := strings.Index(text[from:], phrase)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(phrase)
		before, after := ' ', ' '
		if start > 0 {
			before = []rune(text[:start])[len([]rune(text[:start]))-1]
		}
		if end < len(text) {
			after = []rune(text[end:])[0]
		}
		if !word(before) && !word(after) {
			return true
		}
		from = end
	}
}

// T10/S7: the Spanish tutorial quotes the Spanish interface. Every "/" label
// is a label of the Spanish "/" menu, every quoted preview button is the
// Spanish one, and no lesson, index or step names a button, row, dialog
// choice or message by its English text where the Spanish UI shows another.
func TestSpanishRoutesQuoteTheSpanishInterface(t *testing.T) {
	es := func(k i18n.Key, args ...any) string { return i18n.TFor(i18n.ES, k, args...) }
	en := func(k i18n.Key, args ...any) string { return i18n.TFor(i18n.EN, k, args...) }
	texts := spanishTexts(t)

	var slash []string
	for _, id := range slashIDs {
		slash = append(slash, es(i18n.SlashLabel(id)))
	}
	isSlash := func(s string) bool {
		for _, label := range slash {
			rest, ok := strings.CutPrefix(s, label)
			if ok && (rest == "" || !unicode.IsLetter([]rune(rest)[0]) && !unicode.IsDigit([]rune(rest)[0])) {
				return true
			}
		}
		return false
	}
	buttonKeys := []i18n.Key{
		i18n.PreviewAddRow, i18n.PreviewAddColumn, i18n.PreviewNewPage, i18n.PreviewAddValue,
		i18n.PreviewTimelineNew, i18n.PreviewToday, i18n.PreviewFormAddQuestion, i18n.PreviewMentionButton,
	}
	buttons := map[string]bool{}
	for _, k := range buttonKeys {
		buttons[es(k)] = true
	}
	slashes := 0
	for where, text := range texts {
		for _, line := range strings.Split(text, "\n") {
			for _, m := range slashRe.FindAllStringSubmatch(line, -1) {
				slashes++
				if !isSlash(m[2]) {
					t.Errorf("%s quotes /%s, which is no Spanish \"/\" menu item", where, m[2])
				}
			}
		}
		for _, m := range quotedRe.FindAllStringSubmatch(text, -1) {
			q := m[1]
			if (strings.HasPrefix(q, "+ ") || strings.ContainsAny(q, "‹›⊕")) && !buttons[q] {
				t.Errorf("%s quotes the button `%s`, which the Spanish preview does not show", where, q)
			}
		}
	}
	if slashes < 40 {
		t.Errorf("found %d \"/\" labels, want at least 40", slashes)
	}

	// English texts the Spanish UI translates: none may be quoted.
	english := append([]i18n.Key{
		i18n.SettingAccordion, i18n.SettingHideWidth, i18n.SettingLook, i18n.SettingBorder,
		i18n.SettingTransparent, i18n.SettingHeadingNumbers, i18n.SettingLinkSuggestions,
		i18n.SettingLanguage, i18n.LookNumbers, i18n.LookArrows, i18n.LookPlain,
		i18n.LinkSuggestionsInline, i18n.LinkSuggestionsFinder, i18n.LanguageAuto,
		i18n.AppSaveAll, i18n.AppDiscard, i18n.AppCancel, i18n.AppBothKept, i18n.AppRelinkTitle,
		i18n.AppSynced, i18n.AppSyncing, i18n.AppSyncCheckAgain, i18n.AppSyncRemoveLock, i18n.AppSyncSignIn,
		i18n.AppSyncStateSyncing, i18n.AppSyncStateSynced, i18n.AppSyncStateBusy, i18n.AppSyncStateOffline,
		i18n.AppSyncInfoState, i18n.AppSyncInfoRemote, i18n.AppSyncInfoBranch, i18n.AppSyncInfoTimed,
		i18n.AppSyncInfoLastOK, i18n.AppSyncInfoLastErr, i18n.AppSyncInfoFiles,
		i18n.EditorPatternNotFound, i18n.EditorSearchWrappedTop, i18n.EditorUndoOldest, i18n.EditorTrailingGraph,
		i18n.WelcomeNew, i18n.WelcomeExperienced, i18n.WelcomeLater, i18n.DashRecent, i18n.DashFolders,
	}, buttonKeys...)
	for _, id := range slashIDs {
		english = append(english, i18n.SlashLabel(id))
	}
	checked := 0
	for _, k := range english {
		for _, phrase := range uiPhrases(en(k)) {
			if strings.Contains(es(k), phrase) {
				continue // the same in both languages ("Video", "Audio")
			}
			checked++
			for where, text := range texts {
				if containsWord(text, phrase) {
					t.Errorf("%s names %q, which the Spanish interface shows as %q", where, phrase, es(k))
				}
			}
		}
	}
	if checked < 70 {
		t.Errorf("checked %d English phrases, want at least 70", checked)
	}

	passed, _, _ := strings.Cut(es(i18n.TutorLessonPassed, "B03", "B04"), " · ")
	notFound, _, _ := strings.Cut(es(i18n.EditorPatternNotFound, "zzz"), ": zzz")
	label := func(s string) string {
		s = strings.TrimPrefix(s, "󰓦 ")
		if i := strings.Index(s, ":"); i > 0 {
			s = s[:i]
		}
		if i := strings.Index(s, " ("); i > 0 {
			s = s[:i]
		}
		return s
	}
	want := map[string][]string{
		"B00": {
			"`" + strings.TrimSuffix(es(i18n.TutorStepLine, "B03", 2, 4, "…"), "…") + "…`",
			"`" + es(i18n.TutorStepFlash, 2) + "`",
			"`" + passed + "`",
			"`" + es(i18n.TutorRouteDone, "Básico") + "`",
			"**" + es(i18n.WelcomeNew) + "**",
			"**" + es(i18n.WelcomeExperienced) + "**",
			"**" + es(i18n.WelcomeLater) + "**",
			"`" + es(i18n.TutorAfterTutorial) + ".md`",
		},
		"B22": {"`" + notFound + "`", "`" + es(i18n.EditorSearchWrappedTop) + "`"},
		"B24": {"**" + es(i18n.DashRecent) + "**", "**" + es(i18n.DashFolders) + "**"},
		"B25": {"`" + es(i18n.AppSaveAll) + "`", "`" + es(i18n.AppDiscard) + "`", "`" + es(i18n.AppCancel) + "`"},
		"I10": {"`" + es(i18n.PreviewAddRow) + "`", "`" + es(i18n.PreviewAddColumn) + "`"},
		"I13": {"`" + es(i18n.PreviewNewPage) + "`"},
		"I14": {"`" + es(i18n.PreviewToday) + "`"},
		"I15": {"`" + es(i18n.PreviewTimelineNew) + "`"},
		"I16": {"`" + es(i18n.PreviewAddValue) + "`"},
		"I20": {es(i18n.SettingLinkSuggestions), es(i18n.LinkSuggestionsFinder)},
		"I22": {es(i18n.SettingLinkSuggestions), es(i18n.LinkSuggestionsInline), es(i18n.LinkSuggestionsFinder)},
		"I28": {es(i18n.SettingAccordion)},
		"A07": {"`" + es(i18n.PreviewMentionButton) + "`"},
		"A08": {"`" + es(i18n.AppRelinkTitle, es(i18n.AppLinksInNotes, "N enlaces", "M notas")) + "`"},
		"A10": {es(i18n.AppCancel)},
		"A14": {"`/" + es(i18n.SlashLabel("cover")) + "`", "`" + es(i18n.SlashLabel("cover")) + "`"},
		"A15": {"`" + es(i18n.SlashLabel("form")) + "`"},
		"A16": {"`/" + es(i18n.SlashLabel("form")) + "`"},
		"A17": {"`" + es(i18n.PreviewFormAddQuestion) + "`"},
		"A19": {"`" + strings.TrimSuffix(es(i18n.AppBothKept, ""), " ") + "`"},
		"E01": {
			strings.ToUpper(es(i18n.KeymapGroup("note"))),
			strings.ToUpper(es(i18n.KeymapGroup("view"))),
			strings.ToUpper(es(i18n.KeymapGroup("go"))),
		},
		"E04": {
			es(i18n.SettingAccordion), es(i18n.SettingHideWidth), es(i18n.SettingLook),
			es(i18n.SettingBorder), es(i18n.SettingTransparent), es(i18n.SettingHeadingNumbers),
			es(i18n.SettingLinkSuggestions), es(i18n.SettingLanguage),
			es(i18n.LanguageAuto), i18n.Name(i18n.EN), i18n.Name(i18n.ES),
			"ocho ajustes",
		},
		"E05": {
			es(i18n.SettingLook), es(i18n.SettingBorder), es(i18n.SettingHeadingNumbers),
			"Misoku", es(i18n.LookNumbers), es(i18n.LookArrows), es(i18n.LookPlain),
		},
		"E06": {es(i18n.SettingTransparent)},
		"E07": {es(i18n.SettingHeadingNumbers), es(i18n.SettingLook), es(i18n.LookNumbers)},
		"E11": {es(i18n.WelcomeNew), es(i18n.WelcomeExperienced), es(i18n.WelcomeLater)},
		"E12": {"`" + es(i18n.AppSyncing) + "`", "`" + es(i18n.AppSynced) + "`"},
		"E14": {
			label(es(i18n.AppSyncInfoState, "x")), label(es(i18n.AppSyncInfoRemote, "x")),
			label(es(i18n.AppSyncInfoBranch, "x")), label(es(i18n.AppSyncInfoTimed, "x")),
			label(i18n.NFor(i18n.ES, i18n.AppSyncInfoPending, 2, 2)), label(es(i18n.AppSyncInfoLastOK, "x")),
			label(es(i18n.AppSyncInfoLastErr, "x", "x")), label(es(i18n.AppSyncInfoConflicts, 2)),
			label(es(i18n.AppSyncInfoFiles, 2)),
			i18n.NFor(i18n.ES, i18n.SyncArrivedNew, 2, 2),
		},
		"E15": {es(i18n.AppSyncCheckAgain), es(i18n.AppSyncRemoveLock), es(i18n.AppSyncSignIn)},
		"E16": {
			es(i18n.AppSyncStateSyncing)[len("󰓦 "):],
			es(i18n.AppSyncStateSynced, es(i18n.AppAgoMinutes, 5))[len("󰓦 "):],
			es(i18n.AppSyncStateBusy)[len("󰓦 "):],
		},
		"E18": {label(es(i18n.EditorTrailingGraph, "foo"))},
		"E19": {es(i18n.EditorUndoOldest)},
	}
	for where, quotes := range want {
		text, ok := texts[where]
		if !ok {
			t.Errorf("no Spanish lesson %s", where)
			continue
		}
		for _, q := range quotes {
			if !strings.Contains(text, q) {
				t.Errorf("%s does not quote the Spanish interface's %q", where, q)
			}
		}
	}
}

// esEdit is one Spanish step: the note's edits before the player does it
// (untouched when nil) and once done.
type esEdit struct {
	before, done []noteEdit
}

// T10/S5: the Spanish steps that never passed, or passed before the player
// did anything, pass only once done the way the lesson now says.
func TestSpanishFixedStepsPassOnlyWhenDone(t *testing.T) {
	steps := map[string]esEdit{
		// u after dd brings the line back: the step is the deletion.
		"B05#1": {done: []noteEdit{replaceLine("bórrame", "")}},
		"B05#2": {done: []noteEdit{addLines("ups")}},
		"I02#4": {
			before: []noteEdit{addLines("viento y marea", "faro y ola")},
			done:   []noteEdit{addLines("viento y ", "faro y ola")},
		},
		"I27#5": {
			before: []noteEdit{addLines("---", "estado: en curso", "---", "Plan del día")},
			done:   []noteEdit{addLines("---", "estado: en curso", "lugar: playa", "---", "Plan del día")},
		},
	}
	routes, notes := spanishRoutes(t)
	found := 0
	for _, r := range routes {
		for _, l := range r.Lessons {
			for i := range l.Steps {
				key := tutor.StepKey(l.ID, i+1)
				c, ok := steps[key]
				if !ok {
					continue
				}
				found++
				data, err := fs.ReadFile(notes, l.Path())
				if err != nil {
					t.Fatal(err)
				}
				passes := func(edits []noteEdit) bool {
					s := &tutor.Store{}
					s.Recheck(l, applyEdits(string(data), edits), tutor.Env{})
					return s.Passed(l, i+1)
				}
				if passes(c.before) {
					t.Errorf("%s passes before the player does it", key)
				}
				if !passes(c.done) {
					t.Errorf("%s does not pass once done:\n%s", key, applyEdits(string(data), c.done))
				}
			}
		}
	}
	if found != len(steps) {
		t.Errorf("found %d of the %d steps in the Spanish routes", found, len(steps))
	}
}

// T10/S5: the Spanish B22 searches are typed as the lesson shows them, and
// each finds its match in the practice area itself, with Misoku's
// smartcase: a query with an upper-case letter is case-sensitive.
func TestSpanishRouteBSearchesFindTheirMatchInThePracticeArea(t *testing.T) {
	r, notes := spanishRoute(t, "B")
	checked := 0
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(notes, l.Path())
		if err != nil {
			t.Fatal(err)
		}
		lesson := string(data)
		_, area, _ := strings.Cut(lesson, "## Tu zona de pruebas")
		area, _, _ = strings.Cut(area, "\nSiguiente: ")
		area = inlineCodeRe.ReplaceAllString(area, "")
		for i, st := range l.Steps {
			if st.Check.Event != "search_hit" || len(st.Check.Value) == 0 {
				continue
			}
			checked++
			q := st.Check.Value[0]
			key := tutor.StepKey(l.ID, i+1)
			if !strings.Contains(lesson, "`"+q+"`") {
				t.Errorf("%s waits for a search of %q, which the lesson never asks to type", key, q)
			}
			a := area
			if strings.ToLower(q) == q {
				a, q = strings.ToLower(a), strings.ToLower(q)
			}
			if !strings.Contains(a, q) {
				t.Errorf("%s: a search of %q finds nothing in the practice area", key, st.Check.Value[0])
			}
		}
	}
	if checked != 2 {
		t.Errorf("checked %d searches, want 2 (B22#1 and B22#3)", checked)
	}
}

// T10/S5: the Spanish steps whose old wording asked for something else, or
// left out what the check waits for, now say it. Each entry is a text the
// lesson or the step must hold (want) or no longer hold (not).
func TestSpanishStepsSayWhatTheirCheckWaitsFor(t *testing.T) {
	texts := spanishTexts(t)
	cases := []struct {
		where, want, not string
	}{
		// B05: the deletion is of a line the practice area holds.
		{"B05", "`bórrame`", ""},
		// B22: smartcase, not case-sensitive search.
		{"B22", "`Hotel Miramar`", "La búsqueda distingue mayúsculas"},
		{"B22#3", "", "distingue mayúsculas"},
		// B24: keys 1-9 number the recent notes first, then the folders.
		{"B24#2", "primera carpeta", "luego 2"},
		{"B24", "", "pulsa `2`"},
		// I02: every marea goes, copies included.
		{"I02", "ninguna `marea`", ""},
		{"I02#4", "ninguna marea", ""},
		// I25: tag: and # are typed in the same ctrl+p finder.
		{"I25#2", "ctrl+p", ""},
		{"I25#3", "Bórralo", ""},
		// I27: the template goes into this note, with lugar: playa.
		{"I27#3", "", "otra nota"},
		{"I27#5", "lugar: playa", ""},
		// A03: the properties the table edits are written first.
		{"A03", "escribe un bloque con `---`, `viaje: costa`", ""},
		// A12: a name A08 did not already use, so the node is hollow.
		{"A12", "`[[Mochila]]`", "Maleta"},
		// E17: its step with no check passes on a save with text.
		{"E17", "aquí abajo", ""},
	}
	for _, c := range cases {
		text, ok := texts[c.where]
		if !ok {
			t.Errorf("no Spanish text %s", c.where)
			continue
		}
		if c.want != "" && !strings.Contains(text, c.want) {
			t.Errorf("%s does not say %q", c.where, c.want)
		}
		if c.not != "" && strings.Contains(text, c.not) {
			t.Errorf("%s still says %q", c.where, c.not)
		}
	}
}

// T10/S5: the note A12 asks the player to draft from a hollow node is named
// by no earlier lesson of the Advanced route, since a wikilink resolves by
// suffix to any note of that name the player already made (A08's Maleta).
func TestSpanishRouteAHollowNodeIsANewName(t *testing.T) {
	r, notes := spanishRoute(t, "A")
	earlier := ""
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(notes, l.Path())
		if err != nil {
			t.Fatal(err)
		}
		if l.ID != "A12" {
			earlier += string(data)
			continue
		}
		for _, st := range l.Steps {
			if st.Check.Event != "new_draft" {
				continue
			}
			name := st.Check.Value[0]
			if !strings.Contains(string(data), "`[["+name+"]]`") {
				t.Errorf("A12 drafts %q, which the lesson never asks to link", name)
			}
			if strings.Contains(earlier, "`"+name+"`") || strings.Contains(earlier, "[["+name+"]]") {
				t.Errorf("A12 drafts %q, a note an earlier lesson already creates", name)
			}
			return
		}
	}
	t.Error("no A12 new_draft step in the Spanish route A")
}

// T10/S5: a step with no check passes only when the player saves the lesson
// with something written in it, so every Spanish route E lesson with such a
// step asks the player to write in the note, as the English route E does.
func TestSpanishRoutesAskToWriteForUncheckedSteps(t *testing.T) {
	r, notes := spanishRoute(t, "E")
	for _, l := range r.Lessons {
		unchecked := false
		for _, st := range l.Steps {
			unchecked = unchecked || st.Check.Kind == tutor.KindNone
		}
		if !unchecked {
			continue
		}
		data, err := fs.ReadFile(notes, l.Path())
		if err != nil {
			t.Fatal(err)
		}
		_, turn, _ := strings.Cut(string(data), "## Tu turno")
		if !strings.Contains(turn, "aquí abajo") && !strings.Contains(turn, "en esta nota") {
			t.Errorf("%s has a step with no check but never asks the player to write in the note", l.Path())
		}
	}
}
