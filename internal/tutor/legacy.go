package tutor

// This file is the bridge for the app's pre-routes code path. F3a2 moves the
// app to Catalog and Store, and this file goes away; until then everything
// here is derived from the new engine, so there is one source of truth.

// Lessons flattens the default catalog for the old API: the route index first
// (its Level is 0), then its lessons in play order.
var Lessons = flatten(Default())

// Progress is the app's old view of the tutorial: the levels that pass, read
// from the notes' text. A nil *Progress is a valid, empty tracker.
type Progress struct {
	store *Store
	cat   Catalog
}

// Scan checks every lesson note on disk in dir. It reads files, so call it
// before the program starts, never from an update loop.
func Scan(dir string) *Progress {
	p := &Progress{cat: Default()}
	if dir == "" {
		return p
	}
	s, err := Load(dir)
	if err != nil {
		// The old path has no way to report it: start empty and let the
		// next save rewrite the file.
		s = &Store{}
	}
	p.store = s
	s.ScanDir(p.cat, dir, dirEnv(dir))
	return p
}

// catalog is the injected catalog, or the bundled one on a zero Progress.
func (p *Progress) catalog() Catalog {
	if p != nil && len(p.cat.Routes) > 0 {
		return p.cat
	}
	return Default()
}

// Update re-checks the lesson note file against its live text. It reports
// true only the first time that level passes.
func (p *Progress) Update(file, text string, exists func(string) bool) bool {
	if p == nil {
		return false
	}
	l, ok := p.catalog().Lesson(file)
	if !ok {
		return false
	}
	before := p.store.LessonDone(l)
	p.store.Recheck(l, text, Env{Exists: exists})
	return !before && p.store.LessonDone(l)
}

// Count returns how many lessons pass now and how many there are.
func (p *Progress) Count() (done, total int) {
	if p == nil {
		return 0, 0
	}
	for _, r := range p.catalog().Routes {
		d, t := p.store.RouteCount(r)
		done += d
		total += t
	}
	return done, total
}

// Level returns the level number of the lesson note file (1-based), or 0 when
// file is the index or not a lesson.
func (p *Progress) Level(file string) int {
	if p == nil {
		return 0
	}
	cat := p.catalog()
	name := lessonFile(file)
	for i, l := range flatten(cat) {
		if l.File == name {
			return i
		}
	}
	return 0
}

// Hint returns the hint of a lesson's first unpassed step, or "". It is the
// old path's hint; F3a2 reads Step.Hint from the current step instead.
func Hint(file, text string) string {
	l, ok := Default().Lesson(file)
	if !ok {
		return ""
	}
	for _, st := range l.Steps {
		if st.Hint != "" && !st.Check.Pass(text, Env{}) {
			return st.Hint
		}
	}
	return ""
}

// flatten turns the catalog into the old flat lesson list: each route's index
// followed by its lessons.
func flatten(cat Catalog) []Lesson {
	var out []Lesson
	for _, r := range cat.Routes {
		out = append(out, Lesson{ID: r.ID + "00", File: r.Index, Title: r.Name})
		out = append(out, r.Lessons...)
	}
	return out
}
