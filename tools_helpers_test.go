package main

import (
	"testing"
)

func TestGetBoolArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		expected bool
	}{
		{"explicit true", map[string]any{"key": true}, "key", true},
		{"explicit false", map[string]any{"key": false}, "key", false},
		{"string true", map[string]any{"key": "true"}, "key", true},
		{"string yes", map[string]any{"key": "yes"}, "key", false},
		{"string 1", map[string]any{"key": "1"}, "key", true},
		{"string false", map[string]any{"key": "false"}, "key", false},
		{"string no", map[string]any{"key": "no"}, "key", false},
		{"string 0", map[string]any{"key": "0"}, "key", false},
		{"missing key", map[string]any{}, "key", false},
		{"int true", map[string]any{"key": 1}, "key", true},
		{"int false", map[string]any{"key": 0}, "key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolArg(tt.args, tt.key, false)
			if result != tt.expected {
				t.Errorf("getBoolArg() = %v, want %v", result, tt.expected)
			}
		})
	}
}
