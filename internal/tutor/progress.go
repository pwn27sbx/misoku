package tutor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/pwn27sbx/misoku/internal/core"
)

const (
	// progressFile is the tutorial's stored progress. The store already
	// hides dot-files from the explorer (store.go walk), so the player
	// never sees it. A tutorial vault synced with git or a cloud folder
	// would conflict on this file, because two machines change it at once:
	// no action for now, the tutorial vault is local.
	progressFile = ".misoku-tutor.json"
	// progressVersion is the file's schema. A newer file is refused, never
	// overwritten.
	progressVersion = 1
)

// progressState is the file's content. Steps holds a key per passed step
// ("B03#2") and, for sub-checks and action values, their own keys
// ("B03#2.1", "B03#2.1=2"); a step's key means the whole step passed.
type progressState struct {
	Version   int             `json:"version"`
	Migrated  bool            `json:"migrated,omitempty"`
	LastRoute string          `json:"last_route,omitempty"`
	Steps     map[string]bool `json:"steps"`
}

// Store is a tutorial's progress. Text and disk steps are re-derived from the
// notes; action steps and sub-checks are sticky, so the file remembers them.
// A nil *Store is a valid, empty tracker (the tutorial is off).
type Store struct {
	mu    sync.Mutex
	state progressState
}

// Load reads the progress in dir. A missing file is an empty store; a file
// from a newer version or malformed JSON is an error, never silently
// overwritten.
func Load(dir string) (*Store, error) {
	s := &Store{state: progressState{Version: progressVersion, Steps: map[string]bool{}}}
	if dir == "" {
		return s, errors.New("tutor: no folder for the tutorial")
	}
	path := filepath.Join(dir, progressFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	var st progressState
	if err := json.Unmarshal(data, &st); err != nil {
		return s, fmt.Errorf("tutor: progress in %s: %w", path, err)
	}
	if st.Version > progressVersion {
		return s, fmt.Errorf("tutor: progress in %s is version %d, newer than %d", path, st.Version, progressVersion)
	}
	if st.Version < 1 {
		st.Version = progressVersion
	}
	if st.Steps == nil {
		st.Steps = map[string]bool{}
	}
	s.state = st
	return s, nil
}

// Save writes the progress into dir in one atomic step. The mutex serializes
// writes and the snapshot is taken under it, so a slow write never replaces a
// newer state with an older one.
func (s *Store) Save(dir string) error {
	if s == nil {
		return errors.New("tutor: no progress to save")
	}
	if dir == "" {
		return errors.New("tutor: no folder for the tutorial")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Version = progressVersion
	if s.state.Steps == nil {
		s.state.Steps = map[string]bool{}
	}
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return core.WriteFileAtomic(filepath.Join(dir, progressFile), data)
}

func (s *Store) locked(f func()) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f()
}

// StepKey is the progress key of a step: its lesson ID and 1-based number
// ("B03#2").
func StepKey(lessonID string, step int) string {
	return lessonID + "#" + strconv.Itoa(step)
}

// Passed reports whether a step already passed.
func (s *Store) Passed(l Lesson, step int) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Steps[StepKey(l.ID, step)]
}

// NextStep is the first step of l that has not passed, 1-based, or 0 when
// every step passed. Steps may pass in any order.
func (s *Store) NextStep(l Lesson) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range l.Steps {
		if !s.state.Steps[StepKey(l.ID, i+1)] {
			return i + 1
		}
	}
	return 0
}

// LessonDone reports whether every step of l passed. A lesson with no steps
// (a reference note) is never counted.
func (s *Store) LessonDone(l Lesson) bool {
	if s == nil || len(l.Steps) == 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range l.Steps {
		if !s.state.Steps[StepKey(l.ID, i+1)] {
			return false
		}
	}
	return true
}

// RouteCount returns the lessons of r that count and how many of them passed.
func (s *Store) RouteCount(r Route) (done, total int) {
	total = len(r.Lessons)
	if s == nil {
		return 0, total
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, l := range r.Lessons {
		if s.lessonDoneLocked(l) {
			done++
		}
	}
	return done, total
}

// RouteDone reports whether every lesson of r passed (a route with no lessons
// is never done).
func (s *Store) RouteDone(r Route) bool {
	if s == nil || len(r.Lessons) == 0 {
		return false
	}
	done, total := s.RouteCount(r)
	return done == total
}

func (s *Store) lessonDoneLocked(l Lesson) bool {
	if len(l.Steps) == 0 {
		return false
	}
	for i := range l.Steps {
		if !s.state.Steps[StepKey(l.ID, i+1)] {
			return false
		}
	}
	return true
}

// Migrated reports whether the old level notes were already retired.
func (s *Store) Migrated() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Migrated
}

// SetMigrated records that the old level notes were retired.
func (s *Store) SetMigrated(v bool) {
	s.locked(func() { s.state.Migrated = v })
}

// LastRoute is the route the player played last, "" when none.
func (s *Store) LastRoute() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.LastRoute
}

// SetLastRoute remembers the route the player played last.
func (s *Store) SetLastRoute(id string) {
	s.locked(func() { s.state.LastRoute = id })
}

// Mark records a step as passed without a check (tests and recovery).
func (s *Store) Mark(l Lesson, step int) bool {
	if s == nil {
		return false
	}
	var changed bool
	s.mu.Lock()
	defer s.mu.Unlock()
	changed = s.markKey(StepKey(l.ID, step))
	return changed
}

// wantKind selects the checks a recheck runs.
type wantKind int

const (
	wantText wantKind = 1 << iota
	wantDisk
	wantBoth = wantText | wantDisk
)

// Recheck re-derives the text and disk steps of l from text and env, and
// returns whether any new step or sub-check passed. Passed keys never clear.
func (s *Store) Recheck(l Lesson, text string, env Env) bool {
	return s.recheck(l, text, env, wantBoth)
}

// RecheckText re-derives only the text steps of l, as an edit does; disk
// checks stay for their own command.
func (s *Store) RecheckText(l Lesson, text string, env Env) bool {
	return s.recheck(l, text, env, wantText)
}

// RecheckDisk re-derives only the disk steps of l (on opening a lesson or
// after saving it), without blocking the update loop.
func (s *Store) RecheckDisk(l Lesson, env Env) bool {
	return s.recheck(l, "", env, wantDisk)
}

// recheck evaluates leaves and marks them; a step's own key is set once every
// leaf is stored, so a mixed step (text plus action) waits for both.
func (s *Store) recheck(l Lesson, text string, env Env, want wantKind) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	env.LessonFile = l.File
	env.full = text
	text = withoutOriginal(l.File, text)
	changed := false
	for i := range l.Steps {
		c := l.Steps[i].Check
		if c.Kind == KindNone {
			continue
		}
		base := StepKey(l.ID, i+1)
		if s.markLeaves(base, "", c, text, env, want) {
			changed = true
		}
		if s.checkStored(base, "", c) && s.markKey(base) {
			changed = true
		}
	}
	return changed
}

// markLeaves evaluates the leaves of c that the mode asks for and marks them.
func (s *Store) markLeaves(base, path string, c Check, text string, env Env, want wantKind) bool {
	switch c.Kind {
	case KindAll:
		changed := false
		for i, sub := range c.Checks {
			if s.markLeaves(base, childPath(path, i), sub, text, env, want) {
				changed = true
			}
		}
		return changed
	case KindText, KindTextRaw, KindTextFence:
		if want&wantText == 0 || !c.Pass(text, env) {
			return false
		}
	case KindDisk:
		if want&wantDisk == 0 || !c.Pass(text, env) {
			return false
		}
	default:
		return false
	}
	return s.markKey(base + path)
}

// checkStored reports whether every leaf of c is stored.
func (s *Store) checkStored(base, path string, c Check) bool {
	switch c.Kind {
	case KindAll:
		if len(c.Checks) == 0 {
			return false
		}
		for i, sub := range c.Checks {
			if !s.checkStored(base, childPath(path, i), sub) {
				return false
			}
		}
		return true
	case KindAction:
		if len(c.Value) == 0 {
			return s.state.Steps[base+path]
		}
		for i := range c.Value {
			if !s.state.Steps[valueKey(base, path, i)] {
				return false
			}
		}
		return true
	}
	return s.state.Steps[base+path]
}

// RecordAction credits an event (and its value) to the action steps of l. It
// returns whether anything newly passed. Every value of an action check has
// to be seen; a value outside the list is ignored.
func (s *Store) RecordAction(l Lesson, event, value string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for i := range l.Steps {
		c := l.Steps[i].Check
		if !c.HasAction() {
			continue
		}
		base := StepKey(l.ID, i+1)
		if s.recordAction(base, "", c, event, value) {
			changed = true
		}
		if s.checkStored(base, "", c) && s.markKey(base) {
			changed = true
		}
	}
	return changed
}

func (s *Store) recordAction(base, path string, c Check, event, value string) bool {
	switch c.Kind {
	case KindAll:
		changed := false
		for i, sub := range c.Checks {
			if s.recordAction(base, childPath(path, i), sub, event, value) {
				changed = true
			}
		}
		return changed
	case KindAction:
		if c.Event != event {
			return false
		}
		if len(c.Value) == 0 {
			return s.markKey(base + path)
		}
		for i, want := range c.Value {
			if want == value {
				return s.markKey(valueKey(base, path, i))
			}
		}
	}
	return false
}

// RecordSave credits a Desafío step (a step with no check): it passes when the
// lesson note is saved with its text changed since it was opened.
func (s *Store) RecordSave(l Lesson, opened, saved string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if opened == saved {
		return false
	}
	changed := false
	for i := range l.Steps {
		if l.Steps[i].Check.Kind == KindNone && s.markKey(StepKey(l.ID, i+1)) {
			changed = true
		}
	}
	return changed
}

// ScanDir re-derives every text and disk step of cat from the notes in dir.
// It reads files, so it runs before the program starts, never from an update
// loop. Action steps come only from the progress file.
func (s *Store) ScanDir(cat Catalog, dir string, env Env) bool {
	if s == nil || dir == "" {
		return false
	}
	changed := false
	for _, r := range cat.Routes {
		for _, l := range r.Lessons {
			data, err := os.ReadFile(filepath.Join(dir, l.File))
			if err != nil {
				continue
			}
			if s.Recheck(l, string(data), env) {
				changed = true
			}
		}
	}
	return changed
}

// markKey records key as passed; it reports whether it was new. The caller
// holds the mutex.
func (s *Store) markKey(key string) bool {
	if s.state.Steps == nil {
		s.state.Steps = map[string]bool{}
	}
	if s.state.Steps[key] {
		return false
	}
	s.state.Steps[key] = true
	return true
}

func childPath(path string, i int) string {
	return path + "." + strconv.Itoa(i+1)
}

func valueKey(base, path string, i int) string {
	return base + path + "=" + strconv.Itoa(i+1)
}

// originals caches the lines of every embedded lesson, so a text check can
// ignore what the lesson itself already says (the player's note starts as a
// copy of it).
var (
	originalOnce   sync.Once
	originalByFile = map[string]map[string]bool{}
)

func originals() map[string]map[string]bool {
	originalOnce.Do(func() {
		entries, err := fs.ReadDir(lessons, "lessons")
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			data, err := fs.ReadFile(lessons, "lessons/"+e.Name())
			if err != nil {
				continue
			}
			set := map[string]bool{}
			for _, l := range strings.Split(string(data), "\n") {
				set[strings.TrimRight(l, "\r")] = true
			}
			originalByFile[e.Name()] = set
		}
	})
	return originalByFile
}

// withoutOriginal blanks every line of text that the embedded lesson already
// has, so a lesson never passes on its own text. A fixture lesson that is not
// embedded keeps its text unchanged.
func withoutOriginal(file, text string) string {
	orig := originals()[lessonFile(file)]
	if len(orig) == 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if orig[strings.TrimRight(l, "\r")] {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}
