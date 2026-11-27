package diagram

import (
	"image/color"
	"sort"
	"strings"

	"github.com/pwn27sbx/misoku/internal/rich"
)

const (
	flowCrossGap     = 4
	flowOuterPad     = 3
	flowGroupPad     = 2
	flowGroupTopPad  = 4
	flowMaxLabelCell = 24
	flowMaxEdgeLabel = 18
)

type flowParseState struct {
	diagram   Diagram
	nodeIndex map[string]int
	groups    []int
	nextGroup int
}

func init() {
	registerWithHeader(KindFlowchart, parseFlow, layoutFlow)
}

func parseFlow(header string, lines []string) (Diagram, error) {
	direction, err := flowHeaderDirection(header)
	if err != nil {
		return Diagram{}, err
	}
	p := flowParseState{
		diagram:   Diagram{Kind: KindFlowchart, Direction: direction},
		nodeIndex: make(map[string]int),
	}
	for _, source := range numberedNonBlank(lines) {
		raw := lines[source.Number-2]
		text := strings.TrimSpace(stripMermaidComment(raw))
		for _, statement := range splitStatements(text) {
			if statement == "" {
				continue
			}
			if err := p.statement(statement, source.Number); err != nil {
				return Diagram{}, err
			}
		}
	}
	if len(p.groups) != 0 {
		line := 2
		if len(lines) != 0 {
			line = len(lines) + 1
		}
		return Diagram{}, unknown(KindFlowchart, line, "unclosed subgraph")
	}
	return p.diagram, nil
}

func flowHeaderDirection(header string) (string, error) {
	fields := strings.Fields(header)
	if len(fields) == 0 || len(fields) > 2 {
		return "", unknown(KindFlowchart, 1, header)
	}
	direction := "TD"
	if len(fields) > 1 {
		direction = strings.ToUpper(fields[1])
	}
	if !flowValidDirection(direction) {
		return "", unknown(KindFlowchart, 1, header)
	}
	return direction, nil
}

func flowValidDirection(direction string) bool {
	switch strings.ToUpper(direction) {
	case "TD", "TB", "LR", "RL", "BT":
		return true
	default:
		return false
	}
}

func (p *flowParseState) statement(statement string, line int) error {
	lower := strings.ToLower(statement)
	switch {
	case lower == "end":
		if len(p.groups) == 0 {
			return unknown(KindFlowchart, line, statement)
		}
		p.groups = p.groups[:len(p.groups)-1]
		return nil
	case flowHasKeyword(lower, "subgraph"):
		return p.openGroup(strings.TrimSpace(statement[len("subgraph"):]), line)
	case flowHasKeyword(lower, "direction"):
		direction := strings.ToUpper(strings.TrimSpace(statement[len("direction"):]))
		if !flowValidDirection(direction) {
			return unknown(KindFlowchart, line, statement)
		}
		if len(p.groups) == 0 {
			p.diagram.Direction = direction
		} else {
			p.diagram.Groups[p.groups[len(p.groups)-1]].Direction = direction
		}
		return nil
	case flowHasKeyword(lower, "title"):
		p.diagram.Title = unquote(statement[len("title"):])
		return nil
	case flowHasKeyword(lower, "acctitle"):
		p.diagram.Title = unquote(strings.TrimSpace(strings.TrimPrefix(statement[len("acctitle"):], ":")))
		return nil
	case flowHasKeyword(lower, "classdef"),
		flowHasKeyword(lower, "class"),
		flowHasKeyword(lower, "style"),
		flowHasKeyword(lower, "linkstyle"),
		flowHasKeyword(lower, "click"):
		// These directives customize presentation in Mermaid. The native IR
		// deliberately keeps its own deterministic theme and geometry.
		return nil
	default:
		return p.chain(statement, line)
	}
}

func flowHasKeyword(text, keyword string) bool {
	return text == keyword || strings.HasPrefix(text, keyword+" ") || strings.HasPrefix(text, keyword+"\t") || strings.HasPrefix(text, keyword+":")
}

func (p *flowParseState) openGroup(rest string, line int) error {
	if rest == "" {
		return unknown(KindFlowchart, line, "subgraph")
	}
	var id, label string
	if strings.HasPrefix(rest, "[") {
		tail, title, ok := flowReadShape(rest, "[", "]")
		if !ok || strings.TrimSpace(tail) != "" {
			return unknown(KindFlowchart, line, "subgraph "+rest)
		}
		p.nextGroup++
		id = "subgraph_" + flowDecimal(p.nextGroup)
		label = unquote(title)
	} else if rest[0] == '"' || rest[0] == '\'' {
		token, tail, ok := flowReadQuoted(rest)
		if !ok {
			return unknown(KindFlowchart, line, "subgraph "+rest)
		}
		id = unquote(token)
		tail = strings.TrimSpace(tail)
		label = id
		if tail != "" {
			if !strings.HasPrefix(tail, "[") {
				return unknown(KindFlowchart, line, "subgraph "+rest)
			}
			end, title, ok := flowReadShape(tail, "[", "]")
			if !ok || strings.TrimSpace(end) != "" {
				return unknown(KindFlowchart, line, "subgraph "+rest)
			}
			label = unquote(title)
		}
	} else {
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return unknown(KindFlowchart, line, "subgraph "+rest)
		}
		id = fields[0]
		label = id
		tail := strings.TrimSpace(rest[len(fields[0]):])
		if tail != "" {
			if strings.HasPrefix(tail, "[") {
				end, title, ok := flowReadShape(tail, "[", "]")
				if !ok || strings.TrimSpace(end) != "" {
					return unknown(KindFlowchart, line, "subgraph "+rest)
				}
				label = unquote(title)
			} else {
				label = unquote(tail)
			}
		}
	}
	if id == "" {
		p.nextGroup++
		id = "subgraph_" + flowDecimal(p.nextGroup)
	}
	p.diagram.Groups = append(p.diagram.Groups, Group{ID: id, Label: label})
	p.groups = append(p.groups, len(p.diagram.Groups)-1)
	return nil
}

func flowDecimal(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func (p *flowParseState) chain(statement string, line int) error {
	id, rest, err := p.node(statement, line)
	if err != nil {
		return unknown(KindFlowchart, line, statement)
	}
	previous := id
	for {
		rest = strings.TrimLeft(rest, " \t")
		if rest == "" {
			return nil
		}
		edge, tail, err := parseFlowConnector(rest, line)
		if err != nil {
			return unknown(KindFlowchart, line, statement)
		}
		next, remaining, err := p.node(tail, line)
		if err != nil {
			return unknown(KindFlowchart, line, statement)
		}
		edge.From, edge.To = previous, next
		p.diagram.Edges = append(p.diagram.Edges, edge)
		previous, rest = next, remaining
	}
}

func (p *flowParseState) node(text string, line int) (string, string, error) {
	id, rest, ok := flowReadID(text)
	if !ok {
		return "", "", unknown(KindFlowchart, line, text)
	}
	trimmed := strings.TrimLeft(rest, " \t")
	label, shape, hasShape := id, ShapeRect, false
	if tail, labelText, shapeKind, ok := flowReadNodeShape(trimmed); ok {
		label, shape, hasShape, rest = unquote(labelText), shapeKind, true, tail
	}
	index, exists := p.nodeIndex[id]
	if !exists {
		p.diagram.Nodes = append(p.diagram.Nodes, Node{ID: id, Label: label, Shape: shape})
		index = len(p.diagram.Nodes) - 1
		p.nodeIndex[id] = index
	} else if hasShape {
		p.diagram.Nodes[index].Label = label
		p.diagram.Nodes[index].Shape = shape
	}
	if len(p.groups) != 0 {
		if p.diagram.Nodes[index].Group == "" {
			p.diagram.Nodes[index].Group = p.diagram.Groups[p.groups[len(p.groups)-1]].ID
		}
		for _, groupIndex := range p.groups {
			group := &p.diagram.Groups[groupIndex]
			if !flowContainsID(group.Nodes, id) {
				group.Nodes = append(group.Nodes, id)
			}
		}
	}
	return id, rest, nil
}

func flowContainsID(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func flowReadID(text string) (string, string, bool) {
	text = strings.TrimLeft(text, " \t")
	if text == "" {
		return "", "", false
	}
	if text[0] == '"' || text[0] == '\'' {
		token, rest, ok := flowReadQuoted(text)
		return unquote(token), rest, ok && unquote(token) != ""
	}
	end := 0
	for end < len(text) {
		c := text[end]
		if c == ' ' || c == '\t' || c == '[' || c == '(' || c == '{' || c == ']' || c == ')' || c == '}' || c == '>' || c == '<' || c == '|' || c == ';' || c == '~' {
			break
		}
		if strings.HasPrefix(text[end:], "--") || strings.HasPrefix(text[end:], "-.") || strings.HasPrefix(text[end:], "==") {
			break
		}
		end++
	}
	if end == 0 {
		return "", "", false
	}
	return text[:end], text[end:], true
}

func flowReadNodeShape(text string) (rest, label string, shape NodeShape, ok bool) {
	type delimiter struct {
		open, close string
		shape       NodeShape
	}
	delimiters := [...]delimiter{
		{open: "(((", close: ")))", shape: ShapeDoubleCircle},
		{open: "([", close: "])", shape: ShapeStadium},
		{open: "[(", close: ")]", shape: ShapeCylinder},
		{open: "[[", close: "]]", shape: ShapeSubroutine},
		{open: "{{", close: "}}", shape: ShapeHexagon},
		{open: "((", close: "))", shape: ShapeCircle},
		{open: "[/", close: "/]", shape: ShapeParallelogram},
		{open: "[\\", close: "\\]", shape: ShapeParallelogramAlt},
		{open: "[", close: "]", shape: ShapeRect},
		{open: "(", close: ")", shape: ShapeRounded},
		{open: "{", close: "}", shape: ShapeDiamond},
		{open: ">", close: "]", shape: ShapeAsymmetric},
	}
	for _, delimiter := range delimiters {
		if !strings.HasPrefix(text, delimiter.open) {
			continue
		}
		if tail, content, ok := flowReadShape(text, delimiter.open, delimiter.close); ok {
			return tail, content, delimiter.shape, true
		}
	}
	return "", "", ShapeRect, false
}

func flowReadShape(text, open, close string) (rest, content string, ok bool) {
	if !strings.HasPrefix(text, open) {
		return "", "", false
	}
	depth := 1
	var quote byte
	escaped := false
	for i := len(open); i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if strings.HasPrefix(text[i:], open) {
			depth++
			i += len(open) - 1
			continue
		}
		if strings.HasPrefix(text[i:], close) {
			depth--
			if depth == 0 {
				return text[i+len(close):], text[len(open):i], true
			}
			i += len(close) - 1
		}
	}
	return "", "", false
}

func flowReadQuoted(text string) (token, rest string, ok bool) {
	if text == "" || (text[0] != '"' && text[0] != '\'') {
		return "", "", false
	}
	quote := text[0]
	escaped := false
	for i := 1; i < len(text); i++ {
		if escaped {
			escaped = false
			continue
		}
		if text[i] == '\\' {
			escaped = true
			continue
		}
		if text[i] == quote {
			return text[:i+1], text[i+1:], true
		}
	}
	return "", "", false
}

func parseFlowConnector(text string, line int) (Edge, string, error) {
	original := text
	start := ArrowNone
	if text != "" {
		switch text[0] {
		case '<':
			start = ArrowOpen
			text = text[1:]
		case 'o':
			start = ArrowCircle
			text = text[1:]
		case 'x':
			start = ArrowCross
			text = text[1:]
		}
	}
	style := EdgeSolid
	switch {
	case strings.HasPrefix(text, "~~~"):
		if start != ArrowNone {
			return Edge{}, "", unknown(KindFlowchart, line, original)
		}
		return Edge{Style: EdgeInvisible}, text[3:], nil
	case strings.HasPrefix(text, "--"):
		text = text[2:]
	case strings.HasPrefix(text, "-."):
		style = EdgeDotted
		text = text[2:]
	case strings.HasPrefix(text, "=="):
		style = EdgeThick
		text = text[2:]
	default:
		return Edge{}, "", unknown(KindFlowchart, line, original)
	}

	label := ""
	trimmed := strings.TrimLeft(text, " \t")
	if strings.HasPrefix(trimmed, "|") {
		var ok bool
		label, text, ok = flowReadPipeLabel(trimmed)
		if !ok {
			return Edge{}, "", unknown(KindFlowchart, line, original)
		}
	}

	end, consumed := flowDirectArrowhead(text, style)
	text = text[consumed:]
	if consumed == 0 && len(text) != len(trimmed) && trimmed != "" && !strings.HasPrefix(trimmed, "|") {
		if token, head, index := flowFindClosingArrow(trimmed, style); index >= 0 {
			labelText := strings.TrimSpace(trimmed[:index])
			if label == "" {
				label = unquote(labelText)
			}
			end = head
			text = trimmed[index+len(token):]
		}
	}

	trimmed = strings.TrimLeft(text, " \t")
	if strings.HasPrefix(trimmed, "|") {
		pipe, tail, ok := flowReadPipeLabel(trimmed)
		if !ok {
			return Edge{}, "", unknown(KindFlowchart, line, original)
		}
		if label == "" {
			label = pipe
		}
		text = tail
	}
	return Edge{Label: label, Style: style, Start: start, End: end}, text, nil
}

func flowDirectArrowhead(text string, style EdgeStyle) (Arrowhead, int) {
	if text == "" {
		return ArrowNone, 0
	}
	if style == EdgeDotted && strings.HasPrefix(text, "->") {
		return ArrowOpen, 2
	}
	switch text[0] {
	case '>':
		if style == EdgeThick {
			return ArrowFilled, 1
		}
		return ArrowOpen, 1
	case 'o':
		return ArrowCircle, 1
	case 'x':
		return ArrowCross, 1
	case '-':
		if style == EdgeSolid || style == EdgeDotted {
			return ArrowNone, 1
		}
	case '=':
		if style == EdgeThick {
			return ArrowNone, 1
		}
	}
	return ArrowNone, 0
}

func flowReadPipeLabel(text string) (label, rest string, ok bool) {
	if text == "" || text[0] != '|' {
		return "", "", false
	}
	var quote byte
	escaped := false
	for i := 1; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if c == '|' {
			return unquote(text[1:i]), text[i+1:], true
		}
	}
	return "", "", false
}

func flowFindClosingArrow(text string, style EdgeStyle) (token string, head Arrowhead, index int) {
	tokens := []string{"-->", "--o", "--x", "---"}
	switch style {
	case EdgeDotted:
		tokens = []string{".->", ".-o", ".-x", ".-"}
	case EdgeThick:
		tokens = []string{"==>", "==o", "==x", "==="}
	}
	var quote byte
	escaped := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		for _, candidate := range tokens {
			if !strings.HasPrefix(text[i:], candidate) {
				continue
			}
			switch candidate[len(candidate)-1] {
			case '>':
				if style == EdgeThick {
					return candidate, ArrowFilled, i
				}
				return candidate, ArrowOpen, i
			case 'o':
				return candidate, ArrowCircle, i
			case 'x':
				return candidate, ArrowCross, i
			default:
				return candidate, ArrowNone, i
			}
		}
	}
	return "", ArrowNone, -1
}

type flowOrientation struct {
	horizontal bool
	reverse    bool
}

func flowOrientationFor(direction string) flowOrientation {
	switch strings.ToUpper(direction) {
	case "LR":
		return flowOrientation{horizontal: true}
	case "RL":
		return flowOrientation{horizontal: true, reverse: true}
	case "BT":
		return flowOrientation{reverse: true}
	default:
		return flowOrientation{}
	}
}

type flowLayoutNode struct {
	node   Node
	layer  int
	x, y   int
	w, h   int
	dummy  bool
	domain int
	orient flowOrientation
	moved  bool
}

func (n *flowLayoutNode) cx() int { return n.x + n.w/2 }
func (n *flowLayoutNode) cy() int { return n.y + n.h/2 }

type flowRankSegment struct {
	from, to int
	label    string
	edge     int
}

type flowRanked struct {
	nodes    []*flowLayoutNode
	real     int
	layers   [][]int
	preds    [][]int
	succs    [][]int
	segments []flowRankSegment
	back     []bool
}

func rankFlow(d Diagram, fonts *rich.Fonts) (*flowRanked, error) {
	n := len(d.Nodes)
	r := &flowRanked{nodes: make([]*flowLayoutNode, 0, n), real: n, back: make([]bool, len(d.Edges))}
	index := make(map[string]int, n)
	for _, node := range d.Nodes {
		w, h, err := flowNodeSize(node, fonts)
		if err != nil {
			return nil, err
		}
		index[node.ID] = len(r.nodes)
		r.nodes = append(r.nodes, &flowLayoutNode{node: node, w: w, h: h, domain: -1, orient: flowOrientationFor(d.Direction)})
	}
	out := make([][]int, n)
	endpoints := make([][2]int, len(d.Edges))
	for i, edge := range d.Edges {
		from, ok := index[edge.From]
		if !ok {
			return nil, unknown(KindFlowchart, 0, "edge from unknown node "+edge.From)
		}
		to, ok := index[edge.To]
		if !ok {
			return nil, unknown(KindFlowchart, 0, "edge to unknown node "+edge.To)
		}
		endpoints[i] = [2]int{from, to}
		out[from] = append(out[from], i)
	}

	state := make([]uint8, n)
	var post []int
	var visit func(int)
	visit = func(u int) {
		state[u] = 1
		for _, edgeIndex := range out[u] {
			v := endpoints[edgeIndex][1]
			switch state[v] {
			case 0:
				visit(v)
			case 1:
				r.back[edgeIndex] = true
			}
		}
		state[u] = 2
		post = append(post, u)
	}
	for u := 0; u < n; u++ {
		if state[u] == 0 {
			visit(u)
		}
	}
	for i := len(post) - 1; i >= 0; i-- {
		u := post[i]
		for _, edgeIndex := range out[u] {
			if r.back[edgeIndex] {
				continue
			}
			v := endpoints[edgeIndex][1]
			r.nodes[v].layer = max(r.nodes[v].layer, r.nodes[u].layer+1)
		}
	}

	for edgeIndex, edge := range d.Edges {
		if r.back[edgeIndex] {
			continue
		}
		from, to := endpoints[edgeIndex][0], endpoints[edgeIndex][1]
		previous, label := from, edge.Label
		for layer := r.nodes[from].layer + 1; layer < r.nodes[to].layer; layer++ {
			dummy := &flowLayoutNode{layer: layer, w: 1, h: 1, dummy: true, domain: -1, orient: flowOrientationFor(d.Direction)}
			r.nodes = append(r.nodes, dummy)
			current := len(r.nodes) - 1
			r.segments = append(r.segments, flowRankSegment{from: previous, to: current, label: label, edge: edgeIndex})
			previous, label = current, ""
		}
		r.segments = append(r.segments, flowRankSegment{from: previous, to: to, label: label, edge: edgeIndex})
	}

	depth := 0
	for _, node := range r.nodes {
		depth = max(depth, node.layer+1)
	}
	r.layers = make([][]int, depth)
	for i, node := range r.nodes {
		r.layers[node.layer] = append(r.layers[node.layer], i)
	}
	r.preds = make([][]int, len(r.nodes))
	r.succs = make([][]int, len(r.nodes))
	for _, segment := range r.segments {
		r.preds[segment.to] = append(r.preds[segment.to], segment.from)
		r.succs[segment.from] = append(r.succs[segment.from], segment.to)
	}
	flowBarycenter(r)
	return r, nil
}

func flowNodeSize(node Node, fonts *rich.Fonts) (int, int, error) {
	if fonts == nil {
		return 0, 0, unknown(KindFlowchart, 0, "font metrics")
	}
	label := node.Label
	if label == "" {
		label = node.ID
	}
	lines, width, lineHeight := labelSize(label, fonts, flowMaxLabelCell, fonts.BodyPx, false)
	width = max(width+4, 7)
	height := max(len(lines)*lineHeight+2, 3)
	switch node.Shape {
	case ShapeStadium:
		width = max(width, 10)
	case ShapeSubroutine:
		width = max(width, 9)
	case ShapeCylinder:
		width = max(width, 9)
		height += 2
	case ShapeCircle:
		width = max(width+2, height*2, 10)
		height = max((width+1)/2, height+2, 5)
	case ShapeDiamond:
		width = max(width*2+2, 12)
		height = max(height+2, 5)
	case ShapeHexagon:
		width = max(width+4, 11)
	case ShapeParallelogram, ShapeParallelogramAlt:
		width += 4
	case ShapeDoubleCircle:
		width = max(width+4, height*2+2, 12)
		height = max((width+1)/2, height+2, 6)
	case ShapeAsymmetric:
		width += 3
	}
	return width, height, nil
}

func flowBarycenter(r *flowRanked) {
	order := make([]float64, len(r.nodes))
	renumber := func(layer int) {
		for i, id := range r.layers[layer] {
			order[id] = float64(i)
		}
	}
	for layer := range r.layers {
		renumber(layer)
	}
	sortLayer := func(layer int, neighbors [][]int) {
		keys := make(map[int]float64, len(r.layers[layer]))
		for _, id := range r.layers[layer] {
			key := order[id]
			if adjacent := neighbors[id]; len(adjacent) != 0 {
				sum := 0.0
				for _, other := range adjacent {
					sum += order[other]
				}
				key = sum / float64(len(adjacent))
			}
			keys[id] = key
		}
		sort.SliceStable(r.layers[layer], func(i, j int) bool {
			return keys[r.layers[layer][i]] < keys[r.layers[layer][j]]
		})
		renumber(layer)
	}
	for range 4 {
		for layer := 1; layer < len(r.layers); layer++ {
			sortLayer(layer, r.preds)
		}
		for layer := len(r.layers) - 2; layer >= 0; layer-- {
			sortLayer(layer, r.succs)
		}
	}
}

type flowTrackSpan struct{ a, b int }

func flowPlaceCross(r *flowRanked, horizontal bool) {
	size := func(n *flowLayoutNode) int {
		if horizontal {
			return n.h
		}
		return n.w
	}
	center := func(n *flowLayoutNode) int {
		if horizontal {
			return n.cy()
		}
		return n.cx()
	}
	setCenter := func(n *flowLayoutNode, value int) {
		if horizontal {
			n.y = value - n.h/2
		} else {
			n.x = value - n.w/2
		}
	}
	place := func(layer int, neighbors [][]int) {
		right := -1 << 30
		for _, id := range r.layers[layer] {
			node := r.nodes[id]
			want := center(node)
			if adjacent := neighbors[id]; len(adjacent) != 0 {
				sum := 0
				for _, other := range adjacent {
					sum += center(r.nodes[other])
				}
				want = sum/len(adjacent) - size(node)/2
			}
			value := max(want, right+flowCrossGap)
			setCenter(node, value+size(node)/2)
			right = value + size(node)
		}
	}
	for _, layer := range r.layers {
		position := 0
		for _, id := range layer {
			node := r.nodes[id]
			setCenter(node, position+size(node)/2)
			position += size(node) + flowCrossGap
		}
	}
	for layer := 1; layer < len(r.layers); layer++ {
		place(layer, r.preds)
	}
	for layer := len(r.layers) - 2; layer >= 0; layer-- {
		place(layer, r.succs)
	}
	for layer := 1; layer < len(r.layers); layer++ {
		place(layer, r.preds)
	}
	minimum := 1 << 30
	for _, node := range r.nodes {
		minimum = min(minimum, center(node)-size(node)/2)
	}
	for _, node := range r.nodes {
		setCenter(node, center(node)-minimum+flowOuterPad)
	}
}

func flowAssignTracks(r *flowRanked, d Diagram, fonts *rich.Fonts, horizontal bool) ([][][]flowTrackSpan, []int, int, []int, []int) {
	tracks := make([][][]flowTrackSpan, len(r.layers))
	edgeLane := make([]int, len(d.Edges))
	for i := range edgeLane {
		edgeLane[i] = -1
	}
	center := func(n *flowLayoutNode) int {
		if horizontal {
			return n.cy()
		}
		return n.cx()
	}
	for _, segment := range r.segments {
		a, b := center(r.nodes[segment.from]), center(r.nodes[segment.to])
		if a > b {
			a, b = b, a
		}
		layer := r.nodes[segment.from].layer
		lane := 0
		for ; lane < len(tracks[layer]); lane++ {
			clear := true
			for _, span := range tracks[layer][lane] {
				if !(b+1 < span.a || span.b+1 < a) {
					clear = false
					break
				}
			}
			if clear {
				break
			}
		}
		if lane == len(tracks[layer]) {
			tracks[layer] = append(tracks[layer], nil)
		}
		tracks[layer][lane] = append(tracks[layer][lane], flowTrackSpan{a: a, b: b})
		if edgeLane[segment.edge] < 0 {
			edgeLane[segment.edge] = lane
		}
	}
	labelWidth, labelHeight, laneStep := make([]int, len(r.layers)), make([]int, len(r.layers)), 1
	for _, edge := range d.Edges {
		if edge.Label == "" {
			continue
		}
		_, width, height := flowEdgeLabelSize(edge.Label, fonts)
		for layer := range r.layers {
			labelWidth[layer] = max(labelWidth[layer], width)
			labelHeight[layer] = max(labelHeight[layer], height)
		}
		laneStep = max(laneStep, height)
	}
	return tracks, edgeLane, laneStep, labelWidth, labelHeight
}

func flowPlacePrimary(r *flowRanked, tracks [][][]flowTrackSpan, laneStep, labelStep int, horizontal bool, labelWidth []int) []int {
	if len(r.layers) == 0 {
		return nil
	}
	bands := make([]int, len(r.layers))
	for _, node := range r.nodes {
		size := node.h
		if horizontal {
			size = node.w
		}
		bands[node.layer] = max(bands[node.layer], size, 1)
	}
	starts := make([]int, len(r.layers))
	starts[0] = flowOuterPad
	for layer := 1; layer < len(r.layers); layer++ {
		gap := max(4, len(tracks[layer-1])*laneStep+2, labelStep+2)
		if horizontal {
			gap = max(gap, labelWidth[layer-1]+3)
		}
		starts[layer] = starts[layer-1] + bands[layer-1] + gap
	}
	for _, node := range r.nodes {
		if horizontal {
			node.x = starts[node.layer]
		} else {
			node.y = starts[node.layer]
		}
	}
	return starts
}

func flowGroupParents(d Diagram) []int {
	parents := make([]int, len(d.Groups))
	for i := range parents {
		parents[i] = -1
	}
	for i := range d.Groups {
		for j := i - 1; j >= 0; j-- {
			if flowNodeSetContains(d.Groups[j].Nodes, d.Groups[i].Nodes) {
				parents[i] = j
				break
			}
		}
	}
	return parents
}

func flowNodeSetContains(superset, subset []string) bool {
	for _, id := range subset {
		if !flowContainsID(superset, id) {
			return false
		}
	}
	return true
}

func flowApplyGroupDirections(r *flowRanked, d Diagram) {
	parents := flowGroupParents(d)
	states := make([]flowOrientation, len(d.Groups))
	root := flowOrientationFor(d.Direction)
	for i, group := range d.Groups {
		state := root
		if parents[i] >= 0 {
			state = states[parents[i]]
		}
		if group.Direction != "" {
			state = flowOrientationFor(group.Direction)
		}
		states[i] = state
		if group.Direction == "" || len(group.Nodes) == 0 {
			continue
		}
		inherited := root
		if parents[i] >= 0 {
			inherited = states[parents[i]]
		}
		if state.horizontal != inherited.horizontal {
			flowTransposeNodes(r, group.Nodes)
		} else if state.reverse != inherited.reverse {
			flowReflectNodes(r, group.Nodes, inherited.horizontal)
		}
	}
	for i := 0; i < len(d.Nodes); i++ {
		node := r.nodes[i]
		node.domain = -1
		for groupIndex := len(d.Groups) - 1; groupIndex >= 0; groupIndex-- {
			if flowContainsID(d.Groups[groupIndex].Nodes, node.node.ID) {
				node.domain = groupIndex
				break
			}
		}
		if node.domain >= 0 {
			node.orient = states[node.domain]
		} else {
			node.orient = root
		}
	}
}

func flowTransposeNodes(r *flowRanked, ids []string) {
	minX, minY, _, _, found := flowNodeBounds(r, ids)
	if !found {
		return
	}
	for _, id := range ids {
		node := flowNodeByID(r, id)
		if node == nil {
			continue
		}
		cx, cy := node.cx(), node.cy()
		node.x = minX + cy - minY - node.w/2
		node.y = minY + cx - minX - node.h/2
		node.moved = true
	}
}

func flowReflectNodes(r *flowRanked, ids []string, horizontal bool) {
	minX, minY, maxX, maxY, found := flowNodeBounds(r, ids)
	if !found {
		return
	}
	for _, id := range ids {
		node := flowNodeByID(r, id)
		if node == nil {
			continue
		}
		if horizontal {
			node.x = minX + maxX - (node.x + node.w)
		} else {
			node.y = minY + maxY - (node.y + node.h)
		}
		node.moved = true
	}
}

func flowNodeByID(r *flowRanked, id string) *flowLayoutNode {
	for i := 0; i < r.real; i++ {
		if r.nodes[i].node.ID == id {
			return r.nodes[i]
		}
	}
	return nil
}

func flowNodeBounds(r *flowRanked, ids []string) (minX, minY, maxX, maxY int, found bool) {
	minX, minY, maxX, maxY = 1<<30, 1<<30, -1<<30, -1<<30
	for _, id := range ids {
		node := flowNodeByID(r, id)
		if node == nil {
			continue
		}
		found = true
		minX, minY = min(minX, node.x), min(minY, node.y)
		maxX, maxY = max(maxX, node.x+node.w), max(maxY, node.y+node.h)
	}
	return
}

type flowDrawEdge struct {
	edge       Edge
	from       *flowLayoutNode
	to         *flowLayoutNode
	path       []Point
	label      Point
	lines      []string
	width      int
	height     int
	lineHeight int
}

func flowBuildEdges(r *flowRanked, d Diagram, fonts *rich.Fonts, starts []int, edgeLane []int, laneStep int, horizontal bool) []flowDrawEdge {
	edges := make([]flowDrawEdge, 0, len(d.Edges))
	for i, edge := range d.Edges {
		if r.back[i] {
			continue
		}
		from, to := flowNodeByID(r, edge.From), flowNodeByID(r, edge.To)
		if from == nil || to == nil {
			continue
		}
		path := flowLayeredPath(from, to, starts, edgeLane[i], laneStep, horizontal)
		if from.moved || to.moved || from.domain != to.domain {
			path = flowGenericPath(from, to)
		}
		draw := flowDrawEdge{edge: edge, from: from, to: to, path: path}
		if edge.Label != "" {
			draw.lines, draw.width, draw.lineHeight = flowEdgeLabelSize(edge.Label, fonts)
			draw.height = len(draw.lines) * draw.lineHeight
			draw.label = flowLabelPoint(path, draw.width, draw.height)
		}
		edges = append(edges, draw)
	}
	return edges
}

func flowLayeredPath(from, to *flowLayoutNode, starts []int, lane, laneStep int, horizontal bool) []Point {
	if horizontal {
		start := Point{X: from.x + from.w, Y: from.cy()}
		end := Point{X: to.x, Y: to.cy()}
		if start.Y == end.Y {
			return []Point{start, end}
		}
		track := starts[from.layer] + from.w + 1 + lane*laneStep
		return []Point{start, {X: track, Y: start.Y}, {X: track, Y: end.Y}, end}
	}
	start := Point{X: from.cx(), Y: from.y + from.h}
	end := Point{X: to.cx(), Y: to.y}
	if start.X == end.X {
		return []Point{start, end}
	}
	track := starts[from.layer] + from.h + 1 + lane*laneStep
	return []Point{start, {X: start.X, Y: track}, {X: end.X, Y: track}, end}
}

func flowGenericPath(from, to *flowLayoutNode) []Point {
	horizontal := from.orient.horizontal && to.orient.horizontal
	if from.domain != to.domain {
		horizontal = abs((to.x+to.w/2)-(from.x+from.w/2)) >= abs((to.y+to.h/2)-(from.y+from.h/2))
	}
	if horizontal {
		startX, endX := from.cx(), to.cx()
		start := Point{X: from.x + from.w, Y: from.cy()}
		end := Point{X: to.x, Y: to.cy()}
		if startX > endX {
			start.X = from.x
			end.X = to.x + to.w
		}
		if start.Y == end.Y {
			return []Point{start, end}
		}
		middle := (start.X + end.X) / 2
		return []Point{start, {X: middle, Y: start.Y}, {X: middle, Y: end.Y}, end}
	}
	start := Point{X: from.cx(), Y: from.y + from.h}
	end := Point{X: to.cx(), Y: to.y}
	if from.cy() > to.cy() {
		start.Y = from.y
		end.Y = to.y + to.h
	}
	if start.X == end.X {
		return []Point{start, end}
	}
	middle := (start.Y + end.Y) / 2
	return []Point{start, {X: start.X, Y: middle}, {X: end.X, Y: middle}, end}
}

func flowEdgeLabelSize(label string, fonts *rich.Fonts) ([]string, int, int) {
	lines, width, lineHeight := labelSize(label, fonts, flowMaxEdgeLabel, fonts.BodyPx, false)
	return lines, max(width+2, 3), lineHeight
}

func flowLabelPoint(path []Point, width, height int) Point {
	if len(path) == 0 {
		return Point{}
	}
	if len(path) == 1 {
		return Point{X: path[0].X - width/2, Y: path[0].Y - height/2}
	}
	// Place the label at the midpoint of the longest segment so it never sits
	// on top of the target node endpoint.
	bestIdx, bestLen := 0, -1
	for i := 0; i < len(path)-1; i++ {
		segLen := abs(path[i+1].X-path[i].X) + abs(path[i+1].Y-path[i].Y)
		if segLen > bestLen {
			bestLen = segLen
			bestIdx = i
		}
	}
	a, b := path[bestIdx], path[bestIdx+1]
	mx, my := (a.X+b.X)/2, (a.Y+b.Y)/2
	if a.Y == b.Y {
		return Point{X: mx - width/2, Y: my - height - 1}
	}
	return Point{X: mx - width/2, Y: my - height/2}
}

type flowLoop struct {
	edge       Edge
	path       []Point
	lines      []string
	width      int
	height     int
	lineHeight int
	label      Point
}

func flowBuildLoops(r *flowRanked, d Diagram, fonts *rich.Fonts, horizontal bool) ([]flowLoop, int, int) {
	var loops []flowLoop
	primaryMax, crossMax := 0, 0
	for i := 0; i < len(d.Nodes); i++ {
		node := r.nodes[i]
		if horizontal {
			primaryMax = max(primaryMax, node.x+node.w)
			crossMax = max(crossMax, node.y+node.h)
		} else {
			primaryMax = max(primaryMax, node.y+node.h)
			crossMax = max(crossMax, node.x+node.w)
		}
	}
	crossCursor := crossMax
	for i, edge := range d.Edges {
		if !r.back[i] {
			continue
		}
		from, to := flowNodeByID(r, edge.From), flowNodeByID(r, edge.To)
		if from == nil || to == nil {
			continue
		}
		loop := flowLoop{edge: edge}
		if edge.Label != "" && edge.Style != EdgeInvisible {
			loop.lines, loop.width, loop.lineHeight = flowEdgeLabelSize(edge.Label, fonts)
			loop.height = len(loop.lines) * loop.lineHeight
		}
		lane := crossCursor + 2
		if horizontal {
			if from == to {
				loop.path = []Point{
					{X: from.cx(), Y: from.y + from.h},
					{X: from.cx(), Y: lane},
					{X: from.x + from.w + 2, Y: lane},
					{X: from.x + from.w + 2, Y: from.cy()},
					{X: from.x + from.w, Y: from.cy()},
				}
				primaryMax = max(primaryMax, from.x+from.w+3)
			} else {
				loop.path = []Point{
					{X: from.cx(), Y: from.y + from.h},
					{X: from.cx(), Y: lane},
					{X: to.cx(), Y: lane},
					{X: to.cx(), Y: to.y + to.h},
				}
			}
			loop.label = Point{X: (from.cx()+to.cx())/2 - loop.width/2, Y: lane + 1}
			crossMax = max(crossMax, lane+1+loop.height)
			crossCursor = crossMax
		} else {
			if from == to {
				loop.path = []Point{
					{X: from.x + from.w, Y: from.cy()},
					{X: lane, Y: from.cy()},
					{X: lane, Y: from.y + from.h + 2},
					{X: from.cx(), Y: from.y + from.h + 2},
					{X: from.cx(), Y: from.y + from.h},
				}
				primaryMax = max(primaryMax, from.y+from.h+3)
			} else {
				loop.path = []Point{
					{X: from.x + from.w, Y: from.cy()},
					{X: lane, Y: from.cy()},
					{X: lane, Y: to.cy()},
					{X: to.x + to.w, Y: to.cy()},
				}
			}
			loop.label = Point{X: lane + 1, Y: (from.cy()+to.cy())/2 - loop.height/2}
			crossMax = max(crossMax, lane+1+loop.width)
			crossCursor = crossMax
		}
		loops = append(loops, loop)
	}
	return loops, primaryMax, crossMax
}

type flowGroupBox struct {
	x, y, w, h int
	lines      []string
	lineHeight int
}

func flowBuildGroupBoxes(r *flowRanked, d Diagram, fonts *rich.Fonts, horizontal bool, primaryMax, crossMax int) []flowGroupBox {
	parents := flowGroupParents(d)
	boxes := make([]flowGroupBox, len(d.Groups))
	emptyX := crossMax + 2
	for i, group := range d.Groups {
		lines, width, lineHeight := labelSize(group.Label, fonts, flowMaxLabelCell, fonts.BodyPx, false)
		if group.Label == "" {
			lines, width, lineHeight = labelSize(group.ID, fonts, flowMaxLabelCell, fonts.BodyPx, false)
		}
		boxes[i] = flowGroupBox{lines: lines, lineHeight: lineHeight, w: max(width+4, 12), h: max(len(lines)*lineHeight+2, 4)}
		if len(group.Nodes) == 0 {
			if horizontal {
				boxes[i].x, boxes[i].y = flowOuterPad, emptyX
				emptyX += boxes[i].h + 2
			} else {
				boxes[i].x, boxes[i].y = emptyX, flowOuterPad
				emptyX += boxes[i].w + 2
			}
		}
	}
	for i := len(d.Groups) - 1; i >= 0; i-- {
		if len(d.Groups[i].Nodes) == 0 && parents[i] < 0 {
			continue
		}
		minX, minY, maxX, maxY, found := flowNodeBounds(r, d.Groups[i].Nodes)
		for child := i + 1; child < len(d.Groups); child++ {
			if parents[child] != i {
				continue
			}
			found = true
			minX, minY = min(minX, boxes[child].x), min(minY, boxes[child].y)
			maxX, maxY = max(maxX, boxes[child].x+boxes[child].w), max(maxY, boxes[child].y+boxes[child].h)
		}
		if !found {
			continue
		}
		boxes[i].x = max(0, minX-flowGroupPad)
		boxes[i].y = max(0, minY-flowGroupTopPad)
		boxes[i].w = max(boxes[i].w, maxX-boxes[i].x+flowGroupPad)
		boxes[i].h = max(boxes[i].h, maxY-boxes[i].y+flowGroupPad)
	}
	return boxes
}

func layoutFlow(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	if fonts == nil {
		return Scene{}, unknown(KindFlowchart, 0, "font metrics")
	}
	if d.Direction == "" {
		d.Direction = "TD"
	}
	if !flowValidDirection(d.Direction) {
		return Scene{}, unknown(KindFlowchart, 0, "direction "+d.Direction)
	}
	r, err := rankFlow(d, fonts)
	if err != nil {
		return Scene{}, err
	}
	horizontal := flowOrientationFor(d.Direction).horizontal
	flowPlaceCross(r, horizontal)
	tracks, edgeLane, laneStep, labelWidth, labelHeight := flowAssignTracks(r, d, fonts, horizontal)
	labelStep := 1
	for _, height := range labelHeight {
		labelStep = max(labelStep, height)
	}
	starts := flowPlacePrimary(r, tracks, laneStep, labelStep, horizontal, labelWidth)
	flowApplyGroupDirections(r, d)
	edges := flowBuildEdges(r, d, fonts, starts, edgeLane, laneStep, horizontal)
	loops, primaryMax, crossMax := flowBuildLoops(r, d, fonts, horizontal)
	groups := flowBuildGroupBoxes(r, d, fonts, horizontal, primaryMax, crossMax)

	s := newScene()
	for i := range d.Groups {
		box := groups[i]
		s.Rounded(box.x, box.y, box.w, box.h, 1, color.RGBA{}, accent(palette, i))
		s.TextLines(box.lines, box.x+2, box.y, box.w-4, box.lineHeight, fonts.BodyPx, AlignLeft, accent(palette, i), false)
	}
	stroke := palette.Link
	if stroke.A == 0 {
		stroke = rule(palette)
	}
	for _, edge := range edges {
		flowDrawPath(&s, edge.path, edge.edge, stroke)
	}
	for _, loop := range loops {
		flowDrawPath(&s, loop.path, loop.edge, stroke)
	}
	for i, node := range d.Nodes {
		placed := r.nodes[i]
		fill := palette.CodeBg
		s.Node(node, placed.x, placed.y, placed.w, placed.h, fonts, palette, fill, accent(palette, i))
	}
	for _, edge := range edges {
		if edge.edge.Label == "" || edge.edge.Style == EdgeInvisible {
			continue
		}
		flowDrawLabel(&s, edge.label, edge.lines, edge.width, edge.height, edge.lineHeight, fonts, palette)
	}
	for _, loop := range loops {
		if loop.edge.Label == "" || loop.edge.Style == EdgeInvisible {
			continue
		}
		flowDrawLabel(&s, loop.label, loop.lines, loop.width, loop.height, loop.lineHeight, fonts, palette)
	}

	if horizontal {
		s.Width = max(s.Width, primaryMax+flowOuterPad)
		s.Height = max(s.Height, crossMax+flowOuterPad)
	} else {
		s.Width = max(s.Width, crossMax+flowOuterPad)
		s.Height = max(s.Height, primaryMax+flowOuterPad)
	}
	for _, box := range groups {
		s.Width = max(s.Width, box.x+box.w+flowOuterPad)
		s.Height = max(s.Height, box.y+box.h+flowOuterPad)
	}
	s.Width, s.Height = max(s.Width, 1), max(s.Height, 1)
	return flowMirrorScene(s, d.Direction), nil
}

func flowDrawPath(s *Scene, path []Point, edge Edge, stroke color.RGBA) {
	if edge.Style == EdgeInvisible || len(path) < 2 {
		return
	}
	dashed, thick := edge.Style == EdgeDotted, edge.Style == EdgeThick
	if len(path) == 2 {
		s.Arrow(path[0].X, path[0].Y, path[1].X, path[1].Y, stroke, edge.Style, edge.End, edge.Start)
		return
	}
	s.Polyline(path, stroke, dashed, thick)
	if edge.Start != ArrowNone {
		s.Arrow(path[0].X, path[0].Y, path[1].X, path[1].Y, stroke, edge.Style, ArrowNone, edge.Start)
	}
	if edge.End != ArrowNone {
		last := len(path) - 1
		s.Arrow(path[last-1].X, path[last-1].Y, path[last].X, path[last].Y, stroke, edge.Style, edge.End, ArrowNone)
	}
}

func flowDrawLabel(s *Scene, at Point, lines []string, width, height, lineHeight int, fonts *rich.Fonts, palette rich.Palette) {
	at.X, at.Y = max(at.X, 0), max(at.Y, 0)
	fill := palette.CodeBg
	s.Rect(at.X, at.Y, width, height, fill, color.RGBA{})
	s.TextLines(lines, at.X+1, at.Y, width-2, lineHeight, fonts.BodyPx, AlignCenter, foreground(palette), false)
}

func flowMirrorScene(s Scene, direction string) Scene {
	orientation := flowOrientationFor(direction)
	if !orientation.reverse {
		return s
	}
	out := s
	out.Commands = append([]Command(nil), s.Commands...)
	for i := range out.Commands {
		command := &out.Commands[i]
		switch command.Kind {
		case CommandLine, CommandArrow:
			if orientation.horizontal {
				command.X, command.X2 = s.Width-command.X, s.Width-command.X2
			} else {
				command.Y, command.Y2 = s.Height-command.Y, s.Height-command.Y2
			}
		case CommandPolyline, CommandPolygon:
			for j := range command.Points {
				if orientation.horizontal {
					command.Points[j].X = s.Width - command.Points[j].X
				} else {
					command.Points[j].Y = s.Height - command.Points[j].Y
				}
			}
		default:
			if orientation.horizontal {
				command.X = s.Width - command.X - command.W
				if command.Kind == CommandParallelogram {
					command.Slant = -command.Slant
				}
			} else {
				command.Y = s.Height - command.Y - command.H
			}
		}
	}
	return out
}
