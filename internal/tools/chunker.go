package tools

import (
	"bufio"
	"os"
	"strings"
)

// FileChunk represents a logical portion of a file.
type FileChunk struct {
	StartLine int
	EndLine   int
	Content   string
}

// ChunkFileByLines reads a file and splits it into chunks of at most maxLines.
// It ensures that chunks are split cleanly on line breaks.
func ChunkFileByLines(path string, maxLines int) ([]FileChunk, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var chunks []FileChunk
	var currentLines []string
	startLine := 1
	currentLine := 0

	scanner := bufio.NewScanner(f)
	// Increase buffer size to handle long lines
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	for scanner.Scan() {
		currentLine++
		currentLines = append(currentLines, scanner.Text())

		if len(currentLines) >= maxLines {
			chunks = append(chunks, FileChunk{
				StartLine: startLine,
				EndLine:   currentLine,
				Content:   strings.Join(currentLines, "\n"),
			})
			startLine = currentLine + 1
			currentLines = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Add the remaining lines if any
	if len(currentLines) > 0 {
		chunks = append(chunks, FileChunk{
			StartLine: startLine,
			EndLine:   currentLine,
			Content:   strings.Join(currentLines, "\n"),
		})
	}

	return chunks, nil
}
