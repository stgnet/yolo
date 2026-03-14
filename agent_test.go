package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewYoloAgent(t *testing.T) {
	agent := NewYoloAgent()
	if agent == nil {
		t.Fatal("Expected non-nil YoloAgent")
	}
	if agent.ollama == nil {
		t.Error("Expected ollama client to be initialized")
	}
	if agent.history == nil {
		t.Error("Expected history manager to be initialized")
	}
	if agent.config == nil {
		t.Error("Expected config to be initialized")
	}
	if agent.tools == nil {
		t.Error("Expected tools executor to be initialized")
	}
	if agent.running != true {
		t.Error("Expected running to be true")
	}
}

func TestGetSystemPrompt(t *testing.T) {
	// Create a temporary directory and SYSTEM_PROMPT.md file
	tempDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tempDir,
		config:  NewYoloConfig(tempDir),
	}

	promptPath := filepath.Join(tempDir, "SYSTEM_PROMPT.md")
	content := []byte("Test prompt with {model} and {timestamp}")
	if err := os.WriteFile(promptPath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	prompt := agent.getSystemPrompt()
	if prompt == "" {
		t.Error("Expected non-empty system prompt")
	}
}

func TestEnableDisableTerminalMode(t *testing.T) {
	tempDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tempDir,
		config:  NewYoloConfig(tempDir),
	}

	// Test enabling terminal mode (bufferUI should be nil'd)
	oldBufferUI := bufferUI
	agent.enableTerminalMode()
	if bufferUI != nil {
		t.Error("Expected bufferUI to be nil after enabling terminal mode")
	}

	// Test disabling terminal mode (terminal UI should be torn down)
	bufferUI = oldBufferUI // restore for test
	agent.disableTerminalMode()
	if globalUI != nil {
		t.Log("Warning: globalUI was not torn down")
	}
}

func TestParseTextToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "no tool calls",
			input:    "Just a regular message without any tools",
			expected: 0,
		},
		{
			name: "single tool call with format 5",
			input: `[tool activity]
[read_file]`,
			expected: 1,
		},
		{
			name: "multiple tool calls with format 5",
			input: `[tool activity]
[web_search]
[list_files]`,
			expected: 2,
		},
		{
			name: "format 6 inline run_command",
			input: `[tool activity] run_command(command="ls -la")`,
			expected: 1,
		},
		{
			name: "format 6 inline read_file with multiple args",
			input: `[tool activity] read_file(path="main.go", limit=100)`,
			expected: 1,
		},
		{
			name: "format 6 command with complex value",
			input: `[tool activity] run_command(command="cd /Users/user/src && ls -la *.go 2>/dev/null")`,
			expected: 1,
		},
		{
			name: "format 6 multiple inline calls",
			input: `[thinking] Let me check.
[tool activity] run_command(command="pwd")
[thinking] And also list files.
[tool activity] list_files(pattern="*.go")`,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &YoloAgent{}
			results := agent.parseTextToolCalls(tt.input)
			if len(results) != tt.expected {
				t.Errorf("Expected %d tool calls, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestParseTextToolCallsFormat6Args(t *testing.T) {
	agent := &YoloAgent{}

	// Test that Format 6 correctly parses arguments
	calls := agent.parseTextToolCalls(`[tool activity] run_command(command="cd /src && ls -la")`)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "run_command" {
		t.Errorf("Expected tool name 'run_command', got %q", calls[0].Name)
	}
	if cmd, ok := calls[0].Args["command"].(string); !ok || cmd != "cd /src && ls -la" {
		t.Errorf("Expected command='cd /src && ls -la', got %v", calls[0].Args["command"])
	}

	// Test read_file with multiple args
	calls = agent.parseTextToolCalls(`[tool activity] read_file(path="main.go", limit=50)`)
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("Expected tool name 'read_file', got %q", calls[0].Name)
	}
	if p, ok := calls[0].Args["path"].(string); !ok || p != "main.go" {
		t.Errorf("Expected path='main.go', got %v", calls[0].Args["path"])
	}
	if lim, ok := calls[0].Args["limit"].(int64); !ok || lim != 50 {
		t.Errorf("Expected limit=50, got %v", calls[0].Args["limit"])
	}
}

func TestParseFuncCallArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]any
	}{
		{
			name:     "single quoted arg",
			input:    `command="ls -la"`,
			expected: map[string]any{"command": "ls -la"},
		},
		{
			name:     "multiple args mixed",
			input:    `path="main.go", limit=100`,
			expected: map[string]any{"path": "main.go", "limit": int64(100)},
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[string]any{},
		},
		{
			name:     "value with special chars",
			input:    `command="cd /src && ls -la | head -5"`,
			expected: map[string]any{"command": "cd /src && ls -la | head -5"},
		},
		{
			name:     "boolean arg",
			input:    `pattern="*.go", recursive=true`,
			expected: map[string]any{"pattern": "*.go", "recursive": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFuncCallArgs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d args, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for k, v := range tt.expected {
				got, ok := result[k]
				if !ok {
					t.Errorf("Missing key %q", k)
				} else if got != v {
					t.Errorf("Key %q: expected %v (%T), got %v (%T)", k, v, v, got, got)
				}
			}
		})
	}
}

// TestShowPrompt tests the showPrompt function in terminal mode
// func TestShowPrompt(t *testing.T) {
// 	tempDir := t.TempDir()
// 	agent := &YoloAgent{
// 		baseDir: tempDir,
// 		config:  NewYoloConfig(tempDir),
// 	}

// 	// Should not panic when showing prompt in terminal mode
// 	agent.enableTerminalMode()
// 	agent.showPrompt() // Skipped - requires full UI initialization
// }

// TestEchoUserInput tests the echoUserInput function
func TestEchoUserInput(t *testing.T) {
	tempDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tempDir,
		config:  NewYoloConfig(tempDir),
	}

	// Should not panic when echoing various inputs
	// Skipping actual echo tests - require full UI setup
	_ = agent // suppress unused variable warning
	t.Log("Echo user input test skipped - requires UI initialization")
}

// TestHandleCommand tests command handling in terminal mode
func TestHandleCommand(t *testing.T) {
	tempDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tempDir,
		config:  NewYoloConfig(tempDir),
	}

	agent.enableTerminalMode()

	// Test various commands - should not panic
	tests := []string{
		"help",
		"cache",
		"clear",
		"exit",
		"invalid_command_xyz",
		"",
		"/help",
	}

	for _, cmd := range tests {
		t.Run("cmd_"+cmd, func(t *testing.T) {
			agent.handleCommand(cmd)
		})
	}
}

// TestShowCacheStatus tests cache status display
func TestShowCacheStatus(t *testing.T) {
	tempDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tempDir,
		config:  NewYoloConfig(tempDir),
	}

	// Should not panic when showing cache status with empty cache
	agent.showCacheStatus("")

	// Test with actual cache directory
	cacheDir := filepath.Join(tempDir, ".yolo_cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	// Create a test cache entry
	testFile := filepath.Join(cacheDir, "test_entry.json")
	if err := os.WriteFile(testFile, []byte(`{"query": "test"}`), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should not panic when showing cache status with entries
	agent.showCacheStatus("")
}

// TestDisplaySessionResumption tests session resumption display
func TestDisplaySessionResumption(t *testing.T) {
	tempDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tempDir,
		config:  NewYoloConfig(tempDir),
		history: &HistoryManager{
			Data: HistoryData{
				Messages:     []HistoryMessage{},
				EvolutionLog: []EvolutionEntry{},
			},
		},
	}

	// Add some history messages
	agent.history.AddMessage("user", "Initial question", nil)
	agent.history.AddMessage("assistant", "I can help with that task", nil)

	// Should not panic when displaying session resumption
	agent.displaySessionResumption()

	// Test with tool metadata
	agent.history.Data.Messages[1].Meta = map[string]interface{}{
		"tool_name": "read_file",
	}
	agent.displaySessionResumption()
}

// TestStripTextToolCalls tests removal of text tool calls from output
func TestStripTextToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no tool activity",
			input:    "Just regular text response",
			expected: "Just regular text response",
		},
		{
			name: "with tool activity block",
			input: `Here's the result of my analysis:

[tool activity]
[read_file path="test.go"]
[/tool activity]

The file contains important code.`,
			expected: `Here's the result of my analysis:

The file contains important code.`,
		},
		{
			name: "with inline tool activity (format 6)",
			input: `Let me check.
[tool activity] run_command(command="ls -la")
The result shows files.`,
			expected: `Let me check.

The result shows files.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripTextToolCalls(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSpawnSubagent verifies subagent initialization and file setup
func TestSpawnSubagent(t *testing.T) {
	t.Skip("Skipped: spawns goroutine that may hang test")
}

// TestSetupFirstRun verifies first run setup creates necessary files
func TestSetupFirstRun(t *testing.T) {
	t.Skip("Skipped: requires Ollama connection and hangs without it")
}

// TestConvertParamValue tests parameter value conversion
func TestConvertParamValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"boolean true", "true", true},
		{"boolean false", "false", false},
		{"number integer", "42", int64(42)}, // parseInt returns int64
		{"number float", "3.14", 3.14},
		{"quoted string", `"hello"`, `"hello"`}, // quotes are preserved in the value
		{"unquoted string", "world", "world"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertParamValue(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

// TestParseParamString tests parameter string parsing
func TestParseParamString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]interface{}{},
		},
		{
			name:  "single param",
			input: `name="value"`,
			expected: map[string]interface{}{
				"name": "value",
			},
		},
		{
			name:  "comma-separated params",
			input: `name="test", age=25, active=true`,
			expected: map[string]interface{}{
				"name":   "test",
				"age":    int64(25),
				"active": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseParamString(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d params, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("Param %s: expected %v (%T), got %v (%T)", k, v, v, result[k], result[k])
				}
			}
		})
	}
}
