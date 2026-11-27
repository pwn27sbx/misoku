<div align="center">

# Misoku

A fast, keyboard-first Markdown notebook for your terminal. Plain `.md` files, optional git sync, zero lock-in.

<img src="docs/demo.gif" alt="Misoku in a terminal: finding a note, editing with live preview and [[wiki links]], following a link, and switching to reading view" width="900">

</div>

## Why Misoku?

- **Instant terminal workflow:** single static Go binary (`CGO_ENABLED=0`), Vim keys, splits, tabs, async fuzzy finder, and editable live preview (tables, callouts, boards, Mermaid flowcharts, LaTeX Unicode equations, Kitty graphics in Ghostty/kitty/WezTerm) — no Electron.
- **Plain files, your git:** notes stay ordinary `.md` files on disk with `[[wiki links]]`, backlinks, unlinked mentions, Oil-style folder editing (`m`), and optional private git sync that creates conflict copies instead of `<<<<<<<` markers.
- **Safe by default:** atomic saves (with bounded retry on Windows before falling back to in-place writes if another app locks the file), Trash on macOS and Linux (Windows deletes permanently after confirmation), recovery snapshots on terminal close, and a built-in interactive tutorial (`misoku tutor`).

## Install

**Linux & macOS**

```sh
curl -fsSL https://raw.githubusercontent.com/pwn27sbx/misoku/main/install.sh | sh
```

**Windows (PowerShell 5.1 & 7)**

```powershell
irm https://raw.githubusercontent.com/pwn27sbx/misoku/main/install.ps1 | iex
```

> **Installer notes:** Both scripts verify SHA-256 checksums for every download, leave any existing `gh` installation untouched, and abort before installing if a checksum fails. On Linux no root is needed (`~/.local/bin`); on Windows `%VAR%` entries in your user `PATH` are preserved as `REG_EXPAND_SZ`.

**Go** (requires Go `1.26.6+`)

```sh
go install github.com/pwn27sbx/misoku/cmd/misoku@latest
```

**Manual download:** grab `misoku-macos.tar.gz` (universal), `misoku-linux.tar.gz` (x86-64), `misoku-linux-arm64.tar.gz`, or `misoku-windows.zip` from [GitHub Releases](https://github.com/pwn27sbx/misoku/releases/latest) and place `misoku` on your `PATH`.

A [Nerd Font](https://www.nerdfonts.com/) is recommended for icons.

## First Run

1. Run `misoku` in your terminal.
2. Pick or create your notes folder (or pass one directly: `misoku ~/Notes`).
3. Press `?` inside Misoku for the searchable command palette, or `l` on the start screen (or `misoku tutor`) for the interactive tutorial.

## Common Commands

```sh
misoku                        # open default notes folder (notes_dir / $MISOKU_WORKSPACE / cwd)
misoku ~/Notes                # open a specific folder
misoku --theme catppuccin     # pick one of 97 bundled themes (or custom JSON themes)
misoku tutor                  # interactive tutorial (English & Spanish)
misoku export note.md         # export a note to A4 PDF
misoku sync setup             # connect a private GitHub repo for sync
misoku sync init <git-url>    # connect any git remote (SSH, GitLab, Gitea)
misoku paste-image            # save clipboard image under ./assets
```

## Config Paths

| OS | Config file |
| --- | --- |
| Linux | `~/.config/misoku/config.json` |
| macOS | `~/Library/Application Support/misoku/config.json` |
| Windows | `%APPDATA%\misoku\config.json` |

## Highlights

- **Two full-width views + split:** `space p` toggles editable Reading view (live rendered blocks, click checkboxes/boards/tables), `space r` toggles full-width Markdown editor, both toggle back to split view.
- **Fast search & filters:** `ctrl+p` / `f` opens the non-blocking finder with fuzzy filenames, async full-text search, and qualifiers (`status:active tag:go folder:Projects`).
- **Links & graphs:** `[[wiki links]]` with inline completion, heading/block links and embeds, automatic link updates on rename, local graph (`space g`), global vault graph (`space G`), and unlinked mentions.
- **Oil-style folder editing (`m`):** create, rename, move, and delete notes/folders as text with confirmation and automatic link updates.
- **Daily notes & templates:** `space o d` opens today's daily note (`Daily/YYYY-MM-DD.md` in English, `Diario/` in Spanish or existing vaults); `/template` inserts reusable templates with `{{date}}` and `{{title}}`.
- **Transparent terminal theme support:** `"ui": {"transparent": true}` lets terminal opacity and blur (e.g. Ghostty / Omarchy) show through.

## Documentation

- [Shortcuts & Command Palette](docs/shortcuts.md) — configurable `keys`, leader guide, explorer, and graph bindings
- [Configuration, Appearance & Uninstall](docs/configuration.md) — `ui` options, custom themes, images, flowcharts, equations, PDF export, and uninstall steps
- [Git Sync](docs/sync.md) — GitHub setup, custom remotes, conflict copies, and troubleshooting
- [Contributing](CONTRIBUTING.md) — development setup, architecture, and guidelines
- [Changelog](CHANGELOG.md) — release history

## Credits & License

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and [chroma](https://github.com/alecthomas/chroma). Licensed under [MIT](LICENSE).
