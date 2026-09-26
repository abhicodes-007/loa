package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/laughingmandev/loa/internal/config"
	"github.com/laughingmandev/loa/internal/index"
	"github.com/laughingmandev/loa/internal/memory"
	"github.com/laughingmandev/loa/internal/state"
	"github.com/laughingmandev/loa/internal/tools"
)

func TestWatcher_MarkStale(t *testing.T) {
	root := t.TempDir()
	st := state.NewStore(root, state.NewAgentState())
	cfgv := config.Default()
	cfg := config.NewManager(cfgv)
	idx := index.New(root)

	memStore := memory.NewStore(root, 336.0, 0.03)
	eng := New(root, st, memStore, cfg, &scriptedLLM{}, tools.New(root, nil, cfg.Get, nil), idx)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eng.StartWatcher(ctx); err != nil {
		t.Fatal(err)
	}

	// create a file
	filePath := filepath.Join(root, "test.go")
	if err := os.WriteFile(filePath, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// give watcher some time to debounce
	time.Sleep(800 * time.Millisecond)

	if !eng.hasStaleFiles() {
		t.Fatal("expected test.go to be marked stale after fsnotify event")
	}
}

func TestWatcher_RepairStaleFiles(t *testing.T) {
	root := t.TempDir()

	filePath := filepath.Join(root, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	st := state.NewStore(root, state.NewAgentState())
	cfgv := config.Default()
	cfg := config.NewManager(cfgv)
	idx := index.New(root)
	ctx := context.Background()

	if err := idx.Probe(ctx); err != nil {
		t.Skip("ctags not available")
	}
	if err := idx.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}

	memStore := memory.NewStore(root, 336.0, 0.03)
	err := memStore.AddMemory(state.MemoryItem{
		Kind:     state.MemoryStructural,
		Anchors:  []string{"main.go"},
		FileHash: "deadbeef",
		Text:     "Old frame",
	})
	if err != nil {
		t.Fatal(err)
	}

	fake := &scriptedLLM{json: []string{
		`{"frame":"New frame"}`,
		`{"index_keys":["main"],"reason":"main"}`,
	}}
	eng := New(root, st, memStore, cfg, fake, tools.New(root, nil, cfg.Get, nil), idx)

	// artificially mark stale
	eng.markStale(filePath)

	if err := eng.repairStaleFiles(ctx); err != nil {
		t.Fatal(err)
	}

	if eng.hasStaleFiles() {
		t.Fatal("expected stale files to be cleared")
	}

	items := memStore.Items()
	found := false
	for _, item := range items {
		if len(item.Anchors) > 0 && item.Anchors[0] == "main.go" {
			found = true
			if item.Text != "New frame" {
				t.Fatalf("expected New frame, got %s", item.Text)
			}
		}
	}
	if !found {
		t.Fatal("main.go not found in mem store")
	}
}
