package diagram

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

var registry = struct {
	sync.RWMutex
	parsers       map[Kind]parseFunc
	headerParsers map[Kind]headerParseFunc
	layouts       map[Kind]layoutFunc
}{
	parsers:       make(map[Kind]parseFunc),
	headerParsers: make(map[Kind]headerParseFunc),
	layouts:       make(map[Kind]layoutFunc),
}

func register(kind Kind, parse parseFunc, layout layoutFunc) {
	registry.Lock()
	defer registry.Unlock()
	if parserExistsLocked(kind) {
		panic("diagram: duplicate parser for " + string(kind))
	}
	if _, exists := registry.layouts[kind]; exists {
		panic("diagram: duplicate layout for " + string(kind))
	}
	registry.parsers[kind] = parse
	registry.layouts[kind] = layout
}

// registerWithHeader is for families whose root header carries semantic
// options, such as flowchart direction. Body-only parsers keep using register.
func registerWithHeader(kind Kind, parse headerParseFunc, layout layoutFunc) {
	registry.Lock()
	defer registry.Unlock()
	if parserExistsLocked(kind) {
		panic("diagram: duplicate parser for " + string(kind))
	}
	if _, exists := registry.layouts[kind]; exists {
		panic("diagram: duplicate layout for " + string(kind))
	}
	registry.headerParsers[kind] = parse
	registry.layouts[kind] = layout
}

func parserExistsLocked(kind Kind) bool {
	if _, exists := registry.parsers[kind]; exists {
		return true
	}
	_, exists := registry.headerParsers[kind]
	return exists
}

func unregister(kind Kind) {
	registry.Lock()
	defer registry.Unlock()
	delete(registry.parsers, kind)
	delete(registry.headerParsers, kind)
	delete(registry.layouts, kind)
}

// Parse parses Mermaid source into the shared semantic IR. Unsupported types
// and constructs return *ConstructError so the rich page can show the source.
func Parse(source string) (Diagram, error) {
	lines := prepareLines(source)
	for i, line := range lines {
		header := strings.TrimSpace(line)
		if header == "" {
			continue
		}
		kind, ok := kindForHeader(header)
		if !ok {
			fields := strings.Fields(header)
			candidate := Kind(strings.ToLower(fields[0]))
			registry.RLock()
			ok = parserExistsLocked(candidate)
			registry.RUnlock()
			kind = candidate
		}
		if !ok {
			return Diagram{}, unknown(kind, i+1, header)
		}
		registry.RLock()
		parse, exists := registry.parsers[kind]
		parseHeader, hasHeader := registry.headerParsers[kind]
		registry.RUnlock()
		if !exists && !hasHeader {
			return Diagram{}, unknown(kind, i+1, header)
		}
		var d Diagram
		var err error
		if hasHeader {
			d, err = parseHeader(header, lines[i+1:])
		} else {
			d, err = parse(lines[i+1:])
		}
		if err != nil {
			if constructErr, ok := err.(*ConstructError); ok && i > 0 {
				constructErr.Line += i
			}
			return Diagram{}, err
		}
		if d.Kind == "" {
			d.Kind = kind
		}
		return d, nil
	}
	return Diagram{}, unknown("", 1, "empty diagram")
}

func kindForHeader(header string) (Kind, bool) {
	fields := strings.Fields(header)
	if len(fields) == 0 {
		return "", false
	}
	switch strings.ToLower(fields[0]) {
	case "graph", "flowchart":
		return KindFlowchart, true
	case "sequencediagram":
		return KindSequence, true
	case "classdiagram":
		return KindClass, true
	case "statediagram", "statediagram-v2":
		return KindState, true
	case "erdiagram":
		return KindER, true
	case "pie":
		return KindPie, true
	case "xychart-beta", "xychart":
		return KindXY, true
	case "quadrantchart":
		return KindQuadrant, true
	case "gantt":
		return KindGantt, true
	case "timeline":
		return KindTimeline, true
	case "journey":
		return KindJourney, true
	case "mindmap":
		return KindMindmap, true
	case "gitgraph":
		return KindGit, true
	default:
		return "", false
	}
}

func prepareLines(source string) []string {
	source = strings.TrimPrefix(source, "\ufeff")
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	raw := strings.Split(source, "\n")
	lines := make([]string, len(raw))
	for i, line := range raw {
		lines[i] = strings.TrimRightFunc(stripMermaidComment(line), unicode.IsSpace)
	}
	return lines
}

func stripMermaidComment(line string) string {
	var quote rune
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == '%' && i+1 < len(line) && line[i+1] == '%' {
			return line[:i]
		}
	}
	return line
}

// splitStatements splits Mermaid's optional semicolon-separated statements
// without touching delimiters or quoted labels.
func splitStatements(line string) []string {
	return splitTopLevelString(line, ';')
}

func splitTopLevelString(s string, sep rune) []string {
	var out []string
	start, depth := 0, 0
	var quote rune
	escaped := false
	for i, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		default:
			if r == sep && depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		s = s[1 : len(s)-1]
	}
	return strings.NewReplacer(`\"`, `"`, `\'`, `'`, `\\`, `\`).Replace(s)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

type sourceLine struct {
	Text   string
	Number int
}

func numberedNonBlank(lines []string) []sourceLine {
	var out []sourceLine
	for i, line := range lines {
		if text := strings.TrimSpace(line); text != "" {
			out = append(out, sourceLine{Text: text, Number: i + 2})
		}
	}
	return out
}

func unknown(kind Kind, line int, construct string) error {
	construct = strings.TrimSpace(construct)
	if construct == "" {
		construct = "unknown statement"
	}
	return &ConstructError{Kind: kind, Line: line, Construct: construct}
}

func invalid(kind Kind, line int, format string, args ...any) error {
	return unknown(kind, line, fmt.Sprintf(format, args...))
}
