package core

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// childDirFor resolves the relative directory new children of parentID should
// be written into: the workspace root, an existing plain folder, or the
// sidecar folder belonging to a page.
func (s *Store) childDirFor(parentID string) (string, error) {
	if parentID == "" {
		return "", nil
	}

	abs := s.idToAbs(parentID)
	info, err := os.Stat(abs)
	if err != nil {
		return "", ErrPageNotFound
	}
	if info.IsDir() {
		return parentID, nil
	}
	if !strings.HasSuffix(parentID, mdExt) {
		return "", ErrPageNotFound
	}
	return strings.TrimSuffix(parentID, mdExt), nil
}

// sanitizeTitle strips path separators and surrounding whitespace so a title
// can never escape its intended directory.
func sanitizeTitle(title string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r < 0x20:
			return -1
		case strings.ContainsRune(invalidNameChars, r):
			return '-'
		}
		return r
	}, title)
	cleaned = strings.TrimRight(strings.TrimSpace(cleaned), ". ")
	if cleaned == "" {
		cleaned = "Untitled"
	}
	if reservedName(cleaned) {
		cleaned += "-note"
	}
	return cleaned
}

// invalidNameChars can't appear in file names on Windows (and / nowhere);
// rejecting them everywhere keeps notes portable between machines.
const invalidNameChars = `/\:*?"<>|`

// reservedName reports Windows device names (CON, NUL, COM1, …), which can't
// be used as file names even with an extension.
func reservedName(name string) bool {
	base := strings.ToUpper(strings.TrimSuffix(name, mdExt))
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return true
	}
	return false
}

// writeText writes a note, keeping Windows (CRLF) line endings if the file
// already used them; notes are always edited with plain \n.
func writeText(abs, content string) error {
	if old, err := os.ReadFile(abs); err == nil && strings.Contains(string(old), "\r\n") {
		content = strings.ReplaceAll(content, "\n", "\r\n")
	}
	return writeFileAtomic(abs, []byte(content))
}

func nextUntitled(absDir string) string {
	entries, _ := os.ReadDir(absDir)
	existing := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() {
			name = strings.TrimSuffix(name, mdExt)
		}
		existing[name] = true
	}
	if !existing["Untitled"] {
		return "Untitled"
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("Untitled %d", i)
		if !existing[cand] {
			return cand
		}
	}
}

// Create writes a new page file to disk under parentID ("" for the workspace
// root) and returns the resulting Page. An empty title generates "Untitled"
// (or "Untitled N" if that's taken); a colliding title is disambiguated the
// same way.
func (s *Store) Create(title, parentID string) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	relDir, err := s.childDirFor(parentID)
	if err != nil {
		return Page{}, err
	}

	absDir := filepath.Join(s.root, filepath.FromSlash(relDir))
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return Page{}, fmt.Errorf("failed to create folder: %w", err)
	}

	title = strings.TrimSpace(title)
	if title == "" {
		title = nextUntitled(absDir)
	} else {
		title = sanitizeTitle(title)
	}

	finalTitle := title
	absFile := filepath.Join(absDir, finalTitle+mdExt)
	for n := 2; fileExists(absFile); n++ {
		finalTitle = fmt.Sprintf("%s %d", title, n)
		absFile = filepath.Join(absDir, finalTitle+mdExt)
	}

	if err := os.WriteFile(absFile, []byte(""), 0644); err != nil {
		return Page{}, fmt.Errorf("failed to create page: %w", err)
	}

	id := path.Join(relDir, finalTitle+mdExt)
	return Page{ID: id, Title: finalTitle, ParentID: parentID, UpdatedAt: modTime(absFile)}, nil
}

// Update persists p.Content to disk and, if p.Title no longer matches the
// page's current filename, renames the file (and its sidecar sub-pages
// folder, if any) to match. It returns the page as it now exists on disk,
// since a rename changes its ID.
func (s *Store) Update(p Page) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	abs := s.idToAbs(p.ID)
	info, err := os.Stat(abs)
	if err != nil {
		return Page{}, ErrPageNotFound
	}

	if info.IsDir() {
		return s.renameFolder(p, abs)
	}

	currentTitle := strings.TrimSuffix(path.Base(p.ID), mdExt)
	newTitle := strings.TrimSpace(p.Title)
	if newTitle == "" {
		newTitle = currentTitle
	} else {
		newTitle = sanitizeTitle(newTitle)
	}

	id := p.ID
	if newTitle != currentTitle {
		dir := path.Dir(p.ID)
		if dir == "." {
			dir = ""
		}
		absDir := filepath.Join(s.root, filepath.FromSlash(dir))
		newAbs := filepath.Join(absDir, newTitle+mdExt)

		if fileExists(newAbs) {
			return Page{}, fmt.Errorf("a page named %q already exists here", newTitle)
		}
		if err := os.Rename(abs, newAbs); err != nil {
			return Page{}, fmt.Errorf("failed to rename page: %w", err)
		}

		oldSidecar := strings.TrimSuffix(abs, mdExt)
		if info, err := os.Stat(oldSidecar); err == nil && info.IsDir() {
			newSidecar := strings.TrimSuffix(newAbs, mdExt)
			if err := os.Rename(oldSidecar, newSidecar); err != nil {
				return Page{}, fmt.Errorf("failed to rename sub-pages: %w", err)
			}
		}

		id = path.Join(dir, newTitle+mdExt)
		abs = newAbs
	}

	if err := writeText(abs, p.Content); err != nil {
		return Page{}, fmt.Errorf("failed to save page: %w", err)
	}

	parentID := parentIDFor(s.root, id)
	return Page{ID: id, Title: newTitle, Content: p.Content, ParentID: parentID, UpdatedAt: modTime(abs)}, nil
}

// renameFolder handles Update for a plain folder node (no backing .md file,
// so there's no content to persist — only a possible rename). Caller must
// already hold s.mu.
func (s *Store) renameFolder(p Page, abs string) (Page, error) {
	currentTitle := path.Base(p.ID)
	newTitle := strings.TrimSpace(p.Title)
	if newTitle == "" {
		newTitle = currentTitle
	} else {
		newTitle = sanitizeTitle(newTitle)
	}

	id := p.ID
	if newTitle != currentTitle {
		dir := path.Dir(p.ID)
		if dir == "." {
			dir = ""
		}
		absDir := filepath.Join(s.root, filepath.FromSlash(dir))
		newAbs := filepath.Join(absDir, newTitle)

		if fileExists(newAbs) {
			return Page{}, fmt.Errorf("a folder named %q already exists here", newTitle)
		}
		if err := os.Rename(abs, newAbs); err != nil {
			return Page{}, fmt.Errorf("failed to rename folder: %w", err)
		}

		id = path.Join(dir, newTitle)
		abs = newAbs
	}

	parentID := parentIDFor(s.root, id)
	return Page{ID: id, Title: newTitle, ParentID: parentID, IsFolder: true, UpdatedAt: modTime(abs)}, nil
}

// Delete removes a page (or folder) by ID. Deleting a page cascades to its
// sidecar sub-pages folder, if any; deleting a folder removes everything
// nested inside it.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	abs := s.idToAbs(id)
	info, err := os.Stat(abs)
	if err != nil {
		return ErrPageNotFound
	}

	if info.IsDir() {
		if err := os.RemoveAll(abs); err != nil {
			return fmt.Errorf("failed to delete folder: %w", err)
		}
		return nil
	}

	if err := os.Remove(abs); err != nil {
		return fmt.Errorf("failed to delete page: %w", err)
	}

	sidecar := strings.TrimSuffix(abs, mdExt)
	if sInfo, err := os.Stat(sidecar); err == nil && sInfo.IsDir() {
		_ = os.RemoveAll(sidecar)
	}
	return nil
}
