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
)

// Layout holds the screen geometry of every pane. X offsets are absolute
// columns; all body panes start at row BodyY and are BodyH rows tall.
type Layout struct {
	SidebarW int
	EditorX  int
	EditorW  int
	PreviewX int
	PreviewW int
	BodyY    int
	BodyH    int
}

// Split holds where the user dragged the pane dividers. Zero values mean the
// defaults: a SidebarWidth explorer and an even editor/preview split.
type Split struct {
	SidebarW   int     // explorer width
	EditorFrac float64 // editor's share of the editor+preview width
}

// CalculateLayout splits the window into explorer, editor and preview. A
// one-column divider separates neighbouring panes. In reading mode the editor
// gets no columns and the preview takes the rest, whatever the width and
// whether the preview was open.
func CalculateLayout(totalWidth, totalHeight int, sidebarOpen, previewOpen, reading bool, split Split) Layout {
	l := Layout{BodyY: tablineHeight, BodyH: max(totalHeight-tablineHeight-footerHeight, 1)}
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
		l.EditorX, l.PreviewX, l.PreviewW = x, x, remain
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
