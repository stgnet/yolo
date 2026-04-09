package victron

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeviceType_String(t *testing.T) {
	tests := []struct {
		name     string
		devType  DeviceType
		expected string
	}{
		{name: "SmartSolar", devType: DeviceTypeSmartSolar, expected: "SmartSolar"},
		{name: "SmartShunt", devType: DeviceTypeSmartShunt, expected: "SmartShunt"},
		{name: "VEDirectAdapter", devType: DeviceTypeVEDirectAdapter, expected: "VE.Direct Adapter"},
		{name: "PhoenixInverter", devType: DeviceTypePhoenixInverter, expected: "Phoenix Inverter"},
		{name: "MultiPlus", devType: DeviceTypeMultiPlus, expected: "MultiPlus"},
		{name: "Unknown", devType: DeviceTypeUnknown, expected: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.devType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeviceInfo(t *testing.T) {
	info := DeviceInfo{
		Address:        "AA:BB:CC:DD:EE:FF",
		Name:           "Victron SmartSolar",
		Type:           DeviceTypeSmartSolar,
		SignalStrength: -45,
	}

	assert.Equal(t, "AA:BB:CC:DD:EE:FF", info.Address)
	assert.Equal(t, "Victron SmartSolar", info.Name)
	assert.Equal(t, DeviceTypeSmartSolar, info.Type)
	assert.Equal(t, -45, info.SignalStrength)
}

func TestValue(t *testing.T) {
	now := time.Now()
	val := Value{
		Key:        "V",
		RawValue:   "12.34",
		FloatValue: 12.34,
		Timestamp:  now,
	}

	assert.Equal(t, "V", val.Key)
	assert.Equal(t, "12.34", val.RawValue)
	assert.Equal(t, 12.34, val.FloatValue)
	assert.WithinDuration(t, now, val.Timestamp, time.Second)
}

func TestDevice_Create(t *testing.T) {
	dev := &Device{
		Address:      "AA:BB:CC:DD:EE:FF",
		Name:         "Test Device",
		Type:         DeviceTypeSmartSolar,
		values:       make(map[string]Value),
		valueChan:    make(chan Value, 100),
		valueUpdates: make(chan map[string]Value, 100),
		stopChan:     make(chan struct{}),
	}

	assert.NotNil(t, dev)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", dev.Address)
	assert.Equal(t, "Test Device", dev.Name)
	assert.Equal(t, DeviceTypeSmartSolar, dev.Type)
	assert.False(t, dev.connected)
}

func TestConnectionError_Error(t *testing.T) {
	err := &ConnectionError{
		Address: "AA:BB:CC:DD:EE:FF",
		Err:     nil,
	}

	result := err.Error()
	assert.Contains(t, result, "connection error")
	assert.Contains(t, result, "AA:BB:CC:DD:EE:FF")
}

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{
		Query: "victron",
	}

	result := err.Error()
	assert.Contains(t, result, "no device found matching")
	assert.Contains(t, result, "victron")
}

func TestTimeoutError_Error(t *testing.T) {
	duration := 5 * time.Second
	err := &TimeoutError{
		Operation: "connect",
		Duration:  duration,
	}

	result := err.Error()
	assert.Contains(t, result, "timed out")
	assert.Contains(t, result, "connect")
}

func TestParseError_Error(t *testing.T) {
	err := &ParseError{
		Field: "voltage",
		Raw:   "invalid",
		Err:   nil,
	}

	result := err.Error()
	assert.Contains(t, result, "parse error")
	assert.Contains(t, result, "voltage")
	assert.Contains(t, result, "invalid")
}

func TestNewClient_ReturnsNonNil(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
}
