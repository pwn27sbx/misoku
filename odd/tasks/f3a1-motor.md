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

- **S1. Catalog (D1 + E12).** The catalog is embedded JSON, one file per route
  in `internal/tutor/routes/{b,i,a,e}.json` (`//go:embed`), with exactly the
  schema in `F4-lecciones.md` ("What each unit writes"): route
  `{id, name, index, lessons}`, lesson `{id, title, file, steps}`, step
  `{text, hint?, check, ids?}`. `Check{kind, pred?, args?, fence?, pattern?,
  event?, value?, checks?}` mirrors the schema; `kind` is one of `text`,
  `text-fence`, `text-raw`, `disk`, `action`, `all`, `none`. `BundledRoutes()`
  loads and validates every embedded route (unknown kind/pred/event, bad
  arity, duplicate IDs, a missing file → error, tested; `Default()` degrades
  to an empty catalog at runtime instead of panicking). File names are L1
  slugs (letters, digits, spaces, `, - .`). Routes with no file do not exist.
  The catalog is injectable for tests. Files stay at the tutorial vault root;
  the route index (`B00 Básico.md`) is a route field, not a lesson, and the
  final reference note `Después del tutorial.md` is embedded but belongs to no
  route.
- **S1b. E13 schema additions.** `all` runs every sub-check (`checks`), each
  sticky. An action `value` is a string or an array; every listed value must be
  seen. `config-key [key]` = present, `[key, value]` = equals; new
  `config-changed [key]` = changed since the lesson opened. Paths expand
  `{today}` to `AAAA-MM-DD` from `Env.Today`. Argument orders are fixed (L3),
  with no permissive parsing.
- **S2. Check kinds (D2).** The curriculum's predicates: text
  `line-contains`, `line-starts`, `line-equals`, `line-ends`,
  `n-lines-start`, `absent`, `heading-exists`, `wikilink-resolves`,
  `blockref-exists`, `headinglink-exists`, `table-shape`; `text-fence` uses
  `fence` + `pattern` (a line of that fence contains the pattern); `text-raw`
  keeps inline code; disk `exists`, `folder-exists`, `file-changed`,
  `other-note-text`, `config-key`, `config-changed`, `no-file`; action `event`
  + optional `value` (string or array). Pure evaluation with an injected `Env`
  reader (nil callback = that check fails). Action checks never pass from text.
- **S2b. Baseline (L4/E13).** Text, text-fence and text-raw checks ignore every
  line present in the embedded original of the lesson file, so a lesson never
  passes on its own text; the store subtracts the original before evaluating,
  and table tests prove it.
- **S3b. Sticky sub-checks.** Each leaf of an `all` and each action value has
  its own progress key (`B03#2.1`, `B03#2.1=2`) so a sub-check that passed
  stays passed even when the text changes; the step key is set once every
  leaf is stored.
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
- **S10. E14.** Text checks see the front matter (only the D8 format test
  skips it); `{today}` expands in text and text-fence patterns as well as in
  paths; `lessons/*.png` is embedded too, with one small original `mapa.png`
  (< 50 KB) for the image lessons. The git-init part of E14 is F3a2.
- **S9. Compatibility bridge.** The app still runs the pre-routes API until
  F3a2: `Lessons`, `Scan`, `Progress` (`Update`, `Count`, `Level`), `Hint`,
  `Resume` are derived from the new engine (one source of truth), marked as a
  bridge, and removed when F3a2 wires `Store`.

## Tasks

| ID | Spec | Route | Status | Evidence |
|---|---|---|---|---|
| T1 | S7 checks, S6 content | inline | done | `89275b5` — refactor(tutor): unwrap the placeholder lessons and rename them for route B |
| T2 | S8 export | inline | done | `c9272ee` — feat(core): export WriteFileAtomic |
| T3 | S1, S1b, S2, S2b, S3, S3b, S4, S9, S10 | inline | done | `892ab76` — feat(tutor): routes, steps, embedded checks and stored progress |
| T4 | S5 | inline | done | `892ab76` — the one-time migration lives in `Install` |

## Log

- **L1 (owner, verbatim):** "Read /home/pwnsxb/Documents/MisokuVault/Misoku/2026-11-09-tutorial-rutas/README.md (master plan, with Amendments and 'Parallel work and cleanup'), then .../F3a-motor.md fully, including its Amendments section, which overrides earlier text. Implement ONLY unit F3a1 as the Amendments define it (pure internal/tutor + the core.WriteFileAtomic export, D1, D2, D4, D7, D8, D9, E1, E3-E6). Do not start F3a2. Reply in English. Before reporting, offer the gentle-ai review in this pane. Finish with the report and MISOKU-F3A1-DONE alone on a line."
- **L2:** F3a1 is the pure engine; the app keeps compiling through a temporary
  bridge derived from the engine, and app tests only change the lesson names
  the rename forces. Feedback, Home, first run, `--route`, resume and current
  lesson stay for F3a2.
- **L3:** `B07` already said reading `space p` / editor `space r` at
  origin/main 40a51fb; the E1 fix is verified, not re-applied.
- **L8:** `make all` green on `892ab76`: 25 packages, 1951 top-level tests
  and 774 subtests pass, 0 failures. `go test -race ./internal/tutor/` is
  green too.
- **L7:** README and CHANGELOG are left for F3a2/F4: F3a1 changes no key,
  command or setting the player sees, and the tutorial chapter is rewritten
  when Home, feedback and the real lessons land.
- **L6 (owner, 2026-11-09, verbatim):** "One more small amendment: read
  'Amendment E14' at the end of .../F3a-motor.md. In F3a1 apply the parts that
  belong to internal/tutor: text checks see front matter (only the D8 format
  test skips it), {today} expands in text/text-fence patterns as well as
  paths, and embed lessons/*.png. The git-init part is for F3a2 later."
- **L5 (owner, 2026-11-09, verbatim):** "Another schema update before you
  finish: read 'Amendment E13' at the end of .../F3a-motor.md and the
  referenced Amendments L2-L4 in F4-lecciones.md (kind 'all', value arrays,
  config-key/config-changed, {today}, fixed argument orders, and the baseline
  rule that text checks ignore lines already in the embedded original).
  Include them in F3a1 with table tests." L2 also makes every sub-check
  sticky, so progress stores a key per leaf and per action value.
- **L4 (owner, 2026-11-09, verbatim):** "Important change before you go
  further: read the new section 'Amendment E12' at the end of .../F3a-motor.md
  and the schema in .../F4-lecciones.md ('What each unit writes'). The catalog
  must be loaded from embedded JSON (internal/tutor/routes/*.json) with Go
  types mirroring that schema, validated by tests. Adjust your F3a1 work to it,
  then continue as before." The Go catalog literal from T1's plan is replaced
  by `routes/b.json`; E12 overrides D1's "checks stay in Go".
