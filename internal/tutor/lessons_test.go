package tutor

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// S1/S7: every lesson of the catalog is embedded, has the lesson structure
// and starts unpassed, whatever its own text says.
func TestLessons(t *testing.T) {
	cat := Default()
	if len(cat.Routes) == 0 {
		t.Fatal("the catalog is empty")
	}
	bundled := Bundled()
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
			if !strings.Contains(text, "**Vas a aprender:**") {
				t.Errorf("%s has no learning line", l.Path())
			}
			if !strings.Contains(text, "## Tu turno") {
				t.Errorf("%s has no ## Tu turno", l.Path())
			}
			// L4: the untouched lesson never passes, not even on its own text.
			untouched := &Store{}
			untouched.Recheck(l, text, Env{})
			if got := untouched.NextStep(l); got != 1 {
				t.Errorf("%s starts passed (next step %d)", l.Path(), got)
			}
		}
	}
}

// The index links every lesson of its route.
func TestIndexLinksEveryLesson(t *testing.T) {
	cat := Default()
	for _, r := range cat.Routes {
		data, err := fs.ReadFile(Bundled(), r.IndexPath())
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

// E14: the tutorial ships one small original image, embedded with the notes.
func TestBundledImage(t *testing.T) {
	info, err := fs.Stat(Bundled(), "mapa.png")
	if err != nil {
		t.Fatalf("mapa.png is not embedded: %v", err)
	}
	if info.Size() > 50*1024 {
		t.Fatalf("mapa.png is %d bytes, want under 50 KB", info.Size())
	}
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
		if _, err := fs.Stat(Bundled(), path); err != nil {
			t.Errorf("%s is not embedded: %v", path, err)
		}
	}
	entries, err := fs.ReadDir(Bundled(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != ReferenceNote {
			t.Errorf("%s sits at the vault root; lessons live in their route's folder", e.Name())
		}
	}
}

// S6/F4: a link to a tutorial note resolves by bare name from the note that
// writes it, whatever folder either lives in ("Siguiente:" links included).
func TestTutorialLinksResolveAcrossRouteFolders(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tutor")
	if err := Install(Bundled(), dir, false); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	notes := store.List()
	tutorial := map[string]bool{}
	var paths []string
	for _, r := range Default().Routes {
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
// A09 (now in "3 Avanzado") still finds it from its own folder.
func TestMapResolvesFromTheAdvancedLesson(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tutor")
	if err := Install(Bundled(), dir, false); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := store.FindFile("mapa.png", "3 Avanzado/A09 Incrustar.md")
	if !ok {
		t.Fatal("![[mapa.png]] does not resolve from A09's folder")
	}
	want := filepath.Join(dir, "mapa.png")
	if got != want {
		t.Fatalf("mapa.png resolved to %q, want the vault root %q", got, want)
	}
}

// L7: each route's last lesson links the next route's index, and the last
// route links the reference note.
func TestLastLessonLinksTheNextRoute(t *testing.T) {
	cat := Default()
	for i, r := range cat.Routes {
		if len(r.Lessons) == 0 {
			continue
		}
		want := ReferenceNote
		if i+1 < len(cat.Routes) {
			want = cat.Routes[i+1].Index
		}
		last := r.Lessons[len(r.Lessons)-1]
		data, err := fs.ReadFile(Bundled(), last.Path())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "[["+strings.TrimSuffix(want, ".md")+"]]") {
			t.Errorf("%s does not link %s", last.Path(), want)
		}
	}
}
