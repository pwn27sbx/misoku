# onboarding-learn — first-run dialog and a Learn entry on Home

Branch `feat/onboarding-learn` (from origin/main d82ff30, v0.20.0). Plan:
`~/Documents/MisokuVault/Misoku/2026-11-06-onboarding-learn/README.md`; the `## Specs` below are its binding
summary. Test runner: `go test ./...`; every test and live check uses a temp HOME under `/var/tmp` and
`TMPDIR=/var/tmp`, never the real config, vault or `~/.local/bin/misoku`.
Delivery: conventional commits, CHANGELOG `[Unreleased]`, README, gentle-ai review offered in the pane. No push,
no merge, no release, no tag.

## Specs

- **S1. First open** shows a floating dialog with an original ASCII-only cat, in Spanish like the tutor:
  `Soy nuevo` starts the tutorial, `Ya tengo experiencia` closes it, `Ahora no` closes it. Esc means
  `Ahora no`. Ctrl+C quits without recording anything.
- **S2. Once only.** Any of the three answers is remembered (top-level `learn_offered` in `config.json`), so the
  dialog never appears again on that machine. Existing users see it once after updating: no migration.
- **S3. Home.** A `Learn` entry under `Graph` with key `l`, showing progress such as `Learn 3/7` and a check when
  complete (`Learn 7/7 ✓`). Visible from 13 rows of the home screen (terminal height 14).
- **S4. Progress survives quitting and updates.** In a tutorial session a newly passed level saves its note at
  once, because Misoku has no autosave and `Scan` reads the disk.
- **S5. Resume.** `misoku tutor` opens `00 Empieza aquí` at 0/7 and when every level passes, otherwise the first
  unpassed lesson.
- **S6. Compatibility.** `misoku tutor` and `misoku tutor --reset` keep working. `00 Empieza aquí.md` gains the
  line that Ctrl+C returns to the vault.
- **S7. Gate.** The dialog is offered only with `tutorDir == ""`, stdin and stdout terminals, `Source` one of
  `fromNotesDir`, `fromCwd`, `fromEnv`, `cfgErr == nil`, `cfgPath != ""` and `learn_offered` unset. `--dir` and a
  folder argument never see it and never touch the flag. An opt-in `app.Option`, so test harnesses never see it.
- **S8. No blind keys.** On Home, with no other modal open, and only when the dialog fits: below 60x12 `View`
  shows only "Terminal too small" while `m.confirm` still takes keys, so a resize that stops it fitting closes
  the dialog and keeps it pending.
- **S9. Launch and return.** `execOutside` runs `os.Executable()` with `tutor`; unsaved tabs stay in the parent's
  memory; on return mouse motion reports are turned back on (`tea.EnableMouseAllMotion`, which
  `ReleaseTerminal`/`RestoreTerminal` do not restore) and a non-zero exit is shown with `setError`. No workspace
  switch and no `--theme` passed on.
- **S10. Entry hidden** when `tutor.Dir()` is "" and inside a tutorial session (sessions cannot nest).

## Tasks

- T1 config: `learn_offered` field and `config.SetLearnOffered` on `editFile`. Route: inline. Tests: writes,
  keeps every other key, idempotent, bad file untouched.
- T2 tutor: `tutor.Resume`; `app.WithOpenNote`; main opens the resumed lesson. Route: inline. Tests: 0/7, midway,
  7/7, nil progress, the app opens the note.
- T3 home: `dashboard.SetLearn` entry under Graph, `app.WithLearn`, key `l`, `ActionLearn`, the tutorial launch
  and its return. Route: inline. Tests: label, position, cursor kept, hidden without progress, `l` runs
  `misoku tutor`, the return rescans and re-enables the mouse.
- T4 dialog: `confirmDialog.minInner`, the offer gate, the answer write and the app launch. Route: inline.
  Tests: each answer remembered, Esc, Ctrl+C records nothing, too-small terminal, resize keeps it pending,
  `--dir` gate, config write failure.
- T5 tutor: save a level as soon as it passes. Route: inline. Test: pass, quit with Discard, `Scan` counts it.
- T6 docs: README (key line, Learn section), CHANGELOG `[Unreleased]`, the lesson's Ctrl+C line.
- T7 differential against the base in a scratch copy (Home unchanged except the entry; the banner threshold moves
  up one row, allowed), `go vet`, `go test ./...`, live check in Ghostty and herdr.

## Log

- L1 (user): "Implement the approved plan at
  /home/pwnsxb/Documents/MisokuVault/Misoku/2026-11-06-onboarding-learn/README.md and read it completely first.
  Run git fetch, then work only in a new worktree ~/Projects/misoku-yolo-worktrees/onboarding-learn on a new
  branch feat/onboarding-learn from the latest origin/main, and do not switch or touch the ~/Projects/misoku-yolo
  checkout. Do not touch the folder editor under revision. Never touch the real ~/.config/misoku, the real vault
  or the installed ~/.local/bin/misoku: tests and live checks use a temp HOME under /var/tmp, and TMPDIR under
  /var/tmp. Write RED tests first as the plan says, then the differential against the base in a scratch copy. For
  the live check in Ghostty and herdr, build into scratch and unset all HERDR env vars for any nested herdr.
  Conventional commits, no AI attribution, no Co-Authored-By. No push, no merge, no release, no tag. Before your
  final marker, offer the gentle-ai review in this pane. When finished, report commits, checks with observed
  results, anything not verified, and end with the line MISOKU-LEARN-DONE, or MISOKU-LEARN-BLOCKED if something
  stops you."
