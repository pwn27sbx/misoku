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

```sh
misoku ~/notes                # open a specific folder (same as --dir ~/notes)
misoku                        # no folder: $MISOKU_WORKSPACE, else notes_dir in your config, else this folder
misoku --theme <name>         # pick a theme for this session
misoku --list-themes
misoku --themes-dir <path>    # custom themes folder or single .json file
misoku --version              # print the version
misoku --help                 # every command and flag (also -h)
misoku export note.md         # write note.pdf next to the note (-o to pick the file)
misoku sync setup             # sync this folder through a private GitHub repository (see Sync)
misoku sync init <url>        # sync this folder through any git repository, SSH included
```

`misoku sync setup` and `misoku sync init` set up the folder they are
pointed at: `--dir`, else `$MISOKU_WORKSPACE`, else the current folder. They
never use `notes_dir`.

## Features

- **Plain Markdown files.** No database or lock-in: use git, sync the folder, open notes in any editor.
- **A full workspace.** Start screen, file tree, tabs, statusline and 97 themes.
- **Folders on the start screen**: the top-level folders of your notes, one key away (they follow the Recent notes on 1–9); opening one shows it expanded in the explorer.
- **A start screen that follows your shortcuts**: its letter keys stay fixed (`n`, `f`, `e`, `t`, `?`, `g` for the graph, `s`, `q`, `1`–`9`), and the leader (`space`) sequences and Ctrl/Alt shortcuts from your [`keys`](#configuration) work there too.
- **Modal keyboard editing** with mouse support, plus unsaved tabs and Save As.
- **`/` block menu** for headings, to-dos, tables, callouts, code, images and more.
- **Live preview** laid out like a page, with syntax highlighting, images and cover banners drawn in the terminal.
- **Drag blocks to reorder them** in the preview by their `⠿` handle, or add one below with `+`, like Notion.
- **Boards, calendars, timelines, charts and forms**, stored as text and editable from the preview.
- **Flowcharts** from Mermaid syntax (`graph TD` / `graph LR`), drawn with real shapes: boxes, decisions, circles, databases.
- **Equations** in LaTeX (`/equation`, a `math` block or `$$…$$`), shown as Unicode: `x^2` → `x²`, `\alpha` → `α`.
- **Export to PDF** with `:export`: a clean printable document, with boards, charts and flowcharts included.
- **2, 3 and 4 column layouts** from the `/` menu.
- **Resizable panes**: drag the dividers between the explorer, editor and preview.
- **Narrow windows** (under 120 columns): the explorer starts hidden and `toggle_explorer` (`ctrl+b` by default) opens it over the editor, which keeps its width; choosing a note, clicking outside it or `Esc` closes it. Wider windows keep your last toggle until the width crosses 120 again.
- **Reading mode** (`space r`): hide the editor and read the note in the preview at full width; the statusline says READING.
- **Fast search** across file names and note contents.
- **Live explorer**: notes and folders created, deleted or renamed by other programs (another editor, git, a sync client) show up on their own within a couple of seconds.
- **Open notes stay in sync with the disk**: a note changed by another program reloads in place (the cursor stays put). If you have unsaved edits it is left alone and you are told; saving then keeps both versions: yours in the note, the one on disk in a conflict copy next to it (`note (conflict 2026-10-16 0905).md`), and the status bar says `Both kept: …` with the copy's name. Nothing is asked and nothing is lost.
- **Open notes deleted or renamed elsewhere are kept**: the tab stays open with its text and you are warned once; saving asks before recreating the note at its old path, so a rename is never silently duplicated.
- **Safe by default**: notes are saved atomically (a crash mid-save never leaves half a file), and deleting moves them to the Trash on macOS and Linux (Windows still deletes for good).
- **Links between notes** with `[[Note]]`: jump to headings and blocks, embed one note in another, and see what links back.
- **Messages you can read again**: errors stay in red in the status bar until you press `Esc` (which still does what it always does); other messages clear on the next key. `:messages` lists the last 100, with the time of each (results only: progress such as `Syncing…` is not kept).
- **Graph view** (`:graph`): a zoomable map of how your notes link together, with a live preview beside it.
- **Sync between computers** through a private git repository of your own, with no Misoku server: `:sync setup` (or `s` on the start screen, or `misoku sync setup`) once per computer connects a private GitHub repository for you; `misoku sync init <url>` works with any git host. Then it syncs on its own, and with `:sync` or `space s`. Conflicting edits keep both versions, so nothing is lost ([Sync](#sync)).

## Shortcuts

Every shortcut can be changed in `config.json` (`~/.config/misoku/config.json`,
`~/Library/Application Support/misoku/config.json` on macOS), in a `keys`
section. Listing an action replaces all of its keys; an empty list turns it
off:

```json
{
  "keys": {
    "toggle_explorer": ["space e"],
    "focus_explorer": [],
    "close_tab": ["space q", "ctrl+w"]
  }
}
```

Write keys the way the terminal names them (`ctrl+b`, `tab`, `shift+tab`,
`G`, `\\`), or as a leader sequence: `space e` is Space, then `e`. Ctrl and
Alt shortcuts of `quit`, `save`, `new_note`, `find_note` and
`toggle_explorer` also work while typing; every other shortcut only works
outside INSERT mode, so it never gets in the way of your text. The `?` help
lists the keys you configured. Press Space and pause: after 400 ms a guide
lists the keys that can follow, in three groups (NOTE, VIEW, GO), with the
keys you configured. `Esc` closes it; type the next key sooner and it never
shows. On the home screen, a key that needs an open note (such as `space p`)
says so in the status bar. A misspelled action, a malformed key, or a key given to two actions is
reported in the status bar when Misoku starts (it stays until `Esc`, and
`:messages` shows it again).

Vim editing keys (`hjkl`, `w`, `dd`, …) are not configurable.

**Undo** (editor, NORMAL mode, not configurable)

| Key | What it does |
| --- | --- |
| `u` | Undo the last change; the status bar says which one (`1 change; before #3`) |
| `Ctrl+R` | Redo what `u` undid |

Each INSERT session is one change, and so is each NORMAL command (`dd`, `p`,
`x`) and each edit made from the preview, a table or the `/` menu. Every tab
keeps its own history, in memory only: up to 200 changes or 8 MB, the oldest
dropped first. Undoing never writes to disk, and a note reloaded because it
changed on disk starts a new history.

**General**

| Action | Default keys | What it does |
| --- | --- | --- |
| `new_note` | `ctrl+n`, `space n` | New note (unsaved tab) |
| `save` | `ctrl+s`, `space w` | Save (new notes: pick folder) |
| `find_note` | `ctrl+p`, `\`, `space f` | Find note (opens in this tab) |
| `toggle_explorer` | `ctrl+b` | Toggle explorer |
| `focus_explorer` | `space e` | Focus explorer |
| `next_pane` | `tab` | Next pane |
| `prev_pane` | `shift+tab` | Previous pane |
| `pane_left` | `ctrl+h` | Pane to the left |
| `pane_right` | `ctrl+l` | Pane to the right |
| `next_tab` | `]`, `space tab` | Next tab |
| `prev_tab` | `[`, `space shift+tab` | Previous tab |
| `close_tab` | `space x` | Close tab |
| `toggle_preview` | `space p` | Toggle preview |
| `reading_mode` | `space r` | Reading mode: only the preview, full width (`space r` or `space p` brings the split back) |
| `export_pdf` | `space P` (Space, then Shift+P) | Export note as PDF (same as `:export`) |
| `themes` | `space t` | Themes |
| `graph` | `space g` | Graph of this note's links |
| `home` | `space d` | Home screen |
| `sync_now` | `space s` | Sync the vault with git now (same as `:sync`, see [Sync](#sync)) |
| `help` | `?`, `space h`, `space ?` | This help |
| `quit` | `ctrl+c` | Quit (asks to save) |

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
| `explorer.preview` | `o` | Preview note (folder: open) |
| `explorer.expand` | `l`, `right` | Expand / open |
| `explorer.collapse` | `h`, `left` | Collapse / go to parent |
| `explorer.toggle_fold` | `z` | Fold / unfold |
| `explorer.collapse_all` | `W` | Collapse all |
| `explorer.new_note` | `n`, `%` | New note in this folder |
| `explorer.new_sub_note` | `a` | New sub-note |
| `explorer.rename` | `r`, `e` | Rename |
| `explorer.delete` | `d`, `x`, `delete` | Delete (asks first) |
| `explorer.find` | `\`, `f` | Find note |

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
| `heading_background` | Paints each heading on a full-width band of its color (the H1 underline is then left out) |
| `bullets` | List bullets by depth, cycling |
| `checkbox.unchecked`, `checkbox.checked` | Marks for `- [ ]` and `- [x]` |
| `checkbox.todo` | Mark for `- [-]` (in progress). Without it, `[-]` stays plain text |
| `mode_icon` | One icon for every mode in the statusline, like lualine (`"󱗞"`) |
| `border` | Frame of dialogs, menus and preview blocks: `rounded` (default) or `single` |
| `modified_mark` | Shown before the tab name of a note with unsaved changes, like incline (`"[+]"`) |
| `accordion` | `true` (default): opening a folder in the explorer closes its open siblings. `false` keeps several open |
| `explorer_hide_width` | Terminal width, in columns, below which the explorer hides (default `120`) |

### Settings menu

`:settings` (or **Settings**, key `c`, on the home screen) opens a menu for
four of these keys: accordion folders (on or off), the explorer hide width,
heading icons and bullets (one of the presets Misoku, Numbers, Arrows and
Plain; Misoku removes both keys), and the dialog border. A value picked with Enter applies at once and is saved at once
in `config.json`: only that key changes, and every other setting stays as you
wrote it. Esc goes back without saving, and picking the value already in use
writes nothing. If the file cannot be written, the status bar says why.

## Links

Type `[[` to link to another note. Click a link in the preview, or press
`gd` on it in the editor, to follow it.

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
link to it at the bottom of the preview. Web links (`[text](https://…)` and
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
| `offline · N pending` | The repository could not be reached; N commits wait to be pushed |
| `sync will retry` | The sync stopped short; the next one tries again |
| `sync paused` | Something needs you first; nothing was touched |
| `sync paused, lock left · :sync status` | A git lock file stopped the sync (see [Troubleshooting](#troubleshooting)) |
| `sync error` | The repository refused, or git failed |
| `other window syncs` | Another Misoku window syncs this vault |

A sync that starts on its own only writes to the status line when it made conflict copies, reloaded an open note, or hit a new problem; otherwise the indicator says how it went.

`:sync status` shows the detail: state, remote (without credentials), branch, how often timed sync runs, commits waiting to be pushed, when it last synced, the last error, the conflict copies on disk, and the files that do not sync as they are.

### What does not sync

| What | Why |
| --- | --- |
| Files over **50 MB** | GitHub refuses files over 100 MB, which would block every sync. A file that grows past the limit keeps its last committed version. |
| Files your `.gitignore` ignores | Ignored files are respected |
| Misoku's temporary save files and `.DS_Store` | Listed in `.git/info/exclude` |
| A folder with its own git repository inside the vault | git would only record which commit it is on, not its files. Move it out of the vault, or ignore it. A git submodule syncs only its commit id. |
| What a symbolic link points to outside the vault | Only the link itself syncs |
| New files with a name Windows cannot use (`:`, `*`, `?` and similar, a name ending in a dot or a space, `CON`, `NUL`…) | They would stop every sync on a Windows computer. They stay out until you rename them; meanwhile, deletions are held back, so a note renamed to such a name keeps syncing under its old one. |

Everything else syncs, including the PDFs `:export` and `misoku export` write into the vault. Each of these files gets one line, saying what to do, in `:sync status` and in the output of `misoku sync init`.

### Troubleshooting

| You see | What to do |
| --- | --- |
| `offline, N pending` | Nothing to do: the commits are safe in the vault and the next sync, once the network is back, pushes them. |
| `origin's SSH host key is not known yet` | Run `misoku sync init <url>` in a terminal once and accept the key. |
| `origin refused your SSH key` | Add your key to your account on the git host (and to `ssh-agent`). |
| `could not sign in to origin` | For GitHub over HTTPS run `gh auth setup-git` (or `misoku sync setup`, which runs it), or set up your git credentials. If it still fails, run `misoku sync init <url>` in a terminal. |
| `origin's SSH host key changed` | It can mean an attack or a reinstalled server: check the key with `ssh` in a terminal before syncing again. |
| `another Misoku window syncs this vault` | Only one window syncs a vault; the others see its changes within seconds. When that window closes, the next sync here takes over. |
| `… is in progress in this vault` | A merge, rebase, cherry-pick or revert you started is waiting: finish it (`git merge --continue`…) or cancel it (`git merge --abort`…). Sync waits and touches nothing. |
| `HEAD is detached` | Check out your branch, for example `git switch main`. |
| `a git lock file is in the way (.git/index.lock)` | A git command is running in the vault, or one stopped half-way. If no git is running, open `:sync status` and choose **Remove the lock and sync** (with the arrow keys or Tab, then Enter). If you typed before the status appeared, it leaves the choice out: open it again. Misoku removes a lock on its own only when it knows it left it itself. |
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
