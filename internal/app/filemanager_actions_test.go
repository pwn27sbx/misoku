package app

import (
	"slices"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// The log's absolute paths become vault IDs, Windows-style ones included;
// renames and moves inside the vault are moves, everything else is not.
func TestFileManagerActionMovesMapsTheLogToVaultIDs(t *testing.T) {
	same := func(p string) string { return p }
	for _, tc := range []struct {
		name, root, log string
		want            []core.Move
	}{
		{"rename", "/v", `{"action":"rename","from":"/v/a.md","to":"/v/b.md"}`, []core.Move{{From: "a.md", To: "b.md"}}},
		{"move", "/v", `{"action":"move","from":"/v/a.md","to":"/v/sub/a.md"}`, []core.Move{{From: "a.md", To: "sub/a.md"}}},
		{"windows", `C:\Users\me\vault`, `{"action":"rename","from":"C:/Users/me/vault/x/a.md","to":"C:/Users/me/vault/x/b.md"}`, []core.Move{{From: "x/a.md", To: "x/b.md"}}},
		{"escaped slashes", "/v", `{"action":"move","from":"\/v\/a.md","to":"\/v\/d\/a.md"}`, []core.Move{{From: "a.md", To: "d/a.md"}}},
		{"move out of the vault", "/v", `{"action":"move","from":"/v/a.md","to":"/elsewhere/a.md"}`, nil},
		{"move into the vault", "/v", `{"action":"move","from":"/elsewhere/a.md","to":"/v/a.md"}`, nil},
		{"sibling folder", "/v", `{"action":"rename","from":"/vault2/a.md","to":"/vault2/b.md"}`, nil},
		{"delete", "/v", `{"action":"delete","from":"/v/a.md","to":"/trash/a.md"}`, nil},
		{"copy", "/v", `{"action":"copy","from":"/v/a.md","to":"/v/b.md"}`, nil},
		{"garbage", "/v", "not json\n{\"action\":\"rename\",\"from\":\"/v/a.md\",\"to\":\"/v/b.md\"}", []core.Move{{From: "a.md", To: "b.md"}}},
	} {
		if got := fileManagerActionMoves(tc.root, []byte(tc.log), same); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// Moves made one after another settle into one move per note, from where
// it was to where it ended, sub-notes and folder contents included.
func TestSettleMovesFollowsEachNoteThroughEveryMove(t *testing.T) {
	before := []core.Page{{ID: "a.md"}, {ID: "a/sub.md"}, {ID: "f/n.md"}, {ID: "keep.md"}}
	raw := []core.Move{{From: "a.md", To: "b.md"}, {From: "b.md", To: "c.md"}, {From: "f", To: "g"}, {From: "g/n.md", To: "h/n.md"}}
	want := []core.Move{{From: "a.md", To: "c.md"}, {From: "a/sub.md", To: "c/sub.md"}, {From: "f/n.md", To: "h/n.md"}}
	if got := settleMoves(before, raw); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
