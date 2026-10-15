package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/gitsync"

	tea "github.com/charmbracelet/bubbletea"
)

// statusBar is the bottom row of the screen.
func statusBar(h *harness) string {
	lines := strings.Split(h.view(), "\n")
	return lines[len(lines)-1]
}

func wantBar(t *testing.T, h *harness, want string) {
	t.Helper()
	if bar := statusBar(h); !strings.Contains(bar, want) {
		t.Fatalf("the status bar must show %q:\n%s", want, bar)
	}
}

// S9: syncing, then synced and how long ago, by the app's clock.
func TestSyncIndicatorShowsSyncingAndHowLongAgo(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	now := time.Date(2026, 10, 14, 15, 30, 0, 0, time.Local)
	h := window(t, vault, syncConfig(t, vault), app.WithClock(func() time.Time { return now }))
	cmd := h.update(app.SyncNowMsg{})
	wantBar(t, h, "syncing")
	h.finish(cmd)
	wantBar(t, h, "synced just now")
	now = now.Add(5 * time.Minute)
	wantBar(t, h, "synced 5m ago")
	now = now.Add(3 * time.Hour)
	wantBar(t, h, "synced 3h ago")
}

// S7, S9: offline, with how many commits wait to be pushed.
func TestSyncIndicatorShowsOfflineWithPending(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	if err := os.Rename(remote, remote+".gone"); err != nil {
		t.Fatal(err)
	}
	writeOutside(t, vault, "new.md", "fresh")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "offline · 1 pending")
}

// S9: conflict copies on disk are counted, whichever sync made them.
func TestSyncIndicatorShowsConflictCopies(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	writeOutside(t, vault, "doc (conflict desk 2026-10-14 1530).md", "theirs")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "1 conflict")
}

func TestSyncIndicatorShowsPaused(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	gitIn(t, vault, "checkout", "-q", "--detach")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "sync paused")
}

func TestSyncIndicatorShowsAnotherWindow(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	cfg := syncConfig(t, vault)
	first := window(t, vault, cfg)
	first.finish(first.update(app.SyncNowMsg{}))
	second := window(t, vault, cfg)
	second.finish(second.update(app.SyncNowMsg{}))
	wantBar(t, second, "other window")
}

// S8: an access error shows as an error with its hint, never a prompt.
func TestSyncAccessErrorShowsItsHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake ssh is a shell script")
	}
	_, vault := syncedVault(t)
	script := filepath.Join(t.TempDir(), "fake-ssh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'Host key verification failed.' >&2\nexit 255\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SSH", "")
	t.Setenv("GIT_SSH_COMMAND", "sh '"+script+"'")
	gitIn(t, vault, "remote", "set-url", "origin", "ssh://git@example.invalid/notes.git")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "sync error")
	wantBar(t, h, "misoku sync init")
}

// openSyncStatus types ":sync status" in the editor, checks it asks for
// the status, and runs that request to the end.
func openSyncStatus(h *harness) string {
	h.t.Helper()
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys(":sync status")
	asked := false
	queue := []tea.Cmd{h.update(tea.KeyMsg{Type: tea.KeyEnter})}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := runCmd(c).(type) {
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case core.SyncStatusMsg:
			asked = true
		}
	}
	if !asked {
		h.t.Fatal(":sync status must ask for the status")
	}
	if cmd := h.update(core.SyncStatusMsg{}); cmd != nil {
		h.finish(cmd)
	}
	return h.view()
}

// dialogText is the text inside the dialog boxes of view, with the border,
// the indentation and the line breaks removed, so a long line the dialog
// wrapped (an error quoting a temporary path, whose length depends on the
// OS) reads as one again.
func dialogText(view string) string {
	var b strings.Builder
	for _, line := range strings.Split(view, "\n") {
		first, last := strings.Index(line, "│"), strings.LastIndex(line, "│")
		if first < 0 || last <= first {
			continue
		}
		b.WriteString(strings.TrimSpace(line[first+len("│") : last]))
	}
	return b.String()
}

// S9 `:sync status`: origin without its credentials, the branch, the last
// error and the conflict copies found on disk now.
func TestSyncStatusShowsTheDetail(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	if err := os.Rename(remote, remote+".gone"); err != nil {
		t.Fatal(err)
	}
	h.finish(h.update(app.SyncNowMsg{}))
	gitIn(t, vault, "remote", "set-url", "origin", "https://me:s3cret@example.invalid/notes.git")
	writeOutside(t, vault, "sub/doc (conflict desk 2026-10-14 1530).md", "theirs")

	v := openSyncStatus(h)
	text := dialogText(v)
	for _, want := range []string{"https://example.invalid/notes.git", "Branch: main", "Last error", "does not appear to be a git repository", "sub/doc (conflict desk 2026-10-14 1530).md"} {
		if !strings.Contains(text, want) {
			t.Errorf(":sync status must show %q:\n%s", want, v)
		}
	}
	// Unwrapped, so a credential split across two rows is caught too.
	if strings.Contains(text, "s3cret") || strings.Contains(v, "s3cret") {
		t.Fatalf(":sync status shows the credentials:\n%s", v)
	}
	h.key(tea.KeyEsc)
	if strings.Contains(h.view(), "sub/doc (conflict") {
		t.Error("Esc closes the status")
	}
}

// D2: in a folder that does not sync, `:sync status` says so and runs no git.
func TestSyncStatusRunsNoGitWhenOff(t *testing.T) {
	_, vault := syncedVault(t)
	cfg := syncConfig(t, t.TempDir())
	log := fakeGit(t, "exit 1")
	h := window(t, vault, cfg)
	if v := openSyncStatus(h); !strings.Contains(v, "Sync is off") {
		t.Fatalf(":sync status must say sync is off:\n%s", v)
	}
	if data, err := os.ReadFile(log); err == nil {
		t.Fatalf("git ran: %s", data)
	}
}

func TestSyncCommandRejectsUnknownArguments(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	if v := openSyncStatus(h); !strings.Contains(v, "Sync status") {
		t.Fatalf(":sync status must open the status:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.keys(":sync bogus")
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		drain(h, cmd)
	}
	wantBar(t, h, "E488")
}

// An unreadable config file must not read as "not listed".
func TestUnreadableConfigSaysSo(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := window(t, vault, cfg)
	if cmd := h.update(app.SyncNowMsg{}); cmd != nil {
		t.Fatal("an unreadable config leaves sync off")
	}
	v := h.view()
	if !strings.Contains(v, "could not be read") || strings.Contains(v, "not listed") {
		t.Fatalf("the reason must be the unreadable config:\n%s", v)
	}
}

// D7: a file left out for being too big is named, not dropped silently.
func TestSyncNamesBigFilesLeftOut(t *testing.T) {
	remote, vault := syncedVault(t)
	old := gitsync.MaxFileSize
	gitsync.MaxFileSize = 10
	t.Cleanup(func() { gitsync.MaxFileSize = old })
	writeOutside(t, vault, "big.png", "eleven byte")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "big.png")
	wantBar(t, h, "out of sync")
	if onRemote(t, remote, "big.png") {
		t.Fatal("the big file must stay out of the commit")
	}
}

func TestSyncHelpListsSyncStatus(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("?")
	if v := h.view(); !strings.Contains(v, ":sync status") {
		t.Fatalf("the help must list :sync status:\n%s", v)
	}
}

// A sync that stopped to try again (another device pushed first, a save
// during the sync) is no error: the indicator must not say so in red.
func TestSyncRetryIsNotShownAsAnError(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	gitDir := gitIn(t, vault, "rev-parse", "--absolute-git-dir")
	// A marker naming a git that still runs (this test) makes the sync wait.
	if err := os.WriteFile(filepath.Join(gitDir, "misoku-sync.running"), []byte(fmt.Sprintf("%d %d %d\n", os.Getpid(), time.Now().UnixNano(), int64(time.Minute))), 0o644); err != nil {
		t.Fatal(err)
	}
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	if bar := statusBar(h); strings.Contains(bar, "sync error") || !strings.Contains(bar, "sync will retry") {
		t.Fatalf("a retry must show as one, not as an error:\n%s", bar)
	}
}
