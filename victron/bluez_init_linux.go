//go:build linux && cgo

// Package victron provides BLE client functionality for Victron devices.
package victron

import (
	"context"
	"time"

	"github.com/scottstg/yolo/victron/bluez"
)

// bluezAdapter wraps the BlueZ backend to implement the BLEBackend interface
type bluezAdapter struct {
	backend *bluez.Backend
}

// NewBlueZBackendCompatible creates a new BlueZ backend for Linux systems
func NewBlueZBackendCompatible() BLEBackend {
	b, err := bluez.New()
	if err != nil {
		return nil
	}
	return &bluezAdapter{backend: b}
}

// Initialize initializes the BLE adapter (already done in bluez.New())
func (a *bluezAdapter) Initialize() error {
	return nil // BlueZ backend is initialized on creation
}

// Close shuts down the BLE connection
func (a *bluezAdapter) Close() error {
	return nil // TODO: implement cleanup if needed
}

// ScanForDevices scans for nearby BLE devices
func (a *bluezAdapter) ScanForDevices(ctx context.Context, duration time.Duration) ([]BLEDevice, error) {
	// Convert timeout to milliseconds for bluez.Scan
	timeoutMs := int64(duration.Milliseconds())
	
	// Call BlueZ scan
	devices, err := a.backend.Scan(timeoutMs)
	if err != nil {
		return nil, err
	}
	
	// Convert to our BLEDevice format
	result := make([]BLEDevice, len(devices))
	for i, d := range devices {
		result[i] = BLEDevice{
			Address:   d.Address,
			Name:      d.Name,
			RSSI:      d.RSSI,
		}
	}
	return result, nil
}

// Connect establishes a connection to a device
func (a *bluezAdapter) Connect(address string) (BLEConnection, error) {
	// This needs more implementation to bridge bluez.Backend.Connect
	return nil, nil // TODO: implement proper connection
}

// DiscoverServices discovers GATT services on a connected device
func (a *bluezAdapter) DiscoverServices(conn BLEConnection) ([]BLEService, error) {
	return nil, nil // TODO: implement
}

// DiscoverCharacteristics discovers characteristics in a service
func (a *bluezAdapter) DiscoverCharacteristics(service BLEService) ([]BLECharacteristic, error) {
	return nil, nil // TODO: implement
}
