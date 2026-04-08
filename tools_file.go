package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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
	replaceAll := getBoolArg(args, "replace_all", false)
	occurrence := getIntArg(args, "occurrence", 0) // 0 = first (default), -1 = last, N = Nth
	dryRun := getBoolArg(args, "dry_run", false)
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

	var result string
	matchCount := strings.Count(content, oldText)

	if replaceAll {
		result = strings.ReplaceAll(content, oldText, newText)
		if dryRun {
			return fmt.Sprintf("[dry-run] Would replace all %d occurrences in %s", matchCount, path)
		}
	} else if occurrence == -1 {
		// Replace last occurrence
		lastIdx := strings.LastIndex(content, oldText)
		result = content[:lastIdx] + newText + content[lastIdx+len(oldText):]
		if dryRun {
			lineNum := strings.Count(content[:lastIdx], "\n") + 1
			return fmt.Sprintf("[dry-run] Would replace last occurrence (line ~%d) in %s (%d total matches)", lineNum, path, matchCount)
		}
	} else if occurrence > 1 {
		// Replace Nth occurrence
		if occurrence > matchCount {
			return errorMessage("occurrence %d requested but only %d found in %s", occurrence, matchCount, path)
		}
		idx := 0
		for i := 0; i < occurrence; i++ {
			found := strings.Index(content[idx:], oldText)
			if found == -1 {
				return errorMessage("occurrence %d not found in %s", occurrence, path)
			}
			if i < occurrence-1 {
				idx += found + len(oldText)
			} else {
				idx += found
			}
		}
		result = content[:idx] + newText + content[idx+len(oldText):]
		if dryRun {
			lineNum := strings.Count(content[:idx], "\n") + 1
			return fmt.Sprintf("[dry-run] Would replace occurrence #%d (line ~%d) in %s (%d total matches)", occurrence, lineNum, path, matchCount)
		}
	} else {
		// Default: replace first occurrence
		result = strings.Replace(content, oldText, newText, 1)
		if dryRun {
			firstIdx := strings.Index(content, oldText)
			lineNum := strings.Count(content[:firstIdx], "\n") + 1
			return fmt.Sprintf("[dry-run] Would replace first occurrence (line ~%d) in %s (%d total matches)", lineNum, path, matchCount)
		}
	}

	// Save backup for undo
	saveEditBackup(full, data)

	if err := os.WriteFile(full, []byte(result), 0o644); err != nil {
		return errorMessage("could not write to %s: %v", path, err)
	}

	suffix := ""
	if replaceAll && matchCount > 1 {
		suffix = fmt.Sprintf(" (%d replacements)", matchCount)
	} else if matchCount > 1 {
		suffix = fmt.Sprintf(" (1 of %d matches)", matchCount)
	}
	return fmt.Sprintf("Edited %s%s", path, suffix)
}

// editFileLines replaces a range of lines in a file with new content.
func (t *ToolExecutor) editFileLines(args map[string]any) string {
	path := getStringArg(args, "path", "")
	startLine := getIntArg(args, "start_line", 0)
	endLine := getIntArg(args, "end_line", 0)
	newContent := getStringArg(args, "content", "")
	dryRun := getBoolArg(args, "dry_run", false)

	if path == "" {
		return errorMessage("path is required")
	}
	if startLine < 1 {
		return errorMessage("start_line must be >= 1")
	}
	if endLine < startLine {
		return errorMessage("end_line must be >= start_line")
	}

	full, err := t.safePath(path)
	if err != nil {
		return errorMessage("%v", err)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return errorMessage("could not read %s: %v", path, err)
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	if startLine > totalLines {
		return errorMessage("start_line %d exceeds file length (%d lines)", startLine, totalLines)
	}
	if endLine > totalLines {
		endLine = totalLines
	}

	// Show what would be replaced in dry-run mode
	if dryRun {
		var preview strings.Builder
		preview.WriteString(fmt.Sprintf("[dry-run] Would replace lines %d-%d of %s (%d lines total)\n", startLine, endLine, path, totalLines))
		preview.WriteString("--- Current content:\n")
		for i := startLine - 1; i < endLine && i < totalLines; i++ {
			preview.WriteString(fmt.Sprintf("  %4d  %s\n", i+1, lines[i]))
		}
		newLines := strings.Split(newContent, "\n")
		preview.WriteString(fmt.Sprintf("--- New content (%d lines)", len(newLines)))
		return preview.String()
	}

	// Save backup for undo
	saveEditBackup(full, data)

	// Build new file: before + new content + after
	var result []string
	result = append(result, lines[:startLine-1]...)
	if newContent != "" {
		result = append(result, strings.Split(newContent, "\n")...)
	}
	if endLine < totalLines {
		result = append(result, lines[endLine:]...)
	}

	if err := os.WriteFile(full, []byte(strings.Join(result, "\n")), 0o644); err != nil {
		return errorMessage("could not write to %s: %v", path, err)
	}

	removedCount := endLine - startLine + 1
	newLineCount := len(strings.Split(newContent, "\n"))
	if newContent == "" {
		newLineCount = 0
	}
	return fmt.Sprintf("Edited %s: replaced lines %d-%d (%d lines) with %d new lines", path, startLine, endLine, removedCount, newLineCount)
}

// patchFile applies a unified diff to a file.
func (t *ToolExecutor) patchFile(args map[string]any) string {
	path := getStringArg(args, "path", "")
	diff := getStringArg(args, "diff", "")
	dryRun := getBoolArg(args, "dry_run", false)

	if path == "" {
		return errorMessage("path is required")
	}
	if diff == "" {
		return errorMessage("diff is required")
	}

	full, err := t.safePath(path)
	if err != nil {
		return errorMessage("%v", err)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return errorMessage("could not read %s: %v", path, err)
	}

	result, hunksApplied, err := applyUnifiedDiff(string(data), diff)
	if err != nil {
		return errorMessage("patch failed: %v", err)
	}

	if dryRun {
		return fmt.Sprintf("[dry-run] Would apply %d hunk(s) to %s", hunksApplied, path)
	}

	// Save backup for undo
	saveEditBackup(full, data)

	if err := os.WriteFile(full, []byte(result), 0o644); err != nil {
		return errorMessage("could not write to %s: %v", path, err)
	}

	return fmt.Sprintf("Patched %s: %d hunk(s) applied", path, hunksApplied)
}

// undoEdit reverts the last edit to a file using the saved backup.
func (t *ToolExecutor) undoEdit(args map[string]any) string {
	path := getStringArg(args, "path", "")
	if path == "" {
		return errorMessage("path is required")
	}

	full, err := t.safePath(path)
	if err != nil {
		return errorMessage("%v", err)
	}

	backup, ok := getEditBackup(full)
	if !ok {
		return errorMessage("no edit history for %s (only the most recent edit per file can be undone)", path)
	}

	// Save current content as the new backup (allows re-undo to toggle)
	currentData, _ := os.ReadFile(full)
	saveEditBackup(full, currentData)

	if err := os.WriteFile(full, backup, 0o644); err != nil {
		return errorMessage("could not restore %s: %v", path, err)
	}

	return fmt.Sprintf("Reverted %s to previous version (%d bytes)", path, len(backup))
}

func (t *ToolExecutor) listFiles(args map[string]any) string {
	if t == nil || t.baseDir == "" {
		return errorMessage("ToolExecutor not initialized")
	}
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
	if t == nil || t.baseDir == "" {
		return nil, fmt.Errorf("ToolExecutor not initialized")
	}
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

// ─── Edit Backup System ─────────────────────────────────────────────
//
// Maintains one backup per file (the content before the most recent edit).
// This enables single-level undo for any file-modifying edit tool.

var (
	editBackups   = make(map[string][]byte)
	editBackupsMu sync.Mutex
)

// saveEditBackup stores a copy of file content before an edit.
func saveEditBackup(fullPath string, content []byte) {
	editBackupsMu.Lock()
	defer editBackupsMu.Unlock()
	backup := make([]byte, len(content))
	copy(backup, content)
	editBackups[fullPath] = backup
}

// getEditBackup returns the backup for a file, if one exists.
func getEditBackup(fullPath string) ([]byte, bool) {
	editBackupsMu.Lock()
	defer editBackupsMu.Unlock()
	data, ok := editBackups[fullPath]
	return data, ok
}

// ─── Unified Diff Parser ────────────────────────────────────────────
//
// applyUnifiedDiff applies a unified diff (as produced by `diff -u` or
// `git diff`) to source content. It handles multiple hunks and validates
// context lines. Only the hunk headers (@@ lines) and +/- content lines
// are required; file headers (--- / +++) are optional and ignored.

func applyUnifiedDiff(source, diff string) (string, int, error) {
	sourceLines := strings.Split(source, "\n")
	diffLines := strings.Split(diff, "\n")

	// Parse hunks from the diff
	type hunk struct {
		srcStart int      // 1-based line in original file
		srcCount int      // number of lines from original
		lines    []string // raw diff lines (context, +, -)
	}

	var hunks []hunk
	var currentHunk *hunk

	for _, line := range diffLines {
		// Skip file headers
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue
		}

		// Parse hunk header: @@ -start,count +start,count @@
		if strings.HasPrefix(line, "@@") {
			h := hunk{}
			// Extract source start and count
			parts := strings.SplitN(line, "@@", 3)
			if len(parts) < 2 {
				return "", 0, fmt.Errorf("malformed hunk header: %s", line)
			}
			ranges := strings.TrimSpace(parts[1])
			// Parse -start,count
			for _, r := range strings.Fields(ranges) {
				if strings.HasPrefix(r, "-") {
					r = strings.TrimPrefix(r, "-")
					parts := strings.SplitN(r, ",", 2)
					fmt.Sscanf(parts[0], "%d", &h.srcStart)
					h.srcCount = 1
					if len(parts) == 2 {
						fmt.Sscanf(parts[1], "%d", &h.srcCount)
					}
				}
			}
			if h.srcStart == 0 {
				return "", 0, fmt.Errorf("could not parse source range from: %s", line)
			}
			currentHunk = &h
			hunks = append(hunks, h)
			continue
		}

		// Collect hunk content lines
		if currentHunk != nil {
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
				hunks[len(hunks)-1].lines = append(hunks[len(hunks)-1].lines, line)
			}
			// Skip "\ No newline at end of file" and other noise
		}
	}

	if len(hunks) == 0 {
		return "", 0, fmt.Errorf("no hunks found in diff")
	}

	// Apply hunks in reverse order to preserve line numbers
	result := make([]string, len(sourceLines))
	copy(result, sourceLines)

	for i := len(hunks) - 1; i >= 0; i-- {
		h := hunks[i]
		srcIdx := h.srcStart - 1 // convert to 0-based

		// Build the expected source lines and new lines from the hunk
		var expectedSrc []string
		var newLines []string

		for _, line := range h.lines {
			if strings.HasPrefix(line, " ") {
				expectedSrc = append(expectedSrc, line[1:])
				newLines = append(newLines, line[1:])
			} else if strings.HasPrefix(line, "-") {
				expectedSrc = append(expectedSrc, line[1:])
			} else if strings.HasPrefix(line, "+") {
				newLines = append(newLines, line[1:])
			}
		}

		// Validate context: check that expected source lines match
		for j, expected := range expectedSrc {
			lineIdx := srcIdx + j
			if lineIdx >= len(result) {
				return "", 0, fmt.Errorf("hunk %d: source line %d out of range (file has %d lines)", i+1, lineIdx+1, len(result))
			}
			if result[lineIdx] != expected {
				return "", 0, fmt.Errorf("hunk %d: line %d mismatch:\n  expected: %q\n  actual:   %q", i+1, lineIdx+1, expected, result[lineIdx])
			}
		}

		// Replace the source range with new lines
		after := make([]string, len(result[srcIdx+len(expectedSrc):]))
		copy(after, result[srcIdx+len(expectedSrc):])
		result = append(result[:srcIdx], append(newLines, after...)...)
	}

	return strings.Join(result, "\n"), len(hunks), nil
}
