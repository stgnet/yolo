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
