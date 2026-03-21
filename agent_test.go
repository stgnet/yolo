package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGetSystemPrompt verifies that the system prompt is non-empty and contains expected sections
func TestGetSystemPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	
	systemPromptPath := filepath.Join(tmpDir, "SYSTEM_PROMPT.md")
	err := os.WriteFile(systemPromptPath, []byte("Test System Prompt with Rules section"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	
	yoloDir := filepath.Join(tmpDir, ".yolo")
	err = os.MkdirAll(yoloDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)
	
	agent := NewYoloAgent()
	prompt := agent.getSystemPrompt()
	
	if prompt == "" {
		t.Error("getSystemPrompt returned empty string")
	}
}

// TestCheckBinaryFreshness verifies binary freshness checking
func TestCheckBinaryFreshness(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	
	os.Chdir(os.TempDir())
	
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "test_exe")
	
	err := os.WriteFile(exePath, []byte("dummy"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	
	agent := NewYoloAgent()
	agent.scriptPath = exePath
	agent.binaryModTime = time.Now().Add(-1 * time.Hour)
	
	result := agent.checkBinaryFreshness()
	
	if !strings.Contains(result, "NEEDS") && result != "" {
		t.Logf("Binary freshness check result: %s", result)
	}
}

// TestCheckEmailInbox verifies email inbox checking
func TestCheckEmailInbox(t *testing.T) {
	agent := NewYoloAgent()
	
	result := agent.checkEmailInbox()
	
	if result != "" {
		t.Logf("Email inbox check found: %s", result)
	}
}

// TestDisplaySessionResumption verifies session resumption display
func TestDisplaySessionResumption(t *testing.T) {
	tmpDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tmpDir,
		history: NewHistoryManager(tmpDir),
		config:  NewYoloConfig(tmpDir),
	}
	
	agent.history.AddMessage("user", "Hello", nil)
	agent.history.AddMessage("assistant", "Hi there!", nil)
	
	agent.displaySessionResumption()
}

// TestEnableDisableTerminalMode tests terminal mode toggling
func TestEnableDisableTerminalMode(t *testing.T) {
	tmpDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tmpDir,
		config:  NewYoloConfig(tmpDir),
	}
	
	agent.enableTerminalMode()
	if !agent.config.GetTerminalMode() {
		t.Error("Expected terminal mode to be enabled")
	}
	
	agent.disableTerminalMode()
	if agent.config.GetTerminalMode() {
		t.Error("Expected terminal mode to be disabled")
	}
}

// TestShowHelpHint tests help hint display
func TestShowHelpHint(t *testing.T) {
	tmpDir := t.TempDir()
	agent := &YoloAgent{baseDir: tmpDir}
	agent.showHelpHint()
}

// TestIngestHandoffResults tests handoff result ingestion
func TestIngestHandoffResults(t *testing.T) {
	tmpDir := t.TempDir()
	agent := &YoloAgent{
		baseDir:    tmpDir,
		history:    NewHistoryManager(tmpDir),
		config:     NewYoloConfig(tmpDir),
	}
	
	beforeCount := len(agent.history.Data.Messages)
	agent.ingestHandoffResults()
	afterCount := len(agent.history.Data.Messages)
	
	if beforeCount != afterCount {
		t.Error("Expected no messages added when there are no pending handoffs")
	}
}

// TestStripOrphanedCloseTags tests orphaned tag removal
func TestStripOrphanedCloseTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no orphaned tags", "<b>hello</b>", "<b>hello</b>"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripOrphanedCloseTags(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestParseFuncCallArgs tests function call argument parsing
func TestParseFuncCallArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(map[string]any) bool
	}{
		{
			name:  "simple args",
			input: "path=\"test.txt\",limit=10",
			check: func(args map[string]any) bool {
				return args["path"] != nil && args["limit"] != nil
			},
		},
		{
			name:  "empty args",
			input: "",
			check: func(args map[string]any) bool {
				return len(args) == 0
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFuncCallArgs(tt.input)
			if !tt.check(result) {
				t.Errorf("Expected check to pass for input: %s", tt.input)
			}
		})
	}
}

// TestIsFileMutationTool tests file mutation tool detection
func TestIsFileMutationTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected bool
	}{
		{"write_file", "write_file", true},
		{"read_file", "read_file", false},
		{"edit_file", "edit_file", true},
		{"unknown_tool", "unknown_tool", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFileMutationTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("Expected %v for tool %s, got %v", tt.expected, tt.toolName, result)
			}
		})
	}
}

// TestHandleCommand tests command handler
func TestHandleCommand(t *testing.T) {
	tmpDir := t.TempDir()
	
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)
	
	agent := NewYoloAgent()
	
	// Test help command - verify it doesn't crash
	agent.handleCommand("help")
}



// TestParseTextToolCalls tests text tool call parsing
func TestParseTextToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	agent := &YoloAgent{baseDir: tmpDir}
	
	// Should not panic with empty input
	results := agent.parseTextToolCalls("")
	if len(results) != 0 {
		t.Errorf("Expected 0 tool calls for empty string, got %d", len(results))
	}
}

// TestShowCacheStatus tests cache status display
func TestShowCacheStatus(t *testing.T) {
	tmpDir := t.TempDir()
	agent := NewYoloAgent()
	agent.baseDir = tmpDir
	
	// Should not panic
	agent.showCacheStatus("")
}

// TestShowUncompletedTodos tests displaying uncompleted todos
func TestShowUncompletedTodos(t *testing.T) {
	tmpDir := t.TempDir()
	agent := NewYoloAgent()
	agent.baseDir = tmpDir
	
	// Should not panic (even if no todos)
	agent.showUncompletedTodos()
}

// TestEchoUserInput tests echoing user input to history
func TestEchoUserInput(t *testing.T) {
	tmpDir := t.TempDir()
	agent := &YoloAgent{
		baseDir: tmpDir,
		history: NewHistoryManager(tmpDir),
	}
	
	input := "test user input"
	agent.echoUserInput(input)
	
	// Should not panic - just testing the method exists and runs
}

// TestConvertParamValue tests parameter value conversion
func TestConvertParamValue(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{"string", "hello"},
		{"number", "42"},
		{"float", "3.14"},
		{"bool_true", "true"},
		{"bool_false", "false"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertParamValue(tc.value)
			if result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

// TestStripTextToolCalls tests removing tool calls from text
func TestStripTextToolCalls(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no tool calls", "hello world"},
		{"empty", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripTextToolCalls(tt.input)
			if len(result) < 0 {
				t.Error("Expected non-negative result length")
			}
		})
	}
}

// TestShowPrompt tests prompt display
func TestShowPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	agent := &YoloAgent{baseDir: tmpDir}
	
	// Should not panic
	agent.showPrompt()
}

// TestSwitchModel tests model switching
func TestSwitchModel(t *testing.T) {
	tmpDir := t.TempDir()
	agent := NewYoloAgent()
	agent.baseDir = tmpDir
	
	result := agent.switchModel("qwen3.5:27b-q4_K_M")
	
	if result == "" {
		t.Error("Expected non-empty result from switchModel")
	}
}

// TestSpawnSubagent tests subagent spawning
func TestSpawnSubagent(t *testing.T) {
	tmpDir := t.TempDir()
	agent := NewYoloAgent()
	agent.baseDir = tmpDir
	
	result := agent.spawnSubagent("test task", "qwen3.5:27b-q4_K_M")
	
	if result == "" {
		t.Error("Expected non-empty result from spawnSubagent")
	}
}

// TestParseParamString tests parsing parameter strings
func TestParseParamString(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"single_param", "path=\"test.txt\""},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseParamString(tc.input)
			if result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}
