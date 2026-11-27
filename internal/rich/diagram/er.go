package diagram

import (
	"strings"
	"unicode"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// ERCardinality identifies one endpoint notation in an ER relationship.
type ERCardinality int

const (
	// ERCardinalityZeroOrOne is Mermaid's zero-or-one notation.
	ERCardinalityZeroOrOne ERCardinality = iota
	// ERCardinalityExactlyOne is Mermaid's exactly-one notation.
	ERCardinalityExactlyOne
	// ERCardinalityZeroOrMore is Mermaid's zero-or-more notation.
	ERCardinalityZeroOrMore
	// ERCardinalityOneOrMore is Mermaid's one-or-more notation.
	ERCardinalityOneOrMore
)

// ERAttribute is a typed entity attribute with optional keys and comment.
type ERAttribute struct {
	Type       string
	Name       string
	PrimaryKey bool
	ForeignKey bool
	UniqueKey  bool
	Comment    string
}

// EREntity is one entity block in an ER diagram.
type EREntity struct {
	ID         string
	Attributes []ERAttribute
}

// ERRelationship is a labelled connection between two entities.
type ERRelationship struct {
	From            string
	To              string
	Label           string
	FromCardinality ERCardinality
	ToCardinality   ERCardinality
	Identifying     bool
}

// ERData is the typed semantic payload of an ER diagram.
type ERData struct {
	Entities      []EREntity
	Relationships []ERRelationship
}

func init() {
	register(KindER, parseER, layoutER)
}

func parseER(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindER}
	data := ERData{}
	indices := make(map[string]int)
	current := -1
	currentLine := 0
	currentConstruct := ""
	titleSeen := false
	var firstLine sourceLine

	ensureEntity := func(id string, line int) (int, error) {
		if index, exists := indices[id]; exists {
			return index, nil
		}
		indices[id] = len(data.Entities)
		data.Entities = append(data.Entities, EREntity{ID: id})
		return len(data.Entities) - 1, nil
	}

	for lineIndex, raw := range lines {
		text := strings.TrimSpace(raw)
		if text == "" {
			continue
		}
		lineNumber := lineIndex + 2
		if firstLine.Text == "" {
			firstLine = sourceLine{Text: text, Number: lineNumber}
		}
		if current >= 0 {
			statements := splitStatements(text)
			for statementIndex, statement := range statements {
				statement = strings.TrimSpace(statement)
				if statement == "" && statementIndex == len(statements)-1 {
					continue
				}
				if statement == "" {
					return Diagram{}, unknown(KindER, lineNumber, text)
				}
				if strings.HasSuffix(statement, "}") {
					attribute := strings.TrimSpace(strings.TrimSuffix(statement, "}"))
					if attribute != "" {
						parsed, err := parseERAttribute(attribute, lineNumber)
						if err != nil {
							return Diagram{}, err
						}
						data.Entities[current].Attributes = append(data.Entities[current].Attributes, parsed)
					}
					current = -1
					currentLine = 0
					currentConstruct = ""
					if statementIndex != len(statements)-1 {
						return Diagram{}, unknown(KindER, lineNumber, text)
					}
					continue
				}
				if strings.ContainsAny(statement, "{}") {
					return Diagram{}, unknown(KindER, lineNumber, text)
				}
				parsed, err := parseERAttribute(statement, lineNumber)
				if err != nil {
					return Diagram{}, err
				}
				data.Entities[current].Attributes = append(data.Entities[current].Attributes, parsed)
			}
			continue
		}

		keyword := strings.Fields(text)[0]
		switch keyword {
		case "title":
			if titleSeen {
				return Diagram{}, unknown(KindER, lineNumber, text)
			}
			title := strings.TrimSpace(text[len("title"):])
			if title == "" {
				return Diagram{}, unknown(KindER, lineNumber, text)
			}
			d.Title = unquote(title)
			titleSeen = true
		default:
			if _, relationshipSyntax := erFindConnector(text); relationshipSyntax {
				relationship, err := parseERRelationship(text, lineNumber)
				if err != nil {
					return Diagram{}, err
				}
				if _, err := ensureEntity(relationship.From, lineNumber); err != nil {
					return Diagram{}, err
				}
				if _, err := ensureEntity(relationship.To, lineNumber); err != nil {
					return Diagram{}, err
				}
				data.Relationships = append(data.Relationships, relationship)
				continue
			}
			before, content, open, closed, ok := diagramBlockParts(text)
			if !ok {
				return Diagram{}, unknown(KindER, lineNumber, text)
			}
			if open {
				id, rest, ok := flowReadID(strings.TrimSpace(before))
				if !ok || !diagramValidID(id) || strings.TrimSpace(rest) != "" {
					return Diagram{}, unknown(KindER, lineNumber, text)
				}
				index, err := ensureEntity(id, lineNumber)
				if err != nil {
					return Diagram{}, err
				}
				if closed {
					if err := appendERAttributeStatements(&data.Entities[index].Attributes, content, lineNumber); err != nil {
						return Diagram{}, err
					}
				} else {
					current = index
					currentLine = lineNumber
					currentConstruct = text
				}
				continue
			}
			relationship, err := parseERRelationship(text, lineNumber)
			if err != nil {
				return Diagram{}, err
			}
			if _, err := ensureEntity(relationship.From, lineNumber); err != nil {
				return Diagram{}, err
			}
			if _, err := ensureEntity(relationship.To, lineNumber); err != nil {
				return Diagram{}, err
			}
			data.Relationships = append(data.Relationships, relationship)
		}
	}
	if current >= 0 {
		return Diagram{}, unknown(KindER, currentLine, currentConstruct)
	}
	if len(data.Entities) == 0 {
		return Diagram{}, unknown(KindER, firstLine.Number, firstLine.Text)
	}
	d.Data = data
	return d, nil
}

func appendERAttributeStatements(attributes *[]ERAttribute, content string, line int) error {
	for _, statement := range splitStatements(content) {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		attribute, err := parseERAttribute(statement, line)
		if err != nil {
			return err
		}
		*attributes = append(*attributes, attribute)
	}
	return nil
}

func parseERAttribute(text string, line int) (ERAttribute, error) {
	original := text
	head, comment, err := erAttributeComment(text)
	if err != nil {
		return ERAttribute{}, unknown(KindER, line, original)
	}
	fields := strings.Fields(head)
	if len(fields) < 2 || !erValidType(fields[0]) || !diagramValidID(fields[1]) {
		return ERAttribute{}, unknown(KindER, line, original)
	}
	if len(fields) == 2 && (fields[1] == "PK" || fields[1] == "FK" || fields[1] == "UK") {
		return ERAttribute{}, unknown(KindER, line, original)
	}
	attribute := ERAttribute{Type: fields[0], Name: fields[1], Comment: comment}
	seen := make(map[string]bool, 3)
	for _, key := range fields[2:] {
		if seen[key] {
			return ERAttribute{}, unknown(KindER, line, original)
		}
		switch key {
		case "PK":
			attribute.PrimaryKey = true
		case "FK":
			attribute.ForeignKey = true
		case "UK":
			attribute.UniqueKey = true
		default:
			return ERAttribute{}, unknown(KindER, line, original)
		}
		seen[key] = true
	}
	return attribute, nil
}

func erAttributeComment(text string) (string, string, error) {
	var quote byte
	escaped := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 {
			if c == '\\' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c != '"' && c != '\'' {
			continue
		}
		token, rest, ok := flowReadQuoted(text[i:])
		if !ok || strings.TrimSpace(rest) != "" {
			return "", "", unknown(KindER, 0, text)
		}
		return strings.TrimSpace(text[:i]), unquote(token), nil
	}
	return strings.TrimSpace(text), "", nil
}

func parseERRelationship(text string, line int) (ERRelationship, error) {
	original := text
	index, ok := erFindConnector(text)
	if !ok {
		return ERRelationship{}, unknown(KindER, line, original)
	}
	from := strings.TrimSpace(text[:index-2])
	if !diagramValidID(from) {
		return ERRelationship{}, unknown(KindER, line, original)
	}
	fromCardinality, ok := erCardinality(text[index-2:index], false)
	if !ok {
		return ERRelationship{}, unknown(KindER, line, original)
	}
	toCardinality, ok := erCardinality(text[index+2:index+4], true)
	if !ok {
		return ERRelationship{}, unknown(KindER, line, original)
	}
	after := strings.TrimSpace(text[index+4:])
	parts := splitTopLevelString(after, ':')
	if len(parts) > 2 {
		return ERRelationship{}, unknown(KindER, line, original)
	}
	to := strings.TrimSpace(parts[0])
	if !diagramValidID(to) {
		return ERRelationship{}, unknown(KindER, line, original)
	}
	relationship := ERRelationship{
		From: from, To: to, FromCardinality: fromCardinality, ToCardinality: toCardinality,
		Identifying: text[index:index+2] == "--",
	}
	if len(parts) == 2 {
		relationship.Label = unquote(parts[1])
		if relationship.Label == "" {
			return ERRelationship{}, unknown(KindER, line, original)
		}
	}
	return relationship, nil
}

func erFindConnector(text string) (int, bool) {
	var quote byte
	escaped := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 {
			if c == '\\' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if i < 2 || i+4 > len(text) || text[i-1] == ' ' || text[i-1] == '\t' {
			continue
		}
		if (strings.HasPrefix(text[i:], "--") || strings.HasPrefix(text[i:], "..")) && i+4 < len(text) && (text[i+4] == ' ' || text[i+4] == '\t') {
			return i, true
		}
	}
	return 0, false
}

func erCardinality(notation string, right bool) (ERCardinality, bool) {
	if right {
		switch notation {
		case "o|":
			return ERCardinalityZeroOrOne, true
		case "||":
			return ERCardinalityExactlyOne, true
		case "o{":
			return ERCardinalityZeroOrMore, true
		case "|{":
			return ERCardinalityOneOrMore, true
		default:
			return 0, false
		}
	}
	switch notation {
	case "|o":
		return ERCardinalityZeroOrOne, true
	case "||":
		return ERCardinalityExactlyOne, true
	case "}o":
		return ERCardinalityZeroOrMore, true
	case "}|":
		return ERCardinalityOneOrMore, true
	default:
		return 0, false
	}
}

func erValidType(text string) bool {
	if text == "" {
		return false
	}
	for i, r := range text {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '.' && r != '[' && r != ']' {
			return false
		}
	}
	return true
}

func erData(d Diagram) (ERData, error) {
	data, ok := d.Data.(ERData)
	if !ok || len(data.Entities) == 0 {
		return ERData{}, invalid(KindER, 0, "invalid ER payload")
	}
	indices := make(map[string]int, len(data.Entities))
	for _, entity := range data.Entities {
		if !diagramValidID(entity.ID) {
			return ERData{}, invalid(KindER, 0, "invalid entity identifier")
		}
		if _, exists := indices[entity.ID]; exists {
			return ERData{}, invalid(KindER, 0, "duplicate entity identifier")
		}
		indices[entity.ID] = 0
		for _, attribute := range entity.Attributes {
			if !erValidType(attribute.Type) || !diagramValidID(attribute.Name) {
				return ERData{}, invalid(KindER, 0, "invalid entity attribute")
			}
		}
	}
	for _, relationship := range data.Relationships {
		if _, exists := indices[relationship.From]; !exists {
			return ERData{}, invalid(KindER, 0, "unknown entity endpoint")
		}
		if _, exists := indices[relationship.To]; !exists {
			return ERData{}, invalid(KindER, 0, "unknown entity endpoint")
		}
		if relationship.FromCardinality < ERCardinalityZeroOrOne || relationship.FromCardinality > ERCardinalityOneOrMore || relationship.ToCardinality < ERCardinalityZeroOrOne || relationship.ToCardinality > ERCardinalityOneOrMore {
			return ERData{}, invalid(KindER, 0, "invalid ER cardinality")
		}
	}
	return data, nil
}

type erMeasuredBox struct {
	headerLines []string
	headerH     int
	lineHeight  int
	attributes  []erMeasuredAttribute
	attributesH int
	w, h        int
}

type erMeasuredAttribute struct {
	lines      []string
	lineHeight int
}

func erMeasureBox(entity EREntity, fonts *rich.Fonts) erMeasuredBox {
	headerLines, headerWidth, lineHeight := labelSize(entity.ID, fonts, 24, fonts.BodyPx, false)
	measured := erMeasuredBox{headerLines: headerLines, headerH: max(len(headerLines)*lineHeight+2, 3), lineHeight: lineHeight, w: max(16, headerWidth+4)}
	for _, attribute := range entity.Attributes {
		text := erAttributeDisplay(attribute)
		lines, width, attributeLineHeight := labelSize(text, fonts, 26, fonts.BodyPx, false)
		measured.attributes = append(measured.attributes, erMeasuredAttribute{lines: lines, lineHeight: attributeLineHeight})
		measured.w = max(measured.w, width+4)
		measured.attributesH += len(lines) * attributeLineHeight
		if attribute.Comment != "" {
			comment := `"` + attribute.Comment + `"`
			commentLines, commentWidth, _ := labelSize(comment, fonts, 26, fonts.BodyPx, false)
			measured.attributes = append(measured.attributes, erMeasuredAttribute{lines: commentLines, lineHeight: attributeLineHeight})
			measured.w = max(measured.w, commentWidth+4)
			measured.attributesH += len(commentLines) * attributeLineHeight
		}
	}
	measured.h = measured.headerH + measured.attributesH + 1
	return measured
}

func erAttributeDisplay(attribute ERAttribute) string {
	var text strings.Builder
	for _, key := range []struct {
		set  bool
		name string
	}{{attribute.PrimaryKey, "PK"}, {attribute.ForeignKey, "FK"}, {attribute.UniqueKey, "UK"}} {
		if key.set {
			text.WriteString(key.name)
			text.WriteByte(' ')
		}
	}
	text.WriteString(attribute.Type)
	text.WriteByte(' ')
	text.WriteString(attribute.Name)
	return text.String()
}

func layoutER(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, err := erData(d)
	if err != nil {
		return Scene{}, err
	}
	if fonts == nil {
		return Scene{}, invalid(KindER, 0, "fonts are required")
	}
	measured := make([]erMeasuredBox, len(data.Entities))
	items := make([]diagramGridItem, len(data.Entities))
	maxRelationWidth, maxRelationHeight := 0, 0
	for i, entity := range data.Entities {
		measured[i] = erMeasureBox(entity, fonts)
		items[i] = diagramGridItem{w: measured[i].w, h: measured[i].h}
	}
	for _, relationship := range data.Relationships {
		if relationship.Label != "" {
			lines, width, lineHeight := flowEdgeLabelSize(relationship.Label, fonts)
			maxRelationWidth = max(maxRelationWidth, width)
			maxRelationHeight = max(maxRelationHeight, len(lines)*lineHeight)
		}
	}
	columns := min(3, len(items))
	const margin = 3
	horizontalGap := max(14, maxRelationWidth+4)
	verticalGap := max(8, maxRelationHeight+4)
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	startY := titleHeight + 1 + margin
	boxes, gridWidth, gridHeight := diagramGridPlace(items, columns, horizontalGap, verticalGap, margin, startY)
	s := newScene()
	s.Width = max(titleWidth+2, margin+gridWidth+margin)
	s.Height = startY + gridHeight + margin
	if titleHeight > 0 {
		s.TextLinesBold(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	indices := make(map[string]int, len(data.Entities))
	for i, entity := range data.Entities {
		indices[entity.ID] = i
	}
	stroke := diagramStroke(palette)
	paths := make([][]Point, len(data.Relationships))
	for i, relationship := range data.Relationships {
		path, ok := diagramRoute(boxes[indices[relationship.From]], boxes[indices[relationship.To]], i, s.Width, s.Height, boxes)
		if !ok {
			return Scene{}, invalid(KindER, 0, "unable to route ER relationship")
		}
		paths[i] = path
		diagramDrawRelation(&s, path, erEdgeStyle(relationship), erDiagramMarker(relationship.FromCardinality), erDiagramMarker(relationship.ToCardinality), stroke)
	}
	for i, entity := range data.Entities {
		erDrawBox(&s, entity, measured[i], boxes[i], fonts, palette, i)
	}
	occupied := make([]diagramTextLabel, 0, len(data.Relationships))
	for i, relationship := range data.Relationships {
		if relationship.Label == "" {
			continue
		}
		label := diagramPlaceRelationLabel(paths[i], relationship.Label, 1, boxes, occupied, titleHeight+1, fonts)
		diagramDrawPlacedLabel(&s, label, fonts, palette)
		occupied = append(occupied, label)
	}
	return s, nil
}

func erDrawBox(s *Scene, entity EREntity, measured erMeasuredBox, box diagramGridBox, fonts *rich.Fonts, palette rich.Palette, index int) {
	stroke := accent(palette, index)
	headerFill := fade(stroke, 48)
	s.Rect(box.x, box.y, box.w, box.h, palette.CodeBg, stroke)
	s.Rect(box.x, box.y, box.w, measured.headerH, headerFill, stroke)
	s.Line(box.x, box.y+measured.headerH, box.x+box.w-1, box.y+measured.headerH, rule(palette), false, false)
	headerHeight := len(measured.headerLines) * measured.lineHeight
	s.TextLinesBold(measured.headerLines, box.x+1, box.y+(measured.headerH-headerHeight)/2, box.w-2, measured.lineHeight, fonts.BodyPx, AlignCenter, contrastText(headerFill, palette), false)
	y := box.y + measured.headerH
	for _, attribute := range measured.attributes {
		s.TextLines(attribute.lines, box.x+2, y, box.w-4, attribute.lineHeight, fonts.BodyPx, AlignLeft, foreground(palette), false)
		y += len(attribute.lines) * attribute.lineHeight
	}
}

func erEdgeStyle(relationship ERRelationship) EdgeStyle {
	if relationship.Identifying {
		return EdgeSolid
	}
	return EdgeDotted
}

func erDiagramMarker(cardinality ERCardinality) diagramEndpointMark {
	switch cardinality {
	case ERCardinalityZeroOrOne:
		return diagramMarkZeroOrOne
	case ERCardinalityExactlyOne:
		return diagramMarkExactlyOne
	case ERCardinalityZeroOrMore:
		return diagramMarkZeroOrMore
	case ERCardinalityOneOrMore:
		return diagramMarkOneOrMore
	default:
		return diagramMarkNone
	}
}
