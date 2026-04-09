package victron

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDeviceTypeString tests the String() method of DeviceType
func TestDeviceTypeString(t *testing.T) {
	tests := []struct {
		name     string
		deviceType DeviceType
		expected string
	}{
		{name: "SmartSolar", deviceType: DeviceTypeSmartSolar, expected: "SmartSolar"},
		{name: "SmartShunt", deviceType: DeviceTypeSmartShunt, expected: "SmartShunt"},
		{name: "VEDirectAdapter", deviceType: DeviceTypeVEDirectAdapter, expected: "VE.Direct Adapter"},
		{name: "PhoenixInverter", deviceType: DeviceTypePhoenixInverter, expected: "Phoenix Inverter"},
		{name: "MultiPlus", deviceType: DeviceTypeMultiPlus, expected: "MultiPlus"},
		{name: "Unknown", deviceType: DeviceTypeUnknown, expected: "Unknown"},
		{name: "Invalid", deviceType: DeviceType(999), expected: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.deviceType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConnectionError_Error tests ConnectionError.Error()
func TestConnectionError_Error(t *testing.T) {
	err := &ConnectionError{
		Address: "AA:BB:CC:DD:EE:FF",
		Err:     nil,
	}
	result := err.Error()
	assert.Contains(t, result, "connection error to AA:BB:CC:DD:EE:FF")
}

// TestNotFoundError_Error tests NotFoundError.Error()
func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{Query: "test_query"}
	result := err.Error()
	assert.Contains(t, result, "no device found matching: test_query")
}

// TestTimeoutError_Error tests TimeoutError.Error()
func TestTimeoutError_Error(t *testing.T) {
	duration := 30 * time.Second
	err := &TimeoutError{Operation: "scan", Duration: duration}
	result := err.Error()
	assert.Contains(t, result, "scan timed out after")
	assert.Contains(t, result, "30s")
}

// TestParseError_Error tests ParseError.Error()
func TestParseError_Error(t *testing.T) {
	err := &ParseError{Field: "V", Raw: "invalid", Err: nil}
	result := err.Error()
	assert.Contains(t, result, "parse error for field V")
	assert.Contains(t, result, "value: invalid")
}

// TestValue struct creation and access
func TestValue(t *testing.T) {
	now := time.Now()
	v := Value{
		Key:        "V",
		RawValue:   "12.34",
		FloatValue: 12.34,
		Timestamp:  now,
	}

	assert.Equal(t, "V", v.Key)
	assert.Equal(t, "12.34", v.RawValue)
	assert.Equal(t, 12.34, v.FloatValue)
	assert.NotZero(t, v.Timestamp)
}

// TestDevice struct creation and access
func TestDevice(t *testing.T) {
	d := &Device{
		Address:      "AA:BB:CC:DD:EE:FF",
		Name:         "Test Device",
		Type:         DeviceTypeSmartSolar,
		values:       make(map[string]Value),
		valueChan:    make(chan Value, 10),
		valueUpdates: make(chan map[string]Value, 10),
		stopChan:     make(chan struct{}),
	}

	assert.Equal(t, "AA:BB:CC:DD:EE:FF", d.Address)
	assert.Equal(t, "Test Device", d.Name)
	assert.Equal(t, DeviceTypeSmartSolar, d.Type)
	assert.False(t, d.connected)
	assert.NotNil(t, d.values)
}

// TestDeviceInfo struct
func TestDeviceInfo(t *testing.T) {
	info := &DeviceInfo{
		Address:        "AA:BB:CC:DD:EE:FF",
		Name:           "Test Device",
		Type:           DeviceTypeSmartSolar,
		SignalStrength: -50,
	}

	assert.Equal(t, "AA:BB:CC:DD:EE:FF", info.Address)
	assert.Equal(t, "Test Device", info.Name)
	assert.Equal(t, DeviceTypeSmartSolar, info.Type)
	assert.Equal(t, -50, info.SignalStrength)
}

// Test constants are exported and usable
func TestConstants(t *testing.T) {
	// Just verify the constants are defined and have expected values
	assert.Equal(t, "V", KeySystemVoltage)
	assert.Equal(t, "A", KeySystemCurrent)
	assert.Equal(t, "W", KeySystemPower)
	assert.Equal(t, "P.V", KeyPVInputVoltage1)
	assert.Equal(t, "SoC", KeyStateOfCharge)
	assert.Equal(t, "T", KeyTemperature)
	assert.Equal(t, "/Dev", KeyDeviceVersion)
	assert.Equal(t, "/Sn", KeySerialNumber)
}
