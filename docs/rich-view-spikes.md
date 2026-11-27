# Rich reading view — S0 spike gates

Throwaway spike, 2026-11-27, worktree `~/Projects/misoku-yolo-worktrees/rich-view`
(`feat/rich-view` from `3bb96a0`). No product code: this records the four gates
that decide the rendering stack. The spike lived outside the repo
(`/tmp/opencode/rich-spike`) and is not committed.

Machine: AMD Ryzen 7 4800H, Go 1.27.3, Linux. Fonts: `JetBrainsMono Nerd Font`
for text, `Latin Modern Math` (CTAN `fonts/lm-math/opentype`, 733 736 bytes,
GUST Font License) for maths.

## T0-FONT — shaping and rasterising in pure Go

| Measurement | Result | Gate |
| --- | --- | --- |
| `go-text/typesetting` shaping, 1 000 lines × 70 runes | 30.2 ms (68 000 glyphs) | layout ≤ 40 ms ✔ |
| `golang.org/x/image` face raster, 1 440 × 1 152 tile (48 rows × 120 cols, 12×24 px) + vector rules | 9.7 ms | tile ≤ 20 ms ✔ |
| PNG encode of that tile, `png.DefaultCompression` | 41.8 ms | — |
| PNG encode of that tile, `png.BestSpeed` | 9.3 ms, 31 731 bytes | must use BestSpeed ✔ |
| `tdewolff/canvas` fallback, same tile without PNG | 23.6 ms | fallback viable ✔ |

Decision: **`go-text/typesetting` for shaping + `golang.org/x/image` for
faces/vector**, tiles encoded with `png.BestSpeed`. `tdewolff/canvas` stays as
the fallback and as the SVG renderer (below).

## T0-MATH — own parser and box skeleton

A hand-written TeX tokenizer/box model (no mddog code) over 12 formulas
covering the construct list mddog's engine handles: fractions/binomials,
`\sqrt[n]`, scripts, big operators with limits, `\left…\right`, all matrix
environments incl. `cases`/`aligned`, 14 accents, 8 alphabets, `\text`,
`\overset`/`\stackrel`/`\underset`, `\xrightarrow`, styles, spacing, functions,
Greek and the symbol table.

Result: **0.08 ms** for the corpus, 219 nodes, 206 sized boxes, no parser
panic. The skeleton is thrown away; S4 writes the engine in
`internal/rich/math` from TeX's rules with this construct list as its corpus.

## T0-MM — the 13 Mermaid types

One fixture per type from `mermaid-rs-renderer` 0.3.1's supported list
(flowchart/graph, sequence, class, state, ER, pie, xychart, quadrant, gantt,
timeline, journey, mindmap, gitGraph): **13/13** headers recognised and 72
statements scanned in 0.01 ms. S5 writes one parser/layout per type over the
shared diagram IR.

## T0-SVG — rasteriser gate

Six samples (shapes, path, **text**, polygon, gradient, groups+transform):

| Renderer | Non-blank samples |
| --- | --- |
| `srwiley/oksvg` + `rasterx` | 5/6 — **text renders 0 pixels** |
| `tdewolff/canvas` `ParseSVG` + `rasterizer` | text sample: 924 coloured pixels ✔ |

Decision: **`tdewolff/canvas` parses and rasterises SVG** (it draws text with
the system fonts); `oksvg` is dropped. The SVG task (S2) keeps the
6-sample visual gate and treats any future gap as a labelled box plus an open
parity row, never a silent blank.

## Dependencies for S1

`github.com/yuin/goldmark` (GFM block tree with source lines),
`github.com/go-text/typesetting` (shaping). Both pure Go; no cgo, so Linux,
macOS and Windows build.
