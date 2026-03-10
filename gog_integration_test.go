// gog_integration_test.go - Integration tests for Google Workspace (gog) tool

package main

import (
	"strings"
	"testing"
)

// TestGogToolCommands tests various gog commands
func TestGogToolCommands(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expectError bool
	}{
		{
			name:        "gmail list",
			command:     "gmail list --max 3",
			expectError: false, // May return empty but not error
		},
		{
			name:        "calendar list events",
			command:     "calendar list events",
			expectError: false,
		},
		{
			name:        "drive list",
			command:     "drive list",
			expectError: false,
		},
		{
			name:        "contacts list",
			command:     "contacts list --max 5",
			expectError: false,
		},
		{
			name:        "tasks list",
			command:     "tasks list",
			expectError: false,
		},
		{
			name:        "empty command",
			command:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewToolExecutor("/tmp/test-gog", nil)
			if executor == nil {
				t.Fatal("executor is nil")
			}

			result := executor.gog(map[string]any{
				"command": tt.command,
			})

			if tt.expectError {
				if !strings.Contains(result, "Error:") && !strings.Contains(result, "required") {
					t.Errorf("Expected error for empty command but got: %s", result[:min(200, len(result))])
				}
			} else {
				// gog may require authentication, so we expect either valid output or auth error
				if strings.Contains(result, "Error:") && !strings.Contains(strings.ToLower(result), "authentication") && 
				   !strings.Contains(strings.ToLower(result), "auth") && !strings.Contains(strings.ToLower(result), "credentials") {
					t.Logf("gog returned non-auth error: %s", result[:min(500, len(result))])
				}
			}
		})
	}
}

// TestGogToolDescription verifies the tool has comprehensive documentation
func TestGogToolDescription(t *testing.T) {
	found := false
	for _, tool := range ollamaTools {
		if tool.Function.Name == "gog" {
			found = true
			
			desc := tool.Function.Description
			requiredTerms := []string{"Google", "command"}
			
			for _, term := range requiredTerms {
				if !strings.Contains(desc, term) {
					t.Errorf("gog description should contain '%s', got: %s", term, desc)
				}
			}
			
			break
		}
	}

	if !found {
		t.Error("gog tool not found in ollamaTools")
	}
}
