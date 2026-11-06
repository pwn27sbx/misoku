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
  keeps every other key, idempotent, bad file untouched. DONE in e78c06b.
- T2 tutor: `tutor.Resume`; `app.WithOpenNote`; main opens the resumed lesson. Route: inline. Tests: 0/7, midway,
  7/7, nil progress, the app opens the note. DONE in eb472d8.
- T3 home: `dashboard.SetLearn` entry under Graph, `app.WithLearn`, key `l`, `ActionLearn`, the tutorial launch
  and its return. Route: inline. Tests: label, position, cursor kept, hidden without progress, `l` runs
  `misoku tutor`, the return rescans and re-enables the mouse. DONE in eb472d8.
- T4 dialog: `confirmDialog.minInner`, the offer gate, the answer write and the app launch. Route: inline.
  Tests: each answer remembered, Esc, Ctrl+C records nothing, too-small terminal, resize keeps it pending,
  `--dir` gate, config write failure. DONE in 4cdee18; the terminal streams fixed in 844b9f9.
- T5 tutor: save a level as soon as it passes. Route: inline. Test: pass, quit with Discard, `Scan` counts it.
  DONE in 4bd445e.
- T6 docs: README (key line, Learn section), CHANGELOG `[Unreleased]`, the lesson's Ctrl+C line. DONE in 64a2ce3.
- T7 differential against the base in a scratch copy (Home unchanged except the entry; the banner threshold moves
  up one row, allowed), `go vet`, `go test ./...`, live check in Ghostty and herdr. DONE, see L3-L5.

## Log

- L1 (user): "Implement the approved plan at
  ~/Documents/MisokuVault/Misoku/2026-11-06-onboarding-learn/README.md and read it completely first.
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
- L2 (finding): a program run with `tea.ExecProcess` inherits the program's output writer as its stdout when it
  sets none, and `os/exec` gives a pipe to a writer that is not an `*os.File` (see `syncout.Writer`'s own note).
  A piped terminal reports no size, so the tutorial stayed on "loading…" behind the released terminal. The
  tutorial command now sets `os.Stdin`, `os.Stdout` and `os.Stderr`; `TestTutorialRunsOnTheTerminalItself` pins
  it. The cursor PTY tests now write `learn_offered` in their temp config, as a returning user has it.
- L3 (differential, scratch copy of d82ff30 under /var/tmp): the Home dump at 11 sizes differs only in the Learn
  row, in the vertical centring it moves by one row, and in the banner threshold: at a dashboard height of 27
  (terminal 28) the figlet banner falls back to `M I S O K U`, exactly as allowed. At 13 rows the entry is
  visible; at 12 rows `Quit` is the row clipped (Home already clipped its last rows there).
- L4 (live check, herdr pane inside Ghostty, built binary in /var/tmp): the dialog shows with the cat; Enter
  starts the tutorial on `00 Empieza aquí` and writes `learn_offered: true`; typing `- [x] probando` passes
  level 2 with the statusline `Tutor 1/7` and `"02 Casillas.md" 22L written`; Ctrl+C returns to Home with
  `Learn 1/7`; an unsaved `Untitled-1` survived `l` and the tutorial; `l` resumed at `01 Títulos.md`; an injected
  SGR click opened Settings, so clicks reach the app after returning.
- L5 (live check, a Ghostty window of its own, `script` recording every byte): the dialog with the cat, then the
  tutorial, then Home with `Learn 0/7`; the record shows `\x1b[?1003l` when the terminal was released for the
  tutorial and `\x1b[?1003h` right after `\x1b[?1049h` on the way back, so the terminal sends mouse events again.
- L6 (review): gentle-ai reviewed the branch with the risk, resilience, readability and reliability lenses and
  approved it (advisory findings only, no correction). Its non-blocking notes led to 0f389cc, after the
  acknowledgement: the config test now compares every kept value, `fakeTutorial`'s comment matches what it does,
  and this document names no absolute home path.

