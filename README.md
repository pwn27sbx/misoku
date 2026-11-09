<div align="center">

# Misoku

### ~ Terminal Markdown Notebook ~

A block-based notebook for your terminal.
Your notes stay plain Markdown files.

<img src="docs/demo.gif" alt="Misoku in a terminal: opening a note from the start screen, moving through its blocks, adding a to-do with a [[wiki link]] that the live preview renders, following the link to the other note, and switching to reading mode" width="900">

</div>

https://github.com/user-attachments/assets/e9c584a7-0d56-4a9c-b9d9-bc7cb9ebbcf8

## First run

All you need is a Terminal: the window where you type commands. To open it:

- **macOS:** press Cmd+Space, type `Terminal` and press Enter.
- **Windows:** open the Start menu, type `PowerShell` and press Enter.
- **Linux:** press Ctrl+Alt+T, or open Terminal from your apps.

1. **Install.** Copy the line for your computer, paste it into the Terminal
   and press Enter.

   macOS and Linux:

   ```sh
   curl -fsSL https://raw.githubusercontent.com/pwn27sbx/misoku/main/install.sh | sh
   ```

   Linux: If it says `curl: command not found`, run `sudo apt install curl` first.

   Windows:

   ```powershell
   irm https://raw.githubusercontent.com/pwn27sbx/misoku/main/install.ps1 | iex
   ```

   On a Mac it may ask for your Mac's password. Type it and press Enter (the
   letters do not show while you type). On Linux it may ask
   `Install git? (needs your password) [Y/n]`: press Enter for yes. If it
   ends with `Close this window and open a new Terminal, then run 'misoku'.`,
   do that.

2. **Open Misoku.** Type `misoku` and press Enter.

3. **Choose where your notes live.** The first time, Misoku greets you with
   a welcome screen:

   ```text
   Welcome to Misoku.

   Where do your notes live?

   › 1  Start my notes
     2  Bring my notes from another computer
   ```

   Move with the arrow keys and press Enter, or press the number. Esc (or
   `q`) leaves without changing anything. In a very small Terminal (under
   60×12) the same choices come as a numbered list you answer by typing a
   number and pressing Enter.

   - **Start my notes:** Misoku offers the folder `~/Notes` (a folder called
     Notes in your home folder). Press Enter to accept it, or change it.
     Misoku creates the folder and opens it.
   - **Bring my notes from another computer:** for notes your other
     computer already syncs with Misoku. Misoku offers a folder for them
     (`~/Notes`, or `~/Notes-2` when that one already has files) and
     explains what will happen, in the same screen. Misoku asks a few
     questions; press Enter to accept each answer it offers. Then you sign
     in to GitHub: a one-time code appears and your browser opens; sign in
     there and enter the code. Pick your notes from the list of your
     repositories, and they appear. Esc at any question goes back to the
     choices and undoes what setup created (a git name and email you
     already saved stay saved). No notes elsewhere after all? Choose
     **I don't have notes on another computer → Start my notes**.
   - **Use this folder:** shown as a third choice only when the Terminal is
     already in a folder with notes (`.md` files). Misoku keeps your notes
     there.

4. **Every time after that,** type `misoku` and press Enter. It opens your
   notes, whatever folder the Terminal is in. Press `?` inside Misoku to see
   every shortcut.

5. **Sync between computers.** While your notes do not sync yet, the home
   screen shows **Sync between computers**. Press `s`, sign in to GitHub the
   same way, and confirm. If git does not know your name and email yet (they
   go in your notes' history), Misoku takes them from your GitHub account;
   you only confirm them. From then on your
   notes sync on their own. On your other computer, install Misoku and
   choose **Bring my notes from another computer**.

You need a GitHub account to sync. Misoku works without one, and without
git: both are needed only to sync.

## Installation & Update

The command in [First run](#first-run) installs Misoku. Run the same command
again to update.

On every system this one command installs two programs side by side:
**misoku**, and **gh**, GitHub's command-line tool, which Misoku signs in to
GitHub with to sync your notes. gh is a pinned version whose checksum is
written in the installer, and every download is checked: a mismatch stops
the install.

- **macOS:** both go to `/usr/local/bin`, so `misoku` works in the same
  Terminal. The installer asks for your Mac's password when it must create
  or write that folder. Without git, it starts Apple's installer for it and
  carries on.
- **Linux:** both go to `~/.local/bin`. When that folder is not on your
  `PATH`, it adds one line to your shell's startup file (bash, zsh or fish);
  then close the window and open a new Terminal. Without git, it offers to
  install it with your distribution's package manager (this needs your
  password).
- **Windows:** both go to `%LOCALAPPDATA%\Programs\misoku`, which is added
  to your user `PATH` (`misoku` works in the same window; other windows see
  it once reopened). Without git, it installs it with
  `winget install --id Git.Git -e --source winget` when winget is there
  (Windows may ask for permission; if it fails, the install still finishes),
  and otherwise links to [git-scm.com](https://git-scm.com/download/win).
  Misoku finds a new git in `%ProgramFiles%\Git\cmd` even before the window
  sees the new `PATH`. Works in Windows PowerShell 5.1 and PowerShell 7.
- On macOS and Linux a `misoku` already on your `PATH` is updated where it
  is. On every system a gh you installed yourself (Homebrew, apt, winget…)
  is left alone and used as it is.
  Running the installer again updates only the gh it installed.
- git is needed only to sync; Misoku works without it.

**Arch Linux (and Omarchy):** from the AUR, `yay -S misoku-bin` (or any
other AUR helper). Update with `yay -Syu`. Available once the AUR package
is published.

**Homebrew (macOS):** `brew install --cask pwn27sbx/tap/misoku`. Update with
`brew upgrade --cask misoku`. The binary is not notarized by Apple, so the
cask removes macOS's quarantine flag from it on install; without that, macOS
would refuse to open it.

Both install only `misoku`. To sync with `:sync setup`, also install gh:
`yay -S github-cli` or `brew install gh`.

**Manual download:** grab `misoku-macos.tar.gz`, `misoku-linux.tar.gz`
(x86-64), `misoku-linux-arm64.tar.gz` or `misoku-windows.zip` from the
[latest release](https://github.com/pwn27sbx/misoku/releases/latest) and put
`misoku` on your `PATH`.

**Go:** `go install github.com/pwn27sbx/misoku/cmd/misoku@latest`

**From source:**

```sh
git clone https://github.com/pwn27sbx/misoku
cd misoku
go build -o ~/.local/bin/misoku ./cmd/misoku
```

Misoku is a single binary with no dependencies. It runs on macOS (Intel and
Apple Silicon), Windows 10/11 and any Linux distribution on x86-64 or arm64
(e.g. a Raspberry Pi with a 64-bit system). On other CPUs, install with Go.
A [Nerd Font](https://www.nerdfonts.com/) is recommended for icons.

## Advanced

Misoku opens the first of these that is set:

1. the folder you give: `misoku ~/notes`, or `misoku --dir ~/notes`;
2. the folder in the `MISOKU_WORKSPACE` environment variable;
3. the folder you chose at the welcome: `notes_dir` in your
   [config](#shortcuts), such as `"notes_dir": "~/Notes"`;
4. the folder the Terminal is in.

The welcome shows only when none of the first three is set, and the
Terminal's folder holds no notes (no `.md` file at its first level, and not
listed in [`sync.vaults`](#configuration)) or is your home folder, the root
of a disk or a system folder. It never shows without a terminal to answer it
(in a script, for example). If the folder in `notes_dir` is gone (an
unplugged disk, say), the welcome says `I can't find your notes at …` instead
of creating it empty.

Once a folder is open, the first time Misoku opens one it found on its own
(`notes_dir`, `$MISOKU_WORKSPACE` or the Terminal's folder) a welcome dialog
asks whether this is your first time and offers the tutorial (see
[Learn Misoku](#learn-misoku)). Any of its three answers is remembered as
`learn_offered` in the [config](#shortcuts), so it asks once; a folder you
give (`--dir`, `misoku <folder>`) never sees it, and `misoku tutor` never
offers it either.

```sh
misoku ~/notes                # open a specific folder (same as --dir ~/notes)
misoku                        # no folder: $MISOKU_WORKSPACE, else notes_dir in your config, else this folder
misoku --theme <name>         # pick a theme for this session
misoku --list-themes
misoku --themes-dir <path>    # custom themes folder or single .json file
misoku --version              # print the version
misoku --help                 # every command and flag (also -h)
misoku export note.md         # write note.pdf next to the note (-o to pick the file)
misoku paste-image            # save the picture in the clipboard under ./assets (--dir, --link)
misoku sync setup             # sync this folder through a private GitHub repository (see Sync)
misoku sync init <url>        # sync this folder through any git repository, SSH included
misoku tutor                  # learn Misoku and Markdown level by level (see Learn Misoku)
misoku tutor --reset          # start the tutorial over
```

`misoku sync setup` and `misoku sync init` set up the folder they are
pointed at: `--dir`, else `$MISOKU_WORKSPACE`, else the current folder. They
never use `notes_dir`.

## Features

- **Plain Markdown files.** No database or lock-in: use git, sync the folder, open notes in any editor.
- **A full workspace.** Start screen, file tree, tabs, statusline and 97 themes.
- **Folders on the start screen**: the top-level folders of your notes, one key away (they follow the Recent notes on 1–9); opening one shows it expanded in the explorer.
- **A start screen that follows your shortcuts**: its letter keys stay fixed (`n`, `f`, `e`, `t`, `?`, `g` for the [global graph](#global-graph), `l` for [Learn Misoku](#learn-misoku), `s`, `q`, `1`–`9`), and the leader (`space`) sequences and Ctrl/Alt shortcuts from your [`keys`](#configuration) work there too.
- **Modal keyboard editing** with mouse support, plus unsaved tabs and Save As.
- **`/` block menu** for headings, to-dos, tables, callouts, code, images and more.
- **Live preview** laid out like a page, with syntax highlighting, images and cover banners drawn in the terminal.
- **Sharp pictures**: in Ghostty, kitty, WezTerm and inside herdr, a picture is drawn by the terminal itself, pixel for pixel; anywhere else it is drawn with half blocks, as before. See [Images](#images).
- **Paste a picture from the clipboard**: `ctrl+v` in INSERT mode saves it next to the note and links it, alone on its line, so the preview draws it. Text pastes as it always did.
- **Drag blocks to reorder them** in the preview by their `⠿` handle, or add one below with `+`, like Notion.
- **Boards, calendars, timelines, charts and forms**, stored as text and editable from the preview.
- **Flowcharts** from Mermaid syntax (`graph TD` / `graph LR`), drawn with real shapes: boxes, decisions, circles, databases.
- **Equations** in LaTeX (`/equation`, a `math` block or `$$…$$`), shown as Unicode: `x^2` → `x²`, `\alpha` → `α`.
- **Export to PDF** with `:export`: a clean printable document, with boards, charts and flowcharts included.
- **2, 3 and 4 column layouts** from the `/` menu.
- **Resizable panes**: drag the dividers between the explorer, editor and preview.
- **Visible focus**: the dividers next to the pane with the focus take the theme's focus colour, and the text of the other panes turns grey — except the editor and the preview side by side, which keep their colours whichever of the two you are in, so the note stays readable while it takes shape (pictures and block colours in the preview keep theirs anyway). The explorer and the graphs still grey the panes they leave. `tab`, `ctrl+h`/`ctrl+l` (or `ctrl+←`/`ctrl+→`) or a click move the focus.
- **Dimmed backdrop**: while a dialog, the finder, the command palette or another box that takes the keys is open, the screen behind it turns grey and its pictures hide, so the box is the only thing in colour; closing it brings the colours back. The leader guide and the `/` menu leave the screen as it is.
- **Narrow windows** (under 120 columns): the explorer starts hidden and `toggle_explorer` (`ctrl+b` by default) opens it over the editor, which keeps its width; choosing a note, clicking outside it or `Esc` closes it. `space e` opens and closes it too. Wider windows keep your last toggle until the width crosses 120 again.
- **Two full-width views**, each toggled by its own key and back to the split of editor and side preview:
  - **Reading view** (`space p`): the note alone at full width, like render-markdown.nvim: line numbers hide and every line is drawn rendered (headings without `#`, bold, italic and code without delimiters, links as their text, boxes, `•` bullets and `▎` quotes; a checked task's text is muted and struck through, as in the preview). Code blocks, front matter and tables stay raw. It is read-only: you move, scroll, select and copy, search and follow links (`Enter`, `gd`), but keys that edit only say how to edit; the statusline says READING.
  - **Editor view** (`space r`): the Markdown editor alone at full width, with line numbers and editable as usual.

  Either key switches straight from the other view.
- **Fast search** across file names and note contents.
- **Search filters**: in the finder (and the `[[` link picker), `status:active tag:go folder:Zentro docker` narrows the notes before the words match their names and text. `status:` takes `active`, `onhold` (or `on-hold`), `completed`, `dropped`; `tag:go` or `#go` also finds nested tags like `#go/web`; `folder:Zentro` matches that folder anywhere in the path but not `ZentroOld`, and `folder:"My Notes"` holds spaces. Qualifiers combine; anything else, `foo:bar` included, is plain text. Only qualifiers list every matching note, newest first.
- **Tags**: write `#tag` anywhere in a note (letters, digits, `_`, `-`, `/` for nesting) or list them in the front matter as `tags: a, b`, `[a, b]` or `- a` lines. Tags are highlighted in the editor, the reading view and the preview (never inside code, links, URLs, comments or the front matter); `:tags` lists every tag with its note count and opens the finder on the one you pick (before the notes are indexed it says "Indexing notes…"), and `tag:<name>` (or `#<name>`) filters the finder.
- **Status filter** (`:status [value]`): opens the finder on `status:<value>`; without a value it opens on `status:` and lists `active · onhold · completed · dropped`.
- **Live explorer**: notes and folders created, deleted or renamed by other programs (another editor, git, a sync client) show up on their own within a couple of seconds.
- **Use mini.files**: `M` in the explorer hands the screen to Neovim's [mini.files](https://github.com/nvim-mini/mini.files) on the vault folder, with your own Neovim setup (LazyVim's included). Deleting there goes to mini.files' trash, not for good, and dot files stay hidden. Closing it (`q`) returns to Misoku, which re-reads the folder at once. It needs `nvim` on `PATH` and mini.files installed (the plugin, or the LazyVim extra `editor.mini-files`); without them the status bar says what is missing.
- **Edit a folder like Oil**: `m` in the explorer shows the selected folder (or the one holding the selected note) as editable lines, without Neovim, and works like [Oil.nvim](https://github.com/stevearc/oil.nvim). Type a line with `o`/`O` to create (`test` a note, `test/` a folder, `ideas/test` both, as with `n`); `Enter` while typing only opens a new line below, as in Vim, and nothing touches the disk; `Esc` returns to normal, dropping an empty new line. Change a name with `cw`/`cc`/`i`/`a` to rename; delete with `dd`, or by erasing the line: `Backspace` at the start of a line joins it into the line above and `Delete` at its end joins the line below, so the joined line's entry is deleted; `dd` then `p` in another folder moves. `h`/`l` go to the parent or child folder keeping every folder's edits, `u`/`ctrl+r` undo and redo. With edits pending, `Enter` in normal mode, `=`, `:w` or `ctrl+s` (also while typing) lists the actions under "CONFIRM FILE SYSTEM ACTIONS", with the links they update; Yes applies them all (deletes go to the Trash), tabs and links follow, and the editor stays open on the same folder, re-read from disk and sorted, with the cursor on the entry the line made (still typing if you were); No returns to the lines with the edits kept. With nothing pending, `Enter` enters the folder under the cursor or opens the note in the editor. Empty folders are listed too; folders holding only other files (code, pictures) stay hidden. `Esc`, `q` or `:q` leaves: with edits it shows that same confirm, where Yes applies them and closes, No drops them and closes, and Cancel keeps editing. In the tree, `i`, `I`, `a`, `A`, `o` and `O` are `m` plus that key in one press: the editor opens on the folder holding the selection, with the cursor on it, so `i`/`a` edit its name and `o`/`O` type a new line below or above it.
- **Open notes stay in sync with the disk**: a note changed by another program reloads in place (the cursor stays put). If you have unsaved edits it is left alone and you are told; saving then keeps both versions: yours in the note, the one on disk in a conflict copy next to it (`note (conflict 2026-10-16 0905).md`), and the status bar says `Both kept: …` with the copy's name. Nothing is asked and nothing is lost.
- **Open notes deleted or renamed elsewhere are kept**: the tab stays open with its text and you are warned once; saving asks before recreating the note at its old path, so a rename is never silently duplicated.
- **Safe by default**: notes are saved atomically (a crash mid-save never leaves half a file), and deleting moves them to the Trash on macOS and Linux (Windows still deletes for good).
- **Links between notes** with `[[Note]]`: jump to headings and blocks, embed one note in another, and see what links back.
- **Links follow a rename**: renaming a note or folder in the explorer (`r`) finds the links that named it, in `[[…]]` or Markdown form, and asks `Update N links in M notes?` (`y` updates them, `n` keeps the new name and leaves the links). Aliases, headings and embeds stay as written. An open tab gets the same change and keeps your unsaved edits; `u` undoes it there. A note changed on disk meanwhile is left alone and named in the status bar.
- **Messages you can read again**: errors stay in red in the status bar until you press `Esc` (which still does what it always does); other messages clear on the next key. `:messages` lists the last 100, with the time of each (results only: progress such as `Syncing…` is not kept).
- **Graph view** (`:graph`): a zoomable map of how your notes link together, with a live preview beside it; `:graph all` maps every note of the vault at once.
- **Unlinked mentions**: under the backlinks, the notes that name the open note in plain text without linking to it; click a row to open it, or its `[ link ]` button to turn every mention in that note into a `[[link]]` (after a confirm).
- **Note history** (`:history`): when the vault is in git, lists the commits that changed the open note, newest first, following it across renames, and previews the selected version beside the list (`j`/`k` or the arrows, the mouse wheel, a click). `Enter`, or a second click, puts that version back in the editor as one change: `u` undoes it, and nothing is written until you save. Without git, outside a repository, for a note never committed or a draft never saved, the status bar says why there is no history.
- **Properties**: the preview shows a note's front matter (everything but cover and icon) as a table under the header; click a value to edit it — one change `u` undoes. Lists are shown read-only; add or remove keys in the editor.
- **Daily note**: `space o d` or `:daily` opens today's note, `Diario/2026-11-06.md`, also from the home screen; when it does not exist yet a new note with that name starts in that folder, and saving creates both. `"daily_dir": "Journal"` in config.json picks another folder inside the vault.
- **Templates**: keep notes in `Plantillas/` (`templates_dir` in config.json) and type `/plantilla` (or `/template`) to insert one at the cursor; `{{fecha}}`/`{{date}}` become today and `{{titulo}}`/`{{title}}` the note's title, its front matter merges into the note's (the note's keys win), and one `u` undoes it. Templates stay out of tags, status filters and `:tags`.
- **Note status**: add `status:` to a note's front matter — `active`, `onhold` (also `on hold`/`on-hold`), `completed` or `dropped`, any case — and the explorer and the home screen's recent notes show it in place of the note icon: `●` active (blue), `◐` on hold (yellow), `✓` completed (green), `✗` dropped (muted).
- **Sync between computers** through a private git repository of your own, with no Misoku server: `:sync setup` (or `s` on the start screen, or `misoku sync setup`) once per computer connects a private GitHub repository for you; `misoku sync init <url>` works with any git host. Then it syncs on its own, and with `:sync` or `space s`. Conflicting edits keep both versions, so nothing is lost ([Sync](#sync)).

## Learn Misoku

The first time you open Misoku it greets you with a welcome dialog: the
`MISOKU` banner, a cat, and a short menu asking whether this is your first
time. `Soy nuevo` starts the tutorial, `Ya tengo experiencia` goes straight
to your notes and `Ahora no` leaves it for later on the `Learn` entry; any
answer is remembered. Move with `j`/`k` (or the arrows) and choose with
`Enter`, or press an option's letter (`n`, `e`, `a`); `Esc` is `Ahora no`.
No other key answers it. On a terminal too short for the banner it shows a
compact version, and on one too small for that it waits for a bigger one.

The start screen then keeps a `Learn` entry under `Graph`, on `l`, with the
levels you passed (`Learn 3/7`, and a check once all seven do). Either way
the tutorial is `misoku tutor`: a tutorial vault of its own, opened in the
normal app, with seven levels in Spanish, one note each, plus an index
(`00 Empieza aquí`). They teach headings, checkboxes, the `/` block menu,
`[[links]]`, links to headings, block references and the explorer. Each level
asks for one thing, and Misoku checks the note's text as you type (code is
ignored, so the lesson's own examples never count): a passed level is saved
at once and shows `✓ Nivel N superado (k/7)`, and the statusline shows
`Tutor k/7`. Progress is the lesson notes themselves, so it survives a
restart, and the tutorial comes back to the first lesson you did not pass
(the index at the start and once all seven do). `Ctrl+C` returns to your
vault. The vault lives next to `config.json`, in a `tutor` folder;
`misoku tutor --reset` moves it to the Trash and starts over from the bundled
lessons.

## Images

A picture in a note is an ordinary Markdown image on a line of its own,
`![](assets/photo.png)`, and the file lives next to the note in its `assets/`
folder: sync carries it like any other file.

- **How it is drawn.** In a terminal that speaks the Kitty graphics protocol
  (Ghostty, kitty, WezTerm, and inside herdr, which draws them itself) the
  picture is sent once and the terminal draws it, so it stays sharp while you
  scroll, fold or resize. Everywhere else, and for anything the terminal
  cannot take, Misoku draws it with `▀` half blocks as it always has.
- **Choosing.** `"ui": {"images": "blocks"}` in
  [config](#shortcuts) forces half blocks, `"kitty"` forces the terminal, and
  `"auto"` (the default) decides from the terminal. `MISOKU_IMAGES=blocks` or
  `MISOKU_IMAGES=kitty` wins over the file for one run, which is the quickest
  way to see whether a picture problem is the terminal's.
- **Pasting.** `ctrl+v` in INSERT mode, the `/` menu's *Paste image* and the
  [`paste_image`](#shortcuts) key all do the same: the clipboard's picture is
  saved as `assets/pasted-YYYY-MM-DD_HH-MM-SS.png` (with `-2`, `-3` … when the
  name is taken) and the link is written on its own line. With no picture in
  the clipboard, `ctrl+v` pastes text. Reading the clipboard needs
  `wl-clipboard` on Wayland, `xclip` on X11, `pngpaste` (or AppleScript) on
  macOS; the status bar names the one to install when none is there.
- **Super+V does not paste pictures.** Omarchy's universal paste sends
  Shift+Insert in terminals and Ghostty turns that into "paste the clipboard's
  text", so with an image-only clipboard the key is consumed before any
  application sees it. That is a property of the terminal, not of Misoku: use
  `ctrl+v`, *Paste image* in the `/` menu, or `space i`. Text keeps pasting
  with Super+V as always.
- **`misoku paste-image [--dir <folder>] [--link]`** does the same from a
  shell, for an editor that wants to paste the way Misoku does: it prints the
  path to link (or the whole `![](…)` with `--link`).

Remote pictures (`https://…`) are still shown as text, not downloaded.

## Shortcuts

Every shortcut can be changed in `config.json` (`~/.config/misoku/config.json`,
`~/Library/Application Support/misoku/config.json` on macOS), in a `keys`
section. Listing an action replaces all of its keys; an empty list turns it
off:

```json
{
  "keys": {
    "reading_view": ["space p", "space v"],
    "focus_explorer": [],
    "close_tab": ["space q", "ctrl+w"]
  }
}
```

Write keys the way the terminal names them (`ctrl+b`, `tab`, `shift+tab`,
`G`, `\\`), or as a leader sequence: `space e` is Space, then `e`, and
`space o c` is Space, then `o`, then `c` (use one key, or space plus one or
two keys; a sequence that is the start of another, such as `space o` next
to `space o c`, is reported). Ctrl and
Alt shortcuts of `quit`, `save`, `new_note`, `find_note` and
`toggle_explorer` also work while typing; every other shortcut only works
outside INSERT mode, so it never gets in the way of your text. The `?` help
lists the keys you configured (see [Command palette](#command-palette)). Press Space and pause: after 400 ms a guide
lists the keys that can follow, in three groups (NOTE, VIEW, GO), with the
keys you configured; `o +misoku` leads to the keys of the LazyVim companion,
listed on their own page after `space o`. `Esc` closes it; type the next key sooner and it never
shows. On the home screen, a key that needs an open note (such as `space p`)
says so in the status bar. A misspelled action, a malformed key, or a key given to two actions is
reported in the status bar when Misoku starts (it stays until `Esc`, and
`:messages` shows it again). The action names of older versions still work:
`reading_mode` binds the reading view (`reading_view`) and `toggle_preview` the
editor view (`editor_view`); when a config gives both names, the new one wins.

Vim editing keys (`hjkl`, `w`, `dd`, …) are not configurable.

**Undo** (editor, NORMAL mode, not configurable)

| Key | What it does |
| --- | --- |
| `u` | Undo the last change; the status bar says which one (`1 change; before #3`) |
| `Ctrl+R` | Redo what `u` undid |

Each INSERT session is one change, and so is each NORMAL command (`dd`, `p`,
`x`) and each edit made from the preview, a table or the `/` menu. Every tab
keeps its own history, in memory only: up to 200 changes or 8 MB, the oldest
dropped first; a note over about 4 MB keeps no undo history, and `u` says so.
Undoing never writes to disk, and a note reloaded because it
changed on disk starts a new history.

### Command palette

`?` (or `space h`, `space ?`) opens the command palette, which is also the
help. It lists every key and command, built from the shortcuts in use: each
action with its full keys from your `keys` config, every `:` command of the
editor, and the fixed keys of the editor (including `u` undo and `Ctrl+R`
redo), the preview, the graph and the mouse. With an empty search the ways to
export come first, under **EXPORT**.

Type to search: the filter is fuzzy, like the note finder, so `exp` finds
exporting, the explorer and `:export`. Everything you type goes to the
search, `?`, `q` and Space included; only `Esc` closes the palette. `↑`/`↓`
move, and the bottom explains the selected entry. `Enter` runs it when it is
an action or a command that needs nothing more (`:settings`, `:messages`,
`:sync`, `Reading view`…); a key that only works in its own place (the
explorer, the editor, the graph) is just explained. Opened from the home
screen, `Enter` runs only what the home screen runs; what acts on a note
(`Reading view`, `Close tab`, `:w`, `:export`…) says "Needs an open note".
The list scrolls when it does not fit, so no row is cut.

**General**

| Action | Default keys | What it does |
| --- | --- | --- |
| `new_note` | `ctrl+n`, `space n`, `space o n` | New note (unsaved tab) |
| `save` | `ctrl+s`, `space w` | Save (new notes: pick folder) |
| `find_note` | `ctrl+p`, `\`, `space f`, `space o f`, `space o g` | Find note (opens in this tab) |
| `toggle_explorer` | `ctrl+b` | Toggle explorer |
| `focus_explorer` | `space e` | Focus / close explorer (opens and focuses it; with the focus, closes it and returns to the editor) |
| `next_pane` | `tab` | Next pane |
| `prev_pane` | `shift+tab` | Previous pane |
| `pane_left` | `ctrl+h`, `ctrl+left` | Pane to the left |
| `pane_right` | `ctrl+l`, `ctrl+right` | Pane to the right |
| `next_tab` | `]`, `L`, `space tab` | Next tab |
| `prev_tab` | `[`, `H`, `space shift+tab` | Previous tab |
| `close_tab` | `space x` | Close tab |
| `reading_view` | `space p` | Reading view: the rendered note alone, full width and read-only (`space p` again brings the split back) |
| `editor_view` | `space r` | Editor view: the Markdown editor alone, full width, with line numbers (`space r` again brings the split back) |
| `export_pdf` | `space P` (Space, then Shift+P) | Export note as PDF (same as `:export`) |
| `themes` | `space t` | Themes |
| `graph` | `space g` | Graph of this note's links |
| `global_graph` | `space G` (Space, then Shift+G) | Graph of every note (same as `:graph all`, see [Global graph](#global-graph)) |
| `home` | `space d` | Home screen |
| `daily_note` | `space o d` | Today's daily note, `<daily_dir>/YYYY-MM-DD.md` (same as `:daily`); a new one is a draft until you save it |
| `sync_now` | `space s` | Sync the vault with git now (same as `:sync`, see [Sync](#sync)) |
| `paste_image` | `space i`, `space o p` | Paste the picture in the clipboard next to this note and link it (see [Images](#images)); in INSERT mode `ctrl+v` does it too |
| `toggle_checkbox` | `space o c` | Toggle the box of this line: `[ ]` becomes `[x]`, `[x]` or `[-]` becomes `[ ]`, and a line without one gets `- [ ] `. On a VISUAL or V-LINE selection it toggles every line and keeps the selection; the whole change is one `u` (after `Esc`) |
| `remove_checkbox` | `space o C` (Space, `o`, then Shift+C) | Remove the box of this line or of the selection's lines: `- [x] task` becomes `- task` |
| `next_link` | `space o ]` | Move to the start of the next `[[link]]`, past the last back to the first; links in code are skipped, and a note with none says "No links in this note" |
| `prev_link` | `space o [` | Move to the start of the previous `[[link]]`, wrapping the same way |
| `help` | `?`, `space h`, `space ?` | This help: the [command palette](#command-palette) |
| `quit` | `ctrl+c` | Quit (asks to save) |

On macOS, `ctrl+left` and `ctrl+right` switch desktops (Mission Control) and
never reach the terminal unless that system shortcut is turned off (System
Settings → Keyboard → Keyboard Shortcuts → Mission Control), so `ctrl+h` and
`ctrl+l` remain the reliable way to change pane there. While typing (INSERT
mode) neither pair changes pane.

**Explorer** (only while the explorer has focus)

| Action | Default keys | What it does |
| --- | --- | --- |
| `explorer.up` | `up`, `k` | Move up |
| `explorer.down` | `down`, `j` | Move down |
| `explorer.top` | `g`, `home` | First item |
| `explorer.bottom` | `G`, `end` | Last item |
| `explorer.page_down` | `ctrl+d`, `pgdown` | Half page down |
| `explorer.page_up` | `ctrl+u`, `pgup` | Half page up |
| `explorer.open` | `enter` | Open note / expand |
| `explorer.preview` | `p` | Preview note (folder: open) |
| `explorer.expand` | `l`, `right` | Expand / open |
| `explorer.collapse` | `h`, `left` | Collapse / go to parent |
| `explorer.toggle_fold` | `z` | Fold / unfold |
| `explorer.collapse_all` | `W` | Collapse all |
| `explorer.new_note` | `n`, `%` | New note or folder (end with / for a folder) |
| `explorer.new_sub_note` | `s` | New sub-note |
| `explorer.rename` | `r`, `e` | Rename |
| `explorer.delete` | `d`, `x`, `delete` | Delete (asks first) |
| `explorer.find` | `\`, `f` | Find note |
| `explorer.file_manager` | `M` | Open the vault in Neovim's mini.files |
| `explorer.edit_folder` | `m` | Edit the folder like text (Oil) |
| `explorer.insert` | `i`, `I`, `a`, `A`, `o`, `O` | Start typing in the folder editor |

## Look

The `ui` section of `config.json` changes how notes look in the preview.
Anything left out keeps Misoku's default. For example, the look of a Neovim
with render-markdown:

```json
{
  "ui": {
    "heading_icons": ["① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "],
    "heading_background": true,
    "bullets": ["●", "○", "◆", "◇"],
    "checkbox": { "unchecked": "󰄱", "checked": "󰱒", "todo": "󰥔" },
    "mode_icon": "󱗞",
    "border": "single",
    "modified_mark": "[+]"
  }
}
```

| Key | What it does |
| --- | --- |
| `heading_icons` | Icon before headings of level 1 to 6; missing levels keep the default |
| `heading_numbers` | `true` (default): the preview numbers headings `1`, `1.1`, `2` in place of their level icon. A single `#` that is the note's first heading is its title and keeps its icon. Only the preview changes, never the note. `false` shows the icons |
| `heading_background` | Paints each heading on a full-width band of its color (the H1 underline is then left out) |
| `bullets` | List bullets by depth, cycling |
| `checkbox.unchecked`, `checkbox.checked` | Marks for `- [ ]` and `- [x]` |
| `checkbox.todo` | Mark for `- [-]` (in progress). Without it, `[-]` stays plain text |
| `mode_icon` | One icon for every mode in the statusline, like lualine (`"󱗞"`) |
| `border` | Frame of dialogs, menus and preview blocks: `rounded` (default) or `single` |
| `modified_mark` | Shown before the tab name of a note with unsaved changes, like incline (`"[+]"`) |
| `accordion` | `true` (default): opening a folder in the explorer closes its open siblings. `false` keeps several open |
| `explorer_hide_width` | Terminal width, in columns, below which the explorer hides (default `120`) |
| `cursor_shape` | `"auto"` (default): the cursor itself shows the mode, a block in NORMAL and VISUAL and a thin bar in INSERT, like LazyVim. `"block"` keeps Misoku's drawn block in every mode |
| `transparent` | `true` leaves the background of the editor, preview, explorer, tab line and statusline to the terminal, so a terminal with opacity or blur shows through. Popups go clear too, set apart by their border, and heading bands are off; highlights stay painted. `false` (default) paints the theme's background |

### Transparent background

With `"transparent": true` Misoku stops painting its base background, the way Claude Code draws in a terminal:
your terminal's own background colour shows through, with its opacity and blur if it has them. Pick a theme
whose colours match that background (Omarchy sets both); a terminal without opacity just shows its default
colour. Like Neovim's transparent themes, popups (dialogs, menus, the finder, the palette, toasts) are
transparent and set apart only by their border, and heading bands are off: headings keep their colour, bold
and icon. The cursor line, selections, statusline blocks and badges keep their background. Pictures in the
preview keep an opaque rectangle.

### Cursor

With `"cursor_shape": "auto"` (the default) the mode shows at the cursor itself, as in LazyVim: a block in
NORMAL, VISUAL and VISUAL-LINE, a thin bar in INSERT, and no cursor while the `:` command line has its own.
It is the terminal's own cursor, so it looks the way your terminal draws it; Misoku only says where it goes
and what shape it takes, and a terminal that does not know the shape sequence ignores it and keeps working.
`"cursor_shape": "block"` keeps the drawn block Misoku always had, and `MISOKU_CURSOR_SHAPE=block` (or `auto`)
wins over the file for one run. When Misoku exits it gives the terminal its cursor back; a crash it cannot
catch (a panic or `kill -9`) can leave the shape set until the terminal resets it.

### Settings menu

`:settings` (or **Settings**, key `c`, on the home screen) opens a menu for
six of these keys: accordion folders (on or off), the explorer hide width,
heading icons and bullets (one of the presets Misoku, Numbers, Arrows and
Plain; Misoku removes both keys), the dialog border, the transparent
background (on or off; off removes the key), and heading numbers (on or off). A value picked with Enter applies at once and is saved at once
in `config.json`: only that key changes, and every other setting stays as you
wrote it. Esc goes back without saving, and picking the value already in use
writes nothing. If the file cannot be written, the status bar says why.

## Links

Type `[[` to link to another note. Click a link in the preview, or press
`gd` on it in the editor, to follow it. In NORMAL mode `Enter` on a
`[[link]]` follows it too, as in the LazyVim companion: a link inside a code
block or inline `` `code` `` is skipped, and a link to a note that doesn't
exist says "No note found for this link" instead of starting it (use `gd`
for that). `Enter` on a checkbox line (`- [ ]`, `- [x]`, `- [-]`) toggles
the box; anywhere else it moves down a line. Clicking the box of a to-do
line in the preview toggles it too, as one change `u` undoes; clicking the
text of the line does not. `space o ]` and `space o [`
jump to the next and previous `[[link]]` (the companion's `]o` and `[o`,
whose `]` and `[` switch tabs in Misoku), skipping the same code.

```md
[[Docker]]                          open a note
[[AWS Notes#S3 Storage]]            open it at a heading
[[AWS Notes#S3 Storage|Learn S3]]   show your own text
[[AWS Notes#^s3-basics]]            open it at a block marked ^s3-basics
![[AWS Notes#^s3-basics]]           show that block right here
![[diagram.png]]                    show an image from anywhere in the workspace
```

Mark a paragraph or list item as a block by ending it with ` ^some-id`.
A link to a note that doesn't exist yet is dimmed; following it starts that
note, and nothing is written until you save. Every note lists the notes that
link to it at the bottom of the preview. Below them, "Unlinked mentions" lists the notes that write this note's title (whole word, any case) outside code, links and front matter without linking to it; `[ link ]` rewrites them as `[[Title]]`, or `[[Title|as written]]` when the case differs, skipping a note that changed on disk since and refusing one open with unsaved edits. Tags, emails, domains, paths, reference links and block ids are never counted. Web links (`[text](https://…)` and
`<https://…>`) open in your browser.

### Graph

`:graph` (or `Space g`) opens the graph in a tab, centred on the note you
were in: notes that link to it on the left, the notes it links to on the
right. The preview on the right shows whichever note is selected.

```
                              ╭─● Untitled-1
           Other ●─╮          │            ╭─● EC2
                   ├─● Note ──┼─● AWS ─────┤
 Index ●─── Study ●─╯          │            ╰─● S3
                              ╰─● Docker
```

With the keys:

| Key | What it does |
| --- | --- |
| `h` / `l` | Move left / right: to the note's parent or its first child, as drawn |
| `j` / `k` | Next / previous sibling (notes hanging from the same note) |
| `enter` | Open the selected note |
| `-` / `+` | Zoom out / in |
| `0` | Re-centre the map on the selected note |
| `esc` | Back to the previous centre |

With the mouse:

- Click a note to preview it; click it again to open it.
- Scroll to zoom out and in: zoomed out, the map also shows the links of
  those notes, up to four steps away. The `−` and `+` buttons do the same.
- Drag to move around the map; `reset` puts it back.
- `0 centre` re-centres the map on the selected note; `esc back` returns to
  the previous one.

The bottom row lists these keys, shortened to fit narrow panes.

Close the graph like any tab.
It picks up links as you type them in other tabs.

#### Global graph

`:graph all` (or `Space G`, or `g` on the start screen) opens a second
graph tab with every note of the vault, whether it links anywhere or not,
and the links between them, drawn as a solar system that fills the pane.
The vault's sun `◉` sits in the middle in a small halo with the vault's
name over it, and the notes at the vault root orbit it. Each top-level
folder is a galaxy in a colour of its own on a big oval orbit around it,
big and small galaxies taking turns round it: a sun `✦` with the folder's
name and its number of notes over it (`Omarchy · 14`), and the folder's
notes on rings around that sun, busier notes closer in, so a folder with
more notes is a bigger galaxy. Folders that link to each other sit side by
side on the orbit. Dimmer dotted lines show the structure, so every note is
visibly part of something even without links: each galaxy is joined to the
vault's sun by a gentle curve, each note to its galaxy's sun, and each ring
that holds notes is drawn. Your links run over them as brighter, solid
curves: in the galaxy's colour inside one, light grey between two. Suns
are landmarks only; clicks and keys select notes. Every note is a dot in
its galaxy's colour (`•`, `●` for each galaxy's busiest linked note), and
a link to a note that doesn't exist yet is a hollow `○` named after the
link. The note you
were in is selected; its links are lit and the rest fades without losing
its galaxy's colour. The title row counts notes, links, notes without links
and missing notes.

| Key | What it does |
| --- | --- |
| `h` `j` `k` `l` / arrows | Select the nearest note on that side |
| `n` / `N` | Next / previous note the selected one links to or is linked from |
| `enter` | Open the selected note; on a missing note, start it (nothing is written until you save) |
| `-` / `+` | Zoom out / in: the whole solar system with each folder's name and count and each galaxy's busiest notes named, then the names of the busiest notes, then every name that fits |
| `0` | Centre the view on the selected note; with nothing selected, show the whole map again (or the vault's sun when it doesn't fit) |
| `esc` | Clear the filter; without one, back to the previous selection |
| `/` | Filter: notes whose name or path doesn't hold the text are greyed out, not removed. `enter` selects the best match (a name starting with the text, then a name holding it, then a path; the shortest name first) and keeps the filter; `esc` clears it. While you type, every key is text |
| `o` | Hide / show notes without links (orphans) |
| `u` | Hide / show links to missing notes |

Hidden notes keep their place, so the map keeps its shape; the keys and the
mouse skip them, and if the selected note is hidden the selection moves to
the note you were in, or to the nearest note still shown. The title row
keeps counting them and says which are hidden.

The mouse works as in the graph: click a note to preview it and again to
open it, scroll to zoom around the mouse, drag to move, and the bottom row's
buttons (`−` `+` `0 centre` `esc back` `reset`, and on a wide enough pane
`/ filter` `o orphans` `u unresolved`). Both graph tabs can be open
at once, each keeping its own selection. Keys you bind in
[`keys`](#configuration) win over these letters, as in the graph. `:graph`
with anything but `all` after it is an error.

## Flowcharts

Write a `flow` (or `mermaid`) code block and the preview draws it:

````md
```flow
graph LR
start([Start]) --> check{Is it working?}
check -->|yes| done([Ship it])
check -->|no| fix[Fix it]
fix --> check
```
````

Shapes: `[box]`, `(rounded)`, `([stadium])`, `{decision}`, `{{hexagon}}`, `((circle))`, `[(database)]`, `[[subroutine]]`.
Arrows: `-->`, `-.->` (dotted), `==>` (thick), with labels as `-->|yes|` or `-- yes -->`.

## Equations

Type `/equation`, or write a `math` block or `$$ … $$`. Each line is one equation:

````md
```math
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
E = mc^2
```
````

The preview shows `x = (−b ± √(b² − 4ac))/(2a)` and `E = mc²`. Greek letters, operators, arrows, `^` / `_`, `\frac`, `\sqrt`, `\mathbb` and accents are supported. Unicode has no superscript for some letters, so those print as `^(…)`.

## Export to PDF

In the editor, `:export` (or Space, then Shift+P) writes the open note as a PDF next to it. The `?` help lists every way to export at the top, under **EXPORT**. `:export ~/Desktop/` or `:export report.pdf` picks another place. From the shell:

```sh
misoku export notes/plan.md               # notes/plan.pdf
misoku export notes/plan.md -o plan.pdf
```

The PDF is a printable A4 document with headings, lists, tables, syntax-highlighted code, equations and images. Boards, calendars, timelines, charts and flowcharts print in colour as they look in the preview.

Table cells wrap between words, and after `-` or `/` in long codes and paths. A table too wide for the page prints in slightly smaller type; if it still does not fit, that table gets landscape pages of its own (its heading goes with it) and the rest of the document stays upright. The header row repeats on every page a table spans.

## Sync

Misoku can keep a notes folder (a *vault*) the same on all your computers through a git repository you own: GitHub, GitLab, Gitea or your own server. There is no Misoku server, and your notes stay plain files with their full history in git.

**How it works.** Sync is off until you set it up for a folder, and only folders listed in `sync.vaults` ever sync: Misoku never runs git anywhere else. In a listed vault, a sync commits everything that changed (notes, images, attachments, exported PDFs), fetches what your other computers pushed, joins both, and pushes. A sync runs:

- when Misoku opens the vault;
- every 5 minutes while Misoku is open (the vault's [`interval`](#configuration)), counted from the last sync;
- when you type `:sync` or press `space s` (`sync_now`, configurable like any [shortcut](#shortcuts));
- when you quit: once the screen closes, Misoku pushes what you saved, printing `Pushing changes…` and the result. It gives up after 10 seconds (or at once with Ctrl+C) and the next sync pushes it. Closing the terminal pane or window Misoku runs in also pushes, writing the result to Misoku's log.

**What arrived.** After a sync that brought notes, automatic ones included, the status bar names them, by their path in the vault: `2 new notes from another computer: nota.md, otra.md`, `1 note updated from another computer: a.md`, or `2 new notes, 1 updated from another computer: a.md, b.md, c.md`. A long list is cut after three names (`... +2 more · :sync status`, or just `... +2 more` on the start screen, where `:` opens nothing): `:sync status` and `:messages` list the first ten of the last notes that arrived and count the rest. Deleted notes and files that are not notes are not named, and no dialog opens.

**With Misoku closed, nothing syncs.** Nothing runs in the background: your notes arrive the next time Misoku opens the vault.

**When sync stops, press `space s`** (the bar names your `sync_now` key, and gives up its right-side parts to fit the message; at the narrowest it reads `Sync offline · space s`). No dialog ever opens on its own; only the key opens one, so nothing takes the keys of the note you are writing; if you typed before it could open, the bar says to press the key again:

- **Offline:** it tries again.
- **A leftover git lock:** it opens the dialog to remove the lock, the same as `:sync status`. If the lock is gone by then, it just syncs.
- **GitHub access expired** (an HTTPS origin on github.com, with gh installed): it offers to sign in (with the arrow keys or Tab, then Enter; Enter alone cancels). gh shows a one-time code, opens your browser and keeps the access; once you are back, sync runs. Any other origin that refuses access gets an explanation instead.
- **The history was rewritten, the histories are unrelated, a merge is unfinished, or the vault is not on a branch:** it explains the problem in plain words and changes nothing. **Check again** syncs, so a problem you fixed clears. No key joins the histories: after a force-push, that could upload again something deleted on purpose. `:sync status` keeps the technical detail, with the git commands.

**After you save**, Misoku commits your changes locally once you stop saving for a few seconds, so a burst of saves makes one commit. This commit uses no network: the next sync pushes it. Syncs and these commits run in the background one at a time, so you can keep typing; a save made while one runs goes into the next commit, and a sync you ask for while a commit runs starts right after it. Each git command has a time limit, and git never stops to ask for a password or a key (that only happens in `misoku sync setup` and `misoku sync init`).

**You need** git 2.38 or newer and a **private** repository: the assistant below creates one on GitHub for you. For `misoku sync init` you also need a git identity (`git config --global user.name "Your Name"` and `user.email`) and an empty private repository, or the one your other computer already syncs with.

### Set up a vault with GitHub (recommended)

With a GitHub account, the assistant sets everything up. In Misoku, open the vault and press `s` on the start screen (**Sync between computers**, shown while the folder does not sync yet, that is, while it is not listed in `sync.vaults`; it goes away once setup succeeds) or type `:sync setup`; or, in a terminal:

```sh
misoku sync setup --dir ~/notes
```

`--dir` defaults to `$MISOKU_WORKSPACE`, then to the current folder. The assistant:

1. Says what will happen and waits for **Start**. It refuses your home folder, the root of a disk and system folders, changing nothing (syncing your home folder would push `~/.ssh` too): sync a folder of its own, such as `~/Notes`. `misoku sync init` refuses them too. A folder already listed in `sync.vaults` keeps syncing as before. A vault inside another git repository is first offered a repository of its own ([a vault inside another repository](#set-up-a-vault-with-any-git-url)).
2. Checks git (2.38 or newer) and that **gh**, GitHub's command-line tool, is installed: on your `PATH`, or next to `misoku`, where the installer puts it. On a Mac where Apple's installer for git is still running, it asks you to press Enter once it finishes. If gh is not installed, it says how to install it (`brew install gh` on macOS, `winget install --id GitHub.cli` on Windows, your package manager on Linux, see [cli.github.com](https://cli.github.com)) and stops, changing nothing.
3. If gh is not signed in, runs `gh auth login --web --git-protocol https --hostname github.com`: gh shows a one-time code and opens your browser, where you sign in and enter it (inside Misoku, the screen steps aside while it runs). gh asks nothing else: the flags answer its questions, and it does not ask about git's credentials, which step 6 sets up. On a computer with no browser to open (Linux without a desktop, or WSL without `wslview`), the dialog says so: open https://github.com/login/device on any device, sign in and enter the code gh shows.
4. If git does not know your name and email, asks for them, already filled in from your GitHub account: your profile name (or your login, when the profile has none) and GitHub's private address, `<id>+<login>@users.noreply.github.com`, which keeps your own email out of your notes' history. Change them if you like; they are saved with `git config --global` only once you confirm. If gh cannot tell, you type them.
5. Lets you **create a new private repository** (it offers the vault folder's name; it is always the first entry) or **pick one of yours** from a list. Inside Misoku, type to filter the list (type `/` first to filter by digits); in a list of ten or more, a number shows as `Number: 5_` and is picked with Enter, and a number the list does not have shows as `No item 56` and picks nothing. In a terminal, type part of a name to list the matches. Public (and organization-internal) repositories are marked and refused: your notes are stored unencrypted. To use one, make it private on GitHub first. Before using a repository you already have, and before step 6, setup reads its default branch: one that looks like a code project or a site is refused, changing nothing, so your notes never land in a website's or an app's repository. The signs are `package.json`, `go.mod` or `Cargo.toml` anywhere; `pyproject.toml`, `vercel.json`, `Dockerfile`, `_config.yml`, `config.toml`, `hugo.toml`/`.yaml`/`.json`, `mkdocs.yml`, `netlify.toml`, `CNAME`, `Gemfile`, `.gitmodules`, `src/`, `app/`, `.github/workflows/` at the top; `node_modules` anywhere; and `themes/`, `layouts/`, `_layouts/`, `_posts/` next to a site's config file. A repository holding notes that do not share the vault's history, or one where more than a fifth of the files are neither notes nor attachments (images, videos, audio, PDFs), is used only once you confirm, after it lists what it holds (the default answer is no). An empty repository, or the notes for a new computer's empty folder, is used as before. If setup cannot read the repository, it stops and says why, changing nothing.
6. Runs `gh auth setup-git`, so git uses gh's access and every later sync, in the background, signs in without asking. Then it connects the vault over HTTPS exactly as `misoku sync init` does (all three cases below), and lists it in `sync.vaults`.
7. Says what it did, that the vault now syncs every 5 minutes and when you quit, and what to do on your other computer: install Misoku, run `misoku`, choose **Bring my notes from another computer** and pick the same repository.

**Misoku never stores or sees your GitHub access**: gh keeps it in your system's keychain, and Misoku never prints a token or passes one on a command line. To sign out, run `gh auth logout`. Cancelling (Esc, **Cancel**, or the end of the input in a terminal) changes nothing more: before you confirm your name and email, or before git gets gh's access (step 6), nothing has changed at all; after either, the message says exactly what was done (for example `Your git name and email were saved; GitHub access for git was set up with gh. Nothing else changed.`). Inside Misoku, quitting waits while a step runs, so a setup is never left half done; Esc while a step runs (git bringing the notes, or gh waiting on GitHub) stops it at once and cancels the setup the same way, removing what it created (on the welcome screen, Ctrl+C does too). gh's sign-in, which runs outside the screen, is ended in the terminal. Running the assistant on a vault that already syncs says so and changes nothing.

### Set up a vault with any git URL

For SSH, GitLab, Gitea, your own server, or a repository you created by hand, run this once on each computer, in a terminal:

```sh
misoku sync init git@github.com:you/notes.git --dir ~/notes
```

`--dir` defaults to `$MISOKU_WORKSPACE`, then to the current folder. This is the only step that may ask you something: accepting the SSH key of a new host, or signing in. It works in each of these cases:

| Case | What init does |
| --- | --- |
| **First computer**: the folder has notes, the repository is empty | Creates a git repository (branch `main`), commits every note and pushes them |
| **Another computer**: the folder is empty or does not exist yet (its parent folder must exist), the repository has notes | Creates the folder if needed and brings the notes |
| **Both have notes**, including a repository created with only a README | Joins the two once, keeping both versions of any file that differs (see [Conflicts](#conflicts)) |

It also writes `.gitattributes` with `* -text`, so line endings never change between macOS, Linux and Windows (if the vault already has a `.gitattributes`, it is kept and init tells you to add that line); adds Misoku's temporary files and `.DS_Store` to `.git/info/exclude` (your `.gitignore` is not touched); sets up `origin`; and adds the folder to `sync.vaults` in your config, keeping every other setting. When it finishes, it lists, one line each, the files that do not sync as they are (see [What does not sync](#what-does-not-sync)).

Running init again is safe: when everything is set up it prints `Already set up: … nothing changed.` It changes nothing and says why when the folder already has a different `origin`, when a merge, rebase, detached HEAD or git lock file of yours is pending, or when a Misoku window is syncing the same vault.

**A vault inside another repository** (for example, your home folder is a git repository) cannot sync as it is, because syncing it would commit that whole repository. `misoku sync setup` (and `:sync setup`) asks "This folder is inside another git repository. Create a separate one for your notes?": Enter runs `git init` in the vault folder and goes on, and the outer repository is not changed. If you cancel later in the same run (Ctrl+C at a question cancels too), that new repository is removed again while it has no commits (one with commits is left in place, and the message says so). `misoku sync init` does not ask: it refuses, changing nothing; move the vault outside, or run `git init` in the vault folder yourself, then run init again. When the outer repository tracks files in the vault folder, or has it as a submodule, both stop and change nothing, since your notes would end up in two repositories: move the notes to a folder outside it.

### Configuration

Init writes the entry for you; you can also list a vault by hand in `config.json` (the [same file](#shortcuts) as the shortcuts):

```json
{
  "sync": {
    "vaults": {
      "~/notes": { "device": "laptop", "interval": "10m" }
    }
  }
}
```

- The key is the vault's folder, absolute or starting with `~`.
- `device` (optional) names this computer in commit messages and conflict copies. It defaults to the computer's name.
- `interval` (optional) is how often Misoku syncs on its own while it is open, such as `"10m"`, `"90s"` or `"1h"`. It defaults to `"5m"` and must be at least `"1m"`: a value Misoku cannot use (shorter, zero, negative, or not a duration) is reported in the status bar (with the outcome of the sync on opening) and in `:sync status`, and the 5-minute default is used. `"off"` turns timed sync off: the vault still syncs on opening, when you ask and when you quit, and saves are still committed. `:sync status` shows the interval in use (`Timed sync: every 5m`).

To turn sync off for a moment without editing the config, start Misoku with `MISOKU_SYNC=0`: no git runs at all, and `:sync` says `Sync is off (MISOKU_SYNC=0)`.

### Privacy

Your notes are stored **unencrypted** in the repository: anyone who can read it can read them. **Keep the repository private.**

Misoku never shows credentials: `:sync status`, messages and init's output print the URL without any user name, password or token in it. git itself, though, stores the URL you give exactly as written in the vault's `.git/config`. Prefer SSH keys or a credential helper (for GitHub over HTTPS, `gh auth setup-git`) to a token inside the URL.

### Conflicts

When two computers change different notes, or different parts of the same note, the changes are joined on their own. When both changed the **same lines** of a note (or the same image or file, or each created a note with the same name), **both versions are kept**: your version stays in the note and the other one goes next to it as a *conflict copy*:

```text
plan.md
plan (conflict laptop 2026-10-15 1430).md
```

The name holds the computer the other version came from (left out when another tool made that change) and the local time, with no `:` so it is valid on Windows; ` 2`, ` 3`… is added if the name is taken. No conflict markers (`<<<<<<<`) ever go into your notes.

**To resolve one**, compare the two files, copy what you want to keep into the note, and **delete the copy**. A copy is a file like any other: it syncs to your other computers, and deleting it deletes it everywhere. The status bar shows how many copies the vault holds, and `:sync status` lists them.

**A note you are editing when a sync brings another computer's edit of it** is not reloaded while you have unsaved changes. When you save, your version goes into the note and the other one into a conflict copy next to it, named the same way but without the computer's name (`plan (conflict 2026-10-15 1430).md`); the status bar says `Both kept: …` with the copy's name.

A note deleted on one computer and edited on another is kept, with the edits. A note renamed differently on two computers keeps both names. A note renamed on one computer and edited on the other ends as two notes, the renamed one and the edited one, and both are kept. Nothing is ever lost: the other version is either in a copy or in git's history.

### Status

The status bar shows the state of the last sync:

| Indicator | Meaning |
| --- | --- |
| `syncing…` | A sync is running (the commit after a save shows nothing) |
| `synced just now`, `synced 5m ago` | All in step with the repository |
| `· N files to check` | Some files do not sync as they are (see below) |
| `N conflicts` | Conflict copies to resolve |
| `offline · N pending · space s` | The repository could not be reached; N commits wait to be pushed, and the key retries |
| `sync will retry` | The sync stopped short; the next one tries again |
| `sync paused · space s` | Something needs you first; nothing was touched |
| `sync paused, lock left · space s` | A git lock file stopped the sync (see [Troubleshooting](#troubleshooting)) |
| `sync error` | The repository refused, or git failed |
| `other window syncs` | Another Misoku window syncs this vault |

A sync that starts on its own only writes to the status line when it made conflict copies, reloaded an open note, or hit a new problem; otherwise the indicator says how it went.

`:sync status` shows the detail: state, remote (without credentials), branch, how often timed sync runs, commits waiting to be pushed, when it last synced, the last error, the conflict copies on disk, the files that do not sync as they are, and the notes the last sync that brought any brought.

### What does not sync

| What | Why |
| --- | --- |
| Files over **50 MB** | GitHub refuses files over 100 MB, which would block every sync. A file that grows past the limit keeps its last committed version. |
| Files your `.gitignore` ignores | Ignored files are respected |
| Misoku's temporary save files and `.DS_Store` | Listed in `.git/info/exclude` |
| A folder with its own git repository inside the vault | git would only record which commit it is on, not its files. Move it out of the vault, or ignore it. A git submodule syncs only its commit id. |
| What a symbolic link points to outside the vault | Only the link itself syncs |
| New files with a name Windows cannot use (`:`, `*`, `?` and similar, a name ending in a dot or a space, `CON`, `NUL`…) | They would stop every sync on a Windows computer. They stay out until you rename them; meanwhile, deletions are held back, so a note renamed to such a name keeps syncing under its old one. |

Everything else syncs, including the PDFs `:export` and `misoku export` write into the vault, and the pictures a paste puts in `assets/`. Each of these files gets one line, saying what to do, in `:sync status` and in the output of `misoku sync init`.

### Troubleshooting

| You see | What to do |
| --- | --- |
| `offline, N pending` | Nothing to do: the commits are safe in the vault and the next sync, once the network is back, pushes them. |
| `origin's SSH host key is not known yet` | Run `misoku sync init <url>` in a terminal once and accept the key. |
| `origin refused your SSH key` | Add your key to your account on the git host (and to `ssh-agent`). |
| `GitHub did not let this computer in` | Press `space s` and sign in. |
| `could not sign in to origin` | For GitHub over HTTPS run `gh auth setup-git` (or `misoku sync setup`, which runs it), or set up your git credentials. If it still fails, run `misoku sync init <url>` in a terminal. |
| `origin's SSH host key changed` | It can mean an attack or a reinstalled server: check the key with `ssh` in a terminal before syncing again. |
| `another Misoku window syncs this vault` | Only one window syncs a vault; the others see its changes within seconds. When that window closes, the next sync here takes over. |
| `Sync paused: …` in plain words, `space s` | Press `space s` to read what happened. `:sync status` shows one of the lines below, with its git commands. |
| `… is in progress in this vault` | A merge, rebase, cherry-pick or revert you started is waiting: finish it (`git merge --continue`…) or cancel it (`git merge --abort`…). Sync waits and touches nothing. |
| `HEAD is detached` | Check out your branch, for example `git switch main`. |
| `a git lock file is in the way (.git/index.lock)` | A git command is running in the vault, or one stopped half-way. If no git is running, press `space s` (or open `:sync status`) and choose **Remove the lock and sync** (with the arrow keys or Tab, then Enter). If you typed before the status appeared, it leaves the choice out: open it again. Misoku removes a lock on its own only when it knows it left it itself. |
| `this vault is inside the git repository at …` | See [a vault inside another repository](#set-up-a-vault-with-any-git-url). |
| `… was rewritten (a force-push)` | Someone rewrote the repository's history. Sync stops so nothing is lost: compare with `git fetch origin` and `git log origin/main`, and join them by hand. |
| `unrelated histories` | The vault and the repository share no history: join them once with `misoku sync init <url>`. |
| `Sync stopped, it tries again next time` | Another computer pushed first, a file changed during the sync, or a file could not be replaced because another app has it open (common on Windows): close it if so, and sync again. If the same message keeps coming back, read it in `:sync status`: it names the file in the way (for example a file left out whose name the repository also uses) and how to fix it. |
| `git … is too old` / `git did not run` | Install or update git (2.38 or newer). |
| `git does not know who you are` | Set `user.name` and `user.email` as the message shows, or run `:sync setup`, which asks for them. |
| `gh, GitHub's command-line tool, is not installed` | Install gh as the message says, then run `:sync setup` (or `misoku sync setup`) again. |
| `… is public: anyone could read your notes` | The assistant only uses private repositories. Pick a private one, create a new one, or make that one private on GitHub first. |
| `… looks like a code project (…)` | The repository holds another project, not notes: setup will not mix your notes into it. Pick another repository, or create a new private one. |
| `A repository named … already exists` | Choose another name, or run setup again and pick that repository from the list. |

## Columns

Type `/2 columns` (or 3, 4). Each `+++` line starts a new column, and each column is ordinary Markdown:

```md
~~~columns
## Todo
- [ ] Write the docs
+++
## Notes
Columns stack on narrow screens.
~~~
```

## Custom themes

Drop a JSON palette into the themes folder in your user config directory:

- **Linux:** `~/.config/misoku/themes/<name>.json`
- **macOS:** `~/Library/Application Support/misoku/themes/<name>.json`
- **Windows:** `%APPDATA%\misoku\themes\<name>.json`

User themes appear automatically in the live theme picker (`Space` `t`) and in `--list-themes`. Bundled themes take precedence on a name collision.

Editing a user theme file while Misoku is open takes effect within a couple of seconds, without restarting: handy when another tool (such as a desktop theme switcher) rewrites the palette.

To use themes from a different folder (or a single palette file), set `themes_dir` in your config:

- **config.json:** `{"theme": "onedark", "themes_dir": "~/my-themes"}` — a directory of `*.json` palettes, or one `.json` file
- **flag:** `misoku --themes-dir ~/my-themes` (overrides the config value)

Example `~/.config/misoku/themes/minimal.json`:

```json
{
  "light": false,
  "fg": "#d8dee9",
  "bg": "#2e3440",
  "darker_bg": "#242933",
  "line": "#3b4252",
  "blue": "#88c0d0"
}
```

## Uninstall

**Your notes are never deleted**: the notes folder (`~/Notes`, say) and any
repository on GitHub stay as they are. Uninstalling removes only Misoku, the
gh the installer put next to it, the `PATH` line the installer added, and
Misoku's settings.

1. **Optional: sign gh out of GitHub** first, while gh is still there:

   ```sh
   gh auth logout
   ```

   (if it says you are not logged in, skip this step)

2. **Remove misoku, and gh only if the installer put it there** (it leaves
   a `.misoku-gh-version` file next to the gh it installs; a gh without it
   is yours, from Homebrew, apt or winget, and stays). If misoku was already
   somewhere else on your `PATH` when you installed, the installer updated
   it there (`command -v misoku` shows where): use that folder instead.

   macOS (`/usr/local/bin`; it asks for your Mac's password):

   ```sh
   [ -f /usr/local/bin/.misoku-gh-version ] && sudo rm -f /usr/local/bin/gh /usr/local/bin/.misoku-gh-version
   sudo rm -f /usr/local/bin/misoku
   ```

   Linux, or a Mac where the installer could not use your password
   (`~/.local/bin`):

   ```sh
   [ -f ~/.local/bin/.misoku-gh-version ] && rm -f ~/.local/bin/gh ~/.local/bin/.misoku-gh-version
   rm -f ~/.local/bin/misoku
   ```

   Windows (PowerShell), which also takes the folder off your user `PATH`:

   ```powershell
   $dir = Join-Path $env:LOCALAPPDATA 'Programs\misoku'
   if (Test-Path "$dir\.misoku-gh-version") { Remove-Item "$dir\gh.exe", "$dir\.misoku-gh-version" -Force }
   Remove-Item "$dir\misoku.exe" -Force
   $p = [Environment]::GetEnvironmentVariable('Path', 'User')
   [Environment]::SetEnvironmentVariable('Path', ((($p -split ';') | Where-Object { $_ -and $_ -ne $dir }) -join ';'), 'User')
   ```

3. **macOS and Linux: remove the `PATH` line**, if the installer added one.
   It does so only when its folder was not on your `PATH`, and says
   `Added … to your PATH in …` with the file. The file depends on your shell:
   `~/.zshrc` (zsh), `~/.bashrc` (bash on Linux), `~/.bash_profile` (bash on
   macOS), `~/.config/fish/config.fish` (fish), or `~/.profile` (any other).
   Delete the line that ends in `# added by the Misoku installer`:

   ```sh
   export PATH="$HOME/.local/bin:$PATH" # added by the Misoku installer
   ```

   In fish it reads `set -gx PATH "$HOME/.local/bin" $PATH # added by the Misoku installer`.

4. **Remove Misoku's settings**, which include `notes_dir`, the notes folder
   Misoku remembers, as well as `sync.vaults`, your shortcuts and your
   themes. Delete the `misoku` settings folder:
   - **Linux:** `~/.config/misoku`
   - **macOS:** `~/Library/Application Support/misoku`; in the Terminal,
     `rm -rf ~/Library/Application\ Support/misoku`
   - **Windows:** `%APPDATA%\misoku`

   To keep the other settings for a later install, delete only the
   `"notes_dir"` entry of `config.json` in that folder (and the comma that
   joins it to the entry before or after it).

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

## Credits

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss) (the editor is adapted
from [Bubbles](https://github.com/charmbracelet/bubbles)). Theme palettes are
MIT-licensed community palettes, syntax highlighting by
[chroma](https://github.com/alecthomas/chroma).

## License

[MIT](LICENSE)
