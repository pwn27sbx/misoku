package core

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
)

// Move moves the note or folder id under destParentID ("" for the workspace
// root, a folder, or a note, whose sidecar folder then holds it). A note
// takes its sidecar folder of sub-notes with it; a folder takes everything
// inside it. The name is kept.
//
// It refuses, before writing anything: the workspace root, IDs that are
// absolute, hidden or contain "." or "..", a destination that is the moved
// entry itself or lies inside it, a destination already holding the name (as
// a note or as a folder, since either would merge with the moved entry), and
// a move to the folder the entry is already in. If a rename fails halfway the
// note is put back where it was, so the error leaves the disk as it found it
// as far as the system allows. It returns the page as it now exists on disk.
func (s *Store) Move(id, destParentID string) (Page, error) {
	if id == "" {
		return Page{}, errors.New("the workspace root cannot be moved")
	}
	if err := checkMovePath(id); err != nil {
		return Page{}, err
	}
	if err := checkMovePath(destParentID); err != nil {
		return Page{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	info, err := os.Stat(s.joinID(id))
	if err != nil {
		return Page{}, ErrPageNotFound
	}
	isFolder := info.IsDir()
	if !isFolder && !strings.HasSuffix(id, mdExt) {
		return Page{}, ErrPageNotFound
	}

	destDir, err := s.childDirFor(destParentID)
	if err != nil {
		return Page{}, err
	}

	// srcDir is the folder that moves: the folder itself, or the note's
	// sidecar folder (which may not exist).
	srcDir := strings.TrimSuffix(id, mdExt)
	if destDir == srcDir || strings.HasPrefix(destDir, srcDir+"/") {
		return Page{}, fmt.Errorf("cannot move %q into itself", id)
	}
	if fromDir := path.Dir(id); fromDir == destDir || (fromDir == "." && destDir == "") {
		return Page{}, fmt.Errorf("%q is already in that folder", id)
	}

	name := path.Base(srcDir)
	newDir := path.Join(destDir, name)
	newNote := newDir + mdExt
	if fileExists(s.joinID(newNote)) || fileExists(s.joinID(newDir)) {
		return Page{}, fmt.Errorf("%q already exists in the destination", name)
	}

	absDest := s.joinID(destDir)
	createdDest := !fileExists(absDest)
	if err := os.MkdirAll(absDest, 0755); err != nil {
		return Page{}, fmt.Errorf("failed to create folder: %w", err)
	}
	undoDest := func() {
		if createdDest {
			_ = os.Remove(absDest)
		}
	}

	if isFolder {
		if err := os.Rename(s.joinID(id), s.joinID(newDir)); err != nil {
			undoDest()
			return Page{}, fmt.Errorf("failed to move folder: %w", err)
		}
		return Page{ID: newDir, Title: name, ParentID: parentIDFor(s.root, newDir), IsFolder: true, UpdatedAt: modTime(s.joinID(newDir))}, nil
	}

	if err := os.Rename(s.joinID(id), s.joinID(newNote)); err != nil {
		undoDest()
		return Page{}, fmt.Errorf("failed to move page: %w", err)
	}
	if sInfo, err := os.Stat(s.joinID(srcDir)); err == nil && sInfo.IsDir() {
		if err := os.Rename(s.joinID(srcDir), s.joinID(newDir)); err != nil {
			// Put the note back next to its sub-notes rather than split them.
			if backErr := os.Rename(s.joinID(newNote), s.joinID(id)); backErr == nil {
				undoDest()
				return Page{}, fmt.Errorf("failed to move sub-pages, nothing moved: %w", err)
			}
			return Page{}, fmt.Errorf("moved the page to %s but not its sub-pages folder: %w", newNote, err)
		}
	}

	data, err := os.ReadFile(s.joinID(newNote))
	if err != nil {
		return Page{}, fmt.Errorf("moved the page but could not read it back: %w", err)
	}
	return Page{
		ID:        newNote,
		Title:     name,
		Content:   strings.ReplaceAll(string(data), "\r\n", "\n"),
		ParentID:  parentIDFor(s.root, newNote),
		UpdatedAt: modTime(s.joinID(newNote)),
	}, nil
}

// checkMovePath refuses an ID that could leave the workspace or name a
// hidden entry, through the same checks ParseNewEntry uses (checkRelative,
// checkPathSegment); a "\" inside a name is refused too, since Windows would
// read it as a separator.
func checkMovePath(id string) error {
	if id == "" {
		return nil
	}
	if err := checkRelative(id, "workspace"); err != nil {
		return err
	}
	for _, seg := range strings.Split(id, "/") {
		if err := checkPathSegment(seg); err != nil {
			return err
		}
		if strings.Contains(seg, `\`) {
			return errors.New(`"\" is not allowed in a name`)
		}
	}
	return nil
}
