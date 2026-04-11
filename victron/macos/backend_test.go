//go:build darwin

package macos

import (
	"testing"

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
