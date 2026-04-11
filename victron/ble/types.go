// Package ble provides shared BLE interfaces and types for Victron device communication.
// This package exists separately to avoid import cycles between victron and platform-specific backends.
package ble

import (
	"context"
	"fmt"
	"time"
)

// Backend defines the interface for Bluetooth Low Energy operations
type Backend interface {
	// Initialize initializes the BLE adapter
	Initialize() error

	// Close shuts down the BLE connection
	Close() error

	// ScanForDevices scans for nearby BLE devices
	ScanForDevices(ctx context.Context, duration time.Duration) ([]Device, error)

	// Connect establishes a connection to a device
	Connect(address string) (Connection, error)

	// DiscoverServices discovers GATT services on a connected device
	DiscoverServices(conn Connection) ([]Service, error)

	// DiscoverCharacteristics discovers characteristics in a service
	DiscoverCharacteristics(service Service) ([]Characteristic, error)
}

// Service represents a GATT service discovered on a device
type Service struct {
	UUID        string
	StartHandle uint16
	EndHandle   uint16
	Primary     bool
}

// Characteristic represents a GATT characteristic within a service
type Characteristic struct {
	UUID        string
	Handle      uint16
	ValueHandle uint16
	Properties  uint8
	Description string
	ServiceUUID string // Reference to parent service
}

// Connection represents an active BLE connection to a device
type Connection interface {
	// Address returns the device address
	Address() string

	// UUID returns the device UUID if available
	UUID() string

	// IsConnected checks if connection is active
	IsConnected() bool

	// Close disconnects from the device
	Close() error

	// ReadValue reads a characteristic value
	ReadValue(characteristic Characteristic) ([]byte, error)

	// WriteValue writes to a characteristic
	WriteValue(characteristic Characteristic, data []byte) error

	// SubscribeToNotifications subscribes to notifications from a characteristic
	SubscribeToNotifications(characteristic Characteristic) (<-chan []byte, error)

	// UnsubscribeFromNotifications stops receiving notifications
	UnsubscribeFromNotifications(characteristic Characteristic) error
}

// Device represents a discovered Bluetooth device
type Device struct {
	Address          string
	Name             string
	RSSI             int
	ServiceUUIDs     []string
	ManufacturerData map[uint16][]byte
	IsVictron        bool // Flag to identify Victron devices
}

// DetectAsVictron checks if the device appears to be a Victron product based on name or known service UUIDs
func (d Device) DetectAsVictron() bool {
	nameLower := stringToASCII(stringToLower(d.Name))
	return contains(nameLower, "victron") ||
		contains(nameLower, "glow") ||
		contains(nameLower, "smartsolar") ||
		contains(nameLower, "smartshunt") ||
		contains(nameLower, "cerbo") ||
		contains(nameLower, "venus") ||
		len(d.ServiceUUIDs) > 0 || // Has any service UUIDs
		hasVictronServiceUUIDs(d.ServiceUUIDs) ||
		len(d.ManufacturerData) > 0 // Has manufacturer data (typically indicates Victron)
}

// hasVictronServiceUUIDs checks if the device advertises known Victron service UUIDs
func hasVictronServiceUUIDs(uuids []string) bool {
	// Known Victron BLE service UUIDs (from VE.Direct protocol)
	victronServiceUUIDs := map[string]bool{
		"0000ff00-0000-1000-8000-00805f9b34fb": true, // Victron notification service
		"ffe0":                                  true, // Short form
	}
	
	for _, uuid := range uuids {
		uuidLower := stringToLower(uuid)
		if victronServiceUUIDs[uuidLower] || victronServiceUUIDs[stringToASCII(uuidLower)] {
			return true
		}
	}
	return false
}

// String returns a string representation of the service
func (s Service) String() string {
	return fmt.Sprintf("Service{UUID: %s, Handles: 0x%04X-0x%04X, Primary: %v}",
		s.UUID, s.StartHandle, s.EndHandle, s.Primary)
}

// String returns a string representation of the characteristic
func (c Characteristic) String() string {
	return fmt.Sprintf("Characteristic{UUID: %s, Handle: 0x%04X, ValueHandle: 0x%04X, Properties: 0x%02X, Description: %s}",
		c.UUID, c.Handle, c.ValueHandle, c.Properties, c.Description)
}

// Helper functions for string operations (to avoid import cycle issues in tests)
func stringToLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func stringToASCII(s string) string {
	return string([]byte(s))
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
