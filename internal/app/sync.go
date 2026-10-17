package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	// syncGitTimeout limits each git command of a sync; syncTimeout limits
	// the whole sync, so a hung remote never leaves one running forever.
	syncGitTimeout = time.Minute
	syncTimeout    = 5 * time.Minute
	// statusTimeout limits reading `:sync status`.
	statusTimeout = 30 * time.Second
	// saveCommitDelay is how long saves must pause before they are
	// committed, so a burst of saves makes one commit.
	saveCommitDelay = 3 * time.Second
)

// SyncNowMsg asks the app to sync the vault now, as the sync_now key and
// ":sync" (core.SyncMsg) do.
// Opening a vault listed in sync.vaults sends one. It is exported so
// embedders and tests can ask for a sync; in a folder that does not sync it
// only says why.
type SyncNowMsg struct{}

// syncDoneMsg brings a finished sync command back to Update.
type syncDoneMsg struct {
	local  bool          // only the commit after a save, not a sync
	auto   bool          // started on its own, not asked for
	lock   *gitsync.Lock // the vault's lock when this window holds it
	paused gitsync.State // why the vault cannot sync now (Detect), when Status is Paused
	busy   bool          // another window holds the lock
	err    error         // the lock could not be taken
	res    gitsync.SyncResult
	copies []string // conflict copies on disk after the sync (S9)
	// leftover is git's index.lock the sync stopped on, as it was then
	// (S13): the user is asked before it is removed.
	leftover *gitsync.LeftoverLock
}

// syncKind is what the sync indicator in the status bar shows (S9).
type syncKind int

const (
	syncUnknown syncKind = iota // no sync finished yet in this session
	syncOK
	syncOffline
	syncPaused
	syncError
	syncBusy
	syncRetry // stopped short; the next sync tries again (not an error)
)

// syncState is what the last sync said, for the indicator and `:sync
// status`.
type syncState struct {
	kind     syncKind
	pending  int
	copies   []string  // conflict copies on disk
	lastOK   time.Time // when a sync last finished
	lastErr  string    // the most recent error or pause message
	errAt    time.Time
	warnings []string              // skipped sync.vaults entries (config.Sync.Parse)
	files    []string              // files the last commit left out or warned about (D7, S15)
	leftover *gitsync.LeftoverLock // the lock the last sync stopped on
}

// setupSync reads the "sync" section of the config file and decides, once,
// whether this folder syncs. It runs no git: a folder that is not listed,
// or MISOKU_SYNC=0, must never start one. The home screen offers "Sync
// between computers" while the folder is not connected (see syncOffered).
func (m *Model) setupSync() {
	defer func() { m.dashboard.SetSyncSetup(m.syncOffered()) }()
	m.syncBase = m.clock()
	home, _ := os.UserHomeDir() // none known: the root and system folders are still not offered
	m.syncUnsafe = gitsync.UnsafeFolder(m.store.Root(), home)
	m.syncOff = "Sync is off: this folder is not listed in sync.vaults"
	if gitsync.Disabled() {
		m.syncOff = "Sync is off (MISOKU_SYNC=0)"
		return
	}
	// An unreadable config leaves sync off, the safe side. Startup only
	// reported it on the terminal Misoku then covers, so say it here.
	cfg, err := config.Load(m.configPath)
	if err != nil {
		m.syncOff = "Sync is off: the config file could not be read (" + oneLine(err.Error()) + ")"
		return
	}
	vaults, warnings := cfg.Sync.Parse()
	m.sync.warnings = warnings
	if v, ok := gitsync.Match(vaults, m.store.Root()); ok {
		m.syncVault = &v
	}
	if len(warnings) == 0 {
		return
	}
	// A skipped entry may be the one meant for this folder: say so, both
	// now and when a sync is asked for, instead of only "not listed".
	why := strings.Join(warnings, "; ")
	if m.syncVault == nil {
		m.syncOff += " (" + why + ")"
	} else {
		// The sync on opening replaces this message before the screen
		// first shows it: that sync says it again (finishSync).
		m.syncWarn = why
	}
	if m.status != "" {
		why = m.status + "; " + why
	}
	m.setError(why)
}

// syncOffered reports whether the home screen offers to set up sync: the
// folder is not connected, that is, not listed in sync.vaults (what every
// sync and `:sync status` go by), and the setup assistant can run (there is
// a config file to list it in, MISOKU_SYNC=0 did not turn sync off, and
// the folder is not the home folder, the root of a disk or a system
// folder, which setup refuses).
func (m *Model) syncOffered() bool {
	return m.syncVault == nil && !m.syncUnsafe && m.configPath != "" && !gitsync.Disabled()
}

// syncOnOpen is the sync a listed vault gets when Misoku opens it (S4).
func (m *Model) syncOnOpen() tea.Cmd {
	if m.syncVault == nil {
		return nil
	}
	return func() tea.Msg { return SyncNowMsg{} }
}

// requestSync starts the sync the user asked for in the background, one
// at a time (D6). Asked for while the commit after a save runs, it starts
// as soon as that commit ends.
func (m *Model) requestSync() tea.Cmd {
	switch {
	case m.syncVault == nil:
		m.setStatus(m.syncOff)
		return nil
	case m.syncing && m.syncRun != nil && m.syncRun.local:
		m.syncQueued = true
		m.setStatus("Syncing…")
		return nil
	case m.syncing:
		m.setStatus("Already syncing…")
		return nil
	}
	m.setStatus("Syncing…")
	return m.startSync(false, false)
}

// noteSaved records that a note was written to disk, so that the commit
// after a save (autoSync) picks it up. A folder that does not sync keeps
// nothing: it never runs git.
func (m *Model) noteSaved() {
	if m.syncVault == nil {
		return
	}
	m.commitDue, m.lastSave = true, m.clock()
}

// autoSync is what a listed vault does on its own, on each tick of the
// workspace poll (v0.6.0): a sync every interval, counted from the last
// sync, unless the interval is off (S4); otherwise, once saves have paused
// for saveCommitDelay, a local commit of what was saved, with no network
// (S3): the next sync pushes it. Nothing starts while a sync or commit
// runs: the tick after it ends tries again, so no save is left out.
func (m *Model) autoSync() tea.Cmd {
	if m.syncVault == nil || m.syncing {
		return nil
	}
	now := m.clock()
	if every := m.syncVault.Interval; every > 0 && now.Sub(m.syncBase) >= every {
		return m.startSync(false, true)
	}
	if m.commitDue && now.Sub(m.lastSave) >= saveCommitDelay {
		return m.startSync(true, true)
	}
	return nil
}

// startSync starts a sync, or with local only the commit after a save, in
// the background. Either commits everything saved so far.
func (m *Model) startSync(local, auto bool) tea.Cmd {
	m.syncing, m.commitDue = true, false
	if !local {
		m.syncBase = m.clock()
	}
	run := &syncRun{done: make(chan struct{}), local: local}
	m.syncRun = run
	cmd := runSync(m.store.Root(), *m.syncVault, m.syncLock, local)
	return func() tea.Msg {
		msg := cmd().(syncDoneMsg)
		msg.auto = auto
		run.msg = msg
		close(run.done)
		return msg
	}
}

// runSync is the sync itself, run by Bubble Tea outside Update: it checks
// the vault is ready (S2), takes the vault's lock the first time and then
// keeps it for the whole session (S11), syncs (with local, only commits:
// gitsync.CommitLocal), and looks for the conflict copies left on disk
// (S9).
func runSync(dir string, v config.Vault, held *gitsync.Lock, local bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
		defer cancel()
		r := gitsync.New(dir, syncGitTimeout)
		done := syncDoneMsg{lock: held, local: local}
		st := gitsync.Detect(ctx, r)
		ready := st.Status == gitsync.Ready
		if !ready {
			done.paused = st
		} else if done.lock == nil {
			l, ok, err := gitsync.LockVault(ctx, r)
			done.lock, done.err, done.busy = l, err, err == nil && !ok
		}
		if ready && done.lock != nil {
			opts := gitsync.SyncOptions{Device: v.Device}
			if local {
				done.res = gitsync.CommitLocal(ctx, r, done.lock, opts)
			} else {
				done.res = gitsync.SyncVault(ctx, r, done.lock, opts)
			}
		}
		lock := st.Lock
		var locked *gitsync.LockedError
		if errors.As(done.res.Err, &locked) {
			lock = locked.Path
		}
		if lock != "" {
			// Read now, off the Update loop, so the dialog can later remove
			// exactly the file it showed. Unreadable: no offer, only the
			// message that names it.
			if l, ok, err := gitsync.StatLock(lock); err == nil && ok {
				done.leftover = &l
			}
		}
		// The count is for the indicator: a folder that cannot be read
		// shows as no copies, and the sync's own outcome still shows.
		done.copies, _ = gitsync.FindConflictCopies(dir)
		return done
	}
}

// clock is the app's time.
func (m *Model) clock() time.Time {
	if m.now != nil {
		return m.now()
	}
	return time.Now()
}

// finishSync records a finished sync and shows how it went. When it brought
// changes, open notes and the explorer are refreshed through the same check
// the workspace poll runs, which reloads notes without unsaved edits and
// never touches the ones with edits. A sync that started on its own speaks
// only when it made conflict copies or hit a new problem: the indicator
// shows the rest, and the status line stays with what the user was doing.
// A sync asked for during the commit after a save starts now.
func (m *Model) finishSync(msg syncDoneMsg) tea.Cmd {
	m.syncing = false
	if !msg.local {
		m.syncBase = m.clock()
	}
	if msg.lock != nil {
		m.syncLock = msg.lock
	}
	kind, lastErr := m.sync.kind, m.sync.lastErr
	m.recordSync(msg)
	text, isErr := syncSummary(msg)
	news := isErr && (m.sync.kind != kind || m.sync.lastErr != lastErr)
	speak := !msg.auto || news || len(msg.res.Conflicts) > 0
	if msg.res.Updated {
		status, statusErr := m.status, m.statusErr
		m.status, m.statusErr = "", false
		m.checkWorkspace()
		switch {
		case !speak && m.status == "":
			m.status, m.statusErr = status, statusErr
		case m.status != "":
			text += " · " + m.status
			isErr = isErr || m.statusErr
			speak = true
		}
	}
	if speak {
		if m.syncWarn != "" {
			text, isErr = text+" · "+m.syncWarn, true
			m.syncWarn = ""
		}
		m.status, m.statusErr = text, isErr
	}
	if m.syncQueued {
		m.syncQueued = false
		return m.requestSync()
	}
	return nil
}

// recordSync keeps what a finished sync says for the indicator.
func (m *Model) recordSync(msg syncDoneMsg) {
	s := &m.sync
	s.copies = msg.copies
	s.leftover = msg.leftover
	now := m.clock()
	fail := func(kind syncKind, text string) {
		s.kind, s.lastErr, s.errAt = kind, oneLine(text), now
	}
	res := msg.res
	switch {
	case msg.leftover != nil:
		fail(syncPaused, lockedText)
		return
	case msg.err != nil:
		fail(syncError, msg.err.Error())
		return
	case msg.busy:
		s.kind = syncBusy
		return
	case msg.paused.Status == gitsync.Paused && msg.paused.Reason == gitsync.ReasonGitFailed:
		fail(syncError, msg.paused.Message)
		return
	case msg.paused.Status == gitsync.Paused:
		fail(syncPaused, msg.paused.Message)
		return
	}
	s.pending = res.Pending
	s.files = fileWarnings(res.Commit)
	switch res.Outcome {
	case gitsync.Synced:
		// A local commit says nothing about origin: the indicator keeps
		// the last sync's state, with the new pending count.
		switch {
		case !msg.local:
			s.kind, s.lastOK = syncOK, now
		case s.kind == syncBusy:
			// This window took the lock over: no other window syncs now,
			// and nothing is known of origin until the next sync.
			s.kind = syncUnknown
		}
	case gitsync.SyncOffline:
		fail(syncOffline, errText(res.Err))
	case gitsync.SyncUnrelated, gitsync.SyncRewritten:
		fail(syncPaused, errText(res.Err))
	case gitsync.SyncRetry:
		fail(syncRetry, errText(res.Err))
	default: // failed, denied
		fail(syncError, errText(res.Err))
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// syncSummary is the one-line outcome of a sync for the status bar.
func syncSummary(msg syncDoneMsg) (string, bool) {
	switch {
	case msg.leftover != nil:
		return "Sync paused: " + lockedText + " · :sync status", true
	case msg.err != nil:
		return "Sync failed: " + oneLine(msg.err.Error()), true
	case msg.busy:
		return "Sync: another Misoku window syncs this vault", false
	case msg.paused.Status == gitsync.Paused && msg.paused.Reason == gitsync.ReasonGitFailed:
		return "Sync error: " + oneLine(msg.paused.Message), true
	case msg.paused.Status == gitsync.Paused:
		return "Sync paused: " + oneLine(msg.paused.Message), true
	}
	res := msg.res
	why := oneLine(errText(res.Err))
	switch res.Outcome {
	case gitsync.Synced:
		text := "Synced"
		if n := len(res.Conflicts); n > 0 {
			text += fmt.Sprintf(" · %d conflict %s: %s", n, plural(n, "copy", "copies"), strings.Join(res.Conflicts, ", "))
		}
		// The same list the indicator counts and :sync status shows.
		switch files := fileWarnings(res.Commit); {
		case len(files) == 1:
			return text + " · " + files[0], true
		case len(files) > 1:
			return text + fmt.Sprintf(" · %d files do not sync as they are · :sync status", len(files)), true
		}
		return text, false
	case gitsync.SyncOffline:
		return fmt.Sprintf("Sync: offline, %d pending", res.Pending), true
	case gitsync.SyncRetry:
		return "Sync stopped, it tries again next time: " + why, true
	case gitsync.SyncUnrelated, gitsync.SyncRewritten:
		return "Sync paused: " + why, true
	case gitsync.SyncDenied:
		return "Sync error: " + why, true
	}
	return "Sync failed: " + why, true
}

// fileWarnings lists, one line each, the files a commit left out for being
// too big (D7) and those that do not sync as plain files (S15), for
// `:sync status`.
func fileWarnings(c gitsync.CommitResult) []string {
	var lines []string
	for _, b := range c.Skipped {
		lines = append(lines, b.Path+" is too big and stays out of sync")
	}
	for _, n := range c.Notices {
		lines = append(lines, n.String())
	}
	return lines
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// syncIndicator is the sync part of the status bar (S9): empty when the
// folder does not sync, otherwise the state of its last sync.
func (m *Model) syncIndicator() (string, bool) {
	if m.syncVault == nil {
		return "", false
	}
	if m.syncing && (m.syncRun == nil || !m.syncRun.local) {
		return "󰓦 syncing…", false
	}
	s := m.sync
	if s.leftover != nil {
		return "󰓦 sync paused, lock left · :sync status", true
	}
	switch s.kind {
	case syncBusy:
		return "󰓦 other window syncs", false
	case syncPaused:
		return "󰓦 sync paused", true
	case syncError:
		return "󰓦 sync error", true
	case syncRetry:
		return "󰓦 sync will retry", false
	case syncOffline:
		return fmt.Sprintf("󰓦 offline · %d pending", s.pending), true
	case syncOK:
		if n := len(s.copies); n > 0 {
			return fmt.Sprintf("󰓦 %d %s", n, plural(n, "conflict", "conflicts")), true
		}
		text := "󰓦 synced " + ago(m.clock().Sub(s.lastOK))
		if n := len(s.files); n > 0 {
			return text + fmt.Sprintf(" · %d %s to check", n, plural(n, "file", "files")), true
		}
		return text, false
	}
	return "", false
}

// every says how often timed sync runs.
func every(d time.Duration) string {
	if d <= 0 {
		return "off"
	}
	text := strings.TrimSuffix(d.String(), "0s")
	if strings.HasSuffix(text, "h0m") {
		text = strings.TrimSuffix(text, "0m")
	}
	return "every " + text
}

// ago says how long d is, roughly.
func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
}

// lockedText is the leftover index.lock, in plain words.
const lockedText = "a git lock file is in the way (.git/index.lock)"

// lockRemovedMsg brings the removal of a leftover lock back to Update.
type lockRemovedMsg struct{ err error }

// offerLockRemoval shows the detail of `:sync status` (lines) with the
// offer to remove the leftover index.lock l and sync (S13). Only
// `:sync status`, which the user opens on purpose, makes this offer: a
// dialog that opened by itself after a sync would take keys meant for the
// note being written. Misoku never removes such a lock on its own: it may
// belong to a git the user runs in a terminal. Another open dialog (it may
// be asking about unsaved notes) is left alone.
func (m *Model) offerLockRemoval(l gitsync.LeftoverLock, lines []string) {
	if m.confirm != nil || m.saveAs != nil || m.promptBox != nil || m.menuBox != nil {
		return
	}
	lines = append(append(lines, ""),
		"A git lock file (.git/index.lock) is in this vault, so sync cannot go on.",
		"It may belong to a git command still running, for example a commit in a terminal:",
		"remove it only if no git command is running.",
	)
	m.confirm = &confirmDialog{
		title:    "Sync status",
		message:  m.wrapLines(lines),
		buttons:  []string{"Remove the lock and sync", "Cancel"},
		selected: 1, // Enter alone never removes it
		// No letter may choose Remove: only arrows or Tab, then Enter.
		deliberate: true,
		onChoose: func(_ *Model, choice int) tea.Cmd {
			if choice != 0 {
				return nil
			}
			return func() tea.Msg { return lockRemovedMsg{err: gitsync.RemoveLeftoverLock(l)} }
		},
	}
}

// lockRemoved syncs once the confirmed lock is gone, or says why it stayed.
func (m *Model) lockRemoved(msg lockRemovedMsg) tea.Cmd {
	if msg.err != nil {
		m.setError("Lock not removed: " + oneLine(msg.err.Error()))
		return nil
	}
	m.sync.leftover = nil
	if m.syncing {
		// A sync started on its own meanwhile: this one follows it.
		m.syncQueued = true
		m.setStatus("Syncing…")
		return nil
	}
	return m.requestSync()
}

// syncStatusMsg brings what `:sync status` read back to Update.
type syncStatusMsg struct{ info gitsync.Info }

// requestSyncStatus reads the vault's sync status in the background
// (`:sync status`). A folder that does not sync runs no git: it only says
// why.
func (m *Model) requestSyncStatus() tea.Cmd {
	if m.syncVault == nil {
		m.openSyncStatus([]string{m.syncOff})
		return nil
	}
	m.setStatus("Reading the sync status…")
	m.statusKeys = m.keyCount
	dir := m.store.Root()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), statusTimeout)
		defer cancel()
		return syncStatusMsg{info: gitsync.Inspect(ctx, gitsync.New(dir, statusTimeout))}
	}
}

// maxListed caps the conflict copies the status lists one per line.
const maxListed = 10

// showSyncStatus shows what `:sync status` read, with what this window
// knows of its last sync.
func (m *Model) showSyncStatus(msg syncStatusMsg) {
	if m.status == "Reading the sync status…" {
		m.status, m.statusErr = "", false
	}
	info, s := msg.info, m.sync
	state, _ := m.syncIndicator()
	state = strings.TrimSpace(strings.TrimPrefix(state, "󰓦"))
	if state == "" {
		state = "not synced yet in this window"
	}
	lines := []string{
		"State: " + state,
		"Remote: " + orNone(info.Remote),
		"Branch: " + orNone(info.Branch),
		"Timed sync: " + every(m.syncVault.Interval),
		fmt.Sprintf("Pending: %d %s to push", info.Pending, plural(info.Pending, "commit", "commits")),
	}
	if !s.lastOK.IsZero() {
		lines = append(lines, "Last synced: "+ago(m.clock().Sub(s.lastOK)))
	}
	if s.lastErr != "" {
		lines = append(lines, "Last error ("+ago(m.clock().Sub(s.errAt))+"): "+s.lastErr)
	} else {
		lines = append(lines, "Last error: none")
	}
	if info.Err != nil {
		lines = append(lines, "Could not read: "+oneLine(info.Err.Error()))
	}
	for _, w := range s.warnings {
		lines = append(lines, "Config: "+w)
	}
	if n := len(info.Conflicts); n == 0 {
		lines = append(lines, "Conflict copies: none")
	} else {
		lines = append(lines, fmt.Sprintf("Conflict copies (%d), delete each once resolved:", n))
		for i, c := range info.Conflicts {
			if i == maxListed {
				lines = append(lines, fmt.Sprintf("  … and %d more", n-maxListed))
				break
			}
			lines = append(lines, "  "+c)
		}
	}
	if n := len(s.files); n > 0 {
		lines = append(lines, fmt.Sprintf("Files that do not sync as they are (%d):", n))
		for i, f := range s.files {
			if i == maxListed {
				lines = append(lines, fmt.Sprintf("  … and %d more", n-maxListed))
				break
			}
			lines = append(lines, "  "+f)
		}
	}
	if s.leftover != nil && !m.syncing {
		lines = append(lines, "Sync stopped: "+lockedText+".")
		// Keys pressed since `:sync status` was asked for were meant for
		// the note: the offer would take them (Tab, then Enter, removes).
		if m.keyCount == m.statusKeys {
			m.offerLockRemoval(*s.leftover, lines)
			return
		}
		lines = append(lines, "To remove the lock, run :sync status again.")
	}
	m.openSyncStatus(lines)
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// openSyncStatus shows lines in a dialog that only closes. When another
// dialog is open (it may be asking about unsaved notes) it is left alone
// and the first line goes to the status bar instead.
func (m *Model) openSyncStatus(lines []string) {
	if m.confirm != nil || m.saveAs != nil || m.promptBox != nil || m.menuBox != nil {
		m.setStatus(lines[0])
		return
	}
	m.confirm = &confirmDialog{
		title:    "Sync status",
		message:  m.wrapLines(lines),
		buttons:  []string{"Close"},
		onChoose: func(*Model, int) tea.Cmd { return nil },
	}
}

// wrapLines wraps long lines (an error, a deep path) to fit a dialog: the
// detail is the point.
func (m *Model) wrapLines(lines []string) []string {
	room := max(m.width-12, 20)
	var wrapped []string
	for _, l := range lines {
		r := []rune(l)
		for len(r) > room {
			wrapped = append(wrapped, string(r[:room]))
			r = append([]rune("    "), r[room:]...)
		}
		wrapped = append(wrapped, string(r))
	}
	return wrapped
}

// oneLine keeps git's multi-line messages on the single status row.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// ReleaseSyncLock lets go of the vault's sync lock, so another Misoku window
// can take over syncing it (S11). The window keeps no way back to it: a sync
// asked for afterwards fails instead of running without the lock. Meant for
// when the window is done with the vault; SyncOnQuit releases it itself.
func (m Model) ReleaseSyncLock() {
	m.syncLock.Unlock() // safe on nil: this window never took it
}
