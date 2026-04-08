// Mock BLE backend for testing purposes
package victron

import (
	"context"
	"time"
)

// MockBackend implements BLEBackend for testing without real hardware
type MockBackend struct {
	initialized   bool
	devices       []BLEDevice
	connections   map[string]BLEConnection
}

// NewMockBackend creates a new mock backend for testing
func NewMockBackend() *MockBackend {
	return &MockBackend{
		devices:   make([]BLEDevice, 0),
		connections: make(map[string]BLEConnection),
	}
}

// Initialize implements BLEBackend.Initialize
func (m *MockBackend) Initialize() error {
	m.initialized = true
	return nil
}

// Close implements BLEBackend.Close
func (m *MockBackend) Close() error {
	m.initialized = false
	for _, conn := range m.connections {
		conn.Close()
	}
	m.connections = make(map[string]BLEConnection)
	return nil
}

// ScanForDevices implements BLEBackend.ScanForDevices
func (m *MockBackend) ScanForDevices(ctx context.Context, duration time.Duration) ([]BLEDevice, error) {
	// Return mock devices for testing
	if len(m.devices) == 0 {
		m.devices = []BLEDevice{
			{
				Address:      "AA:BB:CC:DD:EE:FF",
				Name:         "Victron SmartSolar 150/35",
				RSSI:         -65,
				ServiceUUIDs: []string{"fdcd"},
			},
			{
				Address:      "11:22:33:44:55:66",
				Name:         "Victron SmartShunt",
				RSSI:         -70,
				ServiceUUIDs: []string{"fdcd"},
			},
		}
	}
	
	// Simulate scan duration with context
	start := time.Now()
	ticker := time.NewTicker(duration / 10)
	defer ticker.Stop()
	
	for {
		elapsed := time.Since(start)
		if elapsed >= duration {
			break
		}
		
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// Simulating scan progress
		case <-time.After(duration - elapsed):
			break
		}
	}
	
	return m.devices, nil
}

// Connect implements BLEBackend.Connect
func (m *MockBackend) Connect(address string) (BLEConnection, error) {
	conn := &MockConnection{
		address:       address,
		connected:     true,
		valueChan:     make(chan []byte, 10),
	}
	m.connections[address] = conn
	return conn, nil
}

// DiscoverServices implements BLEBackend.DiscoverServices
func (m *MockBackend) DiscoverServices(conn BLEConnection) ([]BLEService, error) {
	return []BLEService{
		{
			UUID:        VictronServiceUUID128,
			StartHandle: 1,
			EndHandle:   10,
			Primary:     true,
		},
	}, nil
}

// DiscoverCharacteristics implements BLEBackend.DiscoverCharacteristics
func (m *MockBackend) DiscoverCharacteristics(service BLEService) ([]BLECharacteristic, error) {
	return []BLECharacteristic{
		{
			UUID:         VEDirectDataCharacteristic,
			Handle:       5,
			ValueHandle:  6,
			Properties:   PropNotify | PropRead,
			Description:  "VE.Direct Data",
		},
	}, nil
}

// MockConnection implements BLEConnection for testing
type MockConnection struct {
	address     string
	connected   bool
	valueChan   chan []byte
	closeChan   chan struct{}
}

// Address implements BLEConnection.Address
func (m *MockConnection) Address() string {
	return m.address
}

// UUID implements BLEConnection.UUID
func (m *MockConnection) UUID() string {
	return ""
}

// IsConnected implements BLEConnection.IsConnected
func (m *MockConnection) IsConnected() bool {
	return m.connected
}

// Close implements BLEConnection.Close
func (m *MockConnection) Close() error {
	m.connected = false
	if m.closeChan != nil {
		select {
		case <-m.closeChan:
			// Already closed
		default:
			close(m.closeChan)
		}
	}
	return nil
}

// ReadValue implements BLEConnection.ReadValue
func (m *MockConnection) ReadValue(characteristic BLECharacteristic) ([]byte, error) {
	// Return mock VE.Direct data
	return []byte("/V(V=12.345)" + string(calculateChecksum([]byte("/V(V=12.345)")))), nil
}

// WriteValue implements BLEConnection.WriteValue
func (m *MockConnection) WriteValue(characteristic BLECharacteristic, data []byte) error {
	// Mock write - no-op for read-only library
	return nil
}

// SubscribeToNotifications implements BLEConnection.SubscribeToNotifications
func (m *MockConnection) SubscribeToNotifications(characteristic BLECharacteristic) (<-chan []byte, error) {
	m.closeChan = make(chan struct{})
	
	// Simulate receiving some mock data
	go func() {
		data := []string{
			"/V(V=12.345)" + string(calculateChecksum([]byte("/V(V=12.345)"))),
			"/A(A=2.500)" + string(calculateChecksum([]byte("/A(A=2.500)"))),
			"/W(W=30.862)" + string(calculateChecksum([]byte("/W(W=30.862)"))),
			"/SoC(SoC=85.500)" + string(calculateChecksum([]byte("/SoC(SoC=85.500)"))),
		}
		
		for _, d := range data {
			select {
			case m.valueChan <- []byte(d):
			case <-m.closeChan:
				return
			case <-time.After(1 * time.Second):
			}
		}
	}()
	
	return m.valueChan, nil
}

// UnsubscribeFromNotifications implements BLEConnection.UnsubscribeFromNotifications
func (m *MockConnection) UnsubscribeFromNotifications(characteristic BLECharacteristic) error {
	select {
	case <-m.closeChan:
		// Already closed
	default:
		close(m.closeChan)
	}
	return nil
}
