package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ─── File Operation Tools ────────────────────────────────────────────

func (t *ToolExecutor) readFile(args map[string]any) string {
	path := getStringArg(args, "path", "")
	if path == "" {
		return errorMessage("path is required")
	}
	offset := getIntArg(args, "offset", 1)
	limit := getIntArg(args, "limit", 200)

	full, err := t.safePath(path)
	if err != nil {
		return errorMessage("%v", err)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return errorMessage("could not read %s: %v", path, err)
	}

	if isBinaryData(data) {
		return errorMessage("%s is a binary file, not a text file", path)
	}

	allLines := strings.Split(string(data), "\n")
	total := len(allLines)
	start := offset - 1
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > total {
		end = total
	}

	var numbered []string
	for i := start; i < end; i++ {
		numbered = append(numbered, fmt.Sprintf("%4d  %s", i+1, allLines[i]))
	}

	header := fmt.Sprintf("[%s: lines %d-%d of %d]", path, start+1, end, total)
	if end < total {
		header += fmt.Sprintf("  (use offset=%d to read more)", end+1)
	}
	return header + "\n" + strings.Join(numbered, "\n")
}

func (t *ToolExecutor) writeFile(args map[string]any) string {
	path := getStringArg(args, "path", "")
	content := getStringArg(args, "content", "")
	if path == "" {
		return errorMessage("path is required")
	}

	full, err := t.safePath(path)
	if err != nil {
		return errorMessage("%v", err)
	}

	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errorMessage("could not create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return errorMessage("could not write to %s: %v", path, err)
	}
	return fmt.Sprintf("Wrote %d chars to %s", len(content), path)
}

func (t *ToolExecutor) editFile(args map[string]any) string {
	path := getStringArg(args, "path", "")
	oldText := getStringArg(args, "old_text", "")
	newText := getStringArg(args, "new_text", "")
	if path == "" {
		return errorMessage("path is required")
	}

	full, err := t.safePath(path)
	if err != nil {
		return errorMessage("%v", err)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return errorMessage("could not read %s: %v", path, err)
	}

	content := string(data)
	if !strings.Contains(content, oldText) {
		return errorMessage("old_text not found in %s", path)
	}

	content = strings.Replace(content, oldText, newText, 1)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return errorMessage("could not write to %s: %v", path, err)
	}

	return fmt.Sprintf("Edited %s", path)
}

func (t *ToolExecutor) listFiles(args map[string]any) string {
	pattern := getStringArg(args, "pattern", "*")

	var matches []string
	var err error

	// Handle recursive glob patterns (**/)
	if strings.Contains(pattern, "**") {
		matches, err = t.globRecursive(pattern)
	} else {
		matches, err = filepath.Glob(filepath.Join(t.baseDir, pattern))
	}

	if err != nil {
		return errorMessage("%v", err)
	}

	var files, dirs []string
	for _, m := range matches {
		rel, _ := filepath.Rel(t.baseDir, m)
		// Skip noise directories
		topDir := strings.SplitN(rel, string(filepath.Separator), 2)[0]
		if topDir == ".yolo" || topDir == ".git" || topDir == "__pycache__" || topDir == "node_modules" {
			continue
		}
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if info.IsDir() {
			dirs = append(dirs, rel+"/")
		} else {
			files = append(files, rel)
		}
	}

	items := append(dirs, files...)
	if len(items) == 0 {
		if !strings.Contains(pattern, "**") && strings.Contains(pattern, ".") {
			return fmt.Sprintf("(no matching files or directories — note: '%s' only matches the top-level directory; use '**/%s' to search recursively)", pattern, pattern)
		}
		return "(no matching files or directories)"
	}

	totalItems := len(items)
	header := fmt.Sprintf("(%d file(s), %d dir(s))", len(files), len(dirs))
	limit := 200
	if totalItems > limit {
		items = items[:limit]
		header += fmt.Sprintf(" [showing first %d of %d — results truncated]", limit, totalItems)
	}
	if !strings.Contains(pattern, "**") && !strings.Contains(pattern, string(filepath.Separator)) {
		header += fmt.Sprintf(" [top-level only — use '**/%s' for recursive]", pattern)
	}
	return header + "\n" + strings.Join(items, "\n")
}

func (t *ToolExecutor) makeDir(args map[string]any) string {
	path := getStringArg(args, "path", "")
	if path == "" {
		return errorMessage("path is required")
	}

	full, err := t.safePath(path)
	if err != nil {
		return errorMessage("%v", err)
	}

	if err := os.MkdirAll(full, 0o755); err != nil {
		return errorMessage("could not create directory %s: %v", path, err)
	}

	return fmt.Sprintf("Created directory: %s", path)
}

func (t *ToolExecutor) removeDir(args map[string]any) string {
	path := getStringArg(args, "path", "")
	if path == "" {
		return errorMessage("path is required")
	}

	full, err := t.safePath(path)
	if err != nil {
		return errorMessage("%v", err)
	}

	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return errorMessage("%s does not exist", path)
		}
		return errorMessage("%v", err)
	}

	if !info.IsDir() {
		return errorMessage("%s is not a directory", path)
	}

	if err := os.RemoveAll(full); err != nil {
		return errorMessage("could not remove directory %s: %v", path, err)
	}

	return fmt.Sprintf("Removed directory: %s", path)
}

func (t *ToolExecutor) copyFile(args map[string]any) string {
	source := getStringArg(args, "source", "")
	dest := getStringArg(args, "dest", "")
	if source == "" {
		return errorMessage("'source' parameter is required")
	}
	if dest == "" {
		return errorMessage("'dest' parameter is required")
	}

	fullSource, err := t.safePath(source)
	if err != nil {
		return errorMessage("%v", err)
	}

	fullDest, err := t.safePath(dest)
	if err != nil {
		return errorMessage("%v", err)
	}

	info, err := os.Stat(fullSource)
	if err != nil {
		if os.IsNotExist(err) {
			return errorMessage("source file %s does not exist", source)
		}
		return errorMessage("%v", err)
	}

	if info.IsDir() {
		return errorMessage("cannot copy directories, source %s is a directory", source)
	}

	destDir := filepath.Dir(fullDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errorMessage("could not create destination directory: %v", err)
	}

	content, err := os.ReadFile(fullSource)
	if err != nil {
		return errorMessage("could not read source file %s: %v", source, err)
	}

	if err := os.WriteFile(fullDest, content, 0644); err != nil {
		return errorMessage("could not write to destination %s: %v", dest, err)
	}

	return fmt.Sprintf("Copied %s -> %s", source, dest)
}

func (t *ToolExecutor) moveFile(args map[string]any) string {
	source := getStringArg(args, "source", "")
	dest := getStringArg(args, "dest", "")
	if source == "" {
		return errorMessage("'source' parameter is required")
	}
	if dest == "" {
		return errorMessage("'dest' parameter is required")
	}

	fullSource, err := t.safePath(source)
	if err != nil {
		return errorMessage("%v", err)
	}

	fullDest, err := t.safePath(dest)
	if err != nil {
		return errorMessage("%v", err)
	}

	info, err := os.Stat(fullSource)
	if err != nil {
		if os.IsNotExist(err) {
			return errorMessage("source file %s does not exist", source)
		}
		return errorMessage("%v", err)
	}

	if info.IsDir() {
		return errorMessage("cannot move directories, source %s is a directory", source)
	}

	destDir := filepath.Dir(fullDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errorMessage("could not create destination directory: %v", err)
	}

	if err := os.Rename(fullSource, fullDest); err != nil {
		data, readErr := os.ReadFile(fullSource)
		if readErr != nil {
			return errorMessage("could not read source file %s: %v", source, readErr)
		}
		if writeErr := os.WriteFile(fullDest, data, info.Mode()); writeErr != nil {
			return errorMessage("could not write to destination %s: %v", dest, writeErr)
		}
		if removeErr := os.Remove(fullSource); removeErr != nil {
			return errorMessage("warning: copied to %s but failed to remove source: %v", dest, removeErr)
		}
	}

	return fmt.Sprintf("File moved successfully from %s to %s", source, dest)
}

// ─── Glob and Search Helpers ─────────────────────────────────────────

// globFiles is a standalone helper for recursive glob pattern matching
func globFiles(baseDir, pattern string) ([]string, error) {
	var matches []string

	if strings.HasPrefix(pattern, "**/") {
		basePattern := pattern[3:]

		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				name := filepath.Base(path)
				if name == ".yolo" || name == ".git" || name == "__pycache__" || name == "node_modules" {
					return filepath.SkipDir
				}
			}

			relPath, _ := filepath.Rel(baseDir, path)
			if relPath == "." {
				relPath = ""
			}

			if !info.IsDir() {
				name := filepath.Base(path)
				matched, _ := filepath.Match(basePattern, name)
				if matched {
					matches = append(matches, path)
				}
			}
			return nil
		})

		return matches, err
	}

	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) == 2 {
		walkBaseDir := baseDir
		if parts[0] != "" {
			walkBaseDir = filepath.Join(baseDir, strings.TrimSuffix(parts[0], "/"))
		}

		err := filepath.Walk(walkBaseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				name := filepath.Base(path)
				if name == ".yolo" || name == ".git" || name == "__pycache__" || name == "node_modules" {
					return filepath.SkipDir
				}
			}

			if !info.IsDir() {
				patternToMatch := parts[1]
				if strings.HasPrefix(patternToMatch, "/") {
					patternToMatch = patternToMatch[1:]
				}

				matched := false
				if m, e := filepath.Match("*"+patternToMatch, filepath.Base(path)); e == nil {
					matched = m
				}
				if !matched {
					if m, e := filepath.Match(patternToMatch, filepath.Base(path)); e == nil {
						matched = m
					}
				}

				if matched {
					matches = append(matches, path)
				}
			}
			return nil
		})

		return matches, err
	}

	return filepath.Glob(filepath.Join(baseDir, pattern))
}

// globRecursive calls the standalone helper with the executor's base directory
func (t *ToolExecutor) globRecursive(pattern string) ([]string, error) {
	return globFiles(t.baseDir, pattern)
}

func (t *ToolExecutor) searchFiles(args map[string]any) string {
	query := getStringArg(args, "query", "")
	pattern := getStringArg(args, "pattern", "**/*")
	if query == "" {
		return errorMessage("query is required")
	}

	re, err := regexp.Compile(query)
	if err != nil {
		return errorMessage("invalid regex: %v", err)
	}

	var hits []string
	err = filepath.Walk(t.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := filepath.Base(path)
			if name == ".yolo" || name == ".git" || name == "__pycache__" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(t.baseDir, path)
		if pattern != "**/*" {
			matched := false
			// Handle **/ prefix patterns by matching against basename (matches anywhere)
			if strings.HasPrefix(pattern, "**/") {
				basePattern := strings.TrimPrefix(pattern, "**/")
				matched, _ = filepath.Match(basePattern, filepath.Base(path))
			} else if !strings.Contains(pattern, "/") {
				// Simple pattern without path separators - only match files in root directory
				if !strings.Contains(rel, string(filepath.Separator)) {
					matched, _ = filepath.Match(pattern, rel)
				}
			} else {
				// Pattern with path separators - match against relative path
				matched, _ = filepath.Match(pattern, rel)
			}
			if !matched {
				return nil
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				hits = append(hits, fmt.Sprintf("%s:%d: %s", rel, lineNum, line))
				if len(hits) >= 50 {
					return io.EOF
				}
			}
		}
		return nil
	})

	truncated := err == io.EOF
	if err != nil && err != io.EOF {
		// Walk errors are mostly ignored
	}

	if len(hits) == 0 {
		return "No matches found"
	}
	result := strings.Join(hits, "\n")
	if truncated {
		result += fmt.Sprintf("\n[results truncated at %d matches — narrow your query or pattern for more specific results]", len(hits))
	}
	return result
}
