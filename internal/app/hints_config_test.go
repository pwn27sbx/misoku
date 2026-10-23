package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
)

func TestWithKeysReachesTheEditorHint(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"find_note": {"ctrl+g"}}))
	h.keys(" x")
	if v := h.view(); !strings.Contains(v, "Ctrl+G  search") {
		t.Fatalf("the editor hint should name the configured find key:\n%s", v)
	}
}
