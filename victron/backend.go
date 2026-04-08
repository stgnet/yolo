// BLE backend interface for Victron device communication
// This provides an abstraction over different BLE implementation libraries
package victron

import (
	"context"
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
	Address        string
	Name           string
	RSSI           int
	ServiceUUIDs   []string
	ManufacturerData map[uint16][]byte
}

// BLEService represents a GATT service
type BLEService struct {
	UUID          string
	StartHandle   uint16
	EndHandle     uint16
	Primary       bool
}

// BLECharacteristic represents a GATT characteristic  
type BLECharacteristic struct {
	UUID           string
	Handle         uint16
	ValueHandle    uint16
	Properties     uint8
	Description    string
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
