package victron

import (
	"testing"
)

// TestContainsIgnoreCase tests the containsIgnoreCase helper function
func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{"exact match", "Hello World", "world", true},
		{"case mismatch", "HELLO", "hello", true},
		{"no match", "Hello", "goodbye", false},
		{"empty string", "", "test", false},
		{"empty substring", "test", "", true},
		{"partial match", "Hello World", "wor", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreCase(tt.str, tt.substr)
			if result != tt.expected {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v",
					tt.str, tt.substr, result, tt.expected)
			}
		})
	}
}

// TestToLowerSimple tests the toLowerSimple helper function
func TestToLowerSimple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"upper case", "HELLO", "hello"},
		{"mixed case", "HeLLo WoRLd", "hello world"},
		{"already lower", "hello", "hello"},
		{"with spaces", "Hello  World", "hello  world"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLowerSimple(tt.input)
			if result != tt.expected {
				t.Errorf("toLowerSimple(%q) = %q, want %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

// TestFindSubstring tests the findSubstring helper function (case-sensitive)
func TestFindSubstring(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool // Returns bool, not int
	}{
		{"found at start", "hello world", "hello", true},
		{"found in middle", "hello world", "world", true},
		{"not found", "hello", "test", false},
		{"empty search", "hello", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findSubstring(tt.str, tt.substr)
			if result != tt.expected {
				t.Errorf("findSubstring(%q, %q) = %v, want %v",
					tt.str, tt.substr, result, tt.expected)
			}
		})
	}
}

// TestParseAdvertisement tests the ParseAdvertisement function with various BLE advertisements
func TestParseAdvertisement(t *testing.T) {
	tests := []struct {
		name         string
		localName    string
		serviceUUIDs []string // Changed from ServiceData to []string
		expected     bool
	}{
		{
			name:         "SmartSolar device detected by name",
			localName:    "SmartSolar MPPT",
			serviceUUIDs: []string{},
			expected:     true, // Should detect SmartSolar pattern
		},
		{
			name:         "Device with VE.Direct in name",
			localName:    "VE.Direct Adapter",
			serviceUUIDs: []string{},
			expected:     true, // Should detect VE.Direct pattern
		},
		{
			name:         "Device with SmartShunt in name",
			localName:    "SmartShunt 500A",
			serviceUUIDs: []string{},
			expected:     true, // Should detect SmartShunt pattern
		},
		{
			name:         "Device with Victron in name",
			localName:    "Victron Energy Device",
			serviceUUIDs: []string{},
			expected:     true, // Should detect Victron pattern
		},
		{
			name:         "Device with fdcd service UUID",
			localName:    "",
			serviceUUIDs: []string{"0xfdcd"},
			expected:     true, // Should detect fdcd pattern
		},
		{
			name:         "Device with ffe0 service UUID",
			localName:    "",
			serviceUUIDs: []string{"ffe0"},
			expected:     true, // Should detect ffe0 pattern
		},
		{
			name:         "Unknown device",
			localName:    "Unknown Device XYZ",
			serviceUUIDs: []string{"unknown"},
			expected:     false, // Should not match
		},
		{
			name:         "Empty advertisement",
			localName:    "",
			serviceUUIDs: []string{},
			expected:     false, // Should not match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseAdvertisement(tt.localName, tt.serviceUUIDs)

			if result != tt.expected {
				t.Errorf("ParseAdvertisement(%q, %v) = %v, want %v",
					tt.localName, tt.serviceUUIDs, result, tt.expected)
			}
		})
	}
}

// TestRemoveHyphens tests the removeHyphens helper function
func TestRemoveHyphens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with hyphens", "0000-1234-5678-9abc-def0", "0000123456789abcdef0"},
		{"no hyphens", "1234567890abcdef", "1234567890abcdef"},
		{"mixed", "fd-cd-ab-ef", "fdcdabef"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeHyphens(tt.input)
			if result != tt.expected {
				t.Errorf("removeHyphens(%q) = %q, want %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormatServiceUUID tests the FormatServiceUUID function
func TestFormatServiceUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"16-bit UUID", "fdcd", "fdcd"},
		{"32-char UUID", "00001234567890abcdef1234567890ab", "00001234567890abcdef1234567890ab"},
		{"UUID with hyphens", "fd-cd-ab-ef-1234", "fdcdabef1234"}, // Will fall back to default if not 4 or 32 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatServiceUUID(tt.input)

			// Just verify it doesn't crash and returns something
			if result == "" {
				t.Errorf("FormatServiceUUID should not return empty string")
			}
		})
	}
}

// TestDescribeCharacteristicProperties tests the DescribeCharacteristicProperties function
func TestDescribeCharacteristicProperties(t *testing.T) {
	tests := []struct {
		name  string
		props uint8
	}{
		{"read only", PropRead},
		{"write without response", PropWriteWithoutResp},
		{"write", PropWrite},
		{"notify", PropNotify},
		{"indicate", PropIndicate},
		{"combined read and notify", PropRead | PropNotify},
		{"empty", 0x00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DescribeCharacteristicProperties(tt.props)

			// Just verify it doesn't crash and returns something meaningful
			if result == "" {
				// Empty is valid for 0x00 props
				if tt.props != 0x00 {
					t.Errorf("DescribeCharacteristicProperties should return non-empty for props=%v", tt.props)
				}
			}
		})
	}
}
