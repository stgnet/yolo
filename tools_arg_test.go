package main

import (
	"testing"
)

// Test getStringArg
func TestGetStringArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		fallback string
		want     string
	}{
		{
			name:     "string value exists",
			args:     map[string]any{"key": "value"},
			key:      "key",
			fallback: "default",
			want:     "value",
		},
		{
			name:     "key not found",
			args:     map[string]any{},
			key:      "key",
			fallback: "default",
			want:     "default",
		},
		{
			name:     "int value converted to string",
			args:     map[string]any{"key": 123},
			key:      "key",
			fallback: "default",
			want:     "123",
		},
		{
			name:     "bool value converted to string",
			args:     map[string]any{"key": true},
			key:      "key",
			fallback: "default",
			want:     "true",
		},
		{
			name:     "empty string value",
			args:     map[string]any{"key": ""},
			key:      "key",
			fallback: "default",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStringArg(tt.args, tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("getStringArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test getIntArg
func TestGetIntArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		fallback int
		want     int
	}{
		{
			name:     "float64 value (from JSON)",
			args:     map[string]any{"key": float64(42)},
			key:      "key",
			fallback: 0,
			want:     42,
		},
		{
			name:     "int value",
			args:     map[string]any{"key": 100},
			key:      "key",
			fallback: 0,
			want:     100,
		},
		{
			name:     "string numeric value",
			args:     map[string]any{"key": "123"},
			key:      "key",
			fallback: 0,
			want:     123,
		},
		{
			name:     "string zero value (should return 0)",
			args:     map[string]any{"key": "0"},
			key:      "key",
			fallback: -1,
			want:     0, // This is a bug! Currently returns fallback (-1) instead of 0
		},
		{
			name:     "string negative value (should return negative)",
			args:     map[string]any{"key": "-5"},
			key:      "key",
			fallback: 10,
			want:     -5, // This is a bug! Currently returns fallback (10) instead of -5
		},
		{
			name:     "key not found returns fallback",
			args:     map[string]any{},
			key:      "missing",
			fallback: 42,
			want:     42,
		},
		{
			name:     "string non-numeric value returns fallback",
			args:     map[string]any{"key": "abc"},
			key:      "key",
			fallback: 99,
			want:     99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getIntArg(tt.args, tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("getIntArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test getBoolArg
func TestGetBoolArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		fallback bool
		want     bool
	}{
		{
			name:     "bool true value",
			args:     map[string]any{"key": true},
			key:      "key",
			fallback: false,
			want:     true,
		},
		{
			name:     "bool false value",
			args:     map[string]any{"key": false},
			key:      "key",
			fallback: true,
			want:     false,
		},
		{
			name:     "string 'true'",
			args:     map[string]any{"key": "true"},
			key:      "key",
			fallback: false,
			want:     true,
		},
		{
			name:     "string 'false'",
			args:     map[string]any{"key": "false"},
			key:      "key",
			fallback: true,
			want:     false,
		},
		{
			name:     "string 'yes'",
			args:     map[string]any{"key": "yes"},
			key:      "key",
			fallback: false,
			want:     true,
		},
		{
			name:     "string 'no'",
			args:     map[string]any{"key": "no"},
			key:      "key",
			fallback: true,
			want:     false,
		},
		{
			name:     "string '1'",
			args:     map[string]any{"key": "1"},
			key:      "key",
			fallback: false,
			want:     true,
		},
		{
			name:     "string '0'",
			args:     map[string]any{"key": "0"},
			key:      "key",
			fallback: true,
			want:     false,
		},
		{
			name:     "float64 1.0",
			args:     map[string]any{"key": 1.0},
			key:      "key",
			fallback: false,
			want:     true,
		},
		{
			name:     "float64 0.0",
			args:     map[string]any{"key": 0.0},
			key:      "key",
			fallback: true,
			want:     false,
		},
		{
			name:     "int 1",
			args:     map[string]any{"key": 1},
			key:      "key",
			fallback: false,
			want:     true,
		},
		{
			name:     "int 0",
			args:     map[string]any{"key": 0},
			key:      "key",
			fallback: true,
			want:     false,
		},
		{
			name:     "key not found returns fallback",
			args:     map[string]any{},
			key:      "missing",
			fallback: true,
			want:     true,
		},
		{
			name:     "string 'TRUE' (case insensitive)",
			args:     map[string]any{"key": "TRUE"},
			key:      "key",
			fallback: false,
			want:     true,
		},
		{
			name:     "string 'YES' (case insensitive)",
			args:     map[string]any{"key": "YES"},
			key:      "key",
			fallback: false,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBoolArg(tt.args, tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("getBoolArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test isBinaryData
func TestIsBinaryData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "empty data",
			data: []byte{},
			want: false,
		},
		{
			name: "plain text",
			data: []byte("Hello, World!"),
			want: false,
		},
		{
			name: "text with null byte",
			data: []byte("Hello\x00World"),
			want: true,
		},
		{
			name: "binary data with control chars",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0x04},
			want: true,
		},
		{
			name: "text with line breaks",
			data: []byte("Line 1\nLine 2\r\nLine 3"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinaryData(tt.data)
			if got != tt.want {
				t.Errorf("isBinaryData() = %v, want %v", got, tt.want)
			}
		})
	}
}
