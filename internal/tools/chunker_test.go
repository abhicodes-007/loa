package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChunkFileByLines(t *testing.T) {
	// Create a temporary file with 10 lines
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\nline 10"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	chunks, err := ChunkFileByLines(path, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}

	if chunks[0].StartLine != 1 || chunks[0].EndLine != 3 {
		t.Errorf("chunk 0 mismatch: %+v", chunks[0])
	}
	if chunks[3].StartLine != 10 || chunks[3].EndLine != 10 {
		t.Errorf("chunk 3 mismatch: %+v", chunks[3])
	}
	if chunks[3].Content != "line 10" {
		t.Errorf("expected chunk 3 content 'line 10', got %q", chunks[3].Content)
	}
}
