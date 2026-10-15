package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// QuitSyncLimit is how long Misoku may take, once its screen has closed,
// to push what was saved (S14).
const QuitSyncLimit = 10 * time.Second

// syncRun is one background sync. Every copy of the Model shares it, so
// the window can still wait for it once Bubble Tea, which stops before the
// sync ends, no longer delivers its result (S14).
type syncRun struct {
	done chan struct{} // closed when the sync has finished
	msg  syncDoneMsg   // what it said; read only once done is closed
}

// SyncOnQuit is what a synced vault gets once Misoku's screen has closed
// (S14): what was saved is committed and pushed, within limit, and the
// sync lock is let go. It writes to w that it is pushing, then the outcome
// in one line; it never asks anything. When the push fails or runs out of
// time, the commit stays and the next sync pushes it. Cancelling ctx (Ctrl+C)
// stops git as the time limit does: only what git finished stays (S13).
//
// A folder that does not sync, and a window that never held the lock
// (another window syncs the vault), run no git and print nothing. A sync
// still running is waited for, never joined by a second git; only what was
// saved after it started is left for the final push.
func (m Model) SyncOnQuit(ctx context.Context, w io.Writer, limit time.Duration) {
	if m.syncVault == nil {
		return
	}
	var run *syncRun
	if m.syncing {
		run = m.syncRun
	}
	if m.syncLock == nil && run == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	// Writes to the terminal are best effort: there is nothing to do
	// about one that fails.
	say := func(text string) { _, _ = fmt.Fprintln(w, text) }
	say("Pushing changes…")

	lock := m.syncLock
	if run != nil {
		select {
		case <-run.done:
			if lock == nil {
				lock = run.msg.lock
			}
		case <-ctx.Done():
			// That sync is not stopped: its git finishes, or dies with
			// Misoku, and the next sync recovers (S13). Its lock is kept
			// until Misoku exits, so no other window runs git beside it.
			say("Not pushed, the next sync tries again: a sync still running did not finish (" + stopReason(ctx, limit) + ")")
			return
		}
	}
	if lock == nil {
		say("Not pushed: another Misoku window syncs this vault")
		return
	}
	defer lock.Unlock()
	say(pushOnQuit(ctx, m.store.Root(), *m.syncVault, lock, limit))
}

// pushOnQuit commits and pushes the vault, holding lock, and says how it
// went in one line.
func pushOnQuit(ctx context.Context, dir string, v config.Vault, lock *gitsync.Lock, limit time.Duration) string {
	r := gitsync.New(dir, limit)
	// The vault may have changed since the last sync (a merge started in a
	// terminal): the same checks as every sync come first (S2).
	st := gitsync.Detect(ctx, r)
	var res gitsync.SyncResult
	if st.Status == gitsync.Ready {
		res = gitsync.PushVault(ctx, r, lock, gitsync.SyncOptions{Device: v.Device})
	}
	switch {
	case ctx.Err() != nil:
		return "Not pushed, the next sync tries again: " + stopReason(ctx, limit)
	case st.Status != gitsync.Ready:
		return "Not pushed, sync is paused: " + oneLine(st.Message)
	case res.Outcome == gitsync.Synced && res.Pushed:
		return "Changes pushed."
	case res.Outcome == gitsync.Synced:
		return "Nothing new to push."
	case res.Outcome == gitsync.SyncRewritten:
		return "Not pushed, sync is paused: " + oneLine(errText(res.Err))
	}
	return "Not pushed, the next sync tries again: " + oneLine(errText(res.Err))
}

// stopReason says why ctx ended: Ctrl+C or the time limit.
func stopReason(ctx context.Context, limit time.Duration) string {
	if errors.Is(ctx.Err(), context.Canceled) {
		return "cancelled"
	}
	return "no answer within " + limit.String()
}
