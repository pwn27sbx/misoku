# Wider reading page: prose grows with the pane, tables and code go wider

Objective: on wide terminals the reading view (`space p`) and the preview stop looking like a thin strip in the middle: prose grows with the pane up to a cap, and tables and code blocks may use more width than prose instead of truncating with «…».
Branch: `main` in `~/misoku` (at `4e26caa`), delivered straight to `main` (owner's standing rule), released by the GitHub Release workflow on the tag, installed locally.
Test runner: `go test`, always `GOMAXPROCS=4 go test -p 2 -count=1 -timeout …` (60s test or mutant, 2m package, 5m suite). Forecast: about 250–400 authored lines.

## Specs

- S1 (L1, L2): «en pantallas mas grandes se ve asi como flaquito al medio». The prose page width becomes `proseWidth(w) = clamp(w*60/100, 88, 120)` for a pane `w` columns wide, and the page stays centred: `pageMargin(w) = max((w - proseWidth(w))/2, minMargin)`, with today's minimum margins (1; 2 from 40 cols; 4 from 70). For `w <= 148`, integer `w*60/100 <= 88`, so the layout is byte-identical to today (S5). At `w = 250`: prose 120, margin 65.
- S2 (L2): «vamos con la combinacion» — tables and plain code blocks may be wider than prose. Their budget is `wideWidth(w) = w - margin - minMargin(w)`: they start at the same left edge as prose (the page margin) and may extend to the right, up to `minMargin` columns from the pane's right edge. A table uses its natural width when it fits in `wideWidth` and only shrinks/truncates («…») beyond that; a code block's box is `clamp(longest painted line, c.width-4, c.wide-4) + 2` wide, so short code blocks keep today's prose-wide box and only long ones grow; a code line is truncated only past `c.wide - 4`. When the natural width is at most the prose width, nothing changes for that block.
- S3: Everything else stays at prose width: paragraphs, headings (including the heading band and H1 underline), lists, tasks, callouts, quotes, horizontal rules, columns, images, embeds, view blocks (chart, mermaid, board…), properties table, backlinks/mentions footer. Embedded sub-documents and columns keep their current widths (no wide budget inside them).
- S4: Mouse and handles keep working on wide blocks: hover, block handles (`⠿`, `+ ⠿`), drag-and-drop and its drop line, clicking a table cell and editing it, «+ Añadir fila / + Añadir columna», fold arrows on code blocks, links. Since wide blocks start at the same left edge as prose, the single shared left margin (preview `m.margin` = content `m.margin`, both from `PageMargin(EditorW)`) and every hit X offset stay valid; only the right-side clip moves.
- S5: Prose renders exactly as today for `w <= 148` (margins and wrapping unchanged) in both the reading view and the split preview. Tables and code blocks whose natural width is above the prose width may use the right margin from `w >= 97` (where `wideWidth` first exceeds prose; e.g. w=100 → 90, w=140 → 110, w=250 → 181); blocks that fit in prose look exactly as today at every width. Existing tests pass untouched except those whose hard-coded width assumption is the reason (each change explained).
- S6: Raw text revealed in INSERT inside the reading view still wraps at the prose width, not the wide width.
- S7: The split preview pane (`internal/preview` `View`/`HitAt`) follows the same rules as the reading view.
- S8: README/CHANGELOG say what changed (English).

Assumptions (plan's choices, changeable in one constant each):
- A1: 60 % and the 120-column cap. 120 keeps lines readable (~15–20 words); the 60 % starts growing only above 148 columns.
- A2: wide blocks extend to the right from the prose edge (Notion-style) instead of being centred. Centring would need a per-block left offset, which breaks the one-margin assumption used by every hit, `sourceAt`, `RenderedCell` and the handles drawn in the margin (map §3, §6). Tradeoff: on a very wide pane a wide table sticks out on the right only.

## Design (from the read-only map, 2026-11-18)

- `internal/preview/preview.go:52-68`: replace `const pageWidth = 88` by `proseWidth(w)` (88 floor, 120 cap, 60 %) and `wideWidth(w)`; `pageMargin(w)` uses `proseWidth`. `look.go:87` `PageMargin` stays the single exported source; export `WideWidth` (the app needs it for the textarea, see Clipping).
- Compiler: add `c.wide` next to `c.width` (`compiler.go:81`, set at `:306`; the `<10 → 40` fallback applies to both). `table()` (`compiler_render.go:386`) uses `c.wide` for `avail`; plain code (`compiler_render.go:70`, box at `:80`, style `compiler.go:68-72`) uses `inner = clamp(longest painted line, max(c.width-4, 6), max(c.wide-4, 6))`. A zero `c.wide` means `c.width` (embeds build a sub-compiler with a struct literal, `links.go:197`); `CompileHits`/`compileBlocks` default wide to the width they receive, so columns (`compiler_render.go:120`, `CompileHits(..., colW, ...)`) never get a wide budget, so every `Compile(md, th, width)` caller and sub-compiler (embeds `links.go:197`, columns `compiler_render.go:110`) behaves as today.
- Clipping, split preview: `recompile` compiles at `m.viewport.Width` (`preview.go:173`), so add an `m.prose` field set in `SetSize`; compile, backlinks, mentions and the drop line (`drag.go:206`) use `m.prose`, while `viewport.Width` (`preview.go:92`) and `ui.Fit(..., m.viewport.Width, ...)` (`preview.go:354`) use `wideWidth`.
- Clipping, reading view: `textarea.SetWidth` derives both `viewport.Width` and `m.width` from one value (`textarea.go:1007-1008`), and `content.fitText` passes `EditorW - margin` (`content.go:149-155`). Add a textarea setter for the view width (viewport = prompt + wide) that leaves `m.width` (prose) alone, so raw wrap, the wrap cache (`textarea.go:1503`) and the row index (`index.go:45`) stay at prose (S6); add a content API (e.g. `SetWideWidth`) and call it from `app.go:323` with an exported `preview.WideWidth(EditorW)`.
- Cursor-line band and padding: `bandRow` (`textarea/display.go:243`) and `writeDisplayRow` pad/band to `m.width`; rows wider than prose keep their own cells and the band stops at prose width (cosmetic, accepted); pinned by a test.
- Recompile on width change already happens via `SetSize → recompile` (new generation); the wide width derives from the same `w`, so no new trigger is needed — a test pins it.
- Handles (`drag.go:24-32`, `DecorateRows` `:181-212`, `PressHandleRow` `:118-131`): unchanged, they live left of the shared margin. The drop line (`drag.go:204-206`, `viewport.Width`) stays at prose width (assumption, low risk).

## Tasks

- [x] T1 (S1, S5): `proseWidth`/`pageMargin` with tests at 60, 88, 100, 148, 149, 200, 250, 400 columns; `TestPreviewCentresPageInWidePane` keeps passing at 140. Route: delegated (opus writer, long-session backstop). Status: done.
- [x] T2 (S2, S3, S6, S7): `c.wide` for tables and code; preview and textarea clip at `wideWidth`; tests: wide table untruncated at 250, truncated as today at 80, code line, prose still wrapped at prose width, raw INSERT wrap at prose width. Route: same writer as T1 (shares files). Status: done.
- [x] T3 (S4, S5): grep app tests at `termW = 140` that use tables or code (`pretty_click_test.go`, `table_hint_test.go`, `pretty_view_test.go`, `undo_test.go`, `tutor_events_b_test.go`) for content wider than 88 or `…` assertions; mouse/handles regression tests on a widened table at 250 (cell click, «+ Añadir columna», hover handle, drag). Route: same writer. Status: done.
- [x] T4 (S8): README + CHANGELOG. Route: same writer. Status: done.
- T5: verify (parent spot check + full suite), commit, push to `main`, tag, Release workflow, install locally, update `Proyectos/Misoku.md`, Buzón report for Omarchy. Route: parent. Status: pending.

## Log

- L1 (2026-11-18, owner): «el modo de visualizacion de misoku de vista bonita con space p siento que puede mejorarse porque como ves en pantallas mas grandes se ve asi como flaquito al medio no se evalualo»
- L2 (2026-11-18, owner, choosing between "page grows with a cap" and "tables/code may go wider"): «vamos con la combinacion»
- L3 (2026-11-18, owner): «como siempre armas el plan lo verificas y todo el proceso y luego al final lo pusheas a main y publicasa la nueva release y actualizas misoku aqui y generas el informe para claud ed emi omarchy»
- L4: map by a read-only explorer (preview/content/textarea width, margin and mouse paths). Parent spot-checked `preview.go:88-96`, `compiler.go:301-306`, `links.go:197`, `compiler_render.go:110`, `drag.go:24-32`, `compiler_render.go:386` (table `avail`), `:70` (code `inner`), `textarea/display.go:248-255` and `textarea.go:1326-1336`.
- L5: parent self-check: corrected the unchanged-layout bound from 146 to 148 (integer division: 148*60/100 = 88); sub-compilers use struct literals, so `c.wide` needs a zero-means-width rule.
- L6: independent clean-memory verifier (read-only). Confirmed: line refs, math up to 148, single left margin keeps hits/handles valid, no other `Compile*` callers. Corrected: S5 (wide blocks change from 97 columns, not 149), code box clamp so short code is unchanged, columns use `CompileHits` not a literal, split-preview `m.prose` field and `ui.Fit` at `:354`, textarea view-width setter, cursor band, tests to grep. Parent re-checked `textarea.go:1007-1008`, `preview.go:173`, `:354`, `compiler_render.go:120`.
- L7 (2026-11-18, owner, approving the plan with right-only wide blocks): «si»
- L8: writer done: S1-S8 covered, 11 new tests, RED observed, 7 mutants killed (cap, floor, table avail, code box, textarea wrap, view width, embeds), gofmt/vet clean, full suite 26 ok. Parent re-ran `go test ./internal/preview/` (ok) and reviewed the preview/textarea diff; edits stayed in the allowed surfaces.
