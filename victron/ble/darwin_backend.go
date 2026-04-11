// BLE backend implementation for macOS using go-ble with CoreBluetooth
//go:build darwin
// +build darwin

package ble

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-ble/ble"
)

// DarwinBackend implements BLE operations on macOS using the go-ble library
type DarwinBackend struct {
	scanning     bool
	discovered   []Device
	scanDuration time.Duration
}

// NewDarwinBackend creates a new Darwin BLE backend
func NewDarwinBackend() (Backend, error) {
	return &DarwinBackend{
		discovered: make([]Device, 0),
	}, nil
}

// Initialize initializes the BLE adapter on macOS
func (b *DarwinBackend) Initialize() error {
	// Set services if needed (for advertising mode)
	// For scanning/discovery, we don't need to set services
	return nil
}

// Close shuts down the BLE connection
func (b *DarwinBackend) Close() error {
	ble.Stop()
	return nil
}

// ScanForDevices scans for nearby BLE devices
func (b *DarwinBackend) ScanForDevices(ctx context.Context, duration time.Duration) ([]Device, error) {
	b.discovered = make([]Device, 0)
	b.scanning = true
	b.scanDuration = duration

	// Create a context with timeout for the scan
	scanCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	// Set up advertisement handler to receive discovered devices
	var mu sync.Mutex
	
	err := ble.Scan(scanCtx, false, func(a ble.Advertisement) {
		mu.Lock()
		defer mu.Unlock()

		// Check if this looks like a Victron device
		isVictron := b.detectAsVictron(a.LocalName(), a.Services())

		device := Device{
			Address:          a.Addr().String(),
			Name:             a.LocalName(),
			RSSI:             a.RSSI(),
			ServiceUUIDs:     b.uuidsToStringSlice(a.Services()),
			ManufacturerData: b.parseManufacturerDataAdvertisement(a),
			IsVictron:        isVictron,
		}

		// Avoid duplicates
		for _, d := range b.discovered {
			if d.Address == device.Address {
				return
			}
		}

		b.discovered = append(b.discovered, device)
	}, nil)

	if err != nil && err != context.DeadlineExceeded {
		b.scanning = false
		return b.discovered, fmt.Errorf("scan failed: %w", err)
	}

	b.scanning = false

	return b.discovered, nil
}

// Connect establishes a connection to a device
func (b *DarwinBackend) Connect(address string) (Connection, error) {
	// Check if BLE is available - ble.Device() should return the default device
	// If not set, we can still try to connect but it may fail
	
	// Parse address
	addr := ble.NewAddr(address)

	// Dial the device
	conn, err := ble.Dial(context.Background(), addr)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	// Wrap the connection in our interface
	return &DarwinConnection{conn: conn, address: address}, nil
}

// DiscoverServices discovers GATT services on a connected device
func (b *DarwinBackend) DiscoverServices(conn Connection) ([]Service, error) {
	darwinConn := conn.(*DarwinConnection)

	// Discover all services
	servicesRaw, err := darwinConn.conn.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	services := make([]Service, 0, len(servicesRaw))
	for _, svc := range servicesRaw {
		services = append(services, Service{
			UUID:        svc.UUID.String(),
			StartHandle: svc.Handle,      // ble.Service uses Handle instead of StartHandle
			EndHandle:   svc.EndHandle,
			Primary:     true,            // go-ble only returns primary services
		})
	}

	return services, nil
}

// DiscoverCharacteristics discovers characteristics in a service
func (b *DarwinBackend) DiscoverCharacteristics(service Service) ([]Characteristic, error) {
	// This needs to be called on the connection, not the backend
	// We need to find the right service from the profile first
	return nil, fmt.Errorf("not implemented - use DiscoverCharacteristicsOnService instead")
}

// DarwinConnection wraps a ble.Client with our Connection interface
type DarwinConnection struct {
	conn    ble.Client
	address string
}

// Address returns the device address
func (c *DarwinConnection) Address() string {
	return c.address
}

// UUID returns the device UUID if available
func (c *DarwinConnection) UUID() string {
	return ""
}

// IsConnected checks if connection is active
func (c *DarwinConnection) IsConnected() bool {
	// Check if we can get the profile (indicates connected state)
	profile := c.conn.Profile()
	return profile != nil || c.conn.Addr() != nil
}

// Close disconnects from the device
func (c *DarwinConnection) Close() error {
	return c.conn.CancelConnection()
}

// ReadValue reads a characteristic value
func (c *DarwinConnection) ReadValue(characteristic Characteristic) ([]byte, error) {
	bleUUID := ble.MustParse(characteristic.UUID)
	bleChar := ble.NewCharacteristic(bleUUID)
	
	return c.conn.ReadCharacteristic(bleChar)
}

// WriteValue writes to a characteristic
func (c *DarwinConnection) WriteValue(characteristic Characteristic, data []byte) error {
	bleUUID := ble.MustParse(characteristic.UUID)
	bleChar := ble.NewCharacteristic(bleUUID)
	
	return c.conn.WriteCharacteristic(bleChar, data, false)
}

// SubscribeToNotifications subscribes to notifications from a characteristic
func (c *DarwinConnection) SubscribeToNotifications(characteristic Characteristic) (<-chan []byte, error) {
	bleUUID := ble.MustParse(characteristic.UUID)
	bleChar := ble.NewCharacteristic(bleUUID)
	
	ch := make(chan []byte, 10)
	
	handler := func(req []byte) {
		select {
		case ch <- req:
		default:
			// Channel full, drop notification
		}
	}
	
	err := c.conn.Subscribe(bleChar, false, handler)
	if err != nil {
		close(ch)
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	return ch, nil
}

// UnsubscribeFromNotifications stops receiving notifications
func (c *DarwinConnection) UnsubscribeFromNotifications(characteristic Characteristic) error {
	bleUUID := ble.MustParse(characteristic.UUID)
	bleChar := ble.NewCharacteristic(bleUUID)
	
	return c.conn.Unsubscribe(bleChar, false)
}

// Helper methods

func (b *DarwinBackend) detectAsVictron(name string, serviceUUIDs []ble.UUID) bool {
	nameLower := ""
	for _, c := range name {
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		nameLower += string(c)
	}

	// Check for Victron-related names
	victronKeywords := []string{
		"victron", "glow", "smartsolar", "smartshunt",
		"cerbo", "venus", "ve.direct", "smart",
	}

	for _, keyword := range victronKeywords {
		for i := 0; i <= len(nameLower)-len(keyword); i++ {
			if nameLower[i:i+len(keyword)] == keyword {
				return true
			}
		}
	}

	// Check for Victron-specific service UUIDs
	return len(serviceUUIDs) > 0
}

func (b *DarwinBackend) uuidsToStringSlice(uuids []ble.UUID) []string {
	result := make([]string, 0, len(uuids))
	for _, u := range uuids {
		result = append(result, u.String())
	}
	return result
}

// parseManufacturerDataAdvertisement extracts manufacturer data from a BLE advertisement
func (b *DarwinBackend) parseManufacturerDataAdvertisement(a ble.Advertisement) map[uint16][]byte {
	// For now, return empty map - extracting manufacturer data from go-ble Advertisement
	// requires proper handling of the ManufacturerData() method which may not be implemented
	// in all versions of go-ble or may require additional parsing
	return nil
}
