package app

import (
	"errors"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/keymap"
)

// miniFilesLua opens Neovim's mini.files on $MISOKU_DIR and quits Neovim
// when the explorer closes. The folder travels in the environment, never in
// this source, so spaces, quotes and backslashes in it are safe.
//
// With LazyVim, mini.files is a lazy-loaded plugin while mini.nvim's own copy
// is already on the runtime path: lazy.load runs first so require returns
// the copy LazyVim configured. Without mini.files at all Neovim exits with
// code 3. The options given to open apply to this session only.
const miniFilesLua = "lua " +
	"pcall(function() require('lazy').load({ plugins = { 'mini.files' } }) end); " +
	"local ok, MF = pcall(require, 'mini.files'); " +
	"if not ok then vim.cmd('cquit 3') return end; " +
	"vim.api.nvim_create_autocmd('User', { pattern = 'MiniFilesExplorerClose', " +
	"callback = function() vim.schedule(function() vim.cmd('qa') end) end }); " +
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
}

// openFileManager hands the screen to Neovim's mini.files on the vault
// folder. MISOKU_ACTIONS names an empty file the session may write its
// actions to; it is removed when Neovim ends.
func (m *Model) openFileManager() tea.Cmd {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		m.setError(m.keys.Hint(keymap.ExplorerFileManager) + " needs Neovim (nvim) on PATH")
		return nil
	}
	log, err := os.CreateTemp("", "misoku-actions-*.jsonl")
	if err != nil {
		m.setError("Could not start mini.files: " + err.Error())
		return nil
	}
	_ = log.Close()
	cmd := exec.Command(nvim, "-c", miniFilesLua)
	cmd.Env = append(os.Environ(), "MISOKU_DIR="+m.store.Root(), "MISOKU_ACTIONS="+log.Name())
	// The callback runs once Neovim ends, outside Update: the folder is read
	// there, so Update never waits on the disk.
	store, kept := m.store, append([]core.Page(nil), m.newFolders...)
	return m.execOutside(cmd, func(err error) tea.Msg {
		_ = os.Remove(log.Name())
		return fileManagerDoneMsg{err: err, pages: listTree(store, kept)}
	})
}

// fileManagerDone shows the tree read when Neovim ended, so what changed in
// mini.files shows without waiting for the poll, and counts a changed tree
// as a save for the commit that follows one. Open notes catch up with the
// next workspace check.
func (m *Model) fileManagerDone(msg fileManagerDoneMsg) {
	if treeSignature(msg.pages) != m.treeSig {
		m.noteSaved()
	}
	m.applyScannedTree(msg.pages)
	var code interface{ ExitCode() int }
	switch {
	case msg.err == nil:
	case errors.As(msg.err, &code) && code.ExitCode() == miniFilesMissingExit:
		m.setError("mini.files is not installed in Neovim (add nvim-mini/mini.files, or the LazyVim extra editor.mini-files)")
	default:
		m.setError("Neovim: " + msg.err.Error())
	}
}
