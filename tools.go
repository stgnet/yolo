package main

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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
	toolDef("reddit", "Fetch posts from Reddit using the public API (no auth required). Can search, list subreddit posts, or get thread details.",
		map[string]ToolParam{
			"action":    {Type: "string", Description: "Action: 'search' (query Reddit), 'subreddit' (list posts from subreddit), 'thread' (get specific post/comments)"},
			"subreddit": {Type: "string", Description: "Subreddit name without 'r/' (e.g., 'golang') - required for 'subreddit' action"},
			"query":     {Type: "string", Description: "Search query - required for 'search' action"},
			"post_id":   {Type: "string", Description: "Post/comment ID for 'thread' action"},
			"limit":     {Type: "integer", Description: "Max results to return (default: 25, max: 100)"},
		}, []string{"action"}),
	toolDef("gog", "Google CLI tool for Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts, Tasks, People, Chat, Classroom. Use 'command' parameter to pass gog subcommands (e.g., 'gmail search inbox:unread', 'calendar list events', 'drive list'). Output is JSON by default.",
		map[string]ToolParam{
			"command": {Type: "string", Description: "gog subcommand and arguments (e.g., 'gmail search newer_than:1d --max 5', 'calendar list events', 'drive list')"},
		}, []string{"command"}),
	toolDef("web_search", "Search the web using DuckDuckGo. Returns instant answers, related topics, and abstract snippets from search results.",
		map[string]ToolParam{
			"query": {Type: "string", Description: "Search query (required)"},
			"count": {Type: "integer", Description: "Number of results to return (default: 5, max: 10)"},
		}, []string{"query"}),
	toolDef("learn", "Autonomously research and discover self-improvement opportunities from the internet. Uses web search and Reddit to find new features, best practices, and improvements for the YOLO agent.",
		map[string]ToolParam{}, nil),
	toolDef("send_email", "Send an email via sendmail from yolo@b-haven.org. Postfix handles DKIM signing automatically.",
		map[string]ToolParam{
			"to":      {Type: "string", Description: "Recipient email address (default: scott@stg.net)"},
			"subject": {Type: "string", Description: "Email subject (required)"},
			"body":    {Type: "string", Description: "Email body (required)"},
		}, []string{"subject", "body"}),
	toolDef("send_report", "Send a progress report email to scott@stg.net from yolo@b-haven.org. Postfix handles DKIM signing automatically.",
		map[string]ToolParam{
			"subject": {Type: "string", Description: "Report subject (default: YOLO Progress Report)"},
			"body":    {Type: "string", Description: "Report body (required)"},
		}, []string{"body"}),
	toolDef("check_inbox", "Read emails from Maildir inbox at /var/mail/b-haven.org/yolo/new/",
		map[string]ToolParam{
			"mark_read": {Type: "boolean", Description: "If true, move processed emails to cur/ directory"},
		}, nil),
}

// ─── Tool Executor ───────────────────────────────────────────────────

// validTools is the canonical list of tool names recognised by Execute.
// It is also used by parseTextToolCalls to filter bracket-format matches.
var validTools = []string{
	"read_file", "write_file", "edit_file", "list_files",
	"search_files", "run_command", "spawn_subagent",
	"list_subagents", "read_subagent_result", "summarize_subagents",
	"list_models", "switch_model", "think", "restart",
	"make_dir", "remove_dir", "copy_file", "move_file", "reddit", "gog", "web_search", "learn", "send_email", "send_report", "check_inbox",
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
	case "reddit":
		return t.reddit(args)
	case "gog":
		return t.gog(args)
	case "web_search":
		return t.webSearch(args)
	case "learn":
		return t.learn(args)
	case "send_email":
		return t.sendEmail(args)
	case "send_report":
		return t.sendReport(args)
	case "check_inbox":
		return t.checkInbox(args)
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

func getBoolArg(args map[string]any, key string, fallback bool) bool {
	if v, ok := args[key]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			return strings.ToLower(val) == "true" || strings.ToLower(val) == "yes" || val == "1"
		case float64:
			return val == 1.0
		case int:
			return val == 1
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

// globFiles is a standalone helper for recursive glob pattern matching
func globFiles(baseDir, pattern string) ([]string, error) {
	var matches []string

	// Handle patterns like **/*.txt or **/directory/*
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

	// Run in a new session so child processes have no controlling terminal.
	// This prevents programs like ssh/git from opening /dev/tty directly to
	// prompt for passwords/passphrases, which would steal keystrokes from
	// yolo and leak output onto the user's terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	// Explicitly connect stdin to /dev/null so child processes that try to
	// read input will get immediate EOF instead of hanging.
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stdin = devNull
		defer devNull.Close()
	}

	// Capture stderr explicitly so it is always available, not just on
	// non-zero exit.  This also ensures stderr never leaks to the terminal.
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	done := make(chan struct{})
	var stdout []byte
	var cmdErr error

	go func() {
		defer close(done)
		stdout, cmdErr = cmd.Output()
	}()

	select {
	case <-done:
		// Command completed
	case <-time.After(time.Duration(CommandTimeout) * time.Second):
		if cmd.Process != nil {
			// Kill the entire process group (negative PID) since we used Setsid.
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return fmt.Sprintf("Error: command timed out (%ds)", CommandTimeout)
	}

	var out strings.Builder
	if len(stdout) > 0 {
		out.Write(stdout)
	}
	stderrStr := stderrBuf.String()
	if len(stderrStr) > 0 {
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString("STDERR: ")
		out.WriteString(stderrStr)
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
	// Strip standalone \r to prevent line overwrites in terminal output.
	// \r\n → \n (preserving real newlines), then remove remaining \r.
	result = strings.ReplaceAll(result, "\r\n", "\n")
	result = strings.ReplaceAll(result, "\r", "")
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
		if len(idMatch) < 2 {
			continue
		}
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
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
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

	// Fully isolate: new session (no controlling terminal) + stdin from /dev/null.
	buildCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if devNull, derr := os.Open(os.DevNull); derr == nil {
		buildCmd.Stdin = devNull
		defer devNull.Close()
	}

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

// ─── Reddit Tool Implementation ────────────────────────────────────────

// redditPost represents a Reddit post/comment structure for parsing API responses
type redditPost struct {
	Kind        string     `json:"kind"`
	Data        redditData `json:"data"`
	IsSelf      bool       `json:"is_self"`
	Subreddit   string     `json:"subreddit"`
	Title       string     `json:"title,omitempty"`
	Selftext    string     `json:"selftext,omitempty"`
	Body        string     `json:"body,omitempty"` // For comments
	URL         string     `json:"url,omitempty"`
	Score       int        `json:"score,omitempty"`
	NumComments int        `json:"num_comments,omitempty"`
	Created     float64    `json:"created_utc,omitempty"`
	ID          string     `json:"id,omitempty"`
	Author      string     `json:"author,omitempty"`
}

type redditData struct {
	Title       string  `json:"title,omitempty"`
	Selftext    string  `json:"selftext,omitempty"`
	Body        string  `json:"body,omitempty"`
	URL         string  `json:"url,omitempty"`
	Score       int     `json:"score,omitempty"`
	NumComments int     `json:"num_comments,omitempty"`
	Created     float64 `json:"created_utc,omitempty"`
	ID          string  `json:"id,omitempty"`
	Author      string  `json:"author,omitempty"`
	Subreddit   string  `json:"subreddit,omitempty"`
}

type redditListing struct {
	Kind string         `json:"kind"`
	Data redditChildren `json:"data"`
}

type redditChildren struct {
	Children []redditPostWrapper `json:"children"`
	After    string              `json:"after"`
	Before   string              `json:"before"`
}

type redditPostWrapper struct {
	Kind string     `json:"kind"`
	Data redditPost `json:"data"`
}

// reddit fetches data from Reddit's public API
func (t *ToolExecutor) reddit(args map[string]any) string {
	action := getStringArg(args, "action", "")
	if action == "" {
		return "Error: 'action' parameter is required. Options: 'search', 'subreddit', 'thread'"
	}

	limit := getIntArg(args, "limit", 25)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 25
	}

	var requestURL string
	var err error

	switch action {
	case "search":
		query := getStringArg(args, "query", "")
		if query == "" {
			return "Error: 'query' parameter is required for 'search' action"
		}
		requestURL = fmt.Sprintf("https://www.reddit.com/search.json?q=%s&limit=%d",
			url.QueryEscape(query), limit)

	case "subreddit":
		subreddit := getStringArg(args, "subreddit", "")
		if subreddit == "" {
			return "Error: 'subreddit' parameter is required for 'subreddit' action"
		}
		// Clean subreddit name - remove r/ prefix if present
		subreddit = strings.TrimPrefix(subreddit, "r/")
		requestURL = fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", subreddit, limit)

	case "thread":
		postID := getStringArg(args, "post_id", "")
		if postID == "" {
			return "Error: 'post_id' parameter is required for 'thread' action"
		}
		requestURL = fmt.Sprintf("https://www.reddit.com/comments/%s/.json", postID)

	default:
		return fmt.Sprintf("Error: unknown action '%s'. Options: 'search', 'subreddit', 'thread'", action)
	}

	// Make HTTP request with timeout
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return fmt.Sprintf("Error creating request: %v", err)
	}

	// Add User-Agent header (Reddit requires it)
	req.Header.Set("User-Agent", "YOLO-Agent/1.0 (by /u/yolo)")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error fetching from Reddit: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("Error: Reddit returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err)
	}

	// Parse based on action type
	switch action {
	case "thread":
		return t.parseThreadResponse(action, body)
	default:
		return t.parseListingResponse(action, body)
	}
}

func (t *ToolExecutor) parseListingResponse(action string, data []byte) string {
	var listing redditListing

	if err := json.Unmarshal(data, &listing); err != nil {
		return fmt.Sprintf("Error parsing JSON: %v", err)
	}

	if len(listing.Data.Children) == 0 {
		return "No results found"
	}

	var sb strings.Builder

	switch action {
	case "search":
		sb.WriteString(fmt.Sprintf("Search results (showing %d of available):\n\n", len(listing.Data.Children)))
	case "subreddit":
		subreddit := listing.Data.Children[0].Data.Subreddit
		sb.WriteString(fmt.Sprintf("Hot posts in r/%s (showing %d):\n\n", subreddit, len(listing.Data.Children)))
	}

	for i, post := range listing.Data.Children {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		title := post.Data.Title
		if title == "" && post.Data.Selftext != "" {
			// Self post without title - use first line of content
			lines := strings.SplitN(post.Data.Selftext, "\n", 2)
			title = "[Self Post] " + lines[0]
		}

		sb.WriteString(fmt.Sprintf("**%s**\n", title))

		if post.Data.Author != "" {
			sb.WriteString(fmt.Sprintf("By: u/%s | Score: %d | Comments: %d\n",
				post.Data.Author, post.Data.Score, post.Data.NumComments))
		}

		if post.Data.URL != "" && !strings.Contains(post.Data.URL, "reddit.com") {
			sb.WriteString(fmt.Sprintf("URL: %s\n", post.Data.URL))
		}

		if post.Data.Selftext != "" {
			// Truncate selftext if too long
			text := strings.TrimSpace(post.Data.Selftext)
			if len(text) > 500 {
				text = text[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("\n%s\n", text))
		}

		// Add Reddit link
		postURL := fmt.Sprintf("https://www.reddit.com%s", post.Data.URL)
		if !strings.HasPrefix(post.Data.URL, "/") {
			postURL = fmt.Sprintf("https://www.reddit.com/r/%s/comments/%s/",
				post.Data.Subreddit, post.Data.ID)
		}
		sb.WriteString(fmt.Sprintf("\n[Read more](%s)", postURL))
	}

	return sb.String()
}

func (t *ToolExecutor) parseThreadResponse(action string, data []byte) string {
	// Thread responses are nested - we get the post + comments tree
	var listing redditListing

	if err := json.Unmarshal(data, &listing); err != nil {
		return fmt.Sprintf("Error parsing JSON: %v", err)
	}

	if len(listing.Data.Children) == 0 {
		return "No results found"
	}

	// First child is usually the original post
	originalPost := listing.Data.Children[0]

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n", originalPost.Data.Title))
	sb.WriteString(fmt.Sprintf("By: u/%s | Score: %d | Posted: %s\n\n",
		originalPost.Data.Author,
		originalPost.Data.Score,
		formatRedditTimestamp(originalPost.Data.Created)))

	if originalPost.Data.Selftext != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", strings.TrimSpace(originalPost.Data.Selftext)))
	}

	if originalPost.Data.URL != "" && !strings.Contains(originalPost.Data.URL, "reddit.com") {
		sb.WriteString(fmt.Sprintf("[External Link](%s)\n\n", originalPost.Data.URL))
	}

	// Now process comments (remaining children are top-level comments)
	if len(listing.Data.Children) > 1 {
		sb.WriteString("\n## Top Comments:\n\n")

		for i := 1; i < len(listing.Data.Children); i++ {
			comment := listing.Data.Children[i]
			t.appendComment(&sb, comment.Data, 0)
			if i < len(listing.Data.Children)-1 {
				sb.WriteString("\n---\n\n")
			}
		}
	}

	return sb.String()
}

func (t *ToolExecutor) appendComment(sb *strings.Builder, post redditPost, depth int) {
	if depth > 3 {
		return // Limit nesting depth
	}

	indent := strings.Repeat("  ", depth)

	// Try Body first (for comments), then Selftext (for posts)
	body := strings.TrimSpace(post.Body)
	if body == "" {
		body = strings.TrimSpace(post.Data.Body)
	}
	if body == "" {
		body = strings.TrimSpace(post.Selftext)
	}
	if body == "" && post.Kind != "t1" {
		body = "[Deleted or removed]"
	}

	// Use Author and Score directly from the outer struct (for comments)
	author := post.Author
	score := post.Score

	sb.WriteString(fmt.Sprintf("%s**%s** (%d points)\n", indent, author, score))
	if body != "" {
		// Truncate long comments
		if len(body) > 1000 {
			body = body[:1000] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", indent, body))
	}
}

func formatRedditTimestamp(timestamp float64) string {
	t := time.Unix(int64(timestamp), 0)
	return t.Format("January 2, 2006 at 3:04 PM MST")
}

// ─── Web Search Tool Implementation ────┬────────────────────────────

// webSearchResult represents a search result from DuckDuckGo Instant Answer
type webSearchResult struct {
	Abstract       string `json:"abstract"`
	AbstractSource string `json:"abstract_source"`
	AbstractURL    string `json:"abstract_url"`
	Url            string `json:"url"`
	Image          string `json:"image"`
	RelatedTopics  []struct {
		Title     string `json:"text,omitempty"`
		TopicName string `json:"topic_name"`
		Content   struct {
			Text string `json:"text"`
		} `json:"text_content,omitempty"`
		FirstValue string `json:"first_value"`
	} `json:"related_topics,omitempty"`
}

// searchCacheEntry represents a cached web search result
type searchCacheEntry struct {
	Result string    `json:"result"`
	Ts     time.Time `json:"ts"` // timestamp when cached
}

// searchCache is a thread-safe in-memory cache for web search results
var searchCache = &sync.Map{}
var searchCacheTTL = 5 * time.Minute // Cache entries expire after 5 minutes

// getSearchCacheKey generates a unique key for caching
func getSearchCacheKey(query string, count int) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%d", strings.ToLower(query), count)))
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes of MD5 as key
}

// getFromSearchCache retrieves a cached result if available and not expired
func (t *ToolExecutor) getFromSearchCache(key string) (string, bool) {
	if entry, ok := searchCache.Load(key); ok {
		if e, ok := entry.(*searchCacheEntry); ok {
			if time.Since(e.Ts) < searchCacheTTL {
				return e.Result, true
			}
		}
		searchCache.Delete(key) // Remove expired entry
	}
	return "", false
}

// addToSearchCache stores a result in the cache
func (t *ToolExecutor) addToSearchCache(key, result string) {
	if result != "" && !t.isEmptySearchResult(result) {
		searchCache.Store(key, &searchCacheEntry{
			Result: result,
			Ts:     time.Now(),
		})
	}
}

// webSearch performs a web search using DuckDuckGo's Instant Answer API with Wikipedia fallback
func (t *ToolExecutor) webSearch(args map[string]any) string {
	query := getStringArg(args, "query", "")
	if query == "" {
		return "Error: 'query' parameter is required"
	}

	count := getIntArg(args, "count", 5)
	if count > 10 {
		count = 10
	}
	if count < 1 {
		count = 5
	}

	cacheKey := getSearchCacheKey(query, count)

	// Check cache first
	if cachedResult, ok := t.getFromSearchCache(cacheKey); ok {
		return fmt.Sprintf("[Cached] %s", cachedResult)
	}

	// Try DuckDuckGo first
	ddgResult := t.searchDuckDuckGo(query, count)

	// If DuckDuckGo returned meaningful results, use them
	if !t.isEmptySearchResult(ddgResult) {
		t.addToSearchCache(cacheKey, ddgResult)
		return ddgResult
	}

	// Fallback to Wikipedia API
	wikiResult := t.searchWikipedia(query, count)

	// If Wikipedia also failed or has no results, combine both
	if t.isEmptySearchResult(wikiResult) {
		return fmt.Sprintf("No search results found for \"%s\". DuckDuckGo and Wikipedia returned no relevant information.\n\nTry:\n- Using more specific keywords\n- Searching for a different topic\n- Checking spelling of terms", query)
	}

	// Combine both if DuckDuckGo had partial info
	if ddgResult != "" && !t.isEmptySearchResult(ddgResult) {
		combined := ddgResult + "\n---\n\n" + wikiResult
		t.addToSearchCache(cacheKey, combined)
		return combined
	}

	t.addToSearchCache(cacheKey, wikiResult)
	return wikiResult
}

func (t *ToolExecutor) isEmptySearchResult(result string) bool {
	emptyPatterns := []string{
		"No results found",
		"Error:",
		"returned no relevant information",
		"Try a different search term",
	}

	for _, pattern := range emptyPatterns {
		if strings.Contains(result, pattern) {
			return true
		}
	}

	// Check if result is just a header with minimal content
	if len(result) < 100 {
		return true
	}

	return false
}

func (t *ToolExecutor) searchDuckDuckGo(query string, count int) string {
	// Use retry logic for transient failures
	result := t.searchDuckDuckGoWithRetry(query, count, 3)
	return result
}

func (t *ToolExecutor) searchDuckDuckGoWithRetry(query string, count int, maxRetries int) string {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		url := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1",
			url.QueryEscape(query))

		client := &http.Client{Timeout: 15 * time.Second}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("error creating DuckDuckGo request: %v", err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; YOLO-Search-Bot/1.0)")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error fetching from DuckDuckGo: %v", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(delay)
			}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("error reading DuckDuckGo response: %v", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(delay)
			}
			continue
		}

		// Try to parse as JSON first (Instant Answer format)
		var iaResult map[string]any
		if err := json.Unmarshal(body, &iaResult); err == nil {
			result := t.parseDuckDuckGoJSON(query, count, body)
			if !t.isEmptySearchResult(result) {
				return result
			}
		}

		// Fall back to HTML parsing (won't work for API endpoint, but kept for completeness)
		return ""
	}

	if lastErr != nil {
		return fmt.Sprintf("Error: DuckDuckGo search failed after %d retries: %v", maxRetries+1, lastErr)
	}

	return ""
}

func (t *ToolExecutor) searchWikipedia(query string, count int) string {
	// Use retry logic for transient failures
	result := t.searchWikipediaWithRetry(query, count, 3)
	return result
}

func (t *ToolExecutor) searchWikipediaWithRetry(query string, count int, maxRetries int) string {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wikipedia Search API - searches titles and content
		urlStr := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&format=json&origin=*&srlimit=%d",
			url.QueryEscape(query), count)

		client := &http.Client{Timeout: 15 * time.Second}
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			lastErr = fmt.Errorf("error creating Wikipedia request: %v", err)
			continue
		}

		req.Header.Set("User-Agent", "YOLO-Search-Bot/1.0 (yolo@b-haven.org)")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error fetching from Wikipedia: %v", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(delay)
			}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("error reading Wikipedia response: %v", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(delay)
			}
			continue
		}

		var result struct {
			Query struct {
				Search []struct {
					Title    string `json:"title"`
					PageID   int    `json:"pageid"`
					Snippet  string `json:"snippet"`
					Fragment string `json:"fragment"`
				} `json:"search"`
			} `json:"query"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = fmt.Errorf("error parsing Wikipedia JSON: %v", err)
			continue
		}

		if len(result.Query.Search) == 0 {
			return ""
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Wikipedia results for \"%s\":\n\n", query))

		for i, article := range result.Query.Search {
			sb.WriteString(fmt.Sprintf("%d. **[%s](https://en.wikipedia.org/wiki/%s)**\n",
				i+1,
				article.Title,
				strings.ReplaceAll(article.Title, " ", "_")))

			// Use fragment if available (shows context around search terms), otherwise snippet
			snippet := article.Snippet
			if article.Fragment != "" {
				snippet = article.Fragment
			}

			// Clean up HTML entities and tags
			snippet = strings.ReplaceAll(snippet, "&amp;", "&")
			snippet = strings.ReplaceAll(snippet, "&lt;", "<")
			snippet = strings.ReplaceAll(snippet, "&gt;", ">")
			snippet = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(snippet, "")

			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}

			sb.WriteString(fmt.Sprintf("   %s\n\n", strings.TrimSpace(snippet)))
		}

		return sb.String()
	}

	if lastErr != nil {
		return fmt.Sprintf("Error: Wikipedia search failed after %d retries: %v", maxRetries+1, lastErr)
	}

	return ""
}

func (t *ToolExecutor) parseDuckDuckGoJSON(query string, count int, data []byte) string {
	var result struct {
		Query          string `json:"query"`
		Results        int    `json:"results"`
		Answer         string `json:"answer"`
		Abstract       string `json:"abstract"`
		AbstractSource string `json:"abstract_source"`
		AbstractURL    string `json:"abstract_url"`
		Image          string `json:"image"`
		RelatedTopics  []struct {
			Title     string          `json:"text,omitempty"`
			TopicName string          `json:"topic_name"`
			Result    json.RawMessage `json:"result,omitempty"`
			Results   []struct {
				Text string `json:"text"`
				Url  string `json:"url"`
			} `json:"results,omitempty"`
		} `json:"related_topics,omitempty"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Sprintf("Error parsing JSON: %v", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for \"%s\":\n\n", query))

	// Direct answer (if any)
	if result.Answer != "" {
		sb.WriteString(fmt.Sprintf("**Answer:** %s\n\n", result.Answer))
	}

	// Abstract/summary
	if result.Abstract != "" {
		sb.WriteString(fmt.Sprintf("**Summary:** %s\n", result.Abstract))
		if result.AbstractSource != "" {
			sb.WriteString(fmt.Sprintf("Source: [from%s](%s)\n\n",
				result.AbstractSource, result.AbstractURL))
		} else {
			sb.WriteString("\n")
		}
	}

	// Image (if any)
	if result.Image != "" {
		sb.WriteString(fmt.Sprintf("[![](%s)](%s)\n\n", result.Image, result.Image))
	}

	// Related topics and results
	resultsCount := 0
	for _, topic := range result.RelatedTopics {
		// Try to extract results from the topic
		var topicResults []struct {
			Text string `json:"text"`
			Url  string `json:"url"`
		}

		// Check if Result field contains raw JSON
		if len(topic.Result) > 0 {
			var singleResult struct {
				Text string `json:"text"`
				Url  string `json:"url"`
			}
			if err := json.Unmarshal(topic.Result, &singleResult); err == nil && singleResult.Text != "" {
				topicResults = append(topicResults, singleResult)
			}
		}

		// Check Results array
		topicResults = append(topicResults, topic.Results...)

		if len(topicResults) > 0 {
			if topic.TopicName != "" || topic.Title != "" {
				title := topic.TopicName
				if title == "" {
					title = topic.Title
				}
				sb.WriteString(fmt.Sprintf("\n### %s\n", title))
			}

			for _, r := range topicResults {
				if resultsCount >= count {
					break
				}
				resultsCount++

				if r.Text != "" {
					sb.WriteString(fmt.Sprintf("%d. **%s**\n", resultsCount, r.Text))
				}
				if r.Url != "" {
					sb.WriteString(fmt.Sprintf("   [%s](%s)\n\n", r.Url, r.Url))
				}
			}
		}
	}

	if resultsCount == 0 && result.Abstract == "" && result.Answer == "" {
		sb.WriteString("No results found for this query. Try a different search term.\n")
	}

	return sb.String()
}

func (t *ToolExecutor) parseDuckDuckGoHTML(query string, count int, data []byte) string {
	// Simple HTML parser - extract snippets and titles from search results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for \"%s\":\n\n", query))

	// Look for result snippets in HTML (DuckDuckGo uses specific classes)
	lines := strings.Split(string(data), "\n")

	type SearchResult struct {
		Title   string
		URL     string
		Snippet string
	}

	var results []SearchResult

	// Parse title links and snippets
	for i, line := range lines {
		// Look for result titles in <a> tags with class containing "result__a"
		if strings.Contains(line, `class="`) && (strings.Contains(line, "result") || strings.Contains(line, "link")) {
			// Extract title
			titleMatch := regexp.MustCompile(`>([^<]+?)<`).FindStringSubmatch(strings.TrimSpace(line))
			if len(titleMatch) > 1 {
				title := titleMatch[1]
				cleanTitle := strings.TrimPrefix(title, "[")
				cleanTitle = strings.TrimSuffix(cleanTitle, "]")

				// Look for URL in this line or nearby lines
				var url string
				startIdx := i - 2
				if startIdx < 0 {
					startIdx = 0
				}
				for j := startIdx; j <= i+2 && j < len(lines); j++ {
					if strings.Contains(lines[j], `href="http`) {
						urlMatch := regexp.MustCompile(`href="(https?://[^"]+)"`).FindStringSubmatch(lines[j])
						if len(urlMatch) > 1 {
							url = urlMatch[1]
							break
						}
					}
				}

				// Look for snippet in following lines
				var snippet string
				for j := i + 1; j < i+5 && j < len(lines); j++ {
					if strings.Contains(lines[j], "<div") || strings.Contains(lines[j], "<span") {
						snippetLines := regexp.MustCompile(`<[^>]+>([^<]+)</[^>]+>`).FindAllStringSubmatch(lines[j], -1)
						for _, sm := range snippetLines {
							if len(sm) > 1 && sm[1] != title && len(sm[1]) > 20 {
								snippet = sm[1]
								break
							}
						}
					}
					if snippet != "" {
						break
					}
				}

				results = append(results, SearchResult{
					Title:   cleanTitle,
					URL:     url,
					Snippet: strings.TrimSpace(snippet),
				})
			}
		}

		if len(results) >= count {
			break
		}
	}

	// Output results
	if len(results) == 0 {
		sb.WriteString("No results found. DuckDuckGo HTML parsing failed.\n")
		return sb.String()
	}

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, r.Title))
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("   [%s](%s)\n", r.URL, r.URL))
		}
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Snippet))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
