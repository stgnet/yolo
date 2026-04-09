//go:build darwin

// Platform-specific initialization for macOS BLE backend.
package macos

import (
	"github.com/scottstg/yolo/victron"
)

func init() {
	// Register the macOS backend creator automatically on macOS
	victron.SetBackendCreator(func() (victron.BLEBackend, error) {
		return New()
	})
}
