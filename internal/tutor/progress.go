package tutor

import (
	"os"
	"path/filepath"
	"strings"
)

// Progress tracks which levels pass. It is derived from the notes' text, never
// stored. A nil *Progress is a valid, empty tracker (the tutorial is off).
type Progress struct {
	done map[string]bool // keyed by Lesson.File: passes now
	seen map[string]bool // has passed at least once (already announced)
}

// Scan checks every lesson note on disk in dir. It reads files, so call it
// before the program starts, never from an update loop.
func Scan(dir string) *Progress {
	p := &Progress{done: map[string]bool{}, seen: map[string]bool{}}
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name+".md"))
		return err == nil
	}
	for _, l := range Lessons {
		if l.Check == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, l.File))
		if err != nil {
			continue
		}
		if Passed(l.Check, string(data), exists) {
			p.done[l.File] = true
			p.seen[l.File] = true
		}
	}
	return p
}

// Update re-checks the lesson note file (a vault-relative path, with or
// without ".md") against its live text. It reports true only the first time
// that level passes; later passes, and notes that are not levels, report false.
func (p *Progress) Update(file, text string, exists func(string) bool) (newlyPassed bool) {
	if p == nil {
		return false
	}
	l, ok := lesson(file)
	if !ok || l.Check == "" {
		return false
	}
	passed := Passed(l.Check, text, exists)
	p.done[l.File] = passed
	if passed && !p.seen[l.File] {
		p.seen[l.File] = true
		return true
	}
	return false
}

// Count returns how many levels pass now and how many levels there are.
func (p *Progress) Count() (done, total int) {
	if p == nil {
		return 0, 0
	}
	for _, l := range Lessons {
		if l.Check == "" {
			continue
		}
		total++
		if p.done[l.File] {
			done++
		}
	}
	return done, total
}

// Level returns the level number of the lesson note file (1-based), or 0 when
// file is the index or not a lesson.
func (p *Progress) Level(file string) int {
	name := lessonFile(file)
	for i, l := range Lessons {
		if l.File == name {
			return i
		}
	}
	return 0
}

func lesson(file string) (Lesson, bool) {
	name := lessonFile(file)
	for _, l := range Lessons {
		if l.File == name {
			return l, true
		}
	}
	return Lesson{}, false
}

// lessonFile normalises a vault-relative path to a lesson file name. Lessons
// live at the vault root, so a path inside a folder never matches.
func lessonFile(file string) string {
	f := filepath.ToSlash(file)
	if strings.Contains(f, "/") {
		return ""
	}
	if !strings.HasSuffix(f, ".md") {
		f += ".md"
	}
	return f
}
