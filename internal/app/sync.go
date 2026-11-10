package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
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
// ":sync" (core.SyncMsg) do; see requestSync for when sync has stopped.
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
	// signIn is ready to sign in to GitHub when the access to origin was
	// denied and signing in can fix it; nil otherwise.
	signIn *gitsync.Setup
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
	stop     syncStop              // why the last sync stopped, when the sync key answers it
	signIn   *gitsync.Setup        // the sign-in that may fix stopAccess (syncDoneMsg.signIn)
	// What the last sync that brought notes brought, for `:sync status`.
	arrivedHead string
	arrived     []string
	arrivedAt   time.Time
	// The line of the bar that takes the parts' room, long and compact
	// (barNotice): the last arrivals, or a stop the sync key answers.
	noticeLong, noticeCompact string
	// A stop's notice also has what the bar must always show of it, the key
	// included, and the shortest way to say that. Empty for any other notice.
	noticeStop, noticeStopTight string
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

// requestSync is the sync key and `:sync`. While the last sync's stop is one a
// sync cannot get past, it answers that stop instead; only someone asking on
// purpose gets here with a stop known, so no dialog opens on its own: the sync
// on opening has no last sync, and automatic syncs start with startSync.
func (m *Model) requestSync() tea.Cmd {
	if !m.syncing {
		switch {
		case m.sync.leftover != nil:
			return m.askLockRemoval()
		case m.sync.stop == stopAccess && m.sync.signIn != nil:
			m.offerSignIn()
			return nil
		case m.sync.stop != stopNone:
			m.explainStop()
			return nil
		}
	}
	return m.askSync()
}

// askSync starts a sync in the background, one at a time (D6). Asked for while
// the commit after a save runs, it starts as soon as that commit ends. It opens
// no dialog.
func (m *Model) askSync() tea.Cmd {
	switch {
	case m.syncVault == nil:
		m.setStatus(m.syncOff)
		return nil
	case m.syncing && m.syncRun != nil && m.syncRun.local:
		m.syncQueued = true
		m.showProgress("Syncing…")
		return nil
	case m.syncing:
		m.setStatus("Already syncing…")
		return nil
	}
	m.showProgress("Syncing…")
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
		m.tutorDefer("sync_run", "timed")
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
		m.tutorDefer("sync_indicator", "syncing")
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
		if done.res.Outcome == gitsync.SyncDenied {
			done.signIn = signInFor(ctx, r, dir)
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
// only when it brought notes, made conflict copies or hit a new problem: the
// indicator shows the rest, and the status line stays with what the user was
// doing. It never opens a dialog, which would take keys meant for the note
// being written. A sync asked for during the commit after a save starts now.
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
	m.tutorCSyncIndicator(kind)
	if !msg.local && msg.res.Outcome == gitsync.Synced {
		m.tutorDefer("sync_run", "ok")
	}
	if m.syncOpen && !msg.local {
		m.syncOpen = false
		m.tutorDefer("sync_run", "on-open")
	}
	text, isErr := syncSummary(msg, m.syncKey())
	summary := text
	arrived := arrivedNotes(msg.res)
	news := isErr && (m.sync.kind != kind || m.sync.lastErr != lastErr)
	speak := !msg.auto || news || len(msg.res.Conflicts) > 0 || len(arrived) > 0
	if msg.res.Updated {
		status, statusErr := m.status, m.statusErr
		m.clearStatus()
		m.checkWorkspace()
		switch {
		case !speak && m.status == "":
			// Put back what was showing: it is already in :messages.
			m.status, m.statusErr = status, statusErr
		case m.status != "":
			// The sync's line says it: :messages lists it once, there.
			m.unsayLast()
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
		alone := text == summary // nothing else joined the line
		text = withArrivals(text, msg.res)
		if compact, tight := stopCompact(msg, m.syncKey()); compact != "" && alone {
			m.sync.noticeStop, m.sync.noticeStopTight = compact, tight
			if len(arrived) > 0 {
				compact += " · " + arrivalCompact(msg.res)
			}
			m.sync.noticeLong, m.sync.noticeCompact = text, compact
		}
		// A row of `:messages` is cut at the dialog's width, so the names the
		// bar left out go there one per row, just before the line itself; a
		// huge list counts the rest, so it does not flush what `:messages` held.
		if len(arrived) > maxBarNames {
			for _, row := range appendListed(nil, arrived) {
				m.say(row, false)
			}
		}
		m.say(text, isErr)
	}
	if m.syncQueued {
		m.syncQueued = false
		return m.askSync()
	}
	return nil
}

// recordSync keeps what a finished sync says for the indicator.
func (m *Model) recordSync(msg syncDoneMsg) {
	s := &m.sync
	s.copies = msg.copies
	s.leftover = msg.leftover
	// A local commit says nothing about origin, so an origin-side stop it did not
	// hit stays. A stop in the vault itself ended if the commit went through.
	stop := stopOf(msg)
	ended := msg.local && msg.leftover == nil && msg.res.Outcome == gitsync.Synced &&
		(s.stop == stopDetached || s.stop == stopInProgress)
	if !msg.local || stop != stopNone || ended {
		s.stop, s.signIn = stop, msg.signIn
	}
	now := m.clock()
	if arrived := arrivedNotes(msg.res); len(arrived) > 0 {
		m.tutorDefer("sync_arrival", strings.Join(arrived, ", "))
		s.arrivedHead, s.arrived, s.arrivedAt = arrivalHead(msg.res), arrived, now
		s.noticeLong = arrivalLine(s.arrivedHead, arrived)
		s.noticeCompact = arrivalCompact(msg.res)
		s.noticeStop, s.noticeStopTight = "", ""
	}
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
		case s.kind == syncBusy, ended:
			// This window took the lock over, or the stop ended: nothing holds
			// the vault now, and nothing is known of origin until the next sync.
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
func syncSummary(msg syncDoneMsg, key string) (string, bool) {
	if words, ok := stopWords[stopOf(msg)]; ok {
		return "Sync paused: " + words.bar + " · " + key, true
	}
	switch {
	case msg.leftover != nil:
		return "Sync paused: " + lockedText + " · " + key, true
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
		return fmt.Sprintf("Sync: offline, %d pending · %s retries", res.Pending, key), true
	case gitsync.SyncRetry:
		return "Sync stopped, it tries again next time: " + why, true
	case gitsync.SyncDenied:
		if msg.signIn != nil {
			return "Sync paused: GitHub did not let this computer in · " + key + " signs in", true
		}
		return "Sync error: " + why + " · " + key, true
	}
	return "Sync failed: " + why, true
}

// maxBarNames is how many notes that arrived the status bar names.
const maxBarNames = 3

// arrivedNotes is every note a sync brought, new ones first, as it may be shown.
func arrivedNotes(res gitsync.SyncResult) []string {
	names := slices.Concat(res.NewNotes, res.UpdatedNotes)
	for i, name := range names {
		names[i] = displayName(name)
	}
	return names
}

// displayName shows a file name safely: git carries any name another computer
// made, and a line break or an escape sequence in one must not reach the
// terminal. Control characters show as a placeholder.
func displayName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return unicode.ReplacementChar
		}
		return r
	}, name)
}

// fromAnotherComputer ends every count of arrivalHead; the compact form of the
// notice leaves it out.
const fromAnotherComputer = " from another computer"

// arrivalHead counts the notes a sync brought, in words.
func arrivalHead(res gitsync.SyncResult) string {
	added, updated := len(res.NewNotes), len(res.UpdatedNotes)
	switch {
	case added > 0 && updated > 0:
		return fmt.Sprintf("%d new %s, %d updated"+fromAnotherComputer, added, plural(added, "note", "notes"), updated)
	case added > 0:
		return fmt.Sprintf("%d new %s"+fromAnotherComputer, added, plural(added, "note", "notes"))
	}
	return fmt.Sprintf("%d %s updated"+fromAnotherComputer, updated, plural(updated, "note", "notes"))
}

// withArrivals adds to a sync's line the notes it brought. A plain "Synced"
// gives way to them; anything else the line says (a conflict copy, a warning,
// the push that failed after the notes came) stays first, so it is not the
// part a narrow bar cuts. With no note arrived, the line is unchanged.
func withArrivals(summary string, res gitsync.SyncResult) string {
	names := arrivedNotes(res)
	if len(names) == 0 {
		return summary
	}
	line := arrivalLine(arrivalHead(res), names)
	if summary == "Synced" {
		return line
	}
	return summary + " · " + line
}

// arrivalLine is head followed by the names, a long list cut for the bar.
func arrivalLine(head string, names []string) string {
	if n := len(names); n > maxBarNames {
		return head + ": " + strings.Join(names[:maxBarNames], ", ") + fmt.Sprintf(" ... +%d more · :sync status", n-maxBarNames)
	}
	return head + ": " + strings.Join(names, ", ")
}

// arrivalCompact is the notice without "from another computer".
func arrivalCompact(res gitsync.SyncResult) string {
	return arrivalLine(strings.TrimSuffix(arrivalHead(res), fromAnotherComputer), arrivedNotes(res))
}

// barNotice returns the compact form of the notice when the bar's message is
// that notice and nothing else: a line that also says something else (a
// conflict copy, a warning, a reload) reads differently, so it draws as any
// other message.
func (m *Model) barNotice() (compact string, ok bool) {
	return m.sync.noticeCompact, m.status != "" && m.status == m.sync.noticeLong
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
	text, answered, alert := m.syncStateText()
	if answered {
		text += " · " + m.syncKey()
	}
	return text, alert
}

// syncStateText is the state of the last sync in words, and whether the sync
// key answers it: a stop the key answers says so on the bar, and any other says
// only what it is.
func (m *Model) syncStateText() (text string, answered, alert bool) {
	if m.syncVault == nil {
		return "", false, false
	}
	if m.syncing && (m.syncRun == nil || !m.syncRun.local) {
		return "󰓦 syncing…", false, false
	}
	s := m.sync
	if s.leftover != nil {
		return "󰓦 sync paused, lock left", true, true
	}
	switch s.kind {
	case syncBusy:
		return "󰓦 other window syncs", false, false
	case syncPaused:
		return "󰓦 sync paused", s.stop != stopNone, true
	case syncError:
		return "󰓦 sync error", s.stop != stopNone, true
	case syncRetry:
		return "󰓦 sync will retry", false, false
	case syncOffline:
		return fmt.Sprintf("󰓦 offline · %d pending", s.pending), true, true
	case syncOK:
		if n := len(s.copies); n > 0 {
			return fmt.Sprintf("󰓦 %d %s", n, plural(n, "conflict", "conflicts")), false, true
		}
		text = "󰓦 synced " + ago(m.clock().Sub(s.lastOK))
		if n := len(s.files); n > 0 {
			return text + fmt.Sprintf(" · %d %s to check", n, plural(n, "file", "files")), false, true
		}
		return text, false, false
	}
	return "", false, false
}

var (
	// homeRestHint is the command after the count of a long list of arrivals.
	homeRestHint = regexp.MustCompile(`( \.\.\. \+\d+ more) · :sync status`)
	// homeKeyHint is what syncKey says with no key bound, and what follows it.
	homeKeyHint = regexp.MustCompile(` · :sync(?: retries| signs in| opens it)?( · |$)`)
)

// barText is text as the bar draws it. The start screen has no command line and
// typing `:` there runs letters as shortcuts, so it leaves out the sync lines'
// commands; `:messages` keeps them, as it is read where they work.
func (m *Model) barText(text string) string {
	if m.viewMode != viewModeDashboard {
		return text
	}
	text = homeRestHint.ReplaceAllString(text, "$1")
	return homeKeyHint.ReplaceAllString(text, "$1")
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
// offer to remove the leftover index.lock l and sync (S13). Only `:sync
// status` and the sync key, which the user presses on purpose, make this offer: a
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
	m.tutorDefer("sync_stop_dialog", "")
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
		if errors.Is(msg.err, gitsync.ErrLockChanged) {
			// What was shown is no longer there: the sync key syncs next, and
			// offers the lock as that sync finds it.
			m.sync.leftover = nil
		}
		return nil
	}
	m.sync.leftover = nil
	return m.syncAfterFix()
}

// syncAfterFix syncs once the user removed what stopped the last sync.
func (m *Model) syncAfterFix() tea.Cmd {
	if m.syncing {
		// A sync started on its own meanwhile: this one follows it.
		m.syncQueued = true
		m.showProgress("Syncing…")
		return nil
	}
	return m.askSync()
}

// syncStop is why the last sync stopped, for a stop that syncing again cannot
// get past and the sync key answers with a dialog. The leftover lock has its
// own field (syncState.leftover); every other stop is stopNone, and the key
// syncs.
type syncStop int

const (
	stopNone syncStop = iota
	stopAccess
	stopRewritten
	stopUnrelated
	stopInProgress
	stopDetached
)

// stopOf reads the stop from what the sync reported, never from its text.
func stopOf(msg syncDoneMsg) syncStop {
	switch {
	case msg.leftover != nil, msg.err != nil, msg.busy:
		return stopNone
	case msg.paused.Status == gitsync.Paused:
		switch msg.paused.Reason {
		case gitsync.ReasonInProgress:
			return stopInProgress
		case gitsync.ReasonDetachedHead:
			return stopDetached
		}
		return stopNone
	}
	switch msg.res.Outcome {
	case gitsync.SyncDenied:
		return stopAccess
	case gitsync.SyncRewritten:
		return stopRewritten
	case gitsync.SyncUnrelated:
		return stopUnrelated
	}
	return stopNone
}

// stopCompact is a stop's line for a narrow bar, keeping the key, and the
// shortest line that still names it; empty for a line that is not a stop the
// sync key answers, which draws as any message.
func stopCompact(msg syncDoneMsg, key string) (compact, tight string) {
	stop := stopOf(msg)
	_, plain := stopWords[stop]
	switch {
	case msg.leftover != nil, plain:
		return "Sync paused · " + key, "Sync paused · " + key
	case stop == stopAccess && msg.signIn != nil:
		return "Sync paused · " + key + " signs in", "Sync paused · " + key
	case stop == stopAccess:
		return "Sync error · " + key, "Sync error · " + key
	case msg.err == nil && !msg.busy && msg.paused.Status != gitsync.Paused && msg.res.Outcome == gitsync.SyncOffline:
		return "Sync offline · " + key + " retries", "Sync offline · " + key
	}
	return "", ""
}

// stopWords says the stops no button fixes in plain words, naming no command.
// gitsync's messages keep the commands: the terminal and `:sync status` use
// them. No button joins the histories: after a force-push that could upload
// again something deleted on purpose.
var stopWords = map[syncStop]struct {
	bar string
	why []string
}{
	stopRewritten: {"this vault and origin no longer match", []string{
		"The history in origin was rewritten, so this vault and origin no longer match.",
		"Joining them could bring back something deleted on purpose, so Misoku does not offer it.",
	}},
	stopUnrelated: {"this vault and origin share no history", []string{
		"This vault and origin share no history: they look like two separate sets of notes.",
		"Joining them could bring back something deleted on purpose, so Misoku does not offer it.",
	}},
	stopInProgress: {"an unfinished merge or rebase is waiting in this vault", []string{
		"A merge, rebase or similar operation started in this vault is not finished.",
		"Misoku does not finish or cancel it: it may hold choices only you can make.",
	}},
	stopDetached: {"this vault is not on a branch", []string{
		"This vault is not on a branch right now, so what you write has nowhere to go in origin.",
		"Misoku does not switch branches: the notes on your screen would change.",
	}},
}

// syncKey names the key that syncs, as the user has it: "space s", "Ctrl+Y".
func (m *Model) syncKey() string {
	key := m.keys.Hint(keymap.SyncNow)
	if key == "" {
		return ":sync"
	}
	if rest, ok := strings.CutPrefix(key, "SPC "); ok {
		return "space " + rest
	}
	return key
}

// askLockRemoval is the sync key on a leftover lock: the dialog of `:sync
// status`, with its guard against keys typed meanwhile. A lock gone by now
// needs no dialog, so the check runs first, off the Update loop.
func (m *Model) askLockRemoval() tea.Cmd {
	path := m.sync.leftover.Path
	read := m.requestSyncStatus()
	return func() tea.Msg {
		if _, there, err := gitsync.StatLock(path); err == nil && !there {
			return lockRemovedMsg{}
		}
		msg := read().(syncStatusMsg)
		msg.fromKey = true
		return msg
	}
}

// explainStop says why sync stopped and changes nothing. Check again is a
// normal sync, so a problem the user fixed clears.
func (m *Model) explainStop() {
	title, why := "Sync is paused", stopWords[m.sync.stop].why
	if m.sync.stop == stopAccess {
		title = "Sync cannot get in"
		why = []string{"Origin did not let this computer in.", m.sync.lastErr}
	}
	lines := append(slices.Clone(why), "Nothing was changed, and your notes are safe here.", "Once it is sorted out, choose Check again.")
	m.tutorDefer("sync_stop_dialog", "")
	m.stopDialog(title, lines, []string{"Check again", "Close"}, 1, func(m *Model) tea.Cmd { return m.askSync() })
}

// offerSignIn offers the setup assistant's sign-in, outside the screen. It
// starts on Cancel: Enter alone launches nothing.
func (m *Model) offerSignIn() {
	s := m.sync.signIn
	lines := []string{
		"GitHub did not let this computer in, so sync stopped. Your notes are safe here.",
		"Sign in: gh shows a one-time code and opens your browser. gh keeps the access; Misoku never sees it.",
		"Sync runs once you are back.",
	}
	m.tutorDefer("sync_auth_offer", "")
	m.stopDialog("Sign in to GitHub", lines, []string{"Sign in", "Cancel"}, 1, func(m *Model) tea.Cmd {
		return m.execOutside(s.SignInCommand(), func(error) tea.Msg {
			// gh said how it went on the terminal. This runs on the Update
			// loop, so what asks gh comes back as a command.
			return syncSignedInMsg{s: s}
		})
	})
}

// stopDialog asks about a stop. No letter chooses: the buttons of one dialog
// share an initial, and a second press of the sync key is not a choice. Another
// open dialog (it may be asking about unsaved notes) is left alone.
func (m *Model) stopDialog(title string, lines, buttons []string, selected int, onGo func(*Model) tea.Cmd) {
	if m.confirm != nil || m.saveAs != nil || m.promptBox != nil || m.menuBox != nil {
		m.setStatus(lines[0])
		return
	}
	m.confirm = &confirmDialog{
		title:      title,
		message:    m.wrapLines(lines),
		buttons:    buttons,
		selected:   selected,
		deliberate: true,
		onChoose: func(m *Model, choice int) tea.Cmd {
			if choice != 0 {
				return nil
			}
			return onGo(m)
		},
	}
}

// signInFor is the sign-in that can fix a denied sync, or nil: only a GitHub
// HTTPS origin with gh available. The origin is read here and only compared.
func signInFor(ctx context.Context, r *gitsync.Runner, dir string) *gitsync.Setup {
	out, err := r.Run(ctx, "remote", "get-url", "origin")
	if err != nil || !gitsync.GitHubHTTPS(string(out)) {
		return nil
	}
	s := gitsync.NewSetup(gitsync.SetupOptions{Dir: dir, Timeout: setupTimeout})
	if !s.FindGH() {
		return nil
	}
	return s
}

// syncSignedInMsg comes twice: when gh's sign-in, run outside the screen,
// ended, and with checked once gh was asked how it went (FinishSignIn).
type syncSignedInMsg struct {
	s       *gitsync.Setup
	checked bool
	err     error
}

// syncSignedIn asks gh, in the background, how the sign-in went, then syncs. If
// it did not work, everything stays as it was and the key offers it again.
func (m *Model) syncSignedIn(msg syncSignedInMsg) tea.Cmd {
	if !msg.checked {
		m.setStatus("Checking the sign-in…")
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), setupStepLimit)
			defer cancel()
			return syncSignedInMsg{checked: true, err: msg.s.FinishSignIn(ctx)}
		}
	}
	if msg.err != nil {
		m.setError("Sync still paused: " + oneLine(msg.err.Error()) + " · " + m.syncKey())
		return nil
	}
	return m.syncAfterFix()
}

// syncStatusMsg brings what `:sync status` read back to Update.
type syncStatusMsg struct {
	info    gitsync.Info
	fromKey bool // asked for by the sync key, on a leftover lock
}

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
		m.clearStatus()
	}
	if msg.fromKey && m.keyCount != m.statusKeys {
		// Keys pressed since the sync key were meant for the note: any dialog
		// would take them.
		m.setStatus("Sync paused: " + lockedText + " · " + m.syncKey() + " opens it")
		return
	}
	info, s := msg.info, m.sync
	state, _, _ := m.syncStateText()
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
		lines = appendListed(lines, info.Conflicts)
	}
	if n := len(s.files); n > 0 {
		lines = append(lines, fmt.Sprintf("Files that do not sync as they are (%d):", n))
		lines = appendListed(lines, s.files)
	}
	if len(s.arrived) > 0 {
		lines = append(lines, s.arrivedHead+" ("+ago(m.clock().Sub(s.arrivedAt))+"):")
		lines = appendListed(lines, s.arrived)
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

// appendListed adds items to lines one per line, at most maxListed of them.
func appendListed(lines, items []string) []string {
	for i, item := range items {
		if i == maxListed {
			return append(lines, fmt.Sprintf("  … and %d more", len(items)-maxListed))
		}
		lines = append(lines, "  "+item)
	}
	return lines
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
	m.tutorCSyncStatus()
	m.confirm = &confirmDialog{
		title:    "Sync status",
		message:  m.wrapLines(lines),
		buttons:  []string{"Close"},
		onChoose: func(*Model, int) tea.Cmd { return nil },
	}
}

// wrapLines wraps long lines (an error, a deep path) to fit a dialog: the
// detail is the point. Lines break between words; only a word longer than
// a row (a deep path) is broken inside.
func (m *Model) wrapLines(lines []string) []string {
	room := max(m.width-12, 20)
	var wrapped []string
	for _, l := range lines {
		wrapped = append(wrapped, strings.Split(ansi.Wrap(l, room, ""), "\n")...)
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
