// Package app acts as the root orchestrator component (<App/>).
package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/gitsync"
	"github.com/pwn27sbx/misoku/internal/globalgraph"
	"github.com/pwn27sbx/misoku/internal/graph"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/sidebar"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

type viewMode int

const (
	viewModeDashboard viewMode = iota
	viewModeWorkspace
)

type focus int

const (
	focusSidebar focus = iota
	focusEditor
	focusPreview
)

// Model represents the root App container state.
type Model struct {
	store *core.Store
	theme theme.Theme // what is drawn: baseTheme, transparent when ui.transparent is on
	// baseTheme is the opaque theme in use, see applyTheme.
	baseTheme theme.Theme
	keys      keymap.Map
	focus     focus
	viewMode  viewMode

	width  int
	height int

	sidebarOpen   bool
	noteView      noteView // the split or one full-width view, see toggleView
	palette       *palette
	leaderPending bool
	leaderSeq     int  // bumps on every Space press, see leaderGuideMsg
	leaderGuide   bool // the leader guide shows while leaderPending
	// leaderKeys are the keys typed after Space so far while a longer
	// sequence waits for its next key: ["o"] after "space o".
	leaderKeys []string

	// The explorer on narrow windows, see narrow.go.
	explorerHideWidth int
	explorerHideSet   bool
	explorerNarrow    bool

	// Open tabs, in order. Unsaved text lives in the buffers, never on disk.
	buffers  []*buffer
	active   string
	draftSeq int

	confirm *confirmDialog
	saveAs  *saveAsDialog
	finder  *finder
	themes  *themePicker
	browser *fileBrowser
	history *historyDialog // ":history", see history.go

	promptBox *promptDialog
	menuBox   *menuDialog
	datePick  *datePicker

	configPath  string // where the theme choice is saved ("" = don't save)
	recoveryDir string // where unsaved tabs are kept when the terminal closes ("" = off), see recovery.go
	quitting    bool   // "Save All" before quitting is in progress

	status    string
	statusErr bool
	// messages are the bar's last messages, oldest first (`:messages`).
	messages []sentMessage

	writeClipboard func(string) error
	openURL        func(string) error // opens links outside Misoku; nil = system opener
	notes          []core.Page        // every note, for resolving [[links]]
	treeSig        string             // shape of the tree last read, see checkWorkspace
	savedOnQuit    map[*buffer]bool   // tabs Save All already wrote, see continueSaveAll
	modeIcon       string             // config ui.mode_icon: one icon for every mode
	modifiedMark   string             // config ui.modified_mark: before a dirty tab name
	settings       *settingsState     // the settings menu's values and saves, see settings.go
	themeStamp     string             // user theme files as last seen, see syncUserTheme
	treeGen        uint64             // bumps on every applyTree, see workspaceScannedMsg
	scanning       bool               // a workspace scan is in flight, see CheckWorkspaceMsg
	leavingEditor  bool               // the folder editor's batch was asked for on leaving, see settleEditor

	toast    string
	toastSeq int
	tutor    *tutorGame  // misoku tutor's lessons, nil outside the tutorial, see tutor.go
	learn    *learnState // the home screen's per-route Learn entries, nil hides them, see learn.go
	// tutorNotice is the one-line migration notice shown when the tutorial
	// opens, see WithTutorNotice.
	tutorNotice string
	// learnPending is the first-run dialog still to be shown and learnBox
	// the dialog as drawn, so a resize tells it from another one; see
	// offerLearn.
	learnPending bool
	learnBox     *confirmDialog
	dragging     bool // a mouse drag started in the editor

	split    Split   // dragged divider positions
	resizing divider // the divider being dragged, if any

	dashboard dashboard.Model
	sidebar   sidebar.Model
	content   content.Model
	preview   preview.Model
	graph     graph.Model

	globalGraph globalgraph.Model // the global graph's tab, see globalgraph.go
	global      globalGraphState

	graphNotes  []core.Page // the workspace's notes and links, see linkGraph
	graphEdges  []core.Edge
	index       linkIndexState // every note's links, for the graph and backlinks
	mentions    mentionState   // the open note's unlinked mentions, see mentions.go
	graphLiveID string         // the note whose unsaved links the graph shows, see liveGraph

	pv previewSync // the preview's copy of the editor text, see preview_refresh.go

	syncVault  *config.Vault    // this folder's sync.vaults entry; nil = sync off
	syncUnsafe bool             // the folder is no place to set up sync (gitsync.UnsafeFolder)
	syncOff    string           // why sync is off, shown when the user asks for one
	syncLock   *gitsync.Lock    // held for the whole session once taken (S11)
	syncing    bool             // a sync command is running
	syncRun    *syncRun         // the sync started last, to wait for on quit (S14)
	now        func() time.Time // the clock; nil means time.Now
	dailyDir   string           // the daily notes' folder in the vault, see daily.go
	templates  string           // the templates' folder in the vault, see template.go
	sync       syncState        // what the last sync said (S9)
	// syncBase is when the last sync started or ended (or the window
	// opened): the next timed sync is due an interval later.
	syncBase time.Time
	// commitDue is set by a save that is not committed yet; lastSave is
	// when the last save was. See autoSync.
	commitDue bool
	lastSave  time.Time
	// syncQueued is a sync asked for while the commit after a save ran:
	// it starts when that commit ends.
	syncQueued bool
	// syncWarn is the config's sync warnings for this vault, still to be
	// shown with the outcome of the first sync.
	syncWarn string

	// keyCount counts the keys pressed; statusKeys is keyCount when
	// `:sync status` was asked for, so its answer can tell whether the
	// user typed meanwhile.
	keyCount   int
	statusKeys int

	// execProcess runs a program outside the screen (tea.ExecProcess);
	// tests replace it.
	execProcess func(*exec.Cmd, tea.ExecCallback) tea.Cmd

	// startNote is the note to open at once, from WithOpenNote (misoku
	// tutor resumes the lesson the player left). startCmd is the command
	// that opening it asked for, run with Init.
	startNote string
	startCmd  tea.Cmd

	// cursor is the terminal's own cursor, when main draws the mode with it
	// (see cursor.go). nil keeps the editor's drawn cursor.
	cursor CursorSink

	// setupBusy: a step of the sync setup assistant runs in the
	// background. setupNotes are the checks it passed so far, shown above
	// each of its questions.
	setupBusy  bool
	setupNotes []string
	// setupRun and setupStep name the step running or last run: a
	// setupMsg of any other is stale and ignored. setupStop cancels the
	// running step (Esc); setupStopped says it was cancelled, so its
	// result ends the assistant as a cancel does.
	setupRun     *gitsync.Setup
	setupStep    int
	setupStop    context.CancelFunc
	setupStopped bool
	// onSetupEnd hears that the assistant ended, on a screen that is not
	// the app (SetupDialogs); nil in the app.
	onSetupEnd func(gitsync.SetupStep) tea.Cmd
}

// Option customises the app at construction.
type Option func(*Model)

// WithTheme starts with the named base46 theme (unknown names are ignored).
func WithTheme(name string) Option {
	return func(m *Model) {
		if th, ok := theme.Get(name); ok {
			m.applyTheme(th, false)
		}
	}
}

// WithClipboard replaces the system clipboard writer (used by tests).
func WithClipboard(write func(string) error) Option {
	return func(m *Model) { m.writeClipboard = write }
}

// WithClipboardImage replaces how a picture is read from the clipboard (used by
// tests, which never touch a clipboard).
func WithClipboardImage(read func(context.Context) (core.ClipboardImageMsg, bool, error)) Option {
	return func(m *Model) { m.content.SetClipboardImage(read) }
}

// WithOpener replaces the system opener for web links and files (used by
// tests).
func WithOpener(open func(string) error) Option {
	return func(m *Model) { m.openURL = open }
}

// WithConfigPath saves theme changes to the given config file, and reads
// its "sync" section to decide whether this folder syncs.
func WithConfigPath(path string) Option {
	return func(m *Model) { m.configPath = path }
}

// WithRecovery keeps the unsaved tabs in dir when the terminal closes (or a
// SIGTERM stops Misoku) and offers them back on the next start. An empty dir
// turns it off, as in tests and embedders.
func WithRecovery(dir string) Option {
	return func(m *Model) { m.recoveryDir = dir }
}

// WithOpenNote opens the note id when the program starts, as picking it in
// the finder would: misoku tutor uses it to resume the lesson the player
// left. A note that is not in the vault opens nothing and says nothing, so
// a tutorial whose folder changed under it still starts.
func WithOpenNote(id string) Option {
	return func(m *Model) { m.startNote = id }
}

// New constructs the root App container for a workspace on disk. Nothing is
// seeded and nothing is opened until the user asks.
func New(store *core.Store, opts ...Option) Model {
	th := theme.DefaultTheme()

	m := Model{
		store:       store,
		theme:       th,
		baseTheme:   th,
		keys:        defaultKeys(),
		focus:       focusSidebar,
		viewMode:    viewModeDashboard,
		sidebarOpen: true,
		dashboard:   dashboard.New(th),
		sidebar:     sidebar.New(th),
		content:     content.New(th),
		preview:     preview.New(th),
		graph:       graph.New(th),
		globalGraph: globalgraph.New(th),
		execProcess: tea.ExecProcess,
		// Before the options: what they say (a broken shortcut in the
		// config) is kept for `:messages` too.
		messages: make([]sentMessage, 0, maxMessages),
	}
	m.sidebar.SetWorkspaceName(filepath.Base(store.Root()))
	m.dashboard.SetWorkspace(tildePath(store.Root()))
	m.reloadTree()
	for _, opt := range opts {
		opt(&m)
	}
	m.setupSync()
	m.themeStamp = theme.UserThemesStamp()
	if m.startNote != "" {
		// Before Init, so the first frame is the note and its cursor.
		m.startCmd = m.openInCurrentBuffer(m.startNote)
	}
	return m
}

func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// Init initializes child component commands.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.content.Init(), checkWorkspaceLater(), m.syncOnOpen(), m.startCmd, m.loadRecovery())
}

func (m *Model) layout() Layout {
	return CalculateLayout(m.width, m.height, m.sidebarOpen && !m.narrowWindow(), m.noteView == viewSplit, m.noteView == viewReading, m.split)
}

// updateLayout recalculates dimensions across child panes.
func (m *Model) updateLayout() {
	m.applyExplorerWidth()
	l := m.layout()
	m.dashboard.SetSize(m.width, max(m.height-footerHeight, 1))
	m.sidebar.SetSize(m.explorerW(l), l.BodyH)
	m.content.SetSize(l.EditorW, l.NoteH())
	m.content.SetMargin(m.editorMargin(l))
	m.preview.SetSize(l.PreviewW, l.NoteH())
	m.graph.SetSize(l.EditorW, l.BodyH)
	m.globalGraph.SetSize(l.EditorW, l.BodyH)
	if m.focus == focusPreview && l.PreviewW == 0 {
		m.focusPane(focusEditor)
	}
}

func (m *Model) reloadTree() {
	m.applyTree(m.store.List())
}

func (m *Model) applyTree(pages []core.Page) {
	m.treeSig = treeSignature(pages)
	m.treeGen++
	m.sidebar.SetPages(pages)
	m.dashboard.SetRecentPages(pages)
	m.applyStatuses()
	m.notes = nil
	for _, p := range pages {
		if !p.IsFolder {
			m.notes = append(m.notes, p)
		}
	}
	m.refreshLinks()
	m.refreshGraph()
}

// setStatus shows s in the bar until the next key, and keeps it for
// `:messages`.
func (m *Model) setStatus(s string) { m.say(s, false) }

// setError shows s in red until Esc or the next message, and keeps it for
// `:messages`.
func (m *Model) setError(s string) { m.say(s, true) }

// focusPane moves keyboard focus, opening the target pane if it was hidden.
// A hidden preview (a full-width view, or too narrow a window) hands the focus to
// the editor.
func (m *Model) focusPane(f focus) tea.Cmd {
	m.viewMode = viewModeWorkspace
	if f == focusSidebar && !m.sidebarOpen {
		m.sidebarOpen = true
		m.updateLayout()
	}
	if f != focusSidebar && m.narrowWindow() {
		m.sidebarOpen = false // the overlay closes when focus leaves it
	}
	if f == focusPreview && m.layout().PreviewW == 0 {
		f = focusEditor
	}
	if f == focusPreview {
		m.flushPreview()
	}
	m.focus = f
	m.sidebar.SetFocused(f == focusSidebar)
	m.preview.SetFocused(f == focusPreview)
	m.graph.SetFocused(f == focusEditor)
	m.globalGraph.SetFocused(f == focusEditor)
	if f == focusEditor {
		return m.content.Focus()
	}
	m.content.SetFocused(false)
	return nil
}

// panes lists the visible panes left to right.
func (m *Model) panes() []focus {
	l := m.layout()
	var out []focus
	if l.SidebarW > 0 || m.explorerOverlay() {
		out = append(out, focusSidebar)
	}
	out = append(out, focusEditor)
	if l.PreviewW > 0 {
		out = append(out, focusPreview)
	}
	return out
}

func (m *Model) movePane(delta int, wrap bool) tea.Cmd {
	ps := m.panes()
	if len(ps) == 0 {
		return nil
	}
	i := 0
	for j, p := range ps {
		if p == m.focus {
			i = j
		}
	}
	i += delta
	if wrap {
		i = (i + len(ps)) % len(ps)
	} else {
		i = max(0, min(i, len(ps)-1))
	}
	return m.focusPane(ps[i])
}

// toggleSidebar is Ctrl+B: show and focus the explorer, or hide it when it
// already has focus. Works from every editor mode.
// toggleSidebar works like Snacks.explorer() in LazyVim: a visible explorer
// is hidden in one press, whether it has the focus or not; a hidden one is
// shown and focused.
func (m *Model) toggleSidebar() tea.Cmd {
	if m.sidebarOpen && m.viewMode == viewModeWorkspace {
		m.sidebarOpen = false
		m.updateLayout()
		return m.focusPane(focusEditor)
	}
	if m.content.Mode() == content.ModeInsert {
		m.content.ExitInsert()
	}
	return m.focusPane(focusSidebar)
}

// toggleExplorerFocus is Space e, the LazyVim <leader>e behaviour: an
// explorer that has the focus closes and gives the focus back to the editor;
// an open one without the focus gets it; a hidden one is shown and focused.
// With no note open there is nothing to go back to, so it stays. The home
// screen keeps opening it, as it did before the toggle.
func (m *Model) toggleExplorerFocus() tea.Cmd {
	if m.viewMode != viewModeWorkspace || !m.sidebarOpen || m.focus != focusSidebar {
		return m.focusPane(focusSidebar)
	}
	if m.activeBuffer() == nil {
		return nil
	}
	if m.explorerOverlay() {
		return m.closeExplorerOverlay()
	}
	m.sidebarOpen = false
	m.updateLayout()
	return m.focusPane(focusEditor)
}

// noteView is how the open note is laid out: the split of editor and side
// preview, or one of the two full-width views.
type noteView int

const (
	viewSplit   noteView = iota // the editor and the side preview
	viewReading                 // Space p: the note rendered, read-only, full width
	viewEditor                  // Space r: the Markdown editor alone, full width
)

// editorMargin is the editor's blank columns each side of the text: the
// preview's page margins in the reading view, where the editor stands in
// for the preview, and none elsewhere.
func (m *Model) editorMargin(l Layout) int {
	if m.noteView != viewReading {
		return 0
	}
	return preview.PageMargin(l.EditorW)
}

// toggleView is Space p (viewReading) and Space r (viewEditor): show v and
// focus the note, or go back to the split when v is already shown. Either
// key switches straight from the other view. The divider positions are
// never touched, so the split comes back as it was.
func (m *Model) toggleView(v noteView) tea.Cmd {
	if m.noteView == v {
		m.setNoteView(viewSplit)
		if m.layout().PreviewW == 0 {
			m.setStatus("Window too narrow for the preview")
		}
		return nil
	}
	m.setNoteView(v)
	return m.focusPane(focusEditor)
}

// setNoteView lays the note out as v: the reading view draws every line
// rendered and refuses edits, the others draw raw Markdown and edit.
func (m *Model) setNoteView(v noteView) {
	m.noteView = v
	m.content.SetLive(v == viewReading)
	m.content.SetReadOnly(v == viewReading)
	if v != viewSplit {
		preview.ClearImages() // the preview is hidden: drop its terminal images
	}
	m.updateLayout() // after SetLive: SetSize recomputes the text width
}

// leaveReading goes from the reading view back to the split, before
// anything that needs the split or typing (the graph, the block menu, a new
// note). The editor view stays: it already edits.
func (m *Model) leaveReading() {
	if m.noteView == viewReading {
		m.setNoteView(viewSplit)
	}
}

func (m *Model) startFind() tea.Cmd { return m.openFinder() }

func (m *Model) goHome() {
	m.stashActive()
	m.reloadTree()
	m.viewMode = viewModeDashboard
	m.content.SetFocused(false)
	m.sidebar.SetFocused(false)
}
