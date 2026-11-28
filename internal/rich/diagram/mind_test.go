package diagram

import "testing"

const mindmapConstructSource = `mindmap
root((Product))
  Research[Research]
    Interviews
    Metrics{{Signals}}
  Build(Runtime)
    Editor
  Launch
`

func TestMindmapDiagramIR(t *testing.T) {
	d, err := Parse(mindmapConstructSource)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindMindmap {
		t.Fatalf("kind = %q", d.Kind)
	}
	data := d.Data.(MindmapData)
	root := data.Root
	if root == nil || root.Text != "Product" || root.Shape != ShapeCircle {
		t.Fatalf("root = %#v", root)
	}
	if len(root.Children) != 3 {
		t.Fatalf("root children = %#v", root.Children)
	}
	if got := root.Children[0]; got.Text != "Research" || got.Shape != ShapeRect || len(got.Children) != 2 {
		t.Fatalf("research = %#v", got)
	}
	if got := root.Children[0].Children[1]; got.Text != "Signals" || got.Shape != ShapeHexagon {
		t.Fatalf("signals = %#v", got)
	}
	if got := root.Children[1]; got.Text != "Runtime" || got.Shape != ShapeRounded {
		t.Fatalf("runtime = %#v", got)
	}
}

func TestMindmapLayout(t *testing.T) {
	d, err := Parse(mindmapConstructSource)
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

func TestMindmapUnknownConstruct(t *testing.T) {
	for _, source := range []string{
		"mindmap\n",
		"mindmap\nroot((Root))\n  Broken[",
		"mindmap\n  Orphan",
	} {
		if _, err := Parse(source); err == nil {
			t.Fatalf("Parse accepted %q", source)
		}
	}
}

func TestMindmapConstructGolden(t *testing.T) {
	assertGolden(t, "mindmap/constructs", renderSource(t, mindmapConstructSource))
}

const gitConstructSource = `gitGraph
    title Release flow
    commit id: "a1" msg: "init" tag: "v0.1"
    branch dev
    commit id: "b1" msg: "feature"
    checkout main
    commit id: "a2" msg: "fix"
    merge dev id: "m1" tag: "v0.2" msg: "merge dev"
`

func TestGitDiagramIR(t *testing.T) {
	d, err := Parse(gitConstructSource)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindGit || d.Title != "Release flow" {
		t.Fatalf("diagram = %#v", d)
	}
	data := d.Data.(GitData)
	if len(data.Branches) != 2 || data.Branches[0].Name != "main" || data.Branches[1].Name != "dev" {
		t.Fatalf("branches = %#v", data.Branches)
	}
	if len(data.Commits) != 4 {
		t.Fatalf("commits = %#v", data.Commits)
	}
	first := data.Commits[0]
	if first.ID != "a1" || first.Message != "init" || first.Tag != "v0.1" || first.Branch != "main" {
		t.Fatalf("first = %#v", first)
	}
	merge := data.Commits[3]
	if merge.ID != "m1" || len(merge.Parents) != 2 || merge.Tag != "v0.2" {
		t.Fatalf("merge = %#v", merge)
	}
}

func TestGitLayout(t *testing.T) {
	d, err := Parse(gitConstructSource)
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

func TestGitUnknownConstruct(t *testing.T) {
	for _, source := range []string{
		"gitGraph\ncherry-pick id: \"a1\"",
		"gitGraph\ncheckout missing",
		"gitGraph\ncommit id:",
	} {
		if _, err := Parse(source); err == nil {
			t.Fatalf("Parse accepted %q", source)
		}
	}
}

func TestGitConstructGolden(t *testing.T) {
	assertGolden(t, "gitgraph/constructs", renderSource(t, gitConstructSource))
}
