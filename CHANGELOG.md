# Changelog

The changes of each release, newest first. Earlier releases are on the
[releases page](https://github.com/pwn27sbx/misoku/releases); the binary's
version comes from the tag, so nothing else in the tree carries it.

## [Unreleased]

### Added

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
