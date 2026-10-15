package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
)

// an https origin with a token that cannot be
// reached (offline) and that refuses (no server: connection refused) never
// shows the token in the status bar or in `:sync status`.
func TestSyncTokenNeverInBarOrStatus(t *testing.T) {
	for _, url := range []string{
		"http://me:s3cretTOKEN@127.0.0.1:1/notes.git",
		"http://x-access-token:s3cretTOKEN@127.0.0.1:1/notes.git",
		"http://127.0.0.1:1/notes.git?token=s3cretTOKEN",
	} {
		_, vault := syncedVault(t)
		gitIn(t, vault, "remote", "set-url", "origin", url)
		writeOutside(t, vault, "new.md", "fresh")
		h := window(t, vault, syncConfig(t, vault))
		h.finish(h.update(app.SyncNowMsg{}))
		if v := h.view(); strings.Contains(v, "s3cret") {
			t.Fatalf("the screen shows the token:\n%s", v)
		}
		wantBar(t, h, "offline · 1 pending")
		v := openSyncStatus(h)
		if strings.Contains(v, "s3cret") {
			t.Fatalf(":sync status shows the token:\n%s", v)
		}
		if !strings.Contains(v, "127.0.0.1:1/notes.git") {
			t.Errorf(":sync status must show the remote:\n%s", v)
		}
	}
}
