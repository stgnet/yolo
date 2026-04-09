//go:build linux && cgo

// Package bluez provides a BLE backend implementation using BlueZ on Linux.
// Requires: BlueZ 5.43+ installed and proper permissions.
//
// Installation:
//
//	sudo apt-get install bluez libbluetooth-dev
//
// Permissions (may require udev rules or running as root):
//
//	The D-Bus Bluetooth service needs to be accessible.
//	Run with sudo or configure polkit/udev rules for D-Bus access.
package bluez

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/muka/go-bluetooth/bluez/profile/gatt"
	"github.com/scottstg/yolo/victron"
)

// Victron GATT UUIDs (from Victron documentation)
const (
	VictronServiceUUID    = "0000fff0-0000-1000-8000-00805f9b34fb" // VE.Direct Service
	VictronCharUUID       = "0000fff1-0000-1000-8000-00805f9b34fb" // VE.Direct Characteristic
	VictronDescriptorUUID = "0000fff2-0000-1000-8000-00805f9b34fb" // Client Characteristic Configuration
)

// Backend implements victron.BLEBackend using BlueZ.
type Backend struct {
	adapter *adapter.Adapter1
}

// New creates a new BlueZ backend.
func New() (*Backend, error) {
	return &Backend{}, nil
}

// Initialize initializes the BLE adapter.
func (b *Backend) Initialize() error {
	err := api.Exit()
	if err != nil {
		return fmt.Errorf("failed to initialize bluetooth: %w", err)
	}

	a, err := api.GetDefaultAdapter()
	if err != nil {
		return fmt.Errorf("failed to get default adapter: %w", err)
	}
	b.adapter = a
	return nil
}

// Close shuts down the BLE backend.
func (b *Backend) Close() error {
	b.adapter = nil
	return nil
}

// ScanForDevices discovers nearby Bluetooth devices.
func (b *Backend) ScanForDevices(ctx context.Context, duration time.Duration) ([]victron.BLEDevice, error) {
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
	timeoutCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	deviceMap := make(map[string]bool)

	for {
		select {
		case <-timeoutCtx.Done():
			return results, nil
		case discovered := <-devicesCh:
			dev, err := device.NewDevice1(discovered.Path)
			if err != nil {
				continue
			}

			deviceAddr, err := dev.GetAddress()
			if err != nil || deviceAddr == "" {
				continue
			}

			if deviceMap[deviceAddr] {
				continue
			}
			deviceMap[deviceAddr] = true

			deviceName, _ := dev.GetName()
			rssi, _ := dev.GetRSSI()
			uuids, _ := dev.GetUUIDs()

			results = append(results, victron.BLEDevice{
				Address:      deviceAddr,
				Name:         deviceName,
				RSSI:         int(rssi),
				ServiceUUIDs: uuids,
			})
		}
	}
}

// bleConnection wraps a BlueZ device connection.
type bleConnection struct {
	dev     *device.Device1
	address string
}

func (c *bleConnection) Address() string { return c.address }
func (c *bleConnection) UUID() string    { return "" }

func (c *bleConnection) IsConnected() bool {
	connected, err := c.dev.GetConnected()
	if err != nil {
		return false
	}
	return connected
}

func (c *bleConnection) Close() error {
	return c.dev.Disconnect()
}

func (c *bleConnection) ReadValue(char victron.BLECharacteristic) ([]byte, error) {
	gc, err := gatt.NewGattCharacteristic1(dbus.ObjectPath(string(c.dev.Path()) + "/service/char" + fmt.Sprintf("%04x", char.Handle)))
	if err != nil {
		return nil, fmt.Errorf("failed to get characteristic: %w", err)
	}
	return gc.ReadValue(nil)
}

func (c *bleConnection) WriteValue(char victron.BLECharacteristic, data []byte) error {
	gc, err := gatt.NewGattCharacteristic1(dbus.ObjectPath(string(c.dev.Path()) + "/service/char" + fmt.Sprintf("%04x", char.Handle)))
	if err != nil {
		return fmt.Errorf("failed to get characteristic: %w", err)
	}
	return gc.WriteValue(data, nil)
}

func (c *bleConnection) SubscribeToNotifications(char victron.BLECharacteristic) (<-chan []byte, error) {
	gc, err := gatt.NewGattCharacteristic1(dbus.ObjectPath(string(c.dev.Path()) + "/service/char" + fmt.Sprintf("%04x", char.Handle)))
	if err != nil {
		return nil, fmt.Errorf("failed to get characteristic: %w", err)
	}

	err = gc.StartNotify()
	if err != nil {
		return nil, fmt.Errorf("failed to start notifications: %w", err)
	}

	ch := make(chan []byte, 20)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			val, err := gc.GetValue()
			if err != nil {
				continue
			}
			if len(val) > 0 {
				select {
				case ch <- val:
				default:
				}
			}
		}
	}()

	return ch, nil
}

func (c *bleConnection) UnsubscribeFromNotifications(char victron.BLECharacteristic) error {
	gc, err := gatt.NewGattCharacteristic1(dbus.ObjectPath(string(c.dev.Path()) + "/service/char" + fmt.Sprintf("%04x", char.Handle)))
	if err != nil {
		return fmt.Errorf("failed to get characteristic: %w", err)
	}
	return gc.StopNotify()
}

// Connect establishes a connection to the device.
func (b *Backend) Connect(address string) (victron.BLEConnection, error) {
	if b.adapter == nil {
		return nil, errors.New("bluez adapter not initialized")
	}

	// Find device by scanning adapter's known devices
	devices, err := b.adapter.GetDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	for _, dev := range devices {
		addr, err := dev.GetAddress()
		if err != nil {
			continue
		}
		if strings.EqualFold(addr, address) {
			err = dev.Connect()
			if err != nil {
				return nil, fmt.Errorf("failed to connect: %w", err)
			}
			return &bleConnection{dev: dev, address: address}, nil
		}
	}

	return nil, fmt.Errorf("device %s not found", address)
}

// DiscoverServices discovers GATT services on a connected device.
func (b *Backend) DiscoverServices(conn victron.BLEConnection) ([]victron.BLEService, error) {
	bc, ok := conn.(*bleConnection)
	if !ok {
		return nil, errors.New("invalid connection type")
	}

	// Wait for services to be resolved
	for i := 0; i < 50; i++ {
		resolved, err := bc.dev.GetServicesResolved()
		if err == nil && resolved {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	uuids, err := bc.dev.GetUUIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get service UUIDs: %w", err)
	}

	var services []victron.BLEService
	for _, uuid := range uuids {
		services = append(services, victron.BLEService{
			UUID:    uuid,
			Primary: true,
		})
	}

	return services, nil
}

// DiscoverCharacteristics discovers characteristics in a service.
func (b *Backend) DiscoverCharacteristics(service victron.BLEService) ([]victron.BLECharacteristic, error) {
	// This would require walking the D-Bus object tree to find characteristics
	// belonging to the given service UUID.
	return nil, fmt.Errorf("DiscoverCharacteristics not yet implemented for BlueZ backend")
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

// ErrNotConnected is returned when operations are performed without connection.
var ErrNotConnected = errors.New("not connected to device")

// IsNotConnected checks if an error is due to disconnection.
func IsNotConnected(err error) bool {
	return err == ErrNotConnected || errors.Is(err, ErrNotConnected)
}
