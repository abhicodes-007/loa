package agent

import (
	"context"
	"fmt"
	"hash/crc32"
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

func TestBootScan_ModifiedFile(t *testing.T) {
	root := t.TempDir()

	filePath := filepath.Join(root, "main.go")
	content := `package main
func main() {}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	st := state.NewStore(root, state.NewAgentState())
	cfgv := config.Default()
	cfg := config.NewManager(cfgv)
	idx := index.New(root)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := idx.Probe(ctx); err != nil {
		t.Skip("ctags not available")
	}
	if err := idx.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}

	memStore := memory.NewStore(root, 336.0, 0.03)

	// Pre-populate with a wrong hash to simulate a modified file
	fakeHash := "deadbeef"
	err := memStore.AddMemory(state.MemoryItem{
		Kind:     state.MemoryStructural,
		Anchors:  []string{"main.go"},
		FileHash: fakeHash,
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

	if err := eng.BootScan(ctx); err != nil {
		t.Fatal(err)
	}

	// Verify
	items := memStore.Items()
	if len(items) == 0 {
		t.Fatal("expected items, got 0")
	}

	found := false
	for _, item := range items {
		if len(item.Anchors) > 0 && item.Anchors[0] == "main.go" {
			found = true
			if item.FileHash == fakeHash {
				t.Fatalf("expected file hash to be updated, got %s", fakeHash)
			}
			expectedHash := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(content)))
			if item.FileHash != expectedHash {
				t.Fatalf("expected hash %s, got %s", expectedHash, item.FileHash)
			}
			if item.Text != "New frame" {
				t.Fatalf("expected new text, got %s", item.Text)
			}
		}
	}
	if !found {
		t.Fatal("main.go not found in mem store")
	}
}

func TestBootScan_DeletedFile(t *testing.T) {
	root := t.TempDir()

	st := state.NewStore(root, state.NewAgentState())
	cfgv := config.Default()
	cfg := config.NewManager(cfgv)
	idx := index.New(root)
	ctx := context.Background()

	// Rebuild index on empty dir
	if err := idx.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}

	memStore := memory.NewStore(root, 336.0, 0.03)

	// Add a memory for a file that doesn't exist
	err := memStore.AddMemory(state.MemoryItem{
		Kind:     state.MemoryStructural,
		Anchors:  []string{"deleted.go"},
		FileHash: "12345678",
		Text:     "Deleted frame",
	})
	if err != nil {
		t.Fatal(err)
	}

	fake := &scriptedLLM{} // Should not be called
	eng := New(root, st, memStore, cfg, fake, tools.New(root, nil, cfg.Get, nil), idx)

	if err := eng.BootScan(ctx); err != nil {
		t.Fatal(err)
	}

	items := memStore.Items()
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}
