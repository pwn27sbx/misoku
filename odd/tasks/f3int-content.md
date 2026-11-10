# f3int-content — integrate the four F4 routes with the engine

Branch `integ/tutor-content` in `~/Projects/misoku-yolo-worktrees/f3a-motor`
(F3a1+F3a2 + the four `content/tutor-*` merges, placeholders removed at
`bf07371`). Plan: `F3a-motor.md` Amendment E16. Test runner `go test ./...`;
`make all` = fmt + vet + test + build; temp HOME/XDG in tests.

## Specs

- **S1. Schema to the engine.** Numeric `args` in the F4 JSON become strings
  (the documented schema is `args: []string`; L2-L3/E13 orders kept). The
  `columns` fence kind is added to the engine's list because I11 is a real
  `~~~columns` fence the curriculum teaches (the engine change is the fence
  list; the JSON keeps `text-fence`).
- **S2. E15 in B13/B15.** B13 checks `# Viaje` and the three subheadings with
  `heading-exists`/`heading-under`; its lesson asks for those exact headings.
  B15 checks the named tasks with `task-status` (the three selected tasks as
  an `all`), so the task text is required.
- **S3. Tests.** No test depends on the removed placeholder lessons; app,
  cmd and tutor tests use real F4 lessons or purpose-built fixtures with the
  same intent.
- **S4. Baseline and events.** Every embedded lesson is checked untouched
  (97 lessons, 316 steps, none passes), and every action event used by a
  route is in the engine's known-event list (80 distinct events).

## Tasks

| ID | Spec | Route | Status | Evidence |
|---|---|---|---|---|
| T1 | S1 | inline | done | `routes/{b,i,a,e}.json`, `catalog.go` |
| T2 | S2 | inline | done | `routes/b.json`, `lessons/B13 Títulos.md` |
| T3 | S3, S4 | inline | done | tutor/app/cmd tests, `TestLessons` |

## Log

- **L1 (owner, verbatim):** "New unit F3-INT: in the same worktree, branch
  integ/tutor-content is checked out (the four F4 routes merged onto your
  F3a2 work, placeholders removed). Read 'Amendment E16' at the end of
  .../F3a-motor.md and do exactly that until the full suite is green. Reply in
  English. Finish with the report and MISOKU-F3INT-DONE alone on a line."
- **L2:** `routes/a.json` and the other routes passed numbers in `args`; fixed
  the JSON to strings rather than loosening the engine's `[]string` schema.
  I11's `columns` fence was missing from the engine's fence list; added it.
  B13/B15 moved to the E15 predicates as E16 asks.
