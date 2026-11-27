package diagram

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// SequenceEventKind identifies an ordered sequence-diagram event.
type SequenceEventKind int

const (
	// SequenceMessage is a labelled or unlabelled message between participants.
	SequenceMessage SequenceEventKind = iota
	// SequenceNoteEvent is a note anchored to one or more participants.
	SequenceNoteEvent
	// SequenceActivate starts a participant activation bar.
	SequenceActivate
	// SequenceDeactivate ends a participant activation bar.
	SequenceDeactivate
	// SequenceFrameOpen starts a loop, alternative, option, parallel, or rectangle frame.
	SequenceFrameOpen
	// SequenceFrameElse starts another branch of an alternative or parallel frame.
	SequenceFrameElse
	// SequenceFrameEnd closes a frame.
	SequenceFrameEnd
)

// SequenceParticipant is one actor or object lifeline.
type SequenceParticipant struct {
	ID    string
	Label string
	Actor bool
	Group string
}

// SequenceGroup is a box containing participant declarations.
type SequenceGroup struct {
	ID     string
	Label  string
	Parent string
}

// SequenceFrame is a nested control-flow frame.
type SequenceFrame struct {
	ID     string
	Kind   string
	Label  string
	Parent string
}

// SequenceNote is a note and its participant-relative position.
type SequenceNote struct {
	Position     string
	Participants []string
	Text         string
}

// SequenceEvent preserves source order for messages, notes, activations, and frames.
type SequenceEvent struct {
	Kind        SequenceEventKind
	From        string
	To          string
	Participant string
	Label       string
	Style       EdgeStyle
	Start       Arrowhead
	End         Arrowhead
	Activate    string
	Deactivate  string
	Note        SequenceNote
	Frame       string
}

// SequenceData is the semantic payload of a sequence diagram.
type SequenceData struct {
	Participants   []SequenceParticipant
	Groups         []SequenceGroup
	Frames         []SequenceFrame
	Events         []SequenceEvent
	Autonumber     bool
	AutonumberStart int
	AutonumberStep  int
}

type sequenceContainer struct {
	kind  string
	index int
	line  int
}

type sequenceParseState struct {
	diagram      Diagram
	data         SequenceData
	participants map[string]int
	containers   []sequenceContainer
	autonumber   bool
}

func init() {
	register(KindSequence, parseSequence, layoutSequence)
}

func parseSequence(lines []string) (Diagram, error) {
	p := sequenceParseState{
		diagram:      Diagram{Kind: KindSequence},
		participants: make(map[string]int),
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
	if len(p.containers) != 0 {
		construct := "unclosed frame"
		if p.containers[len(p.containers)-1].kind == "box" {
			construct = "unclosed box"
		}
		return Diagram{}, unknown(KindSequence, len(lines)+1, construct)
	}
	p.data.Autonumber = p.autonumber
	p.diagram.Data = p.data
	return p.diagram, nil
}

func (p *sequenceParseState) statement(statement string, line int) error {
	lower := strings.ToLower(statement)
	switch {
	case sequenceHasKeyword(lower, "participant"):
		return p.participant(statement[len("participant"):], false, line)
	case sequenceHasKeyword(lower, "actor"):
		return p.participant(statement[len("actor"):], true, line)
	case sequenceHasKeyword(lower, "box"):
		return p.openBox(unquote(strings.TrimSpace(statement[len("box"):])), line)
	case sequenceHasKeyword(lower, "note"):
		return p.note(statement, line)
	case lower == "activate" || sequenceHasKeyword(lower, "activate"):
		return p.activation(strings.TrimSpace(statement[len("activate"):]), true, line)
	case lower == "deactivate" || sequenceHasKeyword(lower, "deactivate"):
		return p.activation(strings.TrimSpace(statement[len("deactivate"):]), false, line)
	case lower == "autonumber" || sequenceHasKeyword(lower, "autonumber"):
		return p.autonumberStatement(statement, line)
	case sequenceHasKeyword(lower, "title"):
		if p.diagram.Title != "" {
			return unknown(KindSequence, line, statement)
		}
		title := unquote(strings.TrimSpace(statement[len("title"):]))
		if title == "" {
			return unknown(KindSequence, line, statement)
		}
		p.diagram.Title = title
		return nil
	case lower == "end":
		return p.closeContainer(line)
	case sequenceHasKeyword(lower, "else"):
		return p.branch("alt", strings.TrimSpace(statement[len("else"):]), line)
	case sequenceHasKeyword(lower, "and"):
		return p.branch("par", strings.TrimSpace(statement[len("and"):]), line)
	case sequenceHasKeyword(lower, "loop"),
		sequenceHasKeyword(lower, "alt"),
		sequenceHasKeyword(lower, "opt"),
		sequenceHasKeyword(lower, "par"),
		sequenceHasKeyword(lower, "rect"):
		keyword := strings.Fields(lower)[0]
		return p.openFrame(keyword, unquote(strings.TrimSpace(statement[len(keyword):])), line)
	default:
		return p.message(statement, line)
	}
}

func sequenceHasKeyword(text, keyword string) bool {
	return text == keyword || strings.HasPrefix(text, keyword+" ") || strings.HasPrefix(text, keyword+"\t")
}

func (p *sequenceParseState) participant(rest string, actor bool, line int) error {
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) != 1 && (len(fields) != 3 || !strings.EqualFold(fields[1], "as")) {
		return unknown(KindSequence, line, "participant "+rest)
	}
	id := fields[0]
	if !sequenceValidID(id) {
		return unknown(KindSequence, line, "participant "+rest)
	}
	label := id
	if len(fields) == 3 {
		tail := strings.TrimSpace(rest[len(fields[0]):])
		tail = strings.TrimSpace(tail[len(fields[1]):])
		label = unquote(tail)
		if label == "" {
			return unknown(KindSequence, line, "participant "+rest)
		}
	}
	p.ensureParticipant(id, label, actor, true)
	return nil
}

func (p *sequenceParseState) ensureParticipant(id, label string, actor, declared bool) int {
	if index, exists := p.participants[id]; exists {
		participant := &p.data.Participants[index]
		if declared {
			participant.Label = label
			participant.Actor = actor
			if group := p.currentGroup(); group != "" && participant.Group == "" {
				participant.Group = group
			}
		}
		return index
	}
	p.data.Participants = append(p.data.Participants, SequenceParticipant{
		ID: id, Label: label, Actor: actor, Group: p.currentGroup(),
	})
	index := len(p.data.Participants) - 1
	p.participants[id] = index
	return index
}

func (p *sequenceParseState) currentGroup() string {
	for i := len(p.containers) - 1; i >= 0; i-- {
		if p.containers[i].kind == "box" {
			return p.data.Groups[p.containers[i].index].ID
		}
	}
	return ""
}

func (p *sequenceParseState) openBox(label string, line int) error {
	if label == "" {
		return unknown(KindSequence, line, "box")
	}
	parent := p.currentGroup()
	id := "box_" + strconv.Itoa(len(p.data.Groups)+1)
	p.data.Groups = append(p.data.Groups, SequenceGroup{ID: id, Label: label, Parent: parent})
	p.containers = append(p.containers, sequenceContainer{kind: "box", index: len(p.data.Groups) - 1, line: line})
	return nil
}

func (p *sequenceParseState) openFrame(kind, label string, line int) error {
	if kind == "rect" && label == "" {
		return unknown(KindSequence, line, "rect")
	}
	parent := ""
	for i := len(p.containers) - 1; i >= 0; i-- {
		if p.containers[i].kind == "frame" {
			parent = p.data.Frames[p.containers[i].index].ID
			break
		}
	}
	id := "frame_" + strconv.Itoa(len(p.data.Frames)+1)
	p.data.Frames = append(p.data.Frames, SequenceFrame{ID: id, Kind: kind, Label: label, Parent: parent})
	p.data.Events = append(p.data.Events, SequenceEvent{Kind: SequenceFrameOpen, Frame: id})
	p.containers = append(p.containers, sequenceContainer{kind: "frame", index: len(p.data.Frames) - 1, line: line})
	return nil
}

func (p *sequenceParseState) branch(wantKind, label string, line int) error {
	if len(p.containers) == 0 {
		return unknown(KindSequence, line, "branch without frame")
	}
	container := p.containers[len(p.containers)-1]
	if container.kind != "frame" {
		return unknown(KindSequence, line, "branch inside box")
	}
	frame := p.data.Frames[container.index]
	if frame.Kind != wantKind {
		return unknown(KindSequence, line, frame.Kind+" branch")
	}
	p.data.Events = append(p.data.Events, SequenceEvent{Kind: SequenceFrameElse, Frame: frame.ID, Label: label})
	return nil
}

func (p *sequenceParseState) closeContainer(line int) error {
	if len(p.containers) == 0 {
		return unknown(KindSequence, line, "end")
	}
	container := p.containers[len(p.containers)-1]
	p.containers = p.containers[:len(p.containers)-1]
	if container.kind == "frame" {
		p.data.Events = append(p.data.Events, SequenceEvent{Kind: SequenceFrameEnd, Frame: p.data.Frames[container.index].ID})
	}
	return nil
}

func (p *sequenceParseState) activation(rest string, activate bool, line int) error {
	id := strings.TrimSpace(rest)
	if !sequenceValidID(id) {
		return unknown(KindSequence, line, "activation")
	}
	p.ensureParticipant(id, id, false, false)
	kind := SequenceDeactivate
	if activate {
		kind = SequenceActivate
	}
	p.data.Events = append(p.data.Events, SequenceEvent{Kind: kind, Participant: id})
	return nil
}

func (p *sequenceParseState) autonumberStatement(statement string, line int) error {
	if p.autonumber {
		return unknown(KindSequence, line, statement)
	}
	p.autonumber = true
	p.data.AutonumberStart, p.data.AutonumberStep = 1, 1
	fields := strings.Fields(strings.TrimSpace(statement[len("autonumber"):]))
	if len(fields) > 2 {
		return unknown(KindSequence, line, statement)
	}
	values := []int{1, 1}
	for i, field := range fields {
		value, err := parseInt(field)
		if err != nil || value <= 0 {
			return unknown(KindSequence, line, statement)
		}
		values[i] = value
	}
	p.data.AutonumberStart, p.data.AutonumberStep = values[0], values[1]
	return nil
}

func (p *sequenceParseState) note(statement string, line int) error {
	left, text, found := strings.Cut(statement, ":")
	if !found {
		return unknown(KindSequence, line, statement)
	}
	fields := strings.Fields(left)
	if len(fields) < 3 {
		return unknown(KindSequence, line, statement)
	}
	position := strings.ToLower(fields[1])
	var participants []string
	switch position {
	case "over":
		participants = sequenceNoteIDs(strings.TrimSpace(left[len("note over"):]))
	case "left", "right":
		if len(fields) < 4 || !strings.EqualFold(fields[2], "of") {
			return unknown(KindSequence, line, statement)
		}
		participants = sequenceNoteIDs(strings.TrimSpace(left[len("note")+len(fields[1])+len(fields[2])+2:]))
	default:
		return unknown(KindSequence, line, statement)
	}
	if len(participants) == 0 {
		return unknown(KindSequence, line, statement)
	}
	for _, id := range participants {
		if !sequenceValidID(id) {
			return unknown(KindSequence, line, statement)
		}
		p.ensureParticipant(id, id, false, false)
	}
	note := SequenceNote{Position: position, Participants: participants, Text: unquote(strings.TrimSpace(text))}
	p.data.Events = append(p.data.Events, SequenceEvent{Kind: SequenceNoteEvent, Note: note})
	return nil
}

func sequenceNoteIDs(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
}

func (p *sequenceParseState) message(statement string, line int) error {
	arrow, style, start, end, index := sequenceArrow(statement)
	if index < 0 {
		return unknown(KindSequence, line, statement)
	}
	from := strings.TrimSpace(statement[:index])
	if !sequenceValidID(from) {
		return unknown(KindSequence, line, statement)
	}
	rest := strings.TrimLeft(statement[index+len(arrow):], " \t")
	activate, deactivate := "", ""
	if rest != "" && (rest[0] == '+' || rest[0] == '-') {
		if rest[0] == '+' {
			activate = "+"
		} else {
			deactivate = "-"
		}
		rest = strings.TrimLeft(rest[1:], " \t")
	}
	endpoint := 0
	for endpoint < len(rest) && rest[endpoint] != ':' && rest[endpoint] != ' ' && rest[endpoint] != '\t' {
		endpoint++
	}
	to := rest[:endpoint]
	if !sequenceValidID(to) {
		return unknown(KindSequence, line, statement)
	}
	remaining := strings.TrimSpace(rest[endpoint:])
	label := ""
	if remaining != "" {
		if remaining[0] != ':' {
			return unknown(KindSequence, line, statement)
		}
		label = unquote(strings.TrimSpace(remaining[1:]))
	}
	p.ensureParticipant(from, from, false, false)
	p.ensureParticipant(to, to, false, false)
	if activate != "" {
		activate = to
	}
	if deactivate != "" {
		deactivate = to
	}
	p.data.Events = append(p.data.Events, SequenceEvent{
		Kind: SequenceMessage, From: from, To: to, Label: label,
		Style: style, Start: start, End: end, Activate: activate, Deactivate: deactivate,
	})
	return nil
}

func sequenceValidID(id string) bool {
	if id == "" || strings.TrimSpace(id) != id {
		return false
	}
	return !strings.ContainsAny(id, " \t\r\n:<>-+;[]{}")
}

func sequenceArrow(statement string) (string, EdgeStyle, Arrowhead, Arrowhead, int) {
	type candidate struct {
		token       string
		style       EdgeStyle
		start, end  Arrowhead
	}
	candidates := []candidate{
		{token: "<<-->>", style: EdgeSolid, start: ArrowFilled, end: ArrowFilled},
		{token: "<<->>", style: EdgeSolid, start: ArrowFilled, end: ArrowFilled},
		{token: "<-->>", style: EdgeSolid, start: ArrowOpen, end: ArrowFilled},
		{token: "<-->", style: EdgeSolid, start: ArrowOpen, end: ArrowOpen},
		{token: "<->", style: EdgeSolid},
		{token: "-->>", style: EdgeSolid, end: ArrowFilled},
		{token: "-.->>", style: EdgeDotted, end: ArrowFilled},
		{token: "-.->", style: EdgeDotted, end: ArrowOpen},
		{token: "-.)", style: EdgeDotted, end: ArrowOpen},
		{token: "-.x", style: EdgeDotted, end: ArrowCross},
		{token: "-->", style: EdgeSolid, end: ArrowOpen},
		{token: "->>", style: EdgeSolid, end: ArrowFilled},
		{token: "--x", style: EdgeSolid, end: ArrowCross},
		{token: "--)", style: EdgeSolid, end: ArrowOpen},
		{token: "->", style: EdgeSolid},
		{token: "-x", style: EdgeSolid, end: ArrowCross},
	}
	best := -1
	var chosen candidate
	for _, item := range candidates {
		index := strings.Index(statement, item.token)
		if index >= 0 && (best < 0 || index < best || index == best && len(item.token) > len(chosen.token)) {
			best, chosen = index, item
		}
	}
	if best < 0 {
		return "", EdgeSolid, ArrowNone, ArrowNone, -1
	}
	return chosen.token, chosen.style, chosen.start, chosen.end, best
}

type sequenceParticipantLayout struct {
	participant SequenceParticipant
	x, y, w, h  int
	lineHeight  int
}

func (p *sequenceParticipantLayout) cx() int { return p.x + p.w/2 }

type sequenceNoteLayout struct {
	note             SequenceNote
	x, y, w, h       int
	lines            []string
	lineHeight       int
}

type sequenceMessageLayout struct {
	event     SequenceEvent
	y, arrowY int
	lines     []string
	width     int
	height    int
	lineHeight int
}

type sequenceFrameLayout struct {
	frame       SequenceFrame
	x, y, w, h  int
	headerLines []string
	lineHeight  int
	separators  []sequenceFrameSeparator
}

type sequenceFrameSeparator struct {
	y     int
	lines []string
}

type sequenceOpenFrame struct {
	index int
}

func layoutSequence(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, err := sequenceData(d)
	if err != nil {
		return Scene{}, err
	}
	if fonts == nil {
		return Scene{}, unknown(KindSequence, 0, "font metrics")
	}
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	s := newScene()
	s.Width = max(titleWidth+2, 8)
	s.Height = titleHeight + 4

	participants := make([]sequenceParticipantLayout, len(data.Participants))
	participantIndex := make(map[string]int, len(participants))
	x, maxHeight := 3, 3
	for i, participant := range data.Participants {
		lines, width, lineHeight := labelSize(participant.Label, fonts, 18, fonts.BodyPx, false)
		w := max(width+4, 10)
		lines, width, lineHeight = labelSize(participant.Label, fonts, max(w-4, 3), fonts.BodyPx, false)
		w = max(w, width+4)
		h := max(len(lines)*lineHeight+2, 3)
		participants[i] = sequenceParticipantLayout{participant: participant, x: x, w: w, h: h, lineHeight: lineHeight}
		participantIndex[participant.ID] = i
		x += w + 6
		maxHeight = max(maxHeight, h)
		_ = lines
	}
	if len(participants) > 0 {
		s.Width = max(s.Width, x-3)
	}

	noteSizes := sequenceNoteSizes(data, fonts)
	leftMost := s.Width
	for _, note := range noteSizes {
		anchor, ok := participantIndex[note.note.Participants[0]]
		if !ok {
			continue
		}
		cx := participants[anchor].cx()
		switch note.note.Position {
		case "left":
			leftMost = min(leftMost, cx-note.w-4)
		case "right":
			s.Width = max(s.Width, cx+4+note.w+2)
		case "over":
			left := cx
			for _, id := range note.note.Participants[1:] {
				if other, exists := participantIndex[id]; exists {
					left = min(left, participants[other].cx())
				}
			}
			right := cx
			for _, id := range note.note.Participants[1:] {
				if other, exists := participantIndex[id]; exists {
					right = max(right, participants[other].cx())
				}
			}
			noteX := (left+right)/2 - note.w/2
			leftMost = min(leftMost, noteX)
			s.Width = max(s.Width, noteX+note.w+2)
		}
	}
	if leftMost < 1 {
		shift := 1 - leftMost
		for i := range participants {
			participants[i].x += shift
		}
		s.Width += shift
	}

	groupMaxHeight := 1
	groupDepth := make([]int, len(data.Groups))
	groupIndex := make(map[string]int, len(data.Groups))
	for i, group := range data.Groups {
		groupIndex[group.ID] = i
		_, _, lineHeight := labelSize(group.Label, fonts, 24, fonts.BodyPx, false)
		groupMaxHeight = max(groupMaxHeight, lineHeight)
	}
	for i := range data.Groups {
		if parent, ok := groupIndex[data.Groups[i].Parent]; ok {
			groupDepth[i] = groupDepth[parent] + 1
		}
	}
	participantY := titleHeight + 1 + (maxSequenceDepth(groupDepth)+1)*groupMaxHeight + 1
	for i := range participants {
		participants[i].y = participantY
	}
	groupBoxes := sequenceGroupBoxes(data, participants, participantIndex, participantY+maxHeight, titleHeight+1, groupMaxHeight, fonts)

	if titleHeight > 0 {
		s.TextLines(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	for _, box := range groupBoxes {
		s.Rounded(box.X, box.Y, box.W, box.H, 1, color.RGBA{}, accent(palette, box.colorIndex))
		s.TextLines(box.lines, box.X+2, box.Y, box.W-4, box.lineHeight, fonts.BodyPx, AlignLeft, accent(palette, box.colorIndex), false)
	}

	y := participantY + maxHeight + 3
	frames := make([]sequenceFrameLayout, len(data.Frames))
	frameIndex := make(map[string]int, len(data.Frames))
	for i, frame := range data.Frames {
		frameIndex[frame.ID] = i
	}
	var stack []sequenceOpenFrame
	active := make(map[string]int)
	var notes []sequenceNoteLayout
	var messages []sequenceMessageLayout
	number := data.AutonumberStart
	if number <= 0 {
		number = 1
	}
	for _, event := range data.Events {
		switch event.Kind {
		case SequenceMessage:
			from, fromOK := participantIndex[event.From]
			to, toOK := participantIndex[event.To]
			if !fromOK || !toOK {
				return Scene{}, unknown(KindSequence, 0, "message endpoint")
			}
			_ = from
			_ = to
			label := event.Label
			if data.Autonumber {
				if label != "" {
					label = strconv.Itoa(number) + ". " + label
				} else {
					label = strconv.Itoa(number) + "."
				}
				number += data.AutonumberStep
			}
			lines, width, lineHeight := labelSize(label, fonts, 24, fonts.BodyPx, false)
			height := max(len(lines)*lineHeight+2, 3)
			messages = append(messages, sequenceMessageLayout{
				event: event, y: y, arrowY: y + len(lines)*lineHeight + 1,
				lines: lines, width: max(width+2, 3), height: height, lineHeight: lineHeight,
			})
			if event.Deactivate != "" {
				if start, ok := active[event.Deactivate]; ok {
					sequenceActivationBar(&s, participants[participantIndex[event.Deactivate]], start, y+1, palette)
					delete(active, event.Deactivate)
				}
			}
			if event.Activate != "" {
				active[event.Activate] = y + 1
			}
			y += height + 1
		case SequenceNoteEvent:
			note, ok := sequencePlaceNote(event.Note, noteSizes, participantIndex, participants, y, s.Width)
			if !ok {
				return Scene{}, unknown(KindSequence, 0, "note participant")
			}
			notes = append(notes, note)
			y += note.h + 2
		case SequenceActivate:
			if _, ok := participantIndex[event.Participant]; !ok {
				return Scene{}, unknown(KindSequence, 0, "activation participant")
			}
			active[event.Participant] = y
		case SequenceDeactivate:
			index, ok := participantIndex[event.Participant]
			if !ok {
				return Scene{}, unknown(KindSequence, 0, "activation participant")
			}
			if start, exists := active[event.Participant]; exists {
				sequenceActivationBar(&s, participants[index], start, y, palette)
				delete(active, event.Participant)
			}
		case SequenceFrameOpen:
			index, ok := frameIndex[event.Frame]
			if !ok {
				return Scene{}, unknown(KindSequence, 0, "frame")
			}
			frame := data.Frames[index]
			label := frame.Kind
			if frame.Label != "" {
				label += " " + frame.Label
			}
			lines, width, lineHeight := labelSize(label, fonts, max(s.Width-8, 12), fonts.BodyPx, false)
			depth := len(stack)
			frameX := min(2+depth*2, max(1, s.Width/2))
			frames[index] = sequenceFrameLayout{
				frame: frame, x: frameX, y: y, w: max(s.Width-frameX-3, 3),
				headerLines: lines, lineHeight: lineHeight,
			}
			_ = width
			stack = append(stack, sequenceOpenFrame{index: index})
			y += len(lines)*lineHeight + 2
		case SequenceFrameElse:
			index, ok := frameIndex[event.Frame]
			if !ok || len(stack) == 0 || stack[len(stack)-1].index != index {
				return Scene{}, unknown(KindSequence, 0, "frame branch")
			}
			label := "else"
			if data.Frames[index].Kind == "par" {
				label = "and"
			}
			if event.Label != "" {
				label += " " + event.Label
			}
			lines, _, lineHeight := labelSize(label, fonts, max(s.Width-8, 12), fonts.BodyPx, false)
			frames[index].separators = append(frames[index].separators, sequenceFrameSeparator{y: y, lines: lines})
			y += len(lines)*lineHeight + 2
		case SequenceFrameEnd:
			index, ok := frameIndex[event.Frame]
			if !ok || len(stack) == 0 || stack[len(stack)-1].index != index {
				return Scene{}, unknown(KindSequence, 0, "frame end")
			}
			frames[index].h = max(y-frames[index].y+1, 3)
			stack = stack[:len(stack)-1]
			y += 2
		default:
			return Scene{}, unknown(KindSequence, 0, "event")
		}
	}
	if len(stack) != 0 {
		return Scene{}, unknown(KindSequence, 0, "unclosed frame")
	}
	for id, start := range active {
		index, ok := participantIndex[id]
		if !ok {
			return Scene{}, unknown(KindSequence, 0, "activation participant")
		}
		sequenceActivationBar(&s, participants[index], start, y, palette)
	}
	y = max(y+2, participantY+maxHeight+4)
	s.Height = max(s.Height, y)

	for _, frame := range frames {
		if frame.h > 0 {
			s.Rounded(frame.x, frame.y, frame.w, frame.h, 1, color.RGBA{}, accent(palette, sequenceFrameColor(frame.frame.Kind)))
		}
	}
	stroke := palette.Link
	if stroke.A == 0 {
		stroke = rule(palette)
	}
	for i, participant := range participants {
		s.Line(participant.cx(), participant.y+participant.h, participant.cx(), y-1, muted(palette), true, false)
		_ = i
	}
	for _, frame := range frames {
		if frame.h == 0 {
			continue
		}
		label := frame.frame.Kind
		if frame.frame.Label != "" {
			label += " " + frame.frame.Label
		}
		width := max(sequenceTextWidth(frame.headerLines, fonts)+2, len([]rune(label))+2)
		flowDrawLabel(&s, Point{X: frame.x + 2, Y: frame.y}, frame.headerLines, width, len(frame.headerLines)*frame.lineHeight, frame.lineHeight, fonts, palette)
		for _, separator := range frame.separators {
			s.Line(frame.x+1, separator.y, frame.x+frame.w-2, separator.y, accent(palette, sequenceFrameColor(frame.frame.Kind)), true, false)
			width := max(sequenceTextWidth(separator.lines, fonts)+2, 3)
			flowDrawLabel(&s, Point{X: frame.x + 2, Y: separator.y}, separator.lines, width, len(separator.lines)*frame.lineHeight, frame.lineHeight, fonts, palette)
		}
	}
	for i, note := range notes {
		s.Rounded(note.x, note.y, note.w, note.h, 1, fade(accent(palette, i+2), 28), accent(palette, i+2))
		s.TextLines(note.lines, note.x+2, note.y+1, note.w-4, note.lineHeight, fonts.BodyPx, AlignCenter, foreground(palette), false)
		s.Line(note.x+note.w-1, note.y+1, note.x+note.w-1, note.y+note.h-2, accent(palette, i+2), false, false)
	}
	for _, message := range messages {
		from := participants[participantIndex[message.event.From]].cx()
		to := participants[participantIndex[message.event.To]].cx()
		s.Arrow(from, message.arrowY, to, message.arrowY, stroke, message.event.Style, message.event.End, message.event.Start)
		if len(message.lines) > 0 && strings.TrimSpace(strings.Join(message.lines, "")) != "" {
			labelX := (from+to)/2 - message.width/2
			labelX = min(max(labelX, 1), max(s.Width-message.width-1, 1))
			flowDrawLabel(&s, Point{X: labelX, Y: message.y}, message.lines, message.width, len(message.lines)*message.lineHeight, message.lineHeight, fonts, palette)
		}
	}
	for i := range participants {
		participant := &participants[i]
		shape := ShapeRounded
		if participant.participant.Actor {
			shape = ShapeStadium
		}
		s.Node(Node{ID: participant.participant.ID, Label: participant.participant.Label, Shape: shape}, participant.x, participant.y, participant.w, participant.h, fonts, palette, palette.CodeBg, accent(palette, i))
	}
	return s, nil
}

func sequenceData(d Diagram) (SequenceData, error) {
	data, ok := d.Data.(SequenceData)
	if !ok {
		return SequenceData{}, unknown(KindSequence, 0, "invalid sequence payload")
	}
	seen := make(map[string]bool, len(data.Participants))
	for _, participant := range data.Participants {
		if !sequenceValidID(participant.ID) || strings.TrimSpace(participant.Label) == "" || seen[participant.ID] {
			return SequenceData{}, unknown(KindSequence, 0, "invalid participant")
		}
		seen[participant.ID] = true
	}
	return data, nil
}

type sequenceGroupBox struct {
	X, Y, W, H int
	lines      []string
	lineHeight int
	colorIndex int
}

func sequenceGroupBoxes(data SequenceData, participants []sequenceParticipantLayout, index map[string]int, bottom, top, lineHeight int, fonts *rich.Fonts) []sequenceGroupBox {
	boxes := make([]sequenceGroupBox, len(data.Groups))
	for i, group := range data.Groups {
		lines, width, lh := labelSize(group.Label, fonts, 24, fonts.BodyPx, false)
		boxes[i] = sequenceGroupBox{Y: top, lines: lines, lineHeight: lh, colorIndex: i, W: max(width+4, 8)}
	}
	depth := make([]int, len(data.Groups))
	groupIndex := make(map[string]int, len(data.Groups))
	for i := range data.Groups {
		groupIndex[data.Groups[i].ID] = i
	}
	for i := range data.Groups {
		if parent, ok := groupIndex[data.Groups[i].Parent]; ok {
			depth[i] = depth[parent] + 1
		}
		boxes[i].Y += depth[i] * lineHeight
	}
	for i, participant := range participants {
		group, ok := groupIndex[participant.participant.Group]
		if !ok {
			continue
		}
		box := &boxes[group]
		if box.H == 0 {
			box.X, box.H = participant.x-2, bottom-box.Y+2
		} else {
			left, right := min(box.X, participant.x-2), max(box.X+box.W, participant.x+participant.w+2)
			box.X, box.W = left, right-left
		}
		_ = i
	}
	for i := range boxes {
		if boxes[i].H == 0 {
			boxes[i].H = max(bottom-boxes[i].Y+2, 3)
		}
		if boxes[i].W < len([]rune(data.Groups[i].Label))+4 {
			lines, _, lh := labelSize(data.Groups[i].Label, fonts, max(boxes[i].W-4, 3), fonts.BodyPx, false)
			boxes[i].lines, boxes[i].lineHeight = lines, lh
		}
	}
	return boxes
}

func maxSequenceDepth(depth []int) int {
	result := 0
	for _, value := range depth {
		result = max(result, value)
	}
	return result
}

type sequenceSizedNote struct {
	note   SequenceNote
	lines  []string
	w, h   int
	lineHeight int
}

func sequenceNoteSizes(data SequenceData, fonts *rich.Fonts) []sequenceSizedNote {
	notes := make([]sequenceSizedNote, 0)
	for _, event := range data.Events {
		if event.Kind != SequenceNoteEvent {
			continue
		}
		lines, width, lineHeight := labelSize(event.Note.Text, fonts, 30, fonts.BodyPx, false)
		notes = append(notes, sequenceSizedNote{
			note: event.Note, lines: lines, w: max(width+4, 8), h: max(len(lines)*lineHeight+2, 3), lineHeight: lineHeight,
		})
	}
	return notes
}

func sequencePlaceNote(note SequenceNote, sizes []sequenceSizedNote, index map[string]int, participants []sequenceParticipantLayout, y, sceneWidth int) (sequenceNoteLayout, bool) {
	for _, sized := range sizes {
		if !reflectSequenceNote(sized.note, note) {
			continue
		}
		anchor, ok := index[note.Participants[0]]
		if !ok {
			return sequenceNoteLayout{}, false
		}
		cx := participants[anchor].cx()
		x := cx - sized.w/2
		switch note.Position {
		case "left":
			x = cx - sized.w - 4
		case "right":
			x = cx + 4
		case "over":
			left, right := cx, cx
			for _, id := range note.Participants[1:] {
				if other, exists := index[id]; exists {
					left = min(left, participants[other].cx())
					right = max(right, participants[other].cx())
				}
			}
			x = (left+right)/2 - sized.w/2
		default:
			return sequenceNoteLayout{}, false
		}
		x = min(max(x, 1), max(sceneWidth-sized.w-1, 1))
		return sequenceNoteLayout{note: note, x: x, y: y, w: sized.w, h: sized.h, lines: sized.lines, lineHeight: sized.lineHeight}, true
	}
	return sequenceNoteLayout{}, false
}

func reflectSequenceNote(a, b SequenceNote) bool {
	if a.Position != b.Position || a.Text != b.Text || len(a.Participants) != len(b.Participants) {
		return false
	}
	for i := range a.Participants {
		if a.Participants[i] != b.Participants[i] {
			return false
		}
	}
	return true
}

func sequenceActivationBar(s *Scene, participant sequenceParticipantLayout, start, end int, palette rich.Palette) {
	if end <= start {
		end = start + 1
	}
	x := participant.cx() + 2
	s.Rect(x, start, 2, end-start, fade(accent(palette, 1), 90), accent(palette, 1))
}

func sequenceTextWidth(lines []string, fonts *rich.Fonts) int {
	width := 1
	for _, line := range lines {
		_, cells, _ := labelSize(line, fonts, 60, fonts.BodyPx, false)
		width = max(width, cells)
	}
	return width
}

func sequenceFrameColor(kind string) int {
	switch kind {
	case "alt":
		return 1
	case "loop":
		return 2
	case "opt":
		return 3
	case "par":
		return 4
	default:
		return 5
	}
}
