//go:build darwin

// macOS BLE backend initialization - currently a placeholder
// 
// NOTE: Full macOS BLE implementation exists in victron/macos/backend.go but cannot be 
// automatically integrated due to Go's import cycle restrictions. The macos package uses 
// types from the parent victron package, making it impossible for victron to import macos
// without creating a circular dependency.
//
// To enable macOS BLE support, one of these approaches is needed:
// 1. Move shared interface definitions (BLEBackend, BLEDevice, etc.) to a separate 'victron/ble' package
// 2. Use code generation to generate the init file at build time
// 3. Manually invoke macos backend from tools_victron.go with build constraints
//
// See victron/macos/backend.go for the complete CoreBluetooth implementation.
package victron

import (
	"errors"
)

func init() {
	SetBackendCreator(func() (BLEBackend, error) {
		return nil, errors.New("macOS BLE not auto-integrated due to import cycle - see victron/macos/backend.go for implementation")
	})
}
