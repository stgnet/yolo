// Package ble provides shared BLE interfaces and types for Victron device communication.
// This package exists separately to avoid import cycles between victron and platform-specific backends.
package ble

import (
	"context"
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

// Service represents a GATT service
type Service struct {
	UUID        string
	StartHandle uint16
	EndHandle   uint16
	Primary     bool
}

// Characteristic represents a GATT characteristic
type Characteristic struct {
	UUID        string
	Handle      uint16
	ValueHandle uint16
	Properties  uint8
	Description string
}
