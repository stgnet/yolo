//go:build darwin

// macOS BLE backend initialization
// 
// Status: Stub implementation - full BLE support requires CoreBluetooth integration
//
// To implement macOS BLE support, you need to:
// 1. Create CGO bindings or use a Go-to-CoreBluetooth bridge library
// 2. Handle Bluetooth permissions in the app bundle's Info.plist
// 3. Implement GATT service/characteristic discovery and connection management
// 4. Add notification subscription for real-time VE.Direct data streaming
//
// See victron/bluez/backend.go for a complete Linux implementation reference.
package victron

import (
	"errors"
)

func init() {
	SetBackendCreator(func() (BLEBackend, error) {
		return nil, errors.New("macOS BLE support not implemented - see victron/macos/backend.go for stub and implementation notes")
	})
}
