package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// EmailMessage represents a parsed email from the mailbox
type EmailMessage struct {
	From        string   `json:"from"`
	Subject     string   `json:"subject"`
	Date        string   `json:"date"`
	Content     string   `json:"content"`
	Filename    string   `json:"filename"`
	ContentType string   `json:"content_type"`
	Size        int64    `json:"size"`
	To          []string `json:"to,omitempty"`
}

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
	toolDef("list_files", "List files matching a glob pattern. Use **/*.ext to search recursively; plain *.ext only matches the top-level directory.",
		map[string]ToolParam{
			"pattern": {Type: "string", Description: "Glob pattern (default: *). Use **/*.ext for recursive matching."},
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
	toolDef("read_webpage", "Fetch a webpage URL and return its text content. HTML is converted to plain text. Useful for reading documentation, articles, or any web page.",
		map[string]ToolParam{
			"url": {Type: "string", Description: "URL to fetch (required). Will be prefixed with https:// if no scheme is provided."},
		}, []string{"url"}),
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
	toolDef("process_inbox_with_response", "Process all inbound emails: read each email, compose an auto-response, send it back to the sender, then delete the original message. This implements the complete email handling workflow: check → respond → delete. Use this tool to automatically handle incoming emails.",
		map[string]ToolParam{},
		nil),
	toolDef("add_todo", "Add a new item to the todo list",
		map[string]ToolParam{
			"title": {Type: "string", Description: "Title/description of the todo item (required)"},
		}, []string{"title"}),
	toolDef("complete_todo", "Mark a todo item as completed by title",
		map[string]ToolParam{
			"title": {Type: "string", Description: "Title of the todo item to complete (required)"},
		}, []string{"title"}),
	toolDef("delete_todo", "Delete a todo item by title (removes it entirely)",
		map[string]ToolParam{
			"title": {Type: "string", Description: "Title of the todo item to delete (required)"},
		}, []string{"title"}),
	toolDef("list_todos", "List all todos (pending and completed) from .todo.json file",
		map[string]ToolParam{},
		nil),
	toolDef("check_ollama_status", "Check Ollama server status and read debug logs. Returns whether Ollama is running, recent log lines, and any errors found.",
		map[string]ToolParam{
			"lines": {Type: "integer", Description: "Number of log lines to return (default: 50)"},
		}, nil),
	toolDef("playwright_mcp", "Playwright MCP for browser automation. Navigate URLs, interact with DOM elements, fill forms, take screenshots, and extract page content.",
		map[string]ToolParam{
			"action":    {Type: "string", Description: "Action to perform: navigate, click, fill, getHTML, screenshot"},
			"url":       {Type: "string", Description: "URL to navigate to (required for navigate action)"},
			"waitUntil": {Type: "string", Description: "When to consider navigation complete (default: domcontentloaded). Options: load, domcontentloaded, networkidle, commit"},
			"selector":  {Type: "string", Description: "CSS selector for element interaction (required for click, fill, getHTML actions)"},
			"value":     {Type: "string", Description: "Text value to fill into input field (required for fill action)"},
			"timeout":   {Type: "integer", Description: "Timeout in milliseconds for operations (default: 5000)"},
			"path":      {Type: "string", Description: "File path for screenshot output (default: /tmp/screenshot.png)"},
		}, []string{"action"}),
}

// ─── Tool Executor ───────────────────────────────────────────────────

// validTools is the canonical list of tool names recognised by Execute.
// It is also used by parseTextToolCalls to filter bracket-format matches.
var validTools = []string{
	"read_file", "write_file", "edit_file", "list_files",
	"search_files", "run_command", "spawn_subagent",
	"list_subagents", "read_subagent_result", "summarize_subagents",
	"list_models", "switch_model", "think", "restart",
	"make_dir", "remove_dir", "copy_file", "move_file", "reddit", "gog", "web_search", "read_webpage", "send_email", "send_report", "check_inbox", "process_inbox_with_response", "add_todo", "complete_todo", "delete_todo", "list_todos", "check_ollama_status", "playwright_mcp",
}

// fileNameRegex extracts the agent ID from filenames like "agent_1.json"
var fileNameRegex = regexp.MustCompile(`^agent_(\d+)\.json$`)

// subagentTools is the subset of ollamaTools exposed to sub-agents.
var subagentToolNames = map[string]bool{
	"read_file": true, "write_file": true, "edit_file": true,
	"list_files": true, "search_files": true, "run_command": true,
	"think": true, "make_dir": true, "remove_dir": true,
	"copy_file": true, "move_file": true, "reddit": true,
	"gog": true, "web_search": true, "read_webpage": true,
}

// emailToolNames extends subagentToolNames with todo tools.
var emailToolNames = map[string]bool{
	"read_file": true, "write_file": true, "edit_file": true,
	"list_files": true, "search_files": true, "run_command": true,
	"think": true, "make_dir": true, "remove_dir": true,
	"copy_file": true, "move_file": true, "reddit": true,
	"gog": true, "web_search": true, "read_webpage": true,
	"add_todo": true, "complete_todo": true, "delete_todo": true, "list_todos": true,
}

// SubagentTools returns the ToolDef slice for sub-agents.
func SubagentTools() []ToolDef {
	var tools []ToolDef
	for _, td := range ollamaTools {
		if subagentToolNames[td.Function.Name] {
			tools = append(tools, td)
		}
	}
	return tools
}

// EmailTools returns the ToolDef slice for email response generation.
func EmailTools() []ToolDef {
	var tools []ToolDef
	for _, td := range ollamaTools {
		if emailToolNames[td.Function.Name] {
			tools = append(tools, td)
		}
	}
	return tools
}

// errorMessage creates a standardized error string prefixed with "Error: ".
// The LLM recognizes this prefix and uses it to self-correct.
func errorMessage(format string, args ...any) string {
	return fmt.Sprintf("Error: "+format, args...)
}

// ─── Tool Executor Core ─────────────────────────────────────────────

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
func (t *ToolExecutor) safePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("path '%s' must be relative, not absolute", path)
	}

	full := filepath.Clean(filepath.Join(t.baseDir, path))

	baseWithSep := t.baseDir + string(filepath.Separator)
	if full != t.baseDir && !strings.HasPrefix(full, baseWithSep) {
		return "", fmt.Errorf("path '%s' is outside working directory", path)
	}

	return full, nil
}

// Execute dispatches a tool call by name.
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
	case "read_webpage":
		return t.readWebpage(args)
	case "send_email":
		return t.sendEmail(args)
	case "send_report":
		return t.sendReport(args)
	case "check_inbox":
		return t.checkInbox(args)
	case "process_inbox_with_response":
		return t.processInboxWithResponse(args)
	case "add_todo":
		return t.addTodo(args)
	case "complete_todo":
		return t.completeTodo(args)
	case "delete_todo":
		return t.deleteTodo(args)
	case "list_todos":
		return t.listTodosTool(args)
	case "check_ollama_status":
		return t.checkOllamaStatus(args)
	case "playwright_mcp":
		return t.playwrightMCP(args)
	default:
		return errorMessage("unknown tool '%s'. Available tools: %s", name, strings.Join(validTools, ", "))
	}
}

// executeWithTimeout runs a tool with a ToolTimeout-second deadline.
func executeWithTimeout(te *ToolExecutor, name string, args map[string]any) string {
	type result struct{ s string }
	done := make(chan result, 1)

	go func() {
		done <- result{te.Execute(name, args)}
	}()

	timeout := time.Duration(ToolTimeout) * time.Second
	switch name {
	case "process_inbox_with_response":
		timeout = 4 * time.Hour
	}

	select {
	case r := <-done:
		return r.s
	case <-time.After(timeout):
		return fmt.Sprintf("Error: tool '%s' timed out after %v (possible deadlock or hang). "+
			"The tool did not respond and has been abandoned. "+
			"Avoid calling this tool again with the same arguments.", name, timeout)
	}
}

// ─── Argument Helpers ───────────────────────────────────────────────

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
			count, _ := fmt.Sscanf(val, "%d", &n)
			if count == 1 {
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

// isBinaryData checks if the given data appears to be binary (not text).
func isBinaryData(data []byte) bool {
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
