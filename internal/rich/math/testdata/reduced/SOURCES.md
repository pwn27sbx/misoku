# Reduced maths fixtures — sources

Each `.tex` file is one formula reduced from a document that mddog's own
fixtures use (`github.com/qu8n/mddog`, MIT, `fixtures/math/`, 2026-11-27). They
are kept here only as test inputs: the engine must lay out formulas written for
real documents, not just the construct table.

| Files | Upstream document | Licence |
| --- | --- | --- |
| `meltcalc-theory-*.tex` | https://github.com/robertsamples/meltcalc/blob/5f4e1ff3b643bec873a38a599e46bbd7f76e58a3/theory.md | MIT |
| `ibl-atlas-readme-*.tex` | https://github.com/int-brain-lab/atlas/blob/2369c06296b972409a2979c26f4ecc424d7bc80b/README.md | MIT |
| `transformers-blocks-readme-*.tex` (from `transformer-blocks-readme.md`) | https://github.com/tonyduan/transformer-blocks/blob/42c0fca7b21ca0a872ede0fe714ead2bb960e141/README.md | MIT |
| `ai-whys-dl-*.tex` | https://github.com/ZHYsfl/ai-whys/blob/534e0066be12659be6cc90861d81928690809c5c/dl.md | Apache-2.0 |
| `ros-ch18-*.tex` | https://github.com/ASKurz/Working-through-Regression-and-other-stories/blob/ac87539c2e901598a425875333d5af7126d818ab/18.md | CC0-1.0 |

Reduction: display (`$$…$$`) and inline (`$…$`) maths were extracted, runs of
whitespace collapsed, duplicates dropped, and formulas shorter than 6 or longer
than 300 characters skipped. Nothing was rewritten: a formula that needs a
command the engine lacks fails `TestReducedFixturesParse`, which is the point.

The Apache-2.0 source is used under the Apache License, Version 2.0
(http://www.apache.org/licenses/LICENSE-2.0).
