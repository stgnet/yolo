// Package ble provides shared BLE interfaces and types for Victron device communication.
package ble

import (
	"testing"
)

func TestDevice_DetectAsVictron(t *testing.T) {
	tests := []struct {
		name             string
		deviceName       string
		serviceUUIDs     []string
		manufacturerData map[uint16][]byte
		expected         bool
	}{
		{
			name:         "Device with Victron in name",
			deviceName:   "Victron Energy",
			serviceUUIDs: nil,
			expected:     true,
		},
		{
			name:         "Glow device",
			deviceName:   "glow",
			serviceUUIDs: nil,
			expected:     true,
		},
		{
			name:         "SmartSolar device",
			deviceName:   "SmartSolar MPPT 100/30",
			serviceUUIDs: nil,
			expected:     true,
		},
		{
			name:         "Non-Victron device",
			deviceName:   "Generic Device",
			serviceUUIDs: nil,
			expected:     false,
		},
		{
			name:             "Device with Victron manufacturer data",
			deviceName:       "Unknown Device",
			serviceUUIDs:     []string{"00000100-98e4-11ea-9a03-0242ac120002"},
			manufacturerData: map[uint16][]byte{0x025A: {0x01}},
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := Device{
				Name:             tt.deviceName,
				ServiceUUIDs:     tt.serviceUUIDs,
				ManufacturerData: tt.manufacturerData,
			}

			if got := device.DetectAsVictron(); got != tt.expected {
				t.Errorf("Device.DetectAsVictron() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestService_String(t *testing.T) {
	service := Service{
		UUID:        "00000100-98e4-11ea-9a03-0242ac120002",
		StartHandle: 0x0010,
		EndHandle:   0x0020,
		Primary:     true,
	}

	expected := "Service{UUID: 00000100-98e4-11ea-9a03-0242ac120002, Handles: 0x0010-0x0020, Primary: true}"
	if got := service.String(); got != expected {
		t.Errorf("Service.String() = %q, want %q", got, expected)
	}
}

func TestCharacteristic_String(t *testing.T) {
	characteristic := Characteristic{
		UUID:        "00002a00-98e4-11ea-9a03-0242ac120002",
		Handle:      0x0015,
		ValueHandle: 0x0016,
		Properties:  0x2A, // Read + Write + Notify
		Description: "Sensor Data",
	}

	expected := "Characteristic{UUID: 00002a00-98e4-11ea-9a03-0242ac120002, Handle: 0x0015, ValueHandle: 0x0016, Properties: 0x2A, Description: Sensor Data}"
	if got := characteristic.String(); got != expected {
		t.Errorf("Characteristic.String() = %q, want %q", got, expected)
	}
}

func TestStringToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HELLO", "hello"},
		{"Hello World", "hello world"},
		{"VICTRON ENERGY", "victron energy"},
		{"Glow123", "glow123"},
		{"", ""},
		{"AlreadyLower", "alreadylower"},
		{"Mixed123CASE", "mixed123case"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := stringToLower(tt.input); got != tt.expected {
				t.Errorf("stringToLower(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStringToASCII(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"Test String", "Test String"},
		{"", ""},
		{"12345", "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := stringToASCII(tt.input); got != tt.expected {
				t.Errorf("stringToASCII(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "lo wo", true},
		{"hello world", "", true}, // empty substring always matches
		{"hello world", "xyz", false},
		{"", "", true},            // both empty
		{"", "a", false},          // searching in empty string
		{"test", "TEST", false},   // case sensitive
		{"aaa", "aa", true},       // overlapping match
		{"abc", "abcd", false},    // substring longer than string
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			if got := contains(tt.s, tt.substr); got != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.expected)
			}
		})
	}
}

func TestDevice_DetectAsVictron_Comprehensive(t *testing.T) {
	tests := []struct {
		name             string
		deviceName       string
		serviceUUIDs     []string
		manufacturerData map[uint16][]byte
		expected         bool
	}{
		// Victron keywords (case insensitive)
		{
			name:         "victron in name (uppercase)",
			deviceName:   "VICTRON",
			serviceUUIDs: nil,
			expected:     true,
		},
		{
			name:         "Victron mixed case",
			deviceName:   "ViCtRoN Device",
			serviceUUIDs: nil,
			expected:     true,
		},
		{
			name:         "smartshunt device",
			deviceName:   "SmartShunt Battery Monitor",
			serviceUUIDs: nil,
			expected:     true,
		},
		{
			name:         "cerbo device",
			deviceName:   "Cerbo GX",
			serviceUUIDs: nil,
			expected:     true,
		},
		{
			name:         "venus device",
			deviceName:   "Venus OS",
			serviceUUIDs: nil,
			expected:     true,
		},
		// Edge cases
		{
			name:         "Empty device name with services",
			deviceName:   "",
			serviceUUIDs: []string{"some-uuid"},
			expected:     true, // Has service UUIDs
		},
		{
			name:             "Empty device name with manufacturer data",
			deviceName:       "",
			serviceUUIDs:     nil,
			manufacturerData: map[uint16][]byte{0x1234: {0x01}},
			expected:         true, // Has manufacturer data
		},
		{
			name:             "Empty device with nothing",
			deviceName:       "",
			serviceUUIDs:     nil,
			manufacturerData: nil,
			expected:         false,
		},
		// Non-Victron devices
		{
			name:         "Generic bluetooth device",
			deviceName:   "My Bluetooth Speaker",
			serviceUUIDs: nil,
			expected:     false,
		},
		{
			name:         "Device with victron as substring in non-keyword context should still match",
			deviceName:   "notvictronish",
			serviceUUIDs: nil,
			expected:     true, // "victron" is contained even if not a word boundary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := Device{
				Name:             tt.deviceName,
				ServiceUUIDs:     tt.serviceUUIDs,
				ManufacturerData: tt.manufacturerData,
			}

			if got := device.DetectAsVictron(); got != tt.expected {
				t.Errorf("Device.DetectAsVictron() = %v, want %v for device: %q", got, tt.expected, tt.deviceName)
			}
		})
	}
}
