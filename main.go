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
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"golang.org/x/term"
)

// ─── Configuration ────────────────────────────────────────────────────

const (
	YoloDir            = ".yolo"
	IdleThinkDelay     = 30  // seconds of no input before autonomous thinking
	ThinkLoopDelay     = 120 // seconds between autonomous think cycles
	MaxContextMessages = 40
	MaxToolOutput      = 0  // 0 = no truncation
	ToolNudgeAfter     = 0  // 0 = disabled
	CommandTimeout     = 30 // shell command timeout in seconds
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

// rawWrite writes text to stdout, converting lone \n to \r\n for raw terminal mode.
// In raw mode, OPOST is disabled so \n only moves the cursor down without returning
// to column 1. This function ensures proper carriage return + line feed behavior.
func rawWrite(s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	fmt.Print(s)
}

func cprint(color, text string) {
	if globalUI != nil {
		globalUI.OutputPrint(fmt.Sprintf("%s%s%s\n", color, text, Reset))
	} else {
		rawWrite(fmt.Sprintf("%s%s%s\n", color, text, Reset))
	}
}

func cprintNoNL(color, text string) {
	if globalUI != nil {
		globalUI.OutputPrint(fmt.Sprintf("%s%s%s", color, text, Reset))
	} else {
		rawWrite(fmt.Sprintf("%s%s%s", color, text, Reset))
	}
}

// ─── Terminal UI (split output/input regions) ────────────────────────

// globalUI is set once the split UI is active. Before that, output goes to stdout directly.
var globalUI *TerminalUI

// stripAnsiCodes removes ANSI escape sequences from text for cursor tracking purposes.
// This ensures color codes don't mess up column/row calculations.
func stripAnsiCodes(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

// TerminalUI manages a split terminal: a scrolling output region on top and
// a fixed input line at the bottom, separated by a divider.
type TerminalUI struct {
	mu       sync.Mutex
	fd       int
	rows     int
	cols     int
	inputBuf []byte // mirrors InputManager's buffer for redraw
	prompt   string
	outRow   int // tracked row of cursor in output region
	outCol   int // tracked col of cursor in output region
}

// wrapText wraps text to the given width, inserting newlines at word boundaries.
// It preserves existing newlines exactly (no adding or doubling) and handles
// words longer than the terminal width by breaking them.
func (ui *TerminalUI) wrapText(text string) string {
	if ui.cols <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if len(line) == 0 {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		var current strings.Builder
		currentLen := 0

		for _, word := range words {
			wordLen := len(word)

			if currentLen > 0 && currentLen+1+wordLen > ui.cols {
				// Current line is full, start a new one
				wrappedLines = append(wrappedLines, current.String())
				current.Reset()
				currentLen = 0
			}

			if currentLen > 0 {
				current.WriteString(" ")
				currentLen++
			}

			// Break words longer than terminal width
			for len(word) > ui.cols && currentLen == 0 {
				wrappedLines = append(wrappedLines, word[:ui.cols])
				word = word[ui.cols:]
			}

			current.WriteString(word)
			currentLen += len(word)
		}

		if current.Len() > 0 {
			wrappedLines = append(wrappedLines, current.String())
		}
	}

	return strings.Join(wrappedLines, "\n")
}

func NewTerminalUI() *TerminalUI {
	fd := int(os.Stdout.Fd())
	cols, rows, err := term.GetSize(fd)
	if err != nil {
		rows = 24
		cols = 80
	}
	return &TerminalUI{
		fd:     fd,
		rows:   rows,
		cols:   cols,
		outRow: 1,
		outCol: 1,
	}
}

// Setup initializes the scroll region and draws the divider.
func (ui *TerminalUI) Setup() {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Refresh terminal size
	if cols, rows, err := term.GetSize(ui.fd); err == nil {
		ui.rows = rows
		ui.cols = cols
	}

	// Clear screen, set scroll region, draw divider
	fmt.Print("\033[2J")
	fmt.Printf("\033[1;%dr", ui.rows-2)
	ui.drawDividerLocked()
	ui.outRow = 1
	ui.outCol = 1
	fmt.Printf("\033[%d;%dH", ui.outRow, ui.outCol)
}

func (ui *TerminalUI) drawDividerLocked() {
	divider := strings.Repeat("─", ui.cols)
	fmt.Printf("\033[%d;1H%s%s%s", ui.rows-1, Gray, divider, Reset)
}

// Teardown restores the full scroll region.
func (ui *TerminalUI) Teardown() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("\033[1;%dr", ui.rows)
	fmt.Printf("\033[%d;1H\n", ui.rows)
}

func (ui *TerminalUI) trackCursorMovement(stripped string) {
	for _, ch := range stripped {
		switch ch {
		case '\n':
			ui.outRow++
			ui.outCol = 1
			if ui.outRow > ui.rows-2 {
				ui.outRow = ui.rows - 2 // scroll region keeps cursor at bottom
			}
		case '\r':
			ui.outCol = 1
		default:
			ui.outCol++
			if ui.outCol > ui.cols {
				ui.outCol = 1
				ui.outRow++
				if ui.outRow > ui.rows-2 {
					ui.outRow = ui.rows - 2
				}
			}
		}
	}
}

// OutputPrint writes text to the output (scrolling) region.
func (ui *TerminalUI) OutputPrint(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Wrap text to terminal width before outputting
	text = ui.wrapText(text)

	// Move cursor to tracked output position within the scroll region
	fmt.Printf("\033[%d;%dH", ui.outRow, ui.outCol)
	// Write the text (rawWrite converts \n to \r\n for raw terminal mode)
	rawWrite(text)
	// Track where the cursor ended up.
	// Strip ANSI codes before counting to avoid off-by-several errors from color codes.
	stripped := stripAnsiCodes(text)
	ui.trackCursorMovement(stripped)
	// Redraw input line (output may have scrolled and clobbered it)
	ui.drawInputLocked()
}

// OutputPrintInline writes text without moving cursor back to input line.
// Used for streaming tokens within a single Chat response.
// Call OutputFinishLine when done with a block of inline output.
func (ui *TerminalUI) OutputPrintInline(text string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	// Do NOT wrap streaming tokens — they are fragments, not complete lines.
	// The terminal handles character-level wrapping, and trackCursorMovement
	// already accounts for it.

	fmt.Printf("\033[%d;%dH", ui.outRow, ui.outCol)
	// rawWrite converts \n to \r\n for raw terminal mode
	rawWrite(text)
	// Strip ANSI codes before counting to avoid off-by-several errors from color codes.
	stripped := stripAnsiCodes(text)
	ui.trackCursorMovement(stripped)
}

// OutputFinishLine redraws the input line after inline output is done.
func (ui *TerminalUI) OutputFinishLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.drawInputLocked()
}

func (ui *TerminalUI) drawInputLocked() {
	// Move to input row and clear it
	fmt.Printf("\033[%d;1H\033[2K", ui.rows) // Clear entire line

	promptStr := ui.prompt
	inputStr := string(ui.inputBuf)

	// Calculate available space after prompt
	promptWidth := len(stripAnsiCodes(promptStr))
	availableWidth := ui.cols - promptWidth - 1 // -1 for cursor space

	if availableWidth <= 0 {
		availableWidth = ui.cols - 2
	}

	// If input fits on one line, show it all (current behavior)
	var displayInput string
	cursorCol := promptWidth + len(inputStr) + 1
	if len(inputStr) <= availableWidth {
		displayInput = inputStr
	} else {
		// Show rightmost portion that fits (horizontal scrolling)
		startPos := len(inputStr) - availableWidth
		displayInput = inputStr[startPos:]
		cursorCol = promptWidth + len(displayInput) + 1
	}

	// Draw prompt and input, then position cursor
	fmt.Printf("%s%s\033[%d;%dH", promptStr, displayInput, ui.rows, cursorCol)
}

// UpdateInput updates the UI's copy of the input state for redrawing.
func (ui *TerminalUI) UpdateInput(prompt string, buf []byte) {
	ui.mu.Lock()
	ui.prompt = prompt
	ui.inputBuf = make([]byte, len(buf))
	copy(ui.inputBuf, buf)
	ui.mu.Unlock()
}

// RedrawInput redraws just the input line.
func (ui *TerminalUI) RedrawInput() {
	ui.mu.Lock()
	ui.drawInputLocked()
	ui.mu.Unlock()
}

// WriteToInputLine writes directly to the input line area (for character echo).
func (ui *TerminalUI) WriteToInputLine(s string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	// Just output the character - next redraw will handle positioning
	fmt.Print(s)
}

// ClearInputLine clears the input line.
func (ui *TerminalUI) ClearInputLine() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	fmt.Printf("\033[%d;1H\033[K", ui.rows)
}

// RefreshSize updates terminal size and redraws layout.
func (ui *TerminalUI) RefreshSize() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if cols, rows, err := term.GetSize(ui.fd); err == nil && (rows != ui.rows || cols != ui.cols) {
		ui.rows = rows
		ui.cols = cols
		fmt.Printf("\033[1;%dr", ui.rows-2)
		ui.drawDividerLocked()
		ui.drawInputLocked()
	}
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
				text := fmt.Sprintf("\r%s%s%s thinking...%s", s.color, s.prefix, frame, Reset)
				if globalUI != nil {
					globalUI.OutputPrintInline(text)
				} else {
					rawWrite(text)
				}
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
	text := fmt.Sprintf("\r%s\r", strings.Repeat(" ", clearLen))
	if globalUI != nil {
		globalUI.OutputPrintInline(text)
	} else {
		rawWrite(text)
	}
}

// ─── Ollama Tool Definitions ─────────────────────────────────────────

type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolSchema struct {
	Type       string               `json:"type"`
	Properties map[string]ToolParam `json:"properties"`
	Required   []string             `json:"required,omitempty"`
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
	toolDef("restart", "Rebuild and restart the program", map[string]ToolParam{}, nil),
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
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
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
	Model    string         `json:"model"`
	Messages []ChatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Tools    []ToolDef      `json:"tools,omitempty"`
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

	// outPrint writes to the output region (inline, no input redraw per token)
	outPrint := func(s string) {
		if globalUI != nil {
			globalUI.OutputPrintInline(s)
		} else {
			rawWrite(s)
		}
	}

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
			outPrint(fmt.Sprintf("%s%syolo>%s ", Blue, Bold, Reset))
		}

		// Handle thinking tokens
		if thinking != "" {
			if !inThinking {
				outPrint(fmt.Sprintf("%s[thinking] ", Gray))
				inThinking = true
			}
			outPrint(thinking)
			thinkingParts = append(thinkingParts, thinking)
		}

		// Handle content tokens
		if content != "" {
			if inThinking {
				outPrint(fmt.Sprintf("%s\n", Reset))
				inThinking = false
			}
			outPrint(content)
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
		outPrint(fmt.Sprintf("%s%syolo>%s ", Blue, Bold, Reset))
	}

	if inThinking {
		outPrint(Reset)
	}
	outPrint("\n")
	// Redraw input line after streaming output is done
	if globalUI != nil {
		globalUI.OutputFinishLine()
	}

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
	"list_models", "switch_model", "think", "restart",
}

type ToolExecutor struct {
	baseDir string
	agent   *YoloAgent
}

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
	case "restart":
		return t.restart(args)
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

// globRecursive handles recursive glob patterns with **/ wildcards
func (t *ToolExecutor) globRecursive(pattern string) ([]string, error) {
	var matches []string

	// Handle patterns like **/*.txt or **/directory/*
	if strings.HasPrefix(pattern, "**/") {
		basePattern := pattern[3:]

		err := filepath.Walk(t.baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
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

func (t *ToolExecutor) spawnSubagent(args map[string]any) string {
	// Validate required parameters
	if args["prompt"] == nil {
		return "Error: required parameter 'prompt' is missing"
	}
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return "Error: 'prompt' cannot be empty"
	}

	name := getStringArg(args, "name", "subagent")
	description := getStringArg(args, "description", "")
	model := getStringArg(args, "model", "")

	// Format the output message
	result := fmt.Sprintf("Starting new agent '%s' with task: %s", name, prompt)
	if description != "" {
		result += fmt.Sprintf("\nDescription: %s", description)
	}
	if model != "" {
		result += fmt.Sprintf("\nModel: %s", model)
	}

	// Actually spawn the subagent using the agent if available
	if t.agent != nil {
		subResult := t.agent.spawnSubagent(prompt, model)
		return result + "\n" + subResult
	}

	return result + "\nError: no agent context"
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

	// Build command - build to the same executable name
	buildCmd := exec.Command("go", "build", "-o", filepath.Base(exePath), cwd+"/main.go")
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

// ─── History Manager ──────────────────────────────────────────────────

type HistoryMessage struct {
	Role    string         `json:"role"`
	Content string         `json:"content"`
	TS      string         `json:"ts"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// ── Backward Compatibility ────────────────

// MessageHistory is an alias for HistoryManager (for test compatibility)
type MessageHistory struct {
	SessionID        string
	CurrentAssistant *MessageHistoryItem
	CurrentUser      *MessageHistoryItem
	Messages         []HistoryMessage
}

type MessageHistoryItem struct {
	Type    string
	Value   string
	Message string // Added for test compatibility
}

// NewMessageHistory creates a new history with initial system message
func NewMessageHistory(sessionID string) *MessageHistory {
	return &MessageHistory{
		SessionID: sessionID,
		Messages: []HistoryMessage{
			{Role: "system", Content: "You are a helpful assistant.", TS: time.Now().Format(time.RFC3339)},
		},
	}
}

func (h *MessageHistory) AddUserMessage(content string) {
	h.Messages = append(h.Messages, HistoryMessage{
		Role:    "user",
		Content: content,
		TS:      time.Now().Format(time.RFC3339),
	})
}

func (h *MessageHistory) AddAssistantMessage(content string) {
	h.Messages = append(h.Messages, HistoryMessage{
		Role:    "assistant",
		Content: content,
		TS:      time.Now().Format(time.RFC3339),
	})
}

func (h *MessageHistory) StartToolCall(name string, args map[string]any) {
	argsJSON, _ := json.Marshal(args)
	newMsg := fmt.Sprintf("%s(%s)", name, string(argsJSON))

	if h.CurrentAssistant != nil && h.CurrentAssistant.Message != "" {
		h.CurrentAssistant.Value = name
		h.CurrentAssistant.Message += " → " + newMsg
	} else {
		h.CurrentAssistant = &MessageHistoryItem{Type: "tool_call", Value: name, Message: newMsg}
	}
}

func (h *MessageHistory) EndToolCall(result string) {
	h.Messages = append(h.Messages, HistoryMessage{
		Role:    "tool",
		Content: result,
		TS:      time.Now().Format(time.RFC3339),
	})
}

func (h *MessageHistory) Save() string {
	// Create temp file for this session's history
	filename := filepath.Join(os.TempDir(), "yolo_history_"+h.SessionID+"_"+strconv.FormatInt(time.Now().UnixNano(), 10)+".json")

	data, err := json.MarshalIndent(h.Messages, "", "  ")
	if err != nil {
		return ""
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return ""
	}

	return filename
}

// LoadMessageHistory loads a saved history by session ID
func LoadMessageHistory(sessionID string, clearOnFailure bool) (*MessageHistory, error) {
	// Find and load the most recent save file for this session
	pattern := filepath.Join(os.TempDir(), "yolo_history_"+sessionID+"*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		if clearOnFailure {
			return &MessageHistory{SessionID: sessionID}, nil
		}
		return nil, os.ErrNotExist
	}

	// Sort to get the most recent file
	sort.Strings(matches)
	filename := matches[len(matches)-1]

	data, err := os.ReadFile(filename)
	if err != nil {
		if clearOnFailure {
			return &MessageHistory{SessionID: sessionID}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var messages []HistoryMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	h := &MessageHistory{SessionID: sessionID}
	h.Messages = messages
	return h, nil
}

// ── Color constants for tests ────────

type Color int

const (
	ColorNone Color = iota
	ColorBold
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorGray
	BGRed
	BGGreen
)

// escapeMarkdown escapes markdown characters for display
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// ────────────────────────────────────────

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
	inputMgr        *InputManager
	running         bool
	busy            bool // true while agent is processing
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
	cprint(Gray, "  Type a message, or /help for commands.")
	cprint(Gray, fmt.Sprintf("  Agent thinks autonomously after %ds of idle.\n", IdleThinkDelay))
}

// ── Chat loop ──

func (a *YoloAgent) chatWithAgent(userMessage string, autonomous bool) {
	// Clear the user's input line so agent output appears cleanly
	if a.inputMgr != nil {
		a.inputMgr.ClearLine()
	}

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
			cprint(Yellow, fmt.Sprintf("  [%s] %s\n", name, shortStr))

			resultStr := a.tools.Execute(name, args)

			preview := resultStr
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", " ")
			cprint(Gray, fmt.Sprintf("  => %s\n", preview))

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
		// Echo AI's response with blue color and prefix
		cprint(Blue, fmt.Sprintf("  [%s] %s\n", "ai", finalText))
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
		cprint(Cyan, "Commands:")
		cprint(Reset, "  /help            Show this help")
		cprint(Reset, "  /model           Current model")
		cprint(Reset, "  /models          List available models")
		cprint(Reset, "  /switch <name>   Switch model")
		cprint(Reset, "  /history         Message count")
		cprint(Reset, "  /clear           Clear conversation history")
		cprint(Reset, "  /status          Agent status")
		cprint(Reset, "  /exit, /quit     Exit YOLO")

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
			cprint(Reset, fmt.Sprintf("    %s%s", m, marker))
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
		cprint(Cyan, "Status:")
		cprint(Reset, fmt.Sprintf("  Model:       %s", a.history.GetModel()))
		cprint(Reset, fmt.Sprintf("  Messages:    %d", len(a.history.Data.Messages)))
		cprint(Reset, fmt.Sprintf("  Evolutions:  %d", len(a.history.Data.EvolutionLog)))
		cprint(Reset, fmt.Sprintf("  Working dir: %s", a.baseDir))
		cprint(Reset, fmt.Sprintf("  Script:      %s", a.scriptPath))
		cprint(Reset, fmt.Sprintf("  Idle delay:  %ds", IdleThinkDelay))
		cprint(Reset, fmt.Sprintf("  Think delay: %ds", ThinkLoopDelay))

	case "/exit", "/quit":
		a.running = false

	default:
		cprint(Red, fmt.Sprintf("  Unknown command: %s  (try /help)", command))
	}
}

// ── Input Manager (async input) ──

// InputLine represents a completed line from the user.
type InputLine struct {
	Text string
	OK   bool // false means EOF/Ctrl-C
}

// InputManager reads from stdin continuously, allowing the user to type
// even while the agent is processing. Completed lines are sent to Lines.
type InputManager struct {
	Lines    chan InputLine
	rawBytes chan byte  // raw bytes from stdin reader goroutine
	rawErr   chan error // errors from stdin reader goroutine
	buf      []byte     // current line being edited
	mu       sync.Mutex // protects buf and prompt state
	prompt   string     // current prompt prefix being displayed
	agent    *YoloAgent
	oldState *term.State
	fd       int
}

func NewInputManager(agent *YoloAgent) *InputManager {
	im := &InputManager{
		Lines:    make(chan InputLine, 8),
		rawBytes: make(chan byte, 64),
		rawErr:   make(chan error, 1),
		agent:    agent,
		fd:       int(os.Stdin.Fd()),
	}
	return im
}

// Start begins reading stdin in raw mode. Call once.
func (im *InputManager) Start() {
	oldState, err := term.MakeRaw(im.fd)
	if err != nil {
		// Fallback: line-buffered mode
		go func() {
			reader := bufio.NewReader(os.Stdin)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					im.Lines <- InputLine{OK: false}
					return
				}
				im.Lines <- InputLine{Text: strings.TrimRight(line, "\r\n"), OK: true}
			}
		}()
		return
	}
	im.oldState = oldState

	// Raw byte reader goroutine
	go func() {
		b := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(b)
			if err != nil {
				im.rawErr <- err
				return
			}
			if n > 0 {
				im.rawBytes <- b[0]
			}
		}
	}()

	// Character processing goroutine
	go im.processLoop()
}

// Stop restores the terminal.
func (im *InputManager) Stop() {
	if im.oldState != nil {
		term.Restore(im.fd, im.oldState)
	}
}

// ShowPrompt displays a prompt and enables editing. Call from the main goroutine
// or when ready for input. The prompt is redisplayed after agent output.
func (im *InputManager) ShowPrompt(prompt string) {
	im.mu.Lock()
	im.prompt = prompt
	im.mu.Unlock()
	im.syncAndRedraw()
}

// syncAndRedraw updates the TerminalUI's copy and redraws the input line.
func (im *InputManager) syncAndRedraw() {
	im.mu.Lock()
	prompt := im.prompt
	buf := make([]byte, len(im.buf))
	copy(buf, im.buf)
	im.mu.Unlock()

	if globalUI != nil {
		globalUI.UpdateInput(prompt, buf)
		globalUI.RedrawInput()
	} else {
		fmt.Printf("\r\033[K%s%s", prompt, string(buf))
	}
}

// ClearLine clears the input line.
func (im *InputManager) ClearLine() string {
	im.mu.Lock()
	text := string(im.buf)
	im.mu.Unlock()
	if globalUI != nil {
		globalUI.ClearInputLine()
	} else {
		fmt.Printf("\r\033[K")
	}
	return text
}

// RedrawAfterOutput redraws the prompt and current buffer after agent output.
func (im *InputManager) RedrawAfterOutput() {
	im.syncAndRedraw()
}

func (im *InputManager) processLoop() {
	for {
		select {
		case ch := <-im.rawBytes:
			im.agent.lastActivity = time.Now()
			im.agent.thinkDelay = IdleThinkDelay

			im.mu.Lock()
			switch {
			case ch == '\r' || ch == '\n': // Enter
				line := string(im.buf)
				im.buf = im.buf[:0]
				im.mu.Unlock()
				// Show a queued indicator if the agent is busy
				im.agent.mu.Lock()
				busy := im.agent.busy
				im.agent.mu.Unlock()
				trimmed := strings.TrimSpace(line)
				if busy && trimmed != "" {
					cprint(Gray, fmt.Sprintf("  [queued: %s]", trimmed))
				}
				// Clear input line after submit
				im.syncAndRedraw()
				im.Lines <- InputLine{Text: line, OK: true}
			case ch == 127 || ch == 8: // Backspace
				if len(im.buf) > 0 {
					im.buf = im.buf[:len(im.buf)-1]
				}
				im.mu.Unlock()
				im.syncAndRedraw()
			case ch == 3: // Ctrl-C
				im.buf = im.buf[:0]
				im.mu.Unlock()
				im.syncAndRedraw()
				// If agent is busy, cancel the current chat
				im.agent.mu.Lock()
				cancel := im.agent.cancelChat
				im.agent.mu.Unlock()
				if cancel != nil {
					cancel()
				} else {
					im.Lines <- InputLine{OK: false}
				}
			case ch == 4: // Ctrl-D
				if len(im.buf) == 0 {
					im.mu.Unlock()
					im.Lines <- InputLine{OK: false}
				} else {
					im.mu.Unlock()
				}
			case ch == 21: // Ctrl-U (kill line)
				im.buf = im.buf[:0]
				im.mu.Unlock()
				im.syncAndRedraw()
			case ch == 23: // Ctrl-W (kill word)
				for len(im.buf) > 0 && im.buf[len(im.buf)-1] == ' ' {
					im.buf = im.buf[:len(im.buf)-1]
				}
				for len(im.buf) > 0 && im.buf[len(im.buf)-1] != ' ' {
					im.buf = im.buf[:len(im.buf)-1]
				}
				im.mu.Unlock()
				im.syncAndRedraw()
			case ch == 27: // Escape sequence
				im.mu.Unlock()
				for i := 0; i < 2; i++ {
					select {
					case <-im.rawBytes:
					case <-time.After(50 * time.Millisecond):
					}
				}
			default:
				if ch >= 32 && unicode.IsPrint(rune(ch)) {
					im.buf = append(im.buf, ch)
				}
				im.mu.Unlock()
				im.syncAndRedraw()
			}

		case <-im.rawErr:
			im.Lines <- InputLine{OK: false}
			return
		}
	}
}

// ── Main loop ──

func (a *YoloAgent) showPrompt() {
	prompt := fmt.Sprintf("%s%syou> %s", Green, Bold, Reset)
	a.inputMgr.ShowPrompt(prompt)
}

func (a *YoloAgent) Run() {
	hasHistory := a.history.Load()

	if hasHistory && a.history.GetModel() != "" {
		a.resumeSession()
	} else {
		a.setupFirstRun()
	}

	a.lastActivity = time.Now()
	a.thinkDelay = IdleThinkDelay

	// Set up split terminal UI
	globalUI = NewTerminalUI()
	globalUI.Setup()
	defer func() {
		globalUI.Teardown()
		globalUI = nil
	}()

	// Start async input manager
	a.inputMgr = NewInputManager(a)
	a.inputMgr.Start()
	defer a.inputMgr.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGWINCH)
	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGWINCH:
				if globalUI != nil {
					globalUI.RefreshSize()
				}
			case syscall.SIGINT:
				a.mu.Lock()
				cancel := a.cancelChat
				a.mu.Unlock()
				if cancel != nil {
					cancel()
				} else {
					a.running = false
					cprint(Cyan, "\n  Interrupted — saving session...")
				}
			}
		}
	}()

	a.showPrompt()

	for a.running {
		select {
		case line := <-a.inputMgr.Lines:
			if !line.OK {
				a.running = false
				break
			}

			stripped := strings.TrimSpace(line.Text)
			lower := strings.ToLower(stripped)
			if lower == "exit" || lower == "quit" {
				a.running = false
			} else if strings.HasPrefix(stripped, "/") {
				a.handleCommand(stripped)
				a.showPrompt()
			} else if stripped != "" {
				a.mu.Lock()
				a.busy = true
				a.mu.Unlock()

				// Echo user's input with green color and prefix
				cprint(Green, fmt.Sprintf("  [%s] %s\n", "you", stripped))

				a.chatWithAgent(stripped, false)

				a.mu.Lock()
				a.busy = false
				a.mu.Unlock()

				// Check for any lines queued while busy
				a.drainQueuedInput()
				a.showPrompt()
			}

		case <-time.After(1 * time.Second):
			// Check for autonomous thinking
			a.inputMgr.mu.Lock()
			bufEmpty := len(a.inputMgr.buf) == 0
			a.inputMgr.mu.Unlock()
			if bufEmpty {
				elapsed := time.Since(a.lastActivity)
				if elapsed >= time.Duration(a.thinkDelay)*time.Second {
					a.inputMgr.ClearLine()
					cprint(Gray, "  [autonomous thinking...]\n")

					a.mu.Lock()
					a.busy = true
					a.mu.Unlock()

					a.chatWithAgent("", true)

					a.mu.Lock()
					a.busy = false
					a.mu.Unlock()

					a.lastActivity = time.Now()
					a.thinkDelay = ThinkLoopDelay
					a.drainQueuedInput()
					a.showPrompt()
				}
			}
		}
	}

	a.history.Save()
	fmt.Print("\r\n")
	cprint(Cyan, "  Session saved. Goodbye!\n")
}

// drainQueuedInput processes any lines that were typed while the agent was busy.
func (a *YoloAgent) drainQueuedInput() {
	for {
		select {
		case line := <-a.inputMgr.Lines:
			if !line.OK {
				a.running = false
				return
			}
			stripped := strings.TrimSpace(line.Text)
			lower := strings.ToLower(stripped)
			if lower == "exit" || lower == "quit" {
				a.running = false
				return
			} else if strings.HasPrefix(stripped, "/") {
				a.handleCommand(stripped)
			} else if stripped != "" {
				cprint(Green, fmt.Sprintf("  [%s] %s\n", "queued", stripped))

				a.mu.Lock()
				a.busy = true
				a.mu.Unlock()

				a.chatWithAgent(stripped, false)

				a.mu.Lock()
				a.busy = false
				a.mu.Unlock()
			}
		default:
			return
		}
	}
}

// ─── Entry Point ──────────────────────────────────────────────────────

func main() {
	agent := NewYoloAgent()
	agent.Run()
}
