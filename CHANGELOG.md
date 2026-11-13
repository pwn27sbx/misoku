# Changelog

The changes of each release, newest first. Earlier releases are on the
[releases page](https://github.com/pwn27sbx/misoku/releases); the binary's
version comes from the tag, so nothing else in the tree carries it.

## [0.34.0] - 2026-11-13

### Added

- Misoku speaks English and Spanish: menus, dialogs, the status bar, help, the command palette, the slash menu, the sync setup, `misoku --help` and every subcommand. With nothing chosen it follows the system's language (`LANGUAGE`, `LC_ALL`, `LC_MESSAGES`, the macOS or Windows language, then `LANG`), and any other language starts it in English.
- `:settings` has a **Language** row: **Auto (system)**, **English** or **Español**. The choice applies at once, without a restart, and is saved as `ui.language` in `config.json`; Auto removes the key.
- The tutorial exists in English, with the same four routes, lessons and steps as the Spanish one, and Misoku plays the one in the interface language. The English tutorial is installed in `tutor-en`, next to the Spanish `tutor`; each keeps its own notes and progress, and `misoku tutor --reset` starts over only the current language's tutorial.

### Changed

- When a list item, a task, a numbered item or a quote wraps in the editor, its continuation rows start under the item's text instead of at the left edge (a hanging indent, like Vim's `breakindent`); a nested item continues under its own text. The indent is display only: the note's text, copying, search, undo, the cursor and the mouse behave as before. It applies in the editor, the split and the Markdown INSERT shows in the reading view, and a marker wider than half the text width wraps as before.
- The slash menu inserts its starter text in the interface language (`## Sin empezar`, `? Pregunta 1`, `Ene: 12` in Spanish), and its items also answer to their Spanish names.
- The Spanish tutorial quotes the Spanish interface (`/Título 1`, `⊕ añadir pregunta`, `Guardar todo`, the `:settings` rows) and fixes steps that could not pass or said something else: B05 now deletes a `bórrame` line, B22 explains that a capital letter makes search case-sensitive, A12 drafts `[[Mochila]]`, and E04 lists the eight `:settings` rows.
- A player whose system resolves to English and who played the Spanish tutorial now gets the English one; the Spanish notes and progress stay in `tutor` and come back by choosing **Español**.
- An already installed Spanish tutorial keeps its old lesson text, because installing never overwrites a note: I13, for example, still says «Hecho» while its step now asks for «Terminado», and blocks inserted before the update keep their English starter text. Run `misoku tutor --reset` in Spanish to get the new text; it moves only the Spanish tutorial to the Trash.

## [0.33.1] - 2026-11-10

### Changed

- In the reading view's NORMAL, `x` does nothing over a rendered block, since the character it would delete is not on screen; press `i` to edit the Markdown. Elsewhere `x` is unchanged.

## [0.33.0] - 2026-11-10

### Changed

- In the reading view (`space p`), NORMAL keeps every block rendered, including the one under the cursor; Markdown shows only while typing (INSERT) or selecting. The cursor's block carries the cursor-line band, and `j`/`k` move block by block.
- In the reading view's NORMAL, `dd`, `yy`, `o`, `O`, `p` and `P` act on the whole block, and new lines start a block of their own; `>>` and `<<` nest or un-nest a list item. Each is one undo step.

### Fixed

- Leaving INSERT in the reading view of a long note draws the block with its new text at once.

## [0.32.0] - 2026-11-10

### Changed

- The reading view (`space p`) now draws the note exactly as the preview does, full width, and is editable: the block under the cursor shows its Markdown and every other block stays rendered, like Obsidian's Live Preview. Tables, callouts, code, math, pictures, embeds, boards, calendars, timelines, charts, forms, flows, columns, properties and cover all appear as in the preview.
- Everything clickable in the preview works in the reading view (links, checkboxes, folds, calendar, board, timeline, chart, form and table controls), and so do the hover `+` and `⠿` handles and dragging blocks.
- Creating a note, a new entry or a block from the block menu stays in the reading view.

### Fixed

- A note with backlinks or mentions no longer loses its pictures in the preview.
- Dragging a folded section moves the whole section, including its blank lines.

## [0.31.2] - 2026-11-10

### Fixed

- The tutorial now says which link keys open a new tab, teaches `ctrl+o` to go back, and mentions that `- []` plus a space completes itself.

## [0.31.1] - 2026-11-10

### Fixed

- A selection no longer disappears behind the cursor-line band: the band is off while text is selected, as in Neovim, and the selection takes its colour from the active theme's `one_bg2`.

## [0.31.0] - 2026-11-10

### Changed

- The preview shows a heading as its text alone: no level icon and no number by default. Numbers are now opt-in (`ui.heading_numbers` or `:settings`), and `ui.heading_icons` still draws icons for the levels it names.

### Fixed

- The continuation lines of a long heading align with its first line and no longer exceed the pane.

## [0.30.0] - 2026-11-10

### Added

- `ctrl+o` (`go_back`) returns to the previous note of the tab, at the line the cursor was on.

### Changed

- `Enter` on a `[[link]]` and a click on a link in the preview open the note in a new tab. `gd` keeps opening it in the same tab.

## [0.29.0] - 2026-11-10

### Added

- Typing a space after `- []` completes it into the standard task `- [ ] `. Links such as `- [](url)`, code blocks and pasted text are left alone.

## [0.28.1] - 2026-11-10

### Fixed

- Folder editor: in an empty folder, `i`, `I`, `a` and `A` now start a new line like `o`, instead of doing nothing and losing what was typed next.

## [0.28.0] - 2026-11-09

### Added

- The tutorial now has four routes (Básico, Intermedio, Avanzado, Experto) with 97 lessons made of steps: every form a lesson teaches is a step, and Misoku checks the note's text, a file, a folder, a setting or an editor action. The home screen shows one entry per route (`Learn · Básico 3/26`), `l` opens the first unfinished one, and `misoku tutor --route I` opens a route directly.
- Each route's lessons live in their own folder of the tutorial vault (`1 Básico/`, `2 Intermedio/`, `3 Avanzado/`, `4 Experto/`), so the explorer shows them in learning order instead of one long list.
- Lessons credit the actions that leave no trace in a note: moving, switching tabs, opening the finder, the palette, the graph, the history, settings, themes, sync and more, only once the action really happened.
- The statusline follows the lesson: `B03 · Paso 2/4 · <step>`, `✓ Paso 2`, `✓ B03 superado · siguiente: <lesson>` and `🏆 Ruta Básico completa`, with a hint for the common mistake. Progress is stored in `.misoku-tutor.json`, so actions that leave no trace in a note count too, and a failed write is reported.
- `misoku tutor` resumes the last route played at its first unfinished step, and the first-run dialog's `Ya tengo experiencia` offers the Intermedio route.
- The tutorial vault becomes a git repository with one initial commit of the lessons when git is installed, for the `:history` lessons; without git those steps say so and count as skipped. The tutorial also ships a small map image.

### Changed

- The tutorial's seven levels are replaced by the routes. The old level notes are moved to the trash once, with a notice; run `misoku tutor --reset` to start the new tutorial from scratch.

## [0.27.0] - 2026-11-09

### Added

- `H` and `L` switch to the previous and next tab from the editor's NORMAL mode, as `[` and `]` already did. With the preview focused they keep shifting its calendars.
- Search inside a note: `/` searches from the cursor, `n` and `N` repeat it forward and backward, wrapping around the note with a status message; the text is literal and case-insensitive unless it has an upper-case letter, a missing match is `E486: Pattern not found`, and it works in the editor and the reading view.
- A hint in the statusline while you type in a table cell (`Tab next cell · / add row or column`), so adding a column no longer has to be guessed.
- Link suggestions understand headings and blocks: after `[[`, `#` lists the headings of the note, `Note#` the headings of another one, and `^`/`Note#^` their block ids, each inserted as `[[#Heading]]`, `[[Note#Heading]]`, `[[#^id]]` or `[[Note#^id]]`. The link picker and the new inline list share one source of suggestions.
- An inline list of link suggestions, on by default: typing `[[` opens a small list under the cursor while the text stays in the note. `↑`/`↓` move, `Enter` or `Tab` completes, `Esc` closes the list and leaves the text, and clicking an item inserts it.
- `ui.link_suggestions` (`inline` or `finder`), also in `:settings` as **Link suggestions**, chooses between the inline list and the big picker.

### Changed

- `space e` (`focus_explorer`) now toggles the explorer like LazyVim: it opens and focuses it, and closes it when it already has the focus. With no note open, or on the home screen, it keeps opening it.
- A table realigns as you type in a cell: the text uses the cell's padding first, so the closing pipes stay put while it fits; when it no longer fits, the whole column widens at once, and deleting gives the padding back. One `u` still undoes the whole run of typing.
- `[[` now opens the inline list instead of the note picker; the picker is one setting away. In the link picker a leading `#` means headings, so tags there are filtered with `tag:`.

### Fixed

- Clicking the box of a to-do line in the preview toggles it, as the help always said: the box itself is the target, so clicking the text does not toggle, and the change is one `u` undoes.
- `:e name` and `:edit name` open that note, resolved as a `[[link]]` written in the open note would be (a path such as `Projects/AWS` works); a note that does not exist yet starts unsaved with that name. Bare `:e`, `:enew` and `:new` still open an empty tab.

## [0.26.1] - 2026-11-07

### Fixed

- Closing the terminal or a `SIGTERM` no longer loses unsaved text: modified tabs are saved to a private recovery file outside the vault, and the next start offers Restore, Discard or Later. Restored text stays unsaved until you save it, and that save goes through the normal conflict check.

## [0.26.0] - 2026-11-07

### Security

- A note ID can no longer leave the workspace: absolute paths, Windows drive paths and anything that climbs out with `..` are refused, so a link cannot read, overwrite or delete a file outside the vault.
- Links that the system might run now ask first: a dialog that defaults to Cancel guards other URL schemes, paths outside the vault and executable file names. `http`, `https`, `mailto` and ordinary files in the vault open as before.

### Fixed

- Renaming a note whose sub-page folder name is already taken is refused with nothing moved, and a failed folder rename puts the note back, so a rename no longer stops halfway.

### Changed

- Releases are refused until CI is green on the tagged commit.
- Tests no longer touch the real tutorial or Trash folders on Windows.

## [0.25.0] - 2026-11-06

### Added

- Unlinked mentions: the preview lists the notes that mention the open note without linking to it, with a button that converts those mentions into wiki links.
- Templates: `/plantilla` inserts a note from the `templates_dir` folder (default `Plantillas`) at the cursor, replacing `{{fecha}}`/`{{date}}` and `{{titulo}}`/`{{title}}` and merging its front matter into the note's as one undoable edit; template notes are left out of tags and status filters.
- Inline `#tags` are highlighted with the theme's cyan accent in the editor, the reading view and the preview; tags inside code, links, URLs, HTML comments (also across lines) and the front matter are left alone.
- Finder filters `status:`, `tag:`/`#tag` (nested tags included) and `folder:` (quotes for spaces), combined with the search words, also in the link picker; `:tags` lists every tag with its note count and opens the finder on the one picked; `:status [value]` opens the finder on a note status.
- Note status glyphs: a note's `status:` front matter field (`active`, `onhold`, `completed`, `dropped`) shows as `●` `◐` `✓` `✗` in place of its icon in the explorer and in the home screen's recent notes, updated when the note is saved.
- Daily note: `space o d` and `:daily` open or start today's `Diario/YYYY-MM-DD.md`, from the home screen too; the folder is the new `daily_dir` setting.
- Properties table at the top of the preview: front-matter fields as a key/value table, scalar values editable with a click (one undoable edit, the key keeps its spelling), lists read-only.
- `:history` lists the git commits that changed the open note, renames included, previews the selected version and restores it into the editor as one undoable change, without writing to disk until you save.

### Changed

- The reading view (`space p`) draws a checked task (`- [x]`) muted and struck through, as the preview does; the line under the cursor stays raw.

### Fixed

- Setting a cover on a note whose front matter ends with `...` (or uses Windows line endings) now adds it to that block instead of creating a second one.

## [0.24.0] - 2026-11-06

### Changed

- **A dimmed backdrop behind dialogs.** While a dialog, the welcome dialog, the finder, the command palette, the theme
  picker, a menu, a prompt, the date picker or the media browser is open, everything behind it turns grey on the plain
  background (none with `ui.transparent`), and pictures behind it are hidden, so the box is the only coloured thing on
  screen. Closing it brings the colours back. The leader guide and the `/` menu are hints next to what you type and
  leave the screen as it is.

## [0.23.0] - 2026-11-06

### Changed

- **A new welcome dialog on first run.** The `MISOKU` banner, a cat and a one-line tagline sit above a vertical menu:
  `Soy nuevo` (the tutorial, with its level count), `Ya tengo experiencia` and `Ahora no`, each with a line saying
  what it does. `j`/`k` or the arrows move, `Enter` chooses, `n`, `e` and `a` choose directly and `Esc` is `Ahora no`;
  no other key answers it, and a click on an option still chooses it. A terminal too short for the banner gets a
  compact version with the small cat. The answers do what they did, and are remembered once as before.

## [0.22.0] - 2026-11-06

### Changed

- **`space p` and `space r` swap roles.** `space p` is now the reading view: the rendered note alone at full width,
  every Markdown marker hidden (the cursor's line too) and read-only, so keys that edit only say how to edit, while
  moving, searching, copying and following links still work. `space r` is now the editor view: the Markdown editor
  alone at full width, with line numbers. Each key goes back to the split, and either switches straight from the
  other view. There is no longer a key that only hides the side preview. The actions are renamed `reading_view` and
  `editor_view`; in config.json the old names keep working: `reading_mode` binds the reading view and
  `toggle_preview` the editor view (the new name wins when both are given).
- **A blank row above and below the note** in the editor, the preview and the full-width views (`Space p`, `Space r`), so the text no
  longer touches the tab bar or the statusline. It is one row, narrower than the side margins, takes the pane's
  background (none with `ui.transparent`), and a body shorter than eight rows gives it back to the text. Clicks,
  scrolling and the terminal cursor follow the shifted rows.

## [0.21.0] - 2026-11-06

### Added

- **Ctrl+← and Ctrl+→ change pane**, like Ctrl+H and Ctrl+L, for anyone who does not use h j k l. On macOS
  they switch desktops unless that Mission Control shortcut is turned off, so Ctrl+H and Ctrl+L stay the sure
  choice there.
- **A first-run dialog**: the first time Misoku opens a folder it found on its own, a floating dialog with a cat
  asks whether this is your first time. `Soy nuevo` starts the tutorial, `Ya tengo experiencia` and `Ahora no`
  close it (Esc means the same), Ctrl+C quits without recording anything, and any answer is remembered as
  `learn_offered` in `config.json`, so it asks once. `--dir`, `misoku <folder>` and `misoku tutor` never offer it.
- **`Learn` on the start screen**, under Graph and on `l`, counting the levels passed (`Learn 3/7`, with a check
  once all seven do). It opens the tutorial without leaving Misoku, and returning gives the terminal's mouse back
  and reads the progress again.
- **The tutorial keeps what you pass**: a passed level is saved at once, so quitting with unsaved tabs keeps it,
  and `misoku tutor` resumes at the first lesson you did not pass (`00 Empieza aquí` at the start and once every
  level passes). The index says Ctrl+C returns to your vault.

## [0.20.0] - 2026-11-05

### Changed

- **The folder editor works like Oil.nvim**: `Enter` while typing only opens a new line below; nothing touches the
  disk until you confirm. With edits pending, `Enter` in normal mode, `=`, `:w` or `ctrl+s` (also while typing)
  shows "CONFIRM FILE SYSTEM ACTIONS" with every planned action; Yes applies them all, No returns to the lines with
  the edits kept. After applying, the editor stays open on the same folder, re-read from disk and sorted, with the
  cursor on the entry the line made, in the mode it was in. With nothing pending, `Enter` enters a folder or opens a
  note. This replaces 0.19.3's `Enter` that created new lines at once.
- **Deleting by erasing a line**: in the folder editor, `Backspace` at the start of a line joins it into the line
  above and `Delete` at its end joins the line below, as in Vim; the entry of a line joined away is deleted on
  apply, like `dd`, to the Trash.

### Fixed

- **Empty folders are listed**: a folder with nothing inside, such as one just made in the folder editor, now shows
  in the explorer and the folder editor, also after restarting. Folders holding only other files (code, pictures)
  stay hidden. Typing an existing empty folder again is refused instead of planned twice.

## [0.19.3] - 2026-11-05

### Changed

- **`Enter` while typing in the folder editor goes to a new line**: it never opens a note or enters a folder. On a
  new line with a name it creates the note or folder on disk at once, with no confirm, since creating loses nothing
  (a trailing `/` makes a folder, `a/b` the nested path), and opens an empty line below in insert, as in Vim. On an
  existing line it opens the line below; a rename stays pending for `=`, `:w` or leaving, with their confirm. On an
  empty new line it ends typing, as `Esc`. In normal mode `Enter` still enters a folder or opens a note.

## [0.19.2] - 2026-11-05

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
