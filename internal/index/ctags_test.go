package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRebuildKeepsPartialIndexOnCtagsError(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	ctags := filepath.Join(binDir, "ctags")
	script := `#!/bin/sh
if [ "$1" = "--list-features" ]; then
  printf 'json\n'
  exit 0
fi
printf '%s\n' '{"_type":"tag","name":"Main","path":"main.go","kind":"function","line":10,"language":"Go"}'
printf '%s\n' 'synthetic parse warning' >&2
exit 1
`
	if err := os.WriteFile(ctags, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	idx := New(root)
	if err := idx.Rebuild(context.Background()); err != nil {
		t.Fatalf("Rebuild should keep partial results, got %v", err)
	}
	if !idx.Available() {
		t.Fatal("index should remain available")
	}
	got := idx.SymbolsForFile("main.go")
	if len(got) != 1 || got[0].Name != "Main" {
		t.Fatalf("unexpected partial symbols: %+v", got)
	}
	if !strings.Contains(idx.Warning(), "partial index kept") {
		t.Fatalf("expected partial-index warning, got %q", idx.Warning())
	}
}

func TestIgnoredPathCoversLoaRuntimeDirectory(t *testing.T) {
	for _, path := range []string{".loa", ".loa/sessions/session.json", "nested/.loa/debug/dump.json", ".loa.json"} {
		if !isIgnoredPath(path) {
			t.Fatalf("expected %q to be ignored", path)
		}
	}
	if isIgnoredPath("internal/loa/session.go") {
		t.Fatal("ordinary project paths containing loa should not be ignored")
	}
}
