<div align="center">

# Misoku

### ~ Terminal Markdown Notebook ~

A block-based notebook for your terminal.
Your notes stay plain Markdown files.

<p>
  <a href="https://github.com/pwn27sbx/misoku/actions/workflows/ci.yml"><img src="https://github.com/pwn27sbx/misoku/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
</p>

</div>

## Installation & Update

There are no release binaries yet. Build it from source with Go:

```sh
git clone https://github.com/pwn27sbx/misoku
cd misoku
go build -o ~/.local/bin/misoku ./cmd/misoku
```

To update, `git pull` and run the same `go build`.

Misoku is a single binary with no dependencies. It runs on macOS (Intel and
Apple Silicon), Windows 10/11 and any x86-64 Linux distribution. On other
CPUs (e.g. Raspberry Pi), install with Go.
A [Nerd Font](https://www.nerdfonts.com/) is recommended for icons.

## Quick start

```sh
cd ~/notes        # any folder; your .md files show up in the explorer
misoku
```

Press `?` inside the app to see every shortcut.

```sh
misoku --dir ~/notes          # open a specific folder
misoku --theme <name>         # pick a theme for this session
misoku --list-themes
misoku --themes-dir <path>    # custom themes folder or single .json file
misoku export note.md         # write note.pdf next to the note (-o to pick the file)
```

## Features

- **Plain Markdown files.** No database or lock-in: use git, sync the folder, open notes in any editor.
- **A full workspace.** Start screen, file tree, tabs, statusline and 97 themes.
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
- **Fast search** across file names and note contents.
- **Live explorer**: notes and folders created, deleted or renamed by other programs (another editor, git, a sync client) show up on their own within a couple of seconds.
- **Open notes stay in sync with the disk**: a note changed by another program reloads in place (the cursor stays put). If you have unsaved edits it is left alone, you are told, and saving asks before overwriting the other version.
- **Safe by default**: notes are saved atomically (a crash mid-save never leaves half a file), and deleting moves them to the Trash on macOS and Linux (Windows still deletes for good).
- **Links between notes** with `[[Note]]`: jump to headings and blocks, embed one note in another, and see what links back.
- **Graph view** (`:graph`): a zoomable map of how your notes link together, with a live preview beside it.

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
lists the keys you configured. A misspelled action, a malformed key, or a key
given to two actions is reported in the status bar when Misoku starts.

Vim editing keys (`hjkl`, `w`, `dd`, …) are not configurable.

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
| `themes` | `space t` | Themes |
| `graph` | `space g` | Graph of this note's links |
| `home` | `space d` | Home screen |
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
| `explorer.toggle_fold` | `z`, `space`, `tab` | Fold / unfold |
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

The graph is used with the mouse:

- Click a note to preview it; click it again to open it.
- Scroll to zoom out and in: zoomed out, the map also shows the links of
  those notes, up to four steps away. The `−` and `+` buttons do the same.
- Drag to move around the map; `reset` puts it back.
- `centre` re-centres the map on the selected note; `back` returns to the
  previous one.

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

In the editor, `:export` writes the open note as a PDF next to it. `:export ~/Desktop/` or `:export report.pdf` picks another place. From the shell:

```sh
misoku export notes/plan.md               # notes/plan.pdf
misoku export notes/plan.md -o plan.pdf
```

The PDF is a printable A4 document with headings, lists, tables, syntax-highlighted code, equations and images. Boards, calendars, timelines, charts and flowcharts print in colour as they look in the preview.

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

Your notes are never touched. Remove the binary the way you installed it:

```sh
rm -f /usr/local/bin/misoku ~/.local/bin/misoku   # macOS / Linux
```

On Windows, delete `%LOCALAPPDATA%\Programs\misoku` and remove it from your
`PATH`. Settings are in `~/.config/misoku` (macOS: `~/Library/Application Support/misoku`,
Windows: `%APPDATA%\misoku`).

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
