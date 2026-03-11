package main

import (
	"testing"
)

// TestGetBoolArg tests the getBoolArg helper function
func TestGetBoolArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		expected bool
	}{
		{
			name:     "boolean true",
			args:     map[string]any{"key": true},
			key:      "key",
			expected: true,
		},
		{
			name:     "boolean false",
			args:     map[string]any{"key": false},
			key:      "key",
			expected: false,
		},
		{
			name:     "string true",
			args:     map[string]any{"key": "true"},
			key:      "key",
			expected: true,
		},
		{
			name:     "string yes",
			args:     map[string]any{"key": "yes"},
			key:      "key",
			expected: true,
		},
		{
			name:     "string 1",
			args:     map[string]any{"key": "1"},
			key:      "key",
			expected: true,
		},
		{
			name:     "string false",
			args:     map[string]any{"key": "false"},
			key:      "key",
			expected: false,
		},
		{
			name:     "string no",
			args:     map[string]any{"key": "no"},
			key:      "key",
			expected: false,
		},
		{
			name:     "string 0",
			args:     map[string]any{"key": "0"},
			key:      "key",
			expected: false,
		},
		{
			name:     "missing key defaults to false",
			args:     map[string]any{},
			key:      "missing",
			expected: false,
		},
		{
			name:     "nil value defaults to false",
			args:     map[string]any{"key": nil},
			key:      "key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolArg(tt.args, tt.key, false)
			if result != tt.expected {
				t.Errorf("getBoolArg(%v, %q, false) = %v, want %v", tt.args, tt.key, result, tt.expected)
			}
		})
	}
}
