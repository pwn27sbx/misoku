package tutor

import (
	"encoding/json"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// ePlay is what the player does for one step of the English route E, as the
// lesson tells it, and what the app reports for it: the events Misoku emits
// (tutor_events_*.go, sync.go, themes.go, settings.go), config.json before
// and after, the vault folders after, a line written in the lesson note, and
// whether the note is saved with changes (a step with no check).
type ePlay struct {
	events []Event
	before string // config.json when the lesson opens
	after  string // config.json once the step is done
	folder string // a vault folder the step creates
	write  string // a line the player writes in the lesson note
	save   bool   // the player saves the note with something written
}

func ev(name, value string) Event {
	return Event{Name: name, Data: map[string]string{"value": value}}
}

// ePlays has one row per step of the English route E.
var ePlays = map[string]ePlay{
	"E01#1": {events: []Event{ev("leader_guide", "")}},
	"E01#2": {events: []Event{ev("leader_guide", "subpage")}},
	"E01#3": {events: []Event{ev("palette_open", "configured-keys")}},
	"E02#1": {after: `{"keys": {"reading_view": ["space p", "z"]}}`},
	"E02#2": {after: `{"keys": {"export_pdf": []}}`},
	"E03#1": {events: []Event{ev("key_clash", "")}},
	"E03#2": {after: `{"keys": {"reading_mode": ["space p"], "toggle_preview": ["space r"]}}`},
	"E04#1": {events: []Event{ev("settings_open", ""), ev("settings_saved", "")}},
	"E04#2": {after: `{"ui": {"explorer_hide_width": 100}}`},
	"E05#1": {after: `{"ui": {"heading_icons": ["① ", "② "], "bullets": ["•", "◦"]}}`},
	// The border went to single, back to rounded and is left on single.
	"E05#2": {before: `{"ui": {"border": "rounded"}}`, after: `{"ui": {"border": "single"}}`},
	"E06#1": {save: true},
	"E06#2": {after: `{"ui": {"transparent": true}}`},
	"E06#3": {after: `{"ui": {"checkbox": {"todo": "◐"}}}`},
	"E06#4": {write: "- [-] call the hotel"},
	// On, then off: the row writes false.
	"E07#1": {after: `{"ui": {"heading_numbers": false}}`},
	"E07#2": {events: []Event{ev("cursor_shape", "block"), ev("cursor_shape", "bar")}},
	"E08#1": {events: []Event{ev("theme_pick", ""), ev("settings_saved", "")}},
	"E08#2": {events: []Event{ev("theme_pick", "revert")}},
	"E08#3": {events: []Event{ev("theme_list", "")}},
	"E09#1": {before: `{"theme": "onedark"}`, after: `{"theme": "catppuccin"}`},
	"E09#2": {events: []Event{ev("theme_list", "")}},
	"E09#3": {after: `{"themes_dir": "~/themes"}`},
	"E10#1": {after: `{"notes_dir": "~/Notes"}`},
	"E10#2": {after: `{"daily_dir": "Diario/Trips"}`},
	"E10#3": {after: `{"templates_dir": "Plantillas"}`},
	"E11#1": {after: `{"sync": {"vaults": {"~/Notes": {}}}}`},
	"E11#2": {after: `{"learn_offered": true}`},
	"E12#1": {events: []Event{ev("sync_run", "ok")}},
	"E12#2": {events: []Event{ev("sync_run", "on-open")}},
	"E12#3": {events: []Event{ev("sync_run", "timed")}},
	"E13#1": {save: true},
	"E13#2": {events: []Event{ev("sync_run", "on-quit")}},
	"E14#1": {events: []Event{ev("sync_status", "")}},
	"E14#2": {events: []Event{ev("sync_status", "left-out")}},
	"E15#1": {events: []Event{ev("sync_stop_dialog", ""), ev("sync_run", "ok")}},
	"E15#2": {save: true},
	"E15#3": {events: []Event{ev("sync_auth_offer", "")}},
	"E16#1": {events: []Event{ev("sync_indicator", "")}},
	"E16#2": {events: []Event{ev("sync_arrival", "")}},
	"E16#3": {events: []Event{ev("sync_indicator", "other-window")}},
	"E17#1": {folder: ".git", after: `{"sync": {"vaults": {"~/Notes": {}}}}`},
	"E17#2": {save: true},
	"E18#1": {events: []Event{ev("graph_move", "")}},
	"E18#2": {events: []Event{ev("graph_filter", "")}},
	"E18#3": {events: []Event{ev("command_error", "E488")}},
	"E19#1": {events: []Event{ev("motion", "scroll-line")}},
	"E19#2": {events: []Event{ev("undo_empty", "")}},
	"E20#1": {save: true},
	"E20#2": {save: true},
	"E20#3": {save: true},
	"E20#4": {save: true},
}

// configReader answers Env.Config from a config.json text the way the app
// reads it (internal/app/tutor_config.go): strings as they are, booleans and
// numbers spelled out, lists and objects as JSON.
func configReader(t *testing.T, text string) func(string) (string, bool) {
	t.Helper()
	var root any = map[string]any{}
	if text != "" {
		if err := json.Unmarshal([]byte(text), &root); err != nil {
			t.Fatalf("config %s: %v", text, err)
		}
	}
	return func(key string) (string, bool) {
		cur := root
		for _, part := range strings.Split(key, ".") {
			m, ok := cur.(map[string]any)
			if !ok {
				return "", false
			}
			if cur, ok = m[part]; !ok {
				return "", false
			}
		}
		switch v := cur.(type) {
		case string:
			return v, true
		case bool:
			return strconv.FormatBool(v), true
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64), true
		}
		data, _ := json.Marshal(cur)
		return string(data), true
	}
}

// ePlayEnv is the app's disk reader for one state of the vault and config.
func ePlayEnv(t *testing.T, before, now string, folder string) Env {
	was, is := configReader(t, before), configReader(t, now)
	return Env{
		Exists:       func(string) bool { return false },
		FolderExists: func(name string) bool { return folder != "" && name == folder },
		Config:       is,
		ConfigChanged: func(key string) bool {
			old, had := was(key)
			cur, has := is(key)
			return had != has || old != cur
		},
	}
}

func englishRouteE(t *testing.T) (Route, fs.FS) {
	t.Helper()
	cat, err := BundledRoutesFor(i18n.EN)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := cat.RouteByID("E")
	if !ok {
		t.Fatal("no English route E")
	}
	if r.Dir != "4 Expert" || r.Name != "Expert" {
		t.Fatalf("route E is %q in %q, want Expert in 4 Expert", r.Name, r.Dir)
	}
	return r, BundledFor(i18n.EN)
}

// S5: every step of the English route E passes when the player does what
// its lesson says, and not before: on the untouched lesson, with config.json
// as it was when the lesson opened and no event from the app.
func TestEnglishRouteEStepsPassOnlyWhenDone(t *testing.T) {
	useLang(t, i18n.EN)
	r, bundled := englishRouteE(t)
	used := map[string]bool{}
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(bundled, l.Path())
		if err != nil {
			t.Fatal(err)
		}
		lesson := string(data)
		for i := range l.Steps {
			key := StepKey(l.ID, i+1)
			p, ok := ePlays[key]
			if !ok {
				t.Errorf("%s has no play in the table", key)
				continue
			}
			used[key] = true

			idle := &Store{}
			idle.Recheck(l, lesson, ePlayEnv(t, p.before, p.before, ""))
			idle.RecordSave(l, lesson, lesson)
			if idle.Passed(l, i+1) {
				t.Errorf("%s passes before the player does it", key)
			}

			done := &Store{}
			after := p.after
			if after == "" {
				after = p.before
			}
			text := lesson
			if p.write != "" {
				text = strings.Replace(lesson, "\n## Your turn\n", "\n## Your turn\n\n"+p.write+"\n", 1)
			}
			done.Recheck(l, text, ePlayEnv(t, p.before, after, p.folder))
			for _, e := range p.events {
				done.RecordEvent(l, e)
			}
			if p.save {
				done.RecordSave(l, lesson, lesson+"\nI did it.\n")
			}
			if !done.Passed(l, i+1) {
				t.Errorf("%s does not pass when the player does what the lesson says", key)
			}
		}
	}
	for key := range ePlays {
		if !used[key] {
			t.Errorf("the table plays %s, which route E does not have", key)
		}
	}
}

// label is the name part of a UI text: what comes before its first ":" or
// " (", without the sync icon.
func label(s string) string {
	s = strings.TrimPrefix(s, "󰓦 ")
	if i := strings.Index(s, ":"); i > 0 {
		s = s[:i]
	}
	if i := strings.Index(s, " ("); i > 0 {
		s = s[:i]
	}
	return s
}

// S7: the English route E quotes the English interface. Each lesson must
// contain the labels it talks about, exactly as the catalog shows them.
func TestEnglishRouteEQuotesTheEnglishLabels(t *testing.T) {
	en := func(k i18n.Key, args ...any) string { return i18n.TFor(i18n.EN, k, args...) }
	want := map[string][]string{
		"E01": {
			strings.ToUpper(en(i18n.KeymapGroup("note"))),
			strings.ToUpper(en(i18n.KeymapGroup("view"))),
			strings.ToUpper(en(i18n.KeymapGroup("go"))),
		},
		"E04": {
			en(i18n.SettingAccordion), en(i18n.SettingHideWidth), en(i18n.SettingLook),
			en(i18n.SettingBorder), en(i18n.SettingTransparent), en(i18n.SettingHeadingNumbers),
			en(i18n.SettingLinkSuggestions), en(i18n.SettingLanguage),
			en(i18n.LanguageAuto), i18n.Name(i18n.EN), i18n.Name(i18n.ES),
			"eight settings",
		},
		"E05": {
			en(i18n.SettingLook), en(i18n.SettingBorder), en(i18n.SettingHeadingNumbers),
			"Misoku", en(i18n.LookNumbers), en(i18n.LookArrows), en(i18n.LookPlain),
		},
		"E06": {en(i18n.SettingTransparent)},
		"E07": {en(i18n.SettingHeadingNumbers), en(i18n.SettingLook), en(i18n.LookNumbers)},
		"E11": {en(i18n.WelcomeNew), en(i18n.WelcomeExperienced), en(i18n.WelcomeLater)},
		"E12": {en(i18n.AppSynced)},
		"E14": {
			label(en(i18n.AppSyncInfoState, "x")), label(en(i18n.AppSyncInfoRemote, "x")),
			label(en(i18n.AppSyncInfoBranch, "x")), label(en(i18n.AppSyncInfoTimed, "x")),
			label(i18n.NFor(i18n.EN, i18n.AppSyncInfoPending, 2, 2)), label(en(i18n.AppSyncInfoLastOK, "x")),
			label(en(i18n.AppSyncInfoLastErr, "x", "x")), label(en(i18n.AppSyncInfoConflicts, 2)),
			label(en(i18n.AppSyncInfoFiles, 2)),
			i18n.NFor(i18n.EN, i18n.SyncArrivedNew, 2, 2),
		},
		"E15": {en(i18n.AppSyncCheckAgain), en(i18n.AppSyncRemoveLock), en(i18n.AppSyncSignIn)},
		"E16": {
			en(i18n.AppSyncStateSyncing)[len("󰓦 "):],
			en(i18n.AppSyncStateSynced, "5m ago")[len("󰓦 "):],
			en(i18n.AppSyncStateBusy)[len("󰓦 "):],
		},
		"E18": {label(en(i18n.EditorTrailingGraph, "foo")) + ": Trailing characters"},
		"E19": {en(i18n.EditorUndoOldest)},
	}
	r, bundled := englishRouteE(t)
	for _, l := range r.Lessons {
		data, err := fs.ReadFile(bundled, l.Path())
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range want[l.ID] {
			if !strings.Contains(string(data), s) {
				t.Errorf("%s does not quote %q", l.Path(), s)
			}
		}
		delete(want, l.ID)
	}
	for id := range want {
		t.Errorf("no lesson %s in the English route E", id)
	}
}

// S5: a step with no check passes only when the player saves the lesson
// with something written in it, so every lesson with such a step, the
// challenge included, asks the player to write in the note.
func TestEnglishRouteEAsksToWriteForUncheckedSteps(t *testing.T) {
	r, bundled := englishRouteE(t)
	for _, l := range r.Lessons {
		unchecked := false
		for _, st := range l.Steps {
			if st.Check.Kind == KindNone {
				unchecked = true
			}
		}
		if !unchecked {
			continue
		}
		data, err := fs.ReadFile(bundled, l.Path())
		if err != nil {
			t.Fatal(err)
		}
		_, turn, _ := strings.Cut(string(data), "## Your turn")
		if !strings.Contains(turn, "below") && !strings.Contains(turn, "in this note") {
			t.Errorf("%s has a step with no check but never asks the player to write in the note", l.Path())
		}
	}
}
