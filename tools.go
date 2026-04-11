package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	toolDef("edit_file", "Replace text in a file. By default replaces first occurrence. Use replace_all for all, occurrence for Nth (-1=last).",
		map[string]ToolParam{
			"path":        {Type: "string", Description: "Relative path"},
			"old_text":    {Type: "string", Description: "Text to find"},
			"new_text":    {Type: "string", Description: "Replacement text"},
			"replace_all": {Type: "boolean", Description: "Replace all occurrences (default: false)"},
			"occurrence":  {Type: "integer", Description: "Which occurrence to replace: 0/omit=first, -1=last, N=Nth (default: first)"},
			"dry_run":     {Type: "boolean", Description: "Preview what would change without modifying the file (default: false)"},
		}, []string{"path", "old_text", "new_text"}),
	toolDef("edit_file_lines", "Replace a range of lines in a file with new content. Lines are 1-based.",
		map[string]ToolParam{
			"path":       {Type: "string", Description: "Relative path"},
			"start_line": {Type: "integer", Description: "First line to replace (1-based)"},
			"end_line":   {Type: "integer", Description: "Last line to replace (inclusive)"},
			"content":    {Type: "string", Description: "New content to insert (empty string to delete lines)"},
			"dry_run":    {Type: "boolean", Description: "Preview what would change without modifying the file"},
		}, []string{"path", "start_line", "end_line", "content"}),
	toolDef("patch_file", "Apply a unified diff (like git diff or diff -u) to a file. Supports multiple hunks.",
		map[string]ToolParam{
			"path":    {Type: "string", Description: "Relative path to the file to patch"},
			"diff":    {Type: "string", Description: "Unified diff content with @@ hunk headers and +/- lines"},
			"dry_run": {Type: "boolean", Description: "Preview what would change without modifying the file"},
		}, []string{"path", "diff"}),
	toolDef("undo_edit", "Revert the most recent edit to a file. Only one level of undo is available per file.",
		map[string]ToolParam{
			"path": {Type: "string", Description: "Relative path to the file to revert"},
		}, []string{"path"}),
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
	toolDef("victron", "Connect to and read values from Victron Energy devices via Bluetooth Low Energy (BLE). Supports SmartSolar MPPT charge controllers, SmartShunt battery monitors, and VE.Direct adapters.",
		map[string]ToolParam{
			"action":   {Type: "string", Description: "Action: 'scan' (find nearby devices), 'connect' (establish connection), 'disconnect' (close connection), 'get_values' (read current values), 'subscribe' (monitor real-time updates), 'device_info' (get device details)"},
			"address":  {Type: "string", Description: "Device MAC address - required for 'connect', 'disconnect', 'get_values', 'subscribe', 'device_info' actions"},
			"duration": {Type: "string", Description: "Scan duration in seconds (default: 10, max: 60)"},
			"timeout":  {Type: "string", Description: "Operation timeout in seconds (default varies by action)"},
		}, []string{"action"}),
	toolDef("gog", "Google CLI tool for Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts, Tasks, People, Chat, Classroom. Use 'command' parameter to pass gog subcommands (e.g., 'gmail search inbox:unread', 'calendar list events', 'drive list'). Output is JSON by default.",
		map[string]ToolParam{
			"command": {Type: "string", Description: "gog subcommand and arguments (e.g., 'gmail search newer_than:1d --max 5', 'calendar list events', 'drive list')"},
		}, []string{"command"}),
	toolDef("web_search", "Search the web. Uses SearXNG (if SEARXNG_URL is set) or DuckDuckGo. Returns search results with titles, URLs, and snippets.",
		map[string]ToolParam{
			"query":  {Type: "string", Description: "Search query (required)"},
			"count":  {Type: "integer", Description: "Number of results to return (default: 5, max: 10)"},
			"engine": {Type: "string", Description: "Search engine: 'searxng', 'duckduckgo'/'ddg', or omit for auto (SearXNG if configured, else DDG)"},
		}, []string{"query"}),
	toolDef("read_webpage", "Fetch a webpage URL and extract readable article content. Uses readability-style extraction to remove boilerplate.",
		map[string]ToolParam{
			"url":  {Type: "string", Description: "URL to fetch (required). Will be prefixed with https:// if no scheme is provided."},
			"mode": {Type: "string", Description: "Extraction mode: 'auto' (smart article detection, default), 'article' (strict article only), 'full' (all text)"},
		}, []string{"url"}),
	toolDef("screenshot", "Capture a screenshot of a web page. Requires Playwright (node) or Chromium.",
		map[string]ToolParam{
			"url":       {Type: "string", Description: "URL to screenshot (required)"},
			"path":      {Type: "string", Description: "Output file path (default: temp dir screenshot.png)"},
			"full_page": {Type: "boolean", Description: "Capture full scrollable page (default: false, viewport only)"},
			"width":     {Type: "integer", Description: "Viewport width in pixels (default: 1280)"},
			"height":    {Type: "integer", Description: "Viewport height in pixels (default: 720)"},
		}, []string{"url"}),
	toolDef("send_email", "Send an email via sendmail. Sender and default recipient are configured in config.json.",
		map[string]ToolParam{
			"to":          {Type: "string", Description: "Recipient email address (uses configured default if omitted)"},
			"subject":     {Type: "string", Description: "Email subject (required)"},
			"body":        {Type: "string", Description: "Email body (required)"},
			"attachments": {Type: "array[string]", Description: "List of file paths to attach to the email"},
		}, []string{"subject", "body"}),
	toolDef("send_report", "Send a progress report email. Sender and default recipient are configured in config.json.",
		map[string]ToolParam{
			"subject": {Type: "string", Description: "Report subject (default: YOLO Progress Report)"},
			"body":    {Type: "string", Description: "Report body (required)"},
		}, []string{"body"}),
	toolDef("check_inbox", "Read emails from the configured Maildir inbox (set inbox_path in config.json).",
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
	toolDef("list_todos", "List all todos (pending and completed)",
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
			"path":      {Type: "string", Description: "File path for screenshot output (default: temp dir screenshot.png)"},
		}, []string{"action"}),
	toolDef("memory_read", "Read the agent's curated MEMORY.md file containing durable facts, user preferences, and key decisions.",
		map[string]ToolParam{}, nil),
	toolDef("memory_write", "Replace the contents of MEMORY.md with curated durable facts. Cap: 100 lines. Include user preferences, project conventions, coding style, key decisions.",
		map[string]ToolParam{
			"content": {Type: "string", Description: "Complete new contents for MEMORY.md (max 100 lines)"},
		}, []string{"content"}),
	toolDef("memory_log", "Append an observation to today's daily context log (memory/YYYY-MM-DD.md). Use for raw observations, discoveries, and session notes.",
		map[string]ToolParam{
			"entry": {Type: "string", Description: "Observation or note to log (will be timestamped automatically)"},
		}, []string{"entry"}),
	toolDef("memory_promote", "Retrieve daily logs for review and distillation into MEMORY.md. Returns current MEMORY.md + recent daily logs with instructions to curate.",
		map[string]ToolParam{
			"days": {Type: "integer", Description: "Number of past days to review (default: 7, max: 30)"},
		}, nil),
	toolDef("memory_search", "Search across all memory files (MEMORY.md and daily logs) for a keyword or phrase.",
		map[string]ToolParam{
			"query": {Type: "string", Description: "Search term (case-insensitive)"},
		}, []string{"query"}),
	toolDef("schedule_add", "Add a scheduled task that fires on a cron schedule. The prompt is injected as a system message when the schedule fires.",
		map[string]ToolParam{
			"name":   {Type: "string", Description: "Human-readable name for the schedule"},
			"cron":   {Type: "string", Description: "Cron expression: 'minute hour day month weekday' (e.g., '0 9 * * *' for 9 AM daily, '*/10 * * * *' for every 10 min)"},
			"prompt": {Type: "string", Description: "Task prompt to execute when the schedule fires"},
		}, []string{"name", "cron", "prompt"}),
	toolDef("schedule_remove", "Remove a scheduled task by ID or name.",
		map[string]ToolParam{
			"id": {Type: "string", Description: "Schedule ID or name to remove"},
		}, []string{"id"}),
	toolDef("schedule_list", "List all scheduled tasks with their status, next run time, and run history.",
		map[string]ToolParam{}, nil),
	toolDef("schedule_toggle", "Enable or disable a scheduled task.",
		map[string]ToolParam{
			"id":      {Type: "string", Description: "Schedule ID or name"},
			"enabled": {Type: "boolean", Description: "true to enable, false to disable"},
		}, []string{"id", "enabled"}),
	toolDef("project_map", "Generate a hierarchical file tree of the project with sizes. Useful for understanding codebase structure at a glance.",
		map[string]ToolParam{
			"max_depth":  {Type: "integer", Description: "Maximum directory depth (default: 4)"},
			"show_sizes": {Type: "boolean", Description: "Show file sizes (default: true)"},
			"pattern":    {Type: "string", Description: "Filter files by glob pattern (e.g., '*.go', '*.py')"},
		}, nil),
	toolDef("dependency_graph", "Parse source file imports to show package/module dependency relationships. Supports Go, Python, JavaScript/TypeScript, and Rust.",
		map[string]ToolParam{
			"package": {Type: "string", Description: "Filter to a specific package/directory (optional)"},
		}, nil),
	toolDef("symbol_search", "Search for function, type, class, and variable definitions across the codebase. Supports Go, Python, JS/TS, Rust, Java, C/C++.",
		map[string]ToolParam{
			"query":   {Type: "string", Description: "Symbol name or substring to search for (case-insensitive)"},
			"kind":    {Type: "string", Description: "Filter by kind: func, type, class, var, const (default: all)"},
			"pattern": {Type: "string", Description: "File glob pattern (default: all source files)"},
		}, []string{"query"}),
	toolDef("project_summary", "Get or refresh a cached summary of the project: file counts, line counts, languages, and per-file metadata stored in project-map.json (in YOLO data dir).",
		map[string]ToolParam{
			"refresh": {Type: "boolean", Description: "Force rescan of all files (default: false, uses cache)"},
		}, nil),
	toolDef("history_search", "Search all past conversation history by keyword. Returns matching messages from the full history database, not just the recent context window. Use this to recall past discussions, decisions, lists, or any prior conversation content.",
		map[string]ToolParam{
			"query": {Type: "string", Description: "Search terms (words are ANDed; use OR for alternatives, quotes for exact phrases)"},
			"limit": {Type: "integer", Description: "Maximum results to return (default: 20)"},
		}, []string{"query"}),
	// Git tools - native git integration
	toolDef("git_status", "Show the current git status in a structured format. Lists modified, added, deleted, and untracked files.",
		map[string]ToolParam{}, nil),
	toolDef("git_diff", "Show the diff of changes. Optionally specify a file to show only that file's changes.",
		map[string]ToolParam{
			"file": {Type: "string", Description: "Optional file path to show changes for (relative path)"},
		}, nil),
	toolDef("git_log", "Show recent commit history with oneline format.",
		map[string]ToolParam{
			"limit": {Type: "integer", Description: "Number of commits to show (default: 10)"},
		}, nil),
	toolDef("git_branch", "List all branches with current branch marked.",
		map[string]ToolParam{}, nil),
	toolDef("git_checkout", "Checkout a branch or restore a file from HEAD.",
		map[string]ToolParam{
			"branch": {Type: "string", Description: "Branch name to checkout (if not restoring a file)"},
			"file":   {Type: "string", Description: "File to restore from HEAD (if not switching branches)"},
		}, []string{}),
	toolDef("git_commit", "Commit staged changes with a message.",
		map[string]ToolParam{
			"message": {Type: "string", Description: "Commit message (required)"},
			"all":     {Type: "boolean", Description: "Auto-stage all changes before committing (default: false)"},
		}, []string{"message"}),
	toolDef("git_add", "Stage files for commit. If no file specified, stages all changes.",
		map[string]ToolParam{
			"file": {Type: "string", Description: "File to stage (optional, defaults to .)"},
		}, nil),
	toolDef("git_show", "Show details of a specific commit.",
		map[string]ToolParam{
			"commit": {Type: "string", Description: "Commit hash or reference (default: HEAD)"},
		}, nil),
	toolDef("git_remote", "Show configured git remotes with URLs.",
		map[string]ToolParam{}, nil),
	toolDef("set_config", "Set a YOLO configuration value in config.json. Available keys: model, email_from, email_to, inbox_path, user_agent, temp_dir, tts_voice, terminal_mode, debug_mode, auto_mode, think_mode, tts_enabled.",
		map[string]ToolParam{
			"key":   {Type: "string", Description: "Configuration key to set"},
			"value": {Type: "string", Description: "Value to set"},
		}, []string{"key", "value"}),
	toolDef("get_config", "Get the current YOLO configuration from config.json. Returns all configuration values.",
		map[string]ToolParam{
			"key": {Type: "string", Description: "Specific config key to retrieve (optional, omit to show all)"},
		}, nil),
	toolDef("victron", "Connect to and read values from Victron Energy devices via Bluetooth Low Energy. Supports SmartSolar MPPT charge controllers, SmartShunt battery monitors, and VE.Direct adapters.",
		map[string]ToolParam{
			"action":   {Type: "string", Description: "Action: 'scan' (find nearby devices), 'connect' (establish connection), 'disconnect' (close connection), 'get_values' (read current values), 'subscribe' (monitor real-time updates), 'device_info' (get device details)"},
			"address":  {Type: "string", Description: "Device MAC address - required for 'connect', 'disconnect', 'get_values', 'subscribe', 'device_info' actions"},
			"duration": {Type: "string", Description: "Scan duration in seconds (default: 10, max: 60)"},
			"timeout":  {Type: "string", Description: "Operation timeout in seconds (default varies by action)"},
		}, []string{"action"}),
}

// ─── Tool Executor ───────────────────────────────────────────────────

// validTools is the canonical list of tool names recognised by Execute.
// It is also used by parseTextToolCalls to filter bracket-format matches.
var validTools = []string{
	"read_file", "write_file", "edit_file", "edit_file_lines", "patch_file", "undo_edit", "list_files",
	"search_files", "run_command", "spawn_subagent",
	"list_subagents", "read_subagent_result", "summarize_subagents",
	"list_models", "switch_model", "think", "restart",
	"make_dir", "remove_dir", "copy_file", "move_file", "reddit", "gog", "web_search", "read_webpage", "screenshot", "send_email", "send_report", "check_inbox", "process_inbox_with_response", "add_todo", "complete_todo", "delete_todo", "list_todos", "check_ollama_status", "playwright_mcp",
	"memory_read", "memory_write", "memory_log", "memory_promote", "memory_search",
	"schedule_add", "schedule_remove", "schedule_list", "schedule_toggle",
	"project_map", "dependency_graph", "symbol_search", "project_summary",
	"history_search", "set_config", "get_config",
	"git_status", "git_diff", "git_log", "git_branch", "git_checkout", "git_commit", "git_add", "git_show", "git_remote",
	"victron",
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
	"git_status": true, "git_diff": true, "git_log": true, "git_branch": true, "git_add": true,
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
	if t == nil {
		return "ToolExecutor not initialized"
	}
	switch name {
	case "read_file":
		return t.readFile(args)
	case "write_file":
		return t.writeFile(args)
	case "edit_file":
		return t.editFile(args)
	case "edit_file_lines":
		return t.editFileLines(args)
	case "patch_file":
		return t.patchFile(args)
	case "undo_edit":
		return t.undoEdit(args)
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
		return t.webSearchEnhanced(args)
	case "read_webpage":
		return t.readWebpageReadable(args)
	case "screenshot":
		return t.screenshot(args)
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
	case "memory_read":
		return t.memoryRead(args)
	case "memory_write":
		return t.memoryWrite(args)
	case "memory_log":
		return t.memoryLog(args)
	case "memory_promote":
		return t.memoryPromote(args)
	case "memory_search":
		return t.memorySearch(args)
	case "schedule_add":
		return t.scheduleAdd(args)
	case "schedule_remove":
		return t.scheduleRemove(args)
	case "schedule_list":
		return t.scheduleList(args)
	case "schedule_toggle":
		return t.scheduleToggle(args)
	case "project_map":
		return t.projectMap(args)
	case "dependency_graph":
		return t.dependencyGraph(args)
	case "symbol_search":
		return t.symbolSearch(args)
	case "project_summary":
		return t.projectSummary(args)
	case "history_search":
		return t.historySearch(args)
	case "set_config":
		return t.setConfig(args)
	case "get_config":
		return t.getConfig(args)
	// Git tools
	case "git_status":
		return t.gitStatus(args)
	case "git_diff":
		return t.gitDiff(args)
	case "git_log":
		return t.gitLog(args)
	case "git_branch":
		return t.gitBranch(args)
	case "git_checkout":
		return t.gitCheckout(args)
	case "git_commit":
		return t.gitCommit(args)
	case "git_add":
		return t.gitAdd(args)
	case "git_show":
		return t.gitShow(args)
	case "git_remote":
		return t.gitRemote(args)
	case "victron":
		return t.victron(args)
	default:
		// Check if this is an MCP tool
		if t.agent != nil && t.agent.mcp != nil && t.agent.mcp.IsMCPTool(name) {
			return t.agent.mcp.ExecuteTool(name, args)
		}
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

// getTempDir returns the configured temporary directory from agent config.
func (t *ToolExecutor) getTempDir() string {
	if t.agent != nil && t.agent.config != nil {
		return t.agent.config.GetTempDir()
	}
	return os.TempDir()
}

// getUserAgent returns the configured User-Agent string from agent config.
func (t *ToolExecutor) getUserAgent() string {
	if t.agent != nil && t.agent.config != nil {
		return t.agent.config.GetUserAgent()
	}
	return "YOLO-Agent/1.0"
}

// ─── History Search Tool ────────────────────────────────────────────

func (t *ToolExecutor) historySearch(args map[string]any) string {
	query := getStringArg(args, "query", "")
	if query == "" {
		return errorMessage("'query' parameter is required")
	}
	limit := getIntArg(args, "limit", 20)

	if t.agent == nil || t.agent.history == nil {
		return errorMessage("no history database available")
	}

	results := t.agent.history.Search(query, limit)
	if len(results) == 0 {
		total := t.agent.history.MessageCount()
		return fmt.Sprintf("No results found for '%s' (searched %d messages in history)", query, total)
	}

	var sb strings.Builder
	total := t.agent.history.MessageCount()
	sb.WriteString(fmt.Sprintf("Found %d result(s) for '%s' (searched %d messages):\n\n", len(results), query, total))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("--- Result %d [%s @ %s] ---\n", i+1, r.Message.Role, r.Message.TS))
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("Snippet: %s\n", r.Snippet))
		}
		content := r.Message.Content
		if len(content) > 2000 {
			content = content[:2000] + "\n...(truncated)"
		}
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// ─── Config Tools ───────────────────────────────────────────────────

func (t *ToolExecutor) setConfig(args map[string]any) string {
	key := getStringArg(args, "key", "")
	value := getStringArg(args, "value", "")
	if key == "" {
		return errorMessage("'key' parameter is required")
	}
	if t.agent == nil {
		return errorMessage("no agent context")
	}
	if err := t.agent.config.SetByName(key, value); err != nil {
		return errorMessage("%v", err)
	}
	return fmt.Sprintf("Config '%s' set to '%s'", key, value)
}

func (t *ToolExecutor) getConfig(args map[string]any) string {
	if t.agent == nil {
		return errorMessage("no agent context")
	}
	key := getStringArg(args, "key", "")
	all := t.agent.config.GetAll()
	if key != "" {
		if v, ok := all[key]; ok {
			return fmt.Sprintf("%s = %s", key, v)
		}
		return errorMessage("unknown config key '%s'", key)
	}
	var sb strings.Builder
	sb.WriteString("Current configuration (config.json):\n\n")
	// Sort keys for consistent output
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := all[k]
		if v == "" {
			v = "(not set)"
		}
		sb.WriteString(fmt.Sprintf("  %-15s = %s\n", k, v))
	}
	return sb.String()
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
