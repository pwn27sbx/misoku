package tutor

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// textLesson builds a fixture lesson whose steps use the given checks.
func textLesson(id string, checks ...Check) Lesson {
	l := Lesson{ID: id, Title: "Fixture", File: id + " Fixture.md"}
	for _, c := range checks {
		l.Steps = append(l.Steps, Step{Text: "Paso", Check: c})
	}
	return l
}

func contains(s string) Check {
	return Check{Kind: KindText, Pred: "line-contains", Args: []string{s}}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l := textLesson("B03", contains("hecho"))
	s := &Store{}
	if !s.RecheckText(l, "hecho", Env{}) {
		t.Fatal("the step should pass")
	}
	s.SetMigrated(true)
	s.SetLastRoute("B")
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, progressFile))
	if err != nil {
		t.Fatal(err)
	}
	var st progressState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	if st.Version != progressVersion || !st.Migrated || st.LastRoute != "B" {
		t.Fatalf("state = %+v", st)
	}
	if !st.Steps["B03#1"] {
		t.Fatalf("steps = %v, want B03#1", st.Steps)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Passed(l, 1) {
		t.Fatal("Load lost the passed step")
	}
	if !loaded.Migrated() || loaded.LastRoute() != "B" {
		t.Fatalf("Load lost the flags: migrated=%v last=%q", loaded.Migrated(), loaded.LastRoute())
	}
}

// S3: Save writes one atomic file and leaves no temporary file behind.
func TestSaveLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	s := &Store{}
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != progressFile {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("files = %v, want only %s", names, progressFile)
	}
}

func TestLoadErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, progressFile)
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("malformed JSON should be an error")
	}
	if err := os.WriteFile(path, []byte(`{"version": 2, "steps": {}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("a newer version should be refused")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir)
	if err != nil || s == nil {
		t.Fatalf("a missing file is an empty store: %v", err)
	}
	if _, err := Load(""); err == nil {
		t.Fatal("Load with no folder should fail")
	}
	if err := s.Save(""); err == nil {
		t.Fatal("Save with no folder should fail")
	}
}

// E4: a passed step stays passed after its text is removed.
func TestStepsAreSticky(t *testing.T) {
	l := textLesson("B01", Check{Kind: KindText, Pred: "heading-exists", Args: []string{"2", "Viaje"}})
	s := &Store{}
	if !s.RecheckText(l, "## Viaje\n", Env{}) {
		t.Fatal("the heading should pass")
	}
	if s.RecheckText(l, "", Env{}) {
		t.Fatal("an empty text reported a new pass")
	}
	if !s.Passed(l, 1) {
		t.Fatal("the step un-passed when its text went away")
	}
}

// D4: action steps come only from the file, never from text.
func TestActionStepsComeOnlyFromTheFile(t *testing.T) {
	dir := t.TempDir()
	l := Lesson{ID: "B01", Title: "Fixture", File: "B01 Fixture.md", Steps: []Step{{
		Text:  "Sal de INSERT",
		Check: Check{Kind: KindAction, Event: "mode_changed", Value: Values{"normal"}},
	}}}
	s := &Store{}
	if s.Recheck(l, "normal mode_changed", Env{}) {
		t.Fatal("text passed an action step")
	}
	if s.RecheckText(l, "normal", Env{}) || s.RecheckDisk(l, Env{}) {
		t.Fatal("a recheck passed an action step")
	}
	if s.RecordAction(l, "mode_changed", "insert") {
		t.Fatal("a value outside the list recorded progress")
	}
	if s.Passed(l, 1) {
		t.Fatal("a wrong value passed the step")
	}
	if !s.RecordAction(l, "mode_changed", "normal") {
		t.Fatal("the second value was not recorded")
	}
	if !s.Passed(l, 1) {
		t.Fatal("both values should pass the step")
	}
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Passed(l, 1) {
		t.Fatal("Load lost the action step")
	}
	if loaded.Recheck(l, "", Env{}) {
		t.Fatal("rechecking after Load reported a new pass")
	}
}

// L2: an action value may be a list, and every value must be seen.
func TestActionValuesAllSeen(t *testing.T) {
	l := textLesson("B08", Check{Kind: KindAction, Event: "focus_explorer", Value: Values{"open", "close"}})
	s := &Store{}
	s.RecordAction(l, "focus_explorer", "open")
	if s.Passed(l, 1) {
		t.Fatal("one value passed the step")
	}
	s.RecordAction(l, "focus_explorer", "other")
	if s.Passed(l, 1) {
		t.Fatal("an unknown value passed the step")
	}
	s.RecordAction(l, "focus_explorer", "close")
	if !s.Passed(l, 1) {
		t.Fatal("both values should pass the step")
	}
	if !s.state.Steps["B08#1=1"] || !s.state.Steps["B08#1=2"] {
		t.Fatalf("value keys = %v", s.state.Steps)
	}
}

// L2: every sub-check of an all is sticky; the step waits for all of them.
func TestAllSubChecksSticky(t *testing.T) {
	l := textLesson("B17", Check{Kind: KindAll, Checks: []Check{contains("negrita"), contains("cursiva")}})
	s := &Store{}
	s.RecheckText(l, "negrita\n", Env{})
	if s.Passed(l, 1) {
		t.Fatal("one sub-check passed the step")
	}
	if !s.state.Steps["B17#1.1"] {
		t.Fatal("the first sub-check was not stored")
	}
	if !s.RecheckText(l, "cursiva\n", Env{}) {
		t.Fatal("the second sub-check should be new")
	}
	if !s.Passed(l, 1) {
		t.Fatal("both sub-checks should pass the step")
	}
	if s.RecheckText(l, "", Env{}) {
		t.Fatal("a later edit reported a new pass")
	}
	if !s.Passed(l, 1) {
		t.Fatal("the sub-checks un-passed with the text")
	}
}

// L2: a mixed step (text plus action) waits for both.
func TestMixedAllWaitsForAction(t *testing.T) {
	l := textLesson("B01", Check{Kind: KindAll, Checks: []Check{
		contains("diario"),
		{Kind: KindAction, Event: "mode_changed", Value: Values{"normal"}},
	}})
	s := &Store{}
	s.RecheckText(l, "diario\n", Env{})
	if s.Passed(l, 1) {
		t.Fatal("the text alone passed a mixed step")
	}
	s.RecordAction(l, "mode_changed", "normal")
	if !s.Passed(l, 1) {
		t.Fatal("text and action should pass the step")
	}
}

// E3: a step with no check passes on a save that changed the note.
func TestNoneStepOnSave(t *testing.T) {
	l := textLesson("B26", Check{Kind: KindNone})
	s := &Store{}
	if s.RecordSave(l, "igual", "igual") {
		t.Fatal("an unchanged save passed the step")
	}
	if !s.RecordSave(l, "antes", "después") {
		t.Fatal("a changed save did not pass the step")
	}
	if !s.Passed(l, 1) || s.NextStep(l) != 0 {
		t.Fatal("the Desafío step should be done")
	}
}

func TestNextStepAnyOrder(t *testing.T) {
	l := textLesson("B03", contains("a"), contains("b"))
	s := &Store{}
	s.Mark(l, 2)
	if got := s.NextStep(l); got != 1 {
		t.Fatalf("NextStep = %d, want 1", got)
	}
	s.Mark(l, 1)
	if got := s.NextStep(l); got != 0 {
		t.Fatalf("NextStep = %d, want 0", got)
	}
	if !s.LessonDone(l) {
		t.Fatal("both steps should complete the lesson")
	}
}

func TestRouteCountAndDone(t *testing.T) {
	one := textLesson("B01", contains("a"))
	two := textLesson("B02", contains("b"))
	route := Route{ID: "B", Name: "Básico", Index: "B00 Básico.md", Lessons: []Lesson{one, two}}
	s := &Store{}
	if done, total := s.RouteCount(route); done != 0 || total != 2 {
		t.Fatalf("RouteCount = %d/%d", done, total)
	}
	s.Mark(one, 1)
	if s.RouteDone(route) {
		t.Fatal("one of two lessons is not the whole route")
	}
	s.Mark(two, 1)
	if !s.RouteDone(route) {
		t.Fatal("both lessons should finish the route")
	}
	var nilStore *Store
	if done, total := nilStore.RouteCount(route); done != 0 || total != 2 {
		t.Fatalf("nil RouteCount = %d/%d", done, total)
	}
	if nilStore.RouteDone(route) {
		t.Fatal("a nil store is never done")
	}
}

// D4: scan re-derives text and disk steps; action steps stay as stored.
func TestScanDir(t *testing.T) {
	dir := t.TempDir()
	cat := Catalog{Routes: []Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []Lesson{{
			ID: "F01", Title: "Fixture", File: "F01 Fixture.md",
			Steps: []Step{
				{Text: "Texto", Check: contains("hecho")},
				{Text: "Disco", Check: Check{Kind: KindDisk, Pred: "exists", Args: []string{"Mi nota"}}},
				{Text: "Acción", Check: Check{Kind: KindAction, Event: "next_tab"}},
			},
		}},
	}}}
	if err := os.WriteFile(filepath.Join(dir, "F01 Fixture.md"), []byte("hecho\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &Store{}
	env := Env{Exists: func(n string) bool { return n == "Mi nota" }}
	s.ScanDir(cat, dir, env)
	if !s.Passed(cat.Routes[0].Lessons[0], 1) || !s.Passed(cat.Routes[0].Lessons[0], 2) {
		t.Fatal("ScanDir did not derive the text and disk steps")
	}
	if s.Passed(cat.Routes[0].Lessons[0], 3) {
		t.Fatal("ScanDir passed an action step")
	}
}

// L4: ScanDir never passes the untouched installed lesson.
func TestScanDirIgnoresTheOriginal(t *testing.T) {
	dir := t.TempDir()
	data, err := fs.ReadFile(Bundled(), "B01 Títulos.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "B01 Títulos.md"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	cat := Default()
	s := &Store{}
	s.ScanDir(cat, dir, Env{})
	l, ok := cat.Lesson("B01 Títulos.md")
	if !ok {
		t.Fatal("B01 is not in the catalog")
	}
	if s.NextStep(l) != 1 {
		t.Fatal("the untouched lesson passed a step")
	}
	if err := os.WriteFile(filepath.Join(dir, "B01 Títulos.md"), append(data, []byte("\n## Viaje\n\n### Día 1\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	s.ScanDir(cat, dir, Env{})
	if !s.LessonDone(l) {
		t.Fatal("the player's own headings should pass the lesson")
	}
}

func TestNilStore(t *testing.T) {
	var s *Store
	l := textLesson("B01", contains("a"))
	if s.Passed(l, 1) || s.NextStep(l) != 0 || s.LessonDone(l) {
		t.Fatal("a nil store reported progress")
	}
	if s.Recheck(l, "a", Env{}) || s.RecheckText(l, "a", Env{}) || s.RecheckDisk(l, Env{}) {
		t.Fatal("a nil store reported a pass")
	}
	if s.RecordAction(l, "next_tab", "") || s.RecordSave(l, "a", "b") || s.Mark(l, 1) {
		t.Fatal("a nil store recorded progress")
	}
	if s.Migrated() || s.LastRoute() != "" {
		t.Fatal("a nil store has flags")
	}
	if err := s.Save(t.TempDir()); err == nil {
		t.Fatal("a nil store should not save")
	}
}

func TestScanDirMissingFile(t *testing.T) {
	cat := Catalog{Routes: []Route{{ID: "B", Name: "B", Index: "B00 B.md", Lessons: []Lesson{
		{ID: "B01", Title: "T", File: "B01 T.md", Steps: []Step{{Text: "x", Check: contains("x")}}},
	}}}}
	s := &Store{}
	if s.ScanDir(cat, t.TempDir(), Env{}) {
		t.Fatal("a missing note passed")
	}
	if s.ScanDir(cat, "", Env{}) {
		t.Fatal("an empty folder passed")
	}
}

func TestStepKey(t *testing.T) {
	if got := StepKey("B03", 2); got != "B03#2" {
		t.Fatalf("StepKey = %q", got)
	}
}

func TestSkippedSteps(t *testing.T) {
	dir := t.TempDir()
	l := textLesson("A07", contains("historia"))
	s := &Store{}
	if !s.MarkSkipped(l, 1) {
		t.Fatal("the first skip should be new")
	}
	if s.MarkSkipped(l, 1) {
		t.Fatal("a second skip of the same step is not new")
	}
	if !s.Skipped(l, 1) || !s.Passed(l, 1) {
		t.Fatal("a skipped step counts as done")
	}
	if s.NextStep(l) != 0 || !s.LessonDone(l) {
		t.Fatal("a skipped step should not block the lesson")
	}
	if !s.Started(l) {
		t.Fatal("a skipped step starts the lesson")
	}
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Skipped(l, 1) || !loaded.Passed(l, 1) {
		t.Fatal("Load lost the skip")
	}
	if !loaded.ClearSkipped(l) {
		t.Fatal("ClearSkipped should report the change")
	}
	if loaded.Skipped(l, 1) || loaded.Passed(l, 1) {
		t.Fatal("ClearSkipped left the skip behind")
	}
}

func TestRouteStarted(t *testing.T) {
	l := textLesson("B01", contains("a"))
	r := Route{ID: "B", Name: "Básico", Index: "B00 B.md", Lessons: []Lesson{l}}
	s := &Store{}
	if s.RouteStarted(r) {
		t.Fatal("a fresh route started")
	}
	s.Mark(l, 1)
	if !s.RouteStarted(r) {
		t.Fatal("a passed step should start the route")
	}
}

func TestRecordEvent(t *testing.T) {
	l := textLesson("B07", Check{Kind: KindAction, Event: "next_tab"})
	s := &Store{}
	if !s.RecordEvent(l, Event{Name: "next_tab"}) {
		t.Fatal("RecordEvent did not credit the action")
	}
	if !s.Passed(l, 1) {
		t.Fatal("the action step did not pass")
	}
}
