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

// Test NewScanFilter
func Test_NewScanFilter(t *testing.T) {
	tests := []struct {
		name         string
		deviceType   DeviceType
		expectedName string
	}{
		{
			name:         "SmartSolar device type",
			deviceType:   DeviceTypeSmartSolar,
			expectedName: "SmartSolar",
		},
		{
			name:         "SmartShunt device type",
			deviceType:   DeviceTypeSmartShunt,
			expectedName: "SmartShunt",
		},
		{
			name:         "VEDirectAdapter device type",
			deviceType:   DeviceTypeVEDirectAdapter,
			expectedName: "VE.Direct",
		},
		{
			name:         "Unknown device type returns empty filter",
			deviceType:   DeviceTypeUnknown,
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewScanFilter(tt.deviceType)
			if result.Name != tt.expectedName {
				t.Errorf("NewScanFilter(%v).Name = %q, want %q", tt.deviceType, result.Name, tt.expectedName)
			}
		})
	}
}

// Test ValidateCharacteristic
func Test_ValidateCharacteristic(t *testing.T) {
	tests := []struct {
		name           string
		characteristic *GATTCharacteristic
		requireNotify  bool
		expectError    bool
	}{
		{
			name: "valid notification characteristic with notify required",
			characteristic: &GATTCharacteristic{
				UUID:       VEDirectDataCharacteristic,
				Properties: PropNotify,
			},
			requireNotify: true,
			expectError:   false,
		},
		{
			name: "valid characteristic without notify when not required",
			characteristic: &GATTCharacteristic{
				UUID:       "some-uuid",
				Properties: PropRead | PropWrite,
			},
			requireNotify: false,
			expectError:   false,
		},
		{
			name: "invalid characteristic without notify when required",
			characteristic: &GATTCharacteristic{
				UUID:       "some-uuid",
				Properties: PropRead | PropWrite,
			},
			requireNotify: true,
			expectError:   true,
		},
		{
			name:           "nil characteristic returns error",
			characteristic: nil,
			requireNotify:  false,
			expectError:    true,
		},
		{
			name: "characteristic with indicate when notify required",
			characteristic: &GATTCharacteristic{
				UUID:       VEDirectDataCharacteristic,
				Properties: PropIndicate,
			},
			requireNotify: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCharacteristic(tt.characteristic, tt.requireNotify)
			if tt.expectError && err == nil {
				t.Errorf("ValidateCharacteristic() = nil, want error")
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateCharacteristic() = %v, want nil", err)
			}
		})
	}
}

// Test FindNotificationCharacteristic
func Test_FindNotificationCharacteristic(t *testing.T) {
	serviceWithNotify := &GATTService{
		UUID: "test-uuid",
		Characteristics: []*GATTCharacteristic{
			{UUID: "read-only", Properties: PropRead},
			{UUID: VEDirectDataCharacteristic, Properties: PropNotify},
			{UUID: "write-only", Properties: PropWrite},
		},
	}

	serviceWithoutNotify := &GATTService{
		UUID: "another-uuid",
		Characteristics: []*GATTCharacteristic{
			{UUID: "read-only", Properties: PropRead},
			{UUID: "write-only", Properties: PropWrite},
		},
	}

	tests := []struct {
		name        string
		service     *GATTService
		expectError bool
	}{
		{
			name:        "find notification characteristic in service",
			service:     serviceWithNotify,
			expectError: false,
		},
		{
			name:        "no notification characteristic returns error",
			service:     serviceWithoutNotify,
			expectError: true,
		},
		{
			name:        "nil service returns error",
			service:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindNotificationCharacteristic(tt.service)

			if tt.expectError {
				if err == nil {
					t.Errorf("FindNotificationCharacteristic() = %v, nil, want error", result)
				}
			} else {
				if err != nil {
					t.Errorf("FindNotificationCharacteristic() = nil, %v, want characteristic, nil", err)
				} else if result == nil {
					t.Errorf("FindNotificationCharacteristic() = nil, error, want characteristic, nil")
				} else if !(result.Properties&(PropNotify|PropIndicate) != 0) {
					t.Errorf("FindNotificationCharacteristic().Properties does not support notifications")
				}
			}
		})
	}
}
