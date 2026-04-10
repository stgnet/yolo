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

// Test all supported actions
func TestVictronToolAllActions(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}

	actions := []string{"scan", "connect", "disconnect", "get_values", "subscribe", "device_info"}
	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			params := map[string]any{"action": action}
			if action != "scan" {
				params["address"] = "XX:XX:XX:XX:XX:XX"
			}

			result := tool.victron(params)
			if !contains(result, `"status"`) {
				t.Errorf("Expected status field for %s action, got: %s", action, result)
			}
		})
	}
}

func TestVictronToolInvalidAction(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{"action": "invalid_action"})

	if !contains(result, `"status":"error"`) || !contains(result, `unknown action`) {
		t.Errorf("Expected error about unknown action, got: %s", result)
	}
}

func TestVictronToolConnectWithTimeout(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{
		"action":  "connect",
		"address": "XX:XX:XX:XX:XX:XX",
		"timeout": "1",
	})

	if !contains(result, `"status"`) {
		t.Errorf("Expected status field in connect result, got: %s", result)
	}
}

func TestVictronToolScanWithDuration(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{
		"action":   "scan",
		"duration": "2",
	})

	if !contains(result, `"status"`) {
		t.Errorf("Expected status field in scan result, got: %s", result)
	}
}

func TestVictronToolDeviceInfo(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{
		"action":  "device_info",
		"address": "XX:XX:XX:XX:XX:XX",
	})

	if !contains(result, `"status"`) {
		t.Errorf("Expected status field in device_info result, got: %s", result)
	}
}

func TestVictronToolSubscribe(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{
		"action":  "subscribe",
		"address": "XX:XX:XX:XX:XX:XX",
	})

	if !contains(result, `"status"`) {
		t.Errorf("Expected status field in subscribe result, got: %s", result)
	}
}

func TestVictronToolDisconnect(t *testing.T) {
	tool := &ToolExecutor{baseDir: "."}
	result := tool.victron(map[string]any{
		"action":  "disconnect",
		"address": "XX:XX:XX:XX:XX:XX",
	})

	if !contains(result, `"status"`) {
		t.Errorf("Expected status field in disconnect result, got: %s", result)
	}
}

func TestVictronToolInferDeviceType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"SmartSolar MPPT 100/30", "SmartSolar"},
		{"MPPT 75/15", "SmartSolar"},
		{"SmartShunt Bluetooth", "SmartShunt"},
		{"VE.Direct Adapter", "VEDirectAdapter"},
		{"victron bluetooth", "UnknownVictron"},
		{"Unknown Device", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferDeviceType(tt.name)
			if result != tt.expected {
				t.Errorf("Expected device type %s for '%s', got %s", tt.expected, tt.name, result)
			}
		})
	}
}

func TestVictronToolEnrichValue(t *testing.T) {
	val := victronValue{
		Key:        "V",
		RawValue:   "13.85",
		FloatValue: 13.85,
		Timestamp:  "2024-01-01T00:00:00Z",
	}

	enriched := enrichValue(val)

	if enriched.Name != "System Voltage" {
		t.Errorf("Expected name 'System Voltage', got '%s'", enriched.Name)
	}
	if enriched.Unit != "V" {
		t.Errorf("Expected unit 'V', got '%s'", enriched.Unit)
	}

	// Test unknown key - should not modify
	val2 := victronValue{
		Key:        "UnknownKey",
		RawValue:   "100",
		FloatValue: 100,
		Timestamp:  "2024-01-01T00:00:00Z",
	}

	enriched2 := enrichValue(val2)
	if enriched2.Name != "" {
		t.Errorf("Expected empty name for unknown key, got '%s'", enriched2.Name)
	}
	if enriched2.Unit != "" {
		t.Errorf("Expected empty unit for unknown key, got '%s'", enriched2.Unit)
	}
}
