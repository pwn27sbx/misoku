package app

import (
	"path"
	"strings"
	"unicode"

	"github.com/pwn27sbx/misoku/internal/core"
)

// finderQuery is a finder search split into its qualifiers and the words
// left to match by name and text: "status:active tag:go folder:Zentro
// docker" is three qualifiers and "docker".
type finderQuery struct {
	// qualified: the query has a qualifier, even an empty one ("status:").
	qualified bool
	// statuses are the status: values as typed; each must match.
	statuses []string
	// tags are the tag: and #tag values, lowercased, without "#".
	tags []string
	// folders are the folder: values, split into path segments.
	folders [][]string
	// text is the rest: the whole query, as typed, when it has no
	// qualifier, so a plain search finds what it always found.
	text string
}

// parseFinderQuery reads the qualifiers of a finder query: status:<value>,
// tag:<tag> or #<tag>, and folder:<folder>, whose value may be quoted
// (folder:"My Notes"). Their names are read in any letter case; any other
// "name:value" is plain text.
func parseFinderQuery(q string) finderQuery { return parseFinderQueryHashes(q, true) }

// parseFinderQueryHashes is parseFinderQuery with "#tag" read as a qualifier
// or as plain text: the "[[" link picker reads a leading "#" as a heading.
func parseFinderQueryHashes(q string, hashes bool) finderQuery {
	q = strings.TrimSpace(q)
	var fq finderQuery
	var words []string
	for _, tok := range splitQuery(q) {
		if fq.qualifier(tok, hashes) {
			fq.qualified = true
			continue
		}
		words = append(words, tok)
	}
	fq.text = q
	if fq.qualified {
		fq.text = strings.Join(words, " ")
	}
	return fq
}

// qualifier takes tok into fq when it is a qualifier, and reports whether
// it was. An empty value ("tag:") is a qualifier that filters nothing yet.
func (fq *finderQuery) qualifier(tok string, hashes bool) bool {
	if hashes {
		if spans := core.TagSpans(tok); len(spans) == 1 && spans[0] == [2]int{0, len(tok)} {
			fq.tags = append(fq.tags, strings.ToLower(tok[1:]))
			return true
		}
	}
	key, val, ok := strings.Cut(tok, ":")
	if !ok {
		return false
	}
	val = strings.ReplaceAll(val, `"`, "")
	switch strings.ToLower(key) {
	case "status":
		fq.statuses = append(fq.statuses, val)
	case "tag":
		if t := strings.Trim(strings.ToLower(val), "#/"); t != "" {
			fq.tags = append(fq.tags, t)
		}
	case "folder":
		var segs []string
		for _, s := range strings.Split(val, "/") {
			if s != "" {
				segs = append(segs, s)
			}
		}
		if len(segs) > 0 {
			fq.folders = append(fq.folders, segs)
		}
	default:
		return false
	}
	return true
}

// splitQuery splits q at spaces outside double quotes, keeping the quotes.
func splitQuery(q string) []string {
	var out []string
	var b strings.Builder
	quoted := false
	for _, r := range q {
		switch {
		case r == '"':
			quoted = !quoted
		case unicode.IsSpace(r) && !quoted:
			if b.Len() > 0 {
				out = append(out, b.String())
				b.Reset()
			}
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

// finderMeta is what the qualifiers read of a note, worked out from its
// text once per finder.
type finderMeta struct {
	status core.NoteStatus
	tags   []string
}

// meta reads the status and tags of note id from the text the finder holds,
// as the link index reads them (core.Tags, core.ParseStatus).
func (f *finder) meta(id string) finderMeta {
	if mt, ok := f.metas[id]; ok {
		return mt
	}
	text := strings.Join(f.lines[id], "\n")
	var mt finderMeta
	if fld, ok := core.ParseFrontMatter(text).Get("status"); ok {
		mt.status = core.ParseStatus(fld.Value)
	}
	mt.tags = core.Tags(text)
	if f.metas == nil {
		f.metas = map[string]finderMeta{}
	}
	f.metas[id] = mt
	return mt
}

// passes reports whether note p passes every qualifier of fq.
func (f *finder) passes(p core.Page, fq finderQuery) bool {
	for _, folder := range fq.folders {
		if !inFolder(p.ID, folder) {
			return false
		}
	}
	if len(fq.statuses) == 0 && len(fq.tags) == 0 {
		return true
	}
	mt := f.meta(p.ID)
	for _, v := range fq.statuses {
		if v == "" {
			continue // still being typed: every note, while the hint lists the values
		}
		if want := core.ParseStatus(v); want == core.StatusNone || mt.status != want {
			return false
		}
	}
	for _, t := range fq.tags {
		if !hasTag(mt.tags, t) {
			return false
		}
	}
	return true
}

// hasTag reports whether tags holds t or a tag nested in it (t/…).
func hasTag(tags []string, t string) bool {
	for _, have := range tags {
		if have == t || strings.HasPrefix(have, t+"/") {
			return true
		}
	}
	return false
}

// inFolder reports whether note id lies in a folder whose path has the
// segments of folder in a row, in any letter case: "Zentro" is in
// "Zentro/x.md" and "Zentro/sub/x.md", not in "ZentroOld/x.md".
func inFolder(id string, folder []string) bool {
	dir := path.Dir(id)
	if dir == "." {
		return false
	}
	segs := strings.Split(dir, "/")
outer:
	for i := 0; i+len(folder) <= len(segs); i++ {
		for k, s := range folder {
			if !strings.EqualFold(segs[i+k], s) {
				continue outer
			}
		}
		return true
	}
	return false
}

// statusHint reports whether the query has a status: without a value, or
// with one Misoku doesn't know: the finder then lists the four.
func (f *finder) statusHint() bool {
	for _, v := range parseFinderQuery(f.input.Value()).statuses {
		if core.ParseStatus(v) == core.StatusNone {
			return true
		}
	}
	return false
}

// statusHintText lists the values status: takes.
const statusHintText = "status: active · onhold · completed · dropped"
