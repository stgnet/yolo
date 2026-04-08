//go:build linux && cgo

// Package bluez provides a BLE backend implementation using BlueZ on Linux.
// Requires: BlueZ 5.43+ installed and proper permissions.
// 
// Installation:
//   sudo apt-get install bluez libbluetooth-dev
//   
// Permissions (may require udev rules or running as root):
//   The D-Bus Bluetooth service needs to be accessible.
//   Run with sudo or configure polkit/udev rules for D-Bus access.
package bluez

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/gatt"
	"github.com/scottstg/yolo/victron"
)

// Victron GATT UUIDs (from Victron documentation)
const (
	VictronServiceUUID     = "0000fff0-0000-1000-8000-00805f9b34fb" // VE.Direct Service
	VictronCharUUID        = "0000fff1-0000-1000-8000-00805f9b34fb" // VE.Direct Characteristic
	VictronDescriptorUUID  = "0000fff2-0000-1000-8000-00805f9b34fb" // Client Characteristic Configuration
)

// Backend implements victron.BLEBackend using BlueZ.
type Backend struct {
	adapter    *adapter.Adapter1
	device     *api.Device
	service    *gatt.GattService1
	characteristic *gatt.GattCharacteristic1
	notify     chan []byte
	done       chan struct{} // For cleanup
}

// New creates a new BlueZ backend.
func New() (*Backend, error) {
	err := api.Exit()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bluetooth: %w", err)
	}

	a, err := api.GetDefaultAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to get default adapter: %w", err)
	}

	return &Backend{
		adapter: a,
		notify:  make(chan []byte, 20),
		done:    make(chan struct{}),
	}, nil
}

// Scan discovers nearby Bluetooth devices.
func (b *Backend) Scan(timeout int64) ([]victron.BLEDevice, error) {
	if b.adapter == nil {
		return nil, errors.New("bluez adapter not initialized")
	}

	filter := &adapter.DiscoveryFilter{
		Transport: "le",
	}

	devicesCh, stopScan, err := api.Discover(b.adapter, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to start discovery: %w", err)
	}
	defer stopScan()

	var results []victron.BLEDevice
	timeoutTimer := time.After(time.Duration(timeout) * time.Millisecond)

	for {
		select {
		case <-timeoutTimer:
			return results, nil
		case deviceDiscovered := <-devicesCh:
			device, err := api.NewDevice(deviceDiscovered.Path)
			if err != nil {
				continue
			}

			props, err := device.GetProperties()
			if err != nil {
				continue
			}

			// Look for Victron devices by name
			nameVariant := props.GetValue("Name")
			if nameVariant == nil {
				continue
			}

			nameBytes, ok := nameVariant.([]uint8)
			if !ok {
				continue
			}
			deviceName := string(nameBytes)
			
			if deviceName == "" {
				continue
			}

			if containsVictronPrefix(deviceName) {
				rssiVariant := props.GetValue("RSSI")
				rssi := int16(0)
				if rssiVariant != nil {
					if r, ok := rssiVariant.(int16); ok {
						rssi = r
					}
				}

				results = append(results, victron.BLEDevice{
					Address: device.Path().String(),
					Name:    deviceName,
					RSSI:    int(rssi),
				})
			}
		}
	}
}

func containsVictronPrefix(name string) bool {
	victronPrefixes := []string{"VE.Direct", "SmartSolar", "SmartShunt"}
	for _, prefix := range victronPrefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// Connect establishes a connection to the device and discovers GATT services.
func (b *Backend) Connect(device victron.BLEDevice) error {
	// Create device from address path
	var err error
	b.device, err = api.NewDevice(api.ObjectPath(device.Address))
	if err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}

	// Connect to the device
	err = b.device.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Get device properties and find GATT services
	props, err := b.device.GetProperties()
	if err != nil {
		return fmt.Errorf("failed to get properties: %w", err)
	}

	servicesVariant := props.GetValue("Services")
	if servicesVariant == nil {
		return errors.New("no GATT services found on device")
	}

	// Services is returned as []dbus.ObjectPath through GetProperty method
	// Use the helper method to get service paths
	servicePaths, err := props.GetServices()
	if err != nil {
		// Fallback: try to parse the variant directly
		return fmt.Errorf("failed to get services: %w", err)
	}

	// Find VE.Direct service by UUID
	for _, svcPath := range servicePaths {
		service, err := gatt.NewGattService1(svcPath)
		if err != nil {
			continue
		}
		
		uuid, err := service.GetUUID()
		if err != nil {
			continue
		}
		
		if strings.EqualFold(uuid, VictronServiceUUID) {
			b.service = service
			break
		}
	}
	
	if b.service == nil {
		return errors.New("VE.Direct service not found")
	}

	// Find the VE.Direct characteristic within the service
	characteristicPaths, err := b.service.GetCharacteristics()
	if err != nil {
		return fmt.Errorf("failed to get characteristics: %w", err)
	}

	for _, charPath := range characteristicPaths {
		char, err := gatt.NewGattCharacteristic1(charPath)
		if err != nil {
			continue
		}
		
		uuid, err := char.GetUUID()
		if err != nil {
			continue
		}
		
		if strings.EqualFold(uuid, VictronCharUUID) {
			b.characteristic = char
			
			// Enable notifications on the characteristic
			err = b.enableNotifications()
			if err != nil {
				return fmt.Errorf("failed to enable notifications: %w", err)
			}
			
			break
		}
	}

	if b.characteristic == nil {
		return errors.New("VE.Direct characteristic not found")
	}

	// Start notification listener
	go b.listenForNotifications()

	return nil
}

func (b *Backend) enableNotifications() error {
	if b.characteristic == nil {
		return errors.New("characteristic not set")
	}

	// Write to Client Characteristic Configuration descriptor
	// Notification value: 0x01
	notificationValue := []byte{0x01, 0x00} // MTU length
	
	props := b.characteristic.Properties
	if props == nil {
		return errors.New("characteristic properties not available")
	}

	// Use WriteValue method to enable notifications
	_, err := b.characteristic.WriteValue(notificationValue)
	if err != nil {
		return fmt.Errorf("failed to write notification value: %w", err)
	}

	return nil
}

func (b *Backend) listenForNotifications() {
	// Subscribe to characteristic value changes
	// The go-bluetooth library provides a Value property that gets updated
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-b.done:
			return
		case <-ticker.C:
			if b.characteristic == nil {
				continue
			}
			
			props, err := b.characteristic.GetProperties()
			if err != nil {
				continue
			}
			
			valueVariant := props.GetValue("Value")
			if valueVariant == nil {
				continue
			}
			
			// Value is typically []byte or [][]uint8
			var data []byte
			
			switch v := valueVariant.(type) {
			case []byte:
				data = v
			case [][]uint8:
				// Concatenate all chunks
				for _, chunk := range v {
					data = append(data, chunk...)
				}
			default:
				continue
			}
			
			if len(data) == 0 {
				continue
			}
			
			// Send to notify channel (non-blocking)
			select {
			case b.notify <- data:
			default:
				// Channel full, skip this notification
			}
		}
	}
}

// Disconnect closes the connection.
func (b *Backend) Disconnect() error {
	close(b.done)
	
	if b.characteristic != nil {
		// Disable notifications by writing 0x00 to descriptor
		disableValue := []byte{0x00, 0x00}
		_, _ = b.characteristic.WriteValue(disableValue)
		b.characteristic = nil
	}
	
	if b.device != nil {
		err := b.device.Disconnect()
		b.device = nil
		return err
	}
	return nil
}

// GetValues retrieves current device values.
func (b *Backend) GetValues() (map[string]string, error) {
	if b.notify == nil || b.characteristic == nil {
		return nil, errors.New("not connected")
	}

	// Wait for the next notification with timeout
	select {
	case data := <-b.notify:
		return parseVedirectData(data), nil
	case <-time.After(5 * time.Second):
		return make(map[string]string), nil
	}
}

func parseVedirectData(data []byte) map[string]string {
	// Parse VE.Direct protocol messages from binary data
	// VE.Direct format: /V(V=12.5&A=3.2&SoC=85)\r\n followed by checksum byte
	
	result := make(map[string]string)
	
	if len(data) == 0 {
		return result
	}

	// Convert to string and parse VE.Direct messages
	message := string(data)
	
	// VE.Direct messages start with '/'
	if !strings.HasPrefix(message, "/") {
		return result
	}

	// Remove checksum (last byte before newline)
	content := strings.TrimSpace(message)
	if len(content) > 2 {
		content = content[1 : len(content)-1] // Remove leading '/' and trailing checksum
	}

	// Parse the message using the victron parser
	// For now, return raw data as-is
	result["raw"] = content
	
	return result
}

// Subscribe sets up a callback for real-time updates.
func (b *Backend) Subscribe(callback func(map[string]string)) (func(), error) {
	if b.notify == nil || b.characteristic == nil {
		return nil, errors.New("not connected")
	}

	unsubscribe := make(chan struct{})

	go func() {
		for {
			select {
			case <-unsubscribe:
				return
			case data := <-b.notify:
				values := parseVedirectData(data)
				callback(values)
			}
		}
	}()

	return func() {
		close(unsubscribe)
	}, nil
}

// ErrNotConnected is returned when operations are performed without connection.
var ErrNotConnected = errors.New("not connected to device")

// IsNotConnected checks if an error is due to disconnection.
func IsNotConnected(err error) bool {
	return err == ErrNotConnected || errors.Is(err, ErrNotConnected)
}
