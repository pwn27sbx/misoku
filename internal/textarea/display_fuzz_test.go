package textarea

import "testing"

var displayFuzzLines = []string{
	"",
	"short",
	"a much longer line that certainly wraps at a narrow editor width",
	"漢字 かな 漢字 かな 漢字 かな 漢字 かな",
	"word word word word word word word word word",
}

// fuzzByte reads the next input byte, or 0 once the input is exhausted.
func fuzzByte(data []byte, at *int) int {
	if *at >= len(data) {
		return 0
	}
	b := data[*at]
	*at++
	return int(b)
}

// fuzzDisplay builds a display of n lines of mixed heights.
func fuzzDisplay(data []byte, at *int, n int) *Display {
	d := &Display{Lines: make([]DisplayLine, n)}
	for i := range d.Lines {
		switch fuzzByte(data, at) % 5 {
		case 0, 1:
			d.Lines[i] = DisplayLine{Raw: true}
		case 2:
			d.Lines[i] = DisplayLine{Hidden: true}
		case 3:
			d.Lines[i] = DisplayLine{}
		default:
			d.Lines[i] = DisplayLine{Rows: make([]string, 1+fuzzByte(data, at)%6)}
		}
	}
	for range fuzzByte(data, at) % 7 {
		d.Footer = append(d.Footer, "footer")
	}
	return d
}

// refDisplayStarts recomputes the row index from the display, independently
// of Model.rows. Its last element is the row just past the text.
func refDisplayStarts(m *Model) []int {
	starts := make([]int, 0, len(m.value)+1)
	acc := 0
	for l, v := range m.value {
		starts = append(starts, acc)
		if m.display != nil && len(m.display.Lines) == len(m.value) && !m.display.Lines[l].Raw {
			acc += len(m.display.Lines[l].Rows)
		} else {
			acc += len(wrap(v, m.width))
		}
	}
	return append(starts, acc)
}

// checkDisplayInvariants fails unless the cursor is valid and its row is on
// screen. A footer taller than the screen is the one place the screen may
// hold no text, and so no cursor.
func checkDisplayInvariants(t *testing.T, m *Model, op string, wasHidden bool) {
	t.Helper()
	if m.row < 0 || m.row >= len(m.value) || m.col < 0 || m.col > len(m.value[m.row]) {
		t.Fatalf("after %s: invalid cursor (%d, %d)", op, m.row, m.col)
	}
	starts := refDisplayStarts(m)
	first := starts[m.row]
	var last int
	rendered := m.display != nil && len(m.display.Lines) == len(m.value) && !m.display.Lines[m.row].Raw
	if rendered {
		last = max(first, starts[m.row+1]-1)
	} else {
		first += m.LineInfo().RowOffset
		last = first
	}
	off, h := m.viewport.YOffset, m.viewport.Height
	if off < starts[len(m.value)] && (last < off || first >= off+h) {
		t.Fatalf("after %s: cursor rows %d..%d off the screen %d..%d", op, first, last, off, off+h-1)
	}
	if !wasHidden && rendered && m.display.Lines[m.row].Hidden && op == "motion" {
		t.Fatalf("after %s: the cursor landed on hidden line %d", op, m.row)
	}
	_ = m.View()
}

// FuzzDisplay runs random motions and clicks over random mixed heights.
func FuzzDisplay(f *testing.F) {
	f.Add([]byte{3, 1, 7, 2, 4, 1, 2, 3, 4, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11})
	f.Add([]byte{0, 9, 40, 2, 2, 2, 2, 2, 2, 6, 1, 8, 1, 200, 8, 8, 3, 9, 77})
	f.Fuzz(func(t *testing.T, data []byte) {
		at := 0
		m := New()
		m.Prompt = ""
		m.ShowLineNumbers = false
		m.SetWidth(8 + fuzzByte(data, &at)%24)
		m.SetHeight(1 + fuzzByte(data, &at)%12)

		n := 1 + fuzzByte(data, &at)%20
		text := ""
		for i := range n {
			if i > 0 {
				text += "\n"
			}
			text += displayFuzzLines[fuzzByte(data, &at)%len(displayFuzzLines)]
		}
		m.SetValue(text)
		m.SetDisplay(fuzzDisplay(data, &at, len(m.value)))

		for op := 0; at < len(data) && op < 300; op++ {
			wasHidden := m.display != nil && len(m.display.Lines) == len(m.value) &&
				!m.display.Lines[m.row].Raw && m.display.Lines[m.row].Hidden
			name := "motion"
			arg := fuzzByte(data, &at)
			switch arg % 11 {
			case 0:
				m.MoveCursorBy(1 + arg%5)
			case 1:
				m.MoveCursorBy(-1 - arg%5)
			case 2:
				name = "ScrollBy"
				m.ScrollBy(arg%21 - 10)
			case 3:
				// ctrl+d and ctrl+u.
				name = "half page"
				half := 1 + arg%6
				if arg%2 == 0 {
					half = -half
				}
				m.ScrollBy(half)
				m.MoveCursorBy(half)
			case 4:
				name = "ClickAt"
				m.ClickAt(arg%40, fuzzByte(data, &at)%30)
			case 5:
				name = "SetDisplay"
				m.SetDisplay(fuzzDisplay(data, &at, len(m.value)))
			case 6:
				name = "SetDisplay(nil)"
				m.SetDisplay(nil)
			case 7:
				name = "goto"
				m.SetRowCol(arg%len(m.value), arg)
			case 8:
				name = "goto"
				if arg%2 == 0 {
					m.GotoTop()
				} else {
					m.GotoBottom()
				}
			case 9:
				m.CursorDown()
				m.EnsureVisible()
			case 10:
				m.CursorUp()
				m.EnsureVisible()
			}
			checkDisplayInvariants(t, &m, name, wasHidden)
			if v := fuzzByte(data, &at); v%13 == 0 {
				line, offset, footer := m.DisplayRowAt(v - 3)
				if line < 0 || line >= len(m.value) || offset < 0 {
					t.Fatalf("DisplayRowAt(%d) = (%d, %d, %v)", v-3, line, offset, footer)
				}
			}
		}
	})
}
