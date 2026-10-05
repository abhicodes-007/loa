//go:build localembed

package embedder

import (
	"fmt"
	"log"
	"os"

	llama "github.com/tcpipuk/llama-go"
)

// Embedder interface defines the contract for embedding text.
type Embedder interface {
	Embed(text string) ([]float32, error)
	Close()
}

// LocalEmbedder manages the locally loaded GGUF embedding model.
type LocalEmbedder struct {
	model *llama.Model
	ctx   *llama.Context
}

// NewLocalEmbedder loads the GGUF model into memory for CPU inference.
// Requires CGO_ENABLED=1 and build tag 'localembed'.
func NewLocalEmbedder(modelPath string) (Embedder, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("local embedding model path is empty")
	}

	// Expand ~/ prefix if present to support portable configuration
	if len(modelPath) >= 2 && modelPath[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			modelPath = home + modelPath[1:]
		}
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("local embedding model file does not exist: %s", modelPath)
	}

	model, err := llama.LoadModel(modelPath)
	if err != nil {
		log.Printf("ERROR: loading embedding model from %s: %v", modelPath, err)
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	// A hardcoded 8192 buffer matches the model used during development, not
	// every GGUF embedding model — smaller trained windows fail at init (#27).
	// Prefer the context size from the model's own GGUF metadata; when that is
	// unavailable, omit WithContext so llama-go fills it from metadata.
	var ctx *llama.Context
	var err2 error
	if stats, sErr := model.Stats(); sErr == nil && resolveEmbedContextSize(stats.Metadata.ContextSize) > 0 {
		ctx, err2 = model.NewContext(
			llama.WithContext(stats.Metadata.ContextSize),
			llama.WithEmbeddings(),
		)
	} else {
		ctx, err2 = model.NewContext(llama.WithEmbeddings())
	}
	if err2 != nil {
		model.Close()
		return nil, fmt.Errorf("failed to create context: %w", err2)
	}

	return &LocalEmbedder{model: model, ctx: ctx}, nil
}

// Embed generates vector embeddings for the provided text.
func (e *LocalEmbedder) Embed(text string) ([]float32, error) {
	if e.ctx == nil {
		return nil, fmt.Errorf("embedder is not initialized")
	}

	embeddings, err := e.ctx.GetEmbeddings(text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	return embeddings, nil
}

// Close frees the C++ memory allocations.
func (e *LocalEmbedder) Close() {
	if e.ctx != nil {
		e.ctx.Close()
		e.ctx = nil
	}
	if e.model != nil {
		e.model.Close()
		e.model = nil
	}
}
