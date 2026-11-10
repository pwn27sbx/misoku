package tutor

import (
	"io/fs"
	"strings"
	"testing"
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
		if _, err := fs.ReadFile(bundled, r.Index); err != nil {
			t.Errorf("route %s index %s not embedded: %v", r.ID, r.Index, err)
		}
		for _, l := range r.Lessons {
			lessons++
			steps += len(l.Steps)
			data, err := fs.ReadFile(bundled, l.File)
			if err != nil {
				t.Errorf("%s not embedded: %v", l.File, err)
				continue
			}
			text := string(data)
			if !strings.HasPrefix(text, "# "+l.ID+" · ") {
				t.Errorf("%s does not start with its title", l.File)
			}
			if !strings.Contains(text, "**Vas a aprender:**") {
				t.Errorf("%s has no learning line", l.File)
			}
			if !strings.Contains(text, "## Tu turno") {
				t.Errorf("%s has no ## Tu turno", l.File)
			}
			// L4: the untouched lesson never passes, not even on its own text.
			untouched := &Store{}
			untouched.Recheck(l, text, Env{})
			if got := untouched.NextStep(l); got != 1 {
				t.Errorf("%s starts passed (next step %d)", l.File, got)
			}
		}
	}
}

// The index links every lesson of its route.
func TestIndexLinksEveryLesson(t *testing.T) {
	cat := Default()
	for _, r := range cat.Routes {
		data, err := fs.ReadFile(Bundled(), r.Index)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range r.Lessons {
			link := "[[" + strings.TrimSuffix(l.File, ".md") + "]]"
			if !strings.Contains(string(data), link) {
				t.Errorf("%s does not link %s", r.Index, link)
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
		data, err := fs.ReadFile(Bundled(), last.File)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "[["+strings.TrimSuffix(want, ".md")+"]]") {
			t.Errorf("%s does not link %s", last.File, want)
		}
	}
}
