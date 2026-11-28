package diagram

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// PieData is the semantic payload of a pie chart.
type PieData struct {
	ShowData bool
	Slices   []PieSlice
}

// PieSlice is one labelled value in a pie chart.
type PieSlice struct {
	Label string
	Value float64
}

// QuadrantData is the semantic payload of a quadrant chart.
type QuadrantData struct {
	XAxis  QuadrantAxis
	YAxis  QuadrantAxis
	Labels [4]string
	Points []QuadrantPoint
}

// QuadrantAxis contains the labels at the low and high ends of an axis.
type QuadrantAxis struct {
	From string
	To   string
}

// QuadrantPoint is a named point in normalized quadrant coordinates.
type QuadrantPoint struct {
	Label string
	X     float64
	Y     float64
}

// TimelineData is the semantic payload of a timeline.
type TimelineData struct {
	Sections []TimelineSection
}

// TimelineSection is an optional heading and its ordered events.
type TimelineSection struct {
	Title  string
	Events []TimelineEvent
}

// TimelineEvent is one event and its displayed period or period range.
type TimelineEvent struct {
	Period string
	Label  string
}

// JourneyData is the semantic payload of a journey diagram.
type JourneyData struct {
	Sections []JourneySection
}

// JourneySection is an optional heading and its ordered tasks.
type JourneySection struct {
	Title string
	Tasks []JourneyTask
}

// JourneyTask is a scored task and the actors taking part in it.
type JourneyTask struct {
	Label  string
	Score  int
	Actors []string
}

// isSpace reports whether b starts whitespace, for the word checks in the
// header parsers.
func isSpace(b byte) bool { return b == ' ' || b == '\t' }

func init() {
	registerWithHeader(KindPie, parsePie, layoutPie)
	register(KindQuadrant, parseQuadrant, layoutQuadrant)
	register(KindTimeline, parseTimeline, layoutTimeline)
	register(KindJourney, parseJourney, layoutJourney)
}

func parsePie(header string, lines []string) (Diagram, error) {
	fields := strings.Fields(header)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "pie") {
		return Diagram{}, unknown(KindPie, 1, header)
	}
	d := Diagram{Kind: KindPie}
	data := PieData{}
	// Mermaid accepts the title on the header line ("pie title Pets"), after
	// an optional showData, as well as on a line of its own.
	rest := strings.TrimSpace(header[len(fields[0]):])
	if len(rest) >= len("showData") && strings.EqualFold(rest[:len("showData")], "showData") &&
		(len(rest) == len("showData") || isSpace(rest[len("showData")])) {
		data.ShowData = true
		rest = strings.TrimSpace(rest[len("showData"):])
	}
	if rest != "" {
		if len(rest) < len("title") || !strings.EqualFold(rest[:len("title")], "title") ||
			(len(rest) > len("title") && !isSpace(rest[len("title")])) {
			return Diagram{}, unknown(KindPie, 1, header)
		}
		title := unquote(strings.TrimSpace(rest[len("title"):]))
		if title == "" {
			return Diagram{}, unknown(KindPie, 1, header)
		}
		d.Title = title
	}
	var firstLine sourceLine
	var firstSlice sourceLine
	for _, line := range numberedNonBlank(lines) {
		if firstLine.Text == "" {
			firstLine = line
		}
		keyword := strings.Fields(line.Text)[0]
		switch keyword {
		case "title":
			if d.Title != "" {
				return Diagram{}, unknown(KindPie, line.Number, line.Text)
			}
			title := strings.TrimSpace(line.Text[len("title"):])
			if title == "" {
				return Diagram{}, unknown(KindPie, line.Number, line.Text)
			}
			d.Title = unquote(title)
		case "showData":
			if line.Text != "showData" || data.ShowData {
				return Diagram{}, unknown(KindPie, line.Number, line.Text)
			}
			data.ShowData = true
		default:
			parts := splitTopLevelString(line.Text, ':')
			if len(parts) != 2 || !basicQuoted(parts[0]) {
				return Diagram{}, unknown(KindPie, line.Number, line.Text)
			}
			label := unquote(parts[0])
			value, err := parseFloat(parts[1])
			if strings.TrimSpace(label) == "" || err != nil || !basicFinite(value) || value < 0 {
				return Diagram{}, unknown(KindPie, line.Number, line.Text)
			}
			if firstSlice.Text == "" {
				firstSlice = line
			}
			data.Slices = append(data.Slices, PieSlice{Label: label, Value: value})
		}
	}
	if len(data.Slices) == 0 {
		return Diagram{}, unknown(KindPie, firstLine.Number, firstLine.Text)
	}
	total := 0.0
	for _, slice := range data.Slices {
		total += slice.Value
	}
	if !basicFinite(total) || total <= 0 {
		return Diagram{}, unknown(KindPie, firstSlice.Number, firstSlice.Text)
	}
	d.Data = data
	return d, nil
}

func parseQuadrant(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindQuadrant}
	data := QuadrantData{}
	var firstLine sourceLine
	var xSeen, ySeen bool
	var labelSeen [4]bool
	for _, line := range numberedNonBlank(lines) {
		if firstLine.Text == "" {
			firstLine = line
		}
		keyword := strings.Fields(line.Text)[0]
		switch keyword {
		case "title":
			if d.Title != "" {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			title := strings.TrimSpace(line.Text[len("title"):])
			if title == "" {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			d.Title = unquote(title)
		case "x-axis":
			if xSeen {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			axis, ok := basicAxis(line.Text[len("x-axis"):])
			if !ok {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			data.XAxis = axis
			xSeen = true
		case "y-axis":
			if ySeen {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			axis, ok := basicAxis(line.Text[len("y-axis"):])
			if !ok {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			data.YAxis = axis
			ySeen = true
		default:
			if strings.HasPrefix(line.Text, "quadrant-") {
				remainder := strings.TrimSpace(strings.TrimPrefix(line.Text, "quadrant-"))
				fields := strings.Fields(remainder)
				if len(fields) == 0 {
					return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
				}
				index, err := parseInt(fields[0])
				if err != nil || index < 1 || index > 4 || labelSeen[index-1] {
					return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
				}
				label := strings.TrimSpace(remainder[len(fields[0]):])
				if label == "" {
					return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
				}
				data.Labels[index-1] = unquote(label)
				labelSeen[index-1] = true
				continue
			}
			parts := splitTopLevelString(line.Text, ':')
			if len(parts) != 2 {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			label := unquote(parts[0])
			coordinateText := strings.TrimSpace(parts[1])
			if strings.TrimSpace(label) == "" || len(coordinateText) < 2 || coordinateText[0] != '[' || coordinateText[len(coordinateText)-1] != ']' {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			coordinates := splitTopLevelString(coordinateText[1:len(coordinateText)-1], ',')
			if len(coordinates) != 2 {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			x, xErr := parseFloat(coordinates[0])
			y, yErr := parseFloat(coordinates[1])
			if xErr != nil || yErr != nil || !basicFinite(x) || !basicFinite(y) || x < 0 || x > 1 || y < 0 || y > 1 {
				return Diagram{}, unknown(KindQuadrant, line.Number, line.Text)
			}
			data.Points = append(data.Points, QuadrantPoint{Label: label, X: x, Y: y})
		}
	}
	if !xSeen || !ySeen {
		return Diagram{}, unknown(KindQuadrant, firstLine.Number, firstLine.Text)
	}
	d.Data = data
	return d, nil
}

func parseTimeline(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindTimeline}
	data := TimelineData{}
	var firstLine sourceLine
	var sectionLines []sourceLine
	current := -1
	eventCount := 0
	for _, line := range numberedNonBlank(lines) {
		if firstLine.Text == "" {
			firstLine = line
		}
		keyword := strings.Fields(line.Text)[0]
		switch keyword {
		case "title":
			if d.Title != "" {
				return Diagram{}, unknown(KindTimeline, line.Number, line.Text)
			}
			title := strings.TrimSpace(line.Text[len("title"):])
			if title == "" {
				return Diagram{}, unknown(KindTimeline, line.Number, line.Text)
			}
			d.Title = unquote(title)
		case "section":
			title := strings.TrimSpace(line.Text[len("section"):])
			if title == "" {
				return Diagram{}, unknown(KindTimeline, line.Number, line.Text)
			}
			data.Sections = append(data.Sections, TimelineSection{Title: unquote(title)})
			sectionLines = append(sectionLines, line)
			current = len(data.Sections) - 1
		default:
			parts := splitTopLevelString(line.Text, ':')
			period := ""
			if len(parts) >= 2 {
				period = unquote(parts[0])
			}
			if len(parts) < 2 || strings.TrimSpace(period) == "" {
				return Diagram{}, unknown(KindTimeline, line.Number, line.Text)
			}
			if current < 0 {
				data.Sections = append(data.Sections, TimelineSection{})
				sectionLines = append(sectionLines, sourceLine{})
				current = 0
			}
			for _, eventText := range parts[1:] {
				label := unquote(eventText)
				if strings.TrimSpace(label) == "" {
					return Diagram{}, unknown(KindTimeline, line.Number, line.Text)
				}
				data.Sections[current].Events = append(data.Sections[current].Events, TimelineEvent{Period: period, Label: label})
				eventCount++
			}
		}
	}
	if eventCount == 0 {
		return Diagram{}, unknown(KindTimeline, firstLine.Number, firstLine.Text)
	}
	for i := range data.Sections {
		if len(data.Sections[i].Events) == 0 {
			line := sectionLines[i]
			if line.Text == "" {
				line = firstLine
			}
			return Diagram{}, unknown(KindTimeline, line.Number, line.Text)
		}
	}
	d.Data = data
	return d, nil
}

func parseJourney(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindJourney}
	data := JourneyData{}
	var firstLine sourceLine
	var sectionLines []sourceLine
	current := -1
	taskCount := 0
	for _, line := range numberedNonBlank(lines) {
		if firstLine.Text == "" {
			firstLine = line
		}
		keyword := strings.Fields(line.Text)[0]
		switch keyword {
		case "title":
			if d.Title != "" {
				return Diagram{}, unknown(KindJourney, line.Number, line.Text)
			}
			title := strings.TrimSpace(line.Text[len("title"):])
			if title == "" {
				return Diagram{}, unknown(KindJourney, line.Number, line.Text)
			}
			d.Title = unquote(title)
		case "section":
			title := strings.TrimSpace(line.Text[len("section"):])
			if title == "" {
				return Diagram{}, unknown(KindJourney, line.Number, line.Text)
			}
			data.Sections = append(data.Sections, JourneySection{Title: unquote(title)})
			sectionLines = append(sectionLines, line)
			current = len(data.Sections) - 1
		default:
			parts := splitTopLevelString(line.Text, ':')
			if len(parts) != 3 {
				return Diagram{}, unknown(KindJourney, line.Number, line.Text)
			}
			label := unquote(parts[0])
			score, err := parseInt(parts[1])
			if strings.TrimSpace(label) == "" || err != nil || score < 1 || score > 5 {
				return Diagram{}, unknown(KindJourney, line.Number, line.Text)
			}
			actorTexts := splitTopLevelString(parts[2], ',')
			actors := make([]string, 0, len(actorTexts))
			seen := make(map[string]struct{}, len(actorTexts))
			for _, actorText := range actorTexts {
				actor := unquote(actorText)
				if actor == "" {
					return Diagram{}, unknown(KindJourney, line.Number, line.Text)
				}
				if _, exists := seen[actor]; exists {
					return Diagram{}, unknown(KindJourney, line.Number, line.Text)
				}
				seen[actor] = struct{}{}
				actors = append(actors, actor)
			}
			if current < 0 {
				data.Sections = append(data.Sections, JourneySection{})
				sectionLines = append(sectionLines, sourceLine{})
				current = 0
			}
			data.Sections[current].Tasks = append(data.Sections[current].Tasks, JourneyTask{Label: label, Score: score, Actors: actors})
			taskCount++
		}
	}
	if taskCount == 0 {
		return Diagram{}, unknown(KindJourney, firstLine.Number, firstLine.Text)
	}
	for i := range data.Sections {
		if len(data.Sections[i].Tasks) == 0 {
			line := sectionLines[i]
			if line.Text == "" {
				line = firstLine
			}
			return Diagram{}, unknown(KindJourney, line.Number, line.Text)
		}
	}
	d.Data = data
	return d, nil
}

func layoutPie(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, err := basicPieData(d)
	if err != nil {
		return Scene{}, err
	}
	if fonts == nil {
		return Scene{}, invalid(KindPie, 0, "fonts are required")
	}
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	const chartW, chartH = 26, 13
	type legendEntry struct {
		lines  []string
		text   string
		width  int
		height int
	}
	entries := make([]legendEntry, len(data.Slices))
	legendWidth, legendHeight := 0, 0
	for i, slice := range data.Slices {
		text := slice.Label
		if data.ShowData {
			text += ": " + strconv.FormatFloat(slice.Value, 'g', -1, 64)
		}
		lines, width, lineHeight := labelSize(text, fonts, 24, fonts.BodyPx, false)
		height := len(lines) * lineHeight
		entries[i] = legendEntry{lines: lines, text: text, width: width, height: height}
		legendWidth = max(legendWidth, width)
		legendHeight += height + 1
	}
	s := newScene()
	s.Width = max(titleWidth+2, 1+chartW+3+legendWidth+2)
	s.Height = titleHeight + max(chartH, legendHeight) + 2
	if titleHeight > 0 {
		s.TextLinesBold(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	chartX, chartY := 1, titleHeight+1
	total := 0.0
	for _, slice := range data.Slices {
		total += slice.Value
	}
	start := -math.Pi / 2
	for i, slice := range data.Slices {
		if slice.Value == 0 {
			continue
		}
		share := slice.Value / total
		end := start + 2*math.Pi*share
		col := accent(palette, i)
		s.Sector(chartX, chartY, chartW, chartH, start, end, col)
		if share >= 0.08 {
			mid := (start + end) / 2
			lx := chartX + chartW/2 + int(math.Round(float64(chartW)*0.28*math.Cos(mid))) - 2
			ly := chartY + chartH/2 + int(math.Round(float64(chartH)*0.28*math.Sin(mid)))
			pct := strconv.Itoa(int(math.Round(share*100))) + "%"
			s.TextBold(pct, lx, ly, 5, 1, fonts.BodyPx*0.9, AlignCenter, contrastText(col, palette), false)
		}
		start = end
	}
	s.Circle(chartX, chartY, chartW, chartH, color.RGBA{}, rule(palette))
	legendX := chartX + chartW + 3
	legendY := chartY + 1
	for i, entry := range entries {
		s.Rect(legendX, legendY, 2, 1, accent(palette, i), color.RGBA{})
		lineHeight := max(entry.height/len(entry.lines), 1)
		s.TextLines(entry.lines, legendX+3, legendY, entry.width, lineHeight, fonts.BodyPx, AlignLeft, foreground(palette), false)
		legendY += entry.height + 1
	}
	return s, nil
}

func layoutQuadrant(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, err := basicQuadrantData(d)
	if err != nil {
		return Scene{}, err
	}
	if fonts == nil {
		return Scene{}, invalid(KindQuadrant, 0, "fonts are required")
	}
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	const plotW = 38
	const labelGutter = 19
	labels := data.Labels
	defaults := [4]string{"Quadrant 1", "Quadrant 2", "Quadrant 3", "Quadrant 4"}
	quadrantLines := make([][]string, 4)
	maxQuadrantHeight := 0
	for i := range labels {
		if labels[i] == "" {
			labels[i] = defaults[i]
		}
		lines, _, lineHeight := labelSize(labels[i], fonts, plotW/2-2, fonts.BodyPx, false)
		quadrantLines[i] = lines
		maxQuadrantHeight = max(maxQuadrantHeight, len(lines)*lineHeight)
	}
	pointLines := make([][]string, len(data.Points))
	pointWidths := make([]int, len(data.Points))
	pointHeights := make([]int, len(data.Points))
	maxPointHeight := 0
	for i, point := range data.Points {
		lines, width, lineHeight := labelSize(point.Label, fonts, 12, fonts.BodyPx, false)
		pointLines[i], pointWidths[i] = lines, width
		pointHeights[i] = len(lines) * lineHeight
		maxPointHeight = max(maxPointHeight, pointHeights[i])
	}
	yFrom, _, yLineHeight := labelSize(data.YAxis.From, fonts, labelGutter-2, fonts.BodyPx, false)
	yTo, _, _ := labelSize(data.YAxis.To, fonts, labelGutter-2, fonts.BodyPx, false)
	xFrom, _, _ := labelSize(data.XAxis.From, fonts, plotW/2-1, fonts.BodyPx, false)
	xTo, _, _ := labelSize(data.XAxis.To, fonts, plotW/2-1, fonts.BodyPx, false)
	topPad := max(len(yTo)*yLineHeight, 1)
	bottomPad := max(len(yFrom)*yLineHeight, len(xFrom)*yLineHeight, len(xTo)*yLineHeight, 1)
	plotH := max(20, 2*maxQuadrantHeight+4, maxPointHeight+2)
	plotX, plotY := 1+labelGutter, titleHeight+topPad+1
	s := newScene()
	s.Width = max(titleWidth+2, plotX+plotW+2)
	s.Height = plotY + plotH + bottomPad + 2
	if titleHeight > 0 {
		s.TextLinesBold(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	halfW, halfH := plotW/2, plotH/2
	quadrants := []struct {
		x, y, w, h int
	}{
		{plotX + halfW, plotY, plotW - halfW, halfH},
		{plotX, plotY, halfW, halfH},
		{plotX, plotY + halfH, halfW, plotH - halfH},
		{plotX + halfW, plotY + halfH, plotW - halfW, plotH - halfH},
	}
	for i, quadrant := range quadrants {
		s.Rect(quadrant.x, quadrant.y, quadrant.w, quadrant.h, fade(accent(palette, i), 38), rule(palette))
		lines := quadrantLines[i]
		_, _, lineHeight := labelSize(labels[i], fonts, plotW/2-2, fonts.BodyPx, false)
		s.TextLines(lines, quadrant.x+1, quadrant.y+1, quadrant.w-2, lineHeight, fonts.BodyPx, AlignCenter, muted(palette), false)
	}
	s.Line(plotX+halfW, plotY, plotX+halfW, plotY+plotH-1, rule(palette), false, true)
	s.Line(plotX, plotY+halfH, plotX+plotW-1, plotY+halfH, rule(palette), false, true)
	s.TextLines(yTo, 1, titleHeight+1, labelGutter-2, yLineHeight, fonts.BodyPx, AlignRight, muted(palette), false)
	s.TextLines(yFrom, 1, plotY+plotH-bottomPad, labelGutter-2, yLineHeight, fonts.BodyPx, AlignRight, muted(palette), false)
	xLabelY := plotY + plotH + 1
	s.TextLines(xFrom, plotX, xLabelY, halfW-1, yLineHeight, fonts.BodyPx, AlignLeft, muted(palette), false)
	s.TextLines(xTo, plotX+halfW+1, xLabelY, plotW-halfW-1, yLineHeight, fonts.BodyPx, AlignRight, muted(palette), false)
	for i, point := range data.Points {
		px := plotX + int(math.Round(point.X*float64(plotW-1)))
		py := plotY + int(math.Round((1-point.Y)*float64(plotH-1)))
		s.Dot(px, py, 9, accent(palette, i), foreground(palette))
		labelX := px + 2
		if labelX+pointWidths[i] > plotX+plotW {
			labelX = px - pointWidths[i] - 2
		}
		labelX = min(max(labelX, plotX+1), plotX+plotW-pointWidths[i]-1)
		labelY := py - pointHeights[i]/2
		labelY = min(max(labelY, plotY+1), plotY+plotH-pointHeights[i]-1)
		_, _, lineHeight := labelSize(point.Label, fonts, 12, fonts.BodyPx, false)
		s.TextLines(pointLines[i], labelX, labelY, pointWidths[i], lineHeight, fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	return s, nil
}

func layoutTimeline(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, err := basicTimelineData(d)
	if err != nil {
		return Scene{}, err
	}
	if fonts == nil {
		return Scene{}, invalid(KindTimeline, 0, "fonts are required")
	}
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	type eventLayout struct {
		periodLines []string
		eventLines  []string
		periodW     int
		eventW      int
		periodH     int
		eventH      int
		lineHeight  int
	}
	type sectionLayout struct {
		title   string
		events  []eventLayout
		width   int
		headerH int
		aboveH  int
		belowH  int
		height  int
	}
	sections := make([]sectionLayout, len(data.Sections))
	sceneWidth := titleWidth + 2
	for i, section := range data.Sections {
		layout := sectionLayout{title: section.Title, events: make([]eventLayout, 0, len(section.Events))}
		columnWidth := 16
		for _, event := range section.Events {
			periodLines, periodW, lineHeight := labelSize(event.Period, fonts, 18, fonts.BodyPx, false)
			eventLines, eventW, _ := labelSize(event.Label, fonts, 22, fonts.BodyPx, false)
			periodH := max(len(periodLines)*lineHeight, 2)
			eventH := max(len(eventLines)*lineHeight, 1)
			layout.events = append(layout.events, eventLayout{
				periodLines: periodLines, eventLines: eventLines,
				periodW: periodW, eventW: eventW, periodH: periodH, eventH: eventH, lineHeight: lineHeight,
			})
			columnWidth = max(columnWidth, periodW+4, eventW+4)
			layout.aboveH = max(layout.aboveH, periodH+eventH+1)
			layout.belowH = max(layout.belowH, eventH+1+periodH)
		}
		layout.width = len(layout.events)*columnWidth + (len(layout.events)-1)*2
		if section.Title != "" {
			titleLines, _, lineHeight := labelSize(section.Title, fonts, max(layout.width-2, 3), fonts.BodyPx, false)
			layout.headerH = len(titleLines)*lineHeight + 1
		}
		layout.height = layout.headerH + layout.aboveH + 2 + layout.belowH + 2
		sections[i] = layout
		sceneWidth = max(sceneWidth, layout.width+2)
	}
	s := newScene()
	s.Width = sceneWidth
	s.Height = titleHeight + 2
	for _, section := range sections {
		s.Height += section.height
	}
	if titleHeight > 0 {
		s.TextLines(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	y := titleHeight + 1
	columnWidth := 16
	if len(sections) > 0 && len(sections[0].events) > 0 {
		// Every section uses the same column width so its timeline reads as one chart.
		columnWidth = (sections[0].width - (len(sections[0].events)-1)*2) / len(sections[0].events)
		for _, section := range sections[1:] {
			if len(section.events) > 0 {
				columnWidth = max(columnWidth, (section.width-(len(section.events)-1)*2)/len(section.events))
			}
		}
	}
	for i := range sections {
		sections[i].width = len(sections[i].events)*columnWidth + (len(sections[i].events)-1)*2
		sceneWidth = max(sceneWidth, sections[i].width+2)
	}
	s.Width = sceneWidth
	for sectionIndex, section := range sections {
		top := y
		col := accent(palette, sectionIndex)
		if section.headerH > 0 {
			s.Rect(1, y, section.width, section.headerH-1, fade(col, 48), color.RGBA{})
			headingLines, _, lineHeight := labelSize(section.title, fonts, max(section.width-2, 3), fonts.BodyPx, false)
			s.TextLinesBold(headingLines, 2, y, section.width-2, lineHeight, fonts.BodyPx, AlignLeft, foreground(palette), false)
		}
		baseline := top + section.headerH + section.aboveH + 1
		s.Line(1, baseline, 1+section.width, baseline, rule(palette), false, true)
		for eventIndex, event := range section.events {
			columnX := 1 + eventIndex*(columnWidth+2)
			above := eventIndex%2 == 0
			periodY := 0
			eventY := 0
			if above {
				periodY = baseline - 1 - event.periodH
				eventY = periodY - event.eventH - 1
				s.Line(columnX+1, baseline, columnX+1, periodY+event.periodH-1, col, false, false)
			} else {
				eventY = baseline + 1
				periodY = eventY + event.eventH + 1
				s.Line(columnX+1, baseline, columnX+1, periodY, col, false, false)
			}
			s.Dot(columnX+1, baseline, 6, col, foreground(palette))
			periodX := columnX + 2
			pillFill := fade(col, 60)
			if len(event.periodLines) == 1 {
				s.Badge(event.periodLines[0], periodX, periodY, max(event.periodW+2, 4), event.periodH, fonts.BodyPx, pillFill, contrastText(pillFill, palette))
			} else {
				s.Rounded(periodX, periodY, max(event.periodW+2, 4), event.periodH, 1, pillFill, col)
				s.TextLines(event.periodLines, periodX+1, periodY, max(event.periodW, 1), event.lineHeight, fonts.BodyPx, AlignCenter, contrastText(pillFill, palette), false)
			}
			s.TextLines(event.eventLines, periodX, eventY, max(event.eventW, 1), event.lineHeight, fonts.BodyPx, AlignCenter, foreground(palette), false)
		}
		y += section.height
	}
	return s, nil
}

func layoutJourney(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, err := basicJourneyData(d)
	if err != nil {
		return Scene{}, err
	}
	if fonts == nil {
		return Scene{}, invalid(KindJourney, 0, "fonts are required")
	}
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	actors := basicJourneyActors(data)
	type taskLayout struct {
		task    JourneyTask
		section int
		lines   []string
		width   int
		height  int
	}
	tasks := make([]taskLayout, 0)
	sectionCounts := make([]int, len(data.Sections))
	columnWidth := 14
	for sectionIndex, section := range data.Sections {
		for _, task := range section.Tasks {
			lines, width, lineHeight := labelSize(task.Label, fonts, 14, fonts.BodyPx, false)
			tasks = append(tasks, taskLayout{task: task, section: sectionIndex, lines: lines, width: width, height: len(lines) * lineHeight})
			columnWidth = max(columnWidth, width+2)
			sectionCounts[sectionIndex]++
		}
	}
	actorLines := make([][]string, len(actors))
	actorWidth, laneHeight := 12, 3
	for i, actor := range actors {
		lines, width, lineHeight := labelSize(actor, fonts, 12, fonts.BodyPx, false)
		actorLines[i] = lines
		actorWidth = max(actorWidth, width+2)
		laneHeight = max(laneHeight, len(lines)*lineHeight+1)
	}
	actorX := 1 + actorWidth
	gridWidth := actorWidth + len(tasks)*columnWidth
	sectionHeights := make([]int, len(data.Sections))
	sectionHeadingLines := make([][]string, len(data.Sections))
	sectionHeight := 0
	for i, section := range data.Sections {
		if section.Title == "" || sectionCounts[i] == 0 {
			continue
		}
		width := sectionCounts[i] * columnWidth
		lines, _, lineHeight := labelSize(section.Title, fonts, max(width-2, 3), fonts.BodyPx, false)
		sectionHeadingLines[i] = lines
		sectionHeights[i] = len(lines)*lineHeight + 1
		sectionHeight = max(sectionHeight, sectionHeights[i])
	}
	headerY := titleHeight + sectionHeight
	taskHeaderHeight := 2
	for _, task := range tasks {
		taskHeaderHeight = max(taskHeaderHeight, task.height+2)
	}
	gridY := headerY + taskHeaderHeight
	legendY := gridY + len(actors)*laneHeight + 1
	s := newScene()
	s.Width = max(titleWidth+2, actorX+len(tasks)*columnWidth+1, actorX+5*4+1)
	s.Height = legendY + 3
	if titleHeight > 0 {
		s.TextLines(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	taskOffset := 0
	for i, section := range data.Sections {
		if sectionHeights[i] > 0 {
			x := actorX + taskOffset*columnWidth
			width := sectionCounts[i] * columnWidth
			s.Rect(x, titleHeight, width, sectionHeights[i]-1, fade(accent(palette, i), 28), color.RGBA{})
			_, _, lineHeight := labelSize(section.Title, fonts, max(width-2, 3), fonts.BodyPx, false)
			s.TextLines(sectionHeadingLines[i], x+1, titleHeight, width-2, lineHeight, fonts.BodyPx, AlignCenter, foreground(palette), false)
		}
		taskOffset += sectionCounts[i]
	}
	s.Rect(1, headerY, gridWidth, taskHeaderHeight, fade(accent(palette, 0), 12), color.RGBA{})
	for taskIndex, task := range tasks {
		x := actorX + taskIndex*columnWidth
		_, _, lineHeight := labelSize(task.task.Label, fonts, 14, fonts.BodyPx, false)
		s.TextLines(task.lines, x+1, headerY+1, columnWidth-2, lineHeight, fonts.BodyPx, AlignCenter, foreground(palette), false)
	}
	for actorIndex, actor := range actors {
		y := gridY + actorIndex*laneHeight
		s.Rect(1, y, gridWidth, laneHeight, fade(accent(palette, actorIndex), 10), color.RGBA{})
		lines := actorLines[actorIndex]
		_, _, lineHeight := labelSize(actor, fonts, 12, fonts.BodyPx, false)
		textHeight := len(lines) * lineHeight
		s.TextLines(lines, 2, y+(laneHeight-textHeight)/2, actorWidth-2, lineHeight, fonts.BodyPx, AlignLeft, foreground(palette), false)
		for taskIndex, task := range tasks {
			cx := actorX + taskIndex*columnWidth + columnWidth/2
			cy := y + laneHeight/2
			participates := basicContainsActor(task.task.Actors, actor)
			if participates {
				fill := accent(palette, task.task.Score-1)
				s.Circle(cx-2, cy-1, 4, 2, fill, rule(palette))
				s.TextBold(strconv.Itoa(task.task.Score), cx-2, cy-1, 4, 2, fonts.BodyPx, AlignCenter, contrastText(fill, palette), false)
			} else {
				s.Dot(cx, cy, 3, muted(palette), muted(palette))
			}
		}
	}
	for x := actorX - 1; x < actorX+len(tasks)*columnWidth; x += columnWidth {
		s.Line(x, headerY, x, gridY+len(actors)*laneHeight-1, rule(palette), false, false)
	}
	for i := 1; i < len(actors); i++ {
		y := gridY + i*laneHeight
		s.Line(1, y, actorX+len(tasks)*columnWidth-1, y, rule(palette), false, false)
	}
	s.Text("Score", 1, legendY, actorWidth-1, 2, fonts.BodyPx, AlignLeft, muted(palette), false)
	for score := 1; score <= 5; score++ {
		x := actorX + (score-1)*4
		fill := accent(palette, score-1)
		s.Badge(strconv.Itoa(score), x, legendY, 3, 2, fonts.BodyPx, fill, contrastText(fill, palette))
	}
	return s, nil
}

func basicPieData(d Diagram) (PieData, error) {
	data, ok := d.Data.(PieData)
	if !ok || len(data.Slices) == 0 {
		return PieData{}, invalid(KindPie, 0, "invalid pie payload")
	}
	total := 0.0
	for _, slice := range data.Slices {
		if strings.TrimSpace(slice.Label) == "" || !basicFinite(slice.Value) || slice.Value < 0 {
			return PieData{}, invalid(KindPie, 0, "invalid pie slice")
		}
		total += slice.Value
	}
	if !basicFinite(total) || total <= 0 {
		return PieData{}, invalid(KindPie, 0, "pie values must have a positive finite total")
	}
	return data, nil
}

func basicQuadrantData(d Diagram) (QuadrantData, error) {
	data, ok := d.Data.(QuadrantData)
	if !ok || strings.TrimSpace(data.XAxis.From) == "" || strings.TrimSpace(data.XAxis.To) == "" || strings.TrimSpace(data.YAxis.From) == "" || strings.TrimSpace(data.YAxis.To) == "" {
		return QuadrantData{}, invalid(KindQuadrant, 0, "invalid quadrant payload")
	}
	for _, point := range data.Points {
		if strings.TrimSpace(point.Label) == "" || !basicFinite(point.X) || !basicFinite(point.Y) || point.X < 0 || point.X > 1 || point.Y < 0 || point.Y > 1 {
			return QuadrantData{}, invalid(KindQuadrant, 0, "invalid quadrant point")
		}
	}
	return data, nil
}

func basicTimelineData(d Diagram) (TimelineData, error) {
	data, ok := d.Data.(TimelineData)
	if !ok || len(data.Sections) == 0 {
		return TimelineData{}, invalid(KindTimeline, 0, "invalid timeline payload")
	}
	for _, section := range data.Sections {
		if len(section.Events) == 0 {
			return TimelineData{}, invalid(KindTimeline, 0, "timeline section has no events")
		}
		for _, event := range section.Events {
			if strings.TrimSpace(event.Period) == "" || strings.TrimSpace(event.Label) == "" {
				return TimelineData{}, invalid(KindTimeline, 0, "invalid timeline event")
			}
		}
	}
	return data, nil
}

func basicJourneyData(d Diagram) (JourneyData, error) {
	data, ok := d.Data.(JourneyData)
	if !ok || len(data.Sections) == 0 {
		return JourneyData{}, invalid(KindJourney, 0, "invalid journey payload")
	}
	for _, section := range data.Sections {
		if len(section.Tasks) == 0 {
			return JourneyData{}, invalid(KindJourney, 0, "journey section has no tasks")
		}
		for _, task := range section.Tasks {
			if strings.TrimSpace(task.Label) == "" || task.Score < 1 || task.Score > 5 || len(task.Actors) == 0 {
				return JourneyData{}, invalid(KindJourney, 0, "invalid journey task")
			}
			seen := make(map[string]struct{}, len(task.Actors))
			for _, actor := range task.Actors {
				if strings.TrimSpace(actor) == "" {
					return JourneyData{}, invalid(KindJourney, 0, "invalid journey actor")
				}
				if _, exists := seen[actor]; exists {
					return JourneyData{}, invalid(KindJourney, 0, "duplicate journey actor")
				}
				seen[actor] = struct{}{}
			}
		}
	}
	return data, nil
}

func basicAxis(value string) (QuadrantAxis, bool) {
	parts := strings.SplitN(strings.TrimSpace(value), "-->", 2)
	if len(parts) != 2 {
		return QuadrantAxis{}, false
	}
	from, to := unquote(parts[0]), unquote(parts[1])
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		return QuadrantAxis{}, false
	}
	return QuadrantAxis{From: from, To: to}, true
}

func basicQuoted(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return false
	}
	quote := value[0]
	if (quote != '"' && quote != '\'') || value[len(value)-1] != quote {
		return false
	}
	escaped := false
	for i := 1; i < len(value)-1; i++ {
		if escaped {
			escaped = false
			continue
		}
		if value[i] == '\\' {
			escaped = true
			continue
		}
		if value[i] == quote {
			return false
		}
	}
	return true
}

func basicFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func basicTitle(title string, fonts *rich.Fonts) ([]string, int, int) {
	if title == "" {
		return nil, 0, 0
	}
	lines, width, lineHeight := labelSize(title, fonts, 60, fonts.BodyPx, false)
	return lines, width, len(lines) * lineHeight
}

func basicJourneyActors(data JourneyData) []string {
	var actors []string
	seen := make(map[string]struct{})
	for _, section := range data.Sections {
		for _, task := range section.Tasks {
			for _, actor := range task.Actors {
				if _, exists := seen[actor]; exists {
					continue
				}
				seen[actor] = struct{}{}
				actors = append(actors, actor)
			}
		}
	}
	return actors
}

func basicContainsActor(actors []string, want string) bool {
	for _, actor := range actors {
		if actor == want {
			return true
		}
	}
	return false
}
