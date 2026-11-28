// Package diagram parses Mermaid source into a native Go intermediate
// representation and lays it out as vector commands for the rich reading view.
// It has no dependency on Mermaid.js, mmdc, or the text preview renderer.
package diagram

import (
	"fmt"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// Kind identifies one of the thirteen Mermaid diagram families in S5.
type Kind string

const (
	// KindFlowchart is Mermaid's graph/flowchart family.
	KindFlowchart Kind = "flowchart"
	// KindSequence is the sequenceDiagram family.
	KindSequence Kind = "sequence"
	// KindClass is the classDiagram family.
	KindClass Kind = "class"
	// KindState is the stateDiagram-v2 family.
	KindState Kind = "state"
	// KindER is the erDiagram family.
	KindER Kind = "er"
	// KindPie is the pie family.
	KindPie Kind = "pie"
	// KindXY is the xychart-beta family.
	KindXY Kind = "xychart"
	// KindQuadrant is the quadrantChart family.
	KindQuadrant Kind = "quadrant"
	// KindGantt is the gantt family.
	KindGantt Kind = "gantt"
	// KindTimeline is the timeline family.
	KindTimeline Kind = "timeline"
	// KindJourney is the journey family.
	KindJourney Kind = "journey"
	// KindMindmap is the mindmap family.
	KindMindmap Kind = "mindmap"
	// KindGit is the gitGraph family.
	KindGit Kind = "gitGraph"
)

// NodeShape is the geometry of a flowchart or mindmap node.
type NodeShape int

const (
	// ShapeRect is a plain rectangle.
	ShapeRect NodeShape = iota
	// ShapeRounded is a rectangle with rounded corners.
	ShapeRounded
	// ShapeStadium is a fully rounded rectangle.
	ShapeStadium
	// ShapeSubroutine is a rectangle with side bars.
	ShapeSubroutine
	// ShapeCylinder is a database cylinder.
	ShapeCylinder
	// ShapeCircle is an ellipse; equal width and height make it a circle.
	ShapeCircle
	// ShapeDiamond is a decision diamond.
	ShapeDiamond
	// ShapeHexagon is a six-sided preparation box.
	ShapeHexagon
	// ShapeParallelogram is an input/output parallelogram.
	ShapeParallelogram
	// ShapeParallelogramAlt is the opposite-slanted parallelogram.
	ShapeParallelogramAlt
	// ShapeDoubleCircle is a start/end state.
	ShapeDoubleCircle
	// ShapeAsymmetric is a flag-shaped asymmetric node.
	ShapeAsymmetric
)

// EdgeStyle controls the stroke of a connector.
type EdgeStyle int

const (
	// EdgeSolid is a solid line.
	EdgeSolid EdgeStyle = iota
	// EdgeDotted is a dotted line.
	EdgeDotted
	// EdgeThick is a thick solid line.
	EdgeThick
	// EdgeInvisible occupies layout space without drawing.
	EdgeInvisible
)

// Arrowhead controls the decoration at one end of a connector.
type Arrowhead int

const (
	// ArrowNone draws no arrowhead.
	ArrowNone Arrowhead = iota
	// ArrowOpen draws an open chevron.
	ArrowOpen
	// ArrowFilled draws a filled triangular arrowhead.
	ArrowFilled
	// ArrowCircle draws a small circle at the endpoint.
	ArrowCircle
	// ArrowCross draws a cross at the endpoint.
	ArrowCross
)

// Diagram is the shared semantic IR. Graph families use Nodes, Edges and
// Groups; the other families put their typed records in Data.
type Diagram struct {
	Kind      Kind
	Title     string
	Direction string
	Nodes     []Node
	Edges     []Edge
	Groups    []Group
	Data      any
}

// Node is a labelled graph node.
type Node struct {
	ID    string
	Label string
	Shape NodeShape
	Group string
}

// Edge is a directed connector between graph nodes.
type Edge struct {
	From  string
	To    string
	Label string
	Start Arrowhead
	End   Arrowhead
	Style EdgeStyle
}

// Group is a subgraph or composite container.
type Group struct {
	ID        string
	Label     string
	Nodes     []string
	Direction string
}

// ConstructError names a source construct that the native renderer cannot
// draw. Callers must show the source with this note instead of guessing.
type ConstructError struct {
	Kind      Kind
	Line      int
	Construct string
}

// Error implements error.
func (e *ConstructError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Line > 0 {
		return fmt.Sprintf("%s line %d: unsupported construct %q", e.Kind, e.Line, e.Construct)
	}
	return fmt.Sprintf("%s: unsupported construct %q", e.Kind, e.Construct)
}

type parseFunc func([]string) (Diagram, error)
type headerParseFunc func(string, []string) (Diagram, error)
type layoutFunc func(Diagram, *rich.Fonts, rich.Palette) (Scene, error)
