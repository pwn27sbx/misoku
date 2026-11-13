package app

import (
	"errors"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/keymap"
)

// miniFilesLua opens Neovim's mini.files on $MISOKU_DIR and quits Neovim
// when the explorer closes. The folder travels in the environment, never in
// this source, so spaces, quotes and backslashes in it are safe.
//
// With LazyVim, mini.files is a lazy-loaded plugin while mini.nvim's own copy
// is already on the runtime path: lazy.load runs first so require returns
// the copy LazyVim configured. Without mini.files at all Neovim exits with
// code 3. The options given to open apply to this session only. Every
// MiniFilesAction* event is appended to $MISOKU_ACTIONS as one JSON line
// ({action, from, to}), read back by readFileManagerMoves.
const miniFilesLua = "lua " +
	"pcall(function() require('lazy').load({ plugins = { 'mini.files' } }) end); " +
	"local ok, MF = pcall(require, 'mini.files'); " +
	"if not ok then vim.cmd('cquit 3') return end; " +
	"vim.api.nvim_create_autocmd('User', { pattern = 'MiniFilesExplorerClose', " +
	"callback = function() vim.schedule(function() vim.cmd('qa') end) end }); " +
	"local log = vim.env.MISOKU_ACTIONS; " +
	"vim.api.nvim_create_autocmd('User', { pattern = { 'MiniFilesActionCreate', 'MiniFilesActionDelete', " +
	"'MiniFilesActionRename', 'MiniFilesActionMove', 'MiniFilesActionCopy' }, callback = function(ev) " +
	"local f = io.open(log, 'a'); if f then f:write(vim.json.encode(ev.data), '\\n'); f:close() end end }); " +
	"MF.open(vim.env.MISOKU_DIR, false, { options = { permanent_delete = false }, " +
	"content = { filter = function(e) return not vim.startswith(e.name, '.') end } })"

// miniFilesMissingExit is the code Neovim ends with when mini.files is not
// installed.
const miniFilesMissingExit = 3

// fileManagerDoneMsg arrives when Neovim ends, with the tree read from
// disk at that moment.
type fileManagerDoneMsg struct {
	err   error
	pages []core.Page
	// moves are the notes mini.files renamed or moved, per note of the
	// list taken before launching; relinks is the rewrite they call for.
	moves   []core.Move
	relinks relinksPlannedMsg
}

// openFileManager hands the screen to Neovim's mini.files on the vault
// folder. MISOKU_ACTIONS names an empty file the session may write its
// actions to; it is removed when Neovim ends.
func (m *Model) openFileManager() tea.Cmd {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		m.setError(i18n.T(i18n.AppNeedsNvim, m.keys.Hint(keymap.ExplorerFileManager)))
		return nil
	}
	log, err := os.CreateTemp("", "misoku-actions-*.jsonl")
	if err != nil {
		m.setError(i18n.T(i18n.AppMiniFilesFailed, err.Error()))
		return nil
	}
	_ = log.Close()
	cmd := exec.Command(nvim, "-c", miniFilesLua)
	cmd.Env = append(os.Environ(), "MISOKU_DIR="+m.store.Root(), "MISOKU_ACTIONS="+log.Name())
	// The callback runs once Neovim ends, outside Update: the folder is read
	// there, so Update never waits on the disk.
	// The log and the notes are read there too, before the tree, so the
	// tree read includes the sub-note folders that followed their notes.
	store := m.store
	before := append([]core.Page(nil), m.notes...)
	return m.execOutside(cmd, func(err error) tea.Msg {
		moves := readFileManagerMoves(store, log.Name(), before)
		_ = os.Remove(log.Name())
		done := fileManagerDoneMsg{err: err, moves: moves}
		if len(moves) > 0 {
			done.relinks = planRelinksIn(store, relinkJob{renamed: fileManagerStatus(moves), before: before, moves: moves})
		}
		done.pages = store.List()
		return done
	})
}

// fileManagerDone shows the tree read when Neovim ended, so what changed in
// mini.files shows without waiting for the poll, and counts a changed tree
// as a save for the commit that follows one. Tabs of renamed and moved
// notes follow them before anything checks the disk (S7), and the links
// that named them are offered for rewriting as after a rename in Misoku.
// Other open notes catch up with the next workspace check.
func (m *Model) fileManagerDone(msg fileManagerDoneMsg) {
	m.followMoves(msg.moves)
	if treeSignature(msg.pages) != m.treeSig {
		m.noteSaved()
	}
	m.applyScannedTree(msg.pages)
	if len(msg.moves) > 0 {
		m.setStatus(msg.relinks.job.renamed)
		m.askRelinks(msg.relinks)
	}
	var code interface{ ExitCode() int }
	switch {
	case msg.err == nil:
	case errors.As(msg.err, &code) && code.ExitCode() == miniFilesMissingExit:
		m.setError(i18n.T(i18n.AppMiniFilesMissing))
	default:
		m.setError(i18n.T(i18n.AppNeovimError, msg.err.Error()))
	}
}
