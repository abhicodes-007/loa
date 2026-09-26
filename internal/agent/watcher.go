package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/laughingmandev/loa/internal/state"
)

func (e *Engine) hasStaleFiles() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.staleFiles) > 0
}

func (e *Engine) markStale(absPath string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.staleFiles == nil {
		e.staleFiles = make(map[string]bool)
	}

	rel, err := filepath.Rel(e.root, absPath)
	if err != nil {
		return
	}

	// Avoid indexing hidden files or loa internal files
	if strings.HasPrefix(rel, ".") || strings.Contains(rel, string(filepath.Separator)+".") {
		return
	}

	e.staleFiles[rel] = true
}

func (e *Engine) repairStaleFiles(ctx context.Context) error {
	e.mu.Lock()
	if len(e.staleFiles) == 0 {
		e.mu.Unlock()
		return nil
	}

	files := make([]string, 0, len(e.staleFiles))
	for f := range e.staleFiles {
		files = append(files, f)
	}
	// Clear the set
	e.staleFiles = make(map[string]bool)
	e.mu.Unlock()

	e.log(state.LogSystem, "Watcher", "Just-In-Time reconciliation triggered for stale files", strings.Join(files, ", "), "")

	for _, file := range files {
		_ = e.memStore.RemoveMemoriesByAnchor(file)
	}

	// Refresh CTAGS index synchronously
	if err := e.index.Rebuild(ctx); err != nil {
		e.log(state.LogSystem, "Watcher", "failed to rebuild index during repair", "", err.Error())
	}

	// RunCrawler seamlessly re-indexes the now missing (stale) files
	return e.RunCrawler(ctx)
}

// StartWatcher initializes a recursive directory watcher and starts
// an event listener that flags modified files for healing.
func (e *Engine) StartWatcher(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = filepath.Walk(e.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			rel, _ := filepath.Rel(e.root, path)
			if rel != "." && (strings.HasPrefix(rel, ".") || strings.Contains(rel, string(filepath.Separator)+".")) {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		watcher.Close()
		return err
	}

	go func() {
		defer watcher.Close()

		// Debounce timer
		var timer *time.Timer
		var timerMu sync.Mutex
		events := make(map[string]bool)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					timerMu.Lock()
					events[event.Name] = true
					if timer != nil {
						timer.Stop()
					}
					debounceMs := e.cfg.Get().FSWatcherDebounceMs
					timer = time.AfterFunc(time.Duration(debounceMs)*time.Millisecond, func() {
						timerMu.Lock()
						pending := make([]string, 0, len(events))
						for f := range events {
							pending = append(pending, f)
						}
						// Clear the batch
						for k := range events {
							delete(events, k)
						}
						timerMu.Unlock()

						for _, f := range pending {
							e.markStale(f)
						}
						e.log(state.LogSystem, "Watcher", "fsnotify event triggered cache invalidation", strings.Join(pending, ", "), "")
					})
					timerMu.Unlock()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				e.log(state.LogSystem, "Watcher", "fsnotify error", "", err.Error())
			}
		}
	}()

	e.log(state.LogSystem, "Watcher", "started OS file watcher", "", "")
	return nil
}
