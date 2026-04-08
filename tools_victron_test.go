package main

import (
	"testing"
)

func TestVictronToolScan(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{
		"action":   "scan",
		"duration": "1",
	})
	
	// Should return a JSON result (even if no devices found)
	if !contains(result, `"status"`) {
		t.Errorf("Expected status field in result, got: %s", result)
	}
}

func TestVictronToolMissingAction(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{})
	
	if !contains(result, `"status":"error"`) || !contains(result, `action`) {
		t.Errorf("Expected error about missing action, got: %s", result)
	}
}

func TestVictronToolConnectWithoutAddress(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{
		"action": "connect",
	})
	
	if !contains(result, `"status":"error"`) || !contains(result, `address`) {
		t.Errorf("Expected error about missing address, got: %s", result)
	}
}

func TestVictronToolGetValuesWithoutConnection(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{
		"action":  "get_values",
		"address": "XX:XX:XX:XX:XX:XX",
	})
	
	if !contains(result, `"status":"error"`) || !contains(result, `not connected`) {
		t.Errorf("Expected error about device not connected, got: %s", result)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
