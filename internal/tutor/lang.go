package tutor

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// The tutorial exists once per language: lessons/<lang> and routes/<lang>
// in the binary, installed in its own folder (DirFor) with its own progress.
// Spanish is the original and complete one; a route another language does
// not have yet is played in Spanish inside that language's tutorial.

// referenceNotes is the note after the last lesson in each language:
// terminal commands and what cannot be practised inside the tutorial. It has
// no steps and belongs to no route, so the catalog does not list it; the
// installer still copies it.
var referenceNotes = map[i18n.Lang]string{
	i18n.ES: "Después del tutorial.md",
	i18n.EN: "After the tutorial.md",
}

// bundle is one language's tutorial: its catalog, its notes rooted at the
// vault's root, and its reference note's file name.
type bundle struct {
	cat   Catalog
	files fs.FS
	ref   string
}

// buildBundle assembles l's tutorial from lessons and routes, both holding
// one folder per language ("es/1 Básico/…", "es/b.json"). Each route l lacks
// is Spanish's, with its folder of notes; so are the reference note and the
// shared root files (mapa.png) when l has none.
func buildBundle(lessons, routes fs.FS, l i18n.Lang) (bundle, error) {
	es, err := loadRoutes(routes, string(i18n.ES))
	if err != nil {
		return bundle{}, err
	}
	esFiles, err := fs.Sub(lessons, string(i18n.ES))
	if err != nil {
		return bundle{}, err
	}
	b := bundle{cat: Catalog{Routes: es}, files: esFiles, ref: referenceNotes[i18n.ES]}
	if l != i18n.ES {
		if b, err = withFallback(lessons, routes, l, b); err != nil {
			return bundle{}, err
		}
	}
	if err := b.cat.validate(); err != nil {
		return bundle{}, err
	}
	return b, nil
}

// withFallback is l's bundle over the Spanish one, es.
func withFallback(lessons, routes fs.FS, l i18n.Lang, es bundle) (bundle, error) {
	own, err := loadRoutes(routes, string(l))
	if err != nil {
		return bundle{}, err
	}
	if len(own) == 0 {
		return es, nil // nothing translated yet: the whole tutorial is Spanish
	}
	files, err := fs.Sub(lessons, string(l))
	if err != nil {
		return bundle{}, err
	}
	has := map[string]Route{}
	for _, r := range own {
		has[r.ID] = r
	}
	borrowed := map[string]bool{} // root entries served from Spanish
	var cat Catalog
	for _, r := range es.cat.Routes {
		if mine, ok := has[r.ID]; ok {
			cat.Routes = append(cat.Routes, mine)
			delete(has, r.ID)
			continue
		}
		cat.Routes = append(cat.Routes, r)
		borrowed[r.Dir] = true
	}
	for _, r := range own { // a route Spanish lacks, in play order
		if _, extra := has[r.ID]; extra {
			cat.Routes = append(cat.Routes, r)
		}
	}
	sort.SliceStable(cat.Routes, func(i, j int) bool {
		return routeRank(cat.Routes[i].ID) < routeRank(cat.Routes[j].ID)
	})
	ref := referenceNotes[l]
	if _, err := fs.Stat(files, ref); err != nil {
		ref = es.ref
		borrowed[ref] = true
	}
	// Shared root files, such as the image the lessons embed by name.
	entries, err := fs.ReadDir(es.files, ".")
	if err != nil {
		return bundle{}, err
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if _, err := fs.Stat(files, e.Name()); errors.Is(err, fs.ErrNotExist) {
			borrowed[e.Name()] = true
		}
	}
	if len(borrowed) > 0 {
		files = overlayFS{primary: files, fallback: es.files, borrowed: borrowed}
	}
	return bundle{cat: cat, files: files, ref: ref}, nil
}

// overlayFS serves the root entries named in borrowed, and everything under
// them, from fallback, and the rest from primary.
type overlayFS struct {
	primary, fallback fs.FS
	borrowed          map[string]bool
}

// Open opens name from the tree that owns its root entry.
func (o overlayFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	return o.owner(name).Open(name)
}

// ReadDir lists name; the root lists both trees' entries, each once.
func (o overlayFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	if name != "." {
		return fs.ReadDir(o.owner(name), name)
	}
	entries, err := fs.ReadDir(o.primary, ".")
	if err != nil {
		return nil, err
	}
	var out []fs.DirEntry
	for _, e := range entries {
		if !o.borrowed[e.Name()] {
			out = append(out, e)
		}
	}
	more, err := fs.ReadDir(o.fallback, ".")
	if err != nil {
		return nil, err
	}
	for _, e := range more {
		if o.borrowed[e.Name()] {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

// owner is the tree that serves name.
func (o overlayFS) owner(name string) fs.FS {
	root, _, _ := strings.Cut(name, "/")
	if o.borrowed[root] {
		return o.fallback
	}
	return o.primary
}

// bundles caches each language's tutorial: the embed never changes, so it
// is read once per language.
var (
	bundlesMu sync.Mutex
	bundles   = map[i18n.Lang]bundleResult{}
)

type bundleResult struct {
	b   bundle
	err error
}

// bundled is l's embedded tutorial. A language Misoku does not speak reads
// as English, as the interface does.
func bundled(l i18n.Lang) (bundle, error) {
	if _, ok := referenceNotes[l]; !ok {
		l = i18n.EN
	}
	bundlesMu.Lock()
	defer bundlesMu.Unlock()
	if r, ok := bundles[l]; ok {
		return r.b, r.err
	}
	var r bundleResult
	lessonsRoot, err := fs.Sub(lessons, "lessons")
	if err == nil {
		var routesRoot fs.FS
		if routesRoot, err = fs.Sub(routesFS, "routes"); err == nil {
			r.b, r.err = buildBundle(lessonsRoot, routesRoot, l)
		}
	}
	if err != nil {
		r.err = err
	}
	bundles[l] = r
	return r.b, r.err
}

// BundledFor returns l's embedded lesson notes, rooted at the tutorial
// vault's root ("1 Básico/B01 …", "mapa.png").
func BundledFor(l i18n.Lang) fs.FS {
	b, err := bundled(l)
	if err != nil {
		panic(err) // the trees are embedded at build time and tested
	}
	return b.files
}

// BundledRoutesFor loads and validates l's routes. A bad catalog is a build
// bug, so tests call this and fail; the player gets an empty catalog from
// CatalogFor instead of a panic.
func BundledRoutesFor(l i18n.Lang) (Catalog, error) {
	b, err := bundled(l)
	return b.cat, err
}

// CatalogFor returns l's bundled catalog. Embedded data is validated by
// tests; at runtime an invalid catalog is empty rather than fatal.
func CatalogFor(l i18n.Lang) Catalog {
	b, err := bundled(l)
	if err != nil {
		return Catalog{}
	}
	return b.cat
}

// ReferenceNoteFor is the file name of l's note after the last lesson.
func ReferenceNoteFor(l i18n.Lang) string {
	b, err := bundled(l)
	if err != nil {
		return referenceNotes[i18n.ES]
	}
	return b.ref
}

// DirFor returns l's tutorial vault folder, next to Misoku's config file, or
// "" when the user config directory is unknown. Spanish keeps the folder
// every player had before ("tutor"); another language has its own sibling
// ("tutor-en"), so installing, playing or resetting it never touches the
// Spanish lessons or their progress.
func DirFor(l i18n.Lang) string {
	p := config.DefaultPath()
	if p == "" {
		return ""
	}
	if _, ok := referenceNotes[l]; !ok {
		l = i18n.EN // a language Misoku does not speak reads as English
	}
	name := "tutor"
	if l != i18n.ES {
		name += "-" + string(l)
	}
	return filepath.Join(filepath.Dir(p), name)
}
