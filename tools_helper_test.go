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

// TestGetBoolArgFloat64 tests the float64 case in getBoolArg
func TestGetBoolArgFloat64(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		expected bool
	}{
		{
			name:     "float64 1.0 returns true",
			args:     map[string]any{"key": float64(1.0)},
			key:      "key",
			expected: true,
		},
		{
			name:     "float64 0.0 returns false",
			args:     map[string]any{"key": float64(0.0)},
			key:      "key",
			expected: false,
		},
		{
			name:     "float64 2.0 returns false",
			args:     map[string]any{"key": float64(2.0)},
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

// TestGetBoolArgInt tests the int case in getBoolArg
func TestGetBoolArgInt(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		expected bool
	}{
		{
			name:     "int 1 returns true",
			args:     map[string]any{"key": int(1)},
			key:      "key",
			expected: true,
		},
		{
			name:     "int 0 returns false",
			args:     map[string]any{"key": int(0)},
			key:      "key",
			expected: false,
		},
		{
			name:     "int 2 returns false",
			args:     map[string]any{"key": int(2)},
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

// TestGetBoolArgFallback tests the fallback parameter
func TestGetBoolArgFallback(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		fallback bool
		expected bool
	}{
		{
			name:     "missing key with true fallback",
			args:     map[string]any{},
			key:      "missing",
			fallback: true,
			expected: true,
		},
		{
			name:     "nil value with true fallback",
			args:     map[string]any{"key": nil},
			key:      "key",
			fallback: true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolArg(tt.args, tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("getBoolArg(%v, %q, %v) = %v, want %v", tt.args, tt.key, tt.fallback, result, tt.expected)
			}
		})
	}
}


