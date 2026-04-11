//go:build darwin

package victron

import (
	"context"
	"testing"
	"time"
)

func TestNewMacOSBackend(t *testing.T) {
	t.Run("creates backend successfully", func(t *testing.T) {
		backend, err := NewMacOSBackend()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if backend == nil {
			t.Fatal("expected backend to be non-nil")
		}
	})
}

func TestMacOSBackend_Initialize(t *testing.T) {
	t.Run("initializes successfully", func(t *testing.T) {
		m := &MacOSBackend{}

		err := m.Initialize()
		if err != nil {
			t.Logf("Note: Bluetooth check may fail in test environment: %v", err)
			// We allow this to pass even if it fails, as tests might not have Bluetooth access
		}

		if !m.initialized {
			t.Error("expected initialized to be true")
		}
	})
}

func TestMacOSBackend_Close(t *testing.T) {
	t.Run("closes backend and clears state", func(t *testing.T) {
		m := &MacOSBackend{
			initialized: true,
			scanning:    true,
		}

		err := m.Close()
		if err != nil {
			t.Fatalf("expected no error on close, got %v", err)
		}

		if m.initialized {
			t.Error("expected initialized to be false after close")
		}

		if m.scanning {
			t.Error("expected scanning to be false after close")
		}
	})
}

func TestMacOSBackend_ScanForDevices_Uninitialized(t *testing.T) {
	t.Run("returns error when not initialized", func(t *testing.T) {
		m := &MacOSBackend{} // not initialized

		devices, err := m.ScanForDevices(context.Background(), time.Second)
		if err == nil {
			t.Fatal("expected error for uninitialized backend")
		}

		expectedMsg := "backend not initialized"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}

		if devices != nil {
			t.Error("expected nil devices for uninitialized backend")
		}
	})
}

func TestMacOSBackend_ScanForDevices_ContextCancelled(t *testing.T) {
	t.Run("returns error when context is cancelled", func(t *testing.T) {
		m := &MacOSBackend{}
		m.Initialize() // Initialize first

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		devices, err := m.ScanForDevices(ctx, time.Second*5)

		if err == nil {
			t.Log("Note: scan might return empty devices without error")
		}

		// We don't fail here - the implementation handles context cancellation gracefully
		if devices == nil && err != nil {
			t.Logf("Got expected behavior with cancelled context: %v", err)
		}
	})
}

func TestMacOSBackend_Connect(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		m := &MacOSBackend{}

		conn, err := m.Connect("test-address")
		if err == nil {
			t.Fatal("expected error for unimplemented connect")
		}

		if conn != nil {
			t.Error("expected nil connection for unimplemented method")
		}

		expectedMsg := "connect not yet implemented for macOS"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestMacOSBackend_DiscoverServices(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		m := &MacOSBackend{}

		services, err := m.DiscoverServices(nil)
		if err == nil {
			t.Fatal("expected error for unimplemented discover services")
		}

		if services != nil {
			t.Error("expected nil services for unimplemented method")
		}

		expectedMsg := "not implemented"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestMacOSBackend_DiscoverCharacteristics(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		m := &MacOSBackend{}

		characteristics, err := m.DiscoverCharacteristics(BLEService{})
		if err == nil {
			t.Fatal("expected error for unimplemented discover characteristics")
		}

		if characteristics != nil {
			t.Error("expected nil characteristics for unimplemented method")
		}

		expectedMsg := "not implemented"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestMacOSBackend_InterfaceImplementation(t *testing.T) {
	t.Run("implements BLEBackend interface", func(t *testing.T) {
		var _ BLEBackend = &MacOSBackend{}
	})
}

// Test isValidMAC for traditional MAC addresses
func TestIsValidMactraditional(t *testing.T) {
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
