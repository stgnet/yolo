//go:build darwin

package macos

import (
	"strings"
	"testing"
	"time"

	"github.com/scottstg/yolo/victron/ble"
)

func TestNewBackend(t *testing.T) {
	b, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if b == nil {
		t.Fatal("New() returned nil")
	}
	if b.initialized {
		t.Error("New() should return uninitialized backend")
	}
}

func TestBackendInitialize(t *testing.T) {
	tests := []struct {
		name         string
		preInitState bool
		expectError  bool
	}{
		{
			name:         "first initialization",
			preInitState: false,
			expectError:  false,
		},
		{
			name:         "already initialized",
			preInitState: true,
			expectError:  false, // Should not error on re-initialization
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Backend{initialized: tt.preInitState}
			err := b.Initialize()
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Initialize() returned unexpected error: %v", err)
			}
			if !tt.preInitState {
				// Should be initialized after first call
				if !b.initialized {
					t.Error("Initialize() should set initialized to true")
				}
			}
		})
	}
}

func TestBackendClose(t *testing.T) {
	tests := []struct {
		name          string
		preInitState  bool
		expectError   bool
		expectedState bool // Expected initialized state after Close()
	}{
		{
			name:          "close initialized backend",
			preInitState:  true,
			expectError:   false,
			expectedState: false,
		},
		{
			name:          "close uninitialized backend",
			preInitState:  false,
			expectError:   false,
			expectedState: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Backend{initialized: tt.preInitState}
			err := b.Close()
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Close() returned unexpected error: %v", err)
			}
			if b.initialized != tt.expectedState {
				t.Errorf("Close() set initialized to %v, expected %v", b.initialized, tt.expectedState)
			}
		})
	}
}

// Test for filtering Victron devices from scan results
func TestFilterVictronDevices(t *testing.T) {
	tests := []struct {
		name     string
		input    []ble.Device
		expected int
	}{
		{
			name: "all Victron devices",
			input: []ble.Device{
				{Name: "GLOW 50/15", Address: "AA:BB:CC:DD:EE:FF"},
				{Name: "SmartShunt 500A", Address: "11:22:33:44:55:66"},
			},
			expected: 2,
		},
		{
			name: "mixed devices",
			input: []ble.Device{
				{Name: "Apple Watch", Address: "AA:BB:CC:DD:EE:FF"},
				{Name: "SmartSolar MPPT", Address: "11:22:33:44:55:66"},
				{Name: "JBL Speaker", Address: "77:88:99:AA:BB:CC"},
			},
			expected: 1,
		},
		{
			name:     "no Victron devices",
			input:    []ble.Device{{Name: "Apple Watch", Address: "AA:BB:CC:DD:EE:FF"}},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterVictronDevices(tt.input)
			if len(result) != tt.expected {
				t.Errorf("filterVictronDevices() returned %d devices, expected %d", len(result), tt.expected)
			}
		})
	}
}

// Test isValidMAC for traditional MAC addresses
func TestIsValidMacTraditional(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "AA:BB:CC:DD:EE:FF", true},
		{"valid uppercase", "aa:bb:cc:dd:ee:ff", true},
		{"valid mixed case", "Aa:Bb:Cc:Dd:Ee:Ff", true},
		{"invalid length short", "AA:BB:CC:DD:EE", false},
		{"invalid length long", "AA:BB:CC:DD:EE:FF:GG", false},
		{"invalid chars", "GH:IJ:KL:MN:OP:QR", false},
		{"missing colon", "AABB:CCDD:EEFF", false},
		{"extra colons", "AA:BB:CC:DD:EE:FF:", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidMAC(tt.input)
			if result != tt.expected {
				t.Errorf("isValidMAC(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test isValidMAC for UUID format (macOS CoreBluetooth)
func TestIsValidMacUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid UUID", "12345678-ABCD-EF01-2345-6789ABCDEF01", true},
		{"valid lowercase UUID", "abcd1234-5678-9abc-def0-123456789abc", true},
		{"invalid UUID length short", "12345678-ABCD-EF01-2345-6789ABCDEF", false},
		{"invalid UUID missing dash", "12345678ABCD-EF01-2345-6789ABCDEF01", false},
		{"invalid UUID extra dash", "12-345678-ABCD-EF01-2345-6789ABCDEF01", false},
		{"invalid chars in UUID", "12345678-GHIJ-KLMN-OPQR-STUVWXYZ123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidMAC(tt.input)
			if result != tt.expected {
				t.Errorf("isValidMAC(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test isVictronDeviceName for Victron keyword detection
func TestIsVictronDeviceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"GLOW device", "GLOW 50/15", true},
		{"SmartShunt", "SmartShunt 500A", true},
		{"SmartSolar MPPT", "SmartSolar MPPT 100/20", true},
		{"VE.Direct adapter", "VE.Direct Adapter", true},
		{"Victron brand", "Victron Energy", true},
		{"lowercase glow", "glow charger", true},
		{"mixed case mppt", "mppt solar controller", true},
		{"non-Victron device", "Apple Watch", false},
		{"empty name", "", false},
		{"random device", "JBL Speaker", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVictronDeviceName(tt.input)
			if result != tt.expected {
				t.Errorf("isVictronDeviceName(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test parseJSONOutput for JSON parsing
func TestParseJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // expected number of devices
	}{
		{
			name: "single device with name",
			input: `{
				"devices": [
					{"address": "AA:BB:CC:DD:EE:FF", "name": "GLOW 50/15"}
				]
			}`,
			expected: 1,
		},
		{
			name: "multiple devices",
			input: `{
				"devices": [
					{"address": "AA:BB:CC:DD:EE:FF", "name": "GLOW 50/15"},
					{"address": "11:22:33:44:55:66", "name": "SmartShunt"}
				]
			}`,
			expected: 2,
		},
		{
			name: "device without name",
			input: `{
				"devices": [
					{"address": "AA:BB:CC:DD:EE:FF"}
				]
			}`,
			expected: 1,
		},
		{
			name:     "empty devices array",
			input:    `{"devices": []}`,
			expected: 0,
		},
		{
			name:     "malformed JSON",
			input:    `{invalid json}`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJSONOutput(tt.input)
			if len(result) != tt.expected {
				t.Errorf("parseJSONOutput() returned %d devices, expected %d", len(result), tt.expected)
			}
			if tt.expected > 0 && len(result) == 0 {
				t.Logf("Input was: %s", tt.input)
			}
		})
	}
}

// Test parseScanOutput for both JSON and legacy formats
func TestParseScanOutput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		minCount  int // minimum expected devices (0 is OK if no error)
	}{
		{
			name: "JSON format with Victron device",
			input: `{
				"devices": [
					{"address": "AA:BB:CC:DD:EE:FF", "name": "GLOW 50/15"}
				]
			}`,
			expectErr: false,
			minCount:  1,
		},
		{
			name: "legacy pipe-delimited format",
			input: `Scanning for devices...
AA:BB:CC:DD:EE:FF|GLOW 50/15
11:22:33:44:55:66|SmartShunt`,
			expectErr: false,
			minCount:  2,
		},
		{
			name: "mixed with error lines",
			input: `Scanning for devices...
Error: connection failed
AA:BB:CC:DD:EE:FF|GLOW 50/15
Traceback (most recent call last):
11:22:33:44:55:66|SmartShunt`,
			expectErr: false,
			minCount:  2,
		},
		{
			name:      "invalid MAC address",
			input:     `INVALID_MAC|SomeDevice`,
			expectErr: true,
			minCount:  0,
		},
		{
			name:      "empty output",
			input:     "",
			expectErr: true,
			minCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseScanOutput(tt.input)
			if tt.expectErr && err == nil {
				t.Error("parseScanOutput() expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("parseScanOutput() returned unexpected error: %v", err)
			}
			if !tt.expectErr && len(result) < tt.minCount {
				t.Errorf("parseScanOutput() returned %d devices, expected at least %d", len(result), tt.minCount)
			}
		})
	}
}

// Test ScanForDevices error handling when not initialized
func TestScanForDevicesNotInitialized(t *testing.T) {
	b := &Backend{initialized: false}
	ctx := t.Context()
	_, err := b.ScanForDevices(ctx, 5*time.Second)
	if err == nil {
		t.Error("ScanForDevices() should return error when backend not initialized")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("Error message should mention 'not initialized', got: %v", err)
	}
}

// Test Connect returns not implemented error
func TestConnectNotImplemented(t *testing.T) {
	b := &Backend{initialized: true}
	_, err := b.Connect("AA:BB:CC:DD:EE:FF")
	if err == nil {
		t.Error("Connect() should return error")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("Error message should mention 'not yet implemented', got: %v", err)
	}
}

// Test DiscoverServices returns not implemented error
func TestDiscoverServicesNotImplemented(t *testing.T) {
	b := &Backend{initialized: true}
	_, err := b.DiscoverServices(nil) // nil connection is fine for this test
	if err == nil {
		t.Error("DiscoverServices() should return error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("Error message should mention 'not implemented', got: %v", err)
	}
}

// Test DiscoverCharacteristics returns not implemented error
func TestDiscoverCharacteristicsNotImplemented(t *testing.T) {
	b := &Backend{initialized: true}
	_, err := b.DiscoverCharacteristics(ble.Service{}) // can use zero value for struct
	if err == nil {
		t.Error("DiscoverCharacteristics() should return error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("Error message should mention 'not implemented', got: %v", err)
	}
}

// mockAdvertisement implements ble.Advertisement for testing (placeholder)
type mockAdvertisement struct {
	name string
	addr string
}

func (m *mockAdvertisement) LocalName() string {
	return m.name
}

// Other required methods for ble.Advertisement interface (stubs)
func (m *mockAdvertisement) ManufacturerData() []byte         { return nil }
func (m *mockAdvertisement) ServiceData() map[interface{}][]byte { return nil }
func (m *mockAdvertisement) Services() []interface{}          { return nil }
func (m *mockAdvertisement) OverflowService() []interface{}   { return nil }
func (m *mockAdvertisement) TxPowerLevel() int                { return 0 }
func (m *mockAdvertisement) Connectable() bool                { return true }
func (m *mockAdvertisement) SolicitedService() []interface{}  { return nil }
func (m *mockAdvertisement) RSSI() int                        { return -70 }
