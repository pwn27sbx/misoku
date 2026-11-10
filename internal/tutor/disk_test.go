package tutor_test

import (
	"strconv"
	"testing"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

// diskPlace is where one step's teaching action really puts its files. The
// vault snapshot is the smallest one where every disk path check of the step
// passes, through tutor.DirEnv's rules: the lesson's route folder first, then
// the vault root. Note names leave out ".md", as the checker's contract says.
type diskPlace struct {
	notes   map[string]string // vault-relative note name (no ".md") -> text
	folders []string          // vault-relative folder paths
	today   string
}

// S5/F3: every disk path check of every route passes when its action is done
// the way the lesson says. The comment of each place is the action it stands
// for and where that action writes today.
func TestEveryDiskCheckPassesWithTheLessonsActions(t *testing.T) {
	places := map[string]diskPlace{
		// space n while the lesson is open: the route folder.
		"A07#3": {notes: map[string]string{"3 Avanzado/Idea": "[[A07 Menciones y backlinks]]"}},
		// r on the lesson note rewrites its own link in place.
		"A08#1": {notes: map[string]string{"3 Avanzado/A08 Renombrar sin romper enlaces": "[[Equipaje]]"}},
		// The folder editor (m) and mini.files (M) create in the selected
		// folder, the lesson's route folder.
		"A20#2": {notes: map[string]string{"3 Avanzado/Notas/equipaje": "x"}},
		"A20#4": {notes: map[string]string{"3 Avanzado/practica-mini": "x"}},
		// :w lista-viaje saves at the vault root (no folder in the argument).
		"B06#1": {notes: map[string]string{"lista-viaje": "x"}},
		// n in the explorer creates inside the selected folder, the lesson's.
		"B09#2": {notes: map[string]string{"1 Básico/compras": "x"}},
		"B09#3": {folders: []string{"1 Básico/ideas"}},
		// B10 renames compras to mercado, then deletes mercado; ideas stays.
		"B10#1": {notes: map[string]string{"1 Básico/mercado": "x"}},
		"B10#2": {folders: []string{"1 Básico/ideas"}},
		// misoku tutor makes the vault a git repository at its root.
		"E17#1": {folders: []string{".git"}},
		// /Page and s create a sub-note under the note, in the route folder.
		"I06#1": {notes: map[string]string{"2 Intermedio/Viaje/Equipaje de playa": "x"}},
		"I06#2": {notes: map[string]string{"2 Intermedio/Viaje/Día 1": "x"}},
		// A pasted image lands in assets/ next to the note.
		"I18#4": {folders: []string{"2 Intermedio/assets"}},
		// The player's own notes, created in the route folder.
		"I20#4": {notes: map[string]string{"2 Intermedio/Plan": "[[Viaje#Equipaje]]"}},
		"I21#3": {notes: map[string]string{"2 Intermedio/Viaje": "reserva ^reserva"}},
		"I23#2": {notes: map[string]string{"2 Intermedio/Viaje": "x"}},
		// Templates and the daily note live at the vault root, as configured.
		"I25#5": {notes: map[string]string{"Plantillas/diario": "x"}},
		"I27#1": {notes: map[string]string{"Diario/2026-11-09": "x"}, today: "2026-11-09"},
		"I27#2": {notes: map[string]string{"Plantillas/diario": "x"}, folders: []string{"Plantillas"}},
	}

	// A13 exports PDFs, which the disk checker cannot see: exists/no-file only
	// understand note names (they add ".md"), so these three steps fail on the
	// base too. F5 keeps their paths and reports the gap; it does not change
	// what exists means.
	knownGaps := map[string]bool{"A13#1": true, "A13#2": true, "A13#3": true}

	cat := tutor.Default()
	pathPreds := map[string]bool{
		"exists": true, "no-file": true, "folder-exists": true, "other-note-text": true,
	}
	leaves, configLeaves, changedLeaves, pathLeaves := 0, 0, 0, 0
	used := map[string]bool{}
	for _, r := range cat.Routes {
		for _, l := range r.Lessons {
			for i, st := range l.Steps {
				key := l.ID + "#" + strconv.Itoa(i+1)
				var walk func(c tutor.Check)
				walk = func(c tutor.Check) {
					if c.Kind == tutor.KindAll {
						for _, sub := range c.Checks {
							walk(sub)
						}
						return
					}
					if c.Kind != tutor.KindDisk {
						return
					}
					leaves++
					switch {
					case c.Pred == "config-key" || c.Pred == "config-changed":
						configLeaves++
					case c.Pred == "file-changed":
						// The app's reader, with the lesson's opened-at time:
						// the app tests cover it (TestTutorDiskCheckOnOpen).
						changedLeaves++
					case pathPreds[c.Pred]:
						pathLeaves++
						if knownGaps[key] {
							env, _ := vaultEnv(r.Dir, nil, nil)
							if c.Pass("", env) {
								t.Errorf("%s now passes; the PDF gap is fixed, drop it from knownGaps", key)
							}
							return
						}
						p, ok := places[key]
						if !ok {
							t.Fatalf("%s is a disk path check with no place in the table", key)
						}
						folders := map[string]bool{}
						for _, f := range p.folders {
							folders[f] = true
						}
						env, tried := vaultEnv(r.Dir, p.notes, folders)
						env.Today = p.today
						if !c.Pass("", env) {
							t.Errorf("%s %s %v does not pass where the action puts it (tried %v)", key, c.Pred, c.Args, *tried)
						}
						used[key] = true
					default:
						t.Errorf("%s uses an unclassified disk predicate %q", key, c.Pred)
					}
				}
				walk(st.Check)
			}
		}
	}
	if leaves != 48 || pathLeaves != 25 || configLeaves != 20 || changedLeaves != 3 {
		t.Errorf("disk leaves = %d (paths %d, config %d, file-changed %d), want 48 (25, 20, 3)",
			leaves, pathLeaves, configLeaves, changedLeaves)
	}
	for key := range places {
		if !used[key] {
			t.Errorf("the table places %s, which no longer has a disk path check", key)
		}
	}
}
