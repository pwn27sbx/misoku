package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

func benchVaultDir(b *testing.B) string {
	b.Helper()
	if dir := os.Getenv("MISOKU_BENCH_VAULT"); dir != "" {
		return dir
	}
	dir := b.TempDir()
	generateSyntheticVault(b, dir, 5000)
	return dir
}

func generateSyntheticVault(tb testing.TB, root string, count int) {
	tb.Helper()
	statuses := []string{"active", "onhold", "completed", "dropped"}
	folders := 50
	for f := 0; f < folders; f++ {
		if err := os.MkdirAll(filepath.Join(root, fmt.Sprintf("folder-%02d", f)), 0o755); err != nil {
			tb.Fatal(err)
		}
	}
	filler := strings.Repeat("lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua ", 10)
	for i := 0; i < count; i++ {
		f := i % folders
		status := statuses[i%len(statuses)]
		next1 := (i + 1) % count
		next2 := (i + 7) % count
		next3 := (i + 13) % count
		next4 := (i + 29) % count
		next5 := (i + 53) % count
		mentionTarget := fmt.Sprintf("Note %04d", (i+3)%count)
		body := fmt.Sprintf(`---
status: %s
tags: [topic-%02d, group-%02d, bench]
---
# Note %04d

This note discusses %s in plain prose without linking it.
Related links: [[Note %04d]], [[Note %04d]], [[Note %04d]], [[Note %04d]], [[Note %04d]].
Inline #tag%02d and #synth/bench.

%s
`, status, i%20, f%10, i, mentionTarget, next1, next2, next3, next4, next5, i%15, filler)
		p := filepath.Join(root, fmt.Sprintf("folder-%02d", f), fmt.Sprintf("Note %04d.md", i))
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
}

func largestNoteID(b *testing.B, store *core.Store) string {
	b.Helper()
	var bestID string
	bestSize := -1
	for _, p := range store.List() {
		if p.IsFolder {
			continue
		}
		full, err := store.Get(p.ID)
		if err != nil {
			continue
		}
		if len(full.Content) > bestSize {
			bestSize = len(full.Content)
			bestID = p.ID
		}
	}
	if bestID == "" {
		b.Fatal("bench vault has no notes")
	}
	return bestID
}

func BenchmarkVaultStartup(b *testing.B) {
	dir := benchVaultDir(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store, err := core.NewStore(dir)
		if err != nil {
			b.Fatal(err)
		}
		m := New(store)
		next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
		m = next.(Model)
		idx := core.NewLinkIndex(preview.IsImageName)
		idx.Update(store.List(), func(id string) (string, error) {
			p, err := store.Get(id)
			return p.Content, err
		})
		_ = m.View()
	}
}

func BenchmarkVaultOpenNote(b *testing.B) {
	dir := benchVaultDir(b)
	store, err := core.NewStore(dir)
	if err != nil {
		b.Fatal(err)
	}
	target := largestNoteID(b, store)
	m := New(store)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	m = next.(Model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.buffers = nil
		m.active = ""
		m.openInCurrentBuffer(target)
		_ = m.View()
	}
}

func BenchmarkVaultFinderOpenAndSearch(b *testing.B) {
	dir := benchVaultDir(b)
	store, err := core.NewStore(dir)
	if err != nil {
		b.Fatal(err)
	}
	m := New(store)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	m = next.(Model)
	if cmd := m.startLinkIndex(); cmd != nil {
		msg := cmd()
		next, _ = m.Update(msg)
		m = next.(Model)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.finder = nil
		cmd := m.openFinder()
		_ = cmd
		m.finder.input.SetValue("note")
		m.finder.refilter()
	}
}

func BenchmarkVaultSaveAndMentions(b *testing.B) {
	dir := benchVaultDir(b)
	// Copy to temp dir so saves never mutate MISOKU_BENCH_VAULT.
	work := b.TempDir()
	if err := copyDirForBench(dir, work); err != nil {
		b.Fatal(err)
	}
	store, err := core.NewStore(work)
	if err != nil {
		b.Fatal(err)
	}
	target := largestNoteID(b, store)
	m := New(store)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	m = next.(Model)
	m.openInCurrentBuffer(target)
	// Warm initial LinkIndex and mentions.
	m.refreshGraph()
	if cmd := m.startLinkIndex(); cmd != nil {
		m.applyLinkIndex(cmd().(linkIndexedMsg))
	}
	if cmd := m.startMentionScan(); cmd != nil {
		m.applyMentions(cmd().(mentionsScannedMsg))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cur := m.activeBuffer()
		if cur == nil {
			b.Fatal("no active buffer")
		}
		m.writeBuffer(cur, nil)
		if cmd := m.startLinkIndex(); cmd != nil {
			m.applyLinkIndex(cmd().(linkIndexedMsg))
		}
		if cmd := m.startMentionScan(); cmd != nil {
			m.applyMentions(cmd().(mentionsScannedMsg))
		}
	}
}

func copyDirForBench(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
