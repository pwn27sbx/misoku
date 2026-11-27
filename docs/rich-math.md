# Rich view — maths engine (`internal/rich/math`)

Stage S4 of the rich reading view: real TeX layout instead of `texToUnicode`
glyphs. The engine is written from scratch in pure Go; the only face it needs
is bundled with the binary.

## What it does

```
source ──Parse──▶ AST ──layout──▶ box (width, height, depth, items) ──render──▶ RGBA
```

- `Parse` tokenises TeX maths, builds an AST (`ast.go`) and records anything it
  does not know in `Expr.Unknown` instead of dropping it.
- layout (`layout.go`) follows TeX's box model: four styles (display, text,
  script, scriptscript), atom classes with Appendix G's spacing table, fraction
  shifts, script shifts with collision avoidance, limits above and below for
  large operators and `\lim`-style functions, delimiter stretching, radicals,
  accents, matrices and alignment environments.
- the OpenType MATH table (`mathdata.go`) supplies the font's real size
  variants: a tall parenthesis, a radical for a fraction and a display-size
  `\sum` are the font's own glyphs, drawn as vector outlines, not a scaled
  copy of the base glyph. Variables are drawn in maths italic by default,
  digits upright.
- `render`/`Box` (`draw.go`, `math.go`) draw glyphs and rules into a
  transparent image and hand the page a `rich.Box`. `Box` caches by source,
  width, colour and cell size; a page rebuild reuses every formula that did not
  change.

Supported constructs (the acceptance corpus is `constructCorpus` in
`corpus_test.go`, one case per construct, plus one case per symbol command):

- fractions `\frac \dfrac \tfrac \cfrac`, `\binom \dbinom \tbinom`,
  `\sqrt[n]{}`;
- text `\text \mbox \textrm \textnormal \textup \textit \textbf`,
  `\operatorname` (also `\operatorname*`), `\mathrm \mathbf \boldsymbol \bm
  \mathbb \mathcal \mathscr \mathit \mathfrak \mathsf \mathtt`;
- `\left … \right` with every delimiter and `.`, `\big \Big \bigg \Bigg`
  (also `l`/`r`/`m` forms), `\middle`;
- environments `array` (column spec `lcr`, `p{}`), `subarray`, `aligned`,
  `alignedat`, `align`, `align*`, `split`, `eqnarray`, `cases`, `matrix`,
  `pmatrix`, `bmatrix`, `Bmatrix`, `vmatrix`, `Vmatrix`, `smallmatrix`;
- accents `\hat \widehat \tilde \widetilde \vec \dot \ddot \bar \overline
  \underline \check \breve \acute \grave \overrightarrow \overleftarrow
  \overleftrightarrow \overbrace \underbrace`;
- big operators `\sum \prod \coprod \bigcup \bigcap \bigoplus \bigotimes
  \bigodot \biguplus \int \iint \iiint \oint \oiint`;
- spacing `\, \: \; \! \quad \qquad \space \enspace \ ` and `~`;
- class overrides `\mathbin \mathrel \mathord \mathop \mathopen \mathclose
  \mathpunct`; `\overset \underset \stackrel`; `\xrightarrow
  \xleftarrow` with optional labels; `\not`; the Greek and symbol tables in
  `symbols.go`; `$x$`, `$$…$$` and ```` ```math ```` all reach the engine
  through `rich.Parse`.

## Font

`fonts/latinmodern-math.otf` (Latin Modern Math, GUST Font License — the
licence file ships beside it), embedded with `//go:embed`, 734 KB. It is the
only face added to the binary: text keeps the system faces the page discovers.
Maths glyphs must be TeX shaped, and coverage cannot be assumed on Linux,
macOS or Windows.

## Integration (one seam)

```go
opts := rich.LayoutOptions{Width: w, Fonts: fonts, Palette: pal}
opts.Math = func(b rich.Block) *rich.Box {
    return math.FromBlock(b, contentPx, math.Config{
        SizePx: fonts.BodyPx, CellW: fonts.CellW, CellH: fonts.CellH,
        Fg: pal.Fg,
    })
}
```

`contentPx` is the content width in pixels (`(Width-2*pageMargin) *
fonts.CellW`). The engine also has `math.Box(src []string, widthPx int,
Config)` for callers without a block, and `math.ResetCache()` for tests.

The badge in `internal/app/richview.go` still passes no `Math` option: that
file is outside this worktree's edit surface. Wiring it is the one-line change
above and is reported to the merge owner.

## Inline maths

`math.LayoutInline(src, Config)` lays a formula out in text style and returns
its width, ascent and depth plus a `Draw(dst, x, baseline)` that respects the
line's baseline (`TestMathInlineBaseline`). It is the seam for `$…$` inside a
paragraph. It cannot be wired yet: `rich.Parse` flattens inline maths into the
paragraph text, and the S1 display list has text runs but no inline box item.
That is a document/display-list contract change, reported rather than made
here.

## Limits (stated, not hidden)

- Delimiters, radicals and big operators take their sizes from the font's
  MATH table. Content taller than the largest variant (very rare on a page)
  falls back to scaling the base glyph; horizontal stretch for wide accents
  still uses that fallback.
- `\text{…}` draws with the maths font's upright letters, so `\text{if}` looks
  right. CJK, emoji and other scripts missing from the font draw as hollow
  boxes (never silently vanish); a text-face fallback is a later task.
- `\middle` uses a fixed 1.6× delimiter instead of matching `\left…\right`.
- Not implemented (rendered literally): `\over`, `\atop`, `\genfrac`,
  `\substack`, `\color`, `\phantom`, `\tag` content, `\hline` in arrays.
- No optical sizing: one face, scaled.

## Credits

- mddog (`github.com/qu8n/mddog`, MIT) was read as a **reference for which
  constructs a viewer must support** (its `src/render/math.rs` command table
  and its fixture set). No code was translated; the parser, box model and
  renderer here are written from TeX's rules (TeXbook Appendix G).
- The reduced fixtures in `testdata/reduced/` come from the documents mddog
  tests with; each source and licence is listed in
  `testdata/reduced/SOURCES.md`.
- Latin Modern Math — GUST Font License, see `fonts/GUST-FONT-LICENSE.txt`.

## Tests

- `TestMathCorpus` — one case per construct (plan §3.6.2); `TestMathCorpusTable`
  — every symbol command.
- `TestReducedFixturesParse` / `TestReducedFixturesLayOut` — 248 formulas from
  real documents: parse, no unknown command, non-empty layout, non-nil box.
- `TestMathForms` — `$$…$$` and ```` ```math ```` become page boxes through
  `rich.Layout`, source lines intact.
- `TestMathGolden` — PNG goldens per construct family (bundled font, so the
  pixels are identical on Linux, macOS and Windows).
- `TestMathMatrixColumnSpec`, `TestMathInlineBaseline`,
  `TestMathFractionAndScriptGeometry`, `TestMathDelimitersStretch`,
  `TestMathBigOpLimits`, `TestMathEnvironmentsLayout`, `TestMathBoxFitsWidth`,
  `TestMathBoxCache`, `TestMathBoxWithinBudget`.
- `BenchmarkMathLayout` / `BenchmarkMathDraw` (2026-11-27, Ryzen 7 4800H):
  layout and rasterise 172 µs, cached draw 50 µs — budgets are 15 ms / 5 ms.
