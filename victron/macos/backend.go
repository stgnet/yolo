//go:build darwin

// Package macos provides a BLE backend implementation for macOS.
//
// CURRENT STATUS: Not yet implemented for macOS
// --------------------------------------------------------
// This backend is currently a stub placeholder. To enable Victron device
// connectivity on macOS, you need to implement CoreBluetooth support.
//
// IMPLEMENTATION REQUIREMENTS:
// =============================
// 1. Use a Go BLE library that supports macOS (e.g., go-ble with CoreBluetooth)
// 2. Handle Bluetooth permissions in app bundle Info.plist
// 3. Implement GATT service/characteristic discovery for Victron devices
// 4. Support VE.Direct protocol data parsing
// 5. Test with actual Victron SmartSolar, SmartShunt, or VE.Direct adapters
//
// REFERENCE: See victron/bluez/backend.go for a complete Linux implementation
// using the BlueZ stack that can serve as a reference for the API surface.
//
// KNOWN LIMITATIONS:
// ==================
// - ScanForDevices returns empty results (no actual BLE scanning)
// - Connect, DiscoverServices, DiscoverCharacteristics return "not implemented"
// - GATT operations (ReadValue, WriteValue, Subscribe) not supported
// - Real-time VE.Direct data streaming unavailable
//
// FOR DEVELOPERS: To contribute macOS support, implement the missing methods
// using Apple's CoreBluetooth framework via CGO bindings or Objective-C bridge.
package macos

import (
	"context"
	"fmt"
	"time"

	"github.com/scottstg/yolo/victron"
)

// Backend implements victron.BLEBackend for macOS as a stub placeholder.
type Backend struct {
	initialized bool
}

// New creates a new BLE backend. Returns error indicating macOS support not implemented.
func New() (*Backend, error) {
	return nil, fmt.Errorf("macOS BLE support not implemented - see victron/macos/backend.go for implementation notes")
}

// Initialize prepares the backend for use. Not implemented.
func (b *Backend) Initialize() error {
	return fmt.Errorf("macOS BLE not implemented")
}

// Close shuts down the backend. No-op on stub.
func (b *Backend) Close() error {
	return nil
}

// ScanForDevices scans for nearby BLE devices. Returns empty list on macOS.
func (b *Backend) ScanForDevices(ctx context.Context, duration time.Duration) ([]victron.BLEDevice, error) {
	fmt.Println("[INFO] macOS BLE scanning not implemented")
	fmt.Println("[INFO] To enable: implement CoreBluetooth support as documented in backend.go header")

	// Return empty list - real implementation would scan for devices
	return []victron.BLEDevice{}, fmt.Errorf("macOS BLE scanning not implemented")
}

// Connect establishes a connection to a device. Not implemented on macOS.
func (b *Backend) Connect(address string) (victron.BLEConnection, error) {
	return nil, fmt.Errorf("macOS BLE not implemented")
}

// DiscoverServices discovers GATT services on a connected device. Not implemented.
func (b *Backend) DiscoverServices(conn victron.BLEConnection) ([]victron.BLEService, error) {
	return []victron.BLEService{}, fmt.Errorf("macOS BLE not implemented")
}

// DiscoverCharacteristics discovers characteristics in a service. Not implemented.
func (b *Backend) DiscoverCharacteristics(service victron.BLEService) ([]victron.BLECharacteristic, error) {
	return []victron.BLECharacteristic{}, fmt.Errorf("macOS BLE not implemented")
}
