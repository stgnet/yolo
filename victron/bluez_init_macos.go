//go:build darwin

// macOS BLE backend initialization
package victron

import (
	"errors"
)

func init() {
	SetBackendCreator(func() (BLEBackend, error) {
		return nil, errors.New("macOS BLE support not yet implemented. See https://github.com/scottstg/yolo/issues/TODO")
	})
}
