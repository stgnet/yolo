//go:build darwin

package macos

import (
	"context"
	"fmt"
	"strings"
	"time"

	go_ble "github.com/go-ble/ble"
	_ "github.com/go-ble/ble/darwin"
	"github.com/scottstg/yolo/victron/ble"
)

const (
	VictronServiceUUID = "0000fff0-0000-1000-8000-00805f9b34fb"
	VictronCharUUID    = "0000fff1-0000-1000-8000-00805f9b34fb"
)

type Backend struct {
	initialized bool
}

func New() (*Backend, error) {
	return &Backend{}, nil
}

func (b *Backend) Initialize() error {
	if b.initialized {
		return nil
	}
	fmt.Println("[INFO] macOS BLE backend ready via CoreBluetooth")
	b.initialized = true
	return nil
}

func (b *Backend) Close() error {
	b.initialized = false
	return nil
}

func (b *Backend) ScanForDevices(ctx context.Context, duration time.Duration) ([]ble.Device, error) {
	if !b.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	fmt.Printf("[INFO] Scanning for BLE devices for %v...\n", duration)

	var devices []ble.Device
	devicesCh := make(chan ble.Device, 100)

	go func() {
		defer close(devicesCh)
		err := go_ble.Scan(ctx, false, func(adv go_ble.Advertisement) {
			device := ble.Device{
				Address: adv.Addr().String(),
				Name:    adv.LocalName(),
			}
			if isVictronDevice(adv) {
				device.IsVictron = true
			}
			devicesCh <- device
		}, nil)
		if err != nil {
			fmt.Printf("[ERROR] Scan failed: %v\n", err)
		}
	}()

	timeout := time.After(duration + 2*time.Second)
	for {
		select {
		case dev, ok := <-devicesCh:
			if !ok {
				return devices, nil
			}
			devices = append(devices, dev)
			fmt.Printf("  Found: %s (%s)", dev.Address, dev.Name)
			if dev.IsVictron {
				fmt.Print(" [VICTRON]")
			}
			fmt.Println()
		case <-timeout:
			fmt.Printf("[INFO] Scan timeout - found %d device(s)\n", len(devices))
			return devices, nil
		case <-ctx.Done():
			return devices, ctx.Err()
		}
	}
}

func isVictronDevice(adv go_ble.Advertisement) bool {
	name := strings.ToLower(adv.LocalName())
	victronNames := []string{"glow", "smartshunt", "smartsolar", "ve.direct", "victron"}
	for _, vName := range victronNames {
		if strings.Contains(name, vName) {
			return true
		}
	}
	return false
}

// bleConnection wraps a go-ble Client connection
type bleConnection struct {
	client  go_ble.Client
	address string
	uuid    string                       // Device UUID (macOS uses client UUID instead of MAC)
	svcMap  map[string]*go_ble.Service   // Cache of discovered services by UUID
}

func (c *bleConnection) Address() string { return c.address }

func (c *bleConnection) UUID() string { return c.uuid }

func (c *bleConnection) IsConnected() bool {
	if c.client == nil {
		return false
	}
	select {
	case <-c.client.Disconnected():
		return false
	default:
		return true
	}
}

func (c *bleConnection) Close() error {
	if c.client != nil {
		return c.client.CancelConnection()
	}
	return nil
}

// findCharacteristic searches for a characteristic by UUID across all discovered services
func (c *bleConnection) findCharacteristic(charUUID, serviceUUID string) (*go_ble.Characteristic, error) {
	charID, err := go_ble.Parse(charUUID)
	if err != nil {
		return nil, fmt.Errorf("invalid characteristic UUID: %w", err)
	}

	// If service UUID is specified, filter to that service
	var targetService *go_ble.Service
	if serviceUUID != "" {
		if svc, ok := c.svcMap[serviceUUID]; ok {
			targetService = svc
		}
	}

	// Search in cached services first
	if c.svcMap != nil {
		for uuid, svc := range c.svcMap {
			if targetService != nil && uuid != serviceUUID {
				continue
			}
			characteristics, err := c.client.DiscoverCharacteristics([]go_ble.UUID{charID}, svc)
			if err == nil && len(characteristics) > 0 {
				return characteristics[0], nil
			}
		}
	}

	return nil, fmt.Errorf("characteristic not found: %s", charUUID)
}

func (c *bleConnection) ReadValue(char ble.Characteristic) ([]byte, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	charPtr, err := c.findCharacteristic(char.UUID, "")
	if err != nil {
		return nil, err
	}
	value, err := c.client.ReadCharacteristic(charPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}
	return value, nil
}

func (c *bleConnection) WriteValue(char ble.Characteristic, data []byte) error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}
	charPtr, err := c.findCharacteristic(char.UUID, "")
	if err != nil {
		return err
	}
	err = c.client.WriteCharacteristic(charPtr, data, false)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (c *bleConnection) SubscribeToNotifications(char ble.Characteristic) (<-chan []byte, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	charPtr, err := c.findCharacteristic(char.UUID, "")
	if err != nil {
		return nil, err
	}
	
	// Create notification channel
	notifCh := make(chan []byte, 10)
	
	// Subscribe to notifications
	err = c.client.Subscribe(charPtr, false, func(data []byte) {
		notifCh <- data
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	
	return notifCh, nil
}

func (c *bleConnection) UnsubscribeFromNotifications(char ble.Characteristic) error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}
	charPtr, err := c.findCharacteristic(char.UUID, "")
	if err != nil {
		return err
	}
	return c.client.Unsubscribe(charPtr, false)
}

func (b *Backend) Connect(address string) (ble.Connection, error) {
	if !b.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	fmt.Printf("[INFO] Connecting to device %s...\n", address)

	addr := go_ble.NewAddr(address)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	client, err := go_ble.Dial(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	conn := &bleConnection{
		client:  client,
		address: address,
		uuid:    "", // Will be set after connection
		svcMap:  make(map[string]*go_ble.Service),
	}

	return conn, nil
}

func (b *Backend) DiscoverServices(conn ble.Connection) ([]ble.Service, error) {
	// Cast to internal type
	internalConn, ok := conn.(*bleConnection)
	if !ok {
		return nil, fmt.Errorf("invalid connection type")
	}

	services, err := internalConn.client.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	result := make([]ble.Service, len(services))
	for i, svc := range services {
		// Convert go-ble UUID (byte slice) to string
		uuidStr := svc.UUID.String()
		result[i] = ble.Service{
			UUID:        uuidStr,
			StartHandle: 0, // Not available in go-ble API
			EndHandle:   0,
			Primary:     true,
		}
		internalConn.svcMap[uuidStr] = svc
		fmt.Printf("  Service: %s\n", uuidStr)
	}

	return result, nil
}

func (b *Backend) DiscoverCharacteristics(service ble.Service) ([]ble.Characteristic, error) {
	// This needs a connection to work properly - simplified for now
	return nil, fmt.Errorf("DiscoverCharacteristics requires a connection, use connection-specific discovery")
}
