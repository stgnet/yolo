//go:build darwin

// Platform-specific initialization for macOS BLE backend.
package macos

import (
	"github.com/scottstg/yolo/victron"
)

func init() {
	// Register the macOS backend creator
	victron.SetBackendCreator(func() (victron.BLEBackend, error) {
		return New()
	})
}
