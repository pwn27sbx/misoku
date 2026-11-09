# f3a1-motor — pure tutorial engine: routes, steps, checks, stored progress

Branch `feat/tutor-routes` from `origin/main` (40a51fb, v0.27.0). Worktree
`~/Projects/misoku-yolo-worktrees/f3a-motor`. Plan:
`~/Documents/MisokuVault/Misoku/2026-11-09-tutorial-rutas/F3a-motor.md`, its
`## Amendments after clean review` (which overrides the text above) and the
master `README.md` amendments. This document is the binding summary; the plan
wins on any conflict.

Unit F3a1 only: pure `internal/tutor` plus the `internal/core` export. F3a2
(app and cmd) is out of scope. Test runner: `go test ./...`; `make all` =
fmt + vet + test + build. Scratch under `TMPDIR=/var/tmp`, never the real
HOME, config, vault or tutorial folder.

## Specs

- **S1. Catalog (D1).** `Catalog{Routes []Route, Extra []Lesson}`;
  `Route{ID, Name, Lessons}`; `Lesson{ID, File, Title, Steps}`;
  `Step{Text, Check, Hint}`. Files stay at the tutorial vault root, named
  `<ID> <Title>.md`; one index note per route (`B00 Básico.md`) and the final
  reference note `Después del tutorial.md`, both without steps. The catalog is
  injectable (tests build their own routes); `Default()` is the bundled one.
- **S2. Check kinds (D2).** `text`, `text-fence`, `text-raw`, `disk`,
  `action`, exactly the curriculum's predicates: `line-contains`,
  `line-starts`, `line-equals`, `line-ends`, `n-lines-start`, `absent`,
  `heading-exists`, `wikilink-resolves`, `blockref-exists`,
  `headinglink-exists`, `table-shape`; fence kind + inner line pattern; disk
  `exists`, `folder-exists`, `file-changed`, `other-note-text`, `config-key`,
  `no-file`; action name + optional value. Pure evaluation with an injected
  `Env` reader (nil callback = that check fails). Action checks never pass
  from text.
- **S3. Stored progress (D4/E6).** `<tutorial dir>/.misoku-tutor.json`:
  `{version, migrated, last_route, steps: {"B03#2": true}}`, keys 1-based
  (`S1`). `Load` tolerates a missing file, refuses a newer version, and
  reports malformed JSON. `Save` goes through the exported
  `core.WriteFileAtomic`, is serialized by a mutex, and marshals under the
  lock so a stale snapshot cannot win. A code comment notes that a synced
  tutorial vault would conflict on this file (no action now).
- **S4. Step semantics (E3/E4).** Steps may pass in any order; a passed step
  is sticky even if its text is later removed. `NextStep` is the first
  unpassed step, 1-based, 0 when none. A step with no check passes when the
  lesson note is saved with its text changed since it was opened.
- **S5. Migration (D7/E6).** `Install` moves the eight old level notes to the
  Trash once, matched by `ReadDir` + Unicode NFC (macOS NFD), never another
  name, and records `migrated: true`. It returns the one-line notice for the
  next open (on Windows it says the notes were deleted, since there is no
  Trash). The renamed `B0x` notes are never touched.
- **S6. Placeholder lessons (D9/E1).** Rename `00 Empieza aquí.md` to
  `B00 Básico.md` and `01 Títulos.md`…`07 Explorador.md` to
  `B01 Títulos.md`…`B07 Explorador.md`, unwrapped to one paragraph per line,
  every internal link rewritten. `B07` states reading view `space p` and
  editor view `space r`. Add `Después del tutorial.md` (no checks).
- **S7. Format test (D8/E6).** Over every bundled lesson: no paragraph is
  hard-wrapped (reuse `stripCode` to skip fences; front matter skipped;
  indented list continuations allowed). Every `Lesson.File` is embedded,
  every embedded file is in the catalog, IDs are unique.
- **S8. Core export (E6).** `core.WriteFileAtomic` replaces the unexported
  `writeFileAtomic`, with a doc comment; `internal/core/store_crud.go` and
  any other call site updated.
- **S9. Compatibility bridge.** The app still runs the pre-routes API until
  F3a2: `Lessons`, `Scan`, `Progress` (`Update`, `Count`, `Level`), `Hint`,
  `Resume` are derived from the new engine (one source of truth), marked as a
  bridge, and removed when F3a2 wires `Store`.

## Tasks

| ID | Spec | Route | Status | Evidence |
|---|---|---|---|---|
| T1 | S7 checks, S6 content | inline | done | commit `refactor(tutor): unwrap the placeholder lessons...` |
| T2 | S8 export | inline | done | commit `feat(core): export WriteFileAtomic` |
| T3 | S1, S2, S3, S4, S9 | inline | done | commit `feat(tutor): ...` |
| T4 | S5 | inline | done | commit `feat(tutor): retire the old level notes...` |

## Log

- **L1 (owner, verbatim):** "Read /home/pwnsxb/Documents/MisokuVault/Misoku/2026-11-09-tutorial-rutas/README.md (master plan, with Amendments and 'Parallel work and cleanup'), then .../F3a-motor.md fully, including its Amendments section, which overrides earlier text. Implement ONLY unit F3a1 as the Amendments define it (pure internal/tutor + the core.WriteFileAtomic export, D1, D2, D4, D7, D8, D9, E1, E3-E6). Do not start F3a2. Reply in English. Before reporting, offer the gentle-ai review in this pane. Finish with the report and MISOKU-F3A1-DONE alone on a line."
- **L2:** F3a1 is the pure engine; the app keeps compiling through a temporary
  bridge derived from the engine, and app tests only change the lesson names
  the rename forces. Feedback, Home, first run, `--route`, resume and current
  lesson stay for F3a2.
- **L3:** `B07` already said reading `space p` / editor `space r` at
  origin/main 40a51fb; the E1 fix is verified, not re-applied.
