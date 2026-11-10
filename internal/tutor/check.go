package tutor

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	headingRe = regexp.MustCompile(`^ {0,3}(#{1,6})[ \t]+(.*?)[ \t#]*$`)
	delimRe   = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)
	wikiRe    = regexp.MustCompile(`\[\[([^\[\]\n]*)\]\]`)
	blockIDRe = regexp.MustCompile(`\s\^([A-Za-z0-9_-]+)\s*$`)
	taskRe    = regexp.MustCompile(`^\s*[-*+]\s+\[([ xX-])\]\s+(.*\S)\s*$`)
)

// Env answers the questions a check asks about the vault and the app without
// the engine doing IO itself. The app supplies it from what it already knows;
// a nil callback fails the check that needs it. LessonFile is the note the
// check belongs to, vault-relative (for file-changed), and LessonDir is the
// route's folder: a disk path is looked for there first and at the vault root
// second. Today is AAAA-MM-DD for {today} paths (empty uses the local date).
type Env struct {
	Exists        func(name string) bool
	FolderExists  func(name string) bool
	Changed       func(file string) bool
	NoteText      func(note string) (string, bool)
	Config        func(key string) (string, bool)
	ConfigChanged func(key string) bool
	LessonFile    string
	LessonDir     string
	Today         string
	// full is the note's text before the original's lines were ignored, so
	// a heading or block the lesson already has can still be the target of
	// the player's new link. The store sets it; empty means text is all
	// there is.
	full string
}

// date is the date {today} expands to.
func (e Env) date() string {
	if e.Today != "" {
		return e.Today
	}
	return time.Now().Format("2006-01-02")
}

// expand replaces the {today} token with the local date as AAAA-MM-DD. It is
// used in vault paths and in text and fence patterns alike (E14).
func (e Env) expand(s string) string { return strings.ReplaceAll(s, "{today}", e.date()) }

// Pass reports whether the check holds for text, a note's content. It is pure
// given env: text and fence checks read text, disk checks call env, and an
// action check never passes from text (the app records its event instead).
func (c Check) Pass(text string, env Env) bool {
	switch c.Kind {
	case KindText:
		return c.match(stripCode(text), env)
	case KindTextRaw:
		return c.match(stripFences(text), env)
	case KindTextFence:
		full := text
		if env.full != "" {
			full = env.full
		}
		return c.fenceHasLine(full, text, env.expand(c.Pattern))
	case KindDisk:
		return c.disk(env)
	case KindAll:
		if len(c.Checks) == 0 {
			return false
		}
		for _, sub := range c.Checks {
			if !sub.Pass(text, env) {
				return false
			}
		}
		return true
	}
	return false
}

// match applies a text predicate to lines already stripped of code.
func (c Check) match(lines []string, env Env) bool {
	arg := func(i int) string {
		if i >= len(c.Args) {
			return ""
		}
		return env.expand(c.Args[i])
	}
	switch c.Pred {
	case "line-contains":
		return anyLine(lines, func(l string) bool { return strings.Contains(l, arg(0)) })
	case "line-starts":
		return anyLine(lines, func(l string) bool { return strings.HasPrefix(l, arg(0)) })
	case "line-equals":
		return anyLine(lines, func(l string) bool { return strings.TrimSpace(l) == arg(0) })
	case "line-ends":
		return anyLine(lines, func(l string) bool { return strings.HasSuffix(strings.TrimRight(l, " \t\r"), arg(0)) })
	case "n-lines-start":
		n, _ := strconv.Atoi(arg(0))
		count := 0
		for _, l := range lines {
			if strings.HasPrefix(l, arg(1)) {
				count++
			}
		}
		return count >= n
	case "absent":
		// Absence must see the whole note: a line the lesson already has is
		// the line the player is asked to delete, so ignoring it would pass
		// the step on the untouched note.
		return !anyLine(contextLines(lines, env), func(l string) bool { return strings.Contains(l, arg(0)) })
	case "heading-exists":
		level, _ := strconv.Atoi(arg(0))
		for _, h := range headings(lines) {
			if h.level == level && strings.EqualFold(strings.TrimSpace(h.text), arg(1)) {
				return true
			}
		}
	case "wikilink-resolves":
		want := arg(0)
		for _, k := range links(lines) {
			if k.note != "" && strings.EqualFold(k.note, want) && env.Exists != nil && env.Exists(want) {
				return true
			}
		}
	case "blockref-exists":
		want := arg(0)
		ids := map[string]bool{}
		for _, l := range contextLines(lines, env) {
			if m := blockIDRe.FindStringSubmatch(l); m != nil {
				ids[m[1]] = true
			}
		}
		if !ids[want] {
			return false
		}
		for _, k := range links(lines) {
			if k.note == "" && strings.HasPrefix(k.sub, "^") && k.sub[1:] == want {
				return true
			}
		}
	case "headinglink-exists":
		want := arg(0)
		titles := map[string]bool{}
		for _, h := range headings(contextLines(lines, env)) {
			titles[strings.ToLower(strings.TrimSpace(h.text))] = true
		}
		if !titles[strings.ToLower(want)] {
			return false
		}
		for _, k := range links(lines) {
			if k.note == "" && k.hasHash && !strings.HasPrefix(k.sub, "^") && strings.EqualFold(k.sub, want) {
				return true
			}
		}
	case "table-shape":
		return c.tableShape(lines, env)
	case "heading-under":
		return c.headingUnder(lines, env, arg(0), arg(1), arg(2), arg(3))
	case "task-status":
		return c.taskStatus(lines, arg(0), arg(1))
	}
	return false
}

// headingUnder reports whether a new heading of childLevel/childText appears
// after the parent heading and before the next heading of the parent's level
// or above. The parent may be the lesson's own heading; the child must be the
// player's.
func (c Check) headingUnder(lines []string, env Env, parentLevelArg, parentText, childLevelArg, childText string) bool {
	parentLevel, _ := strconv.Atoi(parentLevelArg)
	childLevel, _ := strconv.Atoi(childLevelArg)
	full := contextLines(lines, env)
	parentAt := -1
	for i, l := range full {
		m := headingRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		level := len(m[1])
		text := strings.TrimSpace(m[2])
		if parentAt >= 0 && level <= parentLevel && !(level == parentLevel && strings.EqualFold(text, parentText)) {
			parentAt = -1
		}
		if level == parentLevel && strings.EqualFold(text, parentText) {
			parentAt = i
		}
		if parentAt >= 0 && level == childLevel && strings.EqualFold(text, childText) &&
			(env.full == "" || (i < len(lines) && strings.TrimSpace(lines[i]) != "")) {
			return true
		}
	}
	return false
}

// taskStatus reports whether a task line of the given state contains text. It
// takes any bullet, any indentation, and x/X as done.
func (c Check) taskStatus(lines []string, state, text string) bool {
	want := strings.ToLower(strings.TrimSpace(text))
	for _, l := range lines {
		m := taskRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		box := m[1]
		ok := false
		switch state {
		case "done":
			ok = box == "x" || box == "X"
		case "open":
			ok = box == " "
		case "doing":
			ok = box == "-"
		}
		if ok && strings.Contains(strings.ToLower(m[2]), want) {
			return true
		}
	}
	return false
}

// fenceHasLine reports whether a fence of the check's kind has a new line
// that contains the pattern. full has the fence structure; filtered is the
// note with the embedded original's lines blanked, so the matching line must
// be the player's.
func (c Check) fenceHasLine(full, filtered, pattern string) bool {
	lines := strings.Split(full, "\n")
	player := strings.Split(filtered, "\n")
	fence := ""
	matching := false
	for i, l := range lines {
		t := strings.TrimLeft(l, " ")
		if fence == "" {
			if f := fenceOpen(t); f != "" {
				fence = f
				matching = fenceMatches(strings.TrimSpace(strings.TrimPrefix(t, f)), c.Fence)
			}
			continue
		}
		if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]+" \t") == "" {
			fence = ""
			matching = false
			continue
		}
		if matching && strings.Contains(l, pattern) &&
			(i >= len(player) || strings.TrimSpace(player[i]) != "") {
			return true
		}
	}
	return false
}

// tableShape reports whether a table with the check's cell count and a
// separator row exists, with at least one line of it being the player's.
func (c Check) tableShape(lines []string, env Env) bool {
	cells, _ := strconv.Atoi(env.expand(c.Args[0]))
	all := lines
	player := lines
	if env.full != "" {
		all = stripCode(env.full)
		player = lines
	}
	isNew := func(i int) bool {
		return env.full == "" || (i < len(player) && strings.TrimSpace(player[i]) != "")
	}
	for i := 0; i+1 < len(all); i++ {
		if len(splitRow(all[i])) != cells || !strings.Contains(all[i], "|") {
			continue
		}
		next := strings.TrimSpace(all[i+1])
		if len(splitRow(all[i+1])) != cells || !delimRe.MatchString(next) {
			continue
		}
		if isNew(i) || isNew(i+1) {
			return true
		}
		for j := i + 2; j < len(all) && strings.Contains(all[j], "|"); j++ {
			if isNew(j) {
				return true
			}
		}
	}
	return false
}

// fenceMatches reports whether an opening fence's info string is the wanted
// kind. A plain ``` fence is a code block.
func fenceMatches(info, want string) bool {
	first := ""
	if fields := strings.Fields(info); len(fields) > 0 {
		first = strings.ToLower(fields[0])
	}
	if want == "code" {
		return first == "" || first == "code" || first == "text"
	}
	return first == strings.ToLower(want)
}

// vaultPaths are the vault-relative places a disk path may name. While a
// lesson is open the explorer has its note selected, so most actions write
// inside the route's folder; some (`:w name`, a template, the daily note)
// write at the root. The route folder is tried first, then the root.
func (e Env) vaultPaths(p string) []string {
	if e.LessonDir == "" {
		return []string{p}
	}
	return []string{e.LessonDir + "/" + p, p}
}

// noteExists reports whether a note named p is inside the lesson's route
// folder or at the vault root, with the route folder tried first. Without a
// reader there is no evidence, so the check fails.
func (e Env) noteExists(p string) bool {
	if e.Exists == nil {
		return false
	}
	for _, path := range e.vaultPaths(p) {
		if e.Exists(path) {
			return true
		}
	}
	return false
}

// folderExists is noteExists for a folder.
func (e Env) folderExists(p string) bool {
	if e.FolderExists == nil {
		return false
	}
	for _, path := range e.vaultPaths(p) {
		if e.FolderExists(path) {
			return true
		}
	}
	return false
}

// firstNote returns the text of the first note among p's places that exists:
// a note with the same name at the root never shadows the route folder's.
func (e Env) firstNote(p string) (string, bool) {
	if e.NoteText == nil {
		return "", false
	}
	for _, path := range e.vaultPaths(p) {
		if text, ok := e.NoteText(path); ok {
			return text, true
		}
	}
	return "", false
}

// disk applies a disk predicate through env.
func (c Check) disk(env Env) bool {
	arg := func(i int) string {
		if i >= len(c.Args) {
			return ""
		}
		return env.expand(c.Args[i])
	}
	switch c.Pred {
	case "exists":
		return env.noteExists(arg(0))
	case "no-file":
		// No reader means no evidence: the step never passes on a missing
		// callback.
		return env.Exists != nil && !env.noteExists(arg(0))
	case "folder-exists":
		return env.folderExists(arg(0))
	case "file-changed":
		return env.Changed != nil && env.Changed(env.LessonFile)
	case "other-note-text":
		text, ok := env.firstNote(arg(0))
		return ok && strings.Contains(text, arg(1))
	case "config-key":
		if env.Config == nil {
			return false
		}
		value, ok := env.Config(arg(0))
		if !ok {
			return false
		}
		return len(c.Args) == 1 || value == arg(1)
	case "config-changed":
		return env.ConfigChanged != nil && env.ConfigChanged(arg(0))
	}
	return false
}

// HasAction reports whether the check or one of its sub-checks waits for an
// event.
func (c Check) HasAction() bool {
	switch c.Kind {
	case KindAction:
		return true
	case KindAll:
		for _, sub := range c.Checks {
			if sub.HasAction() {
				return true
			}
		}
	}
	return false
}

// contextLines are the lines a link may point at: the full note when the
// store knows it, because a lesson's own heading or block is a valid target,
// and the check's lines otherwise.
func contextLines(lines []string, env Env) []string {
	if env.full == "" {
		return lines
	}
	return stripCode(env.full)
}

func anyLine(lines []string, ok func(string) bool) bool {
	for _, l := range lines {
		if ok(l) {
			return true
		}
	}
	return false
}

type heading struct {
	level int
	text  string
}

func headings(lines []string) []heading {
	var hs []heading
	for _, l := range lines {
		if m := headingRe.FindStringSubmatch(l); m != nil {
			hs = append(hs, heading{level: len(m[1]), text: m[2]})
		}
	}
	return hs
}

// link is a parsed [[note#sub|alias]].
type link struct {
	note    string
	sub     string
	hasHash bool
}

func links(lines []string) []link {
	var ks []link
	for _, l := range lines {
		for _, m := range wikiRe.FindAllStringSubmatch(l, -1) {
			target, _, _ := strings.Cut(m[1], "|")
			note, sub, hash := strings.Cut(target, "#")
			ks = append(ks, link{note: strings.TrimSpace(note), sub: strings.TrimSpace(sub), hasHash: hash})
		}
	}
	return ks
}

// splitRow splits a table row into its trimmed cells, ignoring the outer
// pipes and honoring escaped ones.
func splitRow(line string) []string {
	s := strings.TrimSpace(line)
	if !strings.Contains(s, "|") {
		return nil
	}
	var cells []string
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			b.WriteByte(s[i])
			i++
			b.WriteByte(s[i])
		case s[i] == '|':
			cells = append(cells, b.String())
			b.Reset()
		default:
			b.WriteByte(s[i])
		}
	}
	cells = append(cells, b.String())
	if len(cells) > 0 && strings.TrimSpace(cells[0]) == "" {
		cells = cells[1:]
	}
	if len(cells) > 0 && strings.TrimSpace(cells[len(cells)-1]) == "" {
		cells = cells[:len(cells)-1]
	}
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// stripCode splits text into lines, blanking fenced code blocks (``` or ~~~)
// and removing inline code spans.
func stripCode(text string) []string { return codeLines(text, true) }

// stripFences blanks fenced code blocks but keeps inline code, for the one
// check that practises inline code itself.
func stripFences(text string) []string { return codeLines(text, false) }

func codeLines(text string, stripInlineCode bool) []string {
	lines := strings.Split(text, "\n")
	fence := ""
	for i, l := range lines {
		t := strings.TrimLeft(l, " ")
		if fence != "" {
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]+" \t") == "" {
				fence = ""
			}
			lines[i] = ""
			continue
		}
		if f := fenceOpen(t); f != "" {
			fence = f
			lines[i] = ""
			continue
		}
		if stripInlineCode {
			lines[i] = stripInline(l)
		}
	}
	return lines
}

func fenceOpen(t string) string {
	for _, c := range []byte{'`', '~'} {
		n := 0
		for n < len(t) && t[n] == c {
			n++
		}
		if n >= 3 {
			return t[:n]
		}
	}
	return ""
}

// stripInline removes `code` spans: a run of n backticks up to the next run of
// exactly n. An unmatched run is kept as text.
func stripInline(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '`' {
			b.WriteByte(s[i])
			i++
			continue
		}
		n := run(s, i)
		end := -1
		for j := i + n; j < len(s); {
			if s[j] == '`' {
				m := run(s, j)
				if m == n {
					end = j + m
					break
				}
				j += m
				continue
			}
			j++
		}
		if end < 0 {
			b.WriteString(s[i : i+n])
			i += n
			continue
		}
		i = end
	}
	return b.String()
}

func run(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}

// DirEnv answers disk checks from a tutorial folder, for the scan that runs
// before the program starts. Paths are relative to the tutorial vault; note
// names leave out ".md".
func DirEnv(dir string) Env {
	note := func(name string) string { return filepath.Join(dir, name+".md") }
	return Env{
		Exists: func(name string) bool {
			_, err := os.Stat(note(name))
			return err == nil
		},
		FolderExists: func(name string) bool {
			st, err := os.Stat(filepath.Join(dir, name))
			return err == nil && st.IsDir()
		},
		NoteText: func(name string) (string, bool) {
			b, err := os.ReadFile(note(name))
			return string(b), err == nil
		},
	}
}
