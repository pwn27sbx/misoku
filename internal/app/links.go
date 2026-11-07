package app

import (
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// Links between notes: [[Note]], [[Note#Heading]], [[Note#^block]],
// ![[embeds]], Markdown links and web addresses.

// currentNoteID is the active note's ID ("" for a draft or no tab).
func (m *Model) currentNoteID() string {
	if b := m.activeBuffer(); b != nil && !b.draft() {
		return b.id
	}
	return ""
}

// noteLookup resolves link targets for the preview against the notes in
// the explorer, reading linked notes from their open tab or from disk.
func (m *Model) noteLookup() preview.NoteLookup {
	live := map[string]string{}
	for _, b := range m.buffers {
		if !b.draft() && b.id != m.active {
			live[b.id] = b.text
		}
	}
	return preview.StoreNotes(m.store, m.notes, m.currentNoteID(), live)
}

// refreshLinks re-resolves the preview's links and its backlinks.
func (m *Model) refreshLinks() {
	m.preview.SetNotes(m.noteLookup())
	var back []core.Backlink
	if id := m.currentNoteID(); id != "" {
		back = m.backlinks(id)
	}
	m.preview.SetBacklinks(back)
	m.requestMentions(false)
}

// followHit opens a link clicked in the preview.
func (m *Model) followHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case preview.HitWikiLink:
		return m.followWikiLink(core.ParseWikiLink(h.Arg))
	case preview.HitURLLink:
		return m.followURL(h.Arg)
	case preview.HitNoteLink:
		keep := m.focus
		cmd := m.openInCurrentBuffer(h.Arg)
		if m.active == h.Arg && h.Index > 0 {
			m.content.GotoLine(h.Index)
			m.preview.ScrollToLine(h.Index - 1)
		}
		return tea.Batch(cmd, m.focusPane(keep))
	case preview.HitMentionLink:
		return m.askLinkMentions(h.Arg)
	}
	return nil
}

// followWikiLink opens the note a [[link]] names at its heading or block.
// A link to a note that doesn't exist yet opens a new draft with that
// name; nothing is written until it is saved.
func (m *Model) followWikiLink(l core.WikiLink) tea.Cmd {
	keep := m.focus
	if l.Target == "" {
		m.jumpToSection(l)
		return nil
	}
	if preview.IsImageName(l.Target) {
		if abs, ok := m.store.FindFile(l.Target, m.currentNoteID()); ok {
			return m.openExternal(abs)
		}
		m.setError("Not found: " + l.Target)
		return nil
	}
	p, ok := core.ResolveNote(m.notes, l.Target, m.currentNoteID())
	if !ok {
		return m.newLinkedDraft(l.Target)
	}
	cmd := m.openInCurrentBuffer(p.ID)
	if m.active != p.ID {
		return cmd
	}
	m.jumpToSection(l)
	if keep == focusPreview {
		return tea.Batch(cmd, m.focusPane(focusPreview))
	}
	return cmd
}

// jumpToSection moves the editor and preview to a link's heading or block
// in the open note.
func (m *Model) jumpToSection(l core.WikiLink) {
	if l.Heading == "" && l.Block == "" {
		return
	}
	line := core.LinkLine(m.content.Value(), l)
	if line < 0 {
		m.setError("No section \"" + strings.TrimPrefix(l.Ref(), l.Target+"#") + "\" in this note")
		return
	}
	m.content.GotoLine(line + 1)
	m.preview.ScrollToLine(line)
}

// newLinkedDraft starts the note an unresolved link points to. A target
// with folders ("Projects/AWS") is offered in that folder; a bare name next
// to the linking note.
func (m *Model) newLinkedDraft(target string) tea.Cmd {
	target = strings.Trim(filepath.ToSlash(strings.TrimSuffix(target, ".md")), "/")
	dir, name := path.Split(target)
	if dir == "" {
		if id := m.currentNoteID(); id != "" {
			dir = path.Dir(id)
		} else if b := m.activeBuffer(); b != nil {
			dir = b.dir
		}
	}
	if dir == "." {
		dir = ""
	}
	cmd := m.newDraft("")
	if b := m.activeBuffer(); b != nil && b.draft() {
		b.title, b.dir = name, strings.Trim(dir, "/")
		m.content.SetBuffer(b.page(), true)
		m.preview.SetPage(b.page())
	}
	m.setStatus("New note \"" + name + "\": save to create it")
	return cmd
}

// followURL opens a Markdown link: a note (optionally at "#heading"), a web
// address in the browser, or any other local file with its default app.
func (m *Model) followURL(u string) tea.Cmd {
	u = strings.TrimSpace(u)
	if strings.Contains(u, "://") || strings.HasPrefix(u, "mailto:") {
		return m.openExternal(u)
	}
	file, frag, _ := strings.Cut(u, "#")
	if f, err := url.PathUnescape(file); err == nil {
		file = f
	}
	l := core.WikiLink{Heading: frag}
	if file == "" {
		m.jumpToSection(l)
		return nil
	}
	b := m.activeBuffer()
	if b == nil {
		return nil
	}
	if filepath.IsAbs(file) {
		return m.openExternal(file)
	}
	rel := b.dir
	if !b.draft() {
		rel = path.Dir(b.id)
	}
	id := path.Clean(path.Join(rel, filepath.ToSlash(file)))
	if strings.HasSuffix(strings.ToLower(id), ".md") {
		if _, err := m.store.Get(id); err == nil {
			keep := m.focus
			cmd := m.openInCurrentBuffer(id)
			if m.active == id {
				m.jumpToSection(l)
			}
			if keep == focusPreview {
				return tea.Batch(cmd, m.focusPane(focusPreview))
			}
			return cmd
		}
	}
	abs := filepath.Join(m.store.Root(), filepath.FromSlash(id))
	if _, err := os.Stat(abs); err != nil {
		m.setError("Not found: " + file)
		return nil
	}
	return m.openExternal(abs)
}

// openExternal hands a URL or file to the system's default app. Web and mail
// addresses, and files inside the vault that the system will not run, open at
// once; anything else — another scheme, a path outside the vault, or an
// executable name — asks first, since the opener may run it.
func (m *Model) openExternal(target string) tea.Cmd {
	if !m.openDirectly(target) {
		m.confirm = &confirmDialog{
			title:      "Open " + shortTarget(target) + "?",
			message:    []string{"The system may run it, or it is outside the vault."},
			buttons:    []string{"Open", "Cancel"},
			deliberate: true, // a stray letter must not open it
			selected:   1,    // Cancel is the safe default
			onChoose: func(m *Model, choice int) tea.Cmd {
				if choice != 0 {
					return nil
				}
				return m.openExternalNow(target)
			},
		}
		return nil
	}
	return m.openExternalNow(target)
}

// executableExts are names the system opener can run instead of showing:
// macOS runs .command and .app, Windows rundll32 runs .bat and .exe, and an
// interpreter may take the rest.
var executableExts = map[string]bool{
	".app": true, ".appimage": true, ".bat": true, ".bash": true, ".cmd": true,
	".com": true, ".command": true, ".desktop": true, ".exe": true,
	".fish": true, ".hta": true,
	".jar": true, ".js": true, ".jse": true, ".lnk": true, ".msi": true,
	".pl": true, ".ps1": true, ".psm1": true, ".py": true, ".rb": true,
	".reg": true, ".scr": true, ".sh": true, ".url": true, ".vbe": true,
	".vbs": true, ".wsf": true, ".wsh": true, ".zsh": true,
}

// executableName reports whether a local file name may be run by the system
// opener rather than opened in a viewer.
func executableName(name string) bool {
	return executableExts[strings.ToLower(filepath.Ext(name))]
}

// openDirectly reports whether target may go to the system opener without
// asking: web and mail addresses, and files inside the vault whose name the
// system will not run. Anything else asks first.
func (m *Model) openDirectly(target string) bool {
	if strings.Contains(target, "://") {
		u, err := url.Parse(target)
		if err != nil {
			return false
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https":
			return true
		}
		return false
	}
	if strings.HasPrefix(strings.ToLower(target), "mailto:") {
		return true
	}
	if m.store == nil {
		return false
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(m.store.Root(), abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if info, err := os.Stat(abs); err == nil && info.Mode().Perm()&0o111 != 0 {
		return false // a program without an extension, run by `open` on macOS
	}
	return !executableName(abs)
}

// openExternalNow hands target to the opener, once openExternal allows it.
func (m *Model) openExternalNow(target string) tea.Cmd {
	open := m.openURL
	if open == nil {
		open = systemOpen
	}
	if err := open(target); err != nil {
		m.setError("Could not open " + target + ": " + err.Error())
		return nil
	}
	return m.showToast("Opened " + shortTarget(target))
}

func shortTarget(s string) string {
	if r := []rune(s); len(r) > 48 {
		return string(r[:47]) + "…"
	}
	return s
}

// systemOpen starts the platform's opener without waiting for it.
func systemOpen(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

var (
	editorWikiRe = regexp.MustCompile(`!?\[\[[^\[\]\n]+\]\]`)
	editorMdRe   = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]*)\)`)
	editorURLRe  = regexp.MustCompile(`<?(https?://|mailto:)[^\s<>()\[\]]+>?`)
)

// followEditorLink opens the link under the editor cursor ("gd" / "gx").
func (m *Model) followEditorLink(msg core.FollowLinkMsg) tea.Cmd {
	runes := []rune(msg.Line)
	at := len(string(runes[:min(max(msg.Col, 0), len(runes))]))
	within := func(loc []int) bool { return loc[0] <= at && at < loc[1] }
	for _, loc := range editorWikiRe.FindAllStringIndex(msg.Line, -1) {
		if within(loc) {
			l, _, _ := core.WikiLinkAt(strings.TrimPrefix(msg.Line[loc[0]:loc[1]], "!"))
			return m.followWikiLink(l)
		}
	}
	for _, loc := range editorMdRe.FindAllStringSubmatchIndex(msg.Line, -1) {
		if within(loc) {
			return m.followURL(strings.Trim(msg.Line[loc[2]:loc[3]], "<>"))
		}
	}
	for _, loc := range editorURLRe.FindAllStringIndex(msg.Line, -1) {
		if within(loc) {
			return m.followURL(strings.Trim(msg.Line[loc[0]:loc[1]], "<>"))
		}
	}
	m.setError("No link under the cursor")
	return nil
}

// insertWikiLink finishes a "[[" the user typed with the picked note's
// name, using its path when another note has the same name.
func (m *Model) insertWikiLink(target core.Page) {
	m.insertWikiText(m.linkName(target))
}

// linkName is what a [[link]] to target holds: its name, or its path when
// another note has the same name.
func (m *Model) linkName(target core.Page) string {
	for _, p := range m.notes {
		if p.ID != target.ID && strings.EqualFold(p.Title, target.Title) {
			return strings.TrimSuffix(target.ID, ".md")
		}
	}
	return target.Title
}

func (m *Model) insertWikiText(name string) {
	m.content.InsertText(name + "]]")
	m.preview.SetContent(m.content.Value())
	m.refreshModified()
}
