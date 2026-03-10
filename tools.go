package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// ─── Tool Definitions ────────────────────────────────────────────────

var ollamaTools = []ToolDef{
	toolDef("read_file", "Read a file's contents. For large files, use offset and limit to read in chunks.",
		map[string]ToolParam{
			"path":   {Type: "string", Description: "Relative path to file"},
			"offset": {Type: "integer", Description: "Starting line number (1-based, default: 1)"},
			"limit":  {Type: "integer", Description: "Max number of lines to read (default: 200)"},
		}, []string{"path"}),
	toolDef("write_file", "Create or overwrite a file",
		map[string]ToolParam{
			"path":    {Type: "string", Description: "Relative path"},
			"content": {Type: "string", Description: "File contents"},
		}, []string{"path", "content"}),
	toolDef("edit_file", "Replace first occurrence of old_text with new_text in a file",
		map[string]ToolParam{
			"path":     {Type: "string", Description: "Relative path"},
			"old_text": {Type: "string", Description: "Text to find"},
			"new_text": {Type: "string", Description: "Replacement text"},
		}, []string{"path", "old_text", "new_text"}),
	toolDef("list_files", "List files matching a glob pattern in the working directory",
		map[string]ToolParam{
			"pattern": {Type: "string", Description: "Glob pattern (default: *)"},
		}, nil),
	toolDef("search_files", "Search file contents using regex",
		map[string]ToolParam{
			"query":   {Type: "string", Description: "Regex pattern to search for"},
			"pattern": {Type: "string", Description: "Glob pattern to filter files (default: **/*)"},
		}, []string{"query"}),
	toolDef("run_command", fmt.Sprintf("Execute a shell command (timeout: %ds)", CommandTimeout),
		map[string]ToolParam{
			"command": {Type: "string", Description: "Shell command to run"},
		}, []string{"command"}),
	toolDef("make_dir", "Create a new directory recursively.",
		map[string]ToolParam{
			"path": {Type: "string", Description: "Relative path for the new directory"},
		}, []string{"path"}),
	toolDef("remove_dir", "Remove a directory and all its contents recursively. Only works on directories, not files.",
		map[string]ToolParam{
			"path": {Type: "string", Description: "Relative path to the directory to remove"},
		}, []string{"path"}),
	toolDef("spawn_subagent", "Spawn a background sub-agent for a parallel task",
		map[string]ToolParam{
			"prompt":      {Type: "string", Description: "Task description/prompt for the sub-agent"},
			"name":        {Type: "string", Description: "Name for the sub-agent (optional)"},
			"description": {Type: "string", Description: "Optional description of the sub-agent"},
		}, []string{"prompt"}),
	toolDef("list_subagents", "List all active/background sub-agents with their status and progress",
		map[string]ToolParam{}, nil),
	toolDef("read_subagent_result", "Read the result from a specific sub-agent by ID",
		map[string]ToolParam{
			"id": {Type: "string", Description: "Sub-agent ID to retrieve result for"},
		}, []string{"id"}),
	toolDef("summarize_subagents", "Get summary statistics of all sub-agents (completed/errors)",
		map[string]ToolParam{}, nil),
	toolDef("list_models", "List available Ollama models", map[string]ToolParam{}, nil),
	toolDef("switch_model", "Switch to a different Ollama model",
		map[string]ToolParam{
			"model": {Type: "string", Description: "Model name"},
		}, []string{"model"}),
	toolDef("think", "Record internal reasoning or a plan without taking action",
		map[string]ToolParam{
			"thought": {Type: "string", Description: "Your reasoning"},
		}, []string{"thought"}),
	toolDef("restart", "Rebuild and restart the program", map[string]ToolParam{}, nil),
	toolDef("copy_file", "Copy a file from source to destination. Creates destination directory if needed.",
		map[string]ToolParam{
			"source": {Type: "string", Description: "Relative path to source file"},
			"dest":   {Type: "string", Description: "Relative path for destination"},
		}, []string{"source", "dest"}),
	toolDef("move_file", "Move a file from source to destination. Creates destination directory if needed.",
		map[string]ToolParam{
			"source": {Type: "string", Description: "Relative path to source file"},
			"dest":   {Type: "string", Description: "Relative path for destination"},
		}, []string{"source", "dest"}),
}

// ─── Tool Executor ───────────────────────────────────────────────────

// validTools is the canonical list of tool names recognised by Execute.
// It is also used by parseTextToolCalls to filter bracket-format matches.
var validTools = []string{
	"read_file", "write_file", "edit_file", "list_files",
	"search_files", "run_command", "spawn_subagent",
	"list_subagents", "read_subagent_result", "summarize_subagents",
	"list_models", "switch_model", "think", "restart",
	"make_dir", "remove_dir", "copy_file", "move_file",
}

// ToolExecutor dispatches tool calls from the LLM to concrete
// implementations.  All file operations are sandboxed under baseDir
// via safePath.
type ToolExecutor struct {
	baseDir string     // root directory for file operations
	agent   *YoloAgent // back-reference for sub-agent spawning, model switching, etc.
}

// NewToolExecutor creates an executor rooted at baseDir.
func NewToolExecutor(baseDir string, agent *YoloAgent) *ToolExecutor {
	return &ToolExecutor{baseDir: baseDir, agent: agent}
}

// safePath resolves and validates that a relative path stays within baseDir.
// It returns an absolute, clean path or an error if the path escapes the working directory.
func (t *ToolExecutor) safePath(path string) (string, error) {
	// Reject absolute paths - only relative paths are allowed
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("path '%s' must be relative, not absolute", path)
	}

	// Clean and join with baseDir to get absolute path
	full := filepath.Clean(filepath.Join(t.baseDir, path))

	// Ensure the resolved path is within baseDir using a strict prefix check
	// We add a separator to prevent prefix attacks like /proj vs /projector
	baseWithSep := t.baseDir + string(filepath.Separator)
	if full != t.baseDir && !strings.HasPrefix(full, baseWithSep) {
		return "", fmt.Errorf("path '%s' is outside working directory", path)
	}

	return full, nil
}

// Execute dispatches a tool call by name. It returns a human-readable result
// string; errors are returned inline prefixed with "Error:".
func (t *ToolExecutor) Execute(name string, args map[string]any) string {
	switch name {
	case "read_file":
		return t.readFile(args)
	case "write_file":
		return t.writeFile(args)
	case "edit_file":
		return t.editFile(args)
	case "list_files":
		return t.listFiles(args)
	case "search_files":
		return t.searchFiles(args)
	case "run_command":
		return t.runCommand(args)
	case "spawn_subagent":
		return t.spawnSubagent(args)
	case "list_subagents":
		return t.listSubagents(args)
	case "read_subagent_result":
		return t.readSubagentResult(args)
	case "summarize_subagents":
		return t.summarizeSubagents(args)
	case "list_models":
		return t.listModels()
	case "switch_model":
		return t.switchModel(args)
	case "think":
		return "Thought recorded."
	case "restart":
		return t.restart(args)
	case "make_dir":
		return t.makeDir(args)
	case "remove_dir":
		return t.removeDir(args)
	case "copy_file":
		return t.copyFile(args)
	case "move_file":
		return t.moveFile(args)
	default:
		return fmt.Sprintf("Error: unknown tool '%s'. Available tools: %s", name, strings.Join(validTools, ", "))
	}
}

// ─── Argument Helpers ─────────────────────────────────────────────────

func getStringArg(args map[string]any, key, fallback string) string {
	if v, ok := args[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return fallback
}

func getIntArg(args map[string]any, key string, fallback int) int {
	if v, ok := args[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case string:
			n := 0
			fmt.Sscanf(val, "%d", &n)
			if n > 0 {
				return n
			}
		}
	}
	return fallback
}



func isBinaryData(data []byte) bool {
	// Check the first 8KB for null bytes or high ratio of non-text bytes
	size := len(data)
	if size > 8192 {
		size = 8192
	}
	nonText := 0
	for i := 0; i < size; i++ {
		b := data[i]
		if b == 0 {
			return true
		}
		if b < 7 || (b > 14 && b < 32 && b != 27) {
			nonText++
		}
	}
	if size == 0 {
		return false
	}
	return float64(nonText)/float64(size) > 0.1
}

// ─── Tool Implementations ────────────────────────────────────────────

func (t *ToolExecutor) readFile(args map[string]any) string {
	path := getStringArg(args, "path", "")
	if path == "" {
		return "Error: path is required"
	}
	offset := getIntArg(args, "offset", 1)
	limit := getIntArg(args, "limit", 200)

	full, err := t.safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return fmt.Sprintf("Error reading %s: %v", path, err)
	}

	if isBinaryData(data) {
		return fmt.Sprintf("Error: %s is a binary file, not a text file. Cannot read binary files as source code.", path)
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
		return "Error: path is required"
	}

	full, err := t.safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err)
	}

	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("Error writing %s: %v", path, err)
	}
	return fmt.Sprintf("Wrote %d chars to %s", len(content), path)
}

func (t *ToolExecutor) editFile(args map[string]any) string {
	path := getStringArg(args, "path", "")
	oldText := getStringArg(args, "old_text", "")
	newText := getStringArg(args, "new_text", "")
	if path == "" {
		return "Error: path is required"
	}

	full, err := t.safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return fmt.Sprintf("Error reading %s: %v", path, err)
	}

	content := string(data)
	if !strings.Contains(content, oldText) {
		return fmt.Sprintf("Error: old_text not found in %s", path)
	}

	content = strings.Replace(content, oldText, newText, 1)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("Error writing %s: %v", path, err)
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
		return fmt.Sprintf("Error: %v", err)
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
		return "(no matching files or directories)"
	}

	header := fmt.Sprintf("(%d file(s), %d dir(s))", len(files), len(dirs))
	limit := 200
	if len(items) > limit {
		items = items[:limit]
	}
	return header + "\n" + strings.Join(items, "\n")
}

// ─── make_dir and remove_dir tools ───────────────────────────────────

func (t *ToolExecutor) makeDir(args map[string]any) string {
	path := getStringArg(args, "path", "")
	if path == "" {
		return "Error: path is required"
	}

	full, err := t.safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if err := os.MkdirAll(full, 0o755); err != nil {
		return fmt.Sprintf("Error creating directory %s: %v", path, err)
	}

	return fmt.Sprintf("Created directory: %s", path)
}

func (t *ToolExecutor) removeDir(args map[string]any) string {
	path := getStringArg(args, "path", "")
	if path == "" {
		return "Error: path is required"
	}

	full, err := t.safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// Check if path exists and is a directory
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Error: %s does not exist", path)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	if !info.IsDir() {
		return fmt.Sprintf("Error: %s is not a directory", path)
	}

	if err := os.RemoveAll(full); err != nil {
		return fmt.Sprintf("Error removing directory %s: %v", path, err)
	}

	return fmt.Sprintf("Removed directory: %s", path)
}

func (t *ToolExecutor) copyFile(args map[string]any) string {
	source := getStringArg(args, "source", "")
	dest := getStringArg(args, "dest", "")
	if source == "" {
		return "Error: 'source' parameter is required"
	}
	if dest == "" {
		return "Error: 'dest' parameter is required"
	}

	fullSource, err := t.safePath(source)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	fullDest, err := t.safePath(dest)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// Check if source exists and is a file
	info, err := os.Stat(fullSource)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Error: source file %s does not exist", source)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	if info.IsDir() {
		return fmt.Sprintf("Error: cannot move directories, source %s is a directory", source)
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(fullDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Sprintf("Error creating destination directory: %v", err)
	}

	// Read source file
	content, err := os.ReadFile(fullSource)
	if err != nil {
		return fmt.Sprintf("Error reading source file: %v", err)
	}

	// Write to destination
	if err := os.WriteFile(fullDest, content, 0644); err != nil {
		return fmt.Sprintf("Error writing destination file: %v", err)
	}

	return fmt.Sprintf("Copied %s -> %s", source, dest)
}

func (t *ToolExecutor) moveFile(args map[string]any) string {
	source := getStringArg(args, "source", "")
	dest := getStringArg(args, "dest", "")
	if source == "" {
		return "Error: 'source' parameter is required"
	}
	if dest == "" {
		return "Error: 'dest' parameter is required"
	}

	fullSource, err := t.safePath(source)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	fullDest, err := t.safePath(dest)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// Check if source exists and is a file
	info, err := os.Stat(fullSource)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Error: source file %s does not exist", source)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	if info.IsDir() {
		return "Error: cannot move directories"
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(fullDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Sprintf("Error creating destination directory: %v", err)
	}

	// Try os.Rename first (fast, works within same filesystem)
	if err := os.Rename(fullSource, fullDest); err != nil {
		// Rename fails across filesystems; fall back to copy+delete
		data, readErr := os.ReadFile(fullSource)
		if readErr != nil {
			return fmt.Sprintf("Error reading source file: %v", readErr)
		}
		if writeErr := os.WriteFile(fullDest, data, info.Mode()); writeErr != nil {
			return fmt.Sprintf("Error writing destination file: %v", writeErr)
		}
		if removeErr := os.Remove(fullSource); removeErr != nil {
			return fmt.Sprintf("Warning: copied to %s but failed to remove source: %v", dest, removeErr)
		}
	}

	return fmt.Sprintf("File moved successfully from %s to %s", source, dest)
}

// ─── globRecursive handles recursive glob patterns with **/ wildcards ──

func (t *ToolExecutor) globRecursive(pattern string) ([]string, error) {
	var matches []string

	// Handle patterns like **/*.txt or **/directory/*
	if strings.HasPrefix(pattern, "**/") {
		basePattern := pattern[3:]

		err := filepath.Walk(t.baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				name := filepath.Base(path)
				if name == ".yolo" || name == ".git" || name == "__pycache__" || name == "node_modules" {
					return filepath.SkipDir
				}
			}

			relPath, _ := filepath.Rel(t.baseDir, path)
			if relPath == "." {
				relPath = ""
			}

			if !info.IsDir() {
				// For **/*.txt, match against just the filename
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

	// For patterns like dir/**/*.txt
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) == 2 {
		baseDir := t.baseDir
		if parts[0] != "" {
			baseDir = filepath.Join(t.baseDir, strings.TrimSuffix(parts[0], "/"))
		}

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

	return filepath.Glob(filepath.Join(t.baseDir, pattern))
}

func (t *ToolExecutor) searchFiles(args map[string]any) string {
	query := getStringArg(args, "query", "")
	pattern := getStringArg(args, "pattern", "**/*")
	if query == "" {
		return "Error: query is required"
	}

	re, err := regexp.Compile(query)
	if err != nil {
		return fmt.Sprintf("Error: invalid regex: %v", err)
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
		// Simple glob check for pattern filtering
		if pattern != "**/*" {
			matched, _ := filepath.Match(pattern, filepath.Base(path))
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

	if err != nil && err != io.EOF {
		// Walk errors are mostly ignored
	}

	if len(hits) == 0 {
		return "No matches found"
	}
	return strings.Join(hits, "\n")
}

func (t *ToolExecutor) runCommand(args map[string]any) string {
	command := getStringArg(args, "command", "")
	if command == "" {
		return "Error: command is required"
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = t.baseDir

	// Explicitly connect stdin to /dev/null so child processes that try to
	// read input will get immediate EOF instead of hanging or stealing the
	// terminal's stdin.
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stdin = devNull
		defer devNull.Close()
	}

	done := make(chan struct{})
	var stdout, stderr []byte
	var cmdErr error

	go func() {
		defer close(done)
		stdout, cmdErr = cmd.Output()
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			stderr = exitErr.Stderr
			cmdErr = exitErr
		}
	}()

	select {
	case <-done:
		// Command completed
	case <-time.After(time.Duration(CommandTimeout) * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Sprintf("Error: command timed out (%ds)", CommandTimeout)
	}

	var out strings.Builder
	if len(stdout) > 0 {
		out.Write(stdout)
	}
	if len(stderr) > 0 {
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString("STDERR: ")
		out.Write(stderr)
	}
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			out.WriteString(fmt.Sprintf("\n(exit code %d)", exitErr.ExitCode()))
		}
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return "(no output)"
	}
	return result
}

func (t *ToolExecutor) listSubagents(args map[string]any) string {
	files, err := filepath.Glob(filepath.Join(SubagentDir, "agent_*.json"))
	if err != nil {
		return fmt.Sprintf("Error reading subagent directory: %v", err)
	}

	if len(files) == 0 {
		return "No subagents found"
	}

	var results []string
	for _, file := range files {
		// Extract agent ID from filename (e.g., "agent_1.json" -> "1")
		filename := filepath.Base(file)
		idMatch := fileNameRegex.FindStringSubmatch(filename)
		agentID := idMatch[1]

		// Read the file to get status and task info
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}

		status := getStringArg(result, "status", "")
		task := truncateString(getStringArg(result, "task", ""), 40)
		info, _ := os.Stat(file)
		modTime := info.ModTime().Format("15:04:05")

		results = append(results, fmt.Sprintf("#%s [%s] %s (updated: %s)", agentID, status, task, modTime))
	}

	return "Active subagents:\n" + strings.Join(results, "\n")
}

func (t *ToolExecutor) readSubagentResult(args map[string]any) string {
	agentID := getIntArg(args, "id", 0)
	if agentID == 0 {
		return "Error: required parameter 'id' is missing"
	}

	resultFile := filepath.Join(SubagentDir, fmt.Sprintf("agent_%d.json", agentID))
	data, err := os.ReadFile(resultFile)
	if err != nil {
		return fmt.Sprintf("Error reading subagent result: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Sprintf("Error parsing result: %v", err)
	}

	output := fmt.Sprintf("Sub-agent #%d Result:\n", agentID)
	output += fmt.Sprintf("  Task: %s\n", getStringArg(result, "task", ""))
	output += fmt.Sprintf("  Model: %s\n", getStringArg(result, "model", ""))
	output += fmt.Sprintf("  Status: %s\n", getStringArg(result, "status", ""))
	output += fmt.Sprintf("  Result: %s\n", getStringArg(result, "result", ""))

	return output
}

func (t *ToolExecutor) summarizeSubagents(args map[string]any) string {
	files, err := filepath.Glob(filepath.Join(SubagentDir, "agent_*.json"))
	if err != nil {
		return fmt.Sprintf("Error reading subagent directory: %v", err)
	}

	if len(files) == 0 {
		return "No subagents found"
	}

	completed := 0
	errors := 0
	var summaries []string

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}

		id := getIntArg(result, "id", 0)
		status := getStringArg(result, "status", "")
		task := getStringArg(result, "task", "")

		if status == "complete" {
			completed++
		} else if status == "error" {
			errors++
		}

		summaries = append(summaries, fmt.Sprintf("  #%d [%s]: %s", id, status, truncateString(task, 50)))
	}

	output := fmt.Sprintf("Subagent Summary (%d total):\n", len(files))
	output += fmt.Sprintf("  Completed: %d\n", completed)
	output += fmt.Sprintf("  Errors: %d\n", errors)
	output += "\nRecent subagents:\n" + strings.Join(summaries, "\n")

	return output
}

func (t *ToolExecutor) spawnSubagent(args map[string]any) string {
	// The tool definition uses "prompt" as the parameter name
	prompt := getStringArg(args, "prompt", "")
	if prompt == "" {
		return "Error: 'prompt' parameter is required"
	}

	name := getStringArg(args, "name", "")

	// Actually spawn the subagent using the agent if available
	if t.agent != nil {
		// Build a task description from prompt, optionally with name/description
		task := prompt
		if name != "" {
			task = fmt.Sprintf("[%s] %s", name, prompt)
		}
		return t.agent.spawnSubagent(task, "")
	}

	return "Error: no agent context"
}

func (t *ToolExecutor) listModels() string {
	if t.agent != nil {
		models := t.agent.ollama.ListModels()
		if len(models) == 0 {
			return "No models found"
		}
		return strings.Join(models, "\n")
	}
	return "Error: no agent context"
}

func (t *ToolExecutor) switchModel(args map[string]any) string {
	model := getStringArg(args, "model", "")
	if t.agent != nil {
		return t.agent.switchModel(model)
	}
	return "Error: no agent context"
}

func (t *ToolExecutor) restart(args map[string]any) string {
	// Get the current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Sprintf("Error getting executable path: %v", err)
	}

	// Get the directory containing main.go (current working directory)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("Error getting current directory: %v", err)
	}

	fmt.Fprintf(os.Stderr, "[RESTART] Rebuilding YOLO from source...\n")

	// Build command - build the whole package (not just main.go) to the same executable name
	buildCmd := exec.Command("go", "build", "-o", filepath.Base(exePath), ".")
	buildCmd.Dir = cwd

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Build failed: %v\n%s", err, string(output))
	}

	fmt.Fprintf(os.Stderr, "[RESTART] Build successful. Replacing current process...\n")

	// Get the full path to the new executable in cwd
	newExePath := filepath.Join(cwd, filepath.Base(exePath))

	// Filter out --restart flag to avoid double-compile on restart
	var executableArgs []string
	for _, arg := range os.Args[1:] {
		if arg != "--restart" {
			executableArgs = append(executableArgs, arg)
		}
	}

	// Restore terminal state before exec so it's not left in raw mode
	if t.agent != nil && t.agent.inputMgr != nil {
		t.agent.inputMgr.Stop()
	}

	// Use syscall.Exec to replace current process with the new binary
	// This properly replaces the process without spawning a child
	err = syscall.Exec(newExePath, append([]string{filepath.Base(exePath)}, executableArgs...), os.Environ())
	if err != nil {
		return fmt.Sprintf("Failed to exec new process: %v", err)
	}

	// Should never reach here if exec succeeds
	return "Process replaced"
}

