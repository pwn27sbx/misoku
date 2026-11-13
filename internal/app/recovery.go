package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// recoveryPrefix and recoverySuffix name one snapshot per session: a file
// from a session the user chose to keep is never overwritten by a later one.
const (
	recoveryPrefix = "recovery-"
	recoverySuffix = ".json"
)

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

// RecoverySnapshot is what one recovery file holds: the tabs that were dirty
// and the vault they belong to, so a file is never restored into another one.
type RecoverySnapshot struct {
	Version int        `json:"version"`
	SavedAt time.Time  `json:"saved_at"`
	Root    string     `json:"root"`
	Tabs    []Recovery `json:"tabs"`
}

// RecoveryFile is a snapshot and the file it lives in, so a restore or a
// discard removes exactly the files it acted on.
type RecoveryFile struct {
	Path string
	Snap RecoverySnapshot
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

// SaveRecovery writes the unsaved tabs to a new file in the recovery folder
// and returns the file written, or "" when nothing was dirty or recovery is
// off. A later session never overwrites an earlier snapshot: one file per
// session, so tabs the user chose to keep are still there.
//
// The file is private (0600) and written through a temporary file and a
// rename: a crash mid-write leaves the previous snapshots, never half a
// document.
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
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryMkdir), err)
	}
	if err := os.Chmod(m.recoveryDir, 0o700); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryProtectDir), err)
	}
	data, err := json.MarshalIndent(RecoverySnapshot{
		Version: recoveryVersion,
		SavedAt: time.Now().UTC(),
		Root:    m.store.Root(),
		Tabs:    tabs,
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryEncode), err)
	}

	tmp, err := os.CreateTemp(m.recoveryDir, ".recovery-*.tmp")
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryCreate), err)
	}
	tmpName := tmp.Name()
	// Best effort: after a successful rename this removes nothing, and after
	// a failure it keeps the folder free of half-written snapshots.
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryWrite), err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryFlush), err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryClose), err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryProtect), err)
	}
	final := filepath.Join(m.recoveryDir, fmt.Sprintf("%s%d-%d%s", recoveryPrefix, time.Now().UnixNano(), os.Getpid(), recoverySuffix))
	if err := renameRecovery(tmpName, final); err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryKeep), err)
	}
	return final, nil
}

// LoadRecoveries reads every snapshot in dir, oldest first. A missing folder
// is not an error; an unreadable or truncated file is reported, kept, and
// does not hide the snapshots that still parse. Temporary files a crash left
// mid-write are removed: they were never a snapshot.
func LoadRecoveries(dir string) ([]RecoveryFile, error) {
	if dir == "" {
		return nil, nil
	}
	removeStaleRecoveryTemps(dir)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []RecoveryFile
	var errs []error
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, recoveryPrefix) || !strings.HasSuffix(name, recoverySuffix) {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		var snap RecoverySnapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", i18n.T(i18n.AppRecoveryUnreadable, name), err))
			continue
		}
		if snap.Version != recoveryVersion {
			errs = append(errs, errors.New(i18n.T(i18n.AppRecoveryVersion, name, snap.Version)))
			continue
		}
		files = append(files, RecoveryFile{Path: path, Snap: snap})
	}
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Snap.SavedAt.Equal(files[j].Snap.SavedAt) {
			return files[i].Path < files[j].Path
		}
		return files[i].Snap.SavedAt.Before(files[j].Snap.SavedAt)
	})
	return files, errors.Join(errs...)
}

// RemoveRecoveryFiles deletes the snapshots restore or discard acted on.
func RemoveRecoveryFiles(files []RecoveryFile) error {
	var errs []error
	for _, f := range files {
		if f.Path == "" {
			continue
		}
		if err := os.Remove(f.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
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

// recoveryLoadedMsg is the recovery folder as read at startup.
type recoveryLoadedMsg struct {
	files []RecoveryFile
	err   error
}

// loadRecovery reads the snapshots off the update loop, at startup.
func (m *Model) loadRecovery() tea.Cmd {
	if m.recoveryDir == "" {
		return nil
	}
	dir := m.recoveryDir
	return func() tea.Msg {
		files, err := LoadRecoveries(dir)
		return recoveryLoadedMsg{files: files, err: err}
	}
}

// applyRecovery offers the unsaved tabs back, or clears snapshots with no
// tabs. A snapshot from another vault is kept and named, never restored here.
func (m *Model) applyRecovery(msg recoveryLoadedMsg) {
	var mine []RecoveryFile
	var other []string
	for _, f := range msg.files {
		if f.Snap.Root == m.store.Root() {
			mine = append(mine, f)
		} else {
			other = append(other, f.Snap.Root)
		}
	}
	if len(other) > 0 {
		m.setError(i18n.T(i18n.AppRecoveryElsewhere, strings.Join(other, ", "), m.recoveryDir))
	}
	if msg.err != nil {
		m.setError(i18n.T(i18n.AppRecoveryReadFailed, msg.err.Error()))
	}
	if len(mine) == 0 {
		return
	}
	var tabs []Recovery
	for _, f := range mine {
		tabs = append(tabs, f.Snap.Tabs...)
	}
	if len(tabs) == 0 {
		if err := RemoveRecoveryFiles(mine); err != nil {
			m.setError(i18n.T(i18n.AppRecoveryRemove, err.Error()))
		}
		return
	}
	m.askRecovery(tabs, mine)
}

// askRecovery shows the restore prompt. Discard deletes exactly the snapshots
// offered; Later (Esc) keeps them for the next start.
func (m *Model) askRecovery(tabs []Recovery, files []RecoveryFile) {
	names := make([]string, 0, len(tabs))
	for _, r := range tabs {
		names = append(names, r.Title)
	}
	list := strings.Join(names, ", ")
	if len(names) > 3 {
		list = i18n.T(i18n.AppQuitMore, strings.Join(names[:3], ", "), len(names)-3)
	}
	m.confirm = &confirmDialog{
		title: i18n.T(i18n.AppRecoveryTitle),
		message: []string{
			i18n.N(i18n.AppRecoveryTabs, len(tabs), len(tabs)),
			list,
		},
		buttons:    []string{i18n.T(i18n.AppRecoveryRestore), i18n.T(i18n.AppDiscard), i18n.T(i18n.AppRecoveryLater)},
		deliberate: true, // a stray letter must not throw them away
		selected:   0,    // Restore is the safe default
		onChoose: func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				m.restoreRecovery(tabs)
				if err := RemoveRecoveryFiles(files); err != nil {
					m.setError(i18n.T(i18n.AppRecoveryNotRemoved, err.Error()))
					return nil
				}
				m.setStatus(i18n.N(i18n.AppRecoveryRecovered, len(tabs), len(tabs)))
			case 1:
				if err := RemoveRecoveryFiles(files); err != nil {
					m.setError(i18n.T(i18n.AppRecoveryRemove, err.Error()))
					return nil
				}
				m.setStatus(i18n.T(i18n.AppRecoveryDiscarded))
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
				title = i18n.T(i18n.AppUntitled, m.draftSeq)
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
