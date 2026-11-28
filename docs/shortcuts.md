# Shortcuts & Command Palette

Every shortcut can be changed in `config.json`:

| OS | Config path |
| --- | --- |
| Linux | `~/.config/misoku/config.json` |
| macOS | `~/Library/Application Support/misoku/config.json` |
| Windows | `%APPDATA%\misoku\config.json` |

Listing an action in `"keys"` replaces all of its bindings; an empty list unbinds it:

```json
{
  "keys": {
    "reading_view": ["space p", "space v"],
    "focus_explorer": [],
    "close_tab": ["space q", "ctrl+w"]
  }
}
```

Write keys the way the terminal names them (`ctrl+b`, `tab`, `shift+tab`, `G`, `\\`), or as a leader sequence (`space e`, `space o c`). Ctrl and Alt shortcuts of `quit`, `save`, `new_note`, `find_note`, and `toggle_explorer` also work while typing; all other shortcuts only fire outside INSERT mode.

Press `Space` and pause 400 ms to open the leader guide (NOTE, VIEW, GO). Misspelled actions, malformed keys, or duplicate bindings are reported in the status bar on startup (`:messages` lists them again). Legacy names `reading_mode` (`reading_view`) and `toggle_preview` (`editor_view`) remain supported.

Vim editing keys (`hjkl`, `w`, `dd`, …) are not configurable.

## Command Palette (`?`)

`?` (or `space h`, `space ?`) opens the searchable command palette built from your active `keys` config, every `:` command, and fixed keys for editor, preview, graph, and mouse. Type to fuzzy-filter, `↑`/`↓` to move, `Enter` to run, `Esc` to close.

## Undo & In-Note Search (NORMAL mode)

| Key | What it does |
| --- | --- |
| `u` | Undo the last change (up to 200 changes or 8 MB per tab) |
| `Ctrl+R` | Redo |
| `/` | Open literal search prompt (smart-case, wraps around note) |
| `n` / `N` | Next / previous match |

## General Actions

| Action | Default keys | What it does |
| --- | --- | --- |
| `new_note` | `ctrl+n`, `space n`, `space o n` | New note (unsaved tab) |
| `save` | `ctrl+s`, `space w` | Save (new notes: pick folder) |
| `find_note` | `ctrl+p`, `\`, `space f`, `space o f`, `space o g` | Find note (opens in current tab; `Ctrl+T` new tab) |
| `toggle_explorer` | `ctrl+b` | Toggle explorer |
| `focus_explorer` | `space e` | Focus / close explorer |
| `next_pane` / `prev_pane` | `tab` / `shift+tab` | Cycle panes |
| `pane_left` / `pane_right` | `ctrl+h`, `ctrl+left` / `ctrl+l`, `ctrl+right` | Move focus left / right |
| `next_tab` / `prev_tab` | `]`, `L`, `space tab` / `[`, `H`, `space shift+tab` | Switch tabs |
| `go_back` | `ctrl+o` | Jump back to previous note/line in tab |
| `close_tab` | `space x` | Close tab |
| `reading_view` | `space p` | Full-width editable Reading view (toggles back to split) |
| `editor_view` | `space r` | Full-width Markdown Editor view (toggles back to split) |
| `export_pdf` | `space P` | Export open note to PDF (`:export`) |
| `themes` | `space t` | Theme picker |
| `graph` | `space g` | Local link graph (`:graph`) |
| `global_graph` | `space G` | Global vault graph (`:graph all`) |
| `home` | `space d` | Home screen |
| `daily_note` | `space o d` | Open today's daily note (`:daily`) |
| `sync_now` | `space s` | Sync vault with git (`:sync`) |
| `paste_image` | `space i`, `space o p` | Paste clipboard image into `assets/` (`ctrl+v` in INSERT) |
| `toggle_checkbox` | `space o c` | Toggle `- [ ]` / `- [x]` on line or visual selection |
| `remove_checkbox` | `space o C` | Remove checkbox prefix |
| `next_link` / `prev_link` | `space o ]` / `space o [` | Jump to next / previous `[[link]]` |
| `help` | `?`, `space h`, `space ?` | Command palette & help |
| `quit` | `ctrl+c` | Quit (asks to save) |

> **macOS note:** `ctrl+left` and `ctrl+right` switch desktops in Mission Control by default; use `ctrl+h` and `ctrl+l` unless Mission Control shortcuts are disabled.

## Explorer Actions

| Action | Default keys | What it does |
| --- | --- | --- |
| `explorer.up` / `explorer.down` | `up`, `k` / `down`, `j` | Move selection |
| `explorer.top` / `explorer.bottom` | `g`, `home` / `G`, `end` | Jump to top / bottom |
| `explorer.page_down` / `explorer.page_up` | `ctrl+d`, `pgdown` / `ctrl+u`, `pgup` | Half-page scroll |
| `explorer.open` | `enter` | Open note / expand folder |
| `explorer.preview` | `p` | Preview note without leaving explorer |
| `explorer.expand` / `explorer.collapse` | `l`, `right` / `h`, `left` | Expand / collapse or go to parent |
| `explorer.toggle_fold` / `explorer.collapse_all` | `z` / `W` | Fold / collapse all |
| `explorer.new_note` / `explorer.new_sub_note` | `n`, `%` / `s` | New note/folder (trailing `/` for folder) / sub-note |
| `explorer.rename` / `explorer.delete` | `r`, `e` / `d`, `x`, `delete` | Rename / delete (asks first) |
| `explorer.find` | `\`, `f` | Find note |
| `explorer.edit_folder` | `m` | Edit folder like Oil.nvim |
| `explorer.insert` | `i`, `I`, `a`, `A`, `o`, `O` | Open folder editor directly in edit action |
| `explorer.file_manager` | `M` | Open vault in Neovim `mini.files` (if installed) |

## Rich reading view

`:rich` or `Space V` draws the note as a page: real fonts, maths and Mermaid
diagrams as pictures. Without Kitty graphics the text reading view stays.

| Key | What it does |
| --- | --- |
| `j` / `k`, `d` / `u`, `g` / `G` | Scroll one row, half a page, to the top/bottom |
| `/`, `n` / `N` | Search the page, next/previous match |
| `o` | Open the table of contents (outline) and jump to a heading |
| `f` then a letter | Follow a link hint |
| `H` / `L` | Scroll a wide code block or table sideways |
| `+` / `=`, `-` | Zoom the page in or out (1.15× steps) |
| `enter` | Zoom the selected diagram (or the first one on screen) to the pane |
| click | Zoom a diagram; on any other row open the text view at that line |
| `y` | Copy the selection, or the visible page |
| `esc` | Leave a zoom, a search or a selection; then back to the text view at the line on screen |

While a diagram is zoomed, `esc`, `enter`, `i`, `q` or a click return to the page
at the row you left; every other key keeps working as usual.

## Graph Keys

- **Local graph (`:graph` / `space g`):** `h`/`l` parent/child, `j`/`k` siblings, `enter` open, `-`/`+` zoom, `0` re-centre, `esc` previous centre.
- **Global graph (`:graph all` / `space G`):** `h`/`j`/`k`/`l` nearest note, `n`/`N` linked neighbors, `enter` open/create, `-`/`+` zoom, `0` centre, `/` filter, `o` hide/show orphans, `u` hide/show unresolved links, `esc` clear/back.
