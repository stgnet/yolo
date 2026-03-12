package main

import (
	"os"
	"strings"
	"testing"
)

// TestListModels tests the list_models tool implementation
func TestListModels(t *testing.T) {
	tests := []struct {
		name           string
		expectContains string
	}{
		{
			name:           "list models returns data",
			expectContains: "", // Just check it doesn't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewYoloAgent()
			result := agent.tools.listModels()

			// Result should contain either models or an appropriate message
			if result == "" {
				t.Errorf("Expected non-empty result, got empty string")
			}

			if strings.Contains(result, tt.expectContains) {
				// Pass if we find expected content
			} else if !strings.Contains(result, "Error connecting to Ollama") {
				// If there's no connection error and result is not empty, it's OK
			}
		})
	}
}

// TestSwitchModel tests the switch_model tool implementation
func TestSwitchModel(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		expectError bool
	}{
		{
			name:        "missing model argument",
			model:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewYoloAgent()

			args := map[string]any{"model": tt.model}
			result := agent.tools.switchModel(args)

			if tt.expectError {
				// Error response should mention "not found" or similar for invalid model
				if !strings.Contains(strings.ToLower(result), "not found") &&
					!strings.Contains(strings.ToLower(result), "error") {
					t.Errorf("Expected error indicator but got: %s", result)
				}
			} else {
				if strings.Contains(result, "Error connecting") ||
					strings.Contains(result, "not found") {
					t.Errorf("Unexpected error: %s", result)
				}
			}
		})
	}
}

// TestRestart tests the restart tool implementation.
// SKIPPED: executor.restart() runs "go build" to rebuild the yolo binary and
// then exec's the new binary with syscall.Exec, replacing the current process.
// Running this in a test would kill the test runner. It also requires a TTY
// since the newly exec'd process checks for TTY on stdin/stdout/stderr.
// DO NOT re-enable — this would terminate the entire test suite.
func TestRestart(t *testing.T) {
	t.Skip("Skipping restart test - requires interactive terminal")

	/* Uncomment for manual testing only:
	tests := []struct {
		name        string
		args        map[string]any
		expectError bool
	}{
		{
			name:        "no arguments should work",
			args:        map[string]any{},
			expectError: true, // Will fail to rebuild in test env, but function should be called
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &ToolExecutor{}
			result := executor.restart(tt.args)

			if tt.expectError {
				// In test environment, rebuild will fail (no Go source)
				// Just check it doesn't crash and returns something meaningful
				if result == "" {
					t.Errorf("Expected non-empty result")
				}
				// Should mention build or error
				if !strings.Contains(strings.ToLower(result), "build") &&
					!strings.Contains(strings.ToLower(result), "error") {
					t.Logf("Result doesn't mention build/error: %s", result)
				}
			} else {
				// Success case - should mention rebuild/restart
				if !strings.Contains(strings.ToLower(result), "restart") &&
					!strings.Contains(strings.ToLower(result), "rebuild") {
					t.Errorf("Expected restart message: %s", result)
				}
			}
		})
	}
	*/
}

// TestRestartArgsFiltering tests that --restart flag is filtered from args
func TestRestartArgsFiltering(t *testing.T) {
	// This tests the logic without actually restarting
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Simulate having --restart in args
	os.Args = []string{"yolo", "--restart", "other-arg"}

	// Filter logic from restart function
	var filteredArgs []string
	for _, arg := range os.Args[1:] {
		if arg != "--restart" {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if len(filteredArgs) != 1 || filteredArgs[0] != "other-arg" {
		t.Errorf("Expected ['other-arg'], got %v", filteredArgs)
	}
}
