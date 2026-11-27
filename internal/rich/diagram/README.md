# Native Mermaid diagrams

`internal/rich/diagram` renders all thirteen S5 Mermaid families directly in Go. It returns the S1 `rich.Box{W,H,Draw}` seam; it does not import the text preview, call Mermaid.js, or start `mmdc`.

## Pipeline

```text
Mermaid source → Diagram IR → Scene (cell-sized vector commands) → cached RGBA → rich.Box.Draw
```

- `Parse` dispatches by the root header. Flowchart and pie use `registerWithHeader` because their headers carry direction or `showData` semantics.
- `Diagram` stores graph nodes, edges and containers; other families keep typed records in `Diagram.Data`.
- `Layout` produces deterministic `Scene` commands in terminal-cell units.
- `Renderer` caches one bitmap per source, font stack and palette. `Draw` only clips and blits it.
- Unsupported syntax returns `*ConstructError`; the box shows the source plus the reason. It never silently draws a different diagram.

## Supported families

| Family | Kind |
| --- | --- |
| Flowchart | `KindFlowchart` |
| Sequence | `KindSequence` |
| Class | `KindClass` |
| State | `KindState` |
| ER | `KindER` |
| Pie | `KindPie` |
| XY chart | `KindXY` |
| Quadrant chart | `KindQuadrant` |
| Gantt | `KindGantt` |
| Timeline | `KindTimeline` |
| Journey | `KindJourney` |
| Mindmap | `KindMindmap` |
| Git graph | `KindGit` |

## Verification

```bash
go test ./internal/rich/diagram -count=1
go test ./internal/rich/diagram -run TestDiagramIR -count=1
go test ./internal/rich/diagram -bench BenchmarkDiagram -benchmem
```

Golden tests use only the embedded Go fonts. Regenerate deliberately with `-update`, open every changed PNG and inspect it by eye, then rerun without `-update`. Tests alone do not prove that labels are visually complete.

## Credits and scope

The type list and construct vocabulary are based on Mermaid's public syntax and the MIT-licensed `mermaid-rs-renderer` 0.3.1 documentation. No renderer source was translated or copied. Misoku's existing flowchart semantics and layered longest-path idea are adapted into this package; `internal/preview` remains untouched because the S1 contract forbids cross-package imports.
