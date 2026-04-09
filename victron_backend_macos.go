//go:build darwin

// macOS-specific Victron BLE initialization
// This bypasses the circular import issue by registering the backend from main package

package main

import (
	"github.com/scottstg/yolo/victron"
	"github.com/scottstg/yolo/victron/macos"
)

func init() {
	// Register macOS BLE backend before any victron tool usage
	victron.SetBackendCreator(func() (victron.BLEBackend, error) {
		return macos.New()
	})
}
