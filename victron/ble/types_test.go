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
