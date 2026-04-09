//go:build darwin

// macOS BLE backend initialization
package victron

func init() {
	SetBackendCreator(func() (BLEBackend, error) {
		backend, err := NewMacOSBackend()
		if err != nil {
			return nil, err
		}
		return backend, nil
	})
}
