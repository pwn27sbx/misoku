# Configuration, Appearance & Uninstall

## Config File Location

| OS | Config file | Custom themes directory |
| --- | --- | --- |
| Linux | `~/.config/misoku/config.json` | `~/.config/misoku/themes/<name>.json` |
| macOS | `~/Library/Application Support/misoku/config.json` | `~/Library/Application Support/misoku/themes/<name>.json` |
| Windows | `%APPDATA%\misoku\config.json` | `%APPDATA%\misoku\themes\<name>.json` |

## Workspace Selection Order

Misoku opens the first of these that is set:

1. Explicit path: `misoku ~/notes` or `misoku --dir ~/notes`
2. Environment variable: `MISOKU_WORKSPACE`
3. `notes_dir` in `config.json` (set by the first-run welcome screen)
4. Current working directory

## Daily Notes & Templates

- **Daily notes (`space o d` or `:daily`):** defaults to `Daily/YYYY-MM-DD.md` in English (or `Diario/YYYY-MM-DD.md` in Spanish, or if your vault already contains a `Diario/` folder). Override with `"daily_dir": "Journal"` in `config.json`.
- **Templates (`/template` or `/plantilla`):** defaults to `Templates/` in English (or `Plantillas/` in Spanish, or if your vault already contains a `Plantillas/` folder). Override with `"templates_dir": "Templates"`. Supports `{{date}}`/`{{fecha}}` and `{{title}}`/`{{titulo}}`, merges front matter, and is excluded from `:tags` and status filters.

## Appearance (`ui` section)

Open `:settings` (or press `c` on the home screen) to configure common UI options interactively, or edit `config.json` directly:

```json
{
  "theme": "catppuccin",
  "ui": {
    "heading_icons": ["① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "],
    "heading_background": true,
    "bullets": ["●", "○", "◆", "◇"],
    "checkbox": { "unchecked": "󰄱", "checked": "󰱒", "todo": "󰥔" },
    "mode_icon": "󱗞",
    "border": "single",
    "modified_mark": "[+]",
    "transparent": false,
    "cursor_shape": "auto",
    "images": "auto",
    "language": "en"
  }
}
```

| Key | Description |
| --- | --- |
| `heading_icons` | Icons for H1–H6; empty omits icon |
| `heading_numbers` | `true` numbers headings (`1`, `1.1`, `2`) in preview without modifying Markdown |
| `heading_background` | Full-width coloured bands for headings (disabled when `transparent: true`) |
| `bullets` | Cycling list bullets by depth |
| `checkbox.unchecked` / `checked` / `todo` | Glyphs for `- [ ]`, `- [x]`, `- [-]` |
| `mode_icon` | Single statusline icon for every mode |
| `border` | `rounded` (default) or `single` |
| `modified_mark` | Prefix for dirty tabs (e.g. `"[+]"`) |
| `accordion` | `true` (default) keeps one explorer folder open per level |
| `explorer_hide_width` | Column threshold below which explorer overlays (default `120`) |
| `cursor_shape` | `"auto"` (default: block in NORMAL/VISUAL, bar in INSERT) or `"block"`. Override via `MISOKU_CURSOR_SHAPE` |
| `link_suggestions` | `"inline"` (default) or `"finder"` when typing `[[` |
| `images` | `"auto"` (Kitty graphics in Ghostty/kitty/WezTerm/herdr, half-blocks elsewhere), `"kitty"`, or `"blocks"`. Override via `MISOKU_IMAGES` |
| `transparent` | `true` leaves base backgrounds to your terminal opacity/blur |
| `language` | `"en"`, `"es"`, or omitted for system language |

## Custom Themes

Drop a JSON palette file into your custom themes directory (or point `"themes_dir"` / `--themes-dir` at a folder or single `.json` file). Live reloads within seconds when modified on disk:

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

## Images, Flowcharts, Equations & PDF Export

- **Images:** `ctrl+v` in INSERT mode (or `space i`, `/Paste image`, or `misoku paste-image`) saves clipboard images under `assets/` next to the note. Needs `wl-clipboard` (Wayland), `xclip` (X11), or `pngpaste`/AppleScript (macOS).
- **Flowcharts:** fenced `flow` or `mermaid` blocks (`graph TD` / `graph LR`).
- **Equations:** fenced `math` blocks or `$$ … $$`, rendered in Unicode.
- **Columns:** `~~~columns` blocks separated by `+++` (from `/2 columns`, `/3 columns`, `/4 columns`).
- **PDF export:** `:export` (or `space P`) inside Misoku, or `misoku export note.md [-o out.pdf]` from the shell.

## Uninstall

Your notes folder and any GitHub repository are never deleted.

1. Optional: sign `gh` out (`gh auth logout`).
2. Remove `misoku` (and `gh` only if `.misoku-gh-version` exists next to it):

   **macOS (`/usr/local/bin`):**
   ```sh
   [ -f /usr/local/bin/.misoku-gh-version ] && sudo rm -f /usr/local/bin/gh /usr/local/bin/.misoku-gh-version
   sudo rm -f /usr/local/bin/misoku
   ```

   **Linux (`~/.local/bin`):**
   ```sh
   [ -f ~/.local/bin/.misoku-gh-version ] && rm -f ~/.local/bin/gh ~/.local/bin/.misoku-gh-version
   rm -f ~/.local/bin/misoku
   ```

   **Windows (PowerShell — preserves `%VAR%` entries in `PATH`):**
   ```powershell
   $dir = Join-Path $env:LOCALAPPDATA 'Programs\misoku'
   if (Test-Path "$dir\.misoku-gh-version") { Remove-Item "$dir\gh.exe", "$dir\.misoku-gh-version" -Force }
   Remove-Item "$dir\misoku.exe" -Force
   $k = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $true)
   try {
       $p = [string]$k.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
       $kind = $k.GetValueKind('Path')
       $new = ((($p -split ';') | Where-Object { $_ -and $_ -ne $dir }) -join ';')
       $k.SetValue('Path', $new, $kind)
   } finally { $k.Close() }
   ```

3. On Linux/macOS, remove the `# added by the Misoku installer` line from your shell rc file if added.
4. Remove Misoku's config directory (`~/.config/misoku`, `~/Library/Application Support/misoku`, or `%APPDATA%\misoku`).
