package memory

import (
	"testing"
	"time"

	"github.com/laughingmandev/loa/internal/state"
)

func TestSearchSemanticDominates(t *testing.T) {
	now := time.Now()
	items := []state.MemoryItem{
		{ID: 1, Kind: state.MemoryFact, Text: "tls cert path", Embedding: []float32{1, 0}, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: 2, Kind: state.MemoryFact, Text: "unrelated", Embedding: []float32{0, 1}, CreatedAt: now},
	}
	got := Search(items, nil, []float32{1, 0}, 2, now, nil, 2.0, 0.20, nil)
	if len(got) != 2 || got[0].ID != 1 {
		t.Fatalf("unexpected: %#v", got)
	}
}

func TestSearchHardFilter(t *testing.T) {
	now := time.Now()
	items := []state.MemoryItem{
		// Structural items (must match targets)
		{ID: 1, Kind: state.MemoryStructural, Anchors: []string{"main.go"}, Text: "main func", Embedding: []float32{1, 0}, CreatedAt: now},
		{ID: 2, Kind: state.MemoryStructural, Anchors: []string{"utils.go"}, Text: "utils func", Embedding: []float32{1, 0}, CreatedAt: now},
		// Non-structural items (should bypass filter)
		{ID: 3, Kind: state.MemoryFact, Anchors: []string{"utils.go"}, Text: "a fact", Embedding: []float32{1, 0}, CreatedAt: now},
		{ID: 4, Kind: state.MemoryGlobalConcept, Anchors: nil, Text: "a concept", Embedding: []float32{1, 0}, CreatedAt: now},
	}

	targets := []string{"main.go"}
	got := Search(items, nil, []float32{1, 0}, 10, now, targets, 2.0, 0.20, nil)

	// We expect ID 1 (matches target), ID 3 (non-structural), ID 4 (non-structural).
	// ID 2 should be filtered out.
	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d: %#v", len(got), got)
	}

	foundID2 := false
	for _, c := range got {
		if c.ID == 2 {
			foundID2 = true
		}
	}
	if foundID2 {
		t.Fatalf("expected ID 2 to be filtered out")
	}
}
