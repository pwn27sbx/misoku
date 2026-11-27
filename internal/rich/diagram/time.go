package diagram

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// GanttData is the typed IR for a gantt diagram.
type GanttData struct {
	DateFormat  string
	AxisFormat  string
	TodayMarker string
	Sections    []GanttSection
}

// GanttSection is one gantt section and its tasks.
type GanttSection struct {
	Title string
	Tasks []GanttTask
}

// GanttTask is one scheduled gantt row.
type GanttTask struct {
	ID           string
	Label        string
	Start        time.Time
	End          time.Time
	Done         bool
	Active       bool
	Critical     bool
	Milestone    bool
	Section      string
	Duration     float64
	After        string
	DateExplicit bool
}

// Days returns the rounded duration of the task in days.
func (t GanttTask) Days() int {
	return int(math.Round(t.Duration))
}

// XYData is the typed IR for an xychart-beta diagram.
type XYData struct {
	Categories []string
	XAxis      XYAxis
	YAxis      XYAxis
	Series     []XYSeries
}

// XYAxis is one axis label and numeric range.
type XYAxis struct {
	Label string
	Min   float64
	Max   float64
}

// XYSeriesKind is the visual form of one xychart series.
type XYSeriesKind int

const (
	// XYBar draws vertical bars.
	XYBar XYSeriesKind = iota
	// XYLine draws a connected line.
	XYLine
)

// XYSeries is one named bar or line series.
type XYSeries struct {
	Name   string
	Kind   XYSeriesKind
	Values []float64
}

var durationPattern = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)([dhw])$`)

func init() {
	register(KindGantt, parseGantt, layoutGantt)
	register(KindXY, parseXY, layoutXY)
}

func parseGantt(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindGantt}
	data := GanttData{DateFormat: "YYYY-MM-DD"}
	section := ""
	byID := make(map[string]GanttTask)
	var previousEnd time.Time
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ensureSection := func(title string) int {
		for i := range data.Sections {
			if data.Sections[i].Title == title {
				return i
			}
		}
		data.Sections = append(data.Sections, GanttSection{Title: title})
		return len(data.Sections) - 1
	}
	for _, line := range numberedNonBlank(lines) {
		text := line.Text
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		key := fields[0]
		switch key {
		case "title":
			if d.Title != "" {
				return Diagram{}, unknown(KindGantt, line.Number, text)
			}
			d.Title = unquote(strings.TrimSpace(text[len("title"):]))
		case "dateFormat":
			data.DateFormat = strings.TrimSpace(text[len("dateFormat"):])
			if data.DateFormat == "" {
				return Diagram{}, unknown(KindGantt, line.Number, text)
			}
		case "axisFormat":
			data.AxisFormat = strings.TrimSpace(text[len("axisFormat"):])
		case "todayMarker":
			data.TodayMarker = strings.TrimSpace(text[len("todayMarker"):])
		case "excludes":
			if strings.TrimSpace(text[len("excludes"):]) == "" {
				return Diagram{}, unknown(KindGantt, line.Number, text)
			}
		case "section":
			section = unquote(strings.TrimSpace(text[len("section"):]))
			if section == "" {
				return Diagram{}, unknown(KindGantt, line.Number, text)
			}
			ensureSection(section)
		default:
			if !strings.Contains(text, ":") {
				return Diagram{}, unknown(KindGantt, line.Number, text)
			}
			task, err := parseGanttTask(text, data.DateFormat, line.Number, byID, previousEnd, base)
			if err != nil {
				return Diagram{}, err
			}
			task.Section = section
			if task.Start.IsZero() {
				task.Start = base
			}
			if task.Duration == 0 {
				task.End = task.Start
			} else {
				task.End = task.Start.Add(time.Duration(task.Duration*24) * time.Hour)
			}
			if task.Duration < 0 {
				return Diagram{}, unknown(KindGantt, line.Number, text)
			}
			if task.ID != "" {
				byID[task.ID] = task
			}
			previousEnd = task.End
			index := ensureSection(section)
			data.Sections[index].Tasks = append(data.Sections[index].Tasks, task)
		}
	}
	if len(data.Sections) == 0 {
		return Diagram{}, unknown(KindGantt, 0, "empty gantt")
	}
	d.Data = data
	return d, nil
}

func parseGanttTask(text, dateFormat string, line int, byID map[string]GanttTask, previousEnd, base time.Time) (GanttTask, error) {
	parts := splitTopLevelString(text, ':')
	if len(parts) != 2 {
		return GanttTask{}, unknown(KindGantt, line, text)
	}
	task := GanttTask{Label: unquote(strings.TrimSpace(parts[0]))}
	body := strings.TrimSpace(parts[1])
	if task.Label == "" || body == "" {
		return GanttTask{}, unknown(KindGantt, line, text)
	}
	fields := splitTopLevelString(body, ',')
	var durationToken string
	for _, raw := range fields {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		lower := strings.ToLower(token)
		switch lower {
		case "done":
			task.Done = true
			continue
		case "active":
			task.Active = true
			continue
		case "crit":
			task.Critical = true
			continue
		case "milestone":
			task.Milestone = true
			continue
		}
		if strings.HasPrefix(lower, "after ") {
			task.After = strings.TrimSpace(token[len("after "):])
			continue
		}
		if t, ok := parseGanttDate(token, dateFormat); ok {
			task.Start = t
			task.DateExplicit = true
			continue
		}
		if durationPattern.MatchString(lower) {
			durationToken = lower
			continue
		}
		if task.ID == "" && !strings.ContainsAny(token, " \t") {
			task.ID = token
			continue
		}
		return GanttTask{}, unknown(KindGantt, line, text)
	}
	if durationToken == "" {
		return GanttTask{}, unknown(KindGantt, line, text)
	}
	value, unit, _ := func() (float64, string, bool) {
		m := durationPattern.FindStringSubmatch(durationToken)
		if len(m) != 3 {
			return 0, "", false
		}
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, "", false
		}
		return v, m[2], true
	}()
	switch unit {
	case "d":
		task.Duration = value
	case "h":
		task.Duration = value / 24
	case "w":
		task.Duration = value * 7
	default:
		return GanttTask{}, unknown(KindGantt, line, text)
	}
	if task.Milestone {
		task.Duration = 0
	}
	if task.Start.IsZero() {
		if task.After != "" {
			prior, ok := byID[task.After]
			if !ok {
				return GanttTask{}, unknown(KindGantt, line, text)
			}
			task.Start = prior.End
		} else if !previousEnd.IsZero() {
			task.Start = previousEnd
		} else {
			task.Start = base
		}
	}
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", line)
	}
	return task, nil
}

func parseGanttDate(text, format string) (time.Time, bool) {
	layout := strings.NewReplacer(
		"YYYY", "2006", "YY", "06",
		"MM", "01", "DD", "02",
		"HH", "15", "mm", "04",
	).Replace(format)
	if layout == "" {
		layout = "2006-01-02"
	}
	for _, candidate := range []string{text, strings.TrimSpace(text)} {
		if t, err := time.ParseInLocation(layout, candidate, time.UTC); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func layoutGantt(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, ok := d.Data.(GanttData)
	if !ok {
		return Scene{}, invalid(KindGantt, 0, "missing gantt data")
	}
	minDate, maxDate := time.Time{}, time.Time{}
	taskCount := 0
	for _, section := range data.Sections {
		for _, task := range section.Tasks {
			taskCount++
			if minDate.IsZero() || task.Start.Before(minDate) {
				minDate = task.Start
			}
			if maxDate.IsZero() || task.End.After(maxDate) {
				maxDate = task.End
			}
		}
	}
	if minDate.IsZero() {
		minDate = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if maxDate.Before(minDate.AddDate(0, 0, 1)) {
		maxDate = minDate.AddDate(0, 0, 1)
	}
	s := newScene()
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	if titleHeight == 0 {
		titleHeight = 1
	}
	gutter := 22
	plotW := 60
	s.Width = max(titleWidth+2, gutter+plotW+5)
	y := titleHeight + 1
	if d.Title != "" {
		s.TextLines(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	axisY := y
	s.Line(gutter, axisY, gutter+plotW, axisY, rule(palette), false, false)
	s.Line(gutter, axisY, gutter, axisY+max(taskCount*2+len(data.Sections)*2, 4), rule(palette), false, false)
	span := maxDate.Sub(minDate).Hours()
	if span <= 0 {
		span = 24
	}
	dateWidth := func(t time.Time) int {
		ratio := t.Sub(minDate).Hours() / span
		return int(math.Round(ratio * float64(plotW)))
	}
	axisLabel := func(t time.Time) string {
		if data.AxisFormat != "" {
			return ganttAxisLabel(t, data.AxisFormat)
		}
		return t.Format("Jan 02")
	}
	s.Text(axisLabel(minDate), gutter, axisY-1, 8, 1, fonts.BodyPx, AlignLeft, muted(palette), false)
	s.Text(axisLabel(maxDate), gutter+plotW-8, axisY-1, 8, 1, fonts.BodyPx, AlignRight, muted(palette), false)
	y = axisY + 1
	for sectionIndex, section := range data.Sections {
		s.Text(section.Title, 2, y, gutter-2, 1, fonts.BodyPx, AlignLeft, accent(palette, sectionIndex), false)
		s.Line(1, y+1, s.Width-2, y+1, fade(accent(palette, sectionIndex), 120), false, false)
		y++
		for _, task := range section.Tasks {
			if y+2 >= maxSceneCells {
				return Scene{}, invalid(KindGantt, 0, "gantt layout too tall")
			}
			s.Text(task.Label, 1, y, gutter-2, 1, fonts.BodyPx, AlignRight, foreground(palette), false)
			startX := gutter + dateWidth(task.Start)
			endX := gutter + dateWidth(task.End)
			if endX < startX {
				endX = startX
			}
			if task.Milestone {
				milestoneX := min(startX, gutter+plotW-3)
				s.Diamond(milestoneX, y, 3, 3, accent(palette, 3), rule(palette))
			} else {
				barColor := accent(palette, 0)
				if task.Done {
					barColor = accent(palette, 2)
				} else if task.Critical {
					barColor = accent(palette, 5)
				} else if task.Active {
					barColor = accent(palette, 1)
				}
				s.Rounded(startX, y, max(endX-startX, 2), 2, 1, fade(barColor, 210), barColor)
				if endX-startX >= len([]rune(task.Label))+2 {
					s.Text(task.Label, startX, y, endX-startX, 1, fonts.BodyPx, AlignCenter, foreground(palette), false)
				}
			}
			y += 2
		}
	}
	if data.TodayMarker != "off" && data.TodayMarker != "" {
		today := minDate.AddDate(0, 0, 3)
		s.Line(gutter+dateWidth(today), axisY, gutter+dateWidth(today), y, accent(palette, 4), true, false)
	}
	s.Height = y + 1
	return s, nil
}

func ganttAxisLabel(t time.Time, format string) string {
	// Mermaid accepts strftime-like axis formats; cover the common tokens
	// without pretending to implement strftime in full.
	text := format
	for _, replacement := range []struct{ old, new string }{
		{"%b", t.Format("Jan")},
		{"%B", t.Format("January")},
		{"%d", t.Format("02")},
		{"%m", t.Format("01")},
		{"%Y", t.Format("2006")},
	} {
		text = strings.ReplaceAll(text, replacement.old, replacement.new)
	}
	return strings.TrimSpace(text)
}

func parseXY(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindXY}
	data := XYData{}
	for _, line := range numberedNonBlank(lines) {
		text := line.Text
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "title":
			if d.Title != "" {
				return Diagram{}, unknown(KindXY, line.Number, text)
			}
			d.Title = unquote(strings.TrimSpace(text[len("title"):]))
		case "x-axis":
			axis, categories, err := parseXYAxis(strings.TrimSpace(text[len("x-axis"):]))
			if err != nil {
				return Diagram{}, unknown(KindXY, line.Number, text)
			}
			data.XAxis = axis
			if len(categories) > 0 {
				data.Categories = categories
			}
		case "y-axis":
			axis, _, err := parseXYAxis(strings.TrimSpace(text[len("y-axis"):]))
			if err != nil {
				return Diagram{}, unknown(KindXY, line.Number, text)
			}
			data.YAxis = axis
		case "bar", "line":
			series, err := parseXYSeries(fields[0], strings.TrimSpace(text[len(fields[0]):]))
			if err != nil {
				return Diagram{}, unknown(KindXY, line.Number, text)
			}
			data.Series = append(data.Series, series)
		default:
			return Diagram{}, unknown(KindXY, line.Number, text)
		}
	}
	if len(data.Series) == 0 {
		return Diagram{}, unknown(KindXY, 0, "empty xychart")
	}
	if len(data.Categories) == 0 {
		n := 0
		for _, series := range data.Series {
			n = max(n, len(series.Values))
		}
		for i := 0; i < n; i++ {
			data.Categories = append(data.Categories, strconv.Itoa(i+1))
		}
	}
	minValue, maxValue := 0.0, 0.0
	for _, series := range data.Series {
		for _, value := range series.Values {
			if value < minValue {
				minValue = value
			}
			if value > maxValue {
				maxValue = value
			}
		}
	}
	if data.YAxis.Min == 0 && data.YAxis.Max == 0 {
		data.YAxis.Min, data.YAxis.Max = minValue, maxValue
	}
	if data.YAxis.Max == data.YAxis.Min {
		data.YAxis.Max = data.YAxis.Min + 1
	}
	if data.YAxis.Max < data.YAxis.Min {
		return Diagram{}, invalid(KindXY, 0, "y-axis range is inverted")
	}
	d.Data = data
	return d, nil
}

func parseXYAxis(text string) (XYAxis, []string, error) {
	text = strings.TrimSpace(text)
	var label string
	if strings.HasPrefix(text, "\"") || strings.HasPrefix(text, "'") {
		token, rest, ok := flowReadQuoted(text)
		if !ok {
			return XYAxis{}, nil, fmt.Errorf("unterminated axis label")
		}
		label, text = unquote(token), strings.TrimSpace(rest)
	}
	if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
		parts := splitTopLevelString(text[1:len(text)-1], ',')
		categories := make([]string, 0, len(parts))
		for _, part := range parts {
			if strings.TrimSpace(part) == "" {
				return XYAxis{}, nil, fmt.Errorf("empty category")
			}
			categories = append(categories, unquote(part))
		}
		return XYAxis{Label: label}, categories, nil
	}
	parts := strings.SplitN(text, "-->", 2)
	if len(parts) != 2 {
		return XYAxis{}, nil, fmt.Errorf("invalid axis")
	}
	minValue, err := parseFloat(parts[0])
	if err != nil {
		return XYAxis{}, nil, err
	}
	maxValue, err := parseFloat(parts[1])
	if err != nil {
		return XYAxis{}, nil, err
	}
	return XYAxis{Label: label, Min: minValue, Max: maxValue}, nil, nil
}

func parseXYSeries(kind string, text string) (XYSeries, error) {
	name := ""
	if strings.HasPrefix(text, "\"") || strings.HasPrefix(text, "'") {
		token, rest, ok := flowReadQuoted(text)
		if !ok {
			return XYSeries{}, fmt.Errorf("unterminated series name")
		}
		name, text = unquote(token), strings.TrimSpace(rest)
	}
	if !strings.HasPrefix(text, "[") || !strings.HasSuffix(text, "]") {
		return XYSeries{}, fmt.Errorf("series values must be bracketed")
	}
	parts := splitTopLevelString(text[1:len(text)-1], ',')
	values := make([]float64, 0, len(parts))
	for _, part := range parts {
		value, err := parseFloat(part)
		if err != nil {
			return XYSeries{}, err
		}
		values = append(values, value)
	}
	series := XYSeries{Name: name, Kind: XYBar, Values: values}
	if kind == "line" {
		series.Kind = XYLine
	}
	return series, nil
}

func layoutXY(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, ok := d.Data.(XYData)
	if !ok {
		return Scene{}, invalid(KindXY, 0, "missing xychart data")
	}
	s := newScene()
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	if titleHeight == 0 {
		titleHeight = 1
	}
	labelFor := func(value float64) string {
		if math.Abs(value-math.Round(value)) < 0.0001 {
			return strconv.FormatInt(int64(math.Round(value)), 10)
		}
		return fmt.Sprintf("%.1f", value)
	}
	left := 9
	plotW := max(len(data.Categories)*5, 20)
	plotH := 14
	s.Width = max(titleWidth+2, left+plotW+4)
	s.Height = titleHeight + plotH + 4
	if d.Title != "" {
		s.TextLines(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	top := titleHeight + 1
	s.Line(left, top, left, top+plotH, rule(palette), false, false)
	s.Line(left, top+plotH, left+plotW, top+plotH, rule(palette), false, false)
	for i := 0; i <= 4; i++ {
		y := top + i*plotH/4
		s.Line(left, y, left+plotW, y, fade(rule(palette), 70), false, false)
		value := data.YAxis.Max - (data.YAxis.Max-data.YAxis.Min)*float64(i)/4
		s.Text(labelFor(value), 1, y, left-2, 1, fonts.BodyPx, AlignRight, muted(palette), false)
	}
	if data.YAxis.Label != "" {
		s.Text(data.YAxis.Label, 1, top-1, left-2, 1, fonts.BodyPx, AlignRight, foreground(palette), false)
	}
	for i, category := range data.Categories {
		x := left + i*plotW/max(len(data.Categories), 1) + plotW/max(len(data.Categories), 1)/2
		s.Text(category, x-2, top+plotH+1, 5, 1, fonts.BodyPx, AlignCenter, muted(palette), false)
	}
	rangeY := data.YAxis.Max - data.YAxis.Min
	toY := func(value float64) int {
		if rangeY == 0 {
			return top + plotH
		}
		return top + plotH - int(math.Round((value-data.YAxis.Min)/rangeY*float64(plotH)))
	}
	barCount := 0
	for _, series := range data.Series {
		if series.Kind == XYBar {
			barCount++
		}
	}
	barIndex := 0
	for seriesIndex, series := range data.Series {
		color := accent(palette, seriesIndex)
		if series.Kind == XYBar {
			seriesWidth := max(plotW/max(len(data.Categories), 1)/max(barCount, 1), 2)
			for i, value := range series.Values {
				if i >= len(data.Categories) {
					break
				}
				x := left + i*plotW/max(len(data.Categories), 1) + barIndex*seriesWidth
				y := toY(value)
				s.Rect(x, y, max(seriesWidth-1, 1), top+plotH-y, fade(color, 220), color)
			}
			barIndex++
		} else {
			points := make([]Point, 0, len(series.Values))
			for i, value := range series.Values {
				if i >= len(data.Categories) {
					break
				}
				x := left + i*plotW/max(len(data.Categories), 1) + plotW/max(len(data.Categories), 1)/2
				points = append(points, Point{X: x, Y: toY(value)})
			}
			s.Polyline(points, color, false, true)
			for _, point := range points {
				s.Ellipse(point.X, point.Y, 1, 1, color, color)
			}
		}
	}
	return s, nil
}
