package preview

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func styleOf(n *flowNode) int {
	switch n.shape {
	case shapeRound, shapeStadium:
		return flowRound
	case shapeDecision:
		return flowDecision
	case shapeHexagon:
		return flowHexagon
	case shapeCircle:
		return flowCircle
	case shapeCylinder:
		return flowCylinder
	}
	return flowRect
}

// flowBoxSize is the outer size of a shape holding n lines of text tw wide.
func flowBoxSize(shape flowShape, tw, n int) (w, h int) {
	switch shape {
	case shapeStadium, shapeDecision, shapeHexagon:
		return tw + 6, n + 2
	case shapeCircle:
		return tw + 6, n + 4
	case shapeCylinder:
		return tw + 4, n + 3
	case shapeSubroutine:
		return tw + 8, n + 2
	}
	return tw + 4, n + 2
}

// drawFlowBox draws a node's outline and centres its text inside.
//
//	┌──────┐  ╭──────╮   ╭──────╮   ╱──────╲   ╭──────╮  ┌─┬──────┬─┐
//	│ rect │  │ round│  ( stadium)  < choice >  │╰────╯│  │ │ sub  │ │
//	└──────┘  ╰──────╯   ╰──────╯   ╲──────╱   │ db   │  └─┴──────┴─┘
//	                                           ╰──────╯
func drawFlowBox(cv *flowCanvas, n *flowNode) {
	st := styleOf(n)
	x0, y0, x1, y1 := n.x, n.y, n.x+n.w-1, n.y+n.h-1
	row := func(y, a, b int, left, fill, right rune) {
		cv.set(a, y, left, st)
		for x := a + 1; x < b; x++ {
			cv.set(x, y, fill, st)
		}
		cv.set(b, y, right, st)
	}
	clear := func(y, a, b int) {
		for x := a; x <= b; x++ {
			cv.set(x, y, ' ', flowText)
		}
	}
	// sides picks the left/right edge of text row i of rows for the slanted
	// shapes: a point for one row, a wedge for more.
	sides := func(i, rows int, one [2]rune) (rune, rune) {
		switch {
		case rows == 1:
			return one[0], one[1]
		case i == 0:
			return '╱', '╲'
		case i == rows-1:
			return '╲', '╱'
		}
		return '│', '│'
	}

	textTop, rows := y0+1, len(n.lines)
	switch n.shape {
	case shapeStadium, shapeDecision, shapeHexagon:
		top, bottom := [2]rune{'╭', '╮'}, [2]rune{'╰', '╯'}
		one := [2]rune{'(', ')'}
		if n.shape != shapeStadium {
			top, bottom, one = [2]rune{'╱', '╲'}, [2]rune{'╲', '╱'}, [2]rune{'<', '>'}
		}
		row(y0, x0+1, x1-1, top[0], '─', top[1])
		row(y1, x0+1, x1-1, bottom[0], '─', bottom[1])
		for i := range rows {
			l, r := sides(i, rows, one)
			clear(y0+1+i, x0+1, x1-1)
			cv.set(x0, y0+1+i, l, st)
			cv.set(x1, y0+1+i, r, st)
		}
	case shapeCircle:
		row(y0, x0+2, x1-2, '╭', '─', '╮')
		row(y1, x0+2, x1-2, '╰', '─', '╯')
		clear(y0+1, x0+2, x1-2)
		clear(y1-1, x0+2, x1-2)
		cv.set(x0+1, y0+1, '╱', st)
		cv.set(x1-1, y0+1, '╲', st)
		cv.set(x0+1, y1-1, '╲', st)
		cv.set(x1-1, y1-1, '╱', st)
		for i := range rows {
			clear(y0+2+i, x0+1, x1-1)
			cv.set(x0, y0+2+i, '│', st)
			cv.set(x1, y0+2+i, '│', st)
		}
		textTop = y0 + 2
	case shapeCylinder:
		row(y0, x0, x1, '╭', '─', '╮')
		row(y0+1, x0+1, x1-1, '╰', '─', '╯')
		cv.set(x0, y0+1, '│', st)
		cv.set(x1, y0+1, '│', st)
		for i := range rows {
			clear(y0+2+i, x0+1, x1-1)
			cv.set(x0, y0+2+i, '│', st)
			cv.set(x1, y0+2+i, '│', st)
		}
		row(y1, x0, x1, '╰', '─', '╯')
		textTop = y0 + 2
	case shapeSubroutine:
		row(y0, x0, x1, '┌', '─', '┐')
		row(y1, x0, x1, '└', '─', '┘')
		cv.set(x0+2, y0, '┬', st)
		cv.set(x1-2, y0, '┬', st)
		cv.set(x0+2, y1, '┴', st)
		cv.set(x1-2, y1, '┴', st)
		for i := range rows {
			clear(y0+1+i, x0+1, x1-1)
			for _, x := range []int{x0, x0 + 2, x1 - 2, x1} {
				cv.set(x, y0+1+i, '│', st)
			}
		}
	default:
		corners := [4]rune{'┌', '┐', '└', '┘'}
		if n.shape == shapeRound {
			corners = [4]rune{'╭', '╮', '╰', '╯'}
		}
		row(y0, x0, x1, corners[0], '─', corners[1])
		row(y1, x0, x1, corners[2], '─', corners[3])
		for i := range rows {
			clear(y0+1+i, x0+1, x1-1)
			cv.set(x0, y0+1+i, '│', st)
			cv.set(x1, y0+1+i, '│', st)
		}
	}

	for i, l := range n.lines {
		lw := ansi.StringWidth(l)
		cv.text(x0+(n.w-lw)/2, textTop+i, l, flowText)
	}
}

func renderFlow(body []string, th theme.Theme, width int, _ *[]Hit) []string {
	g := parseFlow(body)
	if len(g.nodes) == 0 {
		return []string{lipgloss.NewStyle().Foreground(th.GreyFg).Render("Empty flowchart: write lines like  a[Start] --> b[Next]")}
	}
	// Shrink the boxes until the diagram fits; a left-to-right diagram
	// that still doesn't fit is drawn top to bottom instead.
	var cv *flowCanvas
	for _, lr := range []bool{g.lr, false} {
		for _, wrapW := range []int{20, 14, 10, 6} {
			r := rankFlow(g, wrapW)
			if lr {
				cv = layoutLR(r)
			} else {
				cv = layoutTD(r)
			}
			if cv.w <= width {
				break
			}
		}
		if cv.w <= width || !lr {
			break
		}
	}

	styles := map[int]lipgloss.Style{
		flowLine:     lipgloss.NewStyle().Foreground(th.GreyFg2),
		flowArrow:    lipgloss.NewStyle().Foreground(th.GreyFg2),
		flowLabel:    lipgloss.NewStyle().Foreground(th.Cyan).Italic(true),
		flowText:     lipgloss.NewStyle().Foreground(th.Fg),
		flowRect:     lipgloss.NewStyle().Foreground(th.Blue),
		flowRound:    lipgloss.NewStyle().Foreground(th.Green),
		flowDecision: lipgloss.NewStyle().Foreground(th.Yellow),
		flowHexagon:  lipgloss.NewStyle().Foreground(th.Orange),
		flowCircle:   lipgloss.NewStyle().Foreground(th.Purple),
		flowCylinder: lipgloss.NewStyle().Foreground(th.Cyan),
	}
	lines := make([]string, cv.h)
	for y, row := range cv.cells {
		var b strings.Builder
		var run []rune
		cur := -1
		flush := func() {
			if len(run) > 0 {
				if cur < 0 {
					b.WriteString(string(run))
				} else {
					b.WriteString(styles[cur].Render(string(run)))
				}
			}
			run = run[:0]
		}
		end := len(row)
		for end > 0 && row[end-1].r == 0 && row[end-1].mask == 0 {
			end--
		}
		for _, c := range row[:end] {
			r, st := c.r, c.style
			switch {
			case r == 0 && c.mask != 0:
				r, st = lineRune(c.mask, c.kind), flowLine
			case r == 0:
				r, st = ' ', -1
			}
			if st != cur {
				flush()
				cur = st
			}
			run = append(run, r)
		}
		flush()
		lines[y] = b.String()
	}
	return lines
}
