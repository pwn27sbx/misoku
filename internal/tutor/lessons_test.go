package tutor

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// lessonMarkers are the learning line and the practice heading every lesson
// of a language has.
var lessonMarkers = map[i18n.Lang][2]string{
	i18n.ES: {"**Vas a aprender:**", "## Tu turno"},
	i18n.EN: {"**You will learn:**", "## Your turn"},
}

// startsPassedPending are the lessons of a draft translation known to pass a
// step untouched, with the task that reconciles them. TestLessons logs them
// instead of failing, and fails once one is fixed so the entry goes.
var startsPassedPending = map[i18n.Lang]map[string]string{
	i18n.EN: {
		// absent "oops" ignores code, and the lesson names `oops` only in
		// code, so step 1 passes before the player types it (L16 has the
		// Spanish twin, which never passes).
		"B05": "T6 reconciles the English route B",
	},
}

// forEachLang runs f once per language, as a subtest named after it, with
// that language's catalog and notes.
func forEachLang(t *testing.T, f func(t *testing.T, l i18n.Lang, cat Catalog, bundled fs.FS)) {
	for _, l := range i18n.Langs {
		t.Run(string(l), func(t *testing.T) {
			cat, err := BundledRoutesFor(l)
			if err != nil {
				t.Fatalf("the embedded catalog is invalid: %v", err)
			}
			f(t, l, cat, BundledFor(l))
		})
	}
}

// S1/S7: every lesson of the catalog is embedded, has the lesson structure
// and starts unpassed, whatever its own text says.
func TestLessons(t *testing.T) {
	forEachLang(t, testLessons)
}

func testLessons(t *testing.T, lang i18n.Lang, cat Catalog, bundled fs.FS) {
	if len(cat.Routes) == 0 {
		t.Fatal("the catalog is empty")
	}
	learn, turn := lessonMarkers[lang][0], lessonMarkers[lang][1]
	lessons, steps := 0, 0
	defer func() { t.Logf("baseline: %d lessons, %d steps; none passes untouched", lessons, steps) }()
	for _, r := range cat.Routes {
		if _, err := fs.ReadFile(bundled, r.IndexPath()); err != nil {
			t.Errorf("route %s index %s not embedded: %v", r.ID, r.IndexPath(), err)
		}
		for _, l := range r.Lessons {
			lessons++
			steps += len(l.Steps)
			data, err := fs.ReadFile(bundled, l.Path())
			if err != nil {
				t.Errorf("%s not embedded: %v", l.Path(), err)
				continue
			}
			text := string(data)
			if !strings.HasPrefix(text, "# "+l.ID+" · ") {
				t.Errorf("%s does not start with its title", l.Path())
			}
			if !strings.Contains(text, learn) {
				t.Errorf("%s has no learning line %s", l.Path(), learn)
			}
			if !strings.Contains(text, turn) {
				t.Errorf("%s has no %s", l.Path(), turn)
			}
			// L4: the untouched lesson never passes, not even on its own text.
			untouched := &Store{}
			untouched.Recheck(l, text, Env{})
			got := untouched.NextStep(l)
			switch why, pending := startsPassedPending[lang][l.ID]; {
			case pending && got != 1:
				t.Logf("%s starts passed (next step %d), pending: %s", l.Path(), got, why)
			case pending:
				t.Errorf("%s no longer starts passed: remove it from startsPassedPending", l.Path())
			case got != 1:
				t.Errorf("%s starts passed (next step %d)", l.Path(), got)
			}
		}
	}
}

// The index links every lesson of its route.
func TestIndexLinksEveryLesson(t *testing.T) {
	forEachLang(t, func(t *testing.T, _ i18n.Lang, cat Catalog, bundled fs.FS) {
		testIndexLinksEveryLesson(t, cat, bundled)
	})
}

func testIndexLinksEveryLesson(t *testing.T, cat Catalog, bundled fs.FS) {
	for _, r := range cat.Routes {
		data, err := fs.ReadFile(bundled, r.IndexPath())
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range r.Lessons {
			link := "[[" + strings.TrimSuffix(l.File, ".md") + "]]"
			if !strings.Contains(string(data), link) {
				t.Errorf("%s does not link %s", r.IndexPath(), link)
			}
		}
	}
}

// E14/S5: the tutorial ships one small original image, embedded with the
// notes of every language.
func TestBundledImage(t *testing.T) {
	forEachLang(t, func(t *testing.T, _ i18n.Lang, _ Catalog, bundled fs.FS) {
		info, err := fs.Stat(bundled, "mapa.png")
		if err != nil {
			t.Fatalf("mapa.png is not embedded: %v", err)
		}
		if info.Size() > 50*1024 {
			t.Fatalf("mapa.png is %d bytes, want under 50 KB", info.Size())
		}
	})
}

// S1/G1: the embed keeps the route folders, so each lesson is copied with its
// folder and the reference note stays at the root.
func TestBundledKeepsTheRouteFolders(t *testing.T) {
	for _, path := range []string{
		"1 Básico/B01 Escribe en 30 segundos.md",
		"2 Intermedio/I01 Palabras, páginas y desplazamiento fino.md",
		"3 Avanzado/A09 Incrustar.md",
		"4 Experto/E01 El mapa de atajos.md",
	} {
		if _, err := fs.Stat(BundledFor(i18n.ES), path); err != nil {
			t.Errorf("%s is not embedded: %v", path, err)
		}
	}
	forEachLang(t, func(t *testing.T, l i18n.Lang, cat Catalog, bundled fs.FS) {
		for _, r := range cat.Routes {
			for _, les := range r.Lessons {
				if _, err := fs.Stat(bundled, r.Dir+"/"+les.File); err != nil {
					t.Errorf("%s is not embedded in its route's folder: %v", les.File, err)
				}
			}
		}
		entries, err := fs.ReadDir(bundled, ".")
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != ReferenceNoteFor(l) {
				t.Errorf("%s sits at the vault root; lessons live in their route's folder", e.Name())
			}
		}
	})
}

// S6/F4: a link to a tutorial note resolves by bare name from the note that
// writes it, whatever folder either lives in ("Siguiente:" links included).
func TestTutorialLinksResolveAcrossRouteFolders(t *testing.T) {
	forEachLang(t, func(t *testing.T, _ i18n.Lang, cat Catalog, bundled fs.FS) {
		testTutorialLinksResolve(t, cat, bundled)
	})
}

func testTutorialLinksResolve(t *testing.T, cat Catalog, bundled fs.FS) {
	dir := filepath.Join(t.TempDir(), "tutor")
	if err := Install(bundled, dir, false); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	notes := store.List()
	tutorial := map[string]bool{}
	var paths []string
	for _, r := range cat.Routes {
		tutorial[strings.TrimSuffix(r.Index, ".md")] = true
		paths = append(paths, r.IndexPath())
		for _, l := range r.Lessons {
			tutorial[strings.TrimSuffix(l.File, ".md")] = true
			paths = append(paths, l.Path())
		}
	}
	link := regexp.MustCompile(`\[\[([^\[\]\n]*)\]\]`)
	checked := 0
	for _, note := range paths {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(note)))
		if err != nil {
			t.Errorf("%s: %v", note, err)
			continue
		}
		for _, m := range link.FindAllStringSubmatch(string(data), -1) {
			target, _, _ := strings.Cut(m[1], "|")
			name, _, _ := strings.Cut(strings.TrimSpace(target), "#")
			name = strings.TrimSpace(name)
			// Only links to the tutorial's own notes: the lessons also show
			// links to notes the player creates.
			if name == "" || strings.Contains(name, "/") || !tutorial[name] {
				continue
			}
			if _, ok := core.ResolveNote(notes, name, note); !ok {
				t.Errorf("%s links [[%s]], which does not resolve", note, target)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no cross-folder tutorial link was checked")
	}
}

// S6/G7: the shared image stays at the vault root, and the wikilink embed of
// A09 (in the Advanced route's folder) still finds it from its own folder.
func TestMapResolvesFromTheAdvancedLesson(t *testing.T) {
	forEachLang(t, func(t *testing.T, _ i18n.Lang, cat Catalog, bundled fs.FS) {
		a09, ok := cat.LessonByID("A09")
		if !ok {
			t.Fatal("A09 is not in the catalog")
		}
		dir := filepath.Join(t.TempDir(), "tutor")
		if err := Install(bundled, dir, false); err != nil {
			t.Fatal(err)
		}
		store, err := core.NewStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := store.FindFile("mapa.png", a09.Path())
		if !ok {
			t.Fatal("![[mapa.png]] does not resolve from A09's folder")
		}
		want := filepath.Join(dir, "mapa.png")
		if got != want {
			t.Fatalf("mapa.png resolved to %q, want the vault root %q", got, want)
		}
	})
}

// L7: each route's last lesson links the next route's index, and the last
// route links the reference note.
func TestLastLessonLinksTheNextRoute(t *testing.T) {
	forEachLang(t, func(t *testing.T, l i18n.Lang, cat Catalog, bundled fs.FS) {
		testLastLessonLinksTheNextRoute(t, cat, bundled, ReferenceNoteFor(l))
	})
}

func testLastLessonLinksTheNextRoute(t *testing.T, cat Catalog, bundled fs.FS, reference string) {
	for i, r := range cat.Routes {
		if len(r.Lessons) == 0 {
			continue
		}
		want := reference
		if i+1 < len(cat.Routes) {
			want = cat.Routes[i+1].Index
		}
		last := r.Lessons[len(r.Lessons)-1]
		data, err := fs.ReadFile(bundled, last.Path())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "[["+strings.TrimSuffix(want, ".md")+"]]") {
			t.Errorf("%s does not link %s", last.Path(), want)
		}
	}
}
