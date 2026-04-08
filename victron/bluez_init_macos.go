//go:build darwin

// macOS BLE backend initialization
package victron

import "fmt"

func init() {
	SetBackendCreator(func() (BLEBackend, error) {
		backend, err := NewMacOSBackend()
		if err != nil {
			return nil, fmt.Errorf("failed to create macOS BLE backend: %w", err)
		}
		return backend, nil
	})
}

// NewBlueZBackendCompatible returns a macOS BLE backend
func NewBlueZBackendCompatible() BLEBackend {
	backend, _ := NewMacOSBackend()
	return backend
}
