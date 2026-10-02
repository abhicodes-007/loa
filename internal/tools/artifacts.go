package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"bufio"
)

func (m *Manager) artifactDir(taskID uint64) string {
	sess := "default"
	if m.GetSessionID != nil {
		if s := m.GetSessionID(); s != "" {
			sess = s
		}
	}
	return filepath.Join(m.Root, ".loa", "artifacts", sess, fmt.Sprintf("task-%d", taskID))
}

func (m *Manager) resolveArtifact(taskID uint64, name string) (string, error) {
	if name == "" {
		return "", errors.New("artifact name cannot be empty")
	}
	dir := m.artifactDir(taskID)
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return "", errors.New("artifact name must be a simple filename, no directories allowed")
	}
	return filepath.Join(dir, name), nil
}

func (m *Manager) artifactList(taskID uint64) (string, *int, error) {
	dir := m.artifactDir(taskID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "No artifacts exist for this task yet.", nil, nil
		}
		return "", nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "No artifacts exist for this task yet.", nil, nil
	}
	return "Artifacts:\n" + strings.Join(names, "\n"), nil, nil
}

func (m *Manager) artifactRead(taskID uint64, name string, start, end int) (string, *int, error) {
	path, err := m.resolveArtifact(taskID, name)
	if err != nil {
		return "", nil, err
	}
	if start == 0 && end == 0 {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", nil, err
		}
		return string(b), nil, nil
	}
	
	if start < 1 {
		start = 1
	}
	if end != 0 && end < start {
		return "", nil, errors.New("end must be >= start")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
	var b strings.Builder
	line := 1
	for sc.Scan() {
		if line >= start && (end == 0 || line <= end) {
			b.WriteString(fmt.Sprintf("%4d | %s\n", line, sc.Text()))
		}
		line++
		if end != 0 && line > end {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return "", nil, err
	}
	return b.String(), nil, nil
}

func (m *Manager) artifactWrite(taskID uint64, name, content string) (string, *int, error) {
	path, err := m.resolveArtifact(taskID, name)
	if err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("Wrote artifact %s (%d bytes)", name, len(content)), nil, nil
}

func (m *Manager) DumpToArtifact(taskID uint64, name, content string) error {
	_, _, err := m.artifactWrite(taskID, name, content)
	return err
}

func (m *Manager) artifactAppend(taskID uint64, name, content string) (string, *int, error) {
	path, err := m.resolveArtifact(taskID, name)
	if err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("Appended %d bytes to artifact %s", len(content), name), nil, nil
}

func (m *Manager) artifactPatch(ctx context.Context, taskID uint64, name, oldStr, newStr string, all bool) (string, *int, error) {
	path, err := m.resolveArtifact(taskID, name)
	if err != nil {
		return "", nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("read artifact: %w", err)
	}
	content := string(b)
	if !strings.Contains(content, oldStr) {
		return "", nil, errors.New("old content not found in artifact")
	}
	if !all && strings.Count(content, oldStr) > 1 {
		return "", nil, errors.New("old content is not unique. Provide a larger snippet or set replace_all=true")
	}
	var patched string
	if all {
		patched = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		patched = strings.Replace(content, oldStr, newStr, 1)
	}
	if err := os.WriteFile(path, []byte(patched), 0644); err != nil {
		return "", nil, fmt.Errorf("write patched artifact: %w", err)
	}
	return fmt.Sprintf("Successfully patched artifact %s", name), nil, nil
}

func (m *Manager) artifactSearch(taskID uint64, query string) (string, *int, error) {
	dir := m.artifactDir(taskID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "No artifacts exist for this task yet.", nil, nil
		}
		return "", nil, err
	}
	var b strings.Builder
	for _, e := range entries {
		if !e.IsDir() {
			path := filepath.Join(dir, e.Name())
			content, err := os.ReadFile(path)
			if err == nil && strings.Contains(string(content), query) {
				lines := strings.Split(string(content), "\n")
				for i, line := range lines {
					if strings.Contains(line, query) {
						b.WriteString(fmt.Sprintf("%s:%d: %s\n", e.Name(), i+1, line))
					}
				}
			}
		}
	}
	if b.Len() == 0 {
		return "No matches found.", nil, nil
	}
	return b.String(), nil, nil
}
