package tutor

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// fakeConfigHome points HOME, the XDG folders and their Windows equivalents
// at a temporary folder, so DirFor and the Trash are never the user's. It
// returns the folder that holds the config file.
func fakeConfigHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	t.Setenv("USERPROFILE", home)
	p := config.DefaultPath()
	if p == "" {
		t.Fatal("no config path in the fake home")
	}
	return filepath.Dir(p)
}

// snapshot reads every file under dir, by slash path, so a test can tell
// whether anything in it changed.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			files[filepath.ToSlash(rel)+"/"] = ""
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// A2: each language has its own tutorial folder. The Spanish one is the
// folder every player had before, so an existing install is where it was.
func TestDirPerLanguage(t *testing.T) {
	cfg := fakeConfigHome(t)
	if got, want := DirFor(i18n.ES), filepath.Join(cfg, "tutor"); got != want {
		t.Errorf("DirFor(es) = %q, want %q", got, want)
	}
	if got, want := DirFor(i18n.EN), filepath.Join(cfg, "tutor-en"); got != want {
		t.Errorf("DirFor(en) = %q, want %q", got, want)
	}
	useLang(t, i18n.EN)
	if Dir() != DirFor(i18n.EN) {
		t.Errorf("Dir() = %q in English, want %q", Dir(), DirFor(i18n.EN))
	}
	i18n.Set(i18n.ES)
	if Dir() != DirFor(i18n.ES) {
		t.Errorf("Dir() = %q in Spanish, want %q", Dir(), DirFor(i18n.ES))
	}
}

// A5/S5: the tutorial played is the one in the interface language: its
// catalog, its notes and its reference note.
func TestBundledPerLanguage(t *testing.T) {
	want := map[i18n.Lang]struct{ name, dir, lesson, ref string }{
		i18n.ES: {"Básico", "1 Básico", "1 Básico/B01 Escribe en 30 segundos.md", "Después del tutorial.md"},
		i18n.EN: {"Basic", "1 Basic", "1 Basic/B01 Write in 30 seconds.md", "After the tutorial.md"},
	}
	for _, l := range i18n.Langs {
		w := want[l]
		cat, err := BundledRoutesFor(l)
		if err != nil {
			t.Fatalf("%s: the embedded catalog is invalid: %v", l, err)
		}
		b, ok := cat.RouteByID("B")
		if !ok || b.Name != w.name || b.Dir != w.dir {
			t.Errorf("%s: route B = %q in %q, want %q in %q", l, b.Name, b.Dir, w.name, w.dir)
		}
		if _, ok := cat.Lesson(w.lesson); !ok {
			t.Errorf("%s: %s is not in the catalog", l, w.lesson)
		}
		for _, path := range []string{w.lesson, w.ref, "mapa.png"} {
			if _, err := fs.Stat(BundledFor(l), path); err != nil {
				t.Errorf("%s: %s is not bundled: %v", l, path, err)
			}
		}
		if got := ReferenceNoteFor(l); got != w.ref {
			t.Errorf("ReferenceNoteFor(%s) = %q, want %q", l, got, w.ref)
		}
	}
	// The functions without a language follow the interface.
	useLang(t, i18n.EN)
	if b, _ := Default().RouteByID("B"); b.Name != "Basic" {
		t.Errorf("Default() in English has route B %q, want Basic", b.Name)
	}
	if _, err := fs.Stat(Bundled(), want[i18n.EN].lesson); err != nil {
		t.Errorf("Bundled() in English lacks the English B01: %v", err)
	}
	i18n.Set(i18n.ES)
	if b, _ := Default().RouteByID("B"); b.Name != "Básico" {
		t.Errorf("Default() in Spanish has route B %q, want Básico", b.Name)
	}
}

// S6/risk 1: the Spanish bundle is exactly the tree every player installed
// before the English one existed: the same root entries, no language folder
// and no English note. Install copies only what a folder lacks, so an
// existing Spanish tutorial gets nothing new.
func TestSpanishBundleIsTheTreeInstalledBefore(t *testing.T) {
	entries, err := fs.ReadDir(BundledFor(i18n.ES), ".")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	want := []string{"1 Básico", "2 Intermedio", "3 Avanzado", "4 Experto", "Después del tutorial.md", "mapa.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the Spanish bundle's root = %q, want %q", got, want)
	}
}

// withoutOriginal strips only the lessons/<lang>/ prefix, so two languages
// must never bundle a note at the same path: one's text would hide the
// other's.
func TestNoNoteIsInBothLanguages(t *testing.T) {
	seen := map[string]i18n.Lang{}
	for _, l := range i18n.Langs {
		err := fs.WalkDir(BundledFor(l), ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			if other, dup := seen[path]; dup && other != l {
				t.Errorf("%s is bundled in %s and %s", path, other, l)
			}
			seen[path] = l
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// And the subtraction finds the English lesson's own text.
	text := "# B01 · Write in 30 seconds\n"
	if got := withoutOriginal("1 Basic/B01 Write in 30 seconds.md", text); strings.TrimSpace(got) != "" {
		t.Errorf("the English lesson's own title was not subtracted: %q", got)
	}
}

// twoLanguageTrees is a tutorial whose Spanish side has routes B and I and
// whose English side has only I, laid out as the embed is.
func twoLanguageTrees(withEnglishRef bool) (lessons, routes fstest.MapFS) {
	note := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }
	lessons = fstest.MapFS{
		"es/1 Básico/B00 Básico.md":             note("# Básico\n"),
		"es/1 Básico/B01 Uno.md":                note("# B01 · Uno\n"),
		"es/2 Intermedio/I00 Intermedio.md":     note("# Intermedio\n"),
		"es/2 Intermedio/I01 Dos.md":            note("# I01 · Dos\n"),
		"es/Después del tutorial.md":            note("# Después\n"),
		"es/mapa.png":                           note("png"),
		"en/2 Intermediate/I00 Intermediate.md": note("# Intermediate\n"),
		"en/2 Intermediate/I01 Two.md":          note("# I01 · Two\n"),
	}
	if withEnglishRef {
		lessons["en/After the tutorial.md"] = note("# After\n")
	}
	route := func(id, name, dir, index, lesson, title, file string) *fstest.MapFile {
		return note(`{"id":"` + id + `","name":"` + name + `","dir":"` + dir + `","index":"` + index +
			`","lessons":[{"id":"` + lesson + `","title":"` + title + `","file":"` + file +
			`","steps":[{"text":"x","check":{"kind":"none"}}]}]}`)
	}
	routes = fstest.MapFS{
		"es/b.json": route("B", "Básico", "1 Básico", "B00 Básico.md", "B01", "Uno", "B01 Uno.md"),
		"es/i.json": route("I", "Intermedio", "2 Intermedio", "I00 Intermedio.md", "I01", "Dos", "I01 Dos.md"),
		"en/i.json": route("I", "Intermediate", "2 Intermediate", "I00 Intermediate.md", "I01", "Two", "I01 Two.md"),
	}
	return lessons, routes
}

// S10: a route with no English version yet is played in Spanish inside the
// English tutorial, with its notes; the routes that have one are English.
func TestEnglishFallsBackToSpanishPerRoute(t *testing.T) {
	lessons, routes := twoLanguageTrees(true)
	b, err := buildBundle(lessons, routes, i18n.EN)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range b.cat.Routes {
		names = append(names, r.ID+":"+r.Name)
	}
	if want := []string{"B:Básico", "I:Intermediate"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("routes = %q, want %q", names, want)
	}
	for _, path := range []string{"1 Básico/B01 Uno.md", "1 Básico/B00 Básico.md", "2 Intermediate/I01 Two.md", "After the tutorial.md", "mapa.png"} {
		if _, err := fs.Stat(b.files, path); err != nil {
			t.Errorf("%s is not in the English tree: %v", path, err)
		}
	}
	for _, path := range []string{"2 Intermedio/I01 Dos.md", "Después del tutorial.md"} {
		if _, err := fs.Stat(b.files, path); err == nil {
			t.Errorf("%s is in the English tree, but English has its own", path)
		}
	}
	var root []string
	entries, err := fs.ReadDir(b.files, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		root = append(root, e.Name())
	}
	if want := []string{"1 Básico", "2 Intermediate", "After the tutorial.md", "mapa.png"}; !reflect.DeepEqual(root, want) {
		t.Errorf("the English tree's root = %q, want %q", root, want)
	}
	if b.ref != "After the tutorial.md" {
		t.Errorf("reference note = %q, want the English one", b.ref)
	}
	// Installing it copies the fallback route's Spanish notes too.
	dir := filepath.Join(t.TempDir(), "tutor-en")
	if err := Install(b.files, dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "1 Básico", "B01 Uno.md")); err != nil {
		t.Errorf("the fallback route was not installed: %v", err)
	}
}

// S10: with no English reference note yet, the Spanish one is the English
// tutorial's; with no English route at all, the English tutorial is the
// Spanish one.
func TestEnglishFallsBackToTheSpanishReferenceNote(t *testing.T) {
	lessons, routes := twoLanguageTrees(false)
	b, err := buildBundle(lessons, routes, i18n.EN)
	if err != nil {
		t.Fatal(err)
	}
	if b.ref != "Después del tutorial.md" {
		t.Errorf("reference note = %q, want the Spanish one", b.ref)
	}
	if _, err := fs.Stat(b.files, "Después del tutorial.md"); err != nil {
		t.Errorf("the Spanish reference note is not in the English tree: %v", err)
	}

	delete(routes, "en/i.json")
	for path := range lessons {
		if strings.HasPrefix(path, "en/") {
			delete(lessons, path)
		}
	}
	b, err = buildBundle(lessons, routes, i18n.EN)
	if err != nil {
		t.Fatal(err)
	}
	es, err := buildBundle(lessons, routes, i18n.ES)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(routeList(b.cat), routeList(es.cat)) || len(b.cat.Routes) != 2 {
		t.Errorf("with no English route the catalog is %v, want Spanish %v", routeList(b.cat), routeList(es.cat))
	}
	if _, err := fs.Stat(b.files, "2 Intermedio/I01 Dos.md"); err != nil {
		t.Errorf("with no English route the Spanish notes are missing: %v", err)
	}
}

// routeList lists a catalog's routes as "ID:Name".
func routeList(c Catalog) []string {
	var out []string
	for _, r := range c.Routes {
		out = append(out, r.ID+":"+r.Name)
	}
	return out
}

// Risk 1, A2/A3/S6: an existing Spanish tutorial, with an edited lesson, a
// note of the player's and its progress, is not touched by installing,
// playing or resetting the English one, nor by installing Spanish again.
func TestTheSpanishTutorialSurvivesTheEnglishOne(t *testing.T) {
	fakeConfigHome(t)
	es := DirFor(i18n.ES)
	if err := Install(BundledFor(i18n.ES), es, false); err != nil {
		t.Fatal(err)
	}
	lesson := filepath.Join(es, "1 Básico", "B01 Escribe en 30 segundos.md")
	if err := os.WriteFile(lesson, []byte("Hoy empiezo mi diario de viaje\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(es, "Mi nota.md"), []byte("mía\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	progress, err := Load(es)
	if err != nil {
		t.Fatal(err)
	}
	b01, _ := CatalogFor(i18n.ES).LessonByID("B01")
	progress.Mark(b01, 1)
	progress.SetLastRoute("B")
	if err := progress.Save(es); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, es)

	useLang(t, i18n.EN)
	if Dir() == es {
		t.Fatalf("the English tutorial would use the Spanish folder %q", es)
	}
	en := Dir()
	if _, err := InstallWithNotice(Bundled(), en, false); err != nil {
		t.Fatal(err)
	}
	enProgress, err := Load(en)
	if err != nil {
		t.Fatal(err)
	}
	enB01, _ := Default().LessonByID("B01")
	if enProgress.Passed(enB01, 1) {
		t.Error("the English tutorial starts with the Spanish progress")
	}
	enB02, _ := Default().LessonByID("B02")
	enProgress.Mark(enB02, 1)
	if err := enProgress.Save(en); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallWithNotice(Bundled(), en, true); err != nil { // --reset in English
		t.Fatal(err)
	}
	if after := snapshot(t, es); !reflect.DeepEqual(before, after) {
		t.Fatalf("the English tutorial changed the Spanish folder:\nbefore %v\nafter  %v", keys(before), keys(after))
	}

	// Back in Spanish, the install every start runs adds and changes nothing.
	i18n.Set(i18n.ES)
	if _, err := InstallWithNotice(Bundled(), Dir(), false); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(t, es); !reflect.DeepEqual(before, after) {
		t.Fatalf("installing Spanish again changed its folder:\nbefore %v\nafter  %v", keys(before), keys(after))
	}
	reloaded, err := Load(Dir())
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Passed(b01, 1) || reloaded.LastRoute() != "B" {
		t.Fatal("the Spanish progress was lost")
	}
}

// keys lists a snapshot's paths, sorted, for a failure message.
func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// shape is what the shape test compares of one check: its kind, predicate,
// fence and event, how many values it waits for and its sub-checks.
func shape(c Check) string {
	s := c.Kind + "/" + c.Pred + "/" + c.Fence + "/" + c.Event + "/" + strings.Repeat("v", len(c.Value))
	if len(c.Checks) > 0 {
		var subs []string
		for _, sub := range c.Checks {
			subs = append(subs, shape(sub))
		}
		s += "[" + strings.Join(subs, ",") + "]"
	}
	return s
}

// The shape test: every route that has an English version has the Spanish
// one's lessons, steps and checks, in the same order, so a lesson ID and a
// progress key ("B03#2") mean the same step in both languages. What a check
// looks for is the lesson's own text and is not compared.
func TestEnglishRoutesHaveTheSpanishShape(t *testing.T) {
	es, err := loadRoutes(routesFS, "routes/es")
	if err != nil {
		t.Fatal(err)
	}
	en, err := loadRoutes(routesFS, "routes/en")
	if err != nil {
		t.Fatal(err)
	}
	if len(en) == 0 {
		t.Skip("no English route is embedded yet")
	}
	spanish := map[string]Route{}
	for _, r := range es {
		spanish[r.ID] = r
	}
	compared := 0
	for _, e := range en {
		s, ok := spanish[e.ID]
		if !ok {
			t.Errorf("English route %s has no Spanish route", e.ID)
			continue
		}
		if len(e.Lessons) != len(s.Lessons) {
			t.Errorf("route %s: %d English lessons, %d Spanish", e.ID, len(e.Lessons), len(s.Lessons))
			continue
		}
		for i, el := range e.Lessons {
			sl := s.Lessons[i]
			if el.ID != sl.ID {
				t.Errorf("route %s lesson %d: English %s, Spanish %s", e.ID, i+1, el.ID, sl.ID)
				continue
			}
			if len(el.Steps) != len(sl.Steps) {
				t.Errorf("%s: %d English steps, %d Spanish", el.ID, len(el.Steps), len(sl.Steps))
				continue
			}
			for j := range el.Steps {
				if got, want := shape(el.Steps[j].Check), shape(sl.Steps[j].Check); got != want {
					t.Errorf("%s step %d: English check %s, Spanish %s", el.ID, j+1, got, want)
				}
				compared++
			}
		}
	}
	t.Logf("%d English routes, %d steps compared", len(en), compared)
}

// The shape test itself must catch a moved step and a different predicate.
func TestShapeCatchesADifferentCheck(t *testing.T) {
	a := Check{Kind: KindAll, Checks: []Check{{Kind: KindText, Pred: "line-contains", Args: []string{"hola"}}}}
	b := Check{Kind: KindAll, Checks: []Check{{Kind: KindText, Pred: "line-contains", Args: []string{"hello"}}}}
	if shape(a) != shape(b) {
		t.Fatal("the text a check looks for is not part of its shape")
	}
	c := Check{Kind: KindAll, Checks: []Check{{Kind: KindText, Pred: "line-starts", Args: []string{"hola"}}}}
	if shape(a) == shape(c) {
		t.Fatal("a predicate inside all is part of the shape")
	}
	d := Check{Kind: KindAction, Event: "paste", Value: Values{"a", "b"}}
	e := Check{Kind: KindAction, Event: "paste", Value: Values{"a"}}
	if shape(d) == shape(e) {
		t.Fatal("the number of values is part of the shape")
	}
}
