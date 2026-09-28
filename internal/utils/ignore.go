package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadLoaignore parses the .loaignore file in the given root directory.
// It returns a slice of paths to exclude (formatted for ctags compatibility)
// and a function to check if a specific relative path is ignored.
func ReadLoaignore(root string) ([]string, func(string) bool) {
	ignorePath := filepath.Join(root, ".loaignore")
	content, err := os.ReadFile(ignorePath)
	if err != nil {
		// Return empty list and a function that always returns false
		return nil, func(string) bool { return false }
	}

	var rawPatterns []string
	var ctagsExcludes []string

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rawPatterns = append(rawPatterns, line)

		// For ctags --exclude, we remove trailing /* or /
		exclude := strings.TrimSuffix(line, "/*")
		exclude = strings.TrimSuffix(exclude, "/")
		if exclude != "" {
			ctagsExcludes = append(ctagsExcludes, exclude)
		}
	}

	isIgnored := func(relPath string) bool {
		relPath = filepath.ToSlash(relPath) // normalize for matching
		for _, pattern := range rawPatterns {
			if strings.HasSuffix(pattern, "/*") {
				prefix := strings.TrimSuffix(pattern, "*")
				if strings.HasPrefix(relPath, prefix) || relPath == strings.TrimSuffix(prefix, "/") {
					return true
				}
			} else if strings.HasSuffix(pattern, "/") {
				if strings.HasPrefix(relPath, pattern) || relPath == strings.TrimSuffix(pattern, "/") {
					return true
				}
			} else {
				// Base name exact match or path match
				matched, _ := filepath.Match(pattern, filepath.Base(relPath))
				if matched {
					return true
				}
				if strings.HasPrefix(relPath, pattern+"/") || relPath == pattern {
					return true
				}
			}
		}
		return false
	}

	return ctagsExcludes, isIgnored
}
