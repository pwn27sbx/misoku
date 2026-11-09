package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// The event convention for the lessons that need something no note shows
// (F3b/F3c): call m.tutorEvent(name, value) at the point the action is
// confirmed done, never inside View, and only when m.tutor is not nil. The
// current lesson is the last lesson note focused (E2), so an event credits
// it even when another tab is active.

// tutorFlashDuration is how long "✓ Paso N" stays in the bar.
const tutorFlashDuration = 1500 * time.Millisecond

// tutorGame is the tutorial being played (misoku tutor).
type tutorGame struct {
	store  *tutor.Store
	cat    tutor.Catalog
	dir    string
	hasGit bool

	current      tutor.Lesson
	hasCurrent   bool
	openedText   string    // the lesson note as it was when it became current
	opened       time.Time // when it became current, for file-changed
	configAtOpen map[string]string

	flash    string
	flashSeq int
	hinted   string // lesson whose hint already showed
}

// WithTutor plays the tutorial in this vault: the current lesson's steps are
// checked as the player edits and acts, and the statusline follows them. A
// nil store leaves the tutorial off.
func WithTutor(store *tutor.Store, cat tutor.Catalog, dir string, hasGit bool) Option {
	return func(m *Model) {
		if store == nil {
			return
		}
		m.tutor = &tutorGame{store: store, cat: cat, dir: dir, hasGit: hasGit}
		if m.tutorNotice != "" {
			m.status = m.tutorNotice
		}
	}
}

// WithTutorNotice shows the one-line notice of a migration when the tutorial
// opens (the old level notes moved to the Trash).
func WithTutorNotice(notice string) Option {
	return func(m *Model) {
		m.tutorNotice = notice
		if notice != "" && m.tutor != nil {
			m.status = notice
		}
	}
}

// tutorSaveMsg is a finished progress write.
type tutorSaveMsg struct{ err error }

// tutorDiskMsg is the result of the disk checks of one lesson.
type tutorDiskMsg struct {
	lesson  string
	before  []bool
	changed bool
}

// tutorOpenMsg carries the lesson's opened-at time and config snapshot.
type tutorOpenMsg struct {
	lesson string
	config map[string]string
}

// tutorFlashMsg clears "✓ Paso N" after its time.
type tutorFlashMsg struct{ seq int }

// tutorSyncCurrent follows the active note: when it is a lesson, that lesson
// becomes current and stays current until another lesson note is focused. It
// is called after every update, so opening, clicking or switching tabs all
// count without touching disk.
func (m *Model) tutorSyncCurrent() tea.Cmd {
	g := m.tutor
	if g == nil {
		return nil
	}
	b := m.activeBuffer()
	if b == nil || b.draft() {
		return nil
	}
	l, ok := g.cat.Lesson(b.id)
	if !ok {
		return nil
	}
	if g.hasCurrent && g.current.ID == l.ID {
		return nil
	}
	g.current, g.hasCurrent = l, true
	g.openedText = b.saved
	g.opened = m.clock()
	g.flash, g.flashSeq = "", 0
	g.hinted = ""
	if r, ok := g.cat.RouteOfLesson(l.ID); ok {
		g.store.SetLastRoute(r.ID)
	}
	// The progress file keeps last_route and any skip decision.
	return tea.Batch(m.tutorSaveCmd(), m.tutorSkipGit(l), m.tutorOpenCmd(l))
}

// tutorSkipGit skips the steps that wait for git when git is missing, and
// clears that decision when git is available again.
func (m *Model) tutorSkipGit(l tutor.Lesson) tea.Cmd {
	g := m.tutor
	if !l.NeedsGit() {
		return nil
	}
	if g.hasGit {
		if g.store.ClearSkipped(l) {
			return m.tutorSaveCmd()
		}
		return nil
	}
	changed := false
	for i := range l.Steps {
		if l.Steps[i].NeedsGit() && g.store.MarkSkipped(l, i+1) {
			changed = true
		}
	}
	if changed {
		return m.tutorSaveCmd()
	}
	return nil
}

// tutorOpenCmd takes the baseline config values and the opening time off the
// update loop, then runs the lesson's disk checks against them.
func (m *Model) tutorOpenCmd(l tutor.Lesson) tea.Cmd {
	path := m.configPath
	keys := l.ConfigKeys()
	return func() tea.Msg {
		vals := readConfigValues(path, keys)
		return tutorOpenMsg{lesson: l.ID, config: vals}
	}
}

// tutorDiskCmd runs the lesson's disk checks in a command, with a snapshot of
// the notes and the lesson's opened-at time.
func (m *Model) tutorDiskCmd(l tutor.Lesson) tea.Cmd {
	before := m.tutorPassed(l)
	env := m.tutorDiskEnv(l)
	store := m.tutor.store
	return func() tea.Msg {
		changed := store.RecheckDisk(l, env)
		return tutorDiskMsg{lesson: l.ID, before: before, changed: changed}
	}
}

// tutorDiskEnv builds the reader the disk checks use. Everything it needs is
// captured here, in the update loop; the callbacks read the disk in the
// command.
func (m *Model) tutorDiskEnv(l tutor.Lesson) tutor.Env {
	notes := append([]core.Page(nil), m.notes...)
	root := m.store.Root()
	opened := m.tutor.opened
	configAtOpen := make(map[string]string, len(m.tutor.configAtOpen))
	for k, v := range m.tutor.configAtOpen {
		configAtOpen[k] = v
	}
	cfgPath := m.configPath
	today := m.clock().Format("2006-01-02")
	return tutor.Env{
		Exists: func(name string) bool {
			_, ok := core.ResolveNote(notes, name, l.File)
			return ok
		},
		FolderExists: func(name string) bool {
			st, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
			return err == nil && st.IsDir()
		},
		Changed: func(name string) bool {
			st, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
			return err == nil && st.ModTime().After(opened)
		},
		NoteText: func(name string) (string, bool) {
			p := filepath.Join(root, filepath.FromSlash(name))
			if !strings.HasSuffix(p, ".md") {
				p += ".md"
			}
			data, err := os.ReadFile(p)
			return string(data), err == nil
		},
		Config: func(key string) (string, bool) { return readConfigValue(cfgPath, key) },
		ConfigChanged: func(key string) bool {
			now, ok := readConfigValue(cfgPath, key)
			old, had := configAtOpen[key]
			return ok != had || now != old
		},
		Today: today,
	}
}

// tutorEnv is the reader for text checks: the explorer's notes, no disk.
func (m *Model) tutorEnv(l tutor.Lesson) tutor.Env {
	notes := m.notes
	return tutor.Env{
		Exists: func(name string) bool {
			_, ok := core.ResolveNote(notes, name, l.File)
			return ok
		},
		Today: m.clock().Format("2006-01-02"),
	}
}

// tutorPassed snapshots which steps pass now, to know which one just did.
func (m *Model) tutorPassed(l tutor.Lesson) []bool {
	done := make([]bool, len(l.Steps))
	for i := range done {
		done[i] = m.tutor.store.Passed(l, i+1)
	}
	return done
}

// tutorEdited re-checks the current lesson's text after an edit. Only text
// checks run here; the disk checks have their own command (E7).
func (m *Model) tutorEdited() tea.Cmd {
	g := m.tutor
	if g == nil || !g.hasCurrent {
		return nil
	}
	id := m.currentNoteID()
	if id == "" || id != g.current.File {
		return nil
	}
	l := g.current
	before := m.tutorPassed(l)
	if !g.store.RecheckText(l, m.content.Value(), m.tutorEnv(l)) {
		return m.tutorHint(l)
	}
	return m.tutorChanged(l, before)
}

// tutorSaved credits a Desafío step (no check) when the lesson note is saved
// with changes since it was opened, and runs the disk checks after the save.
func (m *Model) tutorSaved(b *buffer, text string) tea.Cmd {
	g := m.tutor
	if g == nil || !g.hasCurrent {
		return nil
	}
	l := g.current
	// A save changes the disk, which a lesson may check (create a note and
	// save it): re-run the current lesson's disk checks even when the saved
	// tab is another note.
	cmds := []tea.Cmd{m.tutorDiskCmd(l)}
	if b.id == l.File {
		before := m.tutorPassed(l)
		if g.store.RecordSave(l, g.openedText, text) {
			cmds = append(cmds, m.tutorChanged(l, before))
		}
	}
	return tea.Batch(cmds...)
}

// tutorEvent credits an app action to the current lesson (E9). F3b/F3c call
// it where the action is confirmed done; the current lesson need not be the
// active tab.
func (m *Model) tutorEvent(name, value string) tea.Cmd {
	g := m.tutor
	if g == nil || !g.hasCurrent {
		return nil
	}
	l := g.current
	before := m.tutorPassed(l)
	if !g.store.RecordAction(l, name, value) {
		return nil
	}
	return m.tutorChanged(l, before)
}

// tutorChanged saves the progress after a pass, flashes the step that just
// passed, and saves the lesson note when the lesson is done.
func (m *Model) tutorChanged(l tutor.Lesson, before []bool) tea.Cmd {
	g := m.tutor
	cmds := []tea.Cmd{m.tutorSaveCmd()}
	for i := range l.Steps {
		if i < len(before) && before[i] {
			continue
		}
		if g.store.Passed(l, i+1) {
			g.flash = fmt.Sprintf("✓ Paso %d", i+1)
			g.flashSeq++
			cmds = append(cmds, tutorFlashCmd(g.flashSeq))
			break
		}
	}
	if g.store.LessonDone(l) {
		for _, b := range m.buffers {
			if b.id == l.File && m.isDirty(b) {
				cmds = append(cmds, m.saveBuffer(b, nil))
				break
			}
		}
	}
	return tea.Batch(cmds...)
}

// tutorFlashCmd clears the flash after its time.
func tutorFlashCmd(seq int) tea.Cmd {
	return tea.Tick(tutorFlashDuration, func(time.Time) tea.Msg { return tutorFlashMsg{seq: seq} })
}

// tutorSaveCmd writes the progress file off the update loop.
func (m *Model) tutorSaveCmd() tea.Cmd {
	store, dir := m.tutor.store, m.tutor.dir
	return func() tea.Msg { return tutorSaveMsg{err: store.Save(dir)} }
}

// tutorHint shows the current step's hint once per current lesson, the old
// "- []" behaviour.
func (m *Model) tutorHint(l tutor.Lesson) tea.Cmd {
	g := m.tutor
	step := g.store.NextStep(l)
	if step == 0 {
		return nil
	}
	hint := l.Steps[step-1].Hint
	if hint == "" || g.hinted == l.ID {
		return nil
	}
	g.hinted = l.ID
	return m.showToast(hint)
}

// tutorLine is the persistent tutorial line: the step in play, the lesson
// just passed, or the finished route. "" when no lesson is current.
func (m *Model) tutorLine() string {
	g := m.tutor
	if g == nil || !g.hasCurrent || m.viewMode != viewModeWorkspace {
		return ""
	}
	l := g.current
	skipped := 0
	for i := range l.Steps {
		if g.store.Skipped(l, i+1) {
			skipped++
		}
	}
	if skipped == len(l.Steps) {
		return l.ID + " · Necesitas git para esta lección"
	}
	if g.store.LessonDone(l) {
		r, ok := g.cat.RouteOfLesson(l.ID)
		if ok && g.store.RouteDone(r) {
			return "🏆 Ruta " + r.Name + " completa"
		}
		next := "Después del tutorial"
		if ok {
			if n, yes := g.cat.NextLesson(r.ID, l.ID); yes {
				next = n.Title
			} else {
				for _, nl := range r.Lessons {
					if !g.store.LessonDone(nl) {
						next = nl.Title
						break
					}
				}
			}
		}
		return fmt.Sprintf("✓ %s superado · siguiente: %s", l.ID, next)
	}
	step := g.store.NextStep(l)
	if step == 0 {
		return ""
	}
	return fmt.Sprintf("%s · Paso %d/%d · %s", l.ID, step, len(l.Steps), l.Steps[step-1].Text)
}

// tutorBadge is the statusline's "Tutor k/n" for the current route, or for
// all routes when no lesson is current, "" otherwise.
func (m *Model) tutorBadge() string {
	g := m.tutor
	if g == nil {
		return ""
	}
	if g.hasCurrent {
		if r, ok := g.cat.RouteOfLesson(g.current.ID); ok {
			done, total := g.store.RouteCount(r)
			return fmt.Sprintf("Tutor %d/%d", done, total)
		}
	}
	done, total := 0, 0
	for _, r := range g.cat.Routes {
		if len(r.Lessons) == 0 {
			continue
		}
		d, t := g.store.RouteCount(r)
		done += d
		total += t
	}
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("Tutor %d/%d", done, total)
}

// modeValue is the check's name for an editor mode.
func modeValue(mode content.VimMode) string {
	switch mode {
	case content.ModeInsert:
		return "insert"
	case content.ModeVisual:
		return "visual"
	case content.ModeVisualLine:
		return "visual-line"
	default:
		return "normal"
	}
}
