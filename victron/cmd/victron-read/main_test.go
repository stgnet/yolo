package main

import (
	"testing"

	"github.com/scottstg/yolo/victron"
)

func TestFilterVictronDevices(t *testing.T) {
	devices := []victron.DiscoverableDevice{
		{Name: "SmartSolar", Address: "AA:BB:CC:DD:EE:01", IsVictron: true, RSSI: -50},
		{Name: "Unknown", Address: "AA:BB:CC:DD:EE:02", IsVictron: false, RSSI: -60},
		{Name: "SmartShunt", Address: "AA:BB:CC:DD:EE:03", IsVictron: true, RSSI: -55},
		{Name: "Other", Address: "AA:BB:CC:DD:EE:04", IsVictron: false, RSSI: -70},
	}

	filtered := filterVictronDevices(devices)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 Victron devices, got %d", len(filtered))
	}

	expectedNames := map[string]bool{"SmartSolar": true, "SmartShunt": true}
	for _, d := range filtered {
		if !expectedNames[d.Name] {
			t.Errorf("Unexpected device: %s", d.Name)
		}
	}
}

func TestFilterVictronDevices_Empty(t *testing.T) {
	devices := []victron.DiscoverableDevice{}
	filtered := filterVictronDevices(devices)
	if len(filtered) != 0 {
		t.Errorf("Expected 0 devices, got %d", len(filtered))
	}
}

func TestFilterVictronDevices_AllNonVictron(t *testing.T) {
	devices := []victron.DiscoverableDevice{
		{Name: "Unknown", Address: "AA:BB:CC:DD:EE:02", IsVictron: false, RSSI: -60},
	}
	filtered := filterVictronDevices(devices)
	if len(filtered) != 0 {
		t.Errorf("Expected 0 devices, got %d", len(filtered))
	}
}

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
		{"unknown_key", ""},
	}

	for _, tt := range tests {
		result := getUnit(tt.key)
		if result != tt.expected {
			t.Errorf("getUnit(%s) = %q, expected %q", tt.key, result, tt.expected)
		}
	}
}

func TestGetValue(t *testing.T) {
	values := map[string]victron.Value{
		victron.KeySystemVoltage: {RawValue: "12.5", FloatValue: 12.5},
		victron.KeySystemCurrent: {RawValue: "2.3", FloatValue: 2.3},
	}

	// Test existing key
	result := getValue(values, victron.KeySystemVoltage)
	if result.RawValue != "12.5" {
		t.Errorf("Expected RawValue 12.5, got %s", result.RawValue)
	}

	// Test non-existing key
	result = getValue(values, victron.KeyStateOfCharge)
	if result.RawValue != "" || result.FloatValue != 0 {
		t.Error("Expected empty value for non-existent key")
	}
}

func TestGetValue_EmptyMap(t *testing.T) {
	values := map[string]victron.Value{}
	result := getValue(values, victron.KeySystemVoltage)
	if result.RawValue != "" || result.FloatValue != 0 {
		t.Error("Expected empty value for query on empty map")
	}
}
