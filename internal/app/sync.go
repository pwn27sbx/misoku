package app

import (
	"context"
	"fmt"
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
)

// SyncNowMsg asks the app to sync the vault now, as the sync_now key and
// ":sync" (core.SyncMsg) do.
// Opening a vault listed in sync.vaults sends one. It is exported so
// embedders and tests can ask for a sync; in a folder that does not sync it
// only says why.
type SyncNowMsg struct{}

// syncDoneMsg brings a finished sync command back to Update.
type syncDoneMsg struct {
	lock   *gitsync.Lock // the vault's lock when this window holds it
	paused string        // why the vault cannot sync now (Detect)
	busy   bool          // another window holds the lock
	err    error         // the lock could not be taken
	res    gitsync.SyncResult
}

// setupSync reads the "sync" section of the config file and decides, once,
// whether this folder syncs. It runs no git: a folder that is not listed,
// or MISOKU_SYNC=0, must never start one.
func (m *Model) setupSync() {
	m.syncOff = "Sync is off: this folder is not listed in sync.vaults"
	if gitsync.Disabled() {
		m.syncOff = "Sync is off (MISOKU_SYNC=0)"
		return
	}
	// An unreadable config has already been reported at startup; it simply
	// leaves sync off, the safe side.
	cfg, err := config.Load(m.configPath)
	if err != nil {
		return
	}
	vaults, warnings := cfg.Sync.Parse()
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
	}
	if m.status != "" {
		why = m.status + "; " + why
	}
	m.setError(why)
}

// syncOnOpen is the sync a listed vault gets when Misoku opens it (S4).
func (m *Model) syncOnOpen() tea.Cmd {
	if m.syncVault == nil {
		return nil
	}
	return func() tea.Msg { return SyncNowMsg{} }
}

// requestSync starts a sync in the background, one at a time (D6).
func (m *Model) requestSync() tea.Cmd {
	switch {
	case m.syncVault == nil:
		m.setStatus(m.syncOff)
		return nil
	case m.syncing:
		m.setStatus("Already syncing…")
		return nil
	}
	m.syncing = true
	m.setStatus("Syncing…")
	return runSync(m.store.Root(), *m.syncVault, m.syncLock)
}

// runSync is the sync itself, run by Bubble Tea outside Update: it checks
// the vault is ready (S2), takes the vault's lock the first time and then
// keeps it for the whole session (S11), and syncs.
func runSync(dir string, v config.Vault, held *gitsync.Lock) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
		defer cancel()
		r := gitsync.New(dir, syncGitTimeout)
		done := syncDoneMsg{lock: held}
		if st := gitsync.Detect(ctx, r); st.Status != gitsync.Ready {
			done.paused = st.Message
			return done
		}
		if done.lock == nil {
			l, ok, err := gitsync.LockVault(ctx, r)
			if err != nil || !ok {
				done.err, done.busy = err, err == nil
				return done
			}
			done.lock = l
		}
		done.res = gitsync.SyncVault(ctx, r, done.lock, gitsync.SyncOptions{Device: v.Device})
		return done
	}
}

// finishSync records a finished sync and shows how it went. When it brought
// changes, open notes and the explorer are refreshed through the same check
// the workspace poll runs, which reloads notes without unsaved edits and
// never touches the ones with edits.
func (m *Model) finishSync(msg syncDoneMsg) {
	m.syncing = false
	if msg.lock != nil {
		m.syncLock = msg.lock
	}
	text, isErr := syncSummary(msg)
	if msg.res.Updated {
		m.status, m.statusErr = "", false
		m.checkWorkspace()
		if m.status != "" {
			text += " · " + m.status
			isErr = isErr || m.statusErr
		}
	}
	m.status, m.statusErr = text, isErr
}

// syncSummary is the one-line outcome of a sync for the status bar; the full
// status and error handling come with the sync indicator.
func syncSummary(msg syncDoneMsg) (string, bool) {
	switch {
	case msg.err != nil:
		return "Sync failed: " + oneLine(msg.err.Error()), true
	case msg.busy:
		return "Sync: another Misoku window syncs this vault", false
	case msg.paused != "":
		return "Sync paused: " + oneLine(msg.paused), true
	}
	res := msg.res
	switch res.Outcome {
	case gitsync.Synced:
		if n := len(res.Conflicts); n > 0 {
			return fmt.Sprintf("Synced · %d conflict copies: %s", n, strings.Join(res.Conflicts, ", ")), false
		}
		return "Synced", false
	case gitsync.SyncOffline:
		return fmt.Sprintf("Sync: offline, %d pending", res.Pending), true
	}
	text := fmt.Sprintf("Sync %s, %d pending", res.Outcome, res.Pending)
	if res.Err != nil {
		text += ": " + oneLine(res.Err.Error())
	}
	return text, true
}

// oneLine keeps git's multi-line messages on the single status row.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// ReleaseSyncLock lets go of the vault's sync lock, so another Misoku window
// can take over syncing it (S11). The window keeps no way back to it: a sync
// asked for afterwards fails instead of running without the lock. Meant for
// when the window is done with the vault; nothing calls it yet.
func (m Model) ReleaseSyncLock() {
	m.syncLock.Unlock() // safe on nil: this window never took it
}
