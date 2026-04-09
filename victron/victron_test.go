package victron

import (
	"testing"
)

// Test containsIgnoreCase
func Test_containsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "case insensitive match",
			s:        "Hello World",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "no match",
			s:        "hello world",
			substr:   "goodbye",
			expected: false,
		},
		{
			name:     "empty string s",
			s:        "",
			substr:   "test",
			expected: false,
		},
		{
			name:     "empty substr matches",
			s:        "hello",
			substr:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreCase(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// Test toLowerSimple
func Test_toLowerSimple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "all uppercase",
			input:    "HELLO",
			expected: "hello",
		},
		{
			name:     "mixed case",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "already lowercase",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "with numbers",
			input:    "Test123",
			expected: "test123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLowerSimple(tt.input)
			if result != tt.expected {
				t.Errorf("toLowerSimple(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test findSubstring
func Test_findSubstring(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "simple match",
			s:        "hello world",
			substr:   "lo wo",
			expected: true,
		},
		{
			name:     "no match",
			s:        "hello world",
			substr:   "xyz",
			expected: false,
		},
		{
			name:     "empty s",
			s:        "",
			substr:   "test",
			expected: false,
		},
		{
			name:     "empty substr returns true (no search needed)",
			s:        "hello",
			substr:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findSubstring(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("findSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// Test removeHyphens
func Test_removeHyphens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with hyphens",
			input:    "12-34-56-78-90",
			expected: "1234567890",
		},
		{
			name:     "no hyphens",
			input:    "1234567890",
			expected: "1234567890",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple consecutive hyphens",
			input:    "12--34--56",
			expected: "123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeHyphens(tt.input)
			if result != tt.expected {
				t.Errorf("removeHyphens(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test matchesFilter
func Test_matchesFilter(t *testing.T) {
	tests := []struct {
		name     string
		device   DiscoverableDevice
		filter   ScanFilter
		expected bool
	}{
		{
			name: "no filter matches everything",
			device: DiscoverableDevice{
				Name: "test device",
			},
			filter:   ScanFilter{},
			expected: true,
		},
		{
			name: "name filter matches",
			device: DiscoverableDevice{
				Name: "victron device",
			},
			filter:   ScanFilter{Name: "victron"},
			expected: true,
		},
		{
			name: "name filter case insensitive",
			device: DiscoverableDevice{
				Name: "VICTRON Device",
			},
			filter:   ScanFilter{Name: "victron"},
			expected: true,
		},
		{
			name: "name filter no match",
			device: DiscoverableDevice{
				Name: "other device",
			},
			filter:   ScanFilter{Name: "victron"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesFilter(tt.device, tt.filter)
			if result != tt.expected {
				t.Errorf("matchesFilter() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test calculateChecksum
func Test_calculateChecksum(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected byte
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "single byte",
			data:     []byte{0x41},
			expected: 0x41,
		},
		{
			name:     "two bytes",
			data:     []byte{0x41, 0x42},
			expected: 0x41 ^ 0x42,
		},
		{
			name:     "three bytes",
			data:     []byte{0x00, 0xFF, 0xAA},
			expected: 0x00 ^ 0xFF ^ 0xAA,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateChecksum(tt.data)
			if result != tt.expected {
				t.Errorf("calculateChecksum(%v) = %X, want %X", tt.data, result, tt.expected)
			}
		})
	}
}
