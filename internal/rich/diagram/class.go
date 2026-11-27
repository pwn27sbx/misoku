package diagram

import (
	"image/color"
	"math"
	"strings"
	"unicode"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// ClassMemberKind distinguishes a field from a method.
type ClassMemberKind int

const (
	// ClassField is a typed attribute.
	ClassField ClassMemberKind = iota
	// ClassMethod is a callable member.
	ClassMethod
)

// ClassParameter is one named method parameter and its optional type.
type ClassParameter struct {
	Name string
	Type string
}

// ClassMember is a field or method in a class compartment.
type ClassMember struct {
	Kind       ClassMemberKind
	Visibility string
	Name       string
	Type       string
	Static     bool
	Abstract   bool
	Parameters []ClassParameter
	ReturnType string
}

// Class is one class declaration, including its optional generic parameters.
type Class struct {
	ID             string
	Label          string
	TypeParameters []string
	Members        []ClassMember
}

// ClassRelationKind identifies the UML relationship represented by a connector.
type ClassRelationKind int

const (
	// ClassInheritance is a generalization relationship.
	ClassInheritance ClassRelationKind = iota
	// ClassComposition is a filled-diamond whole/part relationship.
	ClassComposition
	// ClassAggregation is a hollow-diamond whole/part relationship.
	ClassAggregation
	// ClassAssociation is a structural relationship.
	ClassAssociation
	// ClassDependency is a dotted directed relationship.
	ClassDependency
	// ClassRealization is a dotted hollow-triangle relationship.
	ClassRealization
)

// ClassMarker identifies a decoration at one end of a class relation.
type ClassMarker int

const (
	// ClassMarkerNone leaves the endpoint undecorated.
	ClassMarkerNone ClassMarker = iota
	// ClassMarkerTriangle draws a hollow generalization triangle.
	ClassMarkerTriangle
	// ClassMarkerFilledArrow draws a directed association or dependency arrow.
	ClassMarkerFilledArrow
	// ClassMarkerDiamond draws a filled composition diamond.
	ClassMarkerDiamond
	// ClassMarkerHollowDiamond draws a hollow aggregation diamond.
	ClassMarkerHollowDiamond
)

// ClassRelation is a labelled UML relationship between two classes.
type ClassRelation struct {
	From             string
	To               string
	Label            string
	FromMultiplicity string
	ToMultiplicity   string
	Kind             ClassRelationKind
	Style            EdgeStyle
	StartMarker      ClassMarker
	EndMarker        ClassMarker
}

// ClassData is the typed semantic payload of a class diagram.
type ClassData struct {
	Classes   []Class
	Relations []ClassRelation
}

type classDeclaration struct {
	id             string
	label          string
	typeParameters []string
}

func init() {
	register(KindClass, parseClass, layoutClass)
}

func parseClass(lines []string) (Diagram, error) {
	d := Diagram{Kind: KindClass}
	data := ClassData{}
	indices := make(map[string]int)
	current := -1
	currentLine := 0
	currentConstruct := ""
	titleSeen := false
	var firstLine sourceLine

	ensureClass := func(decl classDeclaration, line int) (int, error) {
		if index, exists := indices[decl.id]; exists {
			existing := &data.Classes[index]
			if decl.label != "" {
				if existing.Label != "" && existing.Label != decl.label {
					return 0, unknown(KindClass, line, "conflicting label for "+decl.id)
				}
				existing.Label = decl.label
			}
			if len(decl.typeParameters) > 0 {
				if len(existing.TypeParameters) > 0 && !classSameStrings(existing.TypeParameters, decl.typeParameters) {
					return 0, unknown(KindClass, line, "conflicting generics for "+decl.id)
				}
				existing.TypeParameters = append([]string(nil), decl.typeParameters...)
			}
			return index, nil
		}
		indices[decl.id] = len(data.Classes)
		data.Classes = append(data.Classes, Class{
			ID: decl.id, Label: decl.label,
			TypeParameters: append([]string(nil), decl.typeParameters...),
		})
		return len(data.Classes) - 1, nil
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
					return Diagram{}, unknown(KindClass, lineNumber, text)
				}
				if strings.HasSuffix(statement, "}") {
					member := strings.TrimSpace(strings.TrimSuffix(statement, "}"))
					if member != "" {
						parsed, err := parseClassMember(member, lineNumber)
						if err != nil {
							return Diagram{}, err
						}
						data.Classes[current].Members = append(data.Classes[current].Members, parsed)
					}
					current = -1
					currentLine = 0
					currentConstruct = ""
					if statementIndex != len(statements)-1 {
						return Diagram{}, unknown(KindClass, lineNumber, text)
					}
					continue
				}
				if strings.ContainsAny(statement, "{}") {
					return Diagram{}, unknown(KindClass, lineNumber, text)
				}
				parsed, err := parseClassMember(statement, lineNumber)
				if err != nil {
					return Diagram{}, err
				}
				data.Classes[current].Members = append(data.Classes[current].Members, parsed)
			}
			continue
		}

		keyword := strings.Fields(text)[0]
		switch keyword {
		case "title":
			if titleSeen {
				return Diagram{}, unknown(KindClass, lineNumber, text)
			}
			title := strings.TrimSpace(text[len("title"):])
			if title == "" {
				return Diagram{}, unknown(KindClass, lineNumber, text)
			}
			d.Title = unquote(title)
			titleSeen = true
		case "class":
			before, content, open, closed, ok := diagramBlockParts(text)
			if !ok {
				return Diagram{}, unknown(KindClass, lineNumber, text)
			}
			decl, err := parseClassDeclaration(before, lineNumber)
			if err != nil {
				return Diagram{}, err
			}
			index, err := ensureClass(decl, lineNumber)
			if err != nil {
				return Diagram{}, err
			}
			if open {
				if closed {
					if err := appendClassMemberStatements(&data.Classes[index].Members, content, lineNumber); err != nil {
						return Diagram{}, err
					}
				} else {
					current = index
					currentLine = lineNumber
					currentConstruct = text
				}
			}
		default:
			relation, err := parseClassRelation(text, lineNumber)
			if err != nil {
				return Diagram{}, err
			}
			if _, err := ensureClass(classDeclaration{id: relation.From}, lineNumber); err != nil {
				return Diagram{}, err
			}
			if _, err := ensureClass(classDeclaration{id: relation.To}, lineNumber); err != nil {
				return Diagram{}, err
			}
			data.Relations = append(data.Relations, relation)
		}
	}
	if current >= 0 {
		return Diagram{}, unknown(KindClass, currentLine, currentConstruct)
	}
	if len(data.Classes) == 0 {
		return Diagram{}, unknown(KindClass, firstLine.Number, firstLine.Text)
	}
	d.Data = data
	return d, nil
}

func parseClassDeclaration(text string, line int) (classDeclaration, error) {
	text = strings.TrimSpace(text)
	if !classHasKeyword(text, "class") {
		return classDeclaration{}, unknown(KindClass, line, text)
	}
	rest := strings.TrimSpace(text[len("class"):])
	if rest == "" {
		return classDeclaration{}, unknown(KindClass, line, text)
	}
	decl := classDeclaration{}
	if rest[0] == '"' || rest[0] == '\'' {
		token, after, ok := flowReadQuoted(rest)
		if !ok {
			return classDeclaration{}, unknown(KindClass, line, text)
		}
		decl.label = unquote(token)
		fields := strings.Fields(strings.TrimSpace(after))
		if decl.label == "" || len(fields) != 2 || fields[0] != "as" || !diagramValidID(fields[1]) {
			return classDeclaration{}, unknown(KindClass, line, text)
		}
		decl.id = fields[1]
		return decl, nil
	}

	id, after, ok := flowReadID(rest)
	if !ok || !diagramValidID(id) {
		return classDeclaration{}, unknown(KindClass, line, text)
	}
	decl.id = id
	after = strings.TrimSpace(after)
	if strings.HasPrefix(after, "~") {
		end := strings.IndexByte(after[1:], '~')
		if end < 0 {
			return classDeclaration{}, unknown(KindClass, line, text)
		}
		end++
		parameters, ok := classTypeParameters(after[1:end])
		if !ok {
			return classDeclaration{}, unknown(KindClass, line, text)
		}
		decl.typeParameters = parameters
		after = strings.TrimSpace(after[end+1:])
	}
	if strings.HasPrefix(after, "[") {
		after, label, ok := flowReadShape(after, "[", "]")
		if !ok || strings.TrimSpace(label) == "" || strings.TrimSpace(after) != "" {
			return classDeclaration{}, unknown(KindClass, line, text)
		}
		decl.label = unquote(label)
		after = ""
	}
	if after != "" {
		return classDeclaration{}, unknown(KindClass, line, text)
	}
	return decl, nil
}

func appendClassMemberStatements(members *[]ClassMember, content string, line int) error {
	for _, statement := range splitStatements(content) {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		member, err := parseClassMember(statement, line)
		if err != nil {
			return err
		}
		*members = append(*members, member)
	}
	return nil
}

func parseClassMember(text string, line int) (ClassMember, error) {
	original := text
	text = strings.TrimSpace(text)
	if text == "" {
		return ClassMember{}, unknown(KindClass, line, original)
	}
	member := ClassMember{Visibility: "~"}
	if strings.ContainsRune("+-#~", rune(text[0])) {
		member.Visibility = text[:1]
		text = strings.TrimSpace(text[1:])
	}
	if text == "" {
		return ClassMember{}, unknown(KindClass, line, original)
	}

	if open := strings.IndexByte(text, '('); open >= 0 {
		name := strings.TrimSpace(text[:open])
		if !diagramValidID(name) {
			return ClassMember{}, unknown(KindClass, line, original)
		}
		after, parameterText, ok := flowReadShape(text[open:], "(", ")")
		if !ok {
			return ClassMember{}, unknown(KindClass, line, original)
		}
		member.Kind = ClassMethod
		member.Name = name
		parameters, err := parseClassParameters(parameterText, line, original)
		if err != nil {
			return ClassMember{}, err
		}
		member.Parameters = parameters
		after, static, abstract, ok := classMemberMarkers(after, true)
		if !ok {
			return ClassMember{}, unknown(KindClass, line, original)
		}
		member.Static, member.Abstract = static, abstract
		after = strings.TrimSpace(after)
		if strings.HasPrefix(after, ":") {
			after = strings.TrimSpace(after[1:])
		}
		if after != "" && !classValidType(after) {
			return ClassMember{}, unknown(KindClass, line, original)
		}
		member.ReturnType = after
		return member, nil
	}

	after, static, _, ok := classMemberMarkers(text, false)
	if !ok {
		return ClassMember{}, unknown(KindClass, line, original)
	}
	member.Kind = ClassField
	member.Static = static
	fields := strings.Fields(after)
	if len(fields) == 0 {
		return ClassMember{}, unknown(KindClass, line, original)
	}
	if len(fields) == 1 {
		member.Name = fields[0]
	} else {
		member.Type = fields[0]
		member.Name = strings.TrimSpace(after[len(fields[0]):])
	}
	if !diagramValidID(member.Name) || (member.Type != "" && !classValidType(member.Type)) {
		return ClassMember{}, unknown(KindClass, line, original)
	}
	return member, nil
}

func parseClassParameters(text string, line int, original string) ([]ClassParameter, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	parts := splitTopLevelString(text, ',')
	parameters := make([]ClassParameter, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, unknown(KindClass, line, original)
		}
		name, typeName := part, ""
		if index := strings.IndexAny(part, " \t"); index >= 0 {
			name = strings.TrimSpace(part[:index])
			typeName = strings.TrimSpace(part[index:])
		}
		if strings.HasSuffix(name, ":") {
			name = strings.TrimSpace(strings.TrimSuffix(name, ":"))
			if typeName == "" && strings.Contains(part, ":") {
				typeName = strings.TrimSpace(part[strings.IndexByte(part, ':')+1:])
			}
		}
		if !diagramValidID(name) || (typeName != "" && !classValidType(typeName)) {
			return nil, unknown(KindClass, line, original)
		}
		parameters = append(parameters, ClassParameter{Name: name, Type: typeName})
	}
	return parameters, nil
}

func classMemberMarkers(text string, allowAbstract bool) (string, bool, bool, bool) {
	text = strings.TrimSpace(text)
	static, abstract := false, false
	for {
		changed := false
		if strings.HasPrefix(text, "$") {
			if static {
				return "", false, false, false
			}
			static = true
			text = strings.TrimSpace(text[1:])
			changed = true
		}
		if strings.HasSuffix(text, "$") {
			if static {
				return "", false, false, false
			}
			static = true
			text = strings.TrimSpace(strings.TrimSuffix(text, "$"))
			changed = true
		}
		if allowAbstract {
			if strings.HasPrefix(text, "*") {
				if abstract {
					return "", false, false, false
				}
				abstract = true
				text = strings.TrimSpace(text[1:])
				changed = true
			}
			if strings.HasSuffix(text, "*") {
				if abstract {
					return "", false, false, false
				}
				abstract = true
				text = strings.TrimSpace(strings.TrimSuffix(text, "*"))
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return text, static, abstract, text != "" || static || abstract || (allowAbstract && text == "")
}

func parseClassRelation(text string, line int) (ClassRelation, error) {
	original := text
	token, index := classFindConnector(text)
	if token == "" {
		return ClassRelation{}, unknown(KindClass, line, original)
	}
	left := strings.TrimSpace(text[:index])
	right := strings.TrimSpace(text[index+len(token):])
	from, fromMultiplicity, ok := classLeftEndpoint(left)
	if !ok {
		return ClassRelation{}, unknown(KindClass, line, original)
	}
	labelParts := splitTopLevelString(right, ':')
	if len(labelParts) > 2 {
		return ClassRelation{}, unknown(KindClass, line, original)
	}
	to, toMultiplicity, ok := classRightEndpoint(labelParts[0])
	if !ok {
		return ClassRelation{}, unknown(KindClass, line, original)
	}
	relation := ClassRelation{From: from, To: to, FromMultiplicity: fromMultiplicity, ToMultiplicity: toMultiplicity}
	if len(labelParts) == 2 {
		relation.Label = unquote(labelParts[1])
		if relation.Label == "" {
			return ClassRelation{}, unknown(KindClass, line, original)
		}
	}
	kind, style, start, end, ok := classConnector(token)
	if !ok {
		return ClassRelation{}, unknown(KindClass, line, original)
	}
	relation.Kind, relation.Style = kind, style
	relation.StartMarker, relation.EndMarker = start, end
	return relation, nil
}

func classFindConnector(text string) (string, int) {
	tokens := [...]string{"..|>", "<|..", "<|--", "--|>", "*--", "--*", "o--", "--o", "-->", "<--", "..>", "<..", "--", ".."}
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
		for _, token := range tokens {
			if strings.HasPrefix(text[i:], token) {
				return token, i
			}
		}
	}
	return "", -1
}

func classLeftEndpoint(text string) (string, string, bool) {
	id, rest, ok := flowReadID(strings.TrimSpace(text))
	if !ok || !diagramValidID(id) {
		return "", "", false
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return id, "", true
	}
	if strings.HasPrefix(rest, "\"") || strings.HasPrefix(rest, "'") {
		token, after, ok := flowReadQuoted(rest)
		if !ok || strings.TrimSpace(after) != "" {
			return "", "", false
		}
		multiplicity := unquote(token)
		return id, multiplicity, classValidMultiplicity(multiplicity)
	}
	if len(strings.Fields(rest)) != 1 {
		return "", "", false
	}
	return id, rest, classValidMultiplicity(rest)
}

func classRightEndpoint(text string) (string, string, bool) {
	text = strings.TrimSpace(text)
	first, rest, ok := flowReadID(text)
	if !ok {
		return "", "", false
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return first, "", diagramValidID(first)
	}
	id := rest
	if len(strings.Fields(id)) != 1 || !diagramValidID(id) {
		return "", "", false
	}
	return id, first, classValidMultiplicity(first)
}

func classConnector(token string) (ClassRelationKind, EdgeStyle, ClassMarker, ClassMarker, bool) {
	switch token {
	case "<|--":
		return ClassInheritance, EdgeSolid, ClassMarkerTriangle, ClassMarkerNone, true
	case "--|>":
		return ClassInheritance, EdgeSolid, ClassMarkerNone, ClassMarkerTriangle, true
	case "*--":
		return ClassComposition, EdgeSolid, ClassMarkerDiamond, ClassMarkerNone, true
	case "--*":
		return ClassComposition, EdgeSolid, ClassMarkerNone, ClassMarkerDiamond, true
	case "o--":
		return ClassAggregation, EdgeSolid, ClassMarkerHollowDiamond, ClassMarkerNone, true
	case "--o":
		return ClassAggregation, EdgeSolid, ClassMarkerNone, ClassMarkerHollowDiamond, true
	case "-->":
		return ClassAssociation, EdgeSolid, ClassMarkerNone, ClassMarkerFilledArrow, true
	case "<--":
		return ClassAssociation, EdgeSolid, ClassMarkerFilledArrow, ClassMarkerNone, true
	case "--":
		return ClassAssociation, EdgeSolid, ClassMarkerNone, ClassMarkerNone, true
	case "..>":
		return ClassDependency, EdgeDotted, ClassMarkerNone, ClassMarkerFilledArrow, true
	case "<..":
		return ClassDependency, EdgeDotted, ClassMarkerFilledArrow, ClassMarkerNone, true
	case "..":
		return ClassAssociation, EdgeDotted, ClassMarkerNone, ClassMarkerNone, true
	case "..|>":
		return ClassRealization, EdgeDotted, ClassMarkerNone, ClassMarkerTriangle, true
	case "<|..":
		return ClassRealization, EdgeDotted, ClassMarkerTriangle, ClassMarkerNone, true
	default:
		return 0, 0, 0, 0, false
	}
}

func classTypeParameters(text string) ([]string, bool) {
	parts, ok := diagramSplitGenerics(text)
	if !ok || len(parts) == 0 {
		return nil, false
	}
	for _, part := range parts {
		if part == "" || strings.ContainsAny(part, "~{};") {
			return nil, false
		}
	}
	return parts, true
}

func diagramSplitGenerics(text string) ([]string, bool) {
	var parts []string
	start, depth := 0, 0
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
		switch c {
		case '\'', '"':
			quote = c
		case '<':
			depth++
		case '>':
			depth--
			if depth < 0 {
				return nil, false
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(text[start:i]))
				start = i + 1
			}
		}
	}
	if depth != 0 || quote != 0 {
		return nil, false
	}
	parts = append(parts, strings.TrimSpace(text[start:]))
	return parts, true
}

func classSameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func classHasKeyword(text, keyword string) bool {
	return strings.HasPrefix(text, keyword) && len(text) > len(keyword) && (text[len(keyword)] == ' ' || text[len(keyword)] == '\t')
}

func classValidMultiplicity(text string) bool {
	text = strings.TrimSpace(text)
	return text != "" && len(strings.Fields(text)) == 1 && !strings.ContainsAny(text, "{};:\"'<>|")
}

func classValidType(text string) bool {
	text = strings.TrimSpace(text)
	return text != "" && !strings.HasPrefix(text, "?") && !strings.ContainsAny(text, "{};\"'\t\r\n")
}

func classData(d Diagram) (ClassData, error) {
	data, ok := d.Data.(ClassData)
	if !ok || len(data.Classes) == 0 {
		return ClassData{}, invalid(KindClass, 0, "invalid class payload")
	}
	indices := make(map[string]int, len(data.Classes))
	for i, class := range data.Classes {
		if !diagramValidID(class.ID) {
			return ClassData{}, invalid(KindClass, 0, "invalid class identifier")
		}
		if _, exists := indices[class.ID]; exists {
			return ClassData{}, invalid(KindClass, 0, "duplicate class identifier")
		}
		indices[class.ID] = i
		for _, member := range class.Members {
			if !diagramValidID(member.Name) || (member.Kind != ClassField && member.Kind != ClassMethod) {
				return ClassData{}, invalid(KindClass, 0, "invalid class member")
			}
			if member.Visibility != "" && !strings.ContainsRune("+-#~", rune(member.Visibility[0])) {
				return ClassData{}, invalid(KindClass, 0, "invalid class visibility")
			}
			if member.Kind == ClassMethod {
				for _, parameter := range member.Parameters {
					if !diagramValidID(parameter.Name) {
						return ClassData{}, invalid(KindClass, 0, "invalid class parameter")
					}
				}
			}
		}
	}
	for _, relation := range data.Relations {
		if _, exists := indices[relation.From]; !exists {
			return ClassData{}, invalid(KindClass, 0, "unknown class endpoint")
		}
		if _, exists := indices[relation.To]; !exists {
			return ClassData{}, invalid(KindClass, 0, "unknown class endpoint")
		}
		if relation.Kind < ClassInheritance || relation.Kind > ClassRealization || relation.StartMarker < ClassMarkerNone || relation.StartMarker > ClassMarkerHollowDiamond || relation.EndMarker < ClassMarkerNone || relation.EndMarker > ClassMarkerHollowDiamond {
			return ClassData{}, invalid(KindClass, 0, "invalid class relation")
		}
		if relation.Style != EdgeSolid && relation.Style != EdgeDotted {
			return ClassData{}, invalid(KindClass, 0, "invalid class relation style")
		}
	}
	return data, nil
}

type classMeasuredBox struct {
	headerLines []string
	headerH     int
	lineHeight  int
	fields      []classMeasuredMember
	methods     []classMeasuredMember
	fieldsH     int
	methodsH    int
	w, h        int
}

type classMeasuredMember struct {
	lines      []string
	lineHeight int
}

func classMeasureBox(class Class, fonts *rich.Fonts) classMeasuredBox {
	header := classDisplayName(class)
	headerLines, headerWidth, lineHeight := labelSize(header, fonts, 24, fonts.BodyPx, false)
	measured := classMeasuredBox{headerLines: headerLines, headerH: max(len(headerLines)*lineHeight+2, 3), lineHeight: lineHeight, w: max(14, headerWidth+4)}
	for _, member := range class.Members {
		lines, width, memberLineHeight := labelSize(classMemberDisplay(member), fonts, 24, fonts.BodyPx, false)
		entry := classMeasuredMember{lines: lines, lineHeight: memberLineHeight}
		measured.w = max(measured.w, width+4)
		if member.Kind == ClassField {
			measured.fields = append(measured.fields, entry)
			measured.fieldsH += len(lines) * memberLineHeight
		} else {
			measured.methods = append(measured.methods, entry)
			measured.methodsH += len(lines) * memberLineHeight
		}
	}
	separator := 0
	if len(measured.fields) > 0 && len(measured.methods) > 0 {
		separator = 1
	}
	measured.h = measured.headerH + measured.fieldsH + measured.methodsH + separator + 1
	return measured
}

func classDisplayName(class Class) string {
	name := class.Label
	if name == "" {
		name = class.ID
	}
	if len(class.TypeParameters) > 0 {
		name += "<" + strings.Join(class.TypeParameters, ", ") + ">"
	}
	return name
}

func classMemberDisplay(member ClassMember) string {
	var text strings.Builder
	text.WriteString(member.Visibility)
	if member.Kind == ClassField {
		if member.Type != "" {
			text.WriteString(member.Type)
			text.WriteByte(' ')
		}
		text.WriteString(member.Name)
		if member.Static {
			text.WriteString(" $")
		}
		return text.String()
	}
	text.WriteString(member.Name)
	text.WriteByte('(')
	for i, parameter := range member.Parameters {
		if i > 0 {
			text.WriteString(", ")
		}
		text.WriteString(parameter.Name)
		if parameter.Type != "" {
			text.WriteByte(' ')
			text.WriteString(parameter.Type)
		}
	}
	text.WriteByte(')')
	if member.Abstract {
		text.WriteByte('*')
	}
	if member.Static {
		text.WriteByte('$')
	}
	if member.ReturnType != "" {
		text.WriteString(": ")
		text.WriteString(member.ReturnType)
	}
	return text.String()
}

func layoutClass(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	data, err := classData(d)
	if err != nil {
		return Scene{}, err
	}
	if fonts == nil {
		return Scene{}, invalid(KindClass, 0, "fonts are required")
	}
	measured := make([]classMeasuredBox, len(data.Classes))
	items := make([]diagramGridItem, len(data.Classes))
	maxRelationWidth, maxRelationHeight, maxMultiplicityWidth := 0, 0, 0
	for i, class := range data.Classes {
		measured[i] = classMeasureBox(class, fonts)
		items[i] = diagramGridItem{w: measured[i].w, h: measured[i].h}
	}
	for _, relation := range data.Relations {
		if relation.Label != "" {
			lines, width, lineHeight := flowEdgeLabelSize(relation.Label, fonts)
			maxRelationWidth = max(maxRelationWidth, width)
			maxRelationHeight = max(maxRelationHeight, len(lines)*lineHeight)
		}
		for _, multiplicity := range []string{relation.FromMultiplicity, relation.ToMultiplicity} {
			if multiplicity != "" {
				_, width, _ := flowEdgeLabelSize(multiplicity, fonts)
				maxMultiplicityWidth = max(maxMultiplicityWidth, width)
			}
		}
	}
	columns := min(3, len(items))
	const margin = 8
	horizontalGap := max(26, maxRelationWidth+8, 2*maxMultiplicityWidth+8)
	verticalGap := max(16, maxRelationHeight+8)
	titleLines, titleWidth, titleHeight := basicTitle(d.Title, fonts)
	startY := titleHeight + 1 + margin
	boxes, gridWidth, gridHeight := diagramGridPlace(items, columns, horizontalGap, verticalGap, margin, startY)
	s := newScene()
	s.Width = max(titleWidth+2, margin+gridWidth+margin)
	s.Height = startY + gridHeight + margin
	if titleHeight > 0 {
		s.TextLines(titleLines, 1, 0, s.Width-2, titleHeight/len(titleLines), fonts.BodyPx, AlignLeft, foreground(palette), false)
	}
	indices := make(map[string]int, len(data.Classes))
	for i, class := range data.Classes {
		indices[class.ID] = i
	}
	stroke := diagramStroke(palette)
	paths := make([][]Point, len(data.Relations))
	for i, relation := range data.Relations {
		path, ok := diagramRoute(boxes[indices[relation.From]], boxes[indices[relation.To]], i, s.Width, s.Height, boxes)
		if !ok {
			return Scene{}, invalid(KindClass, 0, "unable to route class relation")
		}
		paths[i] = path
		diagramDrawRelation(&s, path, relation.Style, classDiagramMarker(relation.StartMarker), classDiagramMarker(relation.EndMarker), stroke)
	}
	for i, class := range data.Classes {
		classDrawBox(&s, class, measured[i], boxes[i], fonts, palette, i)
	}
	occupied := make([]diagramTextLabel, 0, len(data.Relations)*3)
	for i, relation := range data.Relations {
		if relation.FromMultiplicity != "" {
			label := diagramPlaceEndpointLabel(paths[i], relation.FromMultiplicity, true, -1, boxes, occupied, titleHeight+1, fonts)
			diagramDrawPlacedLabel(&s, label, fonts, palette)
			occupied = append(occupied, label)
		}
		if relation.ToMultiplicity != "" {
			label := diagramPlaceEndpointLabel(paths[i], relation.ToMultiplicity, false, -1, boxes, occupied, titleHeight+1, fonts)
			diagramDrawPlacedLabel(&s, label, fonts, palette)
			occupied = append(occupied, label)
		}
		if relation.Label != "" {
			label := diagramPlaceRelationLabel(paths[i], relation.Label, 1, boxes, occupied, titleHeight+1, fonts)
			diagramDrawPlacedLabel(&s, label, fonts, palette)
			occupied = append(occupied, label)
		}
	}
	return s, nil
}

func classDrawBox(s *Scene, class Class, measured classMeasuredBox, box diagramGridBox, fonts *rich.Fonts, palette rich.Palette, index int) {
	stroke := accent(palette, index)
	s.Rect(box.x, box.y, box.w, box.h, palette.CodeBg, stroke)
	s.Rect(box.x+1, box.y+1, box.w-2, measured.headerH-1, fade(stroke, 24), color.RGBA{})
	s.Line(box.x, box.y+measured.headerH-1, box.x+box.w-1, box.y+measured.headerH-1, rule(palette), false, false)
	headerHeight := len(measured.headerLines) * measured.lineHeight
	s.TextLines(measured.headerLines, box.x+1, box.y+(measured.headerH-headerHeight)/2, box.w-2, measured.lineHeight, fonts.BodyPx, AlignCenter, foreground(palette), false)
	y := box.y + measured.headerH
	for _, member := range measured.fields {
		s.TextLines(member.lines, box.x+2, y, box.w-4, member.lineHeight, fonts.BodyPx, AlignLeft, foreground(palette), false)
		y += len(member.lines) * member.lineHeight
	}
	if len(measured.fields) > 0 && len(measured.methods) > 0 {
		s.Line(box.x, y, box.x+box.w-1, y, rule(palette), false, false)
		y++
	}
	for _, member := range measured.methods {
		s.TextLines(member.lines, box.x+2, y, box.w-4, member.lineHeight, fonts.BodyPx, AlignLeft, foreground(palette), false)
		y += len(member.lines) * member.lineHeight
	}
}

func classDiagramMarker(marker ClassMarker) diagramEndpointMark {
	switch marker {
	case ClassMarkerTriangle:
		return diagramMarkHollowTriangle
	case ClassMarkerFilledArrow:
		return diagramMarkFilledArrow
	case ClassMarkerDiamond:
		return diagramMarkFilledDiamond
	case ClassMarkerHollowDiamond:
		return diagramMarkHollowDiamond
	default:
		return diagramMarkNone
	}
}

// The helpers below are shared by the class and ER layouts. They provide one
// deterministic grid, obstacle-aware orthogonal router, label placer and native
// endpoint-marker vocabulary without changing the shared IR or renderer.
type diagramGridItem struct{ w, h int }

type diagramGridBox struct{ x, y, w, h int }

func (b diagramGridBox) cx() int { return b.x + b.w/2 }
func (b diagramGridBox) cy() int { return b.y + b.h/2 }

func diagramGridPlace(items []diagramGridItem, columns, horizontalGap, verticalGap, startX, startY int) ([]diagramGridBox, int, int) {
	if len(items) == 0 {
		return nil, 0, 0
	}
	columns = min(max(columns, 1), len(items))
	rows := (len(items) + columns - 1) / columns
	columnWidths := make([]int, columns)
	rowHeights := make([]int, rows)
	for i, item := range items {
		columnWidths[i%columns] = max(columnWidths[i%columns], item.w)
		rowHeights[i/columns] = max(rowHeights[i/columns], item.h)
	}
	columnStarts := make([]int, columns)
	rowStarts := make([]int, rows)
	width, height := 0, 0
	for i := range columnWidths {
		columnStarts[i] = width
		width += columnWidths[i]
		if i+1 < columns {
			width += horizontalGap
		}
	}
	for i := range rowHeights {
		rowStarts[i] = height
		height += rowHeights[i]
		if i+1 < rows {
			height += verticalGap
		}
	}
	boxes := make([]diagramGridBox, len(items))
	for i := range items {
		row, column := i/columns, i%columns
		boxes[i] = diagramGridBox{
			x: startX + columnStarts[column], y: startY + rowStarts[row],
			w: columnWidths[column], h: rowHeights[row],
		}
	}
	return boxes, width, height
}

type diagramSide int

const (
	diagramSideRight diagramSide = iota
	diagramSideLeft
	diagramSideBottom
	diagramSideTop
)

func diagramRoute(from, to diagramGridBox, index, width, height int, boxes []diagramGridBox) ([]Point, bool) {
	fromSide, toSide := diagramFacingSides(from, to, index)
	start := diagramPort(from, fromSide, index)
	end := diagramPort(to, toSide, index+1)
	startOutside := diagramOffsetPoint(start, fromSide, 2)
	endOutside := diagramOffsetPoint(end, toSide, 2)
	if !diagramPointInside(startOutside, width, height) || !diagramPointInside(endOutside, width, height) {
		return nil, false
	}
	blocked := diagramBuildObstacles(boxes, width, height)
	startIndex := startOutside.Y*width + startOutside.X
	endIndex := endOutside.Y*width + endOutside.X
	blocked[startIndex], blocked[endIndex] = false, false
	previous := make([]int, width*height)
	for i := range previous {
		previous[i] = -1
	}
	queue := make([]int, 1, width*height)
	queue[0] = startIndex
	directions := [...]Point{{X: 1, Y: 0}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 0, Y: -1}}
	found := false
	for head := 0; head < len(queue) && !found; head++ {
		current := queue[head]
		if current == endIndex {
			found = true
			break
		}
		x, y := current%width, current/width
		for _, direction := range directions {
			nx, ny := x+direction.X, y+direction.Y
			if nx < 0 || ny < 0 || nx >= width || ny >= height {
				continue
			}
			next := ny*width + nx
			if blocked[next] || previous[next] != -1 {
				continue
			}
			previous[next] = current
			queue = append(queue, next)
		}
	}
	if !found {
		return nil, false
	}
	reversed := make([]Point, 0, len(queue))
	for current := endIndex; current != -1; current = previous[current] {
		reversed = append(reversed, Point{X: current % width, Y: current / width})
		if current == startIndex {
			break
		}
	}
	cells := make([]Point, len(reversed))
	for i := range reversed {
		cells[i] = reversed[len(reversed)-1-i]
	}
	cells = diagramSimplifyPath(cells)
	path := make([]Point, 0, len(cells)+2)
	path = append(path, start)
	path = append(path, cells...)
	path = append(path, end)
	return path, true
}

func diagramFacingSides(from, to diagramGridBox, index int) (diagramSide, diagramSide) {
	if from == to {
		return diagramSideRight, diagramSideBottom
	}
	dx, dy := to.cx()-from.cx(), to.cy()-from.cy()
	if abs(dx) >= abs(dy) {
		if dx >= 0 {
			return diagramSideRight, diagramSideLeft
		}
		return diagramSideLeft, diagramSideRight
	}
	if dy >= 0 {
		return diagramSideBottom, diagramSideTop
	}
	return diagramSideTop, diagramSideBottom
}

func diagramPort(box diagramGridBox, side diagramSide, index int) Point {
	length := box.h
	if side == diagramSideRight || side == diagramSideLeft {
		length = box.w
	}
	offset := diagramPortOffset(index, length)
	switch side {
	case diagramSideRight:
		return Point{X: box.x + box.w - 1, Y: box.cy() + offset}
	case diagramSideLeft:
		return Point{X: box.x, Y: box.cy() + offset}
	case diagramSideBottom:
		return Point{X: box.cx() + offset, Y: box.y + box.h - 1}
	default:
		return Point{X: box.cx() + offset, Y: box.y}
	}
}

func diagramPortOffset(index, length int) int {
	if length < 9 {
		return 0
	}
	offsets := [...]int{0, 2, -2, 4, -4}
	offset := offsets[index%len(offsets)]
	limit := length/2 - 3
	return min(max(offset, -limit), limit)
}

func diagramOffsetPoint(point Point, side diagramSide, distance int) Point {
	switch side {
	case diagramSideRight:
		point.X += distance
	case diagramSideLeft:
		point.X -= distance
	case diagramSideBottom:
		point.Y += distance
	case diagramSideTop:
		point.Y -= distance
	}
	return point
}

func diagramPointInside(point Point, width, height int) bool {
	return point.X >= 0 && point.Y >= 0 && point.X < width && point.Y < height
}

func diagramSimplifyPath(path []Point) []Point {
	out := make([]Point, 0, len(path))
	for _, point := range path {
		if len(out) > 0 && out[len(out)-1] == point {
			continue
		}
		for len(out) >= 2 {
			a, b := out[len(out)-2], out[len(out)-1]
			cross := (b.X-a.X)*(point.Y-b.Y) - (b.Y-a.Y)*(point.X-b.X)
			dot := (b.X-a.X)*(point.X-b.X) + (b.Y-a.Y)*(point.Y-b.Y)
			if cross != 0 || dot < 0 {
				break
			}
			out = out[:len(out)-1]
		}
		out = append(out, point)
	}
	return out
}

func diagramBuildObstacles(boxes []diagramGridBox, width, height int) []bool {
	blocked := make([]bool, width*height)
	for _, box := range boxes {
		for y := max(0, box.y-1); y <= min(height-1, box.y+box.h); y++ {
			for x := max(0, box.x-1); x <= min(width-1, box.x+box.w); x++ {
				blocked[y*width+x] = true
			}
		}
	}
	return blocked
}

type diagramEndpointMark int

const (
	diagramMarkNone diagramEndpointMark = iota
	diagramMarkHollowTriangle
	diagramMarkFilledArrow
	diagramMarkFilledDiamond
	diagramMarkHollowDiamond
	diagramMarkZeroOrOne
	diagramMarkExactlyOne
	diagramMarkZeroOrMore
	diagramMarkOneOrMore
)

func diagramDrawRelation(s *Scene, path []Point, style EdgeStyle, start, end diagramEndpointMark, stroke color.RGBA) {
	if len(path) < 2 {
		return
	}
	flowDrawPath(s, path, Edge{Style: style}, stroke)
	if start != diagramMarkNone {
		diagramDrawEndpointMark(s, start, path[0], path[1], stroke)
	}
	if end != diagramMarkNone {
		last := len(path) - 1
		diagramDrawEndpointMark(s, end, path[last], path[last-1], stroke)
	}
}

func diagramDrawEndpointMark(s *Scene, mark diagramEndpointMark, tip, toward Point, stroke color.RGBA) {
	ux, uy, px, py := diagramMarkFrame(tip, toward)
	point := func(along, side float64) Point {
		return Point{X: int(math.Round(float64(tip.X) + ux*along + px*side)), Y: int(math.Round(float64(tip.Y) + uy*along + py*side))}
	}
	switch mark {
	case diagramMarkHollowTriangle:
		s.Polygon([]Point{tip, point(4, 2), point(4, -2)}, color.RGBA{}, stroke)
	case diagramMarkFilledArrow:
		s.Polygon([]Point{tip, point(4, 2), point(4, -2)}, stroke, stroke)
	case diagramMarkFilledDiamond:
		s.Polygon([]Point{tip, point(2, 2), point(4, 0), point(2, -2)}, stroke, stroke)
	case diagramMarkHollowDiamond:
		s.Polygon([]Point{tip, point(2, 2), point(4, 0), point(2, -2)}, color.RGBA{}, stroke)
	case diagramMarkZeroOrOne:
		diagramDrawBar(s, point(2, 0), point(2, 1), stroke)
		s.Ellipse(point(5, 0).X-1, point(5, 0).Y-1, 3, 3, color.RGBA{}, stroke)
	case diagramMarkExactlyOne:
		diagramDrawBar(s, point(2, 0), point(2, 1), stroke)
		diagramDrawBar(s, point(4, 0), point(4, 1), stroke)
	case diagramMarkZeroOrMore, diagramMarkOneOrMore:
		s.Line(tip.X, tip.Y, point(4, 2).X, point(4, 2).Y, stroke, false, false)
		s.Line(tip.X, tip.Y, point(4, 0).X, point(4, 0).Y, stroke, false, false)
		s.Line(tip.X, tip.Y, point(4, -2).X, point(4, -2).Y, stroke, false, false)
		if mark == diagramMarkZeroOrMore {
			center := point(6, 0)
			s.Ellipse(center.X-1, center.Y-1, 3, 3, color.RGBA{}, stroke)
		} else {
			diagramDrawBar(s, point(2, 0), point(2, 1), stroke)
		}
	}
}

func diagramDrawBar(s *Scene, center, direction Point, stroke color.RGBA) {
	dx, dy := direction.X-center.X, direction.Y-center.Y
	length := math.Hypot(float64(dx), float64(dy))
	if length == 0 {
		return
	}
	px, py := -float64(dy)/length, float64(dx)/length
	s.Line(
		int(math.Round(float64(center.X)-px*2)), int(math.Round(float64(center.Y)-py*2)),
		int(math.Round(float64(center.X)+px*2)), int(math.Round(float64(center.Y)+py*2)),
		stroke, false, false,
	)
}

func diagramMarkFrame(tip, toward Point) (float64, float64, float64, float64) {
	dx, dy := toward.X-tip.X, toward.Y-tip.Y
	length := math.Hypot(float64(dx), float64(dy))
	if length == 0 {
		return 1, 0, 0, 1
	}
	ux, uy := float64(dx)/length, float64(dy)/length
	return ux, uy, -uy, ux
}

type diagramTextLabel struct {
	lines               []string
	x, y, width, height int
	lineHeight          int
}

func diagramNewTextLabel(text string, fonts *rich.Fonts) diagramTextLabel {
	lines, width, lineHeight := flowEdgeLabelSize(text, fonts)
	return diagramTextLabel{lines: lines, width: width, height: len(lines) * lineHeight, lineHeight: lineHeight}
}

func diagramPlaceEndpointLabel(path []Point, text string, start bool, prefer int, boxes []diagramGridBox, occupied []diagramTextLabel, minY int, fonts *rich.Fonts) diagramTextLabel {
	return diagramPlaceLabel(path, text, true, start, prefer, boxes, occupied, minY, fonts)
}

func diagramPlaceRelationLabel(path []Point, text string, prefer int, boxes []diagramGridBox, occupied []diagramTextLabel, minY int, fonts *rich.Fonts) diagramTextLabel {
	return diagramPlaceLabel(path, text, false, false, prefer, boxes, occupied, minY, fonts)
}

func diagramPlaceLabel(path []Point, text string, endpoint, start bool, prefer int, boxes []diagramGridBox, occupied []diagramTextLabel, minY int, fonts *rich.Fonts) diagramTextLabel {
	label := diagramNewTextLabel(text, fonts)
	segments := make([][2]Point, 0, len(path)-1)
	if endpoint {
		if start {
			segments = append(segments, [2]Point{path[0], path[1]})
		} else {
			last := len(path) - 1
			segments = append(segments, [2]Point{path[last], path[last-1]})
		}
	} else {
		for i := 1; i < len(path); i++ {
			segments = append(segments, [2]Point{path[i-1], path[i]})
		}
		sortDiagramSegments(segments)
	}
	trySide := func(side int) bool {
		for _, segment := range segments {
			a, b := segment[0], segment[1]
			for _, fraction := range []float64{0.5, 0.35, 0.65} {
				x := int(math.Round(float64(a.X) + float64(b.X-a.X)*fraction))
				y := int(math.Round(float64(a.Y) + float64(b.Y-a.Y)*fraction))
				var candidate diagramTextLabel
				candidate.lines, candidate.width, candidate.height, candidate.lineHeight = label.lines, label.width, label.height, label.lineHeight
				if a.Y == b.Y {
					candidate.x = x - label.width/2
					if side < 0 {
						candidate.y = y - label.height - 1
					} else {
						candidate.y = y + 1
					}
				} else {
					candidate.y = y - label.height/2
					if side < 0 {
						candidate.x = x - label.width - 1
					} else {
						candidate.x = x + 1
					}
				}
				candidate.x, candidate.y = max(candidate.x, 1), max(candidate.y, minY)
				if diagramLabelFree(candidate, boxes, occupied) {
					label = candidate
					return true
				}
			}
		}
		return false
	}
	if prefer < 0 {
		trySide(-1)
		trySide(1)
	} else {
		trySide(1)
		trySide(-1)
	}
	if label.x == 0 && label.y == 0 {
		point := flowLabelPoint(path, label.width, label.height)
		label.x, label.y = max(point.X, 1), max(point.Y, minY)
	}
	return label
}

func sortDiagramSegments(segments [][2]Point) {
	for i := 1; i < len(segments); i++ {
		for j := i; j > 0; j-- {
			a, b := segments[j-1], segments[j]
			aLength := abs(a[1].X-a[0].X) + abs(a[1].Y-a[0].Y)
			bLength := abs(b[1].X-b[0].X) + abs(b[1].Y-b[0].Y)
			if aLength >= bLength {
				break
			}
			segments[j-1], segments[j] = segments[j], segments[j-1]
		}
	}
}

func diagramLabelFree(label diagramTextLabel, boxes []diagramGridBox, occupied []diagramTextLabel) bool {
	for _, box := range boxes {
		if label.x < box.x+box.w && label.x+label.width > box.x && label.y < box.y+box.h && label.y+label.height > box.y {
			return false
		}
	}
	for _, other := range occupied {
		if label.x < other.x+other.width && label.x+label.width > other.x && label.y < other.y+other.height && label.y+label.height > other.y {
			return false
		}
	}
	return true
}

func diagramDrawPlacedLabel(s *Scene, label diagramTextLabel, fonts *rich.Fonts, palette rich.Palette) {
	flowDrawLabel(s, Point{X: label.x, Y: label.y}, label.lines, label.width, label.height, label.lineHeight, fonts, palette)
}

func diagramStroke(palette rich.Palette) color.RGBA {
	if palette.Link.A != 0 {
		return palette.Link
	}
	return rule(palette)
}

func diagramBlockParts(text string) (before, content string, open, closed, ok bool) {
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
		if c != '{' {
			continue
		}
		before = strings.TrimSpace(text[:i])
		rest := text[i+1:]
		depth := 1
		quote, escaped = 0, false
		for j := 0; j < len(rest); j++ {
			rc := rest[j]
			if escaped {
				escaped = false
				continue
			}
			if quote != 0 {
				if rc == '\\' {
					escaped = true
				} else if rc == quote {
					quote = 0
				}
				continue
			}
			if rc == '"' || rc == '\'' {
				quote = rc
				continue
			}
			switch rc {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					if strings.TrimSpace(rest[j+1:]) != "" {
						return "", "", false, false, false
					}
					return before, rest[:j], true, true, true
				}
			}
		}
		return before, "", true, false, true
	}
	return strings.TrimSpace(text), "", false, false, true
}

func diagramValidID(text string) bool {
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
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return true
}
