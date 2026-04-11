// Platform-specific BLE backend initialization for macOS
//go:build darwin
// +build darwin

package victron

import (
	"github.com/scottstg/yolo/victron/ble"
)

func init() {
	// Set the Darwin backend creator for macOS
	SetBackendCreator(func() (BLEBackend, error) {
		return ble.NewDarwinBackend()
	})
}
