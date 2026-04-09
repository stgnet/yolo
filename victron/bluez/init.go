//go:build linux && cgo

// Platform-specific initialization for Linux BLE backend.
package bluez

import (
	"github.com/scottstg/yolo/victron"
)

func init() {
	// Register the BlueZ backend creator for Linux
	victron.SetBackendCreator(func() (victron.BLEBackend, error) {
		return New()
	})
}
