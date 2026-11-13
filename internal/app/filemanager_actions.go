package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// fileManagerAction is one line of the log the Lua writes: mini.files'
// MiniFilesAction* event data.
type fileManagerAction struct {
	Action string `json:"action"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// fileManagerActionMoves turns the renames and moves in a mini.files action
// log into moves between vault IDs, in log order. Only "rename" and "move"
// count, and only with both ends inside root: a delete goes to mini.files'
// trash, and a move out of the vault is a delete too. resolve maps a path
// before it is compared with root (symlinks, see resolveParent); unreadable
// lines are skipped.
func fileManagerActionMoves(root string, log []byte, resolve func(string) string) []core.Move {
	var moves []core.Move
	sc := bufio.NewScanner(bytes.NewReader(log))
	for sc.Scan() {
		var a fileManagerAction
		if json.Unmarshal(sc.Bytes(), &a) != nil || (a.Action != "rename" && a.Action != "move") {
			continue
		}
		from, okFrom := vaultID(root, resolve(a.From))
		to, okTo := vaultID(root, resolve(a.To))
		if okFrom && okTo && from != to {
			moves = append(moves, core.Move{From: from, To: to})
		}
	}
	return moves
}

// vaultID is p's ID in the vault at root, false when p is not inside it.
// Both sides are compared with forward slashes, so a log written on Windows
// (C:/…) maps against a root such as C:\… on every OS.
func vaultID(root, p string) (string, bool) {
	slash := func(s string) string { return path.Clean(strings.ReplaceAll(s, `\`, "/")) }
	id, ok := strings.CutPrefix(slash(p), strings.TrimSuffix(slash(root), "/")+"/")
	return id, ok && id != ""
}

// resolveParent resolves the symlinks in p's folder; p itself may be gone
// (the old name of a rename). A folder it cannot resolve leaves p as it is.
func resolveParent(p string) string {
	if dir, err := filepath.EvalSymlinks(filepath.Dir(p)); err == nil {
		return filepath.Join(dir, filepath.Base(p))
	}
	return p
}

// settleMoves turns moves made one after another into one move per note of
// before, from where it was to where it ended (a rename then a move of the
// same note is one move). Folders and sub-notes follow through ExpandMoves.
func settleMoves(before []core.Page, raw []core.Move) []core.Move {
	at := make([]core.Page, len(before))
	for i, p := range before {
		at[i] = core.Page{ID: p.ID, IsFolder: p.IsFolder}
	}
	for _, mv := range raw {
		next := map[string]string{}
		for _, e := range core.ExpandMoves(at, []core.Move{mv}) {
			next[e.From] = e.To
		}
		for i := range at {
			if to, ok := next[at[i].ID]; ok {
				at[i].ID = to
			}
		}
	}
	var out []core.Move
	for i, p := range before {
		if at[i].ID != p.ID {
			out = append(out, core.Move{From: p.ID, To: at[i].ID})
		}
	}
	return out
}

// readFileManagerMoves reads the action log mini.files left and moves the
// sub-note folder of each renamed or moved note along with it (Q2), so its
// sub-notes stay under it. It runs when Neovim ends, outside Update. A
// missing or unreadable log means no moves: the tree is only read again.
func readFileManagerMoves(store *core.Store, logPath string, before []core.Page) []core.Move {
	log, err := os.ReadFile(logPath)
	if err != nil {
		return nil
	}
	root := store.Root()
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	raw := fileManagerActionMoves(root, log, resolveParent)
	for _, mv := range raw {
		from, to := strings.TrimSuffix(mv.From, ".md"), strings.TrimSuffix(mv.To, ".md")
		if from == mv.From || to == mv.To {
			continue // a folder: its contents moved with it
		}
		src, dst := filepath.Join(root, filepath.FromSlash(from)), filepath.Join(root, filepath.FromSlash(to))
		if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
			continue
		}
		if _, err := os.Lstat(dst); err == nil {
			continue // never over a folder already there
		}
		// Best effort: a folder that cannot follow stays where it is, and
		// the next workspace check reports its open sub-notes as gone.
		_ = os.Rename(src, dst)
	}
	return settleMoves(before, raw)
}

// fileManagerStatus is the status a set of mini.files moves leaves:
// "Renamed to X" for one note, "Moved N notes" for more.
func fileManagerStatus(moves []core.Move) string {
	if len(moves) == 1 {
		return i18n.T(i18n.AppRenamedTo, moves[0].To)
	}
	return i18n.T(i18n.AppMovedNotes, len(moves))
}
