// BLE backend wrapper for Victron device communication
// This package provides an abstraction over the ble shared types and platform-specific backends
package victron

import (
	"fmt"

	"github.com/scottstg/yolo/victron/ble"
)

// Type aliases to use shared ble types
type BLEBackend = ble.Backend
type BLEConnection = ble.Connection
type BLEDevice = ble.Device
type BLEService = ble.Service
type BLECharacteristic = ble.Characteristic

// DefaultBackend provides the default BLE backend implementation
var DefaultBackend BLEBackend = nil

// SetBackend sets the global BLE backend implementation
func SetBackend(backend BLEBackend) {
	DefaultBackend = backend
}

// GetBackend returns the current BLE backend
func GetBackend() BLEBackend {
	return DefaultBackend
}

// InitializeBackend creates and configures the appropriate backend for the current platform.
// Call this before using any Victron functionality.
func InitializeBackend() error {
	if DefaultBackend != nil {
		return nil // Already initialized
	}

	// Import happens inside function to avoid import cycle
	// This will be populated by platform-specific init functions
	if backendCreator == nil {
		return fmt.Errorf("no BLE backend available for this platform")
	}

	b, err := backendCreator()
	if err != nil {
		return fmt.Errorf("failed to create BLE backend: %w", err)
	}

	// Initialize it
	if err := b.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize BLE backend: %w", err)
	}

	DefaultBackend = b
	return nil
}

// Backend creator function - set by platform-specific init()
var backendCreator func() (BLEBackend, error)

// SetBackendCreator sets the backend creator function for this platform.
// This is used internally by platform-specific initialization code.
func SetBackendCreator(creator func() (BLEBackend, error)) {
	backendCreator = creator
}
