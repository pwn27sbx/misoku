# f3a2-app — wire the routes engine into the app, home, first run and CLI

Branch `feat/tutor-routes` (same worktree as F3a1, `~/Projects/misoku-yolo-worktrees/f3a-motor`).
Plan: `~/Documents/MisokuVault/Misoku/2026-11-09-tutorial-rutas/F3a-motor.md`
(D3, D5, D6, E2, E7–E11, E14 git init) plus E15, and the master `README.md`.
Test runner: `go test ./...`; `make all` = fmt + vet + test + build. Scratch
under `TMPDIR=/var/tmp`, never the real HOME, config, vault or tutorial folder.

## Specs

- **S1. E15 predicates.** `heading-under [parent_level, parent_text, level,
  text]` (child after the parent and before the next heading of the parent's
  level or above; the parent may be the lesson's own, the child must be new)
  and `task-status [state, text]` (`done` x/X, `open` space, `doing` `-`;
  any bullet, any indentation, the task text required). Table tests, schema
  validation, and B01/B02 use them.
- **S2. Event API (D3/E9).** `tutor.Event{Name, Data}` and
  `Store.RecordEvent`; `m.tutorEvent(name, value)` credits the current lesson
  even when another tab is active. `mode_changed` compares the editor mode
  before/after a key; `next_tab`/`prev_tab` fire from the tab switch. The
  convention for F3b/F3c is documented in `internal/app/tutor.go`.
- **S3. Current lesson (E2).** The last lesson note focused stays current
  until another lesson note is focused; opening, clicking and switching tabs
  all count, and focusing a non-lesson never clears it.
- **S4. Feedback (D5/E8).** Persistent `B03 · Paso 2/4 · <step>`;
  `✓ Paso N` for ~1.5 s in tutor state; `✓ B03 superado · siguiente:
  <lesson>`; `🏆 Ruta <name> completa`; the hint once per current lesson;
  the route badge `Tutor k/n`. `message()` renders it after `m.status` and
  combines the table hint.
- **S5. Disk checks (E7).** Run in a `tea.Cmd` on lesson open and after a
  successful save; edits only text checks. The command captures the notes,
  the tutorial root, the opened-at time and the config snapshot; the config
  reader is a small dotted-key JSON lookup. `{today}` is passed in `Env.Today`.
- **S6. Home and first run (D6/E10).** One Learn entry per route with
  lessons (`Learn · Básico 3/26`, ✓ when complete); `l` opens the first
  incomplete route; each entry/`Click` carries its route. `Soy nuevo` opens
  B; `Ya tengo experiencia` opens I (falls back to B while I has no lessons);
  `Ahora no` opens the notes.
- **S7. Resume and `--route` (D6).** `misoku tutor` resumes the last route
  played at its first unfinished step; `--route R` opens that route's index
  when nobody started it, else resumes. Only routes with lessons exist.
- **S8. Tutorial git (E14).** On install, if git is available, the tutorial
  folder becomes a repository with one commit of the lessons (progress file
  ignored) under a timeout; an existing repo is left alone. Without git the
  steps that wait for `history_*` are skipped, not failed, and the bar says
  `Necesitas git para esta lección`; the skip is cleared when git is there.
- **S9. Docs.** README "Learn Misoku" and the command list updated; CHANGELOG
  `[Unreleased]`; a note in `F4-lecciones.md` telling F4-B to use the E15
  predicates for B13/B15.
- **S10. Tests.** App tests for the step line, success line, route complete,
  hint, skip without git, mode/tab events, disk-on-open, home entries, `l`,
  per-entry route, first-run routes/fallback; cmd tests for `--route`, usage,
  and the git init; config reader and skip store round-trip.

## Tasks

| ID | Spec | Route | Status | Evidence |
|---|---|---|---|---|
| T1 | S1 | inline | done | `internal/tutor/{catalog,check}.go`, `routes/b.json`, tests |
| T2 | S2, S3, S4, S5 | inline | done | `internal/app/tutor.go`, `tutor_config.go`, `bars.go`, `app_keys.go`, `buffers.go` |
| T3 | S6, S7, S8 | inline | done | `learn.go`, `welcome.go`, `dashboard.go`, `cmd/misoku/{tutor,git}.go` |
| T4 | S9, S10 | inline | done | README, CHANGELOG, F4 note, tests |

## Log

- **L1 (owner, verbatim):** "F3a1 verified, thanks. Now implement unit F3a2 in the same worktree and branch, exactly as .../F3a-motor.md defines it (D3, D5, D6, E2, E7-E11, the git-init part of E14) plus the new Amendment E15 at the end of that file. README tutorial section and CHANGELOG [Unreleased] entries are now in scope. Same rules: temp HOME/XDG, make all green, no push, gentle-ai review in this pane. Finish with the report and MISOKU-F3A2-DONE alone on a line."
- **L2:** E15 resolves F3a1's open review finding: add the two predicates and
  switch B01/B02 to them; F4-B is told to use them for B13/B15.
- **L3:** The old `internal/tutor` compatibility bridge (Lessons, Progress,
  Scan, Resume, Hint) is removed; the app and cmd use `Store` + `Catalog`.
- **L5:** Implemented in `7ba9e1a`; `make all` green with 25 packages,
  1964 top-level tests and 792 subtests, 0 failures.
- **L4:** Git is optional: the skip rule keys off the `history_*` event
  family, so no content is hardcoded; the skip lives in the progress file.
