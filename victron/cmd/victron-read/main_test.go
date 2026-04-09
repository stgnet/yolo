package main

import (
	"testing"
	"time"

	"github.com/scottstg/yolo/victron"
)

// Test filterVictronDevices filters out non-Victron devices
func TestFilterVictronDevices(t *testing.T) {
	devices := []victron.DiscoverableDevice{
		{Name: "glow", Address: "AA:BB:CC:DD:EE:FF", IsVictron: true, RSSI: -50},
		{Name: "iPhone", Address: "11:22:33:44:55:66", IsVictron: false, RSSI: -60},
		{Name: "SmartSolar", Address: "77:88:99:AA:BB:CC", IsVictron: true, RSSI: -55},
	}

	filtered := filterVictronDevices(devices)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 Victron devices, got %d", len(filtered))
	}

	if filtered[0].Name != "glow" {
		t.Errorf("Expected first device to be 'glow', got '%s'", filtered[0].Name)
	}

	if filtered[1].Name != "SmartSolar" {
		t.Errorf("Expected second device to be 'SmartSolar', got '%s'", filtered[1].Name)
	}
}

// Test filterVictronDevices with empty input
func TestFilterVictronDevicesEmpty(t *testing.T) {
	devices := []victron.DiscoverableDevice{}

	filtered := filterVictronDevices(devices)

	if len(filtered) != 0 {
		t.Errorf("Expected 0 devices, got %d", len(filtered))
	}
}

// Test filterVictronDevices with no Victron devices
func TestFilterVictronDevicesNone(t *testing.T) {
	devices := []victron.DiscoverableDevice{
		{Name: "iPhone", Address: "11:22:33:44:55:66", IsVictron: false, RSSI: -60},
		{Name: "Watch", Address: "77:88:99:AA:BB:CC", IsVictron: false, RSSI: -65},
	}

	filtered := filterVictronDevices(devices)

	if len(filtered) != 0 {
		t.Errorf("Expected 0 Victron devices, got %d", len(filtered))
	}
}

// Test getUnit returns correct units for known keys
func TestGetUnit(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{victron.KeySystemVoltage, " V"},
		{victron.KeySystemCurrent, " A"},
		{victron.KeySystemPower, " W"},
		{victron.KeyPVInputVoltage1, " V"},
		{victron.KeyPVInputCurrent1, " A"},
		{victron.KeyPVInputPower1, " W"},
		{victron.KeyBatteryVoltage, " V"},
		{victron.KeyBatteryCurrent, " A"},
		{victron.KeyStateOfCharge, "%"},
	}

	for _, tt := range tests {
		result := getUnit(tt.key)
		if result != tt.expected {
			t.Errorf("getUnit(%s) = %q, want %q", tt.key, result, tt.expected)
		}
	}
}

// Test getUnit returns empty string for unknown keys
func TestGetUnitUnknown(t *testing.T) {
	result := getUnit("unknown_key")
	if result != "" {
		t.Errorf("getUnit(unknown_key) = %q, want ''", result)
	}
}

// Test getValue retrieves values from the map correctly
func TestGetValue(t *testing.T) {
	values := map[string]victron.Value{
		victron.KeySystemVoltage: {RawValue: "12.5V", FloatValue: 12.5, Timestamp: time.Now()},
	}

	result := getValue(values, victron.KeySystemVoltage)
	if result.RawValue != "12.5V" {
		t.Errorf("Expected RawValue '12.5V', got '%s'", result.RawValue)
	}
	if result.FloatValue != 12.5 {
		t.Errorf("Expected FloatValue 12.5, got %f", result.FloatValue)
	}
}

// Test getValue returns empty value for missing keys
func TestGetValueMissing(t *testing.T) {
	values := map[string]victron.Value{
		victron.KeySystemCurrent: {RawValue: "5.0A", FloatValue: 5.0, Timestamp: time.Now()},
	}
	result := getValue(values, victron.KeySystemVoltage)
	if result.RawValue != "" {
		t.Errorf("Expected empty RawValue for missing key, got '%s'", result.RawValue)
	}
	if result.FloatValue != 0.0 {
		t.Errorf("Expected FloatValue 0.0 for missing key, got %f", result.FloatValue)
	}
}

// Test getValue with empty map
func TestGetValueEmptyMap(t *testing.T) {
	values := map[string]victron.Value{}

	result := getValue(values, victron.KeySystemVoltage)
	if result.RawValue != "" {
		t.Errorf("Expected empty RawValue for empty map, got '%s'", result.RawValue)
	}
	if result.FloatValue != 0.0 {
		t.Errorf("Expected FloatValue 0.0 for empty map, got %f", result.FloatValue)
	}
}
