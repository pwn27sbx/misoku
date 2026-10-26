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
	store    *core.Store
	theme    theme.Theme
	keys     keymap.Map
	focus    focus
	viewMode viewMode

	width  int
	height int

	sidebarOpen   bool
	previewOpen   bool
	reading       bool // reading mode: only the preview, see toggleReading
	palette       *palette
	leaderPending bool
	leaderSeq     int  // bumps on every Space press, see leaderGuideMsg
	leaderGuide   bool // the leader guide shows while leaderPending

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

	promptBox *promptDialog
	menuBox   *menuDialog
	datePick  *datePicker

	configPath string // where the theme choice is saved ("" = don't save)
	quitting   bool   // "Save All" before quitting is in progress

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

	toast    string
	toastSeq int
	dragging bool // a mouse drag started in the editor

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
	graphLiveID string         // the note whose unsaved links the graph shows, see liveGraph

	pv previewSync // the preview's copy of the editor text, see preview_refresh.go

	syncVault  *config.Vault    // this folder's sync.vaults entry; nil = sync off
	syncUnsafe bool             // the folder is no place to set up sync (gitsync.UnsafeFolder)
	syncOff    string           // why sync is off, shown when the user asks for one
	syncLock   *gitsync.Lock    // held for the whole session once taken (S11)
	syncing    bool             // a sync command is running
	syncRun    *syncRun         // the sync started last, to wait for on quit (S14)
	now        func() time.Time // the clock; nil means time.Now
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

// New constructs the root App container for a workspace on disk. Nothing is
// seeded and nothing is opened until the user asks.
func New(store *core.Store, opts ...Option) Model {
	th := theme.DefaultTheme()

	m := Model{
		store:       store,
		theme:       th,
		keys:        defaultKeys(),
		focus:       focusSidebar,
		viewMode:    viewModeDashboard,
		sidebarOpen: true,
		previewOpen: true,
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
	return tea.Batch(m.content.Init(), checkWorkspaceLater(), m.syncOnOpen())
}

func (m *Model) layout() Layout {
	return CalculateLayout(m.width, m.height, m.sidebarOpen && !m.narrowWindow(), m.previewOpen, m.reading, m.split)
}

// updateLayout recalculates dimensions across child panes.
func (m *Model) updateLayout() {
	m.applyExplorerWidth()
	l := m.layout()
	m.dashboard.SetSize(m.width, max(m.height-footerHeight, 1))
	m.sidebar.SetSize(m.explorerW(l), l.BodyH)
	m.content.SetSize(l.EditorW, l.BodyH)
	m.preview.SetSize(l.PreviewW, l.BodyH)
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
// In reading mode the editor is hidden, so focusing it focuses the preview;
// whatever really needs the editor calls leaveReading first.
func (m *Model) focusPane(f focus) tea.Cmd {
	m.viewMode = viewModeWorkspace
	if f == focusEditor && m.reading {
		f = focusPreview
	}
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
	if !m.reading {
		out = append(out, focusEditor)
	}
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

// togglePreview is Space p. In reading mode it only leaves reading mode,
// bringing back the split the user had.
func (m *Model) togglePreview() tea.Cmd {
	if m.reading {
		m.leaveReading()
		return nil
	}
	m.previewOpen = !m.previewOpen
	m.updateLayout()
	if m.previewOpen && m.layout().PreviewW == 0 {
		m.setStatus("Window too narrow for the preview")
	}
	if m.focus == focusPreview && !m.previewOpen {
		return m.focusPane(focusEditor)
	}
	return nil
}

// toggleReading is Space r: show only the preview at full width, or go
// back to the split as it was (the divider positions are never touched).
func (m *Model) toggleReading() tea.Cmd {
	if m.reading {
		m.leaveReading()
		return nil
	}
	m.reading = true
	m.updateLayout()
	return m.focusPane(focusPreview)
}

// leaveReading brings the editor back, for Space r and Space p and before
// anything that needs the editor (the graph, the block menu, ":" commands).
func (m *Model) leaveReading() {
	if !m.reading {
		return
	}
	m.reading = false
	m.updateLayout()
}

func (m *Model) startFind() tea.Cmd { return m.openFinder() }

func (m *Model) goHome() {
	m.stashActive()
	m.reloadTree()
	m.viewMode = viewModeDashboard
	m.content.SetFocused(false)
	m.sidebar.SetFocused(false)
}
