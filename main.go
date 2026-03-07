// YOLO - Your Own Living Operator
// A self-evolving AI agent for software development.
// Continuously runs, thinks, and improves — even when you're not typing.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"golang.org/x/term"
)

// ─── Configuration ────────────────────────────────────────────────────

const (
	YoloDir           = ".yolo"
	IdleThinkDelay    = 120 // seconds of no input before autonomous thinking
	ThinkLoopDelay    = 120 // seconds between autonomous think cycles
	MaxContextMessages = 40
	MaxToolOutput     = 0 // 0 = no truncation
	ToolNudgeAfter    = 0 // 0 = disabled
	CommandTimeout    = 30 // shell command timeout in seconds
)

var (
	HistoryFile = filepath.Join(YoloDir, "history.json")
	OllamaURL   = getEnvDefault("OLLAMA_URL", "http://localhost:11434")
	SubagentDir = filepath.Join(YoloDir, "subagents")
)

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ─── ANSI Colors ──────────────────────────────────────────────────────

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
)

func cprint(color, text string) {
	fmt.Printf("%s%s%s\n", color, text, Reset)
}

func cprintNoNL(color, text string) {
	fmt.Printf("%s%s%s", color, text, Reset)
}

// ─── Spinner ──────────────────────────────────────────────────────────

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Spinner struct {
	prefix string
	color  string
	stop   chan struct{}
	done   chan struct{}
}

func NewSpinner(prefix, color string) *Spinner {
	return &Spinner{
		prefix: prefix,
		color:  color,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				frame := spinnerFrames[i%len(spinnerFrames)]
				fmt.Printf("\r%s%s%s thinking...%s", s.color, s.prefix, frame, Reset)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
	clearLen := len(s.prefix) + 20
	fmt.Printf("\r%s\r", strings.Repeat(" ", clearLen))
}

// ─── Ollama Tool Definitions ─────────────────────────────────────────

type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolSchema struct {
	Type       string                `json:"type"`
	Properties map[string]ToolParam  `json:"properties"`
	Required   []string              `json:"required,omitempty"`
}

type ToolFunction struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  ToolSchema `json:"parameters"`
}

type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

func toolDef(name, desc string, props map[string]ToolParam, required []string) ToolDef {
	return ToolDef{
		Type: "function",
		Function: ToolFunction{
			Name:        name,
			Description: desc,
			Parameters: ToolSchema{
				Type:       "object",
				Properties: props,
				Required:   required,
			},
		},
	}
}

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
	toolDef("spawn_subagent", "Spawn a background sub-agent for a parallel task",
		map[string]ToolParam{
			"task":  {Type: "string", Description: "Task description"},
			"model": {Type: "string", Description: "Ollama model name (optional)"},
		}, []string{"task"}),
	toolDef("list_models", "List available Ollama models", map[string]ToolParam{}, nil),
	toolDef("switch_model", "Switch to a different Ollama model",
		map[string]ToolParam{
			"model": {Type: "string", Description: "Model name"},
		}, []string{"model"}),
	toolDef("think", "Record internal reasoning or a plan without taking action",
		map[string]ToolParam{
			"thought": {Type: "string", Description: "Your reasoning"},
		}, []string{"thought"}),
}

// ─── Ollama Client ────────────────────────────────────────────────────

type OllamaClient struct {
	baseURL string
	client  *http.Client
}

func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

type OllamaModel struct {
	Name string `json:"name"`
}

type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

func (c *OllamaClient) ListModels() []string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(c.baseURL + "/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var data OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}
	models := make([]string, len(data.Models))
	for i, m := range data.Models {
		models[i] = m.Name
	}
	return models
}

// Chat message types
type ChatMessage struct {
	Role      string      `json:"role"`
	Content   string      `json:"content"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Function ToolCallFunc `json:"function"`
}

type ToolCallFunc struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Tools    []ToolDef     `json:"tools,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

type StreamResponse struct {
	Message StreamMessage `json:"message"`
	Done    bool          `json:"done"`
}

type StreamMessage struct {
	Thinking  string     `json:"thinking,omitempty"`
	Content   string     `json:"content"`
	ToolCalls []StreamTC `json:"tool_calls,omitempty"`
}

type StreamTC struct {
	Function StreamTCFunc `json:"function"`
}

type StreamTCFunc struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ChatResult holds the result of a chat call
type ChatResult struct {
	DisplayText string
	ContentText string
	ToolCalls   []ParsedToolCall
}

type ParsedToolCall struct {
	Name string
	Args map[string]any
}

func (c *OllamaClient) Chat(ctx context.Context, model string, messages []ChatMessage, tools []ToolDef) (*ChatResult, error) {
	payload := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Options:  map[string]any{"num_ctx": 8192},
	}
	if len(tools) > 0 {
		payload.Tools = tools
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	spinner := NewSpinner("yolo> ", Blue)
	spinner.Start()

	var thinkingParts, contentParts []string
	var toolCalls []ParsedToolCall
	inThinking := false
	gotFirstOutput := false

	scanner := bufio.NewScanner(resp.Body)
	// Increase scanner buffer for large responses
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var obj StreamResponse
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		msg := obj.Message
		thinking := msg.Thinking
		content := msg.Content
		tcList := msg.ToolCalls

		// On first real output, stop the spinner
		if !gotFirstOutput && (thinking != "" || content != "" || len(tcList) > 0) {
			gotFirstOutput = true
			spinner.Stop()
			fmt.Printf("%s%syolo>%s ", Blue, Bold, Reset)
		}

		// Handle thinking tokens
		if thinking != "" {
			if !inThinking {
				fmt.Printf("%s[thinking] ", Gray)
				inThinking = true
			}
			fmt.Print(thinking)
			thinkingParts = append(thinkingParts, thinking)
		}

		// Handle content tokens
		if content != "" {
			if inThinking {
				fmt.Printf("%s\n", Reset)
				inThinking = false
			}
			fmt.Print(content)
			contentParts = append(contentParts, content)
		}

		// Collect native tool calls
		for _, tc := range tcList {
			if tc.Function.Name != "" {
				toolCalls = append(toolCalls, ParsedToolCall{
					Name: tc.Function.Name,
					Args: tc.Function.Arguments,
				})
			}
		}

		if obj.Done {
			break
		}
	}

	// Clean up spinner if model returned nothing
	if !gotFirstOutput {
		spinner.Stop()
		fmt.Printf("%s%syolo>%s ", Blue, Bold, Reset)
	}

	if inThinking {
		fmt.Print(Reset)
	}
	fmt.Println()

	contentText := strings.Join(contentParts, "")
	thinkingText := strings.Join(thinkingParts, "")
	displayText := contentText
	if displayText == "" {
		displayText = thinkingText
	}

	return &ChatResult{
		DisplayText: displayText,
		ContentText: contentText,
		ToolCalls:   toolCalls,
	}, nil
}

// ─── Tool Executor ───────────────────────────────────────────────────

var validTools = []string{
	"read_file", "write_file", "edit_file", "list_files",
	"search_files", "run_command", "spawn_subagent",
	"list_models", "switch_model", "think",
}

type ToolExecutor struct {
	baseDir string
	agent   *YoloAgent
}

func NewToolExecutor(baseDir string, agent *YoloAgent) *ToolExecutor {
	return &ToolExecutor{baseDir: baseDir, agent: agent}
}

func (t *ToolExecutor) safePath(path string) (string, error) {
	full := filepath.Clean(filepath.Join(t.baseDir, path))
	if !strings.HasPrefix(full, t.baseDir) {
		return "", fmt.Errorf("path '%s' is outside working directory", path)
	}
	return full, nil
}

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
	case "list_models":
		return t.listModels()
	case "switch_model":
		return t.switchModel(args)
	case "think":
		return "Thought recorded."
	default:
		return fmt.Sprintf("Error: unknown tool '%s'. Available tools: %s", name, strings.Join(validTools, ", "))
	}
}

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

	matches, err := filepath.Glob(filepath.Join(t.baseDir, pattern))
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var files, dirs []string
	for _, m := range matches {
		rel, _ := filepath.Rel(t.baseDir, m)
		// Skip hidden directories except .claude*
		if strings.HasPrefix(rel, ".yolo") || strings.HasPrefix(rel, ".git") || strings.HasPrefix(rel, "__pycache__") {
			if !strings.HasPrefix(rel, ".claude") {
				continue
			}
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
			if strings.Contains(path, ".yolo") || strings.Contains(path, ".git") {
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

func (t *ToolExecutor) spawnSubagent(args map[string]any) string {
	task := getStringArg(args, "task", "")
	model := getStringArg(args, "model", "")
	if t.agent != nil {
		return t.agent.spawnSubagent(task, model)
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

// ─── History Manager ──────────────────────────────────────────────────

type HistoryMessage struct {
	Role    string         `json:"role"`
	Content string         `json:"content"`
	TS      string         `json:"ts"`
	Meta    map[string]any `json:"meta,omitempty"`
}

type EvolutionEntry struct {
	TS     string `json:"ts"`
	Action string `json:"action"`
	Detail string `json:"detail"`
}

type HistoryConfig struct {
	Model   string `json:"model"`
	Created string `json:"created"`
}

type HistoryData struct {
	Version      int              `json:"version"`
	Config       HistoryConfig    `json:"config"`
	Messages     []HistoryMessage `json:"messages"`
	EvolutionLog []EvolutionEntry `json:"evolution_log"`
}

type HistoryManager struct {
	yoloDir     string
	historyFile string
	Data        HistoryData
	mu          sync.Mutex
}

func NewHistoryManager(yoloDir string) *HistoryManager {
	h := &HistoryManager{
		yoloDir:     yoloDir,
		historyFile: filepath.Join(yoloDir, "history.json"),
	}
	h.Data = h.empty()
	return h
}

func (h *HistoryManager) empty() HistoryData {
	return HistoryData{
		Version: 1,
		Config: HistoryConfig{
			Model:   "",
			Created: time.Now().Format(time.RFC3339),
		},
		Messages:     []HistoryMessage{},
		EvolutionLog: []EvolutionEntry{},
	}
}

func (h *HistoryManager) Load() bool {
	data, err := os.ReadFile(h.historyFile)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, &h.Data); err != nil {
		cprint(Yellow, "Warning: corrupt history, starting fresh")
		h.Data = h.empty()
		return false
	}
	return true
}

func (h *HistoryManager) Save() {
	h.mu.Lock()
	defer h.mu.Unlock()

	os.MkdirAll(h.yoloDir, 0o755)
	data, err := json.MarshalIndent(h.Data, "", "  ")
	if err != nil {
		return
	}
	tmp := h.historyFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	os.Rename(tmp, h.historyFile)
}

func (h *HistoryManager) AddMessage(role, content string, meta map[string]any) {
	h.mu.Lock()
	msg := HistoryMessage{
		Role:    role,
		Content: content,
		TS:      time.Now().Format(time.RFC3339),
		Meta:    meta,
	}
	h.Data.Messages = append(h.Data.Messages, msg)
	h.mu.Unlock()
	h.Save()
}

func (h *HistoryManager) AddEvolution(action, description string) {
	h.mu.Lock()
	h.Data.EvolutionLog = append(h.Data.EvolutionLog, EvolutionEntry{
		TS:     time.Now().Format(time.RFC3339),
		Action: action,
		Detail: description,
	})
	h.mu.Unlock()
	h.Save()
}

func (h *HistoryManager) GetContextMessages(maxMsgs int) []ChatMessage {
	msgs := h.Data.Messages
	start := 0
	if len(msgs) > maxMsgs {
		start = len(msgs) - maxMsgs
	}
	recent := msgs[start:]

	var out []ChatMessage
	for _, m := range recent {
		switch m.Role {
		case "user", "assistant":
			out = append(out, ChatMessage{Role: m.Role, Content: m.Content})
		case "tool":
			out = append(out, ChatMessage{Role: "user", Content: "[Tool result]\n" + m.Content})
		case "system":
			out = append(out, ChatMessage{Role: "user", Content: "[SYSTEM] " + m.Content})
		}
	}
	return out
}

func (h *HistoryManager) GetModel() string {
	return h.Data.Config.Model
}

func (h *HistoryManager) SetModel(model string) {
	h.Data.Config.Model = model
	h.Save()
}

// ─── Main Agent ───────────────────────────────────────────────────────

type YoloAgent struct {
	baseDir         string
	scriptPath      string
	ollama          *OllamaClient
	history         *HistoryManager
	tools           *ToolExecutor
	running         bool
	subagentCounter int
	lastActivity    time.Time
	thinkDelay      int
	mu              sync.Mutex
	cancelChat      context.CancelFunc // cancels the current Chat HTTP request
}

func NewYoloAgent() *YoloAgent {
	baseDir, _ := os.Getwd()
	execPath, _ := os.Executable()

	a := &YoloAgent{
		baseDir:      baseDir,
		scriptPath:   execPath,
		ollama:       NewOllamaClient(OllamaURL),
		history:      NewHistoryManager(YoloDir),
		running:      true,
		lastActivity: time.Now(),
		thinkDelay:   IdleThinkDelay,
	}
	a.tools = NewToolExecutor(baseDir, a)
	return a
}

func (a *YoloAgent) getSystemPrompt() string {
	return fmt.Sprintf(`You are YOLO (Your Own Living Operator), a self-evolving AI agent for software development.

Working directory: %s
Your own source code: %s
Current model: %s
Timestamp: %s

You CAN and SHOULD read and modify your own source code to improve yourself.

## Rules
1. All file paths are relative to: %s
2. Be careful with destructive operations.
3. Use tools to take concrete action — do not just talk about what you would do.
4. You may call multiple tools per turn.
5. Trust tool output — it is complete and accurate. Do NOT re-call the same tool with a different pattern.

## Autonomy
You are designed to work AUTONOMOUSLY. This is the core purpose of YOLO.
- Do NOT stop to ask the user for permission, confirmation, or input.
- Do NOT ask questions like 'Would you like me to...' or 'Shall I...'
- Just DO the work. Make decisions yourself. Act, don't ask.
- If something fails, try a different approach on your own.
- After completing one improvement, immediately move on to the next.
- Focus on: code quality, bug fixes, tests, self-improvement, documentation.
- Briefly state what you did and what you're doing next, then use tools.`,
		a.baseDir, a.scriptPath, a.history.GetModel(), time.Now().Format(time.RFC3339),
		a.baseDir)
}

// ── Setup ──

func (a *YoloAgent) setupFirstRun() {
	cprint(Cyan+Bold, "\n  YOLO - Your Own Living Operator")
	cprint(Gray, "  A self-evolving AI agent for software development\n")
	cprint(Gray, fmt.Sprintf("  Working directory: %s", a.baseDir))
	fmt.Println()

	cprint(Yellow, "  Connecting to Ollama...")
	models := a.ollama.ListModels()
	if len(models) == 0 {
		cprint(Red, "  Error: Cannot reach Ollama or no models installed.")
		cprint(Red, "  Make sure Ollama is running: ollama serve")
		os.Exit(1)
	}

	cprint(Green, fmt.Sprintf("  Found %d model(s):\n", len(models)))
	for i, m := range models {
		fmt.Printf("    %s%2d%s. %s\n", Bold, i+1, Reset, m)
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("  %sSelect model (1-%d): %s", Green, len(models), Reset)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		var idx int
		if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(models) {
			fmt.Println("  Invalid selection, try again.")
			continue
		}
		a.history.SetModel(models[idx-1])
		a.history.Save()
		cprint(Green, fmt.Sprintf("\n  Model: %s%s%s", Bold, models[idx-1], Reset))
		break
	}
	a.showHelpHint()
}

func (a *YoloAgent) resumeSession() {
	cprint(Cyan+Bold, "\n  YOLO - Your Own Living Operator")
	cprint(Green, fmt.Sprintf("  Resuming — model: %s%s%s", Bold, a.history.GetModel(), Reset))
	n := len(a.history.Data.Messages)
	cprint(Gray, fmt.Sprintf("  History: %d messages loaded", n))
	a.showHelpHint()
}

func (a *YoloAgent) showHelpHint() {
	cprint(Gray, "\n  Type a message, or /help for commands.")
	cprint(Gray, fmt.Sprintf("  Agent thinks autonomously after %ds of idle.\n", IdleThinkDelay))
}

// ── Chat loop ──

func (a *YoloAgent) chatWithAgent(userMessage string, autonomous bool) {
	if userMessage != "" {
		a.history.AddMessage("user", userMessage, nil)
	}

	if autonomous {
		a.history.AddMessage("system",
			"No new user input. You are in autonomous mode. "+
				"Continue making progress on your own — do NOT ask the user "+
				"for input or confirmation. Pick the most impactful next task "+
				"and execute it using tools. Focus on: code quality, bug fixes, "+
				"tests, self-improvement, or new features. "+
				"Act decisively. Do the work, then move to the next thing.", nil)
	}

	// Base context from persistent history
	baseMsgs := []ChatMessage{
		{Role: "system", Content: a.getSystemPrompt()},
	}
	baseMsgs = append(baseMsgs, a.history.GetContextMessages(MaxContextMessages)...)

	// In-memory messages for the current tool-calling chain
	var roundMsgs []ChatMessage
	type toolLogEntry struct {
		name   string
		args   map[string]any
		result string
	}
	var toolLog []toolLogEntry
	var finalText string

	roundNum := 0
	for {
		allMsgs := append(baseMsgs, roundMsgs...)

		ctx, cancel := context.WithCancel(context.Background())
		a.mu.Lock()
		a.cancelChat = cancel
		a.mu.Unlock()

		result, err := a.ollama.Chat(ctx, a.history.GetModel(), allMsgs, ollamaTools)
		cancel()
		a.mu.Lock()
		a.cancelChat = nil
		a.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				cprint(Yellow, "\n  Interrupted.")
				return
			}
			cprint(Red, fmt.Sprintf("\nError: %v", err))
			return
		}

		toolCalls := result.ToolCalls

		// Also check for text-based tool calls as fallback
		if len(toolCalls) == 0 {
			toolCalls = a.parseTextToolCalls(result.DisplayText)
		}

		if len(toolCalls) == 0 {
			finalText = result.DisplayText
			break
		}

		// Build proper assistant message with tool_calls
		var nativeTCs []ToolCall
		for i, tc := range toolCalls {
			argsJSON, _ := json.Marshal(tc.Args)
			nativeTCs = append(nativeTCs, ToolCall{
				ID: fmt.Sprintf("call_%d_%d", roundNum, i),
				Function: ToolCallFunc{
					Name:      tc.Name,
					Arguments: json.RawMessage(argsJSON),
				},
			})
		}
		roundMsgs = append(roundMsgs, ChatMessage{
			Role:      "assistant",
			Content:   result.ContentText,
			ToolCalls: nativeTCs,
		})

		// Execute each tool and add tool-role result
		for _, call := range toolCalls {
			name := call.Name
			args := call.Args
			if args == nil {
				args = map[string]any{}
			}

			shortArgs, _ := json.Marshal(args)
			shortStr := string(shortArgs)
			if len(shortStr) > 80 {
				shortStr = shortStr[:80] + "..."
			}
			cprint(Yellow, fmt.Sprintf("  [%s] %s", name, shortStr))

			resultStr := a.tools.Execute(name, args)

			preview := resultStr
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", " ")
			cprint(Gray, fmt.Sprintf("  => %s", preview))

			roundMsgs = append(roundMsgs, ChatMessage{Role: "tool", Content: resultStr})
			toolLog = append(toolLog, toolLogEntry{name: name, args: args, result: resultStr})
		}

		// Optionally nudge the model to wrap up after many rounds
		if ToolNudgeAfter > 0 && roundNum >= ToolNudgeAfter {
			roundMsgs = append(roundMsgs, ChatMessage{
				Role: "user",
				Content: "[SYSTEM] You have used many tool rounds. " +
					"Please respond to the user with what you have so far. " +
					"You can always use more tools in the next interaction.",
			})
		}

		roundNum++
	}

	// Save to persistent history
	if len(toolLog) > 0 {
		var summaryLines []string
		for _, entry := range toolLog {
			shortResult := entry.result
			if len(shortResult) > 150 {
				shortResult = shortResult[:150]
			}
			shortResult = strings.ReplaceAll(shortResult, "\n", " ")
			summaryLines = append(summaryLines, fmt.Sprintf("[%s] => %s", entry.name, shortResult))
		}
		a.history.AddMessage("assistant", "[tool activity]\n"+strings.Join(summaryLines, "\n"), nil)
	}
	if finalText != "" {
		a.history.AddMessage("assistant", finalText, nil)
	}
}

func (a *YoloAgent) parseTextToolCalls(text string) []ParsedToolCall {
	var calls []ParsedToolCall

	// Format 1: <tool_call>{"name": ..., "args": ...}</tool_call>
	re1 := regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)
	for _, match := range re1.FindAllStringSubmatch(text, -1) {
		var obj map[string]any
		if err := json.Unmarshal([]byte(match[1]), &obj); err == nil {
			if name, ok := obj["name"].(string); ok {
				args, _ := obj["args"].(map[string]any)
				if args == nil {
					args = map[string]any{}
				}
				calls = append(calls, ParsedToolCall{Name: name, Args: args})
			}
		}
	}

	// Format 2: <tool_call><function=name><parameter=key>value</parameter>...</function></tool_call>
	if len(calls) == 0 {
		re2 := regexp.MustCompile(`(?s)<tool_call>\s*<function=(\w+)>(.*?)</function>\s*</tool_call>`)
		reParam := regexp.MustCompile(`(?s)<parameter=(\w+)>\s*(.*?)\s*</parameter>`)
		for _, match := range re2.FindAllStringSubmatch(text, -1) {
			name := match[1]
			body := match[2]
			args := map[string]any{}
			for _, pm := range reParam.FindAllStringSubmatch(body, -1) {
				args[pm[1]] = pm[2]
			}
			calls = append(calls, ParsedToolCall{Name: name, Args: args})
		}
	}

	// Format 3: [tool_name] {"key": "value", ...}
	if len(calls) == 0 {
		re3 := regexp.MustCompile(`(?m)^\s*\[(\w+)\]\s*(\{.*?\})\s*$`)
		validToolSet := map[string]bool{}
		for _, t := range validTools {
			validToolSet[t] = true
		}
		for _, match := range re3.FindAllStringSubmatch(text, -1) {
			name := match[1]
			if validToolSet[name] {
				var args map[string]any
				if err := json.Unmarshal([]byte(match[2]), &args); err == nil {
					calls = append(calls, ParsedToolCall{Name: name, Args: args})
				}
			}
		}
	}

	// Format 4: <tool_name>{"key": "value"}</tool_name> or <tool_name><key>value</key></tool_name>
	if len(calls) == 0 {
		for _, toolName := range validTools {
			re4 := regexp.MustCompile(fmt.Sprintf(`(?s)<%s>(.*?)</%s>`, regexp.QuoteMeta(toolName), regexp.QuoteMeta(toolName)))
			for _, match := range re4.FindAllStringSubmatch(text, -1) {
				body := strings.TrimSpace(match[1])
				var args map[string]any
				if err := json.Unmarshal([]byte(body), &args); err != nil {
					args = map[string]any{}
					// Parse XML-style <key>value</key> params
					reParam := regexp.MustCompile(`<(\w+)>(.*?)</\w+>`)
					for _, pm := range reParam.FindAllStringSubmatch(body, -1) {
						args[pm[1]] = pm[2]
					}
				}
				if len(args) > 0 {
					calls = append(calls, ParsedToolCall{Name: toolName, Args: args})
				}
			}
		}
	}

	return calls
}

// ── Model switching ──

func (a *YoloAgent) switchModel(model string) string {
	models := a.ollama.ListModels()
	found := false
	for _, m := range models {
		if m == model {
			found = true
			break
		}
	}
	if !found {
		return fmt.Sprintf("Model '%s' not found. Available: %s", model, strings.Join(models, ", "))
	}
	old := a.history.GetModel()
	a.history.SetModel(model)
	a.history.AddEvolution("model_switch", fmt.Sprintf("%s -> %s", old, model))
	cprint(Cyan, fmt.Sprintf("  Switched model: %s -> %s", old, model))
	return fmt.Sprintf("Switched from %s to %s", old, model)
}

// ── Sub-agents ──

func (a *YoloAgent) spawnSubagent(task, model string) string {
	a.mu.Lock()
	a.subagentCounter++
	aid := a.subagentCounter
	a.mu.Unlock()

	useModel := model
	if useModel == "" {
		useModel = a.history.GetModel()
	}
	os.MkdirAll(SubagentDir, 0o755)
	resultFile := filepath.Join(SubagentDir, fmt.Sprintf("agent_%d.json", aid))

	go func() {
		cprint(Magenta, fmt.Sprintf("  [sub-agent #%d] started (%s)", aid, useModel))
		msgs := []ChatMessage{
			{
				Role: "system",
				Content: fmt.Sprintf("You are a sub-agent. Complete this task concisely:\n\n%s\n\nWorking directory: %s",
					task, a.baseDir),
			},
		}

		status := "complete"
		result := "done"
		_, err := a.ollama.Chat(context.Background(), useModel, msgs, nil)
		if err != nil {
			result = err.Error()
			status = "error"
		}

		data, _ := json.MarshalIndent(map[string]any{
			"id":     aid,
			"task":   task,
			"model":  useModel,
			"status": status,
			"result": result,
			"ts":     time.Now().Format(time.RFC3339),
		}, "", "  ")
		os.WriteFile(resultFile, data, 0o644)
		cprint(Magenta, fmt.Sprintf("\n  [sub-agent #%d] %s. See %s", aid, status, resultFile))
	}()

	return fmt.Sprintf("Sub-agent #%d spawned (%s). Results -> %s", aid, useModel, resultFile)
}

// ── Slash commands ──

func (a *YoloAgent) handleCommand(cmd string) {
	parts := strings.SplitN(cmd, " ", 2)
	command := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch command {
	case "/help", "/h":
		fmt.Printf(`
%sCommands:%s
  /help            Show this help
  /model           Current model
  /models          List available models
  /switch <name>   Switch model
  /history         Message count
  /clear           Clear conversation history
  /status          Agent status
  /exit, /quit     Exit YOLO
`, Cyan, Reset)

	case "/model":
		cprint(Cyan, fmt.Sprintf("  Model: %s", a.history.GetModel()))

	case "/models":
		models := a.ollama.ListModels()
		cprint(Cyan, "  Available models:")
		for _, m := range models {
			marker := ""
			if m == a.history.GetModel() {
				marker = fmt.Sprintf(" %s<- current%s", Green, Reset)
			}
			fmt.Printf("    %s%s\n", m, marker)
		}

	case "/switch":
		if arg != "" {
			a.switchModel(arg)
		} else {
			cprint(Red, "  Usage: /switch <model-name>")
		}

	case "/history":
		n := len(a.history.Data.Messages)
		e := len(a.history.Data.EvolutionLog)
		cprint(Cyan, fmt.Sprintf("  Messages: %d  |  Evolution events: %d", n, e))

	case "/clear":
		a.history.Data.Messages = []HistoryMessage{}
		a.history.Save()
		cprint(Cyan, "  History cleared (config preserved)")

	case "/status":
		fmt.Printf(`
%sStatus:%s
  Model:       %s
  Messages:    %d
  Evolutions:  %d
  Working dir: %s
  Script:      %s
  Idle delay:  %ds
  Think delay: %ds
`, Cyan, Reset,
			a.history.GetModel(),
			len(a.history.Data.Messages),
			len(a.history.Data.EvolutionLog),
			a.baseDir,
			a.scriptPath,
			IdleThinkDelay,
			ThinkLoopDelay)

	case "/exit", "/quit":
		a.running = false

	default:
		cprint(Red, fmt.Sprintf("  Unknown command: %s  (try /help)", command))
	}
}

// ── Input handling (raw mode) ──

const sentinelThink = "__THINK__"

func (a *YoloAgent) readLine() (string, bool) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback to simple line reading
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", false
		}
		return strings.TrimRight(line, "\r\n"), true
	}
	defer term.Restore(fd, oldState)

	var buf []byte

	// Single persistent reader goroutine to avoid leaks
	readCh := make(chan byte, 16)
	errCh := make(chan error, 1)
	go func() {
		b := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(b)
			if err != nil {
				errCh <- err
				return
			}
			if n > 0 {
				readCh <- b[0]
			}
		}
	}()

	for a.running {
		select {
		case ch := <-readCh:
			a.lastActivity = time.Now()
			a.thinkDelay = IdleThinkDelay

			switch {
			case ch == '\r' || ch == '\n': // Enter
				fmt.Print("\r\n")
				return string(buf), true
			case ch == 127 || ch == 8: // Backspace
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
					fmt.Print("\b \b")
				}
			case ch == 3: // Ctrl-C
				fmt.Print("\r\n")
				return "", false
			case ch == 4: // Ctrl-D
				if len(buf) == 0 {
					return "", false
				}
			case ch == 21: // Ctrl-U (kill line)
				for range buf {
					fmt.Print("\b \b")
				}
				buf = buf[:0]
			case ch == 23: // Ctrl-W (kill word)
				for len(buf) > 0 && buf[len(buf)-1] == ' ' {
					buf = buf[:len(buf)-1]
					fmt.Print("\b \b")
				}
				for len(buf) > 0 && buf[len(buf)-1] != ' ' {
					buf = buf[:len(buf)-1]
					fmt.Print("\b \b")
				}
			case ch == 27: // Escape sequence (arrows etc.)
				// Consume remaining bytes of escape sequence
				for i := 0; i < 2; i++ {
					select {
					case <-readCh:
					case <-time.After(50 * time.Millisecond):
					}
				}
			default:
				if ch >= 32 && unicode.IsPrint(rune(ch)) {
					buf = append(buf, ch)
					fmt.Printf("%c", ch)
				}
			}

		case <-errCh:
			return "", false

		case <-time.After(1 * time.Second):
			// Timeout — check for autonomous thinking
			if len(buf) == 0 {
				elapsed := time.Since(a.lastActivity)
				if elapsed >= time.Duration(a.thinkDelay)*time.Second {
					return sentinelThink, true
				}
			}
		}
	}
	return "", false
}

// ── Main loop ──

func (a *YoloAgent) Run() {
	hasHistory := a.history.Load()

	if hasHistory && a.history.GetModel() != "" {
		a.resumeSession()
	} else {
		a.setupFirstRun()
	}

	a.lastActivity = time.Now()
	a.thinkDelay = IdleThinkDelay

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		for range sigCh {
			a.mu.Lock()
			cancel := a.cancelChat
			a.mu.Unlock()
			if cancel != nil {
				cancel()
			} else {
				a.running = false
				fmt.Println()
				cprint(Cyan, "\n  Interrupted — saving session...")
			}
		}
	}()

	for a.running {
		cprintNoNL(Green+Bold, "you> ")

		line, ok := a.readLine()
		if !ok {
			a.running = false
			break
		}

		if line == sentinelThink {
			// Clear the prompt line and think
			fmt.Printf("\r%s\r", strings.Repeat(" ", 40))
			cprint(Gray, "  [autonomous thinking...]")
			a.chatWithAgent("", true)
			a.lastActivity = time.Now()
			a.thinkDelay = ThinkLoopDelay
		} else {
			stripped := strings.TrimSpace(line)
			lower := strings.ToLower(stripped)
			if lower == "exit" || lower == "quit" {
				a.running = false
			} else if strings.HasPrefix(stripped, "/") {
				a.handleCommand(stripped)
			} else if stripped != "" {
				a.chatWithAgent(stripped, false)
			}
		}
	}

	a.history.Save()
	cprint(Cyan, "  Session saved. Goodbye!\n")
}

// ─── Entry Point ──────────────────────────────────────────────────────

func main() {
	agent := NewYoloAgent()
	agent.Run()
}
