# Changelog

The changes of each release, newest first. Earlier releases are on the
[releases page](https://github.com/pwn27sbx/misoku/releases); the binary's
version comes from the tag, so nothing else in the tree carries it.

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
