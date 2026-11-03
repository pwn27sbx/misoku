# cursor-shape — a block cursor in NORMAL and a bar in INSERT

Branch `feat/cursor-shape`. Plan: `~/Documents/MisokuVault/Misoku/2026-11-02-cursor-shape/README.md`
(its "Verification log" amendments are binding and override the sections above them).
Test runner: `go test ./...`; scratch under `TMPDIR=/var/tmp`, never `/tmp` and never the real HOME.
Delivery: small conventional commits, gentle-ai review offered in the OpenCode pane, PR, CI green on every
platform, merge commit, CHANGELOG `[Unreleased]` -> `[0.17.0]`, tag `v0.17.0` on main, wait for Release, then
`CURSOR-DONE` with the PR and release URLs.

## Specs

- **S1. Five modes.** NORMAL, VISUAL and VISUAL-LINE put a block cursor at the editor cell; INSERT puts a thin
  bar; COMMAND hides the real cursor while the command line has its own drawn cursor.
  (Plan amendment 1: "NORMAL block, VISUAL block, VISUAL-LINE block, INSERT bar, COMMAND: the real cursor is
  hidden while the command line has focus (its own drawn cursor stays)".)
- **S2. Position source.** The cell is `internal/textarea.CursorScreen()` (relative to the textarea's top-left,
  gutter included) plus the app-level offset: `Layout.EditorX` and `Layout.BodyY` (sidebar, tab bar, borders).
  Correct with the explorer open and closed and with several tabs open.
- **S3. Where the logic lives.** The placement is written by a cursor writer between `kitty.Writer` and
  `syncout.Writer`, so its bytes sit inside the same `ESC[?2026h … l` bracket as the frame, under the same
  lock that keeps an image `Send` atomic. `Fd`, `Read` and `Close` forwarding is kept.
- **S4. No `ESC 7` / `ESC 8`.** After the frame the writer appends `CUP row;col` + DECSCUSR + `?25h`. Frames
  are recognised by the leading cursor-home (`ESC[H`), never "before each write": loose writes exist
  (`execute`, `insertTop`, `setIgnoredLines`).
- **S5. Model state drives it.** Shape and position come from the app model (View), and are re-emitted when
  they change even when the frame text is identical (the renderer skips identical frames). A mode switch emits
  its DECSCUSR once, not on every frame.
- **S6. Fake cursor.** The textarea's drawn cursor (`internal/textarea/textarea.go:1207-1211`, `:1362`,
  placeholder included) is hidden only when the real cursor is active for the editor; the finder's and the
  dialogs' drawn cursors are unchanged. The blink setting stays consistent with a steady real cursor
  (`content.go:155-159`).
- **S7. Exit paths.** `ESC[0 q` (and the cursor shown) is emitted right after `p.Run()`, next to
  `preview.ClearImages()`, before every `os.Exit` that follows. SIGHUP goes through `quitOnHangup`, so it
  passes the same point. There is no ctrl+z/suspend in misoku. `tea.ExecProcess` (`internal/app/sync.go:885`,
  `sync_setup.go:188`) resets the shape before the external program and re-applies it after. A Bubble Tea
  panic recovery and SIGKILL cannot be handled: the README says so honestly. The welcome-screen program sets
  no shape.
- **S8. Configuration.** `ui.cursor_shape` in `config.json`: `"auto"` (default, per-mode shapes) and `"block"`
  (always block, today's look, no cursor bytes at all); `MISOKU_CURSOR_SHAPE` wins over the file for one run.
  Under `TERM=dumb` or when output is not a TTY nothing is emitted. Documented in README and CHANGELOG.
- **S9. Not focused.** Where the editor is not focused (finder, dialogs, start screen, graph, reading mode)
  the real cursor is hidden and the existing drawn cursors stay.
- **S10. Release.** CHANGELOG `[Unreleased]` becomes `[0.17.0]` at release time; the tag push cuts the
  release (`CONTRIBUTING.md:113`).

## Tasks

- T1 (spike, option A) — build the cursor writer + app wiring minimally and check frames live in Ghostty and
  inside herdr; decide A or B and record the reason in the plan. Route: inline. DONE: A ships, no frame
  corruption; the plan records the spike result.
- T2 — `internal/cursor`: DECSCUSR/CUP placement writer, frame detection, change-only emission, restore,
  policy from `ui.cursor_shape` / `MISOKU_CURSOR_SHAPE` / TTY / `TERM=dumb`. RED tests first. Route: inline.
  DONE in 9628ff4.
- T3 — app: cursor sink, `cursorCell` (mode, focus, overlays, layout offset), called from View; `execOutside`
  for `tea.ExecProcess`. RED tests through the harness + writer. Route: inline. DONE in 09350f1.
- T4 — fake cursor hidden in the textarea when the real cursor is active; steady blink. RED tests first.
  Route: inline. DONE in 09350f1.
- T5 — main wiring: policy, writer chain order, exit restore. RED tests first. Route: inline. DONE in
  47fa459.
- T6 — README (`Look` table + a short paragraph) and CHANGELOG entry. Route: inline. DONE in 96c4df0.
- T7 — live check in Ghostty and inside herdr; shell cursor back to default afterwards. Route: inline. DONE:
  plain Ghostty and herdr (in Ghostty) showed block, bar, fast typing, scrolling, a pane resize, the finder,
  a dialog and a Kitty image with clean frames, and the default block cursor in the shell after quitting.
- T8 — delivery: review, PR, CI, merge, `[0.17.0]`, tag, Release, `CURSOR-DONE`. Route: inline.

## Log

- L1 (owner request, verbatim, from the plan):
  > mira el cursor yo no se si esta en modo escritura o no, lazyvim ya tiene resuelto esto en modo escritura es
  > como un cursor fino y el gordito como misoku para el otro modo que no me acuerdo como se llama

  Approved: "si arma el plan lo verificas un agente con memoria limpia lo verifica y lo hacen".
- L2 (owner instruction, verbatim): "Implement the plan ~/Documents/MisokuVault/Misoku/2026-11-02-cursor-shape/README.md in this repo. Read it to the end: the "Verification log" amendments are binding and override the sections above them. Start with the time-boxed spike for option A (real terminal cursor); if it corrupts frames in ways the plan cannot fix cleanly, ship option B and record why in the plan. Test discipline: RED first per rule through the public render/output path; test scratch under TMPDIR=/var/tmp (never /tmp, never the real HOME); keep scratch small. Live check in Ghostty and inside herdr as the plan says, and afterwards confirm the shell cursor is back to default. Delivery exactly as the plan: branch feat/cursor-shape, small conventional commits, offer the gentle-ai review in this pane (owner grants it), PR, CI green on every platform (if a Windows test flakes, rerun the job once and report it), merge with a merge commit, CHANGELOG [Unreleased] -> [0.17.0], tag v0.17.0 on main, wait for the Release workflow, then write CURSOR-DONE with the PR and release URLs."
