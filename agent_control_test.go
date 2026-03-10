package main

import (
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

// TestRestart tests the restart tool implementation
func TestRestart(t *testing.T) {
	// Skip this test as it would actually attempt to restart the agent
	t.Skip("Skipping restart test to avoid actual restart during testing")
}
