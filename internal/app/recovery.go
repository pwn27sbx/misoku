package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// recoveryFileName is the one snapshot kept in a recovery folder. It is
// written when the terminal closes or the process is asked to stop with
// unsaved tabs, and removed once the user restores or discards it.
const recoveryFileName = "recovery.json"

// recoveryVersion is the snapshot format; a file from another version is
// reported instead of guessed at.
const recoveryVersion = 1

// renameRecovery is os.Rename, replaceable in tests to simulate a crash
// between the temporary write and the rename.
var renameRecovery = os.Rename

// Recovery is one unsaved tab from a session that ended before it was saved.
type Recovery struct {
	ID      string `json:"id,omitempty"` // note ID; empty for a draft
	Title   string `json:"title"`
	Dir     string `json:"dir,omitempty"` // draft folder offered by Save As
	Content string `json:"content"`
}

// RecoverySnapshot is what a recovery file holds: the tabs that were dirty
// and the vault they belong to, so a file is never restored into another one.
type RecoverySnapshot struct {
	Version int        `json:"version"`
	SavedAt time.Time  `json:"saved_at"`
	Root    string     `json:"root"`
	Tabs    []Recovery `json:"tabs"`
}

// RecoveryDir is the folder where a session keeps its unsaved tabs, or ""
// when the system has no cache folder. Never the vault and never a relative
// path: the file holds note text and must not land with the notes.
func RecoveryDir() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cache, "misoku", "recovery")
}

// SaveRecovery writes the unsaved tabs to the recovery folder and returns the
// file written, or "" when nothing was dirty or recovery is off. The file is
// private (0600) and written through a temporary file and a rename: a crash
// mid-write leaves the previous snapshot, never half a document.
func (m Model) SaveRecovery() (string, error) {
	if m.recoveryDir == "" {
		return "", nil
	}
	var tabs []Recovery
	for _, b := range m.buffers {
		if b.isGraph() || !m.isDirty(b) {
			continue
		}
		r := Recovery{Title: b.title, Dir: b.dir, Content: m.liveText(b)}
		if !b.draft() {
			r.ID = b.id
		}
		tabs = append(tabs, r)
	}
	if len(tabs) == 0 {
		return "", nil
	}

	if err := os.MkdirAll(m.recoveryDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create the recovery folder: %w", err)
	}
	if err := os.Chmod(m.recoveryDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to protect the recovery folder: %w", err)
	}
	data, err := json.MarshalIndent(RecoverySnapshot{
		Version: recoveryVersion,
		SavedAt: time.Now().UTC(),
		Root:    m.store.Root(),
		Tabs:    tabs,
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode the recovery file: %w", err)
	}

	path := filepath.Join(m.recoveryDir, recoveryFileName)
	tmp, err := os.CreateTemp(m.recoveryDir, ".recovery-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create the recovery file: %w", err)
	}
	tmpName := tmp.Name()
	// Best effort: after a successful rename this removes nothing, and after
	// a failure it keeps the folder free of half-written snapshots.
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("failed to write the recovery file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("failed to flush the recovery file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("failed to close the recovery file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return "", fmt.Errorf("failed to protect the recovery file: %w", err)
	}
	if err := renameRecovery(tmpName, path); err != nil {
		return "", fmt.Errorf("failed to keep the recovery file: %w", err)
	}
	return path, nil
}

// LoadRecovery reads the snapshot in dir. A missing file is not an error; a
// file from another version or a truncated one is reported and left alone.
// Temporary files a crash left mid-write are removed: they were never the
// snapshot.
func LoadRecovery(dir string) (RecoverySnapshot, error) {
	if dir == "" {
		return RecoverySnapshot{}, nil
	}
	removeStaleRecoveryTemps(dir)
	data, err := os.ReadFile(filepath.Join(dir, recoveryFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return RecoverySnapshot{}, nil
	}
	if err != nil {
		return RecoverySnapshot{}, err
	}
	var snap RecoverySnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return RecoverySnapshot{}, fmt.Errorf("the recovery file is unreadable: %w", err)
	}
	if snap.Version != recoveryVersion {
		return RecoverySnapshot{}, fmt.Errorf("the recovery file has version %d", snap.Version)
	}
	return snap, nil
}

// removeStaleRecoveryTemps deletes the .recovery-*.tmp files a crash left
// behind. Best effort: a file that cannot be removed is simply ignored.
func removeStaleRecoveryTemps(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".recovery-") && strings.HasSuffix(name, ".tmp") {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

// RemoveRecovery deletes the snapshot in dir, if there is one.
func RemoveRecovery(dir string) error {
	if dir == "" {
		return nil
	}
	err := os.Remove(filepath.Join(dir, recoveryFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// recoveryLoadedMsg is the recovery file as read at startup.
type recoveryLoadedMsg struct {
	snap RecoverySnapshot
	err  error
}

// loadRecovery reads the snapshot off the update loop, at startup.
func (m *Model) loadRecovery() tea.Cmd {
	if m.recoveryDir == "" {
		return nil
	}
	dir := m.recoveryDir
	return func() tea.Msg {
		snap, err := LoadRecovery(dir)
		return recoveryLoadedMsg{snap: snap, err: err}
	}
}

// applyRecovery offers the unsaved tabs back, or clears a stale file. A
// snapshot from another vault is kept and named, never restored here.
func (m *Model) applyRecovery(msg recoveryLoadedMsg) {
	if msg.err != nil {
		m.setError("Recovery file could not be read: " + msg.err.Error())
		return
	}
	if len(msg.snap.Tabs) == 0 {
		if err := RemoveRecovery(m.recoveryDir); err != nil {
			m.setError("Could not remove the recovery file: " + err.Error())
		}
		return
	}
	if msg.snap.Root != m.store.Root() {
		m.setError("Unsaved notes from " + msg.snap.Root + " are kept in " + m.recoveryDir)
		return
	}
	m.askRecovery(msg.snap.Tabs)
}

// askRecovery shows the restore prompt. Discard deletes the file; Later
// (Esc) keeps it for the next start.
func (m *Model) askRecovery(tabs []Recovery) {
	names := make([]string, 0, len(tabs))
	for _, r := range tabs {
		names = append(names, r.Title)
	}
	list := strings.Join(names, ", ")
	if len(names) > 3 {
		list = strings.Join(names[:3], ", ") + fmt.Sprintf(" and %d more", len(names)-3)
	}
	m.confirm = &confirmDialog{
		title: "Recover unsaved notes?",
		message: []string{
			fmt.Sprintf("%d tab(s) were not saved when Misoku closed.", len(tabs)),
			list,
		},
		buttons:    []string{"Restore", "Discard", "Later"},
		deliberate: true, // a stray letter must not throw them away
		selected:   0,    // Restore is the safe default
		onChoose: func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				m.restoreRecovery(tabs)
				if err := RemoveRecovery(m.recoveryDir); err != nil {
					m.setError("Recovered, but the file could not be removed: " + err.Error())
					return nil
				}
				m.setStatus(fmt.Sprintf("Recovered %d unsaved tab(s)", len(tabs)))
			case 1:
				if err := RemoveRecovery(m.recoveryDir); err != nil {
					m.setError("Could not remove the recovery file: " + err.Error())
					return nil
				}
				m.setStatus("Recovery discarded")
			}
			return nil
		},
	}
}

// restoreRecovery opens the recovered tabs. Nothing is written to disk: a
// note comes back as a modified tab against what is on disk (or as a draft
// when the file is gone), and the user decides whether to save it.
func (m *Model) restoreRecovery(tabs []Recovery) {
	var first *buffer
	for _, r := range tabs {
		var b *buffer
		if r.ID != "" {
			saved := ""
			if p, err := m.store.Get(r.ID); err == nil && !p.IsFolder {
				saved = p.Content
			}
			title := r.Title
			if title == "" {
				title = strings.TrimSuffix(path.Base(r.ID), ".md")
			}
			b = &buffer{id: r.ID, title: title, saved: saved, text: r.Content}
			if i := m.bufferIndex(r.ID); i >= 0 {
				m.buffers[i] = b
			} else {
				m.buffers = append(m.buffers, b)
			}
		} else {
			m.draftSeq++
			title := r.Title
			if title == "" {
				title = fmt.Sprintf("Untitled-%d", m.draftSeq)
			}
			b = &buffer{id: fmt.Sprintf("%s%d", draftIDPrefix, m.draftSeq), title: title, dir: r.Dir, text: r.Content}
			m.buffers = append(m.buffers, b)
		}
		if first == nil {
			first = b
		}
	}
	if first == nil {
		return
	}
	m.active = ""
	m.showBuffer(first)
	m.refreshModified()
}
