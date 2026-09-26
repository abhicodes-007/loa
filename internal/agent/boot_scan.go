package agent

import (
	"context"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"

	"github.com/laughingmandev/loa/internal/state"
)

// BootScan checks the LTM memory anchors against the current disk state.
// It deletes memory nodes for files that have been modified or deleted,
// and synchronously triggers the crawler to heal the index.
func (e *Engine) BootScan(ctx context.Context) error {
	e.log(state.LogSystem, "BootScan", "initiating boot-time delta scan", "", "")

	idxFiles := e.index.Snapshot()

	// Get all unique file anchors currently in memory
	memFiles := make(map[string]string) // map[anchor]FileHash
	for _, item := range e.memStore.Items() {
		if item.Kind == state.MemoryStructural && len(item.Anchors) > 0 {
			anchor := item.Anchors[0]
			memFiles[anchor] = item.FileHash
		}
	}

	for anchor, storedHash := range memFiles {
		// 1. Is the file missing from ctags index?
		if _, exists := idxFiles[anchor]; !exists {
			e.log(state.LogSystem, "BootScan", "file missing from index, deleting memories", anchor, "")
			_ = e.memStore.RemoveMemoriesByAnchor(anchor)
			continue
		}

		// 2. Is the file modified on disk?
		absPath := filepath.Join(e.root, anchor)
		b, err := os.ReadFile(absPath)
		if err != nil {
			// If we can't read it, assume it's gone or inaccessible
			e.log(state.LogSystem, "BootScan", "failed to read file, deleting memories", anchor, err.Error())
			_ = e.memStore.RemoveMemoriesByAnchor(anchor)
			continue
		}

		currentHash := fmt.Sprintf("%08x", crc32.ChecksumIEEE(b))
		if currentHash != storedHash {
			e.log(state.LogSystem, "BootScan", "file modified on disk, deleting old memories", anchor, fmt.Sprintf("old: %s, new: %s", storedHash, currentHash))
			_ = e.memStore.RemoveMemoriesByAnchor(anchor)
		}
	}

	// 3. After sweeping, synchronously run the crawler to pick up new or modified files.
	// Since RunCrawler skips files that are already in memStore, it will only process
	// the files we just deleted (modified) and any brand new files discovered in ctags.
	if err := e.RunCrawler(ctx); err != nil {
		e.log(state.LogSystem, "BootScan", "crawler failed during boot scan", "", err.Error())
		return err
	}

	e.log(state.LogSystem, "BootScan", "boot-time delta scan completed successfully", "", "")
	return nil
}
