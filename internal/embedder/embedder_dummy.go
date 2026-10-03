//go:build !localembed

package embedder

import (
	"fmt"
)

// Embedder interface defines the contract for embedding text.
type Embedder interface {
	Embed(text string) ([]float32, error)
	Close()
}

// DummyEmbedder returns an error when localembed tag is missing.
type DummyEmbedder struct{}

// NewLocalEmbedder returns a dummy embedder when built without localembed tag.
func NewLocalEmbedder(modelPath string) (Embedder, error) {
	return nil, fmt.Errorf("loa was not compiled with local CPU embedding support (requires -tags localembed)")
}

// Embed returns an error.
func (e *DummyEmbedder) Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("loa was not compiled with local CPU embedding support")
}

// Close is a no-op.
func (e *DummyEmbedder) Close() {}
