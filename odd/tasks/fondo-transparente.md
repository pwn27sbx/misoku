# fondo-transparente — leave the base background to the terminal, like Claude Code in herdr

Branch `feat/fondo-transparente` (from origin/main 02da7c4). Test runner: `go test ./...` (`make` = fmt + vet + test
+ build); colour assertions force `lipgloss.SetColorProfile(termenv.TrueColor)` and restore it in `t.Cleanup`.
Delivery: small conventional commits, PR, CI green on Linux/macOS/Windows, CHANGELOG `[Unreleased]`.

## Specs

- **S1. The goal.** L1: "misoku siugiente update grande que pueda tener la trasnparencia blur y eso como herdr para
  que se vea tal cual herdr". With the option on, Misoku's base areas are drawn with the terminal's DEFAULT
  background (no background SGR), exactly like Claude Code, so a terminal with opacity/blur (Ghostty
  `background-opacity = 0.8` + blur, herdr-app's terminal, Kitty/Alacritty on Omarchy + Hyprland) shows the
  blurred wallpaper through them. Nothing terminal-specific: Misoku only stops painting.
- **S2. One option, in config.** `ui.transparent` (bool) in `config.json`, default `false`. It is a fact about the
  terminal, not the palette, so it applies to all 97 bundled themes and to user themes alike. No theme key.
  (Tradeoff in the Log, E6.)
- **S3. Settings menu + live.** A fifth row in the Settings menu (`:settings`, key `c` on the home screen):
  "Transparent background: on/off". Enter toggles it, applies at once (no restart) and saves only that key, like
  the accordion row. Hand edits to `config.json` apply on the next start, like every other `ui` key (config.json
  is not watched). A user-theme reload (Omarchy theme switch) keeps the transparency.
- **S4. What becomes transparent.** The large fills: `Bg` (editor, preview, active tab, every padding painted by
  the root `ui.Paint`), `DarkerBg` where it fills a pane (explorer, tab line) and `StatusBg` (statusline fill).
  Kept opaque so they stay visible: cursor line and input fields (`Bg2`), selections and hover rows (`OneBg`,
  `OneBg2`, `OneBg3`), the statusline file block (`LightBg`), mode badges, pills and accent chips, and every
  floating box (dialogs, menus, finder, palette, leader guide, date picker, toast, slash menu, media browser,
  theme picker). Assumption: herdr/Claude Code paint nothing behind ordinary text and keep only highlights, so
  this matches; panes stay delimited by the existing `│` dividers.
- **S5. Default is untouched.** With `ui.transparent` unset/false every frame is byte-for-byte what it is today.
- **S6. Nothing becomes invisible.** Text drawn in the background colour on an accent (mode badge, today mark,
  board pills, calendar day, confirm button, date picker selection) keeps the theme's dark hex. Colour maths that
  blends toward the background (graph fades, heading bands, image/cover compositing, theme-picker swatch) keeps
  the theme's opaque hex.
- **S7. Welcome screen.** The first-run welcome program (`cmd/misoku/welcome_screen.go`) honours the option too.
- **S8. Docs.** README `## Look` table row + Settings-menu paragraph ("four of these keys" becomes five) and a
  short note: the terminal's own background colour shows through, so pick a theme that matches it (Omarchy sets
  both); a terminal without opacity just shows its default colour. CHANGELOG `[Unreleased]` → Added.
- **S9. Portability.** Pure Go, no build tags; behaviour is identical on macOS, Linux and Windows (no OS check:
  "transparent" is only the absence of an SGR code).

## Tasks

- **T1** (S4, S5, S6) — theme: two new fields and one method, no caller changes yet. Route: TBD. Status: done (47123c4).
  - `internal/theme/theme.go`: add `Base lipgloss.Color` (always the palette's opaque `bg`) and `PopupBg
    lipgloss.Color` (always `darker_bg`) in `Get` (`:272-318`); add `func (t Theme) Transparent() Theme`, a copy
    with `Bg`, `DarkerBg`, `StatusBg` set to `""`, leaving `Base`, `PopupBg`, `DarkFg` and every other shade
    (sibling of `Greyed`, `:321-332`). Doc comments on the exported names.
  - `internal/theme/theme_test.go`: RED first — `Transparent()` empties exactly those three, keeps `Base`,
    `PopupBg`, `DarkFg`, `Bg2`, `OneBg*`, `LightBg`; `Greyed()` of a transparent theme stays transparent; a style
    with `Background(lipgloss.Color(""))` renders with no `48;` under TrueColor (pins the lipgloss/termenv
    behaviour of E3).
  - Checks: `go test ./internal/theme/`, `gofmt -l .`, `go vet ./...`. ~60 lines.
- **T2** (S5, S6) — callers stop using `Bg`/`DarkerBg` for the jobs that must stay opaque. Pure refactor: default
  output identical. Route: TBD. Status: done (a367251).
  - Text-on-accent → `DarkFg`: `internal/app/bars.go:235,245,291`, `internal/app/datepicker.go:158`,
    `internal/app/dialogs_confirm.go:97`, `internal/preview/block_calendar.go:119`,
    `internal/preview/block_board.go:78`, `internal/preview/block_timeline.go:131`.
  - Colour maths → `Base`: `internal/globalgraph/render.go:446`, `internal/preview/compiler.go:465`,
    `internal/preview/compiler_render.go:191,211`, `internal/app/themes.go:241` (swatch chip).
  - Floating boxes `DarkerBg` → `PopupBg`: `internal/app/dialogs.go:213`, `internal/app/menu.go:318`,
    `internal/app/finder_view.go:19,77,83,131`, `internal/app/palette.go:280`, `internal/app/themes.go:206,231`,
    `internal/app/media_browser.go:230`, `internal/app/datepicker.go:131`, `internal/app/viewaction.go:85`,
    `internal/app/clipboard.go:58-61`, `internal/app/leader_guide.go:81`, `internal/app/dialogs_confirm.go:15,22,73`,
    `internal/content/slash_view.go:82,108`. Panes keep `DarkerBg`: `internal/sidebar/sidebar_view.go:31,69`,
    `internal/app/bars.go:73`.
  - Before editing, re-grep `DarkerBg|\.Bg\b|StatusBg` (non-test) and classify any hit not listed here.
  - Checks: `go test ./...` must pass unchanged (existing goldens in `internal/globalgraph/testdata/*.golden`
    and `internal/textarea/testdata/*.golden` included); differential: render a few frames (home, editor +
    sidebar, palette, a confirm dialog) under TrueColor at the base commit (copy in `mktemp -d`) and on the branch,
    byte-compare. ~80 lines.
- **T3** (S2, S3, S4, S7) — config, Settings row, live apply. Route: delegated (writer agent, with S8 docs). Status: done, uncommitted.
  - `internal/config/config.go:191-230`: `Transparent bool \`json:"transparent,omitempty"\`` in `UI` with doc.
    `internal/config/settings.go:13-19`: `UITransparent = "transparent"`.
  - `internal/app`: the model remembers the flag (from `app.WithUI`, `cmd/misoku/main.go:139`); `applyTheme`
    (`internal/app/themes.go:19`) applies `th.Transparent()` when on, so the theme picker, `setThemeByName` and
    the user-theme reload (`internal/app/watch.go:222-230`) all inherit it. Keep `m.theme.Name` lookups working
    (they call `theme.Get`, which stays opaque). `internal/app/settings.go`: `settingTransparent` row after
    `settingBorder` (`:49-54`), toggled in `chooseSetting` like `settingAccordion` (`:110-116`), re-applies the
    current theme and `saveUI(map[string]any{config.UITransparent: on})`; `false` writes `nil` (key removed =
    default), mirroring how presets remove keys.
  - `cmd/misoku/welcome_screen.go:247,307,310`: apply `Transparent()` to its theme when the flag is on.
  - Tests, RED first, through the harness (`internal/app/app_test.go:56`) with TrueColor forced (pattern:
    `internal/app/theme_reload_test.go`, `internal/sidebar/dim_test.go:59-60`): (a) with `UI{Transparent:true}`
    the frame contains no `48;2;R;G;B` for the theme's `bg`, `darker_bg` or `status_bg`, and still contains
    `OneBg2`/`Bg2` where the explorer selection / cursor line is; (b) default: the `bg` SGR is present (S5);
    (c) a confirm dialog or the palette open over a transparent editor still paints `PopupBg`; (d) the Settings
    row toggles, the frame changes at once, and `config.json` gets/loses only `ui.transparent` (extend
    `internal/app/settings_test.go`; `internal/config/settings_test.go` for the key); (e) after a user-theme
    file rewrite (`theme_reload_test.go` style) transparency survives; (f) the mode badge text keeps the `DarkFg`
    hex (`38;2;…`). Previous behaviour pinned: accordion and border rows unchanged.
  - Checks: `go test ./...`, `gofmt -l .`, `go vet ./...`. ~180 lines with tests.
- **T4** (S8) — docs. Route: TBD. `README.md` `## Look` example + table (`:383-415`), `### Settings menu`
  paragraph (`:427-435`), the short "your terminal shows through" note; `CHANGELOG.md` `[Unreleased]` Added.
  No help-text/cheatsheet change: no new key binding (`internal/app/modal.go` untouched). No dotfiles template
  change needed (S2): Sebastián sets `"ui": {"transparent": true}` once in his `config.json`. Checks: read back;
  `go test ./...` (README isn't tested, but run once before the commit). ~30 lines.
- **T5** — manual check, not a commit: `go run ./cmd/misoku --dir <scratch>` in Ghostty (opacity 0.8 + blur)
  inside herdr, toggle with `c` → Settings; compare side by side with a Claude Code pane; repeat in herdr-app and,
  when available, Omarchy/Hyprland. Record result in Log.

Order: T1 → T2 → T3 → T4 (T2 depends on T1 fields; T3 on both). ~350 authored lines total, one PR.

## Log

- **L1** (Sebastián, 2026-11-05): "misoku siugiente update grande que pueda tener la trasnparencia blur y eso como
  herdr para que se vea tal cual herdr"
- **L2** (Sebastián, 2026-11-05): "si armas el plan lo verificas yluego un agente lo verifica y lo empiezas"
- **E1. Cause (verified earlier in the session).** Ghostty keeps cells with an explicit background opaque; with
  `background-opacity-cells = true` both layers apply at 0.8 (~96% cover), so Misoku looks near-opaque next to
  Claude Code, which paints no background.
- **E2. Where the base Bg comes from.** One central painter: `internal/app/app_view.go:107`
  `ui.Paint(screen, …Foreground(m.theme.Fg).Background(m.theme.Bg))`; `ui.Paint` (`internal/ui/ui.go:143-158`)
  prefixes every line and re-applies the prefix after every reset, so all unstyled cells and padding get `Bg`.
  Only 21 non-test refs to `.Bg` exist (98 non-test `Background(` calls; 260 incl. tests). Panes fill with
  `DarkerBg` (`sidebar_view.go:31,69`, `bars.go:73`) and `StatusBg` (`bars.go:201,287,350,371`).
- **E3. Mechanism verified in the module cache.** `lipgloss.Color("")` is enough; no `NoColor` needed, and the
  `Theme` fields keep their concrete `lipgloss.Color` type (`theme.go:19-64`), which `NoColor{}` could not be
  assigned to. Chain: lipgloss v1.1.0 `color.go:47` `Color.color` → termenv v0.16.0 `profile.go:84-86`
  `Color("")` returns `nil` → `style.go` `Style.Background(nil)` appends nothing → `Styled` with no styles
  returns the text bare. lipgloss `style.go:331-334` only adds bg when set; `whitespace.go:72-74`
  `WithWhitespaceBackground` goes through the same nil-guarded `Background`, so `Place` fills don't force a
  colour. `ui.Paint` still works: its sample keeps the `Fg` SGR, so `i > 0`.
- **E4. Traps found.** `DarkFg = t.Bg` (`theme.go:311`) and 8 `Foreground(th.Bg)` sites (T2 list) would turn
  badge text into the terminal's default fg; colour maths parses `Bg` as `#rrggbb` (`globalgraph/render.go:446`
  via `fade`, `:451-462`, falls back; `preview/look.go:66-74` `band` returns `bg` → empty band), images/covers
  composite on `bg` (`cover.go:42`, `image.go:101`). Dialogs share `DarkerBg` with the explorer (16+ sites), so a
  separate `PopupBg` is needed to keep boxes opaque while the explorer goes clear. `BorderBackground(DarkerBg)`
  (`clipboard.go:58`, `dialogs_confirm.go:22`, `slash_view.go:108`) moves with its box to `PopupBg`.
- **E5. Themes and config.** Bundled palettes are generated (`palettes_gen.go`), user palettes are JSON parsed
  into `palette` (`theme.go:70-99`, `addUserThemeFile` `:224-237`) from `UserThemesDir` (`:150`; macOS
  `~/Library/Application Support/misoku/themes/`). The Omarchy template
  (`~/.dotfiles/omarchy/.config/omarchy/themed/misoku.json.tpl`) maps `bg`/`darker_bg`/`status_bg` to Omarchy's
  backgrounds. User themes live-reload via `UserThemesStamp` (`watch.go:115,217-230`). `config.json`'s `ui`
  section is `config.UI` (`config.go:191-230`), written by the Settings menu through `config.SetUI`
  (`settings.go:42`) / `saveUI` (`internal/app/settings.go:245`); `config.json` itself is not watched.
- **E6. Option tradeoff.** Config toggle (chosen): one switch for every theme, Settings row, live, no template
  change; a light theme over a dark terminal must be avoided by the user. Theme key `"transparent": true`: would
  follow Omarchy theme switches per theme and need no UI, but only user JSON themes could carry it (bundled ones
  are generated), the palette loader would grow a non-colour key, and it couples a terminal fact to a palette.
- **E7. Tests.** Harness: `internal/app/app_test.go:56-160`; colour-profile pattern in 10 test files (e.g.
  `internal/ui/ui_test.go:15`). Goldens exist only for `globalgraph` and `textarea` (`testdata/*.golden`), not for
  app frames: T3 asserts SGR substrings instead of adding goldens.
- **E8. Limitations to state, not fix.** Images and covers keep an opaque rectangle (composited on `Base`); the
  terminal's own default bg colour shows where it has no opacity.
- Next: plan verified by the orchestrator, then by an independent agent (L2), then T1.

### Verification of the plan (independent verifier, 2026-11-05): sound with fixes, applied here
- V1 (T3): `main.go:139` passes `WithTheme` before `WithUI`, and `WithTheme` already runs `applyTheme` (`app.go:182-187`). `applyUI` (`look.go:20-31`) must re-apply the theme once the flag is known, or the first frame comes out opaque.
- V2 (T3): Turning it off cannot reuse `m.theme` or `p.original` (`themes.go:128`), because both are already the transparent copy. Keep the opaque base theme on the model, and re-apply from that.
- V3 (T2, decision): in narrow mode the explorer floats over the editor (`narrow.go:97`). It stays OPAQUE (`PopupBg`), like every other floating box, so the text under it never shows through.
- V4 (T1/T3 tests): pin a test theme whose `bg`, `darker_bg` and `status_bg` hexes differ from `Bg2`, `OneBg*` and `LightBg`, so that the "no 48;2 for these" assertions mean something.
- Notes: `themes.go:241` needs no change, because `t` comes from `theme.Get`, which stays opaque. The `StatusBg` powerline sites (`bars.go:234,293,309,363`) and the active tab (`bars.go:40`) are covered by the field change. With S7 the welcome card (`welcome_screen.go:307`) also turns clear.
- Decision (orchestrator): the statusline and the tab bar are also transparent (S4), because herdr leaves them unpainted too: "para que se vea tal cual herdr" (L1).
- **T3 + S8 evidence (writer agent, 2026-11-05).** Model keeps the opaque `baseTheme`; `applyTheme` draws
  `Transparent()` when `ui.transparent` is on; `applyUI` re-applies when the flag changes (V1); picker `original`,
  picker Enter and the user-theme reload start from `baseTheme` (V2); Settings row 5 toggles live, saves `true` or
  removes the key; `welcomeTheme(name, transparent)` (S7). RED observed (compile, then behaviour after stubs).
  Mutants killed: M1 applyTheme ignores flag and M2 applyUI no re-apply → `TestTransparentOptionClearsTheBaseFills`;
  M3 picker restores `m.theme` → `TestThemePickerKeepsTransparency`; M4 toggle no re-apply, M5 off writes `false` →
  `TestSettingsTransparentTogglesLiveAndIsSaved`; M6 reload paints opaque → `TestUserThemeReloadKeepsTransparency`;
  M7 key renamed → `TestSetUITransparentSavesAndRemovesTheKey`; M8 welcome ignores flag →
  `TestWelcomeScreenHonoursTransparent`; M9 no menu row → `TestSettingsMenuShowsTransparentBackground`.
  Differential with the flag off (onedark, nord, catppuccin × home, editor, cmdline, dd, leader) at a367251 vs
  worktree: byte-identical (288876 bytes). Docs: README `## Look` row + `### Transparent background` note +
  Settings paragraph ("five"), CHANGELOG `[Unreleased]` Added.
