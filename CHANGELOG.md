# Changelog

The changes of each release, newest first. Earlier releases are on the
[releases page](https://github.com/pwn27sbx/misoku/releases); the binary's
version comes from the tag, so nothing else in the tree carries it.

## [Unreleased]

### Fixed

- **`↑`/`↓` while typing a name in the folder editor**: they keep the typed name as a pending edit and go on typing
  at the end of the line above or below, as in Vim's insert mode; an empty new line is dropped, and the first and
  last lines stay put.

## [0.19.1] - 2026-11-05

### Changed

- **`Enter` saves in the folder editor**: `Enter` while typing a name commits it and shows
  "CONFIRM FILE SYSTEM ACTIONS" at once; `Enter` or `y` applies. It no longer opens a new line: `o`/`O` add lines.
  In normal mode `Enter` enters the folder under the cursor or opens the note, as in the tree, where `Enter` on a
  folder only expands it. While a name is typed the terminal's own cursor shows, as INSERT's bar, where the next
  character goes.

## [0.19.0] - 2026-11-05

### Changed

- **Vim insert keys start typing from the explorer**: `i`, `I`, `a`, `A`, `o` and `O` in the tree open the folder
  editor on the folder holding the selection, with the cursor on it, and act at once, like LazyVim's mini.files:
  `i`/`a` edit the name, `o`/`O` type a new line below or above. `m` still opens the editor alone. To free them,
  preview moves from `o` to `p` and new sub-note from `a` to `s`.

## [0.18.1] - 2026-11-05

### Changed

- **The folder editor (`m`) types like mini.files**: `Enter` while typing a name commits the line and opens a new
  one below, still typing; `Esc` returns to normal and drops an empty new line. A name ending in `/` is a folder
  and `a/b` the nested path, as with `n`. Leaving with edits (`Esc`, `q` or `:q`) shows the same
  "CONFIRM FILE SYSTEM ACTIONS" as `=`: Yes applies and closes, No drops the edits and closes, Cancel keeps
  editing. It replaces "Discard N changes?".

### Fixed

- **`space r` keeps the page margins**: the full-width live editor now uses the same left and right margins as
  the preview, instead of running edge to edge.

## [0.18.0] - 2026-11-05

### Added

- **Edit a folder like mini.files, natively**: `m` in the explorer opens the selected folder as editable lines.
  New lines create notes and folders, edited names rename, `dd` deletes, `dd` then `p` in another folder (`h`/`l`)
  moves, `u`/`ctrl+r` undo and redo. `=` or `:w` shows "CONFIRM FILE SYSTEM ACTIONS" with every action and the
  links it updates, and applies them in one batch after Yes; tabs and links follow. Leaving with edits asks
  "Discard N changes?". `M` still opens Neovim's mini.files.

- **`misoku tutor`**, a game to learn Markdown and Misoku: a tutorial vault of its own, next to `config.json`,
  with an index and seven levels in Spanish (headings, checkboxes, the `/` block menu, links, links to headings,
  block references and the explorer). Each level is checked live as you edit its note, ignoring code: a passed
  level toasts `✓ Nivel N superado (k/7)`, all of them `🏆 Tutorial completo (7/7)`, and the statusline shows
  `Tutor k/7`. Progress comes from the notes themselves. `misoku tutor --reset` moves the tutorial to the Trash
  and starts over. Without `misoku tutor` nothing changes.

- **Numbered headings in the preview**: headings show their outline number, `1`, `1.1`, `2`, in place of the
  level icon, with skipped levels never giving a `0`. A lone `#` that is the first heading stays the title,
  unnumbered. Folding keeps the numbers; the note's text never changes. On by default: `ui.heading_numbers`, or
  **Heading numbers**, the sixth row of `:settings`, turns it off.

- **The LazyVim companion's Misoku keys**: `space o c` (`toggle_checkbox`) toggles the box of the line, `[ ]` to
  `[x]` and `[x]` or `[-]` back to `[ ]`, and gives a line without one a `- [ ] `; `space o C` (`remove_checkbox`)
  takes the box off (`- [x] task` becomes `- task`). Both work on a VISUAL or V-LINE selection, line by line, keep
  the selection and undo with one `u`. `space o f` and `space o g` find a note, `space o n` starts one and
  `space o p` pastes a picture. In NORMAL mode `Enter` follows the `[[link]]` under the cursor (not inside code;
  a missing note says "No note found for this link"), toggles a checkbox line, and otherwise moves down as before.
  `space o ]` (`next_link`) and `space o [` (`prev_link`) jump to the next and previous `[[link]]`, wrapping and
  skipping code, as the companion's `]o` and `[o`; a note without links says "No links in this note".
  Leader sequences may now take two keys after Space (`"space o c"` in `config.json`); the leader guide lists
  `o +misoku` and shows the group's keys after `space o`, and a sequence that starts another is reported.

- **Links follow a rename**: renaming a note or folder in the explorer asks `Update N links in M notes?` and
  rewrites every link that named it (`[[name]]`, `[[name|alias]]`, `[[name#Heading]]`, embeds and Markdown links),
  so they keep pointing at the same note. `n` keeps the rename and leaves the links. Open tabs get the change
  without losing unsaved edits, and `u` undoes it; a note changed on disk meanwhile is skipped and reported. The
  rewrite counts as a save for sync. A rename with no links asks nothing, as before.

- **mini.files from the explorer**: `M` (`explorer.file_manager`) opens Neovim's mini.files on the vault folder,
  with your Neovim configuration (LazyVim's options too), deletes to mini.files' trash and dot files hidden. Closing
  it (`q`) quits Neovim and Misoku re-reads the folder at once, so what you created, renamed or moved there shows
  right away. Without `nvim` on `PATH`, or without mini.files in Neovim, nothing runs and the status bar says why.

- **New notes and folders by name** in the explorer, like LazyVim's mini.files: `n` opens a name line under the
  selected folder. `test` makes a note, `test/` a folder and `ideas/test` both, after a `CREATE ideas/test.md`
  confirmation (`y` or Enter creates, `n` or Esc cancels). The folders on the path open in the explorer, a new note
  opens in the editor and a new folder is selected. A name that already exists is never overwritten: the note is
  opened, or the folder selected. Names that would leave the workspace (`..`, absolute paths, empty or hidden
  names) are refused. An empty name still starts an Untitled note, as `n` did before.

- **Transparent background**: `ui.transparent` (or **Transparent background** in `:settings`, applied at once)
  leaves the background of the editor, preview, explorer, tab line and statusline to the terminal, so a terminal
  with opacity or blur shows through, like Claude Code. Like Neovim's transparent themes, popups go clear and
  keep only their border, and heading bands are off; the cursor line, selections, statusline blocks and badges
  stay painted. The welcome screen honours it too, and a user theme reloaded from disk stays transparent. The
  terminal's own background colour shows through, so pick a theme that matches it.

### Changed

- **`space r` edits the note**: reading mode is now the editor at full width, drawn like render-markdown.nvim.
  Line numbers hide and every line but the cursor's shows rendered, its Markdown markers hidden (`#`s, bold,
  italic and code delimiters, link targets, boxes as icons, `•` bullets, `▎` quotes); the cursor line stays raw,
  with every Vim key. Code blocks, front matter and tables stay raw. The side preview (`space p`) is unchanged.

### Fixed

- `space i` pastes the clipboard picture again: the key was listed but did nothing.

## [0.17.1] - 2026-11-03

### Changed

- **The editor and the preview keep their colours** side by side, whichever of the two has the focus: watching
  the note take shape while writing no longer greys the other half. The explorer and the graphs still grey the
  panes they leave, and with the preview hidden the editor still reads as inactive without the focus.

## [0.17.0] - 2026-11-03

### Added

- **The cursor shows the mode**, like LazyVim: a block in NORMAL, VISUAL and VISUAL-LINE, a thin bar in INSERT,
  drawn by the terminal itself on the editor cell and gone while the `:` command line has its own. The shape is
  `ui.cursor_shape` (`"auto"`, the default, or `"block"` for the drawn block Misoku always had), and
  `MISOKU_CURSOR_SHAPE` wins over the file for one run. Misoku gives the terminal its cursor back when it exits.
- **Sync says which notes arrived**: after a sync that brought notes, automatic ones included, the status bar
  names them (`2 new notes from another computer: nota.md, otra.md`). A long list is cut there; `:sync status` and
  `:messages` list the first ten of them and count the rest. While that notice shows, the bar gives up its right-side parts to fit
  it, and in a narrow window it reads `2 new notes: nota.md, otra.md`. When no note arrived, nothing changes.
- **`space s` answers a stopped sync**, and the status bar says so: offline retries; a leftover lock opens the dialog
  to remove it; expired GitHub access (HTTPS origin, gh installed) offers the sign-in (Enter alone cancels), then syncs; rewritten or
  unrelated histories, an unfinished merge and a vault not on a branch are explained in plain words, with **Check
  again** and **Close**, changing nothing. No dialog opens on its own, and `:sync status` keeps the technical detail. The bar gives up its
  right-side parts to fit these messages, and in a narrow window reads `Sync offline · space s retries`.

## [0.16.0] - 2026-11-02

### Added

- **Notes that arrived from another computer are named**: after a sync that brings notes, automatic ones
  included, the status bar says so, for example `2 new notes from another computer: nota.md, otra.md`. No dialog
  opens on its own, so nothing takes the keys of the note being written. `:messages` and `:sync status` list the
  first ten arrivals and count the rest.
- **The sync key resolves a stopped sync**, and the bar says what it did: offline, it retries; a leftover lock
  opens the lock removal dialog; expired GitHub access offers the sign-in (Enter alone cancels); a rewritten or
  unrelated history, an unfinished merge or a detached HEAD is explained in plain words and nothing is changed,
  because joining two histories could bring back something deleted on purpose.

### Fixed

- Note names with control characters are shown sanitized in the bar, `:messages` and `:sync status`.
- The sync key stays visible in the bar, and a stop that a later local commit disproves is cleared.
- The test suites clear repository-local git variables, so running them from a git hook never touches the
  repository being committed to.

## [0.15.0] - 2026-11-02

### Added

- **Paste a picture from the clipboard**: `ctrl+v` in INSERT mode, `/` → Media → *Paste image*, or the new
  `paste_image` action (`space i`). The picture is saved next to the note in its `assets/` folder as
  `pasted-YYYY-MM-DD_HH-MM-SS.png` (with `-2`, `-3` … when the name is taken) and the link is inserted on a
  line of its own, so the preview draws it. Text pastes exactly as before; a bracketed paste carrying text never
  reads the clipboard. Needs `wl-clipboard` on Wayland, `xclip` on X11, `pngpaste` or AppleScript on macOS; the
  status bar names what to install when none is there.
- **Sharp pictures in the preview**: in a terminal that speaks the Kitty graphics protocol (Ghostty, kitty,
  WezTerm, and inside herdr, which draws pane graphics itself) a picture is sent once and drawn by the terminal,
  pixel for pixel, and stays sharp while scrolling, folding and resizing. Everywhere else, and for a picture too
  large to send, the preview keeps drawing half blocks.
- **`ui.images`** in `config.json` (`"auto"`, `"kitty"`, `"blocks"`) and **`MISOKU_IMAGES`** for one run, which
  wins over the file.
- **`misoku paste-image [--dir <folder>] [--link]`**: the same paste from a shell, for an editor or a script. It
  prints the path to link, or the whole `![](…)` with `--link`.

### Notes

- **Super+V does not paste pictures.** Omarchy's universal paste sends Shift+Insert in terminals, and Ghostty
  turns that into "paste the clipboard's text", consuming the key before any application sees it — even with
  `keybind = performable:shift+insert=paste_from_clipboard`, because Ghostty considers that action performable
  whenever the clipboard request starts, whether or not it holds text (`src/Surface.zig`). Text keeps pasting
  with Super+V; pictures go through `ctrl+v`, the `/` menu or `space i`.
- A PDF export is unchanged: pictures are embedded, and no terminal graphics reach the file.
