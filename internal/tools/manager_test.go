package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/laughingmandev/loa/internal/config"
	"github.com/laughingmandev/loa/internal/state"
)

func testManager(t *testing.T, root string) *Manager {
	t.Helper()
	cfg := config.Default()
	return New(root, nil, func() config.Config { return cfg }, nil)
}

func TestReadRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "loa-outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outside)
	m := testManager(t, root)
	if _, _, err := m.readFile("../loa-outside.txt"); err == nil {
		t.Fatal("expected path escape rejection")
	}
}

func TestWriteRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "target.txt")
	if err := os.WriteFile(outside, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}
	m := testManager(t, root)
	if _, _, err := m.writeFile(context.Background(), "link.txt", "pwned"); err == nil {
		t.Fatal("expected symlink escape rejection")
	}
	b, _ := os.ReadFile(outside)
	if string(b) != "original" {
		t.Fatalf("outside file changed: %q", b)
	}
}

func TestProcessRunsAtProjectRootAndInheritsEnvironment(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOA_TEST_ENV", "works")
	m := testManager(t, root)
	call := state.ToolCall{ID: 1, Kind: state.ToolExecuteProcess, Input: map[string]any{"binary": "/bin/sh", "args": []string{"-c", "printf '%s|%s' \"$LOA_TEST_ENV\" \"$PWD\""}}}
	res := m.Execute(context.Background(), call, 1)
	if !res.Success {
		t.Fatalf("process failed: %+v", res)
	}
	want := "works|" + root
	if res.Output != want {
		t.Fatalf("got %q want %q", res.Output, want)
	}
}

func TestShellUsesSameInheritedPathAsProcess(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	probe := filepath.Join(binDir, "loa-path-probe")
	if err := os.WriteFile(probe, []byte("#!/bin/sh\nprintf 'probe-ok'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	wantPath := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", wantPath)

	m := testManager(t, root)
	call := state.ToolCall{
		ID:   3,
		Kind: state.ToolExecuteShell,
		Input: map[string]any{
			"command": "printf '%s|' \"$PATH\"; loa-path-probe",
		},
	}
	res := m.Execute(context.Background(), call, 1)
	if !res.Success {
		t.Fatalf("shell failed: %+v", res)
	}
	want := wantPath + "|probe-ok"
	if res.Output != want {
		t.Fatalf("shell environment drifted: got %q want %q", res.Output, want)
	}
}

func TestNonZeroCommandIsNormalToolResult(t *testing.T) {
	root := t.TempDir()
	m := testManager(t, root)
	call := state.ToolCall{
		ID:   2,
		Kind: state.ToolExecuteProcess,
		Input: map[string]any{
			"binary": "/bin/sh",
			"args":   []string{"-c", "printf 'expected failure'; exit 7"},
		},
	}
	res := m.Execute(context.Background(), call, 1)
	if res.Success {
		t.Fatalf("expected unsuccessful result: %+v", res)
	}
	if res.ExitCode == nil || *res.ExitCode != 7 {
		t.Fatalf("expected exit code 7, got %+v", res.ExitCode)
	}
	if res.Output != "expected failure" {
		t.Fatalf("expected captured stdout, got %q", res.Output)
	}
}

func TestSearchIgnoresLoaRuntimeDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loa", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loa", "sessions", "hidden.json"), []byte("loa-runtime-secret-marker"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "visible.txt"), []byte("visible marker"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := testManager(t, root)
	paths, _, err := m.searchPath("hidden", 20)
	if err != nil {
		t.Fatal(err)
	}
	if paths != "" {
		t.Fatalf("runtime session file leaked into search_path: %q", paths)
	}
	text, _, err := m.searchText("loa-runtime-secret-marker", false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Fatalf("runtime session contents leaked into search_text: %q", text)
	}
}
