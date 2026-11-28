package diagram

import (
	"reflect"
	"testing"
)

const ganttConstructSource = `gantt
title Release plan
dateFormat YYYY-MM-DD
axisFormat %b %d
todayMarker off
section Build
Design: done, design, 2026-01-01, 3d
Implement: active, impl, after design, 5d
section Ship
Milestone: milestone, m1, 2026-01-10, 0d`

func TestGanttDiagramIR(t *testing.T) {
	d, err := Parse(ganttConstructSource)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindGantt || d.Title != "Release plan" {
		t.Fatalf("diagram = %#v", d)
	}
	data := d.Data.(GanttData)
	if data.DateFormat != "YYYY-MM-DD" || data.AxisFormat != "%b %d" || data.TodayMarker != "off" {
		t.Fatalf("formats = %#v", data)
	}
	if len(data.Sections) != 2 || len(data.Sections[0].Tasks) != 2 || len(data.Sections[1].Tasks) != 1 {
		t.Fatalf("sections = %#v", data.Sections)
	}
	first := data.Sections[0].Tasks[0]
	if first.ID != "design" || !first.Done || first.Start.Format("2006-01-02") != "2026-01-01" || first.Days() != 3 {
		t.Fatalf("first task = %#v", first)
	}
	second := data.Sections[0].Tasks[1]
	if second.ID != "impl" || !second.Active || second.Start.Format("2006-01-02") != "2026-01-04" || second.Days() != 5 {
		t.Fatalf("second task = %#v", second)
	}
	milestone := data.Sections[1].Tasks[0]
	if !milestone.Milestone || milestone.Days() != 0 {
		t.Fatalf("milestone = %#v", milestone)
	}
}

func TestGanttConstructFamily(t *testing.T) {
	source := `gantt
dateFormat YYYY-MM-DD
section Work
Critical: crit, c1, 2026-02-01, 2d
Later: l1, after c1, 1d
`
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	tasks := d.Data.(GanttData).Sections[0].Tasks
	if len(tasks) != 2 || !tasks[0].Critical || tasks[0].End != tasks[0].Start.AddDate(0, 0, 2) {
		t.Fatalf("tasks = %#v", tasks)
	}
	if tasks[1].Start != tasks[0].End {
		t.Fatalf("after task start = %v, want %v", tasks[1].Start, tasks[0].End)
	}
}

func TestGanttLayout(t *testing.T) {
	d, err := Parse(ganttConstructSource)
	if err != nil {
		t.Fatal(err)
	}
	fonts := testFonts(t)
	scene, err := Layout(d, fonts, testPalette())
	if err != nil {
		t.Fatal(err)
	}
	if scene.Width <= 0 || scene.Height <= 0 || len(scene.Commands) == 0 {
		t.Fatalf("scene = %#v", scene)
	}
	if scene.Width > maxSceneCells || scene.Height > maxSceneCells {
		t.Fatalf("scene size = %dx%d", scene.Width, scene.Height)
	}
}

func TestGanttUnknownConstruct(t *testing.T) {
	for _, source := range []string{
		"gantt\ndateFormat YYYY-MM-DD\nBad: id, 2026-01-01, sometime",
		"gantt\nsection Work\nNo duration: id, 2026-01-01",
	} {
		if _, err := Parse(source); err == nil {
			t.Fatalf("Parse accepted %q", source)
		}
	}
}

func TestGanttConstructGolden(t *testing.T) {
	assertGolden(t, "gantt/constructs", renderSource(t, ganttConstructSource))
}

const xyConstructSource = `xychart-beta
title Revenue
x-axis [Jan, Feb, Mar, Apr]
y-axis "USD" 0 --> 100
bar "Actual" [20, 40, 30, 60]
line "Target" [25, 35, 45, 70]`

func TestXYDiagramIR(t *testing.T) {
	d, err := Parse(xyConstructSource)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindXY || d.Title != "Revenue" {
		t.Fatalf("diagram = %#v", d)
	}
	data := d.Data.(XYData)
	if !reflect.DeepEqual(data.Categories, []string{"Jan", "Feb", "Mar", "Apr"}) {
		t.Fatalf("categories = %#v", data.Categories)
	}
	if data.YAxis.Label != "USD" || data.YAxis.Min != 0 || data.YAxis.Max != 100 {
		t.Fatalf("y axis = %#v", data.YAxis)
	}
	if len(data.Series) != 2 || data.Series[0].Kind != XYBar || data.Series[0].Name != "Actual" || len(data.Series[0].Values) != 4 {
		t.Fatalf("series = %#v", data.Series)
	}
}

func TestXYLayout(t *testing.T) {
	d, err := Parse(xyConstructSource)
	if err != nil {
		t.Fatal(err)
	}
	scene, err := Layout(d, testFonts(t), testPalette())
	if err != nil {
		t.Fatal(err)
	}
	if scene.Width <= 0 || scene.Height <= 0 || len(scene.Commands) == 0 {
		t.Fatalf("scene = %#v", scene)
	}
}

func TestXYUnknownConstruct(t *testing.T) {
	for _, source := range []string{
		"xychart-beta\nbar [1, 2, 3]\narea [1, 2, 3]",
		"xychart-beta\nfoo-axis 0 --> 10",
	} {
		if _, err := Parse(source); err == nil {
			t.Fatalf("Parse accepted %q", source)
		}
	}
}

func TestXYConstructGolden(t *testing.T) {
	assertGolden(t, "xychart/constructs", renderSource(t, xyConstructSource))
}
