//go:build darwin

// Package macos provides a BLE backend implementation using go-bluetooth on macOS.
// Requires: macOS 10.12+ with Bluetooth enabled.
package macos

import (
	"context"
	"fmt"
	"time"

	bt "github.com/muka/go-bluetooth/api"
	"github.com/scottstg/yolo/victron"
)

// Backend implements victron.BLEBackend using go-bluetooth library on macOS.
type Backend struct {
	mgr        *bt.Manager
	connected  bool
	deviceAddr string
}

// New creates a new BLE backend.
func New() (*Backend, error) {
	return &Backend{
		connected: false,
	}, nil
}

// Initialize sets up the Bluetooth manager.
func (b *Backend) Initialize() error {
	fmt.Println("[DEBUG] Initializing macOS BLE backend...")
	var err error
	b.mgr, err = bt.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create Bluetooth manager: %w", err)
	}

	// Start the manager
	if err := b.mgr.Start(); err != nil {
		return fmt.Errorf("failed to start Bluetooth manager: %w", err)
	}

	fmt.Println("[DEBUG] Bluetooth manager started successfully")
	return nil
}

// Close shuts down the BLE connection.
func (b *Backend) Close() error {
	if b.mgr != nil {
		b.mgr.Stop()
	}
	return nil
}

// ScanForDevices scans for nearby BLE devices.
func (b *Backend) ScanForDevices(ctx context.Context, duration time.Duration) ([]victron.BLEDevice, error) {
	if b.mgr == nil {
		return nil, fmt.Errorf("backend not initialized")
	}

	fmt.Printf("[DEBUG] Starting Bluetooth scan for %v\n", duration)

	// Start scanning
	devices := make(chan *bt.DeviceInfo, 100)
	if err := b.mgr.SetMonitoring(true); err != nil {
		return nil, fmt.Errorf("failed to start monitoring: %w", err)
	}

	fmt.Println("[DEBUG] Monitoring enabled, starting scan...")

	// Scan for devices
	go func() {
		mgrScanErr := b.mgr.Scan(ctx, devices)
		if mgrScanErr != nil {
			fmt.Printf("[DEBUG] Scan completed with error: %v\n", mgrScanErr)
		} else {
			fmt.Println("[DEBUG] Scan completed successfully")
		}
	}()

	var results []victron.BLEDevice
	
	// Collect devices for the duration
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	timeout := time.After(duration)
	lastReport := time.Now()
	for {
		select {
		case <-timeout:
			b.mgr.StopScan()
			fmt.Printf("[DEBUG] Scan timeout reached. Total devices found: %d\n", len(results))
			return results, nil
		case device, ok := <-devices:
			if !ok {
				fmt.Println("[DEBUG] Device channel closed")
				continue
			}
			
			lastReport = time.Now()
			
			// Filter for devices that might be Victron (based on name or manufacturer data)
			name := ""
			if device.Advertisement != nil && device.Advertisement.LocalName != "" {
				name = device.Advertisement.LocalName
			}
			
			// Check if this looks like a Victron device
			isVictron := false
			lowerName := name
			for _, prefix := range []string{"SmartSolar", "SmartShunt", "VE.Direct", "Phoenix", "MultiPlus"} {
				if len(lowerName) >= len(prefix) && lowerName[:len(prefix)] == prefix {
					isVictron = true
					break
				}
			}
			
			rssi := 0
			if device.Advertisement != nil {
				rssi = int(device.Advertisement.RSSI)
			}
			
			deviceInfo := victron.BLEDevice{
				Address:          device.Addr.String(),
				Name:             name,
				RSSI:             rssi,
				ServiceUUIDs:     []string{},
				ManufacturerData: make(map[uint16][]byte),
			}
			
			results = append(results, deviceInfo)
			
			if isVictron {
				fmt.Printf("[DEBUG] Found Victron device: %s at %s (RSSI: %d)\n", name, device.Addr.String(), rssi)
			}
		case <-ticker.C:
			// Print progress periodically
			if time.Since(lastReport) > 2*time.Second {
				fmt.Printf("[DEBUG] Scan in progress... found %d devices so far\n", len(results))
				lastReport = time.Now()
			}
		case <-ctx.Done():
			b.mgr.StopScan()
			fmt.Printf("[DEBUG] Context cancelled. Total devices found: %d\n", len(results))
			return results, nil
		}
	}
}

// Connect establishes a connection to a device.
func (b *Backend) Connect(address string) (victron.BLEConnection, error) {
	if b.mgr == nil {
		return nil, fmt.Errorf("backend not initialized")
	}

	conn := &BLEConnection{
		backend:  b,
		address:  address,
		connected: false,
	}

	b.deviceAddr = address
	b.connected = true

	return conn, nil
}

// DiscoverServices discovers GATT services on a connected device.
func (b *Backend) DiscoverServices(conn victron.BLEConnection) ([]victron.BLEService, error) {
	// This would require actual GATT service discovery
	// For now, return empty list
	return []victron.BLEService{}, nil
}

// DiscoverCharacteristics discovers characteristics in a service.
func (b *Backend) DiscoverCharacteristics(service victron.BLEService) ([]victron.BLECharacteristic, error) {
	// This would require actual GATT characteristic discovery
	// For now, return empty list
	return []victron.BLECharacteristic{}, nil
}

// BLEConnection implements the BLEConnection interface for macOS.
type BLEConnection struct {
	backend  *Backend
	address  string
	connected bool
	conn     *bt.Connection
}

func (c *BLEConnection) Address() string {
	return c.address
}

func (c *BLEConnection) UUID() string {
	return ""
}

func (c *BLEConnection) IsConnected() bool {
	return c.connected && c.conn != nil
}

func (c *BLEConnection) Close() error {
	if c.conn != nil {
		c.conn.Disconnect()
		c.conn = nil
	}
	c.connected = false
	return nil
}

func (c *BLEConnection) ReadValue(characteristic victron.BLECharacteristic) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *BLEConnection) WriteValue(characteristic victron.BLECharacteristic, data []byte) error {
	return fmt.Errorf("not implemented")
}

func (c *BLEConnection) SubscribeToNotifications(characteristic victron.BLECharacteristic) (<-chan []byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *BLEConnection) UnsubscribeFromNotifications(characteristic victron.BLECharacteristic) error {
	return fmt.Errorf("not implemented")
}
