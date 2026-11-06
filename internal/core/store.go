package core

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// mdExt is the file extension used for page content files.
const mdExt = ".md"

var (
	// ErrPageNotFound indicates the requested page ID does not exist on disk.
	ErrPageNotFound = errors.New("page not found")
)

// Store is a thin, filesystem-backed workspace: every Page is a real ".md"
// file on disk (plus an optional same-name sidecar folder holding its
// sub-pages), and every plain directory it finds is surfaced as a navigable
// folder node. There is no separate persistence format — the directory tree
// IS the source of truth, so IDs (slash-separated paths relative to the
// workspace root) are recomputed fresh from disk on every List/Get call.
type Store struct {
	mu   sync.RWMutex
	root string
}

// NewStore opens (creating if necessary) a filesystem-backed workspace rooted at dir.
func NewStore(dir string) (*Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace path: %w", err)
	}

	info, statErr := os.Stat(abs)
	switch {
	case os.IsNotExist(statErr):
		if mkErr := os.MkdirAll(abs, 0755); mkErr != nil {
			return nil, fmt.Errorf("failed to create workspace directory: %w", mkErr)
		}
	case statErr != nil:
		return nil, fmt.Errorf("failed to access workspace directory: %w", statErr)
	case !info.IsDir():
		return nil, fmt.Errorf("workspace path %q is not a directory", abs)
	}

	return &Store{root: abs}, nil
}

// Root returns the absolute workspace directory backing this store.
func (s *Store) Root() string {
	return s.root
}

func (s *Store) idToAbs(id string) string {
	return filepath.Join(s.root, filepath.FromSlash(id))
}

// parentIDFor derives the ID of id's parent page/folder purely from its path.
func parentIDFor(root, id string) string {
	dir := path.Dir(id)
	if dir == "." || dir == "/" {
		return ""
	}
	mdCandidate := dir + mdExt
	if fileExists(filepath.Join(root, filepath.FromSlash(mdCandidate))) {
		return mdCandidate
	}
	return dir
}

func fileExists(abs string) bool {
	_, err := os.Stat(abs)
	return err == nil
}

// List walks the workspace directory tree and returns a flattened list of
// pages and folders, each carrying its real ParentID so callers can rebuild
// the hierarchy. Folders are ordered before files, both alphabetically.
// A folder shows when it holds a Markdown file somewhere inside, or no file
// at all (an empty folder, as Oil lists it); one holding only other files
// (code, pictures) is left out, and so is anything starting with ".".
func (s *Store) List() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pages []Page
	s.walk("", "", &pages)
	return pages
}

// walk appends relDir's notes and folders to out and reports whether
// relDir is to be shown: it holds a Markdown file somewhere inside, or no
// file at all.
func (s *Store) walk(relDir, parentID string, out *[]Page) bool {
	absDir := filepath.Join(s.root, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return false
	}
	found, others := false, false

	mdBases := map[string]bool{}
	dirBases := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		switch {
		case e.IsDir():
			dirBases[name] = true
		case strings.HasSuffix(name, mdExt):
			mdBases[strings.TrimSuffix(name, mdExt)] = true
		default:
			others = true
		}
	}

	var expandable, leaves []string
	seen := map[string]bool{}
	for base := range mdBases {
		seen[base] = true
		if dirBases[base] {
			expandable = append(expandable, base)
		} else {
			leaves = append(leaves, base)
		}
	}
	for base := range dirBases {
		if !seen[base] {
			expandable = append(expandable, base)
		}
	}

	caseInsensitive := func(list []string) func(i, j int) bool {
		return func(i, j int) bool { return strings.ToLower(list[i]) < strings.ToLower(list[j]) }
	}
	sort.Slice(expandable, caseInsensitive(expandable))
	sort.Slice(leaves, caseInsensitive(leaves))

	for _, base := range expandable {
		relChild := path.Join(relDir, base)
		if mdBases[base] {
			relFile := relChild + mdExt
			id := relFile
			*out = append(*out, Page{
				ID:        id,
				Title:     base,
				ParentID:  parentID,
				UpdatedAt: modTime(filepath.Join(s.root, filepath.FromSlash(relFile))),
			})
			s.walk(relChild, id, out)
			found = true
		} else {
			id := relChild
			folder := Page{
				ID:        id,
				Title:     base,
				ParentID:  parentID,
				IsFolder:  true,
				UpdatedAt: modTime(filepath.Join(s.root, filepath.FromSlash(relChild))),
			}
			var children []Page
			if s.walk(relChild, id, &children) {
				*out = append(*out, folder)
				*out = append(*out, children...)
				found = true
			} else {
				others = true // a hidden folder holds other files
			}
		}
	}

	for _, base := range leaves {
		relFile := path.Join(relDir, base) + mdExt
		*out = append(*out, Page{
			ID:        relFile,
			Title:     base,
			ParentID:  parentID,
			UpdatedAt: modTime(filepath.Join(s.root, filepath.FromSlash(relFile))),
		})
		found = true
	}
	return found || !others
}

func modTime(abs string) time.Time {
	info, err := os.Stat(abs)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// Get fetches a single page (or folder) by ID, reading its content from disk.
func (s *Store) Get(id string) (Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id == "" {
		return Page{}, ErrPageNotFound
	}

	abs := s.idToAbs(id)
	info, err := os.Stat(abs)
	if err != nil {
		return Page{}, ErrPageNotFound
	}

	parentID := parentIDFor(s.root, id)
	title := strings.TrimSuffix(path.Base(id), mdExt)

	if info.IsDir() {
		return Page{ID: id, Title: title, ParentID: parentID, IsFolder: true, UpdatedAt: info.ModTime()}, nil
	}

	if !strings.HasSuffix(id, mdExt) {
		return Page{}, ErrPageNotFound
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return Page{}, ErrPageNotFound
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	return Page{ID: id, Title: title, Content: content, ParentID: parentID, UpdatedAt: info.ModTime()}, nil
}
