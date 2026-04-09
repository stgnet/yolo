// BLE backend interface for Victron device communication
// This provides an abstraction over different BLE implementation libraries
package victron

import (
	"context"
	"fmt"
	"time"
)

// BLEBackend defines the interface for Bluetooth Low Energy operations
type BLEBackend interface {
	// Initialize initializes the BLE adapter
	Initialize() error

	// Close shuts down the BLE connection
	Close() error

	// ScanForDevices scans for nearby BLE devices
	ScanForDevices(ctx context.Context, duration time.Duration) ([]BLEDevice, error)

	// Connect establishes a connection to a device
	Connect(address string) (BLEConnection, error)

	// DiscoverServices discovers GATT services on a connected device
	DiscoverServices(conn BLEConnection) ([]BLEService, error)

	// DiscoverCharacteristics discovers characteristics in a service
	DiscoverCharacteristics(service BLEService) ([]BLECharacteristic, error)
}

// BLEConnection represents an active BLE connection to a device
type BLEConnection interface {
	// Address returns the device address
	Address() string

	// UUID returns the device UUID if available
	UUID() string

	// IsConnected checks if connection is active
	IsConnected() bool

	// Close disconnects from the device
	Close() error

	// ReadValue reads a characteristic value
	ReadValue(characteristic BLECharacteristic) ([]byte, error)

	// WriteValue writes to a characteristic
	WriteValue(characteristic BLECharacteristic, data []byte) error

	// SubscribeToNotifications subscribes to notifications from a characteristic
	SubscribeToNotifications(characteristic BLECharacteristic) (<-chan []byte, error)

	// UnsubscribeFromNotifications stops receiving notifications
	UnsubscribeFromNotifications(characteristic BLECharacteristic) error
}

// BLEDevice represents a discovered Bluetooth device
type BLEDevice struct {
	Address          string
	Name             string
	RSSI             int
	ServiceUUIDs     []string
	ManufacturerData map[uint16][]byte
	IsVictron        bool // Flag to identify Victron devices
}

// BLEService represents a GATT service
type BLEService struct {
	UUID        string
	StartHandle uint16
	EndHandle   uint16
	Primary     bool
}

// BLECharacteristic represents a GATT characteristic
type BLECharacteristic struct {
	UUID        string
	Handle      uint16
	ValueHandle uint16
	Properties  uint8
	Description string
}

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
