package diagram

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// MindmapData is the typed IR for a mindmap.
type MindmapData struct {
	Root *MindNode
}

// MindNode is one mindmap branch.
type MindNode struct {
	Text     string
	Shape    NodeShape
	Children []*MindNode
}

// GitData is the typed IR for a git graph.
type GitData struct {
	Branches []GitBranch
	Commits  []GitCommit
}

// GitBranch is one gitGraph lane.
type GitBranch struct {
	Name  string
	Start int
	Head  int
}

// GitCommit is one commit or merge in a git graph.
type GitCommit struct {
	ID      string
	Message string
	Tag     string
	Branch  string
	Parents []string
	Merge   bool
}

var gitOptionPattern = regexp.MustCompile(`([A-Za-z][A-Za-z0-9_-]*)\s*:\s*(?:"((?:\\.|[^"])*)"|'((?:\\.|[^'])*)'|([^\s]+))`)

func init() {
	register(KindMindmap, parseMindmap, layoutMindmap)
	register(KindGit, parseGit, layoutGit)
}

func parseMindmap(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindMindmap}
	var root *MindNode
	var stack []*MindNode
	var stackIndent []int
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		indent := mindIndent(raw)
		node, err := parseMindNode(trimmed)
		if err != nil {
			return Diagram{}, err
		}
		if root == nil {
			if indent != 0 {
				return Diagram{}, unknown(KindMindmap, 0, raw)
			}
			root = node
			stack = []*MindNode{root}
			stackIndent = []int{indent}
			continue
		}
		if indent == 0 {
			return Diagram{}, unknown(KindMindmap, 0, raw)
		}
		for len(stack) > 1 && indent <= stackIndent[len(stackIndent)-1] {
			stack = stack[:len(stack)-1]
			stackIndent = stackIndent[:len(stackIndent)-1]
		}
		if indent <= stackIndent[len(stackIndent)-1] {
			return Diagram{}, unknown(KindMindmap, 0, raw)
		}
		parent := stack[len(stack)-1]
		parent.Children = append(parent.Children, node)
		stack = append(stack, node)
		stackIndent = append(stackIndent, indent)
	}
	if root == nil {
		return Diagram{}, unknown(KindMindmap, 0, "empty mindmap")
	}
	d.Data = MindmapData{Root: root}
	return d, nil
}

func mindIndent(raw string) int {
	count := 0
	for _, r := range raw {
		switch r {
		case ' ':
			count++
		case '\t':
			count += 2
		default:
			return count
		}
	}
	return count
}

func parseMindNode(text string) (*MindNode, error) {
	node := &MindNode{}
	switch {
	case strings.Contains(text, "((") && strings.HasSuffix(text, "))"):
		open := strings.Index(text, "((")
		node.Text, node.Shape = strings.TrimSpace(text[open+2:len(text)-2]), ShapeCircle
	case strings.Contains(text, "{{") && strings.HasSuffix(text, "}}"):
		open := strings.Index(text, "{{")
		node.Text, node.Shape = strings.TrimSpace(text[open+2:len(text)-2]), ShapeHexagon
	case strings.Contains(text, "[") && strings.HasSuffix(text, "]"):
		open := strings.IndexByte(text, '[')
		node.Text, node.Shape = strings.TrimSpace(text[open+1:len(text)-1]), ShapeRect
	case strings.Contains(text, "(") && strings.HasSuffix(text, ")"):
		open := strings.IndexByte(text, '(')
		node.Text, node.Shape = strings.TrimSpace(text[open+1:len(text)-1]), ShapeRounded
	default:
		node.Text, node.Shape = text, ShapeRounded
	}
	node.Text = unquote(node.Text)
	if node.Text == "" || strings.ContainsAny(node.Text, "[](){}") {
		return nil, unknown(KindMindmap, 0, text)
	}
	return node, nil
}

type mindPlaced struct {
	node     *MindNode
	children []*mindPlaced
	x, y     int
	w, h     int
	row      int
}

func layoutMindmap(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, ok := d.Data.(MindmapData)
	if !ok || data.Root == nil {
		return Scene{}, invalid(KindMindmap, 0, "missing mindmap root")
	}
	s := newScene()
	widths := make([]int, 1)
	placed := mindMeasure(data.Root, 0, fonts, &widths)
	mindAssign(placed, 0, widths, 1, new(int))
	mindDraw(&s, placed, fonts, palette, 0, 0)
	s.Width += 2
	s.Height += 2
	return s, nil
}

func mindMeasure(node *MindNode, depth int, fonts *rich.Fonts, widths *[]int) *mindPlaced {
	if depth >= len(*widths) {
		*widths = append(*widths, 0)
	}
	lines, width, lineHeight := labelSize(node.Text, fonts, 30, fonts.BodyPx, false)
	height := max(len(lines)*lineHeight, 1) + 2
	width += 4
	if width < 8 {
		width = 8
	}
	if node.Shape == ShapeCircle {
		width = max(width+2, 12)
		height = max((width+1)/2, height, 5)
		if height%2 == 0 {
			height++
		}
	}
	(*widths)[depth] = max((*widths)[depth], width)
	placed := &mindPlaced{node: node, w: width, h: height}
	for _, child := range node.Children {
		placed.children = append(placed.children, mindMeasure(child, depth+1, fonts, widths))
	}
	return placed
}

func mindAssign(placed *mindPlaced, depth int, widths []int, rowSize int, nextRow *int) {
	if len(placed.children) == 0 {
		placed.row = *nextRow
		*nextRow = *nextRow + 1
	} else {
		for _, child := range placed.children {
			mindAssign(child, depth+1, widths, rowSize, nextRow)
		}
		placed.row = (placed.children[0].row + placed.children[len(placed.children)-1].row) / 2
	}
	x := 1
	for i := 0; i < depth; i++ {
		x += widths[i] + 5
	}
	placed.x = x
	placed.y = 1 + placed.row*4
}

func mindDraw(s *Scene, placed *mindPlaced, fonts *rich.Fonts, palette rich.Palette, depth, branchIdx int) {
	col := accent(palette, branchIdx)
	if depth == 0 {
		col = accent(palette, 0)
	}
	for i, child := range placed.children {
		childBranch := branchIdx
		if depth == 0 {
			childBranch = i + 1
		}
		edgeCol := accent(palette, childBranch)
		startX, startY := placed.x+placed.w, placed.y+placed.h/2
		endX, endY := child.x, child.y+child.h/2
		midX := startX + (endX-startX)/2
		s.Polyline([]Point{{X: startX, Y: startY}, {X: midX, Y: startY}, {X: midX, Y: endY}, {X: endX, Y: endY}}, edgeCol, false, true)
	}
	fillAlpha := uint8(55)
	if depth == 0 {
		fillAlpha = 75
	}
	s.Node(Node{ID: placed.node.Text, Label: placed.node.Text, Shape: placed.node.Shape}, placed.x, placed.y, placed.w, placed.h, fonts, palette, fade(col, fillAlpha), col)
	for i, child := range placed.children {
		childBranch := branchIdx
		if depth == 0 {
			childBranch = i + 1
		}
		mindDraw(s, child, fonts, palette, depth+1, childBranch)
	}
}

func parseGit(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindGit}
	data := GitData{}
	branchIndex := map[string]int{"main": 0}
	lastHead := map[string]string{}
	current := "main"
	data.Branches = append(data.Branches, GitBranch{Name: "main"})
	auto := 1
	addCommit := func(options map[string]string, mergeParents []string, line int) error {
		id := options["id"]
		if id == "" {
			id = "c" + strconv.Itoa(auto)
		}
		auto++
		for _, existing := range data.Commits {
			if existing.ID == id {
				return invalid(KindGit, line, "duplicate commit id %q", id)
			}
		}
		message := options["msg"]
		if message == "" {
			message = options["message"]
		}
		if message == "" {
			message = id
		}
		parents := mergeParents
		if len(parents) == 0 {
			if head := lastHead[current]; head != "" {
				parents = []string{head}
			}
		}
		commit := GitCommit{ID: id, Message: message, Tag: options["tag"], Branch: current, Parents: parents, Merge: len(parents) > 1}
		data.Commits = append(data.Commits, commit)
		lastHead[current] = id
		if index, ok := branchIndex[current]; ok {
			data.Branches[index].Head = len(data.Commits) - 1
		}
		return nil
	}
	for _, line := range numberedNonBlank(lines) {
		text := line.Text
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "title":
			if d.Title != "" {
				return Diagram{}, unknown(KindGit, line.Number, text)
			}
			d.Title = unquote(strings.TrimSpace(text[len("title"):]))
		case "commit":
			options, err := parseGitOptions(strings.TrimSpace(text[len("commit"):]))
			if err != nil {
				return Diagram{}, unknown(KindGit, line.Number, text)
			}
			if err := addCommit(options, nil, line.Number); err != nil {
				return Diagram{}, err
			}
		case "branch":
			rest := strings.TrimSpace(text[len("branch"):])
			name := firstGitToken(rest)
			if _, exists := branchIndex[name]; name == "" || exists {
				return Diagram{}, unknown(KindGit, line.Number, text)
			}
			branchIndex[name] = len(data.Branches)
			data.Branches = append(data.Branches, GitBranch{Name: name, Start: len(data.Commits), Head: len(data.Commits) - 1})
			if head := lastHead[current]; head != "" {
				lastHead[name] = head
			}
			current = name
		case "checkout", "switch":
			name := strings.TrimSpace(text[len(fields[0]):])
			if _, ok := branchIndex[name]; !ok || strings.ContainsAny(name, " \t") {
				return Diagram{}, unknown(KindGit, line.Number, text)
			}
			current = name
		case "merge":
			rest := strings.TrimSpace(text[len("merge"):])
			name := firstGitToken(rest)
			if _, ok := branchIndex[name]; !ok || name == current {
				return Diagram{}, unknown(KindGit, line.Number, text)
			}
			optionsText := strings.TrimSpace(strings.TrimPrefix(rest, name))
			options, err := parseGitOptions(optionsText)
			if err != nil {
				return Diagram{}, unknown(KindGit, line.Number, text)
			}
			parents := []string{}
			if head := lastHead[current]; head != "" {
				parents = append(parents, head)
			}
			if head := lastHead[name]; head != "" {
				parents = append(parents, head)
			}
			if len(parents) != 2 {
				return Diagram{}, unknown(KindGit, line.Number, text)
			}
			if options["msg"] == "" && options["message"] == "" {
				options["msg"] = "merge " + name
			}
			if err := addCommit(options, parents, line.Number); err != nil {
				return Diagram{}, err
			}
		default:
			return Diagram{}, unknown(KindGit, line.Number, text)
		}
	}
	if len(data.Commits) == 0 {
		return Diagram{}, unknown(KindGit, 0, "empty gitGraph")
	}
	d.Data = data
	return d, nil
}

func parseGitOptions(text string) (map[string]string, error) {
	options := make(map[string]string)
	text = strings.TrimSpace(text)
	if text == "" {
		return options, nil
	}
	matches := gitOptionPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid options")
	}
	var consumed strings.Builder
	for _, match := range matches {
		start, end := match[0], match[1]
		if strings.TrimSpace(text[consumed.Len():start]) != "" {
			return nil, fmt.Errorf("invalid option boundary")
		}
		key := strings.ToLower(text[match[2]:match[3]])
		value := ""
		for i := 4; i+1 < len(match); i += 2 {
			if match[i] >= 0 {
				value = text[match[i]:match[i+1]]
				break
			}
		}
		if key == "" || value == "" {
			return nil, fmt.Errorf("empty option")
		}
		if _, exists := options[key]; exists {
			return nil, fmt.Errorf("duplicate option")
		}
		options[key] = unquote(value)
		consumed.WriteString(text[consumed.Len():end])
	}
	if strings.TrimSpace(text[consumed.Len():]) != "" {
		return nil, fmt.Errorf("trailing option text")
	}
	return options, nil
}

func firstGitToken(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func layoutGit(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, ok := d.Data.(GitData)
	if !ok || len(data.Commits) == 0 {
		return Scene{}, invalid(KindGit, 0, "missing git data")
	}
	s := newScene()
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	if titleHeight == 0 {
		titleHeight = 1
	}
	laneY := make(map[string]int, len(data.Branches))
	branchLabelW := 6
	for _, branch := range data.Branches {
		_, w, _ := labelSize(branch.Name, fonts, 14, fonts.BodyPx*0.9, false)
		branchLabelW = max(branchLabelW, w+2)
	}
	for i, branch := range data.Branches {
		laneY[branch.Name] = titleHeight + 4 + i*5
	}
	orderX := func(order int) int { return branchLabelW + 5 + order*10 }
	positions := make(map[string]Point, len(data.Commits))
	commitBranch := make(map[string]string, len(data.Commits))
	for i, commit := range data.Commits {
		positions[commit.ID] = Point{X: orderX(i), Y: laneY[commit.Branch]}
		commitBranch[commit.ID] = commit.Branch
	}
	// Draw branch labels and rails first so commits sit cleanly on top.
	for i, branch := range data.Branches {
		y := laneY[branch.Name]
		col := accent(palette, i)
		s.Badge(branch.Name, 1, y-1, branchLabelW, 1, fonts.BodyPx*0.9, col, contrastText(col, palette))
		first, last := -1, -1
		for j, commit := range data.Commits {
			if commit.Branch == branch.Name {
				if first < 0 {
					first = j
				}
				last = j
			}
		}
		if first >= 0 && last > first {
			s.Line(orderX(first), y, orderX(last), y, col, false, true)
		}
	}
	// Draw parent/fork/merge edges next.
	for _, commit := range data.Commits {
		p := positions[commit.ID]
		col := accent(palette, branchColor(data.Branches, commit.Branch))
		for _, parentID := range commit.Parents {
			parent, ok := positions[parentID]
			if !ok {
				continue
			}
			if parent.Y == p.Y {
				s.Line(parent.X, parent.Y, p.X, p.Y, col, false, true)
			} else if commit.Merge {
				// Merge: continue along the merged branch's lane, then rise into the merge commit.
				edgeCol := accent(palette, branchColor(data.Branches, commitBranch[parentID]))
				s.Polyline([]Point{{X: parent.X, Y: parent.Y}, {X: p.X, Y: parent.Y}, {X: p.X, Y: p.Y}}, edgeCol, false, true)
			} else {
				// Fork: drop from parent commit down to child branch's lane, then continue right.
				s.Polyline([]Point{{X: parent.X, Y: parent.Y}, {X: parent.X, Y: p.Y}, {X: p.X, Y: p.Y}}, col, false, true)
			}
		}
	}
	// Draw commit nodes, tags, and messages on top.
	for _, commit := range data.Commits {
		p := positions[commit.ID]
		col := accent(palette, branchColor(data.Branches, commit.Branch))
		if commit.Merge {
			s.Dot(p.X, p.Y, 10, palette.CodeBg, col)
			s.Dot(p.X, p.Y, 5, col, col)
		} else {
			s.Dot(p.X, p.Y, 9, col, foreground(palette))
		}
		if commit.Tag != "" {
			tagW := max(len([]rune(commit.Tag))+2, 5)
			tagFill := accent(palette, 3)
			s.Badge(commit.Tag, p.X-tagW/2, p.Y-2, tagW, 1, fonts.BodyPx*0.85, tagFill, contrastText(tagFill, palette))
		}
		if commit.Message != "" {
			lines, w, _ := labelSize(commit.Message, fonts, 12, fonts.BodyPx*0.85, false)
			if len(lines) > 0 {
				msgW := max(w, 6)
				s.Text(lines[0], p.X-msgW/2, p.Y+1, msgW, 1, fonts.BodyPx*0.85, AlignCenter, muted(palette), false)
			}
		}
	}
	if d.Title != "" {
		s.TextLinesBold(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	s.Width = max(s.Width, titleWidth+2, orderX(len(data.Commits))+4)
	s.Height = max(s.Height, titleHeight+4+len(data.Branches)*5+2)
	return s, nil
}

func branchColor(branches []GitBranch, name string) int {
	for i, branch := range branches {
		if branch.Name == name {
			return i
		}
	}
	return 0
}
