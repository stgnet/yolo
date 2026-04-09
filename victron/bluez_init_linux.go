//go:build linux && cgo

// Package victron provides BLE client functionality for Victron devices.
package victron

import "github.com/scottstg/yolo/victron/bluez"

func init() {
	SetBackendCreator(func() (BLEBackend, error) {
		b, err := bluez.New()
		if err != nil {
			return nil, err
		}
		return b, nil
	})
}
