//go:build darwin

package macos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-ble/ble"
	_ "github.com/go-ble/ble/darwin"
	"github.com/scottstg/yolo/victron"
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

func (b *Backend) ScanForDevices(ctx context.Context, duration time.Duration) ([]victron.BLEDevice, error) {
	if !b.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}

	fmt.Printf("[INFO] Scanning for BLE devices for %v...\n", duration)

	var devices []victron.BLEDevice
	devicesCh := make(chan victron.BLEDevice, 100)

	go func() {
		defer close(devicesCh)
		err := ble.Scan(ctx, false, func(adv ble.Advertisement) {
			device := victron.BLEDevice{
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

func isVictronDevice(adv ble.Advertisement) bool {
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
	client  ble.Client
	address string
	uuid    string                  // Device UUID (macOS uses client UUID instead of MAC)
	svcMap  map[string]*ble.Service // Cache of discovered services by UUID
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
func (c *bleConnection) findCharacteristic(charUUID, serviceUUID string) (*ble.Characteristic, error) {
	charID, err := ble.Parse(charUUID)
	if err != nil {
		return nil, fmt.Errorf("invalid characteristic UUID: %w", err)
	}

	// If service UUID is specified, filter to that service
	var targetService *ble.Service
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
			characteristics, err := c.client.DiscoverCharacteristics([]ble.UUID{charID}, svc)
			if err == nil && len(characteristics) > 0 {
				return characteristics[0], nil
			}
		}
	}

	return nil, fmt.Errorf("characteristic not found: %s", charUUID)
}

func (c *bleConnection) ReadValue(char victron.BLECharacteristic) ([]byte, error) {
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

func (c *bleConnection) WriteValue(char victron.BLECharacteristic, data []byte) error {
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

func (c *bleConnection) SubscribeToNotifications(char victron.BLECharacteristic) (<-chan []byte, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	charPtr, err := c.findCharacteristic(char.UUID, "")
	if err != nil {
		return nil, err
	}
	ch := make(chan []byte, 20)
	err = c.client.Subscribe(charPtr, false, func(value []byte) {
		select {
		case ch <- value:
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	return ch, nil
}

func (c *bleConnection) UnsubscribeFromNotifications(char victron.BLECharacteristic) error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}
	charPtr, err := c.findCharacteristic(char.UUID, "")
	if err != nil {
		return err
	}
	return c.client.Unsubscribe(charPtr, false)
}

func (b *Backend) Connect(address string) (victron.BLEConnection, error) {
	if !b.initialized {
		return nil, fmt.Errorf("backend not initialized")
	}
	fmt.Printf("[INFO] Connecting to device %s...\n", address)

	ctx := context.Background()

	// Parse MAC address from hex string (e.g., "A4:C1:38:xx:xx:xx" or raw hex)
	cleanAddr := strings.ReplaceAll(address, ":", "")
	addrBytes := make([]byte, 6)
	var addr ble.Addr
	if len(cleanAddr) == 12 {
		for i := 0; i < 6; i++ {
			fmt.Sscanf(cleanAddr[i*2:i*2+2], "%02x", &addrBytes[i])
		}
		addr = ble.NewAddr(fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
			addrBytes[0], addrBytes[1], addrBytes[2],
			addrBytes[3], addrBytes[4], addrBytes[5]))
	} else {
		return nil, fmt.Errorf("invalid MAC address format: %s", address)
	}

	client, err := ble.Dial(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Printf("[INFO] Connected to device %s\n", address)

	conn := &bleConnection{
		client:  client,
		address: address,
		uuid:    client.Addr().String(),
		svcMap:  make(map[string]*ble.Service),
	}

	return conn, nil
}

func (b *Backend) DiscoverServices(conn victron.BLEConnection) ([]victron.BLEService, error) {
	bc, ok := conn.(*bleConnection)
	if !ok || bc.client == nil {
		return nil, fmt.Errorf("invalid or disconnected")
	}

	services, err := bc.client.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	var victronServices []victron.BLEService
	victronSvcUUID, _ := ble.Parse(VictronServiceUUID)

	for _, svc := range services {
		svcUUID := string(svc.UUID)

		victronSvc := victron.BLEService{
			UUID:    svcUUID,
			Primary: true,
		}

		bc.svcMap[svcUUID] = svc

		if svc.UUID.Equal(victronSvcUUID) || strings.Contains(strings.ToLower(svcUUID), "fff0") {
			fmt.Printf("[INFO] Found Victron service: %s\n", svcUUID)
		}

		victronServices = append(victronServices, victronSvc)
	}

	fmt.Printf("[INFO] Discovered %d service(s)\n", len(victronServices))
	return victronServices, nil
}

func (b *Backend) DiscoverCharacteristics(service victron.BLEService) ([]victron.BLECharacteristic, error) {
	// This method is not fully implemented because we don't have the connection here
	// The characteristics need to be discovered from a connected client
	return nil, fmt.Errorf("use DiscoverServices first, then call DiscoverCharacteristics on the returned service")
}

func isVictronCharacteristic(uuid ble.UUID) bool {
	victronCharID, _ := ble.Parse(VictronCharUUID)
	return uuid.Equal(victronCharID) || strings.Contains(strings.ToLower(uuid.String()), "fff1")
}
