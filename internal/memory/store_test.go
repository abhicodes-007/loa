package memory

import (
	"testing"

	"github.com/laughingmandev/loa/internal/state"
)

func TestStore_ClearFileMemories(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root, 336.0, 0.03)

	// Add a global concept memory
	err := store.AddMemory(state.MemoryItem{
		Kind: state.MemoryGlobalConcept,
		Text: "This is a global concept",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add a structural memory
	err = store.AddMemory(state.MemoryItem{
		Kind: state.MemoryStructural,
		Anchors: []string{"main.go"},
		Text: "This is a structural chunk",
	})
	if err != nil {
		t.Fatal(err)
	}
	
	// Add another file-anchored memory of unknown kind
	err = store.AddMemory(state.MemoryItem{
		Kind: "custom_kind",
		Anchors: []string{"utils.go"},
		Text: "This is a custom file chunk",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(store.Items()) != 3 {
		t.Fatalf("expected 3 items, got %d", len(store.Items()))
	}

	if err := store.ClearFileMemories(); err != nil {
		t.Fatal(err)
	}

	items := store.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 item to remain, got %d", len(items))
	}

	if items[0].Kind != state.MemoryGlobalConcept {
		t.Fatalf("expected remaining item to be GlobalConcept, got %s", items[0].Kind)
	}
}
