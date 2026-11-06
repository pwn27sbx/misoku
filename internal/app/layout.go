package app

const (
	// SidebarWidth is the explorer width when open (nvim-tree's default).
	SidebarWidth = 30
	// MinTermWidth is the minimum required terminal width.
	MinTermWidth = 60
	// MinTermHeight is the minimum required terminal height.
	MinTermHeight = 12
	// minPreviewTotal is the narrowest terminal that still shows the preview.
	minPreviewTotal = 90

	// minSidebarW and maxSidebarW bound a dragged explorer width.
	minSidebarW = 16
	maxSidebarW = 60
	// minPaneW is the narrowest a divider drag may make the editor or preview.
	minPaneW = 20

	tablineHeight = 1
	footerHeight  = 1 // statusline (the ":" prompt takes its place while typing)

	// notePadY is the blank rows above and below the note in the editor and
	// the preview, so the text does not touch the tab bar or the statusline.
	notePadY = 1
	// minPaddedBodyH is the shortest body that keeps those rows: a shorter
	// one gives them back to the text.
	minPaddedBodyH = 8
)

// Layout holds the screen geometry of every pane. X offsets are absolute
// columns; all body panes start at row BodyY and are BodyH rows tall. The
// note's panes (editor and preview) keep PadY blank rows at the top and the
// bottom of that body, so their text starts at NoteY and is NoteH rows tall.
type Layout struct {
	SidebarW int
	EditorX  int
	EditorW  int
	PreviewX int
	PreviewW int
	BodyY    int
	BodyH    int
	PadY     int
}

// NoteY is the screen row of the note's first text row in the editor and
// the preview.
func (l Layout) NoteY() int { return l.BodyY + l.PadY }

// NoteH is how many text rows the editor and the preview have between their
// padding rows.
func (l Layout) NoteH() int { return max(l.BodyH-2*l.PadY, 1) }

// Split holds where the user dragged the pane dividers. Zero values mean the
// defaults: a SidebarWidth explorer and an even editor/preview split.
type Split struct {
	SidebarW   int     // explorer width
	EditorFrac float64 // editor's share of the editor+preview width
}

// CalculateLayout splits the window into explorer, editor and preview. A
// one-column divider separates neighbouring panes. In reading mode the
// preview gets no columns and the editor takes the rest, whatever the width
// and whether the preview was open.
func CalculateLayout(totalWidth, totalHeight int, sidebarOpen, previewOpen, reading bool, split Split) Layout {
	l := Layout{BodyY: tablineHeight, BodyH: max(totalHeight-tablineHeight-footerHeight, 1)}
	if l.BodyH >= minPaddedBodyH {
		l.PadY = notePadY
	}
	if totalWidth < MinTermWidth || totalHeight < MinTermHeight {
		return l
	}

	x := 0
	if sidebarOpen {
		w := SidebarWidth
		if split.SidebarW > 0 {
			w = clamp(split.SidebarW, minSidebarW, min(maxSidebarW, totalWidth-1-minPaneW))
		}
		l.SidebarW = w
		x = w + 1
	}
	remain := totalWidth - x

	if reading {
		l.EditorX, l.EditorW = x, remain
		return l
	}
	if previewOpen && totalWidth >= minPreviewTotal {
		avail := remain - 1
		l.EditorW = avail / 2
		if split.EditorFrac > 0 {
			lo := min(minPaneW, avail/2)
			l.EditorW = clamp(int(float64(avail)*split.EditorFrac+0.5), lo, avail-lo)
		}
		l.PreviewW = remain - 1 - l.EditorW
		l.EditorX = x
		l.PreviewX = x + l.EditorW + 1
		return l
	}
	l.EditorX = x
	l.EditorW = remain
	return l
}

func clamp(v, lo, hi int) int {
	return max(lo, min(v, hi))
}
