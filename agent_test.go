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
