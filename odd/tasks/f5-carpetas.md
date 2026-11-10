# f5-carpetas — one folder per route in the tutorial vault

Branch `feat/tutor-folders` in `~/Projects/misoku-yolo-worktrees/f5-carpetas`,
from `integ/tutor-content` at `83408e0`. Brief:
`<vault>/Misoku/2026-11-09-tutorial-rutas/F5-carpetas.md` (with
Amendments G1-G10, which override the sections above). Test runner
`go test ./...`; `make all` = fmt + vet + test + build; temp HOME/XDG/`TMPDIR`
under `mktemp -d`; `GIT_CONFIG_GLOBAL` at an empty temp file.

## Specs

- **S1 (F1/G1).** The embedded bundle keeps the folders: `//go:embed lessons`
  covers subfolders, `Bundled()` stays rooted at `lessons`, and
  `fs.Stat(Bundled(), "1 Básico/B01 Escribe en 30 segundos.md")` succeeds.
  Each route JSON has `"dir"` (`"1 Básico"`, `"2 Intermedio"`,
  `"3 Avanzado"`, `"4 Experto"`); `file`/`index` stay bare names.
- **S2 (F1/G7).** Catalog validation requires `dir`: it passes
  `checkFileName` (no `/`), is unique across routes, and starts with the
  route's number and a space. `Después del tutorial.md` and `mapa.png` stay at
  the vault root.
- **S3 (F2/G3/G4).** Lesson identity is the vault-relative path with `.md`:
  `lessonFile(p)` returns `filepath.ToSlash(p)` in NFC with `.md`, and no
  longer refuses `/`. Everything compares `dir + "/" + File` (or
  `dir + "/" + Index`): `Catalog.Lesson`, `RouteOf`, `ResumeNote` (index and
  lesson), `progress.recheck`/`ScanDir`, and the app's
  `tutor.go`/`tutor_events_c.go` comparisons. A bare name or a root copy is
  no longer the lesson.
- **S4 (G2).** `originals()` walks the embed recursively and keys each note by
  its vault-relative path; a freshly installed, unedited lesson in a route
  folder passes no step.
- **S5 (F3/G5).** Disk checks accept the lesson's route folder first, then the
  vault root: `Env.LessonDir` is set by the store, and `exists`, `no-file`,
  `folder-exists` and `other-note-text` try `LessonDir/arg` then `arg`.
  `file-changed` keeps using `Env.LessonFile` (now the full path). The app
  `Env` and `DirEnv` agree, proven by one shared table test.
- **S6 (F4/G7).** Lesson text that names a place matches the folder layout
  (`A09`, and `B00` says the routes live in folders). Wikilinks across route
  folders still resolve by bare name, `Siguiente:` links included, and
  `![[mapa.png]]` from `3 Avanzado/A09 …` still renders from the root.
- **S7 (F5/G9).** `misoku tutor` opens the current lesson; the explorer
  already expands the route's folder through `SetSelectedID`. Prove it with a
  harness test; no new explorer feature.
- **S8 (F6).** No flat→folders migration: progress is keyed by step ID, the
  flat layout was never released, and `--reset` trashes the old folder. The
  0.27 migration still works and upgrades straight to the folder layout.
- **S9 (G6).** Format and coverage tests walk the embed with vault-relative
  paths and assert 101 notes (97 lessons + 4 indexes); neither passes with
  zero files.
- **S10 (G10/F7).** No event value is compared against a full note ID that now
  carries a folder. README documents one folder per route; CHANGELOG is left
  to integration.

## Tasks

| ID | Spec | Route | Status | Evidence |
|---|---|---|---|---|
| T1 | S1, S2 | inline | done | `9a8fa93`: embed + `git mv` 101 notes + `dir` in routes |
| T2 | S3, S4 | inline | done | `9a8fa93`: `Path`/`IndexPath`, `lessonFile`, `originals` |
| T3 | S5 | inline | done | `dca19a4`: `Env.LessonDir`, `vaultPaths`, table tests |
| T4 | S6, S10 | inline | done | `b429821`: A09/B00 text; link + map tests |
| T5 | S7 | inline | done | `9a8fa93`: `TestTutorExpandsTheRouteFolderAtStart` |
| T6 | S8, S9 | inline | done | `9a8fa93`: upgrade test, WalkDir tests |
| T7 | S10 | inline | done | `b429821`: README section |

## Where each teaching action writes (F3 decision table)

| Action | Lands in | Check accepts |
|---|---|---|
| `space n`, explorer `n`, dirbuf `m`, `M` (mini.files), drafts from bare `[[links]]`, `/Page`, `s` (sub-note) | the lesson's route folder | route folder, then root |
| `:w name` | vault root (`SaveAs` gets no folder) | route folder, then root |
| `:export` (no arg / name), `space P` | the note's folder | route folder, then root |
| `/Image`, `/Paste image` | `assets/` next to the note | route folder, then root |
| daily note (`space o d`), `/Template` | `Diario/`, `Plantillas/` at the vault root | route folder, then root |
| `misoku tutor` git init | `.git` at the vault root | route folder, then root |
| config steps | `config.json` | unchanged (values) |

## Log

- **L1 (owner, verbatim):** "New task: implement
  <vault>/Misoku/2026-11-09-tutorial-rutas/F5-carpetas.md
  (read it fully, including Amendments G1-G10, which override the sections
  above; read README.md in that folder first). You are already in the worktree
  ~/Projects/misoku-yolo-worktrees/f5-carpetas on branch feat/tutor-folders at
  83408e0. Tests use temp HOME/XDG dirs and GIT_CONFIG_GLOBAL; never touch
  ~/.config/misoku or the global git config. No push, no PR, no merge. Offer
  the gentle-ai review in this pane before reporting, then print
  MISOKU-F5-DONE alone on a line."
- **L2:** Baseline `go build ./...` and `go test ./internal/tutor` green at
  `83408e0`.
- **L3:** `make all` green at `b429821`: 25 packages, 2082 top-level tests.
  `go vet ./...` clean, `gofmt -l` empty.
- **L4 (G6 count):** the embed holds 102 notes, not 101: 97 lessons + 4
  indexes (101) plus `Después del tutorial.md`. The tests assert 102 and say
  so in their message.
- **L5 (F3 gap found, fixed in L9):** A13's three `exists` checks look for
  PDFs, but `exists`/`no-file` only understood note names (they add `.md`, and
  `core.ResolveNote` trims only `.md`), so those steps failed at `83408e0`
  too. F5 kept their paths and pinned the gap in `disk_test.go`.
- **L6:** The statusline hides the note's folder during a tutorial session:
  with the folder segment the tutor's success line no longer fitted at 140
  columns (`TestTutorShowsTheStepAndThenTheSuccess`). The route folder is the
  same for 97 notes and the explorer already shows it.
- **L7 (review):** the whole feature as one candidate (229 files) was refused
  as `lens_context_budget_exceeded`; no authority was created. Split: the
  commits after the layout (`dca19a4..a6f3ed2`: disk checks, lesson text,
  README, the log) were reviewed (risk, readability, reliability), approved
  and acknowledged. Its advisories found a real defect — `no-file` passed with
  no `Exists` reader — fixed in `8ad6064` and the personal vault path in the
  log, fixed in `e617540`. The fix delta (`e617540..8ad6064`) was reviewed
  (readability, reliability), approved and acknowledged; one readability
  suggestion (the three existence helpers repeat one loop shape) stays as
  advice.
- **L8 (review limit):** the layout commit alone (223 files, 4347 lines) also
  exceeds the native lens budget, so no review authority can be created for
  the note move. The move is mechanical: `git diff -M --summary` shows 101
  renames with content edits only in `A09` and `B00`, and `make all` pins the
  layout.

- **L9 (follow-up, owner):** fix the A13 gap before the owner plays that
  lesson. `tutor.DiskFileName` gives a disk path its file: a bare name gets
  ".md", and a path with its own extension (".md", ".pdf") is used as written.
  `DirEnv` and the app `Env` both build their `Exists` on it (the app still
  resolves notes by name first), so `exists`/`no-file` read an exported PDF in
  the route folder or at the root. A13's lesson text already matches: step 2
  exports `informe.pdf` with a name, which the app writes in the note's
  folder. Tests: `TestExportStepsPassWithRealPDFs` exports the three PDFs with
  the real writer and requires each step to fail before and pass after;
  `TestDiskFileName`, `TestDiskExistsAcceptsAnMDSuffix` and the app/DirEnv
  agreement table pin the `.md` and `.pdf` forms.
