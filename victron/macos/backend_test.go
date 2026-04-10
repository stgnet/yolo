//go:build darwin

package macos

import (
	"testing"

	"github.com/go-ble/ble"
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

func TestIsVictronDevice(t *testing.T) {
	tests := []struct {
		name       string
		deviceName string
		expected   bool
	}{
		{
			name:     "GLOW device",
			deviceName: "GLOW 50/15",
			expected: true,
		},
		{
			name:     "SmartShunt device",
			deviceName: "SmartShunt 500A",
			expected: true,
		},
		{
			name:     "SmartSolar device",
			deviceName: "SmartSolar MPPT 150/35",
			expected: true,
		},
		{
			name:     "VE.Direct device",
			deviceName: "VE.Direct Adapter",
			expected: true,
		},
		{
			name:     "victron lowercase",
			deviceName: "victron energy",
			expected: true,
		},
		{
			name:     "VICTRON uppercase",
			deviceName: "VICTRON BATTERY",
			expected: true,
		},
		{
			name:     "Mixed case glow",
			deviceName: "My GLow Charger",
			expected: true,
		},
		{
			name:     "Empty device name",
			deviceName: "",
			expected: false,
		},
		{
			name:     "Non-Victron device",
			deviceName: "Apple Watch",
			expected: false,
		},
		{
			name:     "Bluetooth speaker",
			deviceName: "JBL Speaker",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock advertisement that returns the device name
			mockAdv := &mockAdvertisement{name: tt.deviceName}
			result := isVictronDevice(mockAdv)
			if result != tt.expected {
				t.Errorf("isVictronDevice(%q) = %v, expected %v", tt.deviceName, result, tt.expected)
			}
		})
	}
}

// mockAdvertisement implements ble.Advertisement for testing
type mockAdvertisement struct {
	name string
	addr string
}

func (m *mockAdvertisement) LocalName() string {
	return m.name
}

func (m *mockAdvertisement) Addr() ble.Addr {
	if m.addr == "" {
		m.addr = "00:00:00:00:00:00"
	}
	addr := ble.NewAddr(m.addr)
	return addr
}

// Other required methods for ble.Advertisement interface (stubs)
func (m *mockAdvertisement) ManufacturerData() []byte       { return nil }
func (m *mockAdvertisement) ServiceData() []ble.ServiceData { return nil }
func (m *mockAdvertisement) Services() []ble.UUID           { return nil }
func (m *mockAdvertisement) OverflowService() []ble.UUID    { return nil }
func (m *mockAdvertisement) TxPowerLevel() int              { return 0 }
func (m *mockAdvertisement) Connectable() bool              { return true }
func (m *mockAdvertisement) SolicitedService() []ble.UUID   { return nil }
func (m *mockAdvertisement) RSSI() int                      { return -70 }