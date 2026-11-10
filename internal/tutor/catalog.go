package tutor

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// The catalog is data ("Amendment E12"): the lessons are written in parallel
// as embedded JSON, one file per route, with the schema of F4's brief. The
// player cannot edit it, and a broken catalog fails a test instead of the
// player's tutorial.
//
//go:embed routes/*.json
var routesFS embed.FS

// ReferenceNote is the note after the last lesson: terminal commands and what
// cannot be practised inside the tutorial. It has no steps and belongs to no
// route, so the catalog does not list it; the installer still copies it.
const ReferenceNote = "Después del tutorial.md"

// Catalog is the tutorial's content: the routes in play order.
type Catalog struct {
	Routes []Route
}

// Route is one route of the tutorial (Básico, Intermedio, …).
type Route struct {
	// ID is the route's letter ("B").
	ID string `json:"id"`
	// Name is the route's name as shown to the player ("Básico").
	Name string `json:"name"`
	// Dir is the route's folder in the tutorial vault ("1 Básico"). Its
	// number keeps the routes in learning order in the explorer.
	Dir string `json:"dir"`
	// Index is the note that introduces the route ("B00 Básico.md"). It has
	// no steps and is not part of Lessons.
	Index string `json:"index"`
	// Lessons are the route's lessons in play order.
	Lessons []Lesson `json:"lessons"`
}

// Lesson is one note of the tutorial and the steps it asks for.
type Lesson struct {
	// ID is the lesson's ID ("B01").
	ID string `json:"id"`
	// Title is the lesson's name as shown to the player.
	Title string `json:"title"`
	// File is the note's file name inside the route's folder.
	File string `json:"file"`
	// Steps are the lesson's steps in play order.
	Steps []Step `json:"steps"`

	// dir is the route's folder (Route.Dir), set when the catalog loads. It
	// is empty for a fixture lesson that lives at the vault root.
	dir string
}

// Path is the note's vault-relative path, the route's folder and the file
// ("1 Básico/B01 Escribe en 30 segundos.md"). A fixture lesson with no folder
// is its file.
func (l Lesson) Path() string {
	if l.dir == "" {
		return l.File
	}
	return l.dir + "/" + l.File
}

// IndexPath is the route index's vault-relative path, with the route's folder.
func (r Route) IndexPath() string {
	if r.Dir == "" {
		return r.Index
	}
	return r.Dir + "/" + r.Index
}

// Step is one thing the player does, with the check that proves it.
type Step struct {
	// Text is the short Spanish instruction for the status line.
	Text string `json:"text"`
	// Hint is an optional short tip shown while the step is not met.
	Hint string `json:"hint,omitempty"`
	// Check is what proves the step. Kind "none" (a Desafío step) passes
	// when the note is saved with changes.
	Check Check `json:"check"`
	// IDs are the curriculum's inventory rows this step practises; they are
	// informational and never checked.
	IDs []string `json:"ids,omitempty"`
}

// Check is one proof of a step. Its JSON mirrors F4's brief exactly; the
// allowed fields depend on Kind (see validate).
type Check struct {
	Kind    string   `json:"kind"`
	Pred    string   `json:"pred,omitempty"`
	Args    []string `json:"args,omitempty"`
	Fence   string   `json:"fence,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
	Event   string   `json:"event,omitempty"`
	Value   Values   `json:"value,omitempty"`
	Checks  []Check  `json:"checks,omitempty"`
}

// Values is an action's required value: JSON may write one string or an
// array, and every listed value must be seen.
type Values []string

// UnmarshalJSON accepts a string or an array of strings.
func (v *Values) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*v = nil
		return nil
	}
	if trimmed[0] == '[' {
		var list []string
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return err
		}
		*v = list
		return nil
	}
	var one string
	if err := json.Unmarshal(trimmed, &one); err != nil {
		return err
	}
	*v = Values{one}
	return nil
}

// Check kinds.
const (
	KindText      = "text"
	KindTextFence = "text-fence"
	KindTextRaw   = "text-raw"
	KindDisk      = "disk"
	KindAction    = "action"
	KindAll       = "all"
	KindNone      = "none"
)

// routeOrder is how the routes are shown; a route with no embedded file is
// simply absent from the catalog.
var routeOrder = []string{"B", "I", "A", "E"}

// routeNames are the only route letters a catalog may use.
var routeNames = map[string]bool{"B": true, "I": true, "A": true, "E": true}

// textPreds are the predicates a text or text-raw check may name.
var textPreds = map[string]bool{
	"line-contains":      true,
	"line-starts":        true,
	"line-equals":        true,
	"line-ends":          true,
	"n-lines-start":      true,
	"absent":             true,
	"heading-exists":     true,
	"wikilink-resolves":  true,
	"blockref-exists":    true,
	"headinglink-exists": true,
	"table-shape":        true,
	"heading-under":      true,
	"task-status":        true,
}

// diskPreds are the predicates a disk check may name.
var diskPreds = map[string]bool{
	"exists":          true,
	"folder-exists":   true,
	"file-changed":    true,
	"other-note-text": true,
	"config-key":      true,
	"config-changed":  true,
	"no-file":         true,
}

// fenceKinds are the fenced blocks a text-fence check may name.
var fenceKinds = map[string]bool{
	"board": true, "calendar": true, "timeline": true, "chart": true,
	"form": true, "flow": true, "math": true, "code": true,
	"columns": true,
}

// actionEvents are the event names the curriculum's "Necesidades del motor"
// defines; F3b/F3c report them and the lessons may wait for them.
var actionEvents = map[string]bool{
	"new_draft": true, "tab_closed": true, "followed_link": true,
	"mode_changed": true, "motion": true, "paste": true, "yank": true,
	"undo_empty":     true,
	"focus_explorer": true, "focus_cycle": true, "split_view": true,
	"pane_resized": true, "next_tab": true, "prev_tab": true,
	"reading_view": true, "editor_view": true,
	"save_dialog": true, "quit": true, "cmdline": true, "command_error": true,
	"note_opened": true, "note_previewed": true, "tree_state": true,
	"tree_refresh": true,
	"slash_open":   true, "slash_close": true, "table_hint": true,
	"cell_next":  true,
	"search_hit": true, "search_next": true, "search_prev": true,
	"search_wrap": true, "search_error": true,
	"palette_open": true, "palette_run": true, "leader_guide": true,
	"messages_open": true, "tags_list": true, "finder_open": true,
	"finder_filter":  true,
	"links_headings": true, "links_blocks": true, "link_footer": true,
	"link_next": true,
	"home":      true, "home_open": true, "home_key": true,
	"status_glyph": true, "settings_open": true, "settings_saved": true,
	"key_clash": true, "theme_pick": true, "theme_list": true,
	"cursor_shape":     true,
	"preview_rendered": true, "preview_scroll": true, "preview_fold": true,
	"calendar_month": true, "backlinks": true, "mention_open": true,
	"backlink_open": true, "block_handles": true,
	"graph_open": true, "graph_move": true, "graph_mouse": true,
	"graph_refresh": true, "graph_filter": true,
	"history_open": true, "history_preview": true, "history_error": true,
	"link_confirm": true, "external_change": true, "dirbuf_open": true,
	"sync_run": true, "sync_status": true, "sync_stop_dialog": true,
	"sync_auth_offer": true, "sync_indicator": true, "sync_arrival": true,
	"error_shown": true,
}

// dirRe is a route folder's name: the same characters as a lesson file name,
// without ".md" and without "/", so it is one entry at the vault root.
var dirRe = regexp.MustCompile(`^[\p{L}\p{N} ,.\-]+$`)

// fileRe is L1's slug: letters (accents and ñ included), digits, spaces and
// ", - ." only (go:embed refuses “`” and “:”, and “/” makes a folder).
var fileRe = regexp.MustCompile(`^[\p{L}\p{N} ,.\-]+\.md$`)

// lessonIDRe is a route letter and a two-digit number ("B01"), the shape the
// progress keys assume.
var lessonIDRe = regexp.MustCompile(`^[BIEA][0-9]{2}$`)

var (
	bundledOnce sync.Once
	bundled     Catalog
	bundledErr  error
)

// BundledRoutes loads and validates every embedded route. A bad catalog is a
// build bug, so tests call this and fail; the player gets an empty catalog
// from Default instead of a panic.
func BundledRoutes() (Catalog, error) {
	bundledOnce.Do(func() { bundled, bundledErr = loadBundled() })
	return bundled, bundledErr
}

func loadBundled() (Catalog, error) {
	entries, err := fs.ReadDir(routesFS, "routes")
	if err != nil {
		return Catalog{}, err
	}
	var cat Catalog
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := fs.ReadFile(routesFS, "routes/"+e.Name())
		if err != nil {
			return Catalog{}, err
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		var r Route
		if err := dec.Decode(&r); err != nil {
			return Catalog{}, fmt.Errorf("tutor: routes/%s: %w", e.Name(), err)
		}
		cat.Routes = append(cat.Routes, r)
	}
	sort.SliceStable(cat.Routes, func(i, j int) bool {
		return routeRank(cat.Routes[i].ID) < routeRank(cat.Routes[j].ID)
	})
	for i := range cat.Routes {
		r := &cat.Routes[i]
		for j := range r.Lessons {
			r.Lessons[j].dir = r.Dir
		}
	}
	if err := cat.validate(); err != nil {
		return Catalog{}, err
	}
	return cat, nil
}

func routeRank(id string) int {
	for i, r := range routeOrder {
		if r == id {
			return i
		}
	}
	return len(routeOrder)
}

// Default returns the bundled catalog. Embedded data is validated by tests;
// at runtime an invalid catalog is empty rather than fatal.
func Default() Catalog {
	cat, err := BundledRoutes()
	if err != nil {
		return Catalog{}
	}
	return cat
}

// validate checks the whole catalog: unique routes, lessons and files, the
// route/lesson ID relation, the L1 file names, the step text length and every
// check against the schema. It is the test's gate: an unknown kind, pred or
// event never reaches the player unnoticed.
func (c Catalog) validate() error {
	routeIDs := map[string]bool{}
	lessonIDs := map[string]bool{}
	files := map[string]bool{}
	indexes := map[string]bool{}
	dirs := map[string]bool{}
	for _, r := range c.Routes {
		if !routeNames[r.ID] {
			return fmt.Errorf("tutor: unknown route id %q", r.ID)
		}
		if routeIDs[r.ID] {
			return fmt.Errorf("tutor: duplicate route %q", r.ID)
		}
		routeIDs[r.ID] = true
		if strings.TrimSpace(r.Name) == "" {
			return fmt.Errorf("tutor: route %s has no name", r.ID)
		}
		if err := checkDirName(r.Dir); err != nil {
			return fmt.Errorf("tutor: route %s folder: %w", r.ID, err)
		}
		if dirs[r.Dir] {
			return fmt.Errorf("tutor: duplicate folder %q", r.Dir)
		}
		dirs[r.Dir] = true
		if want := strconv.Itoa(routeRank(r.ID)+1) + " "; !strings.HasPrefix(r.Dir, want) {
			return fmt.Errorf("tutor: route %s folder %q must start with %q", r.ID, r.Dir, want)
		}
		if err := checkFileName(r.Index); err != nil {
			return fmt.Errorf("tutor: route %s index: %w", r.ID, err)
		}
		if !strings.HasPrefix(r.Index, r.ID+"00 ") {
			return fmt.Errorf("tutor: route %s index %q must be %s00 <name>.md", r.ID, r.Index, r.ID)
		}
		if files[r.Index] || indexes[r.Index] {
			return fmt.Errorf("tutor: duplicate file %q", r.Index)
		}
		indexes[r.Index] = true
		for _, l := range r.Lessons {
			if !lessonIDRe.MatchString(l.ID) || !strings.HasPrefix(l.ID, r.ID) {
				return fmt.Errorf("tutor: lesson %q must be %s and two digits", l.ID, r.ID)
			}
			if lessonIDs[l.ID] {
				return fmt.Errorf("tutor: duplicate lesson id %q", l.ID)
			}
			lessonIDs[l.ID] = true
			if strings.TrimSpace(l.Title) == "" {
				return fmt.Errorf("tutor: lesson %s has no title", l.ID)
			}
			if err := checkFileName(l.File); err != nil {
				return fmt.Errorf("tutor: lesson %s: %w", l.ID, err)
			}
			if !strings.HasPrefix(l.File, l.ID+" ") {
				return fmt.Errorf("tutor: lesson %s file %q must start with %q", l.ID, l.File, l.ID+" ")
			}
			if files[l.File] || indexes[l.File] {
				return fmt.Errorf("tutor: duplicate file %q", l.File)
			}
			files[l.File] = true
			if len(l.Steps) == 0 {
				return fmt.Errorf("tutor: lesson %s has no steps", l.ID)
			}
			for i, st := range l.Steps {
				if err := st.validate(); err != nil {
					return fmt.Errorf("tutor: %s step %d: %w", l.ID, i+1, err)
				}
			}
		}
	}
	return nil
}

func checkFileName(name string) error {
	if !fileRe.MatchString(name) {
		return fmt.Errorf("file name %q must use letters, digits, spaces and , - . and end in .md", name)
	}
	return nil
}

// checkDirName checks a route's folder name: the file alphabet, one folder
// deep, never an empty name.
func checkDirName(name string) error {
	if !dirRe.MatchString(name) {
		return fmt.Errorf("folder name %q must use letters, digits, spaces and , - . and no /", name)
	}
	return nil
}

// validate checks one step's text and check.
func (st Step) validate() error {
	text := strings.TrimSpace(st.Text)
	if text == "" {
		return fmt.Errorf("empty text")
	}
	if n := utf8.RuneCountInString(st.Text); n > 60 {
		return fmt.Errorf("text is %d characters, want 60 or fewer", n)
	}
	if st.Hint != "" && strings.TrimSpace(st.Hint) == "" {
		return fmt.Errorf("hint is blank")
	}
	return st.Check.validate()
}

// validate checks one check against the schema, including the fields the
// chosen kind must not carry.
func (c Check) validate() error {
	switch c.Kind {
	case KindNone:
		if c.Pred != "" || len(c.Args) > 0 || c.Fence != "" || c.Pattern != "" ||
			c.Event != "" || len(c.Value) > 0 || len(c.Checks) > 0 {
			return fmt.Errorf("kind none carries no other field")
		}
		return nil
	case KindText, KindTextRaw:
		if c.Fence != "" || c.Pattern != "" || c.Event != "" ||
			len(c.Value) > 0 || len(c.Checks) > 0 {
			return fmt.Errorf("kind %s carries only pred and args", c.Kind)
		}
		return validateTextPred(c.Pred, c.Args)
	case KindTextFence:
		if c.Pred != "" || len(c.Args) > 0 || c.Event != "" ||
			len(c.Value) > 0 || len(c.Checks) > 0 {
			return fmt.Errorf("kind text-fence carries only fence and pattern")
		}
		if !fenceKinds[c.Fence] {
			return fmt.Errorf("unknown fence %q", c.Fence)
		}
		if strings.TrimSpace(c.Pattern) == "" {
			return fmt.Errorf("text-fence needs a pattern")
		}
		return nil
	case KindDisk:
		if c.Fence != "" || c.Pattern != "" || c.Event != "" ||
			len(c.Value) > 0 || len(c.Checks) > 0 {
			return fmt.Errorf("kind disk carries only pred and args")
		}
		return validateDiskPred(c.Pred, c.Args)
	case KindAction:
		if c.Pred != "" || len(c.Args) > 0 || c.Fence != "" ||
			c.Pattern != "" || len(c.Checks) > 0 {
			return fmt.Errorf("kind action carries only event and value")
		}
		if !actionEvents[c.Event] {
			return fmt.Errorf("unknown event %q", c.Event)
		}
		seen := map[string]bool{}
		for _, v := range c.Value {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("action value is blank")
			}
			if seen[v] {
				return fmt.Errorf("action value %q repeats", v)
			}
			seen[v] = true
		}
		return nil
	case KindAll:
		if c.Pred != "" || len(c.Args) > 0 || c.Fence != "" ||
			c.Pattern != "" || c.Event != "" || len(c.Value) > 0 {
			return fmt.Errorf("kind all carries only checks")
		}
		if len(c.Checks) == 0 {
			return fmt.Errorf("kind all needs at least one check")
		}
		for i, sub := range c.Checks {
			if sub.Kind == KindNone {
				return fmt.Errorf("sub-check %d is kind none", i+1)
			}
			if err := sub.validate(); err != nil {
				return fmt.Errorf("sub-check %d: %w", i+1, err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown kind %q", c.Kind)
	}
}

func validateTextPred(pred string, args []string) error {
	if !textPreds[pred] {
		return fmt.Errorf("unknown text pred %q", pred)
	}
	switch pred {
	case "line-contains", "line-starts", "line-equals", "line-ends", "absent":
		return argc(pred, args, 1)
	case "n-lines-start":
		if err := argc(pred, args, 2); err != nil {
			return err
		}
		if n, err := strconv.Atoi(args[0]); err != nil || n < 1 {
			return fmt.Errorf("%s needs a positive count first", pred)
		}
		if strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("%s needs a prefix", pred)
		}
	case "heading-exists":
		if err := argc(pred, args, 2); err != nil {
			return err
		}
		level, err := strconv.Atoi(args[0])
		if err != nil || level < 1 || level > 6 {
			return fmt.Errorf("%s needs a heading level 1-6 first", pred)
		}
		if strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("%s needs a heading", pred)
		}
	case "wikilink-resolves", "blockref-exists", "headinglink-exists":
		if err := argc(pred, args, 1); err != nil {
			return err
		}
		if strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("%s needs a target", pred)
		}
	case "table-shape":
		if err := argc(pred, args, 1); err != nil {
			return err
		}
		if cells, err := strconv.Atoi(args[0]); err != nil || cells < 1 {
			return fmt.Errorf("%s needs a positive cell count", pred)
		}
	case "heading-under":
		if err := argc(pred, args, 4); err != nil {
			return err
		}
		parent, err := strconv.Atoi(args[0])
		if err != nil || parent < 1 || parent > 6 {
			return fmt.Errorf("%s needs a parent level 1-6 first", pred)
		}
		child, err := strconv.Atoi(args[2])
		if err != nil || child < 1 || child > 6 {
			return fmt.Errorf("%s needs a child level 1-6 third", pred)
		}
		if child <= parent {
			return fmt.Errorf("%s needs a child level below the parent level", pred)
		}
		if strings.TrimSpace(args[1]) == "" || strings.TrimSpace(args[3]) == "" {
			return fmt.Errorf("%s needs a parent and a child heading", pred)
		}
	case "task-status":
		if err := argc(pred, args, 2); err != nil {
			return err
		}
		switch args[0] {
		case "done", "open", "doing":
		default:
			return fmt.Errorf("%s state %q must be done, open or doing", pred, args[0])
		}
		if strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("%s needs the task text", pred)
		}
	}
	return nil
}

func validateDiskPred(pred string, args []string) error {
	if !diskPreds[pred] {
		return fmt.Errorf("unknown disk pred %q", pred)
	}
	switch pred {
	case "file-changed":
		return argc(pred, args, 0)
	case "exists", "folder-exists", "no-file", "config-changed":
		if err := argc(pred, args, 1); err != nil {
			return err
		}
		if strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("%s needs a path or key", pred)
		}
	case "other-note-text":
		if err := argc(pred, args, 2); err != nil {
			return err
		}
		if strings.TrimSpace(args[0]) == "" || strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("%s needs a note and a text", pred)
		}
	case "config-key":
		if len(args) != 1 && len(args) != 2 {
			return fmt.Errorf("config-key takes [key] or [key, value], got %d args", len(args))
		}
		if strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("config-key needs a key")
		}
	}
	return nil
}

func argc(pred string, args []string, n int) error {
	if len(args) != n {
		return fmt.Errorf("%s takes %d args, got %d", pred, n, len(args))
	}
	return nil
}

// Lesson returns the lesson whose note is file (a vault-relative path with or
// without ".md"). The route index is not a lesson; a note with the lesson's
// name in another folder is not it either.
func (c Catalog) Lesson(file string) (Lesson, bool) {
	name := lessonFile(file)
	if name == "" {
		return Lesson{}, false
	}
	for _, r := range c.Routes {
		for _, l := range r.Lessons {
			if l.Path() == name {
				return l, true
			}
		}
	}
	return Lesson{}, false
}

// LessonByID returns the lesson with that ID.
func (c Catalog) LessonByID(id string) (Lesson, bool) {
	for _, r := range c.Routes {
		for _, l := range r.Lessons {
			if l.ID == id {
				return l, true
			}
		}
	}
	return Lesson{}, false
}

// RouteOf returns the route that owns file, whether it is the index or a
// lesson.
func (c Catalog) RouteOf(file string) (Route, bool) {
	name := lessonFile(file)
	if name == "" {
		return Route{}, false
	}
	for _, r := range c.Routes {
		if r.IndexPath() == name {
			return r, true
		}
		for _, l := range r.Lessons {
			if l.Path() == name {
				return r, true
			}
		}
	}
	return Route{}, false
}

// LessonCount returns how many lessons of the route count (an index note is
// not a lesson).
func (r Route) LessonCount() int { return len(r.Lessons) }

// lessonFile normalises a vault-relative note path to the form the catalog
// compares: slash separated, NFC (macOS writes decomposed accents), with
// ".md". The route's folder stays part of it, because that is where the lesson
// lives.
func lessonFile(file string) string {
	f := norm.NFC.String(filepath.ToSlash(strings.TrimSpace(file)))
	f = strings.TrimPrefix(f, "./")
	if f == "" {
		return ""
	}
	if !strings.HasSuffix(f, ".md") {
		f += ".md"
	}
	return f
}

// RouteByID returns the route with that ID.
func (c Catalog) RouteByID(id string) (Route, bool) {
	for _, r := range c.Routes {
		if r.ID == id {
			return r, true
		}
	}
	return Route{}, false
}

// FirstRoute returns the first route with lessons (or the first route).
func (c Catalog) FirstRoute() (Route, bool) {
	for _, r := range c.Routes {
		if len(r.Lessons) > 0 {
			return r, true
		}
	}
	if len(c.Routes) > 0 {
		return c.Routes[0], true
	}
	return Route{}, false
}

// RouteOfLesson returns the route that owns the lesson ID.
func (c Catalog) RouteOfLesson(id string) (Route, bool) {
	for _, r := range c.Routes {
		for _, l := range r.Lessons {
			if l.ID == id {
				return r, true
			}
		}
	}
	return Route{}, false
}

// NextLesson returns the lesson after lessonID in the route.
func (c Catalog) NextLesson(routeID, lessonID string) (Lesson, bool) {
	r, ok := c.RouteByID(routeID)
	if !ok {
		return Lesson{}, false
	}
	for i, l := range r.Lessons {
		if l.ID == lessonID {
			if i+1 < len(r.Lessons) {
				return r.Lessons[i+1], true
			}
			return Lesson{}, false
		}
	}
	return Lesson{}, false
}

// NeedsGit reports whether a step of l waits for a git event (":history"):
// without git those steps are skipped instead of failed.
func (l Lesson) NeedsGit() bool {
	for _, st := range l.Steps {
		if st.NeedsGit() {
			return true
		}
	}
	return false
}

// NeedsGit reports whether this step waits for a git event.
func (st Step) NeedsGit() bool {
	need := false
	walkCheck(st.Check, func(c Check) {
		if c.Kind == KindAction && strings.HasPrefix(c.Event, "history_") {
			need = true
		}
	})
	return need
}

// ConfigKeys returns the config keys the lesson's checks read, for the
// snapshot config-changed compares against.
func (l Lesson) ConfigKeys() []string {
	seen := map[string]bool{}
	var keys []string
	walkChecks(l, func(c Check) {
		if c.Kind != KindDisk || (c.Pred != "config-key" && c.Pred != "config-changed") || len(c.Args) == 0 {
			return
		}
		if !seen[c.Args[0]] {
			seen[c.Args[0]] = true
			keys = append(keys, c.Args[0])
		}
	})
	return keys
}

// walkChecks calls fn for every check of the lesson, inner checks included.
func walkChecks(l Lesson, fn func(Check)) {
	for _, st := range l.Steps {
		walkCheck(st.Check, fn)
	}
}

// walkCheck calls fn for a check and its inner checks.
func walkCheck(c Check, fn func(Check)) {
	fn(c)
	for _, sub := range c.Checks {
		walkCheck(sub, fn)
	}
}
