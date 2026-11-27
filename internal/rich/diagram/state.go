package diagram

import (
	"image/color"
	"sort"
	"strings"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// StateNodeKind identifies a normal state or a state-diagram pseudostate.
type StateNodeKind int

const (
	// StateNormal is a labelled state or composite state.
	StateNormal StateNodeKind = iota
	// StateStart is an initial pseudostate.
	StateStart
	// StateEnd is a final pseudostate.
	StateEnd
	// StateChoice is a choice pseudostate.
	StateChoice
	// StateFork is a fork pseudostate.
	StateFork
	// StateJoin is a join pseudostate.
	StateJoin
)

// StateNode is a state, pseudostate, or composite-state boundary.
type StateNode struct {
	ID          string
	Label       string
	Description string
	Kind        StateNodeKind
	Parent      string
}

// StateGroup is a composite state block.
type StateGroup struct {
	ID        string
	Label     string
	Parent    string
	Direction string
}

// StateTransition is a labelled directed transition.
type StateTransition struct {
	From  string
	To    string
	Label string
}

// StateNote is a note anchored to a state and placed to its left or right.
type StateNote struct {
	Anchor   string
	Position string
	Text     string
}

// StateData is the semantic payload of a state diagram.
type StateData struct {
	Nodes       []StateNode
	Groups      []StateGroup
	Transitions []StateTransition
	Notes       []StateNote
}

type stateParseState struct {
	diagram  Diagram
	data     StateData
	nodes    map[string]int
	declared map[string]bool
	groups   map[string]int
	stack    []int
	starts   int
	ends     int
}

func init() {
	register(KindState, parseState, layoutState)
}

func parseState(lines []string) (Diagram, error) {
	p := stateParseState{
		diagram:  Diagram{Kind: KindState},
		nodes:    make(map[string]int),
		declared: make(map[string]bool),
		groups:   make(map[string]int),
	}
	for _, source := range numberedNonBlank(lines) {
		for _, statement := range splitStatements(source.Text) {
			if statement == "" {
				continue
			}
			if err := p.statement(statement, source.Number); err != nil {
				return Diagram{}, err
			}
		}
	}
	if len(p.stack) != 0 {
		return Diagram{}, unknown(KindState, len(lines)+1, "unclosed composite state")
	}
	p.diagram.Data = p.data
	return p.diagram, nil
}

func (p *stateParseState) statement(statement string, line int) error {
	lower := strings.ToLower(statement)
	switch {
	case sequenceHasKeyword(lower, "title"):
		if p.diagram.Title != "" {
			return unknown(KindState, line, statement)
		}
		title := unquote(strings.TrimSpace(statement[len("title"):]))
		if title == "" {
			return unknown(KindState, line, statement)
		}
		p.diagram.Title = title
		return nil
	case sequenceHasKeyword(lower, "direction"):
		direction := strings.ToUpper(strings.TrimSpace(statement[len("direction"):]))
		if direction != "TD" && direction != "LR" {
			return unknown(KindState, line, statement)
		}
		if len(p.stack) == 0 {
			p.diagram.Direction = direction
		} else {
			p.data.Groups[p.stack[len(p.stack)-1]].Direction = direction
		}
		return nil
	case sequenceHasKeyword(lower, "note"):
		return p.note(statement, line)
	case sequenceHasKeyword(lower, "state"):
		return p.state(statement[len("state"):], line)
	case statement == "}":
		if len(p.stack) == 0 {
			return unknown(KindState, line, statement)
		}
		p.stack = p.stack[:len(p.stack)-1]
		return nil
	default:
		if strings.Contains(statement, "-->") {
			return p.transition(statement, line)
		}
		return p.description(statement, line)
	}
}

func (p *stateParseState) currentGroup() string {
	if len(p.stack) == 0 {
		return ""
	}
	return p.data.Groups[p.stack[len(p.stack)-1]].ID
}

func (p *stateParseState) state(rest string, line int) error {
	original := "state " + strings.TrimSpace(rest)
	rest = strings.TrimSpace(rest)
	composite := strings.HasSuffix(rest, "{")
	if composite {
		rest = strings.TrimSpace(strings.TrimSuffix(rest, "{"))
	}
	kind := StateNormal
	if strings.HasSuffix(rest, ">>") {
		start := strings.LastIndex(rest, "<<")
		if start < 0 {
			return unknown(KindState, line, original)
		}
		switch strings.ToLower(strings.TrimSpace(rest[start+2 : len(rest)-2])) {
		case "choice":
			kind = StateChoice
		case "fork":
			kind = StateFork
		case "join":
			kind = StateJoin
		default:
			return unknown(KindState, line, original)
		}
		rest = strings.TrimSpace(rest[:start])
	} else if strings.Contains(rest, "<<") || strings.Contains(rest, ">>") {
		return unknown(KindState, line, original)
	}

	id, label, ok := stateDeclaration(rest)
	if !ok || !stateValidID(id) || strings.TrimSpace(label) == "" {
		return unknown(KindState, line, original)
	}
	parent := p.currentGroup()
	if index, exists := p.nodes[id]; exists {
		if p.declared[id] {
			return unknown(KindState, line, original)
		}
		p.data.Nodes[index].Label = label
		p.data.Nodes[index].Kind = kind
		p.data.Nodes[index].Parent = parent
	} else {
		p.data.Nodes = append(p.data.Nodes, StateNode{ID: id, Label: label, Kind: kind, Parent: parent})
		p.nodes[id] = len(p.data.Nodes) - 1
	}
	p.declared[id] = true
	if composite {
		if kind != StateNormal {
			return unknown(KindState, line, original)
		}
		p.data.Groups = append(p.data.Groups, StateGroup{ID: id, Label: label, Parent: parent})
		p.groups[id] = len(p.data.Groups) - 1
		p.stack = append(p.stack, len(p.data.Groups)-1)
	}
	return nil
}

func stateDeclaration(rest string) (string, string, bool) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", "", false
	}
	if rest[0] == '"' || rest[0] == '\'' {
		token, tail, ok := flowReadQuoted(rest)
		if !ok {
			return "", "", false
		}
		fields := strings.Fields(strings.TrimSpace(tail))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "as") {
			return "", "", false
		}
		return fields[1], unquote(token), true
	}
	fields := strings.Fields(rest)
	if len(fields) == 1 {
		return fields[0], fields[0], true
	}
	if len(fields) >= 3 && strings.EqualFold(fields[1], "as") {
		tail := strings.TrimSpace(rest[len(fields[0]):])
		tail = strings.TrimSpace(tail[len(fields[1]):])
		return fields[0], unquote(tail), true
	}
	return "", "", false
}

func stateValidID(id string) bool {
	if id == "" || strings.TrimSpace(id) != id {
		return false
	}
	return !strings.ContainsAny(id, " \t\r\n:{}<>[];") && !strings.Contains(id, "-->")
}

func (p *stateParseState) ensureNode(id string, line int) (string, error) {
	if !stateValidID(id) {
		return "", unknown(KindState, line, id)
	}
	if _, exists := p.nodes[id]; !exists {
		p.data.Nodes = append(p.data.Nodes, StateNode{ID: id, Label: id, Parent: p.currentGroup()})
		p.nodes[id] = len(p.data.Nodes) - 1
	}
	return id, nil
}

func (p *stateParseState) pseudostate(end bool, line int) string {
	if end {
		p.ends++
		if p.ends == 1 {
			return "@end"
		}
		return "@end_" + flowDecimal(p.ends)
	}
	p.starts++
	if p.starts == 1 {
		return "@start"
	}
	return "@start_" + flowDecimal(p.starts)
}

func (p *stateParseState) transition(statement string, line int) error {
	index := strings.Index(statement, "-->")
	fromText := strings.TrimSpace(statement[:index])
	right := strings.TrimSpace(statement[index+3:])
	toText, label, _ := strings.Cut(right, ":")
	toText = strings.TrimSpace(toText)
	label = unquote(strings.TrimSpace(label))
	if fromText == "" || toText == "" || fromText == "[*]" && toText == "[*]" {
		return unknown(KindState, line, statement)
	}
	var from, to string
	var err error
	if fromText == "[*]" {
		from = p.pseudostate(false, line)
		p.data.Nodes = append(p.data.Nodes, StateNode{ID: from, Kind: StateStart, Parent: p.currentGroup()})
		p.nodes[from] = len(p.data.Nodes) - 1
	} else {
		from, err = p.ensureNode(fromText, line)
		if err != nil {
			return err
		}
	}
	if toText == "[*]" {
		to = p.pseudostate(true, line)
		p.data.Nodes = append(p.data.Nodes, StateNode{ID: to, Kind: StateEnd, Parent: p.currentGroup()})
		p.nodes[to] = len(p.data.Nodes) - 1
	} else {
		to, err = p.ensureNode(toText, line)
		if err != nil {
			return err
		}
	}
	p.data.Transitions = append(p.data.Transitions, StateTransition{From: from, To: to, Label: label})
	return nil
}

func (p *stateParseState) description(statement string, line int) error {
	id, text, found := strings.Cut(statement, ":")
	id = strings.TrimSpace(id)
	if !found || !stateValidID(id) {
		return unknown(KindState, line, statement)
	}
	index, exists := p.nodes[id]
	if !exists {
		return unknown(KindState, line, statement)
	}
	p.data.Nodes[index].Description = unquote(strings.TrimSpace(text))
	return nil
}

func (p *stateParseState) note(statement string, line int) error {
	left, text, found := strings.Cut(statement, ":")
	if !found {
		return unknown(KindState, line, statement)
	}
	fields := strings.Fields(left)
	if len(fields) < 4 {
		return unknown(KindState, line, statement)
	}
	position := strings.ToLower(fields[1])
	var anchorText string
	switch position {
	case "over":
		anchorText = strings.TrimSpace(left[len("note over"):])
	case "left", "right":
		if !strings.EqualFold(fields[2], "of") {
			return unknown(KindState, line, statement)
		}
		anchorText = strings.TrimSpace(left[len("note")+len(fields[1])+len(fields[2])+2:])
	default:
		return unknown(KindState, line, statement)
	}
	anchors := sequenceNoteIDs(anchorText)
	if len(anchors) != 1 || !stateValidID(anchors[0]) {
		return unknown(KindState, line, statement)
	}
	anchor, err := p.ensureNode(anchors[0], line)
	if err != nil {
		return err
	}
	p.data.Notes = append(p.data.Notes, StateNote{Anchor: anchor, Position: position, Text: unquote(strings.TrimSpace(text))})
	return nil
}

func stateData(d Diagram) (StateData, error) {
	data, ok := d.Data.(StateData)
	if !ok || len(data.Nodes) == 0 {
		return StateData{}, unknown(KindState, 0, "invalid state payload")
	}
	nodes := make(map[string]bool, len(data.Nodes))
	groups := make(map[string]bool, len(data.Groups))
	for _, group := range data.Groups {
		if !stateValidID(group.ID) || groups[group.ID] {
			return StateData{}, unknown(KindState, 0, "invalid composite state")
		}
		groups[group.ID] = true
	}
	for _, node := range data.Nodes {
		if !stateValidID(node.ID) || nodes[node.ID] || node.Kind < StateNormal || node.Kind > StateJoin {
			return StateData{}, unknown(KindState, 0, "invalid state node")
		}
		if node.Parent != "" && !groups[node.Parent] {
			return StateData{}, unknown(KindState, 0, "invalid composite parent")
		}
		nodes[node.ID] = true
	}
	for _, group := range data.Groups {
		if group.Parent != "" && !groups[group.Parent] || !nodes[group.ID] {
			return StateData{}, unknown(KindState, 0, "invalid composite state")
		}
	}
	for _, transition := range data.Transitions {
		if !nodes[transition.From] || !nodes[transition.To] {
			return StateData{}, unknown(KindState, 0, "invalid state transition")
		}
	}
	for _, note := range data.Notes {
		if !nodes[note.Anchor] || note.Position != "left" && note.Position != "right" && note.Position != "over" {
			return StateData{}, unknown(KindState, 0, "invalid state note")
		}
	}
	return data, nil
}

type stateLayoutItem struct {
	id          string
	label       string
	description string
	kind        StateNodeKind
	x, y, w, h  int
	lines       []string
	lineHeight  int
	child       *Scene
	colorIndex  int
}

type stateLayoutEdge struct {
	edge       Edge
	back       bool
	lines      []string
	width      int
	height     int
	lineHeight int
}

type stateSizedNote struct {
	note       StateNote
	lines      []string
	w, h       int
	lineHeight int
}

type stateLayoutBuilder struct {
	data        StateData
	fonts       *rich.Fonts
	palette     rich.Palette
	nodeIndex   map[string]int
	groupIndex  map[string]int
	levelEdges  map[string][]stateLayoutEdge
	groupScenes map[string]*Scene
}

func layoutState(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, err := stateData(d)
	if err != nil {
		return Scene{}, err
	}
	if fonts == nil {
		return Scene{}, unknown(KindState, 0, "font metrics")
	}
	builder := &stateLayoutBuilder{
		data: data, fonts: fonts, palette: palette,
		nodeIndex:   make(map[string]int, len(data.Nodes)),
		groupIndex:  make(map[string]int, len(data.Groups)),
		levelEdges:  make(map[string][]stateLayoutEdge),
		groupScenes: make(map[string]*Scene),
	}
	for i, node := range data.Nodes {
		builder.nodeIndex[node.ID] = i
	}
	for i, group := range data.Groups {
		builder.groupIndex[group.ID] = i
	}
	builder.assignTransitions()
	root, err := builder.buildLevel("", strings.ToUpper(d.Direction))
	if err != nil {
		return Scene{}, err
	}
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	s := newScene()
	s.Width = max(root.Width, titleWidth+2)
	s.Height = root.Height + titleHeight + 1
	stateAppendTranslated(&s, root, 0, titleHeight+1)
	if titleHeight > 0 {
		s.TextLines(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	return s, nil
}

func (b *stateLayoutBuilder) assignTransitions() {
	for _, transition := range b.data.Transitions {
		from := b.data.Nodes[b.nodeIndex[transition.From]]
		to := b.data.Nodes[b.nodeIndex[transition.To]]
		scope := stateCommonScope(from.Parent, to.Parent, b.groupIndex, b.data.Groups)
		edge := stateLayoutEdge{edge: Edge{From: stateBubble(transition.From, scope, b.nodeIndex, b.data, b.groupIndex), To: stateBubble(transition.To, scope, b.nodeIndex, b.data, b.groupIndex), Label: transition.Label, Style: EdgeSolid, End: ArrowOpen}}
		b.levelEdges[scope] = append(b.levelEdges[scope], edge)
	}
}

func stateCommonScope(a, b string, groups map[string]int, records []StateGroup) string {
	for {
		if stateScopeContains(a, b, groups, records) {
			return a
		}
		if a == "" {
			return ""
		}
		a = records[groups[a]].Parent
	}
}

func stateScopeContains(scope, descendant string, groups map[string]int, records []StateGroup) bool {
	for {
		if scope == descendant {
			return true
		}
		if descendant == "" {
			return false
		}
		descendant = records[groups[descendant]].Parent
	}
}

func stateBubble(id, scope string, nodes map[string]int, data StateData, groups map[string]int) string {
	owner := data.Nodes[nodes[id]].Parent
	for owner != scope {
		id = owner
		owner = data.Groups[groups[owner]].Parent
	}
	return id
}

func (b *stateLayoutBuilder) buildGroup(index int, inherited string) (*Scene, error) {
	group := b.data.Groups[index]
	if scene, ok := b.groupScenes[group.ID]; ok {
		return scene, nil
	}
	direction := inherited
	if group.Direction != "" {
		direction = strings.ToUpper(group.Direction)
	}
	scene, err := b.buildLevel(group.ID, direction)
	if err != nil {
		return nil, err
	}
	stored := scene
	b.groupScenes[group.ID] = &stored
	return &stored, nil
}

func (b *stateLayoutBuilder) buildLevel(scope, direction string) (Scene, error) {
	if direction == "" {
		direction = "TD"
	}
	horizontal := direction == "LR"
	items := make([]*stateLayoutItem, 0)
	for i, node := range b.data.Nodes {
		if node.Parent != scope {
			continue
		}
		if _, composite := b.groupIndex[node.ID]; composite {
			continue
		}
		item, err := b.newNodeItem(node, i)
		if err != nil {
			return Scene{}, err
		}
		items = append(items, item)
	}
	for i, group := range b.data.Groups {
		if group.Parent != scope {
			continue
		}
		child, err := b.buildGroup(i, direction)
		if err != nil {
			return Scene{}, err
		}
		lines, width, lineHeight := labelSize(group.Label, b.fonts, max(child.Width-2, 12), b.fonts.BodyPx, false)
		headerHeight := len(lines)*lineHeight + 2
		w := max(child.Width+4, width+4, 10)
		h := child.Height + headerHeight + 2
		items = append(items, &stateLayoutItem{id: group.ID, label: group.Label, kind: StateNormal, w: w, h: h, lines: lines, lineHeight: lineHeight, child: child, colorIndex: i})
	}
	if len(items) == 0 {
		return Scene{Width: 4, Height: 4}, nil
	}
	index := make(map[string]int, len(items))
	for i, item := range items {
		index[item.id] = i
	}
	edges := make([]stateLayoutEdge, len(b.levelEdges[scope]))
	copy(edges, b.levelEdges[scope])
	for i := range edges {
		if _, ok := index[edges[i].edge.From]; !ok {
			return Scene{}, unknown(KindState, 0, "transition endpoint outside composite")
		}
		if _, ok := index[edges[i].edge.To]; !ok {
			return Scene{}, unknown(KindState, 0, "transition endpoint outside composite")
		}
		if edges[i].edge.Label != "" {
			edges[i].lines, edges[i].width, edges[i].lineHeight = labelSize(edges[i].edge.Label, b.fonts, 18, b.fonts.BodyPx, false)
			edges[i].width = max(edges[i].width+2, 3)
			edges[i].height = len(edges[i].lines) * edges[i].lineHeight
		}
	}
	layers, back := stateRankItems(items, edges)
	layerOf := make([]int, len(items))
	for layer, ids := range layers {
		for _, id := range ids {
			layerOf[id] = layer
		}
	}
	notes := b.scopeNotes(scope, b.fonts)
	leftGutter, rightGutter := 0, 0
	for _, note := range notes {
		if note.note.Position == "left" || note.note.Position == "over" {
			leftGutter = max(leftGutter, note.w+3)
		}
		if note.note.Position == "right" || note.note.Position == "over" {
			rightGutter = max(rightGutter, note.w+3)
		}
	}
	externalLanes, laneStep := stateExternalLanes(edges, back, layerOf, index)
	rightExtra := externalLanes * laneStep
	gaps, crossSize := stateLayerMeasurements(items, layers, edges, back, layerOf, index, horizontal)
	contentLeft := leftGutter + 1
	contentWidth, contentHeight := crossSize, crossSize
	if horizontal {
		contentWidth = statePrimarySize(layers, items, index, gaps, true)
		contentHeight = crossSize
	} else {
		contentWidth = crossSize
		contentHeight = statePrimarySize(layers, items, index, gaps, false)
	}
	statePlaceItems(items, layers, gaps, horizontal, contentLeft, contentWidth, contentHeight)
	s := newScene()
	s.Width = contentLeft + contentWidth + rightExtra + rightGutter + 2
	s.Height = contentHeight + 4
	stroke := b.palette.Link
	if stroke.A == 0 {
		stroke = rule(b.palette)
	}
	forwardUsed := make([]int, max(len(gaps), 1))
	externalUsed := 0
	for i, edge := range edges {
		if back[i] {
			stateDrawExternalEdge(&s, items, index, edge, contentLeft+contentWidth+2+externalUsed*laneStep, b.fonts, b.palette)
			externalUsed++
			continue
		}
		fromLayer, toLayer := layerOf[index[edge.edge.From]], layerOf[index[edge.edge.To]]
		if fromLayer == toLayer {
			stateDrawExternalEdge(&s, items, index, edge, contentLeft+contentWidth+2+externalUsed*laneStep, b.fonts, b.palette)
			externalUsed++
			continue
		}
		channel, offset := 0, 0
		if edge.edge.Label != "" {
			offset = forwardUsed[fromLayer]
		}
		if horizontal {
			channel = stateColumnEnd(layers, items, index, gaps, fromLayer, contentLeft) + 1 + offset
			if edge.edge.Label != "" {
				forwardUsed[fromLayer] += edge.width + 2
			}
		} else {
			channel = stateRowEnd(layers, items, index, gaps, fromLayer) + 1 + offset
			if edge.edge.Label != "" {
				forwardUsed[fromLayer] += edge.height + 1
			}
		}
		stateDrawForwardEdge(&s, items, index, edge, channel, horizontal, stroke, b.fonts, b.palette)
	}
	b.drawNotes(&s, notes, items, index, contentLeft+contentWidth+rightExtra+2, b.fonts)
	for _, item := range items {
		b.drawItem(&s, item)
	}
	return s, nil
}

func (b *stateLayoutBuilder) newNodeItem(node StateNode, colorIndex int) (*stateLayoutItem, error) {
	item := &stateLayoutItem{id: node.ID, label: node.Label, description: node.Description, kind: node.Kind, colorIndex: colorIndex}
	switch node.Kind {
	case StateStart:
		item.w, item.h = 4, 2
	case StateEnd:
		item.w, item.h = 4, 2
	case StateChoice:
		lines, width, lineHeight := labelSize(node.Label, b.fonts, 14, b.fonts.BodyPx, false)
		item.lines, item.lineHeight = lines, lineHeight
		item.w, item.h = max(width*2+2, 10), max(len(lines)*lineHeight+2, 5)
	case StateFork, StateJoin:
		lines, width, lineHeight := labelSize(node.Label, b.fonts, 16, b.fonts.BodyPx, false)
		item.lines, item.lineHeight = lines, lineHeight
		item.w, item.h = max(width+4, 9), max(len(lines)*lineHeight+2, 3)
	default:
		labelLines, labelWidth, lineHeight := labelSize(node.Label, b.fonts, 18, b.fonts.BodyPx, false)
		item.lines = append(item.lines, labelLines...)
		item.lineHeight = lineHeight
		width := labelWidth
		if node.Description != "" {
			descriptionLines, descriptionWidth, _ := labelSize(node.Description, b.fonts, 18, b.fonts.BodyPx, false)
			item.lines = append(item.lines, descriptionLines...)
			width = max(width, descriptionWidth)
		}
		item.w, item.h = max(width+4, 10), max(len(item.lines)*lineHeight+2, 3)
	}
	return item, nil
}

func (b *stateLayoutBuilder) scopeNotes(scope string, fonts *rich.Fonts) []stateSizedNote {
	var notes []stateSizedNote
	for _, note := range b.data.Notes {
		node := b.data.Nodes[b.nodeIndex[note.Anchor]]
		if node.Parent != scope {
			continue
		}
		lines, width, lineHeight := labelSize(note.Text, fonts, 18, fonts.BodyPx, false)
		notes = append(notes, stateSizedNote{note: note, lines: lines, w: max(width+4, 8), h: max(len(lines)*lineHeight+2, 3), lineHeight: lineHeight})
	}
	return notes
}

func stateRankItems(items []*stateLayoutItem, edges []stateLayoutEdge) ([][]int, []bool) {
	index := make(map[string]int, len(items))
	for i, item := range items {
		index[item.id] = i
	}
	out := make([][]int, len(items))
	targets := make([]int, len(edges))
	for i, edge := range edges {
		from, to := index[edge.edge.From], index[edge.edge.To]
		out[from] = append(out[from], i)
		targets[i] = to
	}
	state := make([]uint8, len(items))
	back := make([]bool, len(edges))
	var post []int
	var visit func(int)
	visit = func(node int) {
		state[node] = 1
		for _, edgeIndex := range out[node] {
			target := targets[edgeIndex]
			if state[target] == 0 {
				visit(target)
			} else if state[target] == 1 {
				back[edgeIndex] = true
			}
		}
		state[node] = 2
		post = append(post, node)
	}
	for i := range items {
		if state[i] == 0 {
			visit(i)
		}
	}
	layers := make([]int, len(items))
	for i := len(post) - 1; i >= 0; i-- {
		node := post[i]
		for _, edgeIndex := range out[node] {
			if back[edgeIndex] {
				continue
			}
			target := targets[edgeIndex]
			layers[target] = max(layers[target], layers[node]+1)
		}
	}
	depth := 1
	for _, layer := range layers {
		depth = max(depth, layer+1)
	}
	result := make([][]int, depth)
	for i, layer := range layers {
		result[layer] = append(result[layer], i)
	}
	return result, back
}

func stateExternalLanes(edges []stateLayoutEdge, back []bool, layerOf []int, index map[string]int) (int, int) {
	count, step := 0, 4
	for i, edge := range edges {
		if back[i] || layerOf[index[edge.edge.From]] == layerOf[index[edge.edge.To]] {
			count++
			step = max(step, edge.width+3)
		}
	}
	return count, step
}

func stateLayerMeasurements(items []*stateLayoutItem, layers [][]int, edges []stateLayoutEdge, back []bool, layerOf []int, index map[string]int, horizontal bool) ([]int, int) {
	gaps := make([]int, max(len(layers)-1, 0))
	for i := range gaps {
		gaps[i] = 2
	}
	for i, edge := range edges {
		if back[i] || edge.edge.Label == "" {
			continue
		}
		from, to := layerOf[index[edge.edge.From]], layerOf[index[edge.edge.To]]
		if from >= to {
			continue
		}
		if horizontal {
			gaps[from] += edge.width + 2
		} else {
			gaps[from] += edge.height + 1
		}
	}
	cross := 0
	for _, layer := range layers {
		total := 0
		for _, id := range layer {
			if horizontal {
				total += items[id].h
			} else {
				total += items[id].w
			}
		}
		if len(layer) > 1 {
			total += (len(layer) - 1) * 4
		}
		cross = max(cross, total)
	}
	return gaps, max(cross, 4)
}

func statePrimarySize(layers [][]int, items []*stateLayoutItem, index map[string]int, gaps []int, horizontal bool) int {
	total := 0
	for layerIndex, layer := range layers {
		size := 0
		for _, id := range layer {
			if horizontal {
				size = max(size, items[id].w)
			} else {
				size = max(size, items[id].h)
			}
		}
		total += size
		if layerIndex < len(gaps) {
			total += gaps[layerIndex]
		}
	}
	_ = index
	return max(total, 4)
}

func statePlaceItems(items []*stateLayoutItem, layers [][]int, gaps []int, horizontal bool, left, width, height int) {
	if horizontal {
		x := left
		for layerIndex, layer := range layers {
			columnHeight := 0
			for _, id := range layer {
				columnHeight += items[id].h
			}
			if len(layer) > 1 {
				columnHeight += (len(layer) - 1) * 4
			}
			y := 2 + (height-columnHeight)/2
			for _, id := range layer {
				item := items[id]
				item.x, item.y = x, y
				y += item.h + 4
			}
			x += stateLayerPrimary(items, layer, true)
			if layerIndex < len(gaps) {
				x += gaps[layerIndex]
			}
		}
		return
	}
	y := 2
	for layerIndex, layer := range layers {
		rowWidth := 0
		for _, id := range layer {
			rowWidth += items[id].w
		}
		if len(layer) > 1 {
			rowWidth += (len(layer) - 1) * 4
		}
		x := left + (width-rowWidth)/2
		for _, id := range layer {
			item := items[id]
			item.x, item.y = x, y
			x += item.w + 4
		}
		y += stateLayerPrimary(items, layer, false)
		if layerIndex < len(gaps) {
			y += gaps[layerIndex]
		}
	}
}

func stateLayerPrimary(items []*stateLayoutItem, layer []int, horizontal bool) int {
	result := 0
	for _, id := range layer {
		if horizontal {
			result = max(result, items[id].w)
		} else {
			result = max(result, items[id].h)
		}
	}
	return result
}

func stateRowEnd(layers [][]int, items []*stateLayoutItem, index map[string]int, gaps []int, layer int) int {
	result := 2
	for i := 0; i < layer; i++ {
		result += stateLayerPrimary(items, layers[i], false)
		if i < len(gaps) {
			result += gaps[i]
		}
	}
	_ = index
	return result + stateLayerPrimary(items, layers[layer], false) - 1
}

func stateColumnEnd(layers [][]int, items []*stateLayoutItem, index map[string]int, gaps []int, layer, left int) int {
	result := left
	for i := 0; i < layer; i++ {
		result += stateLayerPrimary(items, layers[i], true)
		if i < len(gaps) {
			result += gaps[i]
		}
	}
	_ = index
	return result + stateLayerPrimary(items, layers[layer], true) - 1
}

func stateDrawForwardEdge(s *Scene, items []*stateLayoutItem, index map[string]int, edge stateLayoutEdge, channel int, horizontal bool, stroke color.RGBA, fonts *rich.Fonts, palette rich.Palette) {
	from, to := items[index[edge.edge.From]], items[index[edge.edge.To]]
	var path []Point
	if horizontal {
		start := Point{X: from.x + from.w, Y: from.cy()}
		end := Point{X: to.x, Y: to.cy()}
		if start.Y == end.Y {
			path = []Point{start, end}
		} else {
			path = []Point{start, {X: channel, Y: start.Y}, {X: channel, Y: end.Y}, end}
		}
	} else {
		start := Point{X: from.cx(), Y: from.y + from.h}
		end := Point{X: to.cx(), Y: to.y}
		if start.X == end.X {
			path = []Point{start, end}
		} else {
			path = []Point{start, {X: start.X, Y: channel}, {X: end.X, Y: channel}, end}
		}
	}
	flowDrawPath(s, path, edge.edge, stroke)
	if edge.edge.Label == "" {
		return
	}
	if horizontal {
		x := (from.x+from.w+to.x)/2 - edge.width/2
		y := (from.cy()+to.cy())/2 - edge.height - 1
		if y < 0 {
			y = (from.cy()+to.cy())/2 + 1
		}
		flowDrawLabel(s, Point{X: max(x, 1), Y: y}, edge.lines, edge.width, edge.height, edge.lineHeight, fonts, palette)
		return
	}
	x := (from.cx()+to.cx())/2 - edge.width/2
	y := (from.y+from.h+to.y)/2 - edge.height/2
	flowDrawLabel(s, Point{X: max(x, 1), Y: max(y, 0)}, edge.lines, edge.width, edge.height, edge.lineHeight, fonts, palette)
}

func stateDrawExternalEdge(s *Scene, items []*stateLayoutItem, index map[string]int, edge stateLayoutEdge, lane int, fonts *rich.Fonts, palette rich.Palette) {
	from, to := items[index[edge.edge.From]], items[index[edge.edge.To]]
	start := Point{X: from.x + from.w, Y: from.cy()}
	end := Point{X: to.x + to.w, Y: to.cy()}
	path := []Point{start, {X: lane, Y: start.Y}, {X: lane, Y: end.Y}, end}
	if from == to {
		path = []Point{start, {X: lane, Y: start.Y}, {X: lane, Y: start.Y + 2}, {X: start.X + 1, Y: start.Y + 2}, {X: start.X, Y: start.Y + 1}}
	}
	stroke := palette.Link
	if stroke.A == 0 {
		stroke = rule(palette)
	}
	flowDrawPath(s, path, edge.edge, stroke)
	if edge.edge.Label != "" {
		y := (from.cy()+to.cy())/2 - edge.height/2
		flowDrawLabel(s, Point{X: lane + 1, Y: y}, edge.lines, edge.width, edge.height, edge.lineHeight, fonts, palette)
	}
}

func (b *stateLayoutBuilder) drawNotes(s *Scene, notes []stateSizedNote, items []*stateLayoutItem, index map[string]int, rightX int, fonts *rich.Fonts) {
	leftNotes, rightNotes := []stateSizedNote{}, []stateSizedNote{}
	for _, note := range notes {
		if note.note.Position == "right" {
			rightNotes = append(rightNotes, note)
		} else {
			leftNotes = append(leftNotes, note)
		}
	}
	sort.SliceStable(leftNotes, func(i, j int) bool {
		return items[index[leftNotes[i].note.Anchor]].cy() < items[index[leftNotes[j].note.Anchor]].cy()
	})
	sort.SliceStable(rightNotes, func(i, j int) bool {
		return items[index[rightNotes[i].note.Anchor]].cy() < items[index[rightNotes[j].note.Anchor]].cy()
	})
	draw := func(list []stateSizedNote, left bool) {
		y := 2
		for i, note := range list {
			item := items[index[note.note.Anchor]]
			y = max(y, item.cy()-note.h/2)
			x := 1
			if !left {
				x = rightX
			}
			col := accent(b.palette, i+2)
			s.Rounded(x, y, note.w, note.h, 1, fade(col, 24), col)
			s.TextLines(note.lines, x+2, y+1, note.w-4, note.lineHeight, fonts.BodyPx, AlignCenter, foreground(b.palette), false)
			if left {
				s.Line(x+note.w, item.cy(), item.x, item.cy(), muted(b.palette), true, false)
			} else {
				s.Line(item.x+item.w, item.cy(), x, item.cy(), muted(b.palette), true, false)
			}
			y += note.h + 1
			s.Height = max(s.Height, y+1)
		}
	}
	draw(leftNotes, true)
	draw(rightNotes, false)
}

func (b *stateLayoutBuilder) drawItem(s *Scene, item *stateLayoutItem) {
	col := accent(b.palette, item.colorIndex)
	if item.child != nil {
		headerHeight := len(item.lines)*item.lineHeight + 2
		s.Rounded(item.x, item.y, item.w, item.h, 1, color.RGBA{}, col)
		s.TextLines(item.lines, item.x+2, item.y+1, item.w-4, item.lineHeight, b.fonts.BodyPx, AlignLeft, col, false)
		childX := item.x + (item.w-item.child.Width)/2
		stateAppendTranslated(s, *item.child, childX, item.y+headerHeight)
		return
	}
	switch item.kind {
	case StateStart:
		s.Circle(item.x, item.y, item.w, item.h, col, col)
	case StateEnd:
		s.Circle(item.x, item.y, item.w, item.h, b.palette.CodeBg, col)
		s.Dot(item.x+item.w/2, item.y+item.h/2, 6, col, col)
	case StateChoice:
		s.Diamond(item.x, item.y, item.w, item.h, b.palette.CodeBg, col)
		pad := max(item.w/3, 2)
		s.TextLines(item.lines, item.x+pad, item.y+(item.h-len(item.lines)*item.lineHeight)/2, item.w-2*pad, item.lineHeight, b.fonts.BodyPx, AlignCenter, foreground(b.palette), false)
	case StateFork, StateJoin:
		s.Rect(item.x, item.y, item.w, item.h, b.palette.CodeBg, col)
		s.Line(item.x+2, item.y+item.h/2, item.x+item.w-3, item.y+item.h/2, col, false, true)
		if len(item.lines) > 0 {
			s.TextLines(item.lines, item.x+2, item.y, item.w-4, item.lineHeight, b.fonts.BodyPx, AlignCenter, foreground(b.palette), false)
		}
	default:
		s.Rounded(item.x, item.y, item.w, item.h, 1, b.palette.CodeBg, col)
		s.TextLines(item.lines, item.x+2, item.y+(item.h-len(item.lines)*item.lineHeight)/2, item.w-4, item.lineHeight, b.fonts.BodyPx, AlignCenter, foreground(b.palette), false)
	}
}

func stateAppendTranslated(dst *Scene, src Scene, x, y int) {
	for _, command := range src.Commands {
		command.X += x
		command.Y += y
		command.X2 += x
		command.Y2 += y
		command.Points = append([]Point(nil), command.Points...)
		for i := range command.Points {
			command.Points[i].X += x
			command.Points[i].Y += y
		}
		dst.Commands = append(dst.Commands, command)
	}
	dst.Width = max(dst.Width, x+src.Width)
	dst.Height = max(dst.Height, y+src.Height)
}

func (item *stateLayoutItem) cx() int { return item.x + item.w/2 }
func (item *stateLayoutItem) cy() int { return item.y + item.h/2 }
