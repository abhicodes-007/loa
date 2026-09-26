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

func TestCrawlerSynchronousExecution(t *testing.T) {
	root := t.TempDir()

	// Write a dummy file to be indexed
	filePath := filepath.Join(root, "main.go")
	content := `package main
import "fmt"
func main() { fmt.Println("hello") }
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfgv := config.Default()
	cfg := config.NewManager(cfgv)
	st := state.NewStore(root, state.NewAgentState())
	idx := index.New(root)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := idx.Probe(ctx); err != nil {
		t.Skip("ctags not available, skipping test")
	}
	if err := idx.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}

	fake := &scriptedLLM{json: []string{
		// 1. Narrative Frame
		`{"frame":"This is the main entry point"}`,
		// 2. Extract Keywords
		`{"index_keys":["main", "entry"],"reason":"main func"}`,
	}}

	memStore := memory.NewStore(root, 336.0, 0.03)
	eng := New(root, st, memStore, cfg, fake, tools.New(root, nil, cfg.Get, nil), idx)

	err := eng.RunCrawler(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Verify LTM
	items := memStore.Items()
	if len(items) == 0 {
		t.Fatal("expected LTM to be populated with structural memory, got 0")
	}

	found := false
	for _, item := range items {
		if item.Kind == state.MemoryStructural && len(item.Anchors) > 0 && item.Anchors[0] == "main.go" {
			found = true
			if item.Text != "This is the main entry point" {
				t.Errorf("unexpected frame text: %s", item.Text)
			}
			if len(item.IndexKeys) != 2 || item.IndexKeys[0] != "main" {
				t.Errorf("unexpected index keys: %v", item.IndexKeys)
			}
			break
		}
	}

	if !found {
		t.Errorf("MemoryStructural node for main.go not found in LTM")
	}
}
