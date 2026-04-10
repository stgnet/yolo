//go:build darwin

// macOS BLE backend initialization
// This init function enables the CoreBluetooth implementation from victron/macos
package victron

import (
	"github.com/scottstg/yolo/victron/macos"
)

func init() {
	SetBackendCreator(func() (BLEBackend, error) {
		return macos.New()
	})
}
